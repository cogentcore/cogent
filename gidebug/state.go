// Copyright (c) 2020, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidebug

import (
	"path/filepath"
	"strings"
)

// This file contains all the state structs used in communciating with the
// debugger and returning values from it.

// Thread is a system-level thread within the debugged process.
// For many languages, this is synonymous with functional threads,
// but if HasTasks() is true from GiDebug, then there are separate
// Task constructs as well, which provide a finer-grained processing
// within the Thread. For example, go routines in Go are represented
// by Tasks in the GiDebug framework.
type Thread struct {
	ID   int      `desc:"thread identifier"`
	Loc  Location `desc:"where is the thread currently located?"`
	Task int      `desc:"id of the current Task within this system thread (if relevant)"`
}

// Task is an optional finer-grained, lighter-weight thread, e.g.,
// a goroutine in the Go language.  if GiDebug HasTasks() == false then
// it is not used.
type Task struct {
	ID        int      `desc:"task identifier"`
	Loc       Location `desc:"where is the task currently located?"`
	Thread    int      `desc:"id of the current Thread this task is running on"`
	StartLoc  Location `tableview:"-" desc:"where did this task first start running?"`
	LaunchLoc Location `tableview:"-" desc:"at what point was this task launched from another task?"`
}

// Location holds program location information.
type Location struct {
	PC    uint64 `format:"%X" desc:"program counter (address) -- may be subset of multiple"`
	File  string `desc:"file name (trimmed up to point of project base path)"`
	Line  int    `desc:"line within file"`
	FPath string `view:"-" tableview:"-" desc:"full path to file"`
	Func  string `desc:"the name of the function"`
}

// Frame describes one frame in a stack trace.
type Frame struct {
	Depth int         `desc:"depth in overall stack -- 0 is the bottom (currently executing) frame, and it counts up from there"`
	Loc   Location    `desc:"execution location"`
	Vars  []*Variable `tableview:"-" desc:"values of the local variables at this frame"`
	Args  []*Variable `tableview:"-" desc:"values of the local function args at this frame"`
}

// Break describes one breakpoint
type Break struct {
	ID    int      `desc:"unique numerical ID of the breakpoint"`
	Loc   Location `desc:"location of the breakpoint"`
	Cond  string   `desc:"condition for conditional breakbpoint"`
	Trace bool     `desc:"if true, execution does not stop -- just a message is reported when this point is hit"`
}

// Variable describes a variable.
type Variable struct {
	Name  string      `desc:"name of variable"`
	Addr  uintptr     `desc:"address where variable is located in memory"`
	Type  string      `desc:"type of variable"`
	Value string      `desc:"value of variable -- may be truncated if long"`
	Len   int64       `desc:"length of variable (slices, maps, strings etc)"`
	Cap   int64       `tableview:"-" desc:"capacity of vaiable"`
	Heap  bool        `desc:"if true, the variable is stored in the main memory heap, not the stack"`
	Els   []*Variable `tableview:"-" desc:"elements of compount variables (struct fields, list / map elements)"`
	Loc   Location    `tableview:"-" desc:"location where the variable was defined in source"`
}

// State represents the current immediate execution state of the debugger.
type State struct {
	Thread     Thread `desc:"currently executing system thread"`
	Task       Task   `desc:"currently executing task"`
	Running    bool   `desc:"true if the process is running and no other information can be collected."`
	NextUp     bool   `desc:"if true, a Next or Step is already in progress and another should not be attempted until after a Continue"`
	Exited     bool   `desc:"if true, the program has exited"`
	ExitStatus int    `desc:"indicates the exit status if Exited"`
	Err        error  `desc:"error communicated to client -- if non-empty, something bad happened"`
}

// AllState holds all relevant state information.
// This can be maintained and updated in the debug view.
type AllState struct {
	State     State       `desc:"current run state"`
	CurThread int         `desc:"id of the current system thread to examine"`
	CurTask   int         `desc:"id of the current task to examine"`
	CurFrame  int         `desc:"frame number within current thread"`
	Breaks    []*Break    `desc:"current breakpoints"`
	Threads   []*Thread   `desc:"all system threads"`
	Tasks     []*Task     `desc:"all tasks"`
	Stack     []*Frame    `desc:"current stack frame for current thread / task"`
	Vars      []*Variable `desc:"current local variables for current frame"`
	Args      []*Variable `desc:"current args for current frame"`
	AllVars   []*Variable `desc:"all variables for current thread / task"`
}

// CurStackFrame safely returns the current stack frame
// based on CurFrame value -- nil if out of range.
func (as *AllState) CurStackFrame() *Frame {
	if as.CurFrame < 0 || as.CurFrame > len(as.Stack) {
		return nil
	}
	return as.Stack[as.CurFrame]
}

// Params are parameters controlling the behavior of the debugger
type Params struct {
	FollowPointers     bool `desc:"requests pointers to be automatically dereferenced."`
	MaxVariableRecurse int  `desc:"how far to recurse when evaluating nested types."`
	MaxStringLen       int  `desc:"the maximum number of bytes read from a string"`
	MaxArrayValues     int  `desc:"the maximum number of elements read from an array, a slice or a map."`
	MaxStructFields    int  `desc:"the maximum number of fields read from a struct, -1 will read all fields."`
}

// DefaultParams are default parameter values
var DefaultParams = Params{
	FollowPointers:     true,
	MaxVariableRecurse: 2,
	MaxStringLen:       200,
	MaxArrayValues:     100,
	MaxStructFields:    -1,
}

// RelFile returns the file name relative to given root file path, if it is
// under that root -- otherwise it returns the final dir and file name.
func RelFile(file, root string) string {
	rp, err := filepath.Rel(root, file)
	if err == nil && !strings.HasPrefix(rp, "..") {
		return rp
	}
	dir, fnm := filepath.Split(file)
	return filepath.Join(filepath.Base(dir), fnm)
}
