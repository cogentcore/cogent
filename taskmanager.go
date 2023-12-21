// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

//go:generate goki generate

import (
	"time"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/giv"
	"goki.dev/goosi/events"
	"goki.dev/grr"
	"goki.dev/icons"

	"github.com/shirou/gopsutil/v3/process"
)

func main() { gimain.Run(app) }

type Task struct { //gti:add
	*process.Process `view:"-"`

	// The name of this task
	Name string

	// The percentage of the CPU time this task uses
	CPU float64 `format:"%.3g%%"`

	// The percentage of total RAM this task uses
	RAM float32 `format:"%.3g%%"`

	// The number of threads this task uses
	Threads int32

	// The user that started this task
	User string

	// The Process ID (PID) of this task
	PID int32
}

func app() {
	b := gi.NewAppBody("goki-task-manager")

	ts := getTasks(b)
	tv := giv.NewTableView(b).SetSlice(&ts)
	tv.SetReadOnly(true)
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
			tv.SortSliceAction(1)
			tv.SortSliceAction(1)
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
			Name:    grr.Log1(p.Name()),
			CPU:     grr.Log1(p.CPUPercent()),
			RAM:     grr.Log1(p.MemoryPercent()),
			Threads: grr.Log1(p.NumThreads()),
			User:    grr.Log1(p.Username()),
			PID:     p.Pid,
		}
		ts[i] = t
	}
	return ts
}
