// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

//go:generate core generate

import (
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/glop/datasize"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/icons"

	"github.com/shirou/gopsutil/v3/process"
)

type Task struct { //gti:add
	*process.Process `view:"-"`

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

func main() {
	b := gi.NewAppBody("Cogent Task Manager")

	ts := getTasks(b)
	tv := giv.NewTableView(b)
	tv.SetReadOnly(true)
	tv.SetSlice(&ts)
	tv.SortSliceAction(1)
	tv.SortSliceAction(1)

	tv.OnDoubleClick(func(e events.Event) {
		t := ts[tv.SelIdx]
		d := gi.NewBody().AddTitle("Task info")
		giv.NewStructView(d).SetStruct(&t).SetReadOnly(true)
		d.AddOkOnly().NewDialog(b).Run()
	})

	tick := time.NewTicker(time.Second)
	paused := false

	go func() {
		for range tick.C {
			if paused {
				continue
			}
			ts = getTasks(b)
			updt := tv.UpdateStartAsync()
			tv.SortSlice()
			tv.UpdateWidgets()
			tv.UpdateEndAsyncRender(updt)
		}
	}()

	b.AddAppBar(func(tb *gi.Toolbar) {
		gi.NewButton(tb).SetText("End task").SetIcon(icons.Cancel).
			SetTooltip("Stop the currently selected task").
			OnClick(func(e events.Event) {
				t := ts[tv.SelIdx]
				gi.ErrorSnackbar(tv, t.Kill(), "Error ending task")
			})
		pause := gi.NewButton(tb).SetText("Pause").SetIcon(icons.Pause).
			SetTooltip("Stop updating the list of tasks")
		pause.OnClick(func(e events.Event) {
			updt := pause.UpdateStart()
			paused = !paused
			if paused {
				pause.SetText("Resume").SetIcon(icons.Resume).
					SetTooltip("Resume updating the list of tasks")
			} else {
				pause.SetText("Pause").SetIcon(icons.Pause).
					SetTooltip("Stop updating the list of tasks")
			}
			pause.Update()
			pause.UpdateEndLayout(updt)
		})
	})

	b.NewWindow().Run().Wait()
}

func getTasks(b *gi.Body) []*Task {
	ps, err := process.Processes()
	gi.ErrorDialog(b, err, "Error getting system processes")

	ts := make([]*Task, len(ps))
	for i, p := range ps {
		t := &Task{
			Process: p,
			Name:    grr.Ignore1(p.Name()),
			CPU:     grr.Ignore1(p.CPUPercent()),
			RAMPct:  grr.Ignore1(p.MemoryPercent()),
			Threads: grr.Ignore1(p.NumThreads()),
			User:    grr.Ignore1(p.Username()),
			PID:     p.Pid,
		}
		mi := grr.Ignore1(p.MemoryInfo())
		if mi != nil {
			t.RAM = datasize.Size(mi.RSS)
		}
		ts[i] = t
	}
	return ts
}
