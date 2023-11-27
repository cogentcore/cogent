// Copyright (c) 2020, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"fmt"
	"log"
	"strings"
	"time"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/gide/v2/gidebug"
	"goki.dev/gide/v2/gidebug/gidelve"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/grr"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"goki.dev/pi/v2/filecat"
)

// DebugBreakStatus is the status of a given breakpoint
type DebugBreakStatus int32

const (
	// DebugBreakInactive is an inactive break point
	DebugBreakInactive DebugBreakStatus = iota

	// DebugBreakActive is an active break point
	DebugBreakActive

	// DebugBreakCurrent is the current break point
	DebugBreakCurrent

	// DebugPCCurrent is the current program execution point,
	// updated for every ShowFile action
	DebugPCCurrent

	DebugBreakStatusN
)

// DebugBreakColors are the colors indicating different breakpoint statuses
var DebugBreakColors = [DebugBreakStatusN]string{"pink", "red", "orange", "lightblue"}

// Debuggers is the list of supported debuggers
var Debuggers = map[filecat.Supported]func(path, rootPath string, outbuf *texteditor.Buf, pars *gidebug.Params) (gidebug.GiDebug, error){
	filecat.Go: func(path, rootPath string, outbuf *texteditor.Buf, pars *gidebug.Params) (gidebug.GiDebug, error) {
		return gidelve.NewGiDelve(path, rootPath, outbuf, pars)
	},
}

// NewDebugger returns a new debugger for given supported file type
func NewDebugger(sup filecat.Supported, path, rootPath string, outbuf *texteditor.Buf, pars *gidebug.Params) (gidebug.GiDebug, error) {
	df, ok := Debuggers[sup]
	if !ok {
		err := fmt.Errorf("Gi Debug: File type %v not supported -- change the MainLang in File/Project Prefs.. to a supported language (Go only option so far)", sup)
		log.Println(err)
		return nil, err
	}
	dbg, err := df(path, rootPath, outbuf, pars)
	if err != nil {
		log.Println(err)
	}
	return dbg, err
}

// DebugView is the debugger
type DebugView struct {
	gi.Layout

	// supported file type to determine debugger
	Sup filecat.Supported

	// path to executable / dir to debug
	ExePath string

	// time when dbg was last restarted
	DbgTime time.Time

	// the debugger
	Dbg gidebug.GiDebug `set:"-" json:"-" xml:"-"`

	// all relevant debug state info
	State gidebug.AllState `set:"-" json:"-" xml:"-"`

	// current ShowFile location -- cleared before next one or run
	CurFileLoc gidebug.Location `set:"-" json:"-" xml:"-"`

	// backup breakpoints list -- to track deletes
	BBreaks []*gidebug.Break `set:"-" json:"-" xml:"-"`

	// output from the debugger
	OutBuf *texteditor.Buf `set:"-" json:"-" xml:"-"`

	// parent gide project
	Gide Gide `set:"-" json:"-" xml:"-"`
}

// DbgIsActive means debugger is started.
func (dv *DebugView) DbgIsActive() bool {
	if dv.Dbg != nil && dv.Dbg.IsActive() {
		return true
	}
	return false
}

// DbgIsAvail means the debugger is started AND process is not currently running --
// it is available for command input.
func (dv *DebugView) DbgIsAvail() bool {
	if !dv.DbgIsActive() {
		return false
	}
	if dv.State.State.Running {
		return false
	}
	return true
}

// DbgCanStep means the debugger is started AND process is not currently running,
// AND it is not already waiting for a next step
func (dv *DebugView) DbgCanStep() bool {
	if !dv.DbgIsAvail() {
		return false
	}
	if dv.State.State.NextUp {
		return false
	}
	return true
}

func (dv *DebugView) Destroy() {
	fmt.Println("debug destroy")
	dv.Detach()
	dv.DeleteAllBreaks()
	dv.DeleteCurPCInBuf()
	dv.Gide.ClearDebug()
	dv.Layout.Destroy()
}

// Detach from debugger
func (dv *DebugView) Detach() {
	killProc := true
	if dv.State.Mode == gidebug.Attach {
		killProc = false
	}
	if dv.DbgIsAvail() {
		dv.Dbg.Detach(killProc)
	} else if dv.DbgIsActive() {
		dv.Stop()
		dv.Dbg.Detach(killProc)
	}
	dv.Dbg = nil
}

// Start starts the debuger
func (dv *DebugView) Start() {
	if dv.Gide == nil {
		return
	}
	console := dv.ConsoleText()
	console.Clear()
	rebuild := false
	if dv.Dbg != nil && dv.State.Mode != gidebug.Attach {
		lmod := dv.Gide.FileTree().LatestFileMod(filecat.Code)
		rebuild = lmod.After(dv.DbgTime) || dv.Gide.LastSaveTime().After(dv.DbgTime)
	}
	if dv.Dbg == nil || rebuild {
		dv.SetStatus(gidebug.Building)
		if dv.Dbg != nil {
			dv.Detach()
		}
		rootPath := string(dv.Gide.ProjPrefs().ProjRoot)
		pars := &dv.Gide.ProjPrefs().Debug
		dv.State.Mode = pars.Mode
		pars.StatFunc = func(stat gidebug.Status) {
			if stat == gidebug.Ready && dv.State.Mode == gidebug.Attach {
				dv.UpdateFmState()
			}
			dv.SetStatus(stat)
			if stat == gidebug.Error {
				dv.Dbg = nil
			}
		}
		dbg, err := NewDebugger(dv.Sup, dv.ExePath, rootPath, dv.OutBuf, pars)
		if err == nil {
			dv.Dbg = dbg
			dv.DbgTime = time.Now()
		} else {
			dv.SetStatus(gidebug.Error)
		}
	} else {
		dv.Dbg.Restart()
		dv.SetStatus(gidebug.Ready)
	}
}

