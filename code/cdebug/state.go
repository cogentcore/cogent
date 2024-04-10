// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cdebug

import (
	"fmt"
	"sort"
	"strings"

	"cogentcore.org/core/gox/dirs"
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

	// thread identifier
	ID int `format:"%#X"`

	// program counter (address) -- may be subset of multiple
	PC uint64 `format:"%#X"`

	// file name (trimmed up to point of project base path)
	File string

	// line within file
	Line int

	// full path to file
	FPath string `tableview:"-"`

	// the name of the function
	Func string `width:"80"`

	// id of the current Task within this system thread (if relevant)
	Task int
}

// ThreadByID returns the given thread by ID from full list, and index.
// returns nil, -1 if not found.
func ThreadByID(thrs []*Thread, id int) (*Thread, int) {
	for i, thr := range thrs {
		if thr.ID == id {
			return thr, i
		}
	}
	return nil, -1
}

func (th *Thread) String() string {
	return fmt.Sprintf("id: %v  %v:%v", th.ID, th.File, th.Line)
}

// Task is an optional finer-grained, lighter-weight thread, e.g.,
// a goroutine in the Go language.  if GiDebug HasTasks() == false then
// it is not used.
type Task struct {

	// task identifier
	ID int

	// program counter (address) -- may be subset of multiple
	PC uint64 `format:"%#X"`

	// file name (trimmed up to point of project base path)
	File string

	// line within file
	Line int

	// full path to file
	FPath string `tableview:"-" tableview:"-"`

	// the name of the function
	Func string `width:"80"`

	// id of the current Thread this task is running on
	Thread int `format:"%#X"`

	// where did this task first start running?
	StartLoc Location `tableview:"-"`

	// at what point was this task launched from another task?
	LaunchLoc Location `tableview:"-"`
}

// TaskByID returns the given thread by ID from full list, and index.
// returns nil, -1 if not found.
func TaskByID(thrs []*Task, id int) (*Task, int) {
	for i, thr := range thrs {
		if thr.ID == id {
			return thr, i
		}
	}
	return nil, -1
}

func (th *Task) String() string {
	return fmt.Sprintf("id: %v  %v:%v", th.ID, th.File, th.Line)
}

// Location holds program location information.
type Location struct {

	// program counter (address) -- may be subset of multiple
	PC uint64 `format:"%#X"`

	// file name (trimmed up to point of project base path)
	File string

	// line within file
	Line int

	// full path to file
	FPath string `view:"-" tableview:"-"`

	// the name of the function
	Func string `width:"80"`
}

// Frame describes one frame in a stack trace.
type Frame struct {

	// depth in overall stack -- 0 is the bottom (currently executing) frame, and it counts up from there
	Depth int

	// the Task or Thread id that this frame belongs to
	ThreadID int

	// program counter (address) -- may be subset of multiple
	PC uint64 `format:"%#X"`

	// file name (trimmed up to point of project base path)
	File string

	// line within file
	Line int

	// full path to file
	FPath string `tableview:"-" tableview:"-"`

	// the name of the function
	Func string `width:"80"`

	// values of the local variables at this frame
	Vars []*Variable `tableview:"-"`

	// values of the local function args at this frame
	Args []*Variable `tableview:"-"`
}

// Break describes one breakpoint
type Break struct {

	// unique numerical ID of the breakpoint
	ID int `edit:"-"`

	// whether the breakpoint is currently enabled
	On bool `width:"4"`

	// program counter (address) -- may be subset of multiple
	PC uint64 `edit:"-" format:"%#X"`

	// file name (trimmed up to point of project base path)
	File string `edit:"-"`

	// line within file
	Line int `edit:"-"`

	// full path to file
	FPath string `edit:"-" view:"-" tableview:"-"`

	// the name of the function
	Func string `edit:"-" width:"80"`

	// condition for conditional breakbpoint
	Cond string

	// if true, execution does not stop -- just a message is reported when this point is hit
	Trace bool `width:"7"`
}

// BreakByID returns the given breakpoint by ID from full list, and index.
// returns nil, -1 if not found.
func BreakByID(bks []*Break, id int) (*Break, int) {
	for i, br := range bks {
		if br.ID == id {
			return br, i
		}
	}
	return nil, -1
}

// BreakByFile returns the given breakpoint by file path and line from list.
// returns nil, -1 if not found.
func BreakByFile(bks []*Break, fpath string, line int) (*Break, int) {
	for i, br := range bks {
		if br.FPath == fpath && br.Line == line {
			return br, i
		}
	}
	return nil, -1
}

// SortBreaks sorts breaks by id
func SortBreaks(brk []*Break) {
	sort.Slice(brk, func(i, j int) bool {
		return brk[i].ID < brk[j].ID
	})
}

// State represents the current immediate execution state of the debugger.
type State struct {

	// currently executing system thread
	Thread Thread

	// currently executing task
	Task Task

	// true if the process is running and no other information can be collected.
	Running bool

	// if true, a Next or Step is already in progress and another should not be attempted until after a Continue
	NextUp bool

	// if true, the program has exited
	Exited bool

	// indicates the exit status if Exited
	ExitStatus int

	// error communicated to client -- if non-empty, something bad happened
	Err error

	// if this is > 0, then we just hit that tracepoint -- the Continue process will continue execution
	CurTrace int
}

