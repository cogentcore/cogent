// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
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
	giv.NewTableView(b).SetSlice(&ts).SetReadOnly(true)

	b.NewWindow().Run().Wait()
}