// UpdateView updates current view of state
func (dv *DebugView) UpdateView() {
	ds, err := dv.Dbg.GetState()
	if err != nil {
		return
	}
	dv.InitState(ds)
}

// Continue continues running from current point -- this MUST be called
// in a separate goroutine!
func (dv *DebugView) Continue() {
	if !dv.DbgIsAvail() {
		return
	}
	dv.SetBreaks()
	dv.State.State.Running = true
	dv.SetStatus(gidebug.Running)
	dsc := dv.Dbg.Continue(&dv.State)
	var ds *gidebug.State
	for ds = range dsc { // get everything
		if dv.This() == nil || dv.Is(ki.Deleted) {
			return
		}
	}
	if dv.Gide != nil {
		sc := dv.Gide.Scene()
		if sc != nil && sc.MainStageMgr() != nil {
			sc.MainStageMgr().RenderWin.Raise()
		}
	}
	if ds != nil {
		dv.InitState(ds)
	} else {
		dv.State.State.Running = false
		dv.SetStatus(gidebug.Finished)
	}
}

// StepOver continues to the next source line, not entering function calls.
func (dv *DebugView) StepOver() {
	if !dv.DbgCanStep() {
		return
	}
	dv.SetBreaks()
	ds, err := dv.Dbg.StepOver()
	if err != nil {
		return
	}
	dv.InitState(ds)
}

// StepInto continues to the next source line, entering function calls.
func (dv *DebugView) StepInto() {
	if !dv.DbgCanStep() {
		return
	}
	dv.SetBreaks()
	ds, err := dv.Dbg.StepInto()
	if err != nil {
		return
	}
	dv.InitState(ds)
}

// StepOut continues to the return point of the current function
func (dv *DebugView) StepOut() {
	if !dv.DbgCanStep() {
		return
	}
	dv.SetBreaks()
	ds, err := dv.Dbg.StepOut()
	if err != nil {
		return
	}
	dv.InitState(ds)
}

// StepSingle steps a single cpu instruction.
func (dv *DebugView) SingleStep() {
	if !dv.DbgCanStep() {
		return
	}
	dv.SetBreaks()
	ds, err := dv.Dbg.StepSingle()
	if err != nil {
		return
	}
	dv.InitState(ds)
}

// Stop stops a running process
func (dv *DebugView) Stop() {
	// if !dv.DbgIsActive() || dv.DbgIsAvail() {
	// 	return
	// }
	_, err := dv.Dbg.Stop()
	if err != nil {
		return
	}
	// note: it will auto update from continue stopping.
}

// SetBreaks sets the current breakpoints from State, call this prior to running
func (dv *DebugView) SetBreaks() {
	if !dv.DbgIsAvail() {
		return
	}
	dv.DeleteCurPCInBuf()
	dv.State.CurBreak = 0 // reset
	dv.Dbg.UpdateBreaks(&dv.State.Breaks)
	dv.UpdateAllBreaks()
	dv.ShowBreaks(false)
}

// AddBreak adds a breakpoint at given file path and line number.
// note: all breakpoints are just set in our master list and
// uploaded to the system right before starting running.
func (dv *DebugView) AddBreak(fpath string, line int) {
	dv.State.AddBreak(fpath, line)
	dv.ShowBreaks(true)
}

// DeleteBreak deletes given breakpoint.  If debugger is not yet
// activated then it just deletes from master list.
// Note that breakpoints can be turned on and off directly using On flag.
func (dv *DebugView) DeleteBreak(fpath string, line int) {
	if dv.Is(ki.Deleted) {
		return
	}
	dv.DeleteBreakImpl(fpath, line)
	dv.ShowBreaks(true)
}

// DeleteBreakImpl deletes given breakpoint with no other updates
func (dv *DebugView) DeleteBreakImpl(fpath string, line int) {
	if !dv.DbgIsAvail() {
		dv.State.DeleteBreakByFile(fpath, line) // already doing this!
		return
	}
	bk, _ := dv.State.BreakByFile(fpath, line)
	if bk != nil {
		dv.Dbg.ClearBreak(bk.ID)
		dv.State.DeleteBreakByID(bk.ID)
	}
}

// DeleteBreakIdx deletes break at given index in list of breaks
func (dv *DebugView) DeleteBreakIdx(bidx int) {
	if bidx < 0 || bidx >= len(dv.BBreaks) {
		return
	}
	bk := dv.BBreaks[bidx]
	dv.DeleteBreakInBuf(bk.FPath, bk.Line)
	if !dv.DbgIsAvail() {
		dv.State.DeleteBreakByFile(bk.FPath, bk.Line)
		return
	}
	dv.Dbg.ClearBreak(bk.ID)
	dv.State.DeleteBreakByID(bk.ID)
	dv.BackupBreaks()
}

