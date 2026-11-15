// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package code

import (
	"os/exec"
	"slices"
	"testing"
	"time"

	"cogentcore.org/cogent/code/cdebug"
	_ "cogentcore.org/cogent/code/cdebug/cdelve" // the debugger of Go
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// debugProgram is the program that is debugged in the tests.
const debugProgram = `package main

import "fmt"

var global = 3

func add(a, b int) int {
	c := a + b
	return c
}

func main() {
	x := add(1, 2)
	fmt.Println("x", x, global)
}
`

// TestCodeDebug tests debugging a program with delve: stopping at a
// breakpoint, looking at its variables and stack, stepping, and
// continuing to the end.
func TestCodeDebug(t *testing.T) {
	if _, err := exec.LookPath("dlv"); err != nil {
		t.Skip("dlv not found in PATH:", err)
	}
	t.Setenv("GOWORK", "off")
	tmp := tempDir(t)
	path := writeFiles(t, tmp, map[string]string{
		"go.mod":  "module prog\n\ngo 1.21\n",
		"main.go": debugProgram,
	})
	b, cv := newTestCode(t, tmp)
	cv.Settings.RunExec = core.Filename(path("prog"))
	cv.Settings.MainLang = fileinfo.Go
	core.AssertRender(t, b, "debug", func() {
		var dv *DebugPanel
		status := func() cdebug.Status {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return dv.State.Status
		}
		locked(b, func() {
			cv.ViewFileInIndex(path("main.go"), 0)
			cv.Debug()
			dv = cv.CurDebug()
			require.NotNil(t, dv)
			dv.AddBreak(path("main.go"), 8) // c := a + b
			assert.Len(t, dv.State.Breaks, 1)
		})
		waitCode(t, 2*time.Minute, "the debugger ready", func() bool { return status() == cdebug.Ready })

		go dv.Continue()
		waitCode(t, time.Minute, "stopped at the breakpoint", func() bool { return status() == cdebug.Breakpoint })
		locked(b, func() {
			assert.True(t, dv.DbgIsAvail())
			assert.True(t, dv.DbgCanStep())
			cf := dv.State.StackFrame(dv.State.CurFrame)
			require.NotNil(t, cf)
			assert.Equal(t, 8, cf.Line)
			assert.Equal(t, "add", dv.State.Stack[0].Func[len(dv.State.Stack[0].Func)-3:])
			assert.Equal(t, "1", dv.VarValue("a"))
			assert.Equal(t, "2", dv.VarValue("b"))
			assert.Equal(t, "", dv.VarValue("none"))
			assert.Equal(t, "3", dv.VarValue("main.global"))
			assert.Equal(t, cv.ActiveLines().Filename(), path("main.go"), "the file is shown")

			dv.ListGlobalVars("main.")
			gi := slices.IndexFunc(dv.State.GlobalVars, func(v *cdebug.Variable) bool { return v.Name == "main.global" })
			require.GreaterOrEqual(t, gi, 0, "main.global in %v", dv.State.GlobalVars)
			assert.Equal(t, "3", dv.State.GlobalVars[gi].Value)
			// double-clicking a global variable shows it
			globals := dv.Tabs().TabByName(DebugTabGlobals).Child(0).(*core.Table)
			globals.SelectedIndex = gi
			globals.Send(events.DoubleClick)
			assert.NotNil(t, windowBody("Var View: main.global"), "a window for the variable")
			dv.FindFrames("main.go", 13)
			assert.NotEmpty(t, dv.State.FindFrames, "main is calling")
			dv.FindFrames("none.go", 1) // a snackbar
			dv.SetFrame(1)              // main
			assert.Equal(t, 1, dv.State.CurFrame)
			dv.SetFrame(0)
			dv.SetThreadIndex(0)
			dv.SetThreadIndex(-1)
			dv.SetThreadIndex(1000)
			dv.SetThread(dv.State.CurTask)
			require.NoError(t, dv.ShowVar("a")) // a dialog
			assert.Error(t, dv.ShowVar("none"))
			dv.ShowBreakFile(0)
			dv.ShowBreakFile(5)
			dv.UpdateView()

			dv.StepOver()
			assert.Equal(t, 9, dv.State.StackFrame(dv.State.CurFrame).Line)
			assert.Equal(t, "3", dv.VarValue("c"))
			assert.Equal(t, 9, dv.CurFileLoc.Line, "the current line is marked")
			dv.StepOut()
			assert.Equal(t, "main.main", dv.State.StackFrame(dv.State.CurFrame).Func)
			dv.StepInto()
			dv.SingleStep()

			// the breakpoints and variables in the editor
			tv := cv.EditorByIndex(0)
			cv.ViewFileInIndex(path("main.go"), 0)
			assert.False(t, tv.HasBreakpoint(12))
			tv.LineNumberDoubleClick(textpos.Pos{Line: 12})
			assert.True(t, tv.HasBreakpoint(12))
			_, bi := dv.State.BreakByFile(path("main.go"), 13)
			assert.GreaterOrEqual(t, bi, 0, "a break in the debugger")
			tv.ToggleBreakpoint(12)
			assert.False(t, tv.HasBreakpoint(12))
			tv.FindFrames(12)
			dbg, has := tv.CurDebug()
			assert.True(t, has)
			assert.Same(t, dv, dbg)

			dv.AddBreak(path("main.go"), 14)
			dv.DeleteBreak(path("main.go"), 14)
			assert.Len(t, dv.State.Breaks, 1)
		})

		go dv.Continue()
		waitCode(t, time.Minute, "the program finished", func() bool { return status() == cdebug.Finished })
		locked(b, func() {
			assert.Contains(t, dv.OutputBuffer.String(), "x 3 3", "the output of the program")
			dv.DeleteBreakIndex(0)
			dv.DeleteBreakIndex(5)
			// (stopping and detaching kill the delve process of this test)
			require.Equal(t, path("prog"), dv.ExePath)
			dv.Stop()
			dv.Detach()
			assert.False(t, dv.DbgIsActive())
		})
	})
}
