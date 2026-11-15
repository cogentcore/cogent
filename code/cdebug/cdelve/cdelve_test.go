// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package cdelve

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"cogentcore.org/cogent/code/cdebug"
	"cogentcore.org/core/text/lines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// program is the program that is debugged in the tests.
const program = `package main

import "fmt"

type T struct {
	Name string
	N    int
	P    *T
}

var global = &T{Name: "g", N: 1}

func use(t *T, s []int, m map[string]int) int {
	n := t.N + len(s) + len(m) // use
	return n
}

func main() {
	t := &T{Name: "t", N: 2, P: global}
	s := []int{1, 2, 3}
	m := map[string]int{"a": 1}
	fmt.Println("result", use(t, s, m)) // main
}
`

// programTest is a test of the program.
const programTest = `package main

import "testing"

func TestUse(t *testing.T) {
	if use(&T{N: 1}, nil, nil) != 1 { // test
		t.Fail()
	}
}
`

// lineOf returns the 1-based line of the given marker comment in src.
func lineOf(src, marker string) int {
	return strings.Count(src[:strings.Index(src, "// "+marker)], "\n") + 1
}

// startDelve starts delve in the given mode on the program in a new
// temporary directory, waiting until it is ready, and returns it with the
// path of the program file.
func startDelve(t *testing.T, mode cdebug.Modes) (*GiDelve, string, *lines.Lines) {
	t.Helper()
	if _, err := exec.LookPath("dlv"); err != nil {
		t.Skip("dlv not found in PATH:", err)
	}
	t.Setenv("GOWORK", "off")
	t.Setenv("GO111MODULE", "on")
	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	for name, text := range map[string]string{"go.mod": "module prog\n\ngo 1.21\n", "main.go": program, "main_test.go": programTest} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644))
	}
	var mu sync.Mutex
	var stats []cdebug.Status
	pars := cdebug.DefaultParams
	pars.Mode = mode
	pars.TestName = "TestUse"
	pars.StatFunc = func(st cdebug.Status) {
		mu.Lock()
		stats = append(stats, st)
		mu.Unlock()
	}
	out := lines.NewLines()
	gd, err := NewGiDelve(filepath.Join(dir, "prog"), dir, out, &pars)
	require.NoError(t, err)
	t.Cleanup(func() {
		// (detaching kills the delve process that this test started)
		require.Equal(t, dir, gd.cmd.Dir)
		gd.Detach(true)
	})
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		require.NotContains(t, stats, cdebug.Error, out.String())
		return slices.Contains(stats, cdebug.Ready)
	}, 2*time.Minute, 10*time.Millisecond, "delve is ready")
	assert.True(t, gd.IsActive())
	assert.True(t, gd.HasTasks())
	return gd, filepath.Join(dir, "main.go"), out
}