// DeleteBreakInBuf delete breakpoint in its TextBuf
// line is 1-based line number
func (dv *DebugView) DeleteBreakInBuf(fpath string, line int) {
	if dv.Gide == nil || dv.Gide.Is(ki.Deleted) {
		return
	}
	tb := dv.Gide.TextBufForFile(fpath, false)
	if tb != nil {
		tb.DeleteLineColor(line - 1)
		tb.Update()
	}
}

// DeleteAllBreaks deletes all breakpoints
func (dv *DebugView) DeleteAllBreaks() {
	if dv.Gide == nil || dv.Gide.Is(ki.Deleted) {
		return
	}
	for _, bk := range dv.State.Breaks {
		dv.DeleteBreakInBuf(bk.FPath, bk.Line)
	}
}

// UpdateBreakInBuf updates break status in its TextBuf
// line is 1-based line number
func (dv *DebugView) UpdateBreakInBuf(fpath string, line int, stat DebugBreakStatus) {
	if dv.Gide == nil || dv.Gide.Is(ki.Deleted) {
		return
	}
	tb := dv.Gide.TextBufForFile(fpath, false)
	if tb != nil {
		tb.SetLineColor(line-1, grr.Log1(colors.FromName(DebugBreakColors[stat])))
		tb.Update()
	}
}

// UpdateAllBreaks updates all breakpoints
func (dv *DebugView) UpdateAllBreaks() {
	if dv.Gide == nil || dv.Gide.Is(ki.Deleted) {
		return
	}
	updt := dv.UpdateStart()
	dv.UpdateEndRender(updt)
	for _, bk := range dv.State.Breaks {
		if bk.ID == dv.State.CurBreak {
			dv.UpdateBreakInBuf(bk.FPath, bk.Line, DebugBreakCurrent)
		} else if bk.On {
			dv.UpdateBreakInBuf(bk.FPath, bk.Line, DebugBreakActive)
		} else {
			dv.UpdateBreakInBuf(bk.FPath, bk.Line, DebugBreakInactive)
		}
	}
}

// BackupBreaks makes a backup copy of current breaks
func (dv *DebugView) BackupBreaks() {
	dv.BBreaks = make([]*gidebug.Break, len(dv.State.Breaks))
	for i, b := range dv.State.Breaks {
		dv.BBreaks[i] = b
	}
}

// InitState updates the State and View from given debug state
// Call this when debugger returns from any action update
func (dv *DebugView) InitState(ds *gidebug.State) {
	dv.State.State = *ds
	if ds.Running {
		return
	}
	if ds.Exited {
		dv.SetStatus(gidebug.Finished)
	} else {
		dv.SetStatus(gidebug.Stopped)
	}
	err := dv.Dbg.InitAllState(&dv.State)
	if err == gidebug.IsRunningErr {
		dv.SetStatus(gidebug.Running)
		return
	}
	dv.UpdateFmState()
}

// UpdateFmState updates the view from current debugger state
func (dv *DebugView) UpdateFmState() {
	if dv == nil || dv.This() == nil || dv.Is(ki.Deleted) || dv.Dbg == nil {
		return
	}
	updt := dv.UpdateStart()
	defer dv.UpdateEndLayout(updt)

	cb, err := dv.Dbg.ListBreaks()
	if err == nil {
		dv.State.CurBreaks = cb
		dv.State.MergeBreaks()
	}
	cf := dv.State.StackFrame(dv.State.CurFrame)
	if cf != nil {
		dv.ShowFile(cf.FPath, cf.Line)
		if dv.State.CurBreak > 0 {
			dv.SetStatus(gidebug.Breakpoint)
		}
	}
	dv.UpdateAllBreaks()
	dv.ShowBreaks(false)
	dv.ShowStack(false)
	dv.ShowVars(false)
	dv.ShowThreads(false)
	if dv.Dbg.HasTasks() {
		dv.ShowTasks(false)
	}
	dv.UpdateToolbar()
	dv.Update()
}

// SetFrame sets the given frame depth level as active
func (dv *DebugView) SetFrame(depth int) {
	if !dv.DbgIsAvail() {
		return
	}
	cf := dv.State.StackFrame(depth)
	if cf != nil {
		dv.Dbg.UpdateAllState(&dv.State, dv.State.CurTask, depth) // todo: CurTask is not general!
	}
	dv.UpdateFmState()
}

// SetThread sets the given thread as active -- this must be TaskID if HasTasks
// and ThreadID if not.
func (dv *DebugView) SetThread(threadID int) {
	if !dv.DbgIsAvail() {
		return
	}
	dv.Dbg.UpdateAllState(&dv.State, threadID, 0)
	dv.UpdateFmState()
}

// SetThreadIdx sets the given thread by index in threads list as active
// this must be TaskID if HasTasks and ThreadID if not.
func (dv *DebugView) SetThreadIdx(thridx int) {
	if !dv.DbgIsAvail() || thridx < 0 {
		return
	}
	thid := 0
	if dv.Dbg.HasTasks() {
		if thridx >= len(dv.State.Tasks) {
			return
		}
		th := dv.State.Tasks[thridx]
		thid = th.ID
	} else {
		if thridx >= len(dv.State.Threads) {
			return
		}
		th := dv.State.Threads[thridx]
		thid = th.ID
	}
	dv.Dbg.UpdateAllState(&dv.State, thid, 0)
	dv.UpdateFmState()
}