func (st *State) String() string {
	return fmt.Sprintf("th: %s  ta: %s  Run: %v", st.Thread.String(), st.Task.String(), st.Running)
}

// AllState holds all relevant state information.
// This can be maintained and updated in the debug view.
type AllState struct {

	// mode we're running in
	Mode Modes

	// overall debugger status
	Status Status

	// current run state
	State State

	// id of the current system thread to examine
	CurThread int

	// id of the current task to examine
	CurTask int

	// frame number within current thread
	CurFrame int

	// current breakpoint that we stopped at -- will be 0 if none, after UpdateState
	CurBreak int

	// all breakpoints that have been set -- some may not be On
	Breaks []*Break

	// current, active breakpoints as retrieved from debugger
	CurBreaks []*Break

	// all system threads
	Threads []*Thread

	// all tasks
	Tasks []*Task

	// current stack frame for current thread / task
	Stack []*Frame

	// current local variables and args for current frame
	Vars []*Variable

	// global variables for current thread / task
	GlobalVars []*Variable

	// current find-frames result
	FindFrames []*Frame
}

// BlankState initializes state with a blank initial state with the various slices
// having a single entry -- for GUI initialization.
func (as *AllState) BlankState() {
	as.Status = NotInit
	as.Breaks = []*Break{}
	as.Threads = []*Thread{}
	as.Tasks = []*Task{}
	as.Stack = []*Frame{}
	as.Vars = []*Variable{}
	as.GlobalVars = []*Variable{}
	as.FindFrames = []*Frame{}
}

// StackFrame safely returns the given stack frame -- nil if out of range
func (as *AllState) StackFrame(idx int) *Frame {
	if idx < 0 || idx >= len(as.Stack) {
		return nil
	}
	return as.Stack[idx]
}

// BreakByID returns the given breakpoint by ID from full list, and index.
// returns nil, -1 if not found.
func (as *AllState) BreakByID(id int) (*Break, int) {
	return BreakByID(as.Breaks, id)
}

// BreakByFile returns the given breakpoint by file path and line from list.
// returns nil, -1 if not found.
func (as *AllState) BreakByFile(fpath string, line int) (*Break, int) {
	return BreakByFile(as.Breaks, fpath, line)
}

// AddBreak adds file path and line to full list of breaks.
// checks for an existing and turns it on if so.
func (as *AllState) AddBreak(fpath string, line int) *Break {
	br, _ := as.BreakByFile(fpath, line)
	if br != nil {
		br.On = true
		return br
	}
	br = &Break{}
	br.On = true
	br.FPath = fpath
	br.File = dirs.DirAndFile(fpath)
	br.Line = line
	as.Breaks = append(as.Breaks, br)
	return br
}

// DeleteBreakByID deletes given break by ID from full list.
// Returns true if deleted.
func (as *AllState) DeleteBreakByID(id int) bool {
	for i, br := range as.Breaks {
		if br.ID == id {
			as.Breaks = append(as.Breaks[:i], as.Breaks[i+1:]...)
			return true
		}
	}
	return false
}

// DeleteBreakByFile deletes given break by File and Line from full list.
// Returns true if deleted.
func (as *AllState) DeleteBreakByFile(fpath string, line int) bool {
	for i, br := range as.Breaks {
		if br.FPath == fpath && br.Line == line {
			as.Breaks = append(as.Breaks[:i], as.Breaks[i+1:]...)
			return true
		}
	}
	return false
}

// MergeBreaks merges the current breaks with AllBreaks -- any not in
// Cur are indicated as !On
func (as *AllState) MergeBreaks() {
	for _, br := range as.Breaks {
		br.On = false
	}
	for _, br := range as.CurBreaks {
		if br.ID <= 0 {
			continue
		}
		ab, _ := as.BreakByID(br.ID)
		if ab == nil {
			br.On = true
			as.Breaks = append(as.Breaks, br)
		} else {
			*ab = *br
			ab.On = true
		}
	}
	SortBreaks(as.Breaks)
}

// VarByName returns variable with the given name, or nil if not found
func (as *AllState) VarByName(varNm string) *Variable {
	for _, vr := range as.Vars {
		if vr.Nm == varNm {
			return vr
		}
	}
	nmspl := strings.Split(varNm, ".")
	for _, vr := range as.GlobalVars {
		spl := strings.Split(vr.Nm, ".")
		if len(spl) == len(nmspl) && vr.Nm == varNm {
			return vr
		}
		if len(nmspl) == 1 && len(spl) == 2 && spl[1] == varNm {
			return vr
		}
	}
	return nil
}

// Status of the debugger
type Status int32 //enums:enum

const (
	// NotInit is not initialized
	NotInit Status = iota

	// Error means the debugger has an error -- usually from building
	Error

	// Building is building the exe for debugging
	Building

	// Ready means executable is built and ready to start (or restarted)
	Ready

	// Running means the process is running
	Running

	// Stopped means the process has stopped
	// (at a breakpoint, crash, or from single stepping)
	Stopped

	// Breakpoint means the process has stopped at a breakpoint
	Breakpoint

	// Finished means the process has finished running.
	// See console for output and return value etc
	Finished
)
