// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"time"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/giv"
	"goki.dev/grr"

	"github.com/shirou/gopsutil/v3/process"
)

func main() { gimain.Run(app) }

type Task struct {
	Name string
	CPU  float64 `label:"CPU %"`
	RAM  float32 `label:"RAM %"`
	PID  int32
}

func app() {
	b := gi.NewAppBody("goki-task-manager")

	ts := getTasks(b)
	tv := giv.NewTableView(b).SetSlice(&ts)
	tv.SetReadOnly(true)
	tv.SortSliceAction(1)
	tv.SortSliceAction(1)

	t := time.NewTicker(time.Second)
	go func() {
		for range t.C {
			ts = getTasks(b)
			tv.SortSliceAction(1)
			tv.SortSliceAction(1)
		}
	}()

	b.NewWindow().Run().Wait()
}

func getTasks(b *gi.Body) []*Task {
	ps, err := process.Processes()
	gi.ErrorDialog(b, err, "Error getting system processes")

	ts := make([]*Task, len(ps))
	for i, p := range ps {
		t := &Task{
			Name: grr.Log1(p.Name()),
			CPU:  grr.Log1(p.CPUPercent()),
			RAM:  grr.Log1(p.MemoryPercent()),
			PID:  p.Pid,
		}
		ts[i] = t
	}
	return ts
}