// FindFrames finds the frames where given file and line are active
// Selects the one that is closest and shows the others in Find Tab
func (dv *DebugView) FindFrames(fpath string, line int) {
	if !dv.DbgIsAvail() {
		return
	}
	fr, err := dv.Dbg.FindFrames(&dv.State, fpath, line)
	if fr == nil || err != nil {
		gi.NewBody().AddTitle("No frames found").
			AddText(fmt.Sprintf("Could not find any stack frames for file name: %v, err: %v", fpath, err)).
			AddOkOnly().NewDialog(dv).Run()
	}
	dv.State.FindFrames = fr
	dv.ShowFindFrames(true)
}

// ListGlobalVars lists global vars matching given optional filter in Global Vars tab
func (dv *DebugView) ListGlobalVars(filter string) {
	if !dv.DbgIsAvail() {
		return
	}
	vrs, err := dv.Dbg.ListGlobalVars(filter)
	if err != nil {
		return
	}
	dv.State.GlobalVars = vrs
	dv.ShowGlobalVars(true)
}

// ShowFile shows the file name in gide
func (dv *DebugView) ShowFile(fpath string, line int) {
	if fpath == "" || fpath == "?" {
		return
	}
	// fmt.Printf("File: %s:%d\n", fpath, ln)
	updt := dv.UpdateStart()
	dv.UpdateEndRender(updt)

	dv.DeleteCurPCInBuf()
	dv.Gide.ShowFile(fpath, line)
	dv.SetCurPCInBuf(fpath, line)
}

// SetCurPCInBuf sets the current PC location in given file
// line is 1-based line number
func (dv *DebugView) SetCurPCInBuf(fpath string, line int) {
	tb := dv.Gide.TextBufForFile(fpath, false)
	if tb != nil {
		if !tb.HasLineColor(line - 1) {
			tb.SetLineColor(line-1, grr.Log1(colors.FromName(DebugBreakColors[DebugPCCurrent])))
			tb.Update()
			dv.CurFileLoc.FPath = fpath
			dv.CurFileLoc.Line = line
		}
	}
}

// DeleteCurPCInBuf deletes the current PC location in given file
// line is 1-based line number
func (dv *DebugView) DeleteCurPCInBuf() {
	fpath := dv.CurFileLoc.FPath
	line := dv.CurFileLoc.Line
	if fpath != "" && line > 0 {
		tb := dv.Gide.TextBufForFile(fpath, false)
		if tb != nil {
			tb.DeleteLineColor(line - 1)
			tb.Update()
		}
	}
	dv.CurFileLoc.FPath = ""
	dv.CurFileLoc.Line = 0
}

// ShowBreakFile shows the file for given break index
func (dv *DebugView) ShowBreakFile(bidx int) {
	if bidx < 0 || bidx >= len(dv.State.Breaks) {
		return
	}
	bk := dv.State.Breaks[bidx]
	dv.ShowFile(bk.FPath, bk.Line)
}

// ShowBreaks shows the current breaks
func (dv *DebugView) ShowBreaks(selTab bool) {
	if selTab {
		dv.Tabs().SelectTabByLabel("Breaks")
	}
	sv := dv.BreakVw()
	sv.ShowBreaks()
}

// ShowStack shows the current stack
func (dv *DebugView) ShowStack(selTab bool) {
	if selTab {
		dv.Tabs().SelectTabByLabel("Stack")
	}
	sv := dv.StackVw()
	sv.ShowStack()
}

// ShowVars shows the current vars
func (dv *DebugView) ShowVars(selTab bool) {
	if selTab {
		dv.Tabs().SelectTabByLabel("Vars")
	}
	sv := dv.VarVw()
	sv.ShowVars()
}

// ShowTasks shows the current tasks
func (dv *DebugView) ShowTasks(selTab bool) {
	if selTab {
		dv.Tabs().SelectTabByLabel("Tasks")
	}
	sv := dv.TaskVw()
	sv.ShowTasks()
}

// ShowThreads shows the current threads
func (dv *DebugView) ShowThreads(selTab bool) {
	if selTab {
		dv.Tabs().SelectTabByLabel("Threads")
	}
	sv := dv.ThreadVw()
	sv.ShowThreads()
}

// ShowFindFrames shows the current find frames
func (dv *DebugView) ShowFindFrames(selTab bool) {
	if selTab {
		dv.Tabs().SelectTabByLabel("Find Frames")
	}
	sv := dv.FindFramesVw()
	sv.ShowStack()
}

// ShowGlobalVars shows the current allvars
func (dv *DebugView) ShowGlobalVars(selTab bool) {
	if selTab {
		dv.Tabs().SelectTabByLabel("Global Vars")
	}
	sv := dv.AllVarVw()
	sv.ShowVars()
}

// ShowVar shows info on a given variable within the current frame scope in a text view dialog
func (dv *DebugView) ShowVar(name string) error {
	if !dv.DbgIsAvail() {
		return nil
	}
	vv, err := dv.Dbg.GetVar(name, dv.State.CurTask, dv.State.CurFrame)
	if err != nil {
		return err
	}
	frinfo := ""
	cf := dv.State.StackFrame(dv.State.CurFrame)
	if cf != nil {
		frinfo = "at: " + cf.FPath + fmt.Sprintf(":%d  Thread: %d  Depth: %d", cf.Line, dv.State.CurTask, dv.State.CurFrame)
	}
	VarViewDialog(vv, frinfo, dv)
	return nil
}

