// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

//go:generate core generate

import (
	_ "embed"
	"slices"
	"sync/atomic"
	"time"

	"cogentcore.org/core/base/datasize"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/tree"

	"github.com/shirou/gopsutil/v3/process"
)

type Task struct { //types:add
	*process.Process `display:"-"`

	// The name of this task
	Name string `grow:"1"`

	// The percentage of the CPU time this task uses
	CPU float64 `format:"%.3g%%"`

	// The actual number of bytes of RAM this task uses (RSS)
	RAM datasize.Size

	// The percentage of total RAM this task uses
	RAMPct float32 `label:"RAM %" format:"%.3g%%"`

	// The number of threads this task uses
	Threads int32

	// The user that started this task
	User string

	// The Process ID (PID) of this task
	PID int32
}

// appIcon is the icon of the app, which core build also sets, but go
// install does not, as for an app installed with AppGoSymlink.
//
//go:embed icon.svg
var appIcon string

func main() {
	core.AppIcon = appIcon
	b := core.NewBody("Cogent Task Manager")
	makeApp(b, time.Second)
	b.RunMainWindow()
}

// makeApp makes the app in the given body, updating the tasks at the
// given interval, and returns its table of tasks.
func makeApp(b *core.Body, interval time.Duration) *core.Table {
	ts := getTasks(b)
	tv := core.NewTable(b)
	tv.SetReadOnly(true)
	tv.SetSlice(&ts)

	// selected is the PID of the selected task, which stays selected when
	// the tasks are updated and sorted again, so that the actions apply to
	// the task that the user selected, not the one now at its index.
	selected := int32(-1)
	tv.OnSelect(func(e events.Event) {
		selected = -1
		if tv.SelectedIndex >= 0 && tv.SelectedIndex < len(ts) {
			selected = ts[tv.SelectedIndex].PID
		}
	})
	selectedTask := func() *Task {
		for _, t := range ts {
			if t.PID == selected {
				return t
			}
		}
		return nil
	}

	tv.OnDoubleClick(func(e events.Event) {
		t := selectedTask()
		if t == nil {
			return
		}
		d := core.NewBody("Task info")
		core.NewForm(d).SetStruct(&t).SetReadOnly(true)
		d.AddOKOnly().RunDialog(b)
	})

	tick := time.NewTicker(interval)
	var paused atomic.Bool

	b.OnShow(func(e events.Event) {
		tv.SortColumn(1)
		tv.SortColumn(1)
		go func() {
			for range tick.C {
				if paused.Load() {
					continue
				}
				nts := getTasks(b)
				if !tv.AsyncLockIfShown() { // closed
					tick.Stop()
					return
				}
				if paused.Load() { // paused while getting them
					tv.AsyncUnlock()
					continue
				}
				ts = nts // (with the lock, as the table shows it)
				tv.SortSlice()
				// the selected task stays selected, at its new index
				tv.ResetSelectedIndexes()
				tv.SelectedIndex = slices.IndexFunc(ts, func(t *Task) bool { return t.PID == selected })
				if tv.SelectedIndex >= 0 {
					tv.SelectIndex(tv.SelectedIndex)
				}
				tv.Update()
				tv.AsyncUnlock()
			}
		}()
	})

	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(func(p *tree.Plan) {
			tree.Add(p, func(w *core.Button) {
				w.SetText("End task").SetIcon(icons.Cancel).
					SetTooltip("Stop the currently selected task").
					OnClick(func(e events.Event) {
						t := selectedTask()
						if t == nil {
							core.MessageSnackbar(tv, "No task is selected")
							return
						}
						core.ErrorSnackbar(tv, t.Kill(), "Error ending task")
					})
			})
			tree.Add(p, func(w *core.Button) {
				w.SetText("Pause").SetIcon(icons.Pause).
					SetTooltip("Stop updating the list of tasks").
					OnClick(func(e events.Event) {
						paused.Store(!paused.Load())
						if paused.Load() {
							w.SetText("Resume").SetIcon(icons.Resume).
								SetTooltip("Resume updating the list of tasks")
						} else {
							w.SetText("Pause").SetIcon(icons.Pause).
								SetTooltip("Stop updating the list of tasks")
						}
						w.Update()
					})
			})
		})
	})
	return tv
}

// getTasks returns the current tasks, in an error dialog if it fails
// (in the window of the given body, which it locks).
func getTasks(b *core.Body) []*Task {
	ps, err := process.Processes()
	if errors.Log(err) != nil && b.AsyncLockIfShown() {
		core.ErrorDialog(b, err, "Error getting system processes")
		b.AsyncUnlock()
	}

	ts := make([]*Task, len(ps))
	for i, p := range ps {
		t := &Task{
			Process: p,
			Name:    errors.Ignore1(p.Name()),
			CPU:     errors.Ignore1(p.CPUPercent()),
			RAMPct:  errors.Ignore1(p.MemoryPercent()),
			Threads: errors.Ignore1(p.NumThreads()),
			User:    errors.Ignore1(p.Username()),
			PID:     p.Pid,
		}
		mi := errors.Ignore1(p.MemoryInfo())
		if mi != nil {
			t.RAM = datasize.Size(mi.RSS)
		}
		ts[i] = t
	}
	return ts
}