// TestDelve tests debugging a program: breakpoints, the state when
// stopped at one, variables, and stepping.
func TestDelve(t *testing.T) {
	gd, fpath, out := startDelve(t, cdebug.Exec)
	assert.Greater(t, gd.ProcessPid(), 0)
	assert.False(t, gd.LastModified().IsZero())
	assert.False(t, gd.AttachedToExistingProcess())

	var all cdebug.AllState
	all.BlankState()
	useLine, mainLine := lineOf(program, "use"), lineOf(program, "main")
	all.AddBreak(fpath, useLine)
	off := all.AddBreak(fpath, mainLine)
	off.On = false
	require.NoError(t, gd.UpdateBreaks(&all.Breaks))
	bks, err := gd.ListBreaks()
	require.NoError(t, err)
	bk, _ := cdebug.BreakByFile(bks, fpath, useLine)
	require.NotNil(t, bk, "%v", bks)
	assert.Nil(t, func() *cdebug.Break { b, _ := cdebug.BreakByFile(bks, fpath, mainLine); return b }(), "off")
	gb, err := gd.GetBreak(bk.ID)
	require.NoError(t, err)
	assert.Equal(t, useLine, gb.Line)
	_, err = gd.GetBreak(1000)
	assert.Error(t, err)
	ub, _ := all.BreakByFile(fpath, useLine)
	ub.Cond = "t.N > 0"
	require.NoError(t, gd.UpdateBreaks(&all.Breaks), "amended with the condition")
	gb, _ = gd.GetBreak(bk.ID)
	assert.Equal(t, "t.N > 0", gb.Cond)

	var ds *cdebug.State
	for ds = range gd.Continue(&all) {
	}
	require.NotNil(t, ds)
	assert.False(t, ds.Exited)
	assert.Equal(t, useLine, ds.Task.Line)
	all.State = *ds
	require.NoError(t, gd.InitAllState(&all))
	assert.Equal(t, bk.ID, all.CurBreak)
	assert.NotEmpty(t, all.Threads)
	assert.NotEmpty(t, all.Tasks)
	require.NotEmpty(t, all.Stack)
	assert.Equal(t, "main.use", all.Stack[0].Func)
	assert.Equal(t, "main.main", all.Stack[1].Func)
	assert.Equal(t, all.CurTask, gd.CurThreadID(&all))

	// variables
	vr := all.VarByName("t")
	require.NotNil(t, vr, "%v", all.Vars)
	assert.Equal(t, "*main.T", vr.FullTypeStr)
	sv := all.VarByName("s")
	require.NotNil(t, sv)
	assert.Equal(t, int64(3), sv.Len)
	assert.Equal(t, []string{"1", "2", "3"}, sv.List)
	mv := all.VarByName("m")
	require.NotNil(t, mv)
	assert.Equal(t, map[string]string{"a": "1"}, mv.Map)
	tn, err := gd.GetVar("t.Name", all.CurTask, 0)
	require.NoError(t, err)
	assert.Equal(t, "t", tn.Value)
	tv, err := gd.GetVar("t", all.CurTask, 0)
	require.NoError(t, err)
	assert.Equal(t, "*main.T", tv.FullTypeStr)
	assert.NoError(t, gd.FollowPtr(tv))
	_, err = gd.GetVar("none", all.CurTask, 0)
	assert.Error(t, err)
	gvs, err := gd.ListGlobalVars("main.global")
	require.NoError(t, err)
	require.Len(t, gvs, 1)
	gv, err := gd.GetVar("main.global.N", all.CurTask, 0)
	require.NoError(t, err)
	assert.Equal(t, "1", gv.Value)
	require.NoError(t, gd.SetVar("t.N", "5", all.CurTask, 0))
	nv, _ := gd.GetVar("t.N", all.CurTask, 0)
	assert.Equal(t, "5", nv.Value)

	// the state of the frame of main, and of the task
	require.NoError(t, gd.UpdateAllState(&all, all.CurTask, 1))
	assert.NotNil(t, all.VarByName("s"))
	require.NoError(t, gd.UpdateAllState(&all, all.Tasks[len(all.Tasks)-1].ID, 0))
	fr, err := gd.FindFrames(&all, "main.go", mainLine)
	require.NoError(t, err)
	assert.NotEmpty(t, fr)
	th, err := gd.GetThread(all.State.Thread.ID)
	require.NoError(t, err)
	assert.Equal(t, all.State.Thread.ID, th.ID)
	_, err = gd.SwitchThread(all.State.Thread.ID)
	assert.NoError(t, err)
	_, err = gd.SwitchTask(all.State.Task.ID)
	assert.NoError(t, err)
	st, err := gd.GetState()
	require.NoError(t, err)
	assert.Equal(t, useLine, st.Task.Line)

	srcs, err := gd.ListSources("main.go")
	require.NoError(t, err)
	assert.Contains(t, srcs, fpath)
	fns, err := gd.ListFuncs("main.use")
	require.NoError(t, err)
	assert.Contains(t, fns, "main.use")
	tys, err := gd.ListTypes("main.T")
	require.NoError(t, err)
	assert.Contains(t, tys, "main.T")

	// stepping
	ds, err = gd.StepOver()
	require.NoError(t, err)
	assert.Equal(t, useLine+1, ds.Task.Line)
	ds, err = gd.StepOut()
	require.NoError(t, err)
	assert.Equal(t, "main.main", ds.Task.Func)
	_, err = gd.StepInto()
	assert.NoError(t, err)
	gd.StepSingle() // (an error from delve on some platforms)
	assert.NoError(t, gd.CancelNext())

	require.NoError(t, gd.ClearBreak(bk.ID))
	_, err = gd.ClearBreakByName("none")
	assert.Error(t, err)
	nb, err := gd.SetBreak(fpath, useLine)
	require.NoError(t, err)
	_, err = gd.GetBreakByName("none")
	assert.Error(t, err)
	require.NoError(t, gd.AmendBreak(nb.ID, fpath, useLine, "", true))

	for ds = range gd.Continue(&all) {
	}
	assert.True(t, ds.Exited, "the tracepoint does not stop")
	assert.Eventually(t, func() bool { return strings.Contains(out.String(), "result 9") }, 10*time.Second, 10*time.Millisecond, out.String())

	require.NoError(t, gd.Restart())
	require.Equal(t, filepath.Dir(fpath), gd.cmd.Dir, "the delve process of this test")
	require.NoError(t, gd.Detach(true))
	assert.False(t, gd.IsActive())
	assert.Error(t, gd.StartedCheck())
	assert.Nil(t, gd.Continue(&all))
	_, err = gd.StepOver()
	assert.Error(t, err)
	assert.Equal(t, -1, gd.ProcessPid())
}

// TestDelveTest tests debugging a test of a program.
func TestDelveTest(t *testing.T) {
	gd, fpath, _ := startDelve(t, cdebug.Test)
	var all cdebug.AllState
	all.BlankState()
	testLine := lineOf(programTest, "test")
	all.AddBreak(filepath.Join(filepath.Dir(fpath), "main_test.go"), testLine)
	require.NoError(t, gd.UpdateBreaks(&all.Breaks))
	var ds *cdebug.State
	for ds = range gd.Continue(&all) {
	}
	require.NotNil(t, ds)
	assert.Equal(t, testLine, ds.Task.Line)
	assert.Equal(t, "prog.TestUse", ds.Task.Func)
	require.NoError(t, gd.Disconnect(false))
}