// VarValue returns the value of given variable, first looking in local stack vars
// and then in global vars
func (dv *DebugView) VarValue(varNm string) string {
	if !dv.DbgIsAvail() {
		return ""
	}
	if strings.Contains(varNm, ".") {
		vv, err := dv.Dbg.GetVar(varNm, dv.State.CurTask, dv.State.CurFrame)
		if err == nil {
			return vv.Value
		}
	} else {
		vr := dv.State.VarByName(varNm)
		if vr != nil {
			return vr.Value
		}
	}
	return ""
}

var DebugStatusColors = map[gidebug.Status]string{
	gidebug.NotInit:    "grey",
	gidebug.Error:      "#FF8080",
	gidebug.Building:   "yellow",
	gidebug.Ready:      "#80FF80",
	gidebug.Running:    "#FF80FF",
	gidebug.Stopped:    "#8080FF",
	gidebug.Breakpoint: "#8080FF",
	gidebug.Finished:   "tan",
}

func (dv *DebugView) SetStatus(stat gidebug.Status) {
	if dv == nil || dv.This() == nil || dv.Is(ki.Deleted) {
		return
	}
	dv.State.Status = stat
	tb := dv.Toolbar()
	stl := tb.ChildByName("status", 1).(*gi.Label)
	lbl := stat.String()
	if stat == gidebug.Breakpoint {
		lbl = fmt.Sprintf("Break: %d", dv.State.CurBreak)
	}
	stl.SetTextUpdate(lbl)
	// tb.UpdateActions()
}

//////////////////////////////////////////////////////////////////////////////////////
//    GUI config

func (dv *DebugView) ConfigWidget() {
	// dv.ConfigDebugView()
}

// ConfigDebugView configures the view -- parameters for the job must have
// already been set in ge.ProjParams.Debug.
func (dv *DebugView) ConfigDebugView(sup filecat.Supported, exePath string) {
	dv.Sup = sup
	dv.ExePath = exePath
	if dv.HasChildren() {
		return
	}
	dv.Gide, _ = ParentGide(dv)
	dv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	gi.NewToolbar(dv, "toolbar")
	gi.NewTabs(dv, "tabs")
	dv.State.BlankState()
	dv.OutBuf = texteditor.NewBuf()
	dv.OutBuf.Filename = gi.FileName("debug-outbuf")
	dv.ConfigToolbar()
	dv.ConfigTabs()
	dv.State.Breaks = nil // get rid of dummy
	// dv.Start()
}

// Toolbar returns the find toolbar
func (dv *DebugView) Toolbar() *gi.Toolbar {
	return dv.ChildByName("toolbar", 0).(*gi.Toolbar)
}

// Tabs returns the tabs
func (dv *DebugView) Tabs() *gi.Tabs {
	return dv.ChildByName("tabs", 1).(*gi.Tabs)
}

// BreakVw returns the break view from tabs
func (dv DebugView) BreakVw() *BreakView {
	tv := dv.Tabs()
	return tv.TabByLabel("Breaks").Child(0).(*BreakView)
}

// StackVw returns the stack view from tabs
func (dv DebugView) StackVw() *StackView {
	tv := dv.Tabs()
	return tv.TabByLabel("Stack").Child(0).(*StackView)
}

// VarVw returns the vars view from tabs
func (dv DebugView) VarVw() *VarsView {
	tv := dv.Tabs()
	return tv.TabByLabel("Vars").Child(0).(*VarsView)
}

// TaskVw returns the task view from tabs
func (dv DebugView) TaskVw() *TaskView {
	tv := dv.Tabs()
	return tv.TabByLabel("Tasks").Child(0).(*TaskView)
}

// ThreadVw returns the thread view from tabs
func (dv DebugView) ThreadVw() *ThreadView {
	tv := dv.Tabs()
	return tv.TabByLabel("Threads").Child(0).(*ThreadView)
}

// FindFramesVw returns the find frames view from tabs
func (dv DebugView) FindFramesVw() *StackView {
	tv := dv.Tabs()
	return tv.TabByLabel("Find Frames").Child(0).(*StackView)
}

// AllVarVw returns the all vars view from tabs
func (dv DebugView) AllVarVw() *VarsView {
	tv := dv.Tabs()
	return tv.TabByLabel("Global Vars").Child(0).(*VarsView)
}

// ConsoleText returns the console TextEditor
func (dv DebugView) ConsoleText() *texteditor.Editor {
	tv := dv.Tabs()
	cv := tv.TabByLabel("Console").Child(0).(*texteditor.Editor)
	return cv
}

// ConfigTabs configures the tabs
func (dv *DebugView) ConfigTabs() {
	tb := dv.Tabs()
	tb.DeleteTabButtons = false
	if tb.NTabs() > 0 {
		return
	}
	ctv := texteditor.NewEditor(tb.NewTab("Console"), "dbg-console")
	ConfigOutputTextEditor(ctv)
	dv.OutBuf.Opts.LineNos = false
	ctv.SetBuf(dv.OutBuf)
	NewBreakView(tb.NewTab("Breaks")).ConfigBreakView(dv)
	NewStackView(tb.NewTab("Stack")).ConfigStackView(dv, false)
	if dv.Sup == filecat.Go { // dv.Dbg.HasTasks() { // todo: not avail here yet
		NewTaskView(tb.NewTab("Tasks")).ConfigTaskView(dv)
	}
	NewVarsView(tb.NewTab("Vars")).ConfigVarsView(dv, false)
	NewThreadView(tb.NewTab("Threads")).ConfigThreadView(dv)
	NewStackView(tb.NewTab("Find Frames")).ConfigStackView(dv, true) // find frames
	NewVarsView(tb.NewTab("Global Vars")).ConfigVarsView(dv, true)   // all vars
}

