// Copyright (c) 2020, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidebug

import (
	"path/filepath"
	"sort"
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
	ID    int    `desc:"thread identifier"`
	PC    uint64 `format:"%X" desc:"program counter (address) -- may be subset of multiple"`
	File  string `desc:"file name (trimmed up to point of project base path)"`
	Line  int    `desc:"line within file"`
	FPath string `tableview:"-" tableview:"-" desc:"full path to file"`
	Func  string `desc:"the name of the function"`
	Task  int    `desc:"id of the current Task within this system thread (if relevant)"`
}

// Task is an optional finer-grained, lighter-weight thread, e.g.,
// a goroutine in the Go language.  if GiDebug HasTasks() == false then
// it is not used.
type Task struct {
	ID        int      `desc:"task identifier"`
	PC        uint64   `format:"%X" desc:"program counter (address) -- may be subset of multiple"`
	File      string   `desc:"file name (trimmed up to point of project base path)"`
	Line      int      `desc:"line within file"`
	FPath     string   `tableview:"-" tableview:"-" desc:"full path to file"`
	Func      string   `desc:"the name of the function"`
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
	PC    uint64      `format:"%X" desc:"program counter (address) -- may be subset of multiple"`
	File  string      `desc:"file name (trimmed up to point of project base path)"`
	Line  int         `desc:"line within file"`
	FPath string      `tableview:"-" tableview:"-" desc:"full path to file"`
	Func  string      `desc:"the name of the function"`
	Vars  []*Variable `tableview:"-" desc:"values of the local variables at this frame"`
	Args  []*Variable `tableview:"-" desc:"values of the local function args at this frame"`
}

// Break describes one breakpoint
type Break struct {
	ID    int    `inactive:"+" desc:"unique numerical ID of the breakpoint"`
	On    bool   `desc:"whether the breakpoint is currently enabled"`
	PC    uint64 `inactive:"+" format:"%X" desc:"program counter (address) -- may be subset of multiple"`
	File  string `inactive:"+" desc:"file name (trimmed up to point of project base path)"`
	Line  int    `inactive:"+" desc:"line within file"`
	FPath string `inactive:"+" view:"-" tableview:"-" desc:"full path to file"`
	Func  string `inactive:"+" desc:"the name of the function"`
	Cond  string `desc:"condition for conditional breakbpoint"`
	Trace bool   `desc:"if true, execution does not stop -- just a message is reported when this point is hit"`
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

// Variable describes a variable.
type Variable struct {
	Name  string      `desc:"name of variable"`
	Type  string      `desc:"type of variable"`
	Value string      `desc:"value of variable -- may be truncated if long"`
	Len   int64       `desc:"length of variable (slices, maps, strings etc)"`
	Cap   int64       `tableview:"-" desc:"capacity of vaiable"`
	Addr  uintptr     `desc:"address where variable is located in memory"`
	Heap  bool        `desc:"if true, the variable is stored in the main memory heap, not the stack"`
	Els   []*Variable `tableview:"-" desc:"elements of compount variables (struct fields, list / map elements)"`
	Loc   Location    `tableview:"-" desc:"location where the variable was defined in source"`
}

// SortVars sorts vars by name
func SortVars(vrs []*Variable) {
	sort.Slice(vrs, func(i, j int) bool {
		return vrs[i].Name < vrs[j].Name
	})
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
	Breaks    []*Break    `desc:"all breakpoints that have been set -- some may not be On"`
	CurBreaks []*Break    `desc:"current, active breakpoints as retrieved from debugger"`
	Threads   []*Thread   `desc:"all system threads"`
	Tasks     []*Task     `desc:"all tasks"`
	Stack     []*Frame    `desc:"current stack frame for current thread / task"`
	Vars      []*Variable `desc:"current local variables and args for current frame"`
	AllVars   []*Variable `desc:"all variables for current thread / task"`
}

// BlankState initializes state with a blank initial state with the various slices
// having a single entry -- for GUI initialization.
func (as *AllState) BlankState() {
	as.Breaks = []*Break{{ID: 0}}
	as.Threads = []*Thread{{ID: 0}}
	as.Tasks = []*Task{{ID: 0}}
	as.Stack = []*Frame{{Depth: 0}}
	as.Vars = []*Variable{{Name: ""}}
	as.AllVars = []*Variable{{Name: ""}}
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
	br.File = DirAndFile(fpath)
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
	MaxVariableRecurse: 5,
	MaxStringLen:       200,
	MaxArrayValues:     100,
	MaxStructFields:    -1,
}

// DirAndFile returns the final dir and file name.
func DirAndFile(file string) string {
	dir, fnm := filepath.Split(file)
	return filepath.Join(filepath.Base(dir), fnm)
}

// RelFile returns the file name relative to given root file path, if it is
// under that root -- otherwise it returns the final dir and file name.
func RelFile(file, root string) string {
	rp, err := filepath.Rel(root, file)
	if err == nil && !strings.HasPrefix(rp, "..") {
		return rp
	}
	return DirAndFile(file)
}
