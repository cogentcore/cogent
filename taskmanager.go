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

type Process struct {
	Name string
	PID  int32
}

func app() {
	b := gi.NewAppBody("goki-task-manager")

	procs := grr.Log1(process.Processes())
	ps := make([]*Process, len(procs))
	for i, proc := range procs {
		p := &Process{grr.Log1(proc.Name()), proc.Pid}
		ps[i] = p
	}
	giv.NewTableView(b).SetSlice(&ps)

	b.NewWindow().Run().Wait()
}