// ActionActivate is the update function for actions that depend on the debugger being avail
// for input commands
func (dv *DebugView) ActionActivate(act *gi.Button) {
	// act.SetActiveStateUpdt(dv.DbgIsAvail())
}

func (dv *DebugView) UpdateToolbar() {
	tb := dv.Toolbar()
	tb.UpdateButtons()
}

func (dv *DebugView) ConfigToolbar() {
	tb := dv.Toolbar()
	if tb.HasChildren() {
		return
	}

	// rb := dv.ReplBar()
	// rb.SetStretchMaxWidth()

	gi.NewLabel(tb, "status").SetText("Building..   ").Style(func(s *styles.Style) {
		clr := grr.Log(colors.FromString(DebugStatusColors[dv.State.Status], nil))
		s.BackgroundColor.SetSolid(clr)
	})

	gi.NewButton(tb).SetIcon(icons.Refresh).
		SetTooltip("(re)start the debugger on exe:" + dv.ExePath + " -- automatically rebuilds exe if any source files have changed").
		OnClick(func(e events.Event) {
			dv.Start()
		})

	gi.NewButton(tb).SetText("Cont").SetIcon(icons.PlayArrow).
		SetTooltip("continue execution from current point").
		SetShortcut("Control+Alt+R").Style(func(s *styles.Style) {
		dv.SetState(!dv.DbgIsAvail(), states.Disabled)
	}).OnClick(func(e events.Event) {
		go dv.Continue()
	})

	gi.NewLabel(tb).SetText("Step: ")

	gi.NewButton(tb).SetText("Over").SetIcon(icons.StepOver).
		SetTooltip("continues to the next source line, not entering function calls").
		SetShortcut("F6").Style(func(s *styles.Style) {
		dv.SetState(!dv.DbgIsAvail(), states.Disabled)
	}).OnClick(func(e events.Event) {
		dv.StepOver()
	})

	gi.NewButton(tb).SetText("Into").SetIcon(icons.StepInto).
		SetTooltip("continues to the next source line, entering into function calls").
		SetShortcut("F7").Style(func(s *styles.Style) {
		dv.SetState(!dv.DbgIsAvail(), states.Disabled)
	}).OnClick(func(e events.Event) {
		dv.StepInto()
	})

	gi.NewButton(tb).SetText("Out").SetIcon(icons.StepOut).
		SetTooltip("continues to the return point of the current function").
		SetShortcut("F8").Style(func(s *styles.Style) {
		dv.SetState(!dv.DbgIsAvail(), states.Disabled)
	}).OnClick(func(e events.Event) {
		dv.StepOut()
	})

	gi.NewButton(tb).SetText("Single").SetIcon(icons.Step).
		SetTooltip("steps a single CPU instruction").Style(func(s *styles.Style) {
		dv.SetState(!dv.DbgIsAvail(), states.Disabled)
	}).OnClick(func(e events.Event) {
		dv.StepOut()
	})

	gi.NewButton(tb).SetText("Stop").SetIcon(icons.Stop).
		SetTooltip("stop execution").Style(func(s *styles.Style) {
		dv.SetState(!dv.DbgIsAvail(), states.Disabled)
	}).OnClick(func(e events.Event) {
		dv.Stop()
	})

	gi.NewSeparator(tb, "sep-av")

	gi.NewButton(tb).SetText("Global Vars").SetIcon(icons.Search).
		SetTooltip("list variables at global scope, subject to filter (name contains)").
		Style(func(s *styles.Style) {
			dv.SetState(!dv.DbgIsAvail(), states.Disabled)
		}).OnClick(func(e events.Event) {
		giv.CallFunc(dv, dv.ListGlobalVars)
	})
}

//////////////////////////////////////////////////////////////////////////////////////
//  StackView

// StackView is a view of the stack trace
type StackView struct {
	gi.Layout

	// if true, this is a find frames, not a regular stack
	FindFrames bool
}

func (sv *StackView) DebugVw() *DebugView {
	dv := sv.ParentByType(DebugViewType, ki.Embeds).(*DebugView)
	return dv
}

func (sv *StackView) ConfigStackView(dv *DebugView, findFrames bool) {
	sv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	sv.FindFrames = findFrames
	tv := giv.NewTableView(sv, "stack")
	tv.OnDoubleClick(func(e events.Event) {
		idx := tv.SelIdx
		if sv.FindFrames {
			if idx >= 0 && idx < len(dv.State.FindFrames) {
				fr := dv.State.FindFrames[idx]
				dv.SetThread(fr.ThreadID)
			}
		} else {
			dv.SetFrame(idx)
		}
	})
	tv.SetFlag(false, giv.SliceViewShowIndex)
	tv.SetReadOnly(true)
	if sv.FindFrames {
		tv.SetSlice(&dv.State.FindFrames)
	} else {
		tv.SetSlice(&dv.State.Stack)
	}
}

// TableView returns the tableview
func (sv *StackView) TableView() *giv.TableView {
	return sv.ChildByName("stack", 0).(*giv.TableView)
}

// ShowStack triggers update of view of State.Stack
func (sv *StackView) ShowStack() {
	tv := sv.TableView()
	dv := sv.DebugVw()
	tv.SetReadOnly(true)
	if sv.FindFrames {
		tv.SetSlice(&dv.State.FindFrames)
	} else {
		tv.SelIdx = dv.State.CurFrame
		tv.SetSlice(&dv.State.Stack)
	}
}

//////////////////////////////////////////////////////////////////////////////////////
//  BreakView

// BreakView is a view of the breakpoints
type BreakView struct {
	gi.Layout
}

func (sv *BreakView) DebugVw() *DebugView {
	dv := sv.ParentByType(DebugViewType, ki.Embeds).(*DebugView)
	return dv
}

func (sv *BreakView) ConfigBreakView(dv *DebugView) {
	sv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	tv := giv.NewTableView(sv, "breaks")
	tv.OnDoubleClick(func(e events.Event) {
		idx := tv.SelIdx
		dv.ShowBreakFile(idx)
	})
	// todo:
	// 	} else if sig == int64(giv.SliceViewDeleted) {
	// 		idx := data.(int)
	// 		dv.DeleteBreakIdx(idx)
	// 	}
	tv.SetFlag(false, giv.SliceViewShowIndex)
	tv.SetFlag(true, giv.SliceViewNoAdd)
	tv.SetSlice(&dv.State.Breaks)
}

// TableView returns the tableview
func (sv *BreakView) TableView() *giv.TableView {
	return sv.ChildByName("breaks", 0).(*giv.TableView)
}

// ShowBreaks triggers update of view of State.Breaks
func (sv *BreakView) ShowBreaks() {
	tv := sv.TableView()
	dv := sv.DebugVw()
	if dv.State.CurBreak > 0 {
		_, idx := gidebug.BreakByID(dv.State.Breaks, dv.State.CurBreak)
		if idx >= 0 {
			tv.SelIdx = idx
		}
	}
	tv.SetSlice(&dv.State.Breaks)
	dv.BackupBreaks()
}

//////////////////////////////////////////////////////////////////////////////////////
//  ThreadView

// ThreadView is a view of the threads
type ThreadView struct {
	gi.Layout
}

func (sv *ThreadView) DebugVw() *DebugView {
	dv := sv.ParentByType(DebugViewType, ki.Embeds).(*DebugView)
	return dv
}

func (sv *ThreadView) ConfigThreadView(dv *DebugView) {
	sv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	tv := giv.NewTableView(sv, "threads")
	tv.OnDoubleClick(func(e events.Event) {
		idx := tv.SelIdx
		if dv.Dbg != nil && !dv.Dbg.HasTasks() {
			dv.SetThreadIdx(idx)
		}
	})
	tv.SetReadOnly(true)
	tv.SetFlag(false, giv.SliceViewShowIndex)
	tv.SetSlice(&dv.State.Threads)
}

// TableView returns the tableview
func (sv *ThreadView) TableView() *giv.TableView {
	return sv.ChildByName("threads", 0).(*giv.TableView)
}

// ShowThreads triggers update of view of State.Threads
func (sv *ThreadView) ShowThreads() {
	tv := sv.TableView()
	dv := sv.DebugVw()
	tv.SetReadOnly(true)
	_, idx := gidebug.ThreadByID(dv.State.Threads, dv.State.CurThread)
	if idx >= 0 {
		tv.SelIdx = idx
	}
	tv.SetSlice(&dv.State.Threads)
}

//////////////////////////////////////////////////////////////////////////////////////
//  TaskView

// TaskView is a view of the threads
type TaskView struct {
	gi.Layout
}

func (sv *TaskView) DebugVw() *DebugView {
	dv := sv.ParentByType(DebugViewType, ki.Embeds).(*DebugView)
	return dv
}

func (sv *TaskView) ConfigTaskView(dv *DebugView) {
	sv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	tv := giv.NewTableView(sv, "tasks")
	tv.OnDoubleClick(func(e events.Event) {
		idx := tv.SelIdx
		if dv.Dbg != nil && dv.Dbg.HasTasks() {
			dv.SetThreadIdx(idx)
		}
	})
	tv.SetFlag(false, giv.SliceViewShowIndex)
	tv.SetReadOnly(true)
	tv.SetSlice(&dv.State.Tasks)
}

// TableView returns the tableview
func (sv *TaskView) TableView() *giv.TableView {
	return sv.ChildByName("tasks", 0).(*giv.TableView)
}

// ShowTasks triggers update of view of State.Tasks
func (sv *TaskView) ShowTasks() {
	tv := sv.TableView()
	dv := sv.DebugVw()
	tv.SetReadOnly(true)
	_, idx := gidebug.TaskByID(dv.State.Tasks, dv.State.CurTask)
	if idx >= 0 {
		tv.SelIdx = idx
	}
	tv.SetSlice(&dv.State.Tasks)
}

//////////////////////////////////////////////////////////////////////////////////////
//  VarsView

// VarsView is a view of the variables
type VarsView struct {
	gi.Layout

	// if true, this is global vars, not local ones
	GlobalVars bool
}

func (sv *VarsView) DebugVw() *DebugView {
	dv := sv.ParentByType(DebugViewType, ki.Embeds).(*DebugView)
	return dv
}

func (sv *VarsView) ConfigVarsView(dv *DebugView, globalVars bool) {
	sv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	sv.GlobalVars = globalVars
	tv := giv.NewTableView(sv, "vars")
	tv.OnDoubleClick(func(e events.Event) {
		idx := tv.SelIdx
		if sv.GlobalVars {
			vr := dv.State.GlobalVars[idx]
			dv.ShowVar(vr.Nm)
		} else {
			vr := dv.State.Vars[idx]
			dv.ShowVar(vr.Nm)
		}
	})
	tv.SetFlag(false, giv.SliceViewShowIndex)
	tv.SetReadOnly(true)
	if sv.GlobalVars {
		tv.SetSlice(&dv.State.GlobalVars)
	} else {
		tv.SetSlice(&dv.State.Vars)
	}
}

// TableView returns the tableview
func (sv *VarsView) TableView() *giv.TableView {
	return sv.ChildByName("vars", 0).(*giv.TableView)
}

// ShowVars triggers update of view of State.Vars
func (sv *VarsView) ShowVars() {
	tv := sv.TableView()
	dv := sv.DebugVw()
	tv.SetReadOnly(true)
	if sv.GlobalVars {
		tv.SetSlice(&dv.State.GlobalVars)
	} else {
		tv.SetSlice(&dv.State.Vars)
	}
}

//////////////////////////////////////////////////////////////////////////////////////
//  VarView

// VarView represents a struct, creating a property editor of the fields --
// constructs Children widgets to show the field names and editor fields for
// each field, within an overall frame with an optional title, and a button
// box at the bottom where methods can be invoked
type VarView struct {
	gi.Frame

	// variable being edited
	Var *gidebug.Variable `set:"-"`

	// frame info
	FrameInfo string `set:"-"`

	// parent DebugView
	DbgView *DebugView `json:"-" xml:"-"`
}

// SetVar sets the source variable and ensures configuration
func (vv *VarView) SetVar(vr *gidebug.Variable, frinfo string) {
	vv.FrameInfo = frinfo
	updt := false
	if vv.Var != vr {
		updt = vv.UpdateStart()
		vv.Var = vr
	}
	vv.ConfigVarView()
	vv.UpdateEnd(updt)
}

// Config configures the widget
func (vv *VarView) ConfigVarView() {
	if vv.Var == nil {
		return
	}
	if vv.HasChildren() {
		return
	}
	vv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	gi.NewLabel(vv, "frame-info")
	gi.NewSplits(vv, "splitview")
	vv.SetFrameInfo(vv.FrameInfo)
	vv.ConfigSplits()
	return
}

// SplitView returns the main SplitView
func (vv *VarView) SplitView() *gi.Splits {
	return vv.ChildByName("splitview", 1).(*gi.Splits)
}

// TreeView returns the main TreeView
func (vv *VarView) TreeView() *giv.TreeView {
	return vv.SplitView().Child(0).Child(0).(*giv.TreeView)
}

// StructView returns the main StructView
func (vv *VarView) StructView() *giv.StructView {
	return vv.SplitView().Child(1).(*giv.StructView)
}

// // Toolbar returns the toolbar widget
// func (vv *VarView) Toolbar() *gi.Toolbar {
// 	return vv.ChildByName("toolbar", 0).(*gi.Toolbar)
// }

// SetFrameInfo sets the frame info
func (vv *VarView) SetFrameInfo(finfo string) {
	lab := vv.ChildByName("frame-info", 0).(*gi.Label)
	lab.Text = finfo
}

// // ConfigToolbar adds a VarView toolbar.
// func (vv *VarView) ConfigToolbar() {
// 	tb := vv.Toolbar()
// 	if tb != nil && tb.HasChildren() {
// 		return
// 	}
// 	tb.SetStretchMaxWidth()
// 	giv.ToolbarView(vv, vv.Viewport, tb)
// }

// ConfigSplits configures the SplitView.
func (vv *VarView) ConfigSplits() {
	if vv.Var == nil {
		return
	}
	split := vv.SplitView()
	// split.Dim = mat32.Y
	split.Dim = mat32.X

	if len(split.Kids) == 0 {
		tvfr := gi.NewFrame(split, "tvfr")
		tv := giv.NewTreeView(tvfr, "tv")
		sv := giv.NewStructView(split, "sv")
		tv.OnSelect(func(e events.Event) {
			sv.SetStruct(tv.SyncNode)
		})
		split.SetSplits(.3, .7)
	}
	tv := vv.TreeView()
	tv.SyncRootNode(vv.Var)
	sv := vv.StructView()
	sv.SetStruct(vv.Var)
}

// VarViewDialog opens an interactive editor of the given Ki tree, at its
// root, returns VarView and window
func VarViewDialog(vr *gidebug.Variable, frinfo string, dbgVw *DebugView) *VarView {
	if gi.RecycleDialog(vr) {
		return nil
	}
	wnm := "var-view"
	wti := "Var View"
	if vr != nil {
		wnm += "-" + vr.Name()
		wti += ": " + vr.Name()
	}
	b := gi.NewBody() // wnm)
	b.Title = wti
	vv := NewVarView(b, "view")
	vv.DbgView = dbgVw
	vv.SetVar(vr, frinfo)
	// tb := vv.Toolbar()
	// tb.UpdateActions()
	b.NewWindow().Run()
	return vv
}
