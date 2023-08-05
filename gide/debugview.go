// Copyright (c) 2020, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gide/gidebug"
	"github.com/goki/gide/gidebug/gidelve"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
	"github.com/goki/pi/filecat"
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
var Debuggers = map[filecat.Supported]func(path, rootPath string, outbuf *giv.TextBuf, pars *gidebug.Params) (gidebug.GiDebug, error){
	filecat.Go: func(path, rootPath string, outbuf *giv.TextBuf, pars *gidebug.Params) (gidebug.GiDebug, error) {
		return gidelve.NewGiDelve(path, rootPath, outbuf, pars)
	},
}

// NewDebugger returns a new debugger for given supported file type
func NewDebugger(sup filecat.Supported, path, rootPath string, outbuf *giv.TextBuf, pars *gidebug.Params) (gidebug.GiDebug, error) {
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
	Sup filecat.Supported `desc:"supported file type to determine debugger"`

	// path to executable / dir to debug
	ExePath string `desc:"path to executable / dir to debug"`

	// time when dbg was last restarted
	DbgTime time.Time `desc:"time when dbg was last restarted"`

	// the debugger
	Dbg gidebug.GiDebug `json:"-" xml:"-" desc:"the debugger"`

	// all relevant debug state info
	State gidebug.AllState `json:"-" xml:"-" desc:"all relevant debug state info"`

	// current ShowFile location -- cleared before next one or run
	CurFileLoc gidebug.Location `json:"-" xml:"-" desc:"current ShowFile location -- cleared before next one or run"`

	// backup breakpoints list -- to track deletes
	BBreaks []*gidebug.Break `json:"-" xml:"-" desc:"backup breakpoints list -- to track deletes"`

	// output from the debugger
	OutBuf *giv.TextBuf `json:"-" xml:"-" desc:"output from the debugger"`

	// parent gide project
	Gide Gide `json:"-" xml:"-" desc:"parent gide project"`
}

var KiT_DebugView = kit.Types.AddType(&DebugView{}, DebugViewProps)

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
		if dv.IsDeleted() || dv.IsDestroyed() {
			return
		}
	}
	if dv.Gide != nil {
		vp := dv.Gide.VPort()
		if vp != nil && vp.Win != nil {
			vp.Win.OSWin.Raise()
		}
	}
	if ds != nil {
		updt := dv.UpdateStart()
		dv.InitState(ds)
		dv.UpdateEnd(updt)
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
	if dv.IsDeleted() {
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
	if dv.Gide == nil || dv.Gide.IsDeleted() {
		return
	}
	tb := dv.Gide.TextBufForFile(fpath, false)
	if tb != nil {
		tb.DeleteLineColor(line - 1)
		tb.Refresh()
	}
}

// DeleteAllBreaks deletes all breakpoints
func (dv *DebugView) DeleteAllBreaks() {
	if dv.Gide == nil || dv.Gide.IsDeleted() {
		return
	}
	for _, bk := range dv.State.Breaks {
		dv.DeleteBreakInBuf(bk.FPath, bk.Line)
	}
}

// UpdateBreakInBuf updates break status in its TextBuf
// line is 1-based line number
func (dv *DebugView) UpdateBreakInBuf(fpath string, line int, stat DebugBreakStatus) {
	if dv.Gide == nil || dv.Gide.IsDeleted() {
		return
	}
	tb := dv.Gide.TextBufForFile(fpath, false)
	if tb != nil {
		tb.SetLineColor(line-1, DebugBreakColors[stat])
		tb.Refresh()
	}
}

// UpdateAllBreaks updates all breakpoints
func (dv *DebugView) UpdateAllBreaks() {
	if dv.Gide == nil || dv.Gide.IsDeleted() {
		return
	}
	wupdt := dv.TopUpdateStart()
	for _, bk := range dv.State.Breaks {
		if bk.ID == dv.State.CurBreak {
			dv.UpdateBreakInBuf(bk.FPath, bk.Line, DebugBreakCurrent)
		} else if bk.On {
			dv.UpdateBreakInBuf(bk.FPath, bk.Line, DebugBreakActive)
		} else {
			dv.UpdateBreakInBuf(bk.FPath, bk.Line, DebugBreakInactive)
		}
	}
	dv.TopUpdateEnd(wupdt)
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
	if dv == nil || dv.This() == nil || dv.IsDeleted() || dv.IsDestroyed() || dv.Dbg == nil {
		return
	}
	wupdt := dv.TopUpdateStart()
	defer dv.TopUpdateEnd(wupdt)
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
	dv.UpdateToolBar()
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
		gi.PromptDialog(dv.Viewport, gi.DlgOpts{Title: "No Frames Found", Prompt: fmt.Sprintf("Could not find any stack frames for file name: %v, err: %v", fpath, err)}, gi.AddOk, gi.NoCancel, nil, nil)
		return
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
	wupdt := dv.TopUpdateStart()
	dv.DeleteCurPCInBuf()
	dv.Gide.ShowFile(fpath, line)
	dv.SetCurPCInBuf(fpath, line)
	dv.TopUpdateEnd(wupdt)
}

// SetCurPCInBuf sets the current PC location in given file
// line is 1-based line number
func (dv *DebugView) SetCurPCInBuf(fpath string, line int) {
	tb := dv.Gide.TextBufForFile(fpath, false)
	if tb != nil {
		if !tb.HasLineColor(line - 1) {
			tb.SetLineColor(line-1, DebugBreakColors[DebugPCCurrent])
			tb.Refresh()
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
			tb.Refresh()
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
		dv.Tabs().SelectTabByName("Breaks")
	}
	sv := dv.BreakVw()
	sv.ShowBreaks()
}

// ShowStack shows the current stack
func (dv *DebugView) ShowStack(selTab bool) {
	if selTab {
		dv.Tabs().SelectTabByName("Stack")
	}
	sv := dv.StackVw()
	sv.ShowStack()
}

// ShowVars shows the current vars
func (dv *DebugView) ShowVars(selTab bool) {
	if selTab {
		dv.Tabs().SelectTabByName("Vars")
	}
	sv := dv.VarVw()
	sv.ShowVars()
}

// ShowTasks shows the current tasks
func (dv *DebugView) ShowTasks(selTab bool) {
	if selTab {
		dv.Tabs().SelectTabByName("Tasks")
	}
	sv := dv.TaskVw()
	sv.ShowTasks()
}

// ShowThreads shows the current threads
func (dv *DebugView) ShowThreads(selTab bool) {
	if selTab {
		dv.Tabs().SelectTabByName("Threads")
	}
	sv := dv.ThreadVw()
	sv.ShowThreads()
}

// ShowFindFrames shows the current find frames
func (dv *DebugView) ShowFindFrames(selTab bool) {
	if selTab {
		dv.Tabs().SelectTabByName("Find Frames")
	}
	sv := dv.FindFramesVw()
	sv.ShowStack()
}

// ShowGlobalVars shows the current allvars
func (dv *DebugView) ShowGlobalVars(selTab bool) {
	if selTab {
		dv.Tabs().SelectTabByName("Global Vars")
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
	if dv == nil || dv.This() == nil || dv.IsDeleted() || dv.IsDestroyed() {
		return
	}
	dv.State.Status = stat
	tb := dv.ToolBar()
	stl := tb.ChildByName("status", 1).(*gi.Label)
	clr := DebugStatusColors[stat]
	stl.CurBgColor.SetString(clr, nil)
	if gi.Prefs.IsDarkMode() {
		stl.CurBgColor = stl.CurBgColor.Darker(75)
	}
	lbl := stat.String()
	if stat == gidebug.Breakpoint {
		lbl = fmt.Sprintf("Break: %d", dv.State.CurBreak)
	}
	stl.SetText(lbl)
	tb.UpdateActions()
}

//////////////////////////////////////////////////////////////////////////////////////
//    GUI config

// Config configures the view -- parameters for the job must have
// already been set in ge.ProjParams.Debug.
func (dv *DebugView) Config(ge Gide, sup filecat.Supported, exePath string) {
	dv.Gide = ge
	dv.Sup = sup
	dv.ExePath = exePath
	dv.Lay = gi.LayoutVert
	dv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "toolbar")
	config.Add(gi.KiT_TabView, "tabs")
	mods, updt := dv.ConfigChildren(config)
	if mods {
		dv.State.BlankState()
		dv.OutBuf = &giv.TextBuf{}
		dv.OutBuf.InitName(dv.OutBuf, "debug-outbuf")
		dv.ConfigToolBar()
		dv.ConfigTabs()
		dv.State.Breaks = nil // get rid of dummy
	} else {
		updt = dv.UpdateStart()
	}
	dv.Start()
	dv.SetFullReRender()
	dv.UpdateEnd(updt)
}

// ToolBar returns the find toolbar
func (dv *DebugView) ToolBar() *gi.ToolBar {
	return dv.ChildByName("toolbar", 0).(*gi.ToolBar)
}

// Tabs returns the tabs
func (dv *DebugView) Tabs() *gi.TabView {
	return dv.ChildByName("tabs", 1).(*gi.TabView)
}

// BreakVw returns the break view from tabs
func (dv DebugView) BreakVw() *BreakView {
	tv := dv.Tabs()
	return tv.TabByName("Breaks").(*BreakView)
}

// StackVw returns the stack view from tabs
func (dv DebugView) StackVw() *StackView {
	tv := dv.Tabs()
	return tv.TabByName("Stack").(*StackView)
}

// VarVw returns the vars view from tabs
func (dv DebugView) VarVw() *VarsView {
	tv := dv.Tabs()
	return tv.TabByName("Vars").(*VarsView)
}

// TaskVw returns the task view from tabs
func (dv DebugView) TaskVw() *TaskView {
	tv := dv.Tabs()
	return tv.TabByName("Tasks").(*TaskView)
}

// ThreadVw returns the thread view from tabs
func (dv DebugView) ThreadVw() *ThreadView {
	tv := dv.Tabs()
	return tv.TabByName("Threads").(*ThreadView)
}

// FindFramesVw returns the find frames view from tabs
func (dv DebugView) FindFramesVw() *StackView {
	tv := dv.Tabs()
	return tv.TabByName("Find Frames").(*StackView)
}

// AllVarVw returns the all vars view from tabs
func (dv DebugView) AllVarVw() *VarsView {
	tv := dv.Tabs()
	return tv.TabByName("Global Vars").(*VarsView)
}

// ConsoleText returns the console TextView
func (dv DebugView) ConsoleText() *giv.TextView {
	tv := dv.Tabs()
	cv := tv.TabByName("Console").Child(0).(*giv.TextView)
	return cv
}

// ConfigTabs configures the tabs
func (dv *DebugView) ConfigTabs() {
	tb := dv.Tabs()
	tb.NoDeleteTabs = true
	cv := tb.RecycleTab("Console", gi.KiT_Layout, false).(*gi.Layout)
	otv := ConfigOutputTextView(cv)
	dv.OutBuf.Opts.LineNos = false
	otv.SetBuf(dv.OutBuf)
	bv := tb.RecycleTab("Breaks", KiT_BreakView, false).(*BreakView)
	bv.Config(dv)
	sv := tb.RecycleTab("Stack", KiT_StackView, false).(*StackView)
	sv.Config(dv, false) // reg stack
	vv := tb.RecycleTab("Vars", KiT_VarsView, false).(*VarsView)
	vv.Config(dv, false)
	if dv.Sup == filecat.Go { // dv.Dbg.HasTasks() { // todo: not avail here yet
		ta := tb.RecycleTab("Tasks", KiT_TaskView, false).(*TaskView)
		ta.Config(dv)
	}
	th := tb.RecycleTab("Threads", KiT_ThreadView, false).(*ThreadView)
	th.Config(dv)
	ff := tb.RecycleTab("Find Frames", KiT_StackView, false).(*StackView)
	ff.Config(dv, true) // find frames
	av := tb.RecycleTab("Global Vars", KiT_VarsView, false).(*VarsView)
	av.Config(dv, true) // all vars
}

// ActionActivate is the update function for actions that depend on the debugger being avail
// for input commands
func (dv *DebugView) ActionActivate(act *gi.Action) {
	act.SetActiveStateUpdt(dv.DbgIsAvail())
}

func (dv *DebugView) UpdateToolBar() {
	tb := dv.ToolBar()
	tb.UpdateActions()
}

func (dv *DebugView) ConfigToolBar() {
	tb := dv.ToolBar()
	if tb.HasChildren() {
		return
	}
	tb.SetStretchMaxWidth()

	// rb := dv.ReplBar()
	// rb.SetStretchMaxWidth()

	// cb.AddAction(gi.ActOpts{Label: "Updt", Icon: "update", Tooltip: "update current state"}, dv.This(),
	// 	func(recv, send ki.Ki, sig int64, data interface{}) {
	// 		dvv := recv.Embed(KiT_DebugView).(*DebugView)
	// 		dvv.UpdateView()
	// 		cb.UpdateActions()
	// 	})
	stl := gi.AddNewLabel(tb, "status", "Building..   ")
	stl.Redrawable = true
	stl.CurBgColor.SetString("yellow", nil)
	if gi.Prefs.IsDarkMode() {
		stl.CurBgColor = stl.CurBgColor.Darker(75)
	}
	tb.AddAction(gi.ActOpts{Label: "Restart", Icon: "update", Tooltip: "(re)start the debugger on exe:" + dv.ExePath + " -- automatically rebuilds exe if any source files have changed"}, dv.This(),
		func(recv, send ki.Ki, sig int64, data any) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			dvv.Start()
			tb.UpdateActions()
		})
	tb.AddAction(gi.ActOpts{Label: "Cont", Icon: "play", Tooltip: "continue execution from current point", Shortcut: "Control+Alt+R", UpdateFunc: dv.ActionActivate}, dv.This(),
		func(recv, send ki.Ki, sig int64, data any) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			go dvv.Continue()
			tb.UpdateActions()
		})
	gi.AddNewLabel(tb, "step", "Step: ")
	tb.AddAction(gi.ActOpts{Label: "Over", Icon: "step-over", Tooltip: "continues to the next source line, not entering function calls", Shortcut: "F6", UpdateFunc: dv.ActionActivate}, dv.This(),
		func(recv, send ki.Ki, sig int64, data any) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			dvv.StepOver()
			tb.UpdateActions()
		})
	tb.AddAction(gi.ActOpts{Label: "Into", Icon: "step-into", Tooltip: "continues to the next source line, entering into function calls", Shortcut: "F7", UpdateFunc: dv.ActionActivate}, dv.This(),
		func(recv, send ki.Ki, sig int64, data any) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			dvv.StepInto()
			tb.UpdateActions()
		})
	tb.AddAction(gi.ActOpts{Label: "Out", Icon: "step-out", Tooltip: "continues to the return point of the current function", Shortcut: "F8", UpdateFunc: dv.ActionActivate}, dv.This(),
		func(recv, send ki.Ki, sig int64, data any) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			dvv.StepOut()
			tb.UpdateActions()
		})
	tb.AddAction(gi.ActOpts{Label: "Single", Icon: "step-fwd", Tooltip: "steps a single CPU instruction", UpdateFunc: dv.ActionActivate}, dv.This(),
		func(recv, send ki.Ki, sig int64, data any) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			dvv.StepOut()
			tb.UpdateActions()
		})
	tb.AddAction(gi.ActOpts{Label: "Stop", Icon: "stop", Tooltip: "stop execution"}, dv.This(),
		func(recv, send ki.Ki, sig int64, data any) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			dvv.Stop()
			tb.UpdateActions()
		})
	tb.AddSeparator("sep-av")
	tb.AddAction(gi.ActOpts{Label: "Global Vars", Icon: "search", Tooltip: "list variables at global scope, subject to filter (name contains)"}, dv.This(),
		func(recv, send ki.Ki, sig int64, data any) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			giv.CallMethod(dvv, "ListGlobalVars", dvv.Viewport)
			tb.UpdateActions()
		})
}

// DebugViewProps are style properties for DebugView
var DebugViewProps = ki.Props{
	"EnumType:Flag": gi.KiT_NodeFlags,
	"max-width":     -1,
	"max-height":    -1,
	"CallMethods": ki.PropSlice{
		{"ListGlobalVars", ki.Props{
			"Args": ki.PropSlice{
				{"Filter", ki.Props{
					"width": 40,
				}},
			},
		}},
	},
}

//////////////////////////////////////////////////////////////////////////////////////
//  StackView

// StackView is a view of the stack trace
type StackView struct {
	gi.Layout

	// if true, this is a find frames, not a regular stack
	FindFrames bool `desc:"if true, this is a find frames, not a regular stack"`
}

var KiT_StackView = kit.Types.AddType(&StackView{}, StackViewProps)

func (sv *StackView) DebugVw() *DebugView {
	dv := sv.ParentByType(KiT_DebugView, ki.Embeds).Embed(KiT_DebugView).(*DebugView)
	return dv
}

func (sv *StackView) Config(dv *DebugView, findFrames bool) {
	sv.Lay = gi.LayoutVert
	sv.FindFrames = findFrames
	config := kit.TypeAndNameList{}
	config.Add(giv.KiT_TableView, "stack")
	mods, updt := sv.ConfigChildren(config)
	tv := sv.TableView()
	if mods {
		tv.SliceViewSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(giv.SliceViewDoubleClicked) {
				idx := data.(int)
				if sv.FindFrames {
					if idx >= 0 && idx < len(dv.State.FindFrames) {
						fr := dv.State.FindFrames[idx]
						dv.SetThread(fr.ThreadID)
					}
				} else {
					dv.SetFrame(idx)
				}
			}
		})
	} else {
		updt = sv.UpdateStart()
	}
	tv.SetStretchMax()
	tv.SetInactive()
	if sv.FindFrames {
		tv.SetSlice(&dv.State.FindFrames)
	} else {
		tv.SetSlice(&dv.State.Stack)
	}
	sv.UpdateEnd(updt)
}

// TableView returns the tableview
func (sv *StackView) TableView() *giv.TableView {
	return sv.ChildByName("stack", 0).(*giv.TableView)
}

// ShowStack triggers update of view of State.Stack
func (sv *StackView) ShowStack() {
	tv := sv.TableView()
	dv := sv.DebugVw()
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	tv.SetInactive()
	if sv.FindFrames {
		tv.SetSlice(&dv.State.FindFrames)
	} else {
		tv.SelectedIdx = dv.State.CurFrame
		tv.SetSlice(&dv.State.Stack)
	}
	sv.UpdateEnd(updt)
}

// StackViewProps are style properties for DebugView
var StackViewProps = ki.Props{
	"EnumType:Flag": gi.KiT_NodeFlags,
	"max-width":     -1,
	"max-height":    -1,
}

//////////////////////////////////////////////////////////////////////////////////////
//  BreakView

// BreakView is a view of the breakpoints
type BreakView struct {
	gi.Layout
}

var KiT_BreakView = kit.Types.AddType(&BreakView{}, BreakViewProps)

func (sv *BreakView) DebugVw() *DebugView {
	dv := sv.ParentByType(KiT_DebugView, ki.Embeds).Embed(KiT_DebugView).(*DebugView)
	return dv
}

func (sv *BreakView) Config(dv *DebugView) {
	sv.Lay = gi.LayoutVert
	config := kit.TypeAndNameList{}
	config.Add(giv.KiT_TableView, "breaks")
	mods, updt := sv.ConfigChildren(config)
	tv := sv.TableView()
	if mods {
		tv.SliceViewSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(giv.SliceViewDoubleClicked) {
				idx := data.(int)
				dv.ShowBreakFile(idx)
			} else if sig == int64(giv.SliceViewDeleted) {
				idx := data.(int)
				dv.DeleteBreakIdx(idx)
			}
		})
	} else {
		updt = sv.UpdateStart()
	}
	tv.SetStretchMax()
	tv.NoAdd = true
	tv.SetSlice(&dv.State.Breaks)
	sv.UpdateEnd(updt)
}

// TableView returns the tableview
func (sv *BreakView) TableView() *giv.TableView {
	return sv.ChildByName("breaks", 0).(*giv.TableView)
}

// ShowBreaks triggers update of view of State.Breaks
func (sv *BreakView) ShowBreaks() {
	tv := sv.TableView()
	dv := sv.DebugVw()
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	if dv.State.CurBreak > 0 {
		_, idx := gidebug.BreakByID(dv.State.Breaks, dv.State.CurBreak)
		if idx >= 0 {
			tv.SelectedIdx = idx
		}
	}
	tv.SetSlice(&dv.State.Breaks)
	dv.BackupBreaks()
	sv.UpdateEnd(updt)
}

// BreakViewProps are style properties for DebugView
var BreakViewProps = ki.Props{
	"EnumType:Flag": gi.KiT_NodeFlags,
	"max-width":     -1,
	"max-height":    -1,
}

//////////////////////////////////////////////////////////////////////////////////////
//  ThreadView

// ThreadView is a view of the threads
type ThreadView struct {
	gi.Layout
}

var KiT_ThreadView = kit.Types.AddType(&ThreadView{}, ThreadViewProps)

func (sv *ThreadView) DebugVw() *DebugView {
	dv := sv.ParentByType(KiT_DebugView, ki.Embeds).Embed(KiT_DebugView).(*DebugView)
	return dv
}

func (sv *ThreadView) Config(dv *DebugView) {
	sv.Lay = gi.LayoutVert
	config := kit.TypeAndNameList{}
	config.Add(giv.KiT_TableView, "threads")
	mods, updt := sv.ConfigChildren(config)
	tv := sv.TableView()
	if mods {
		tv.SliceViewSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(giv.SliceViewDoubleClicked) {
				idx := data.(int)
				if dv.Dbg != nil && !dv.Dbg.HasTasks() {
					dv.SetThreadIdx(idx)
				}
			}
		})
	} else {
		updt = sv.UpdateStart()
	}
	tv.SetStretchMax()
	tv.SetInactive()
	tv.SetSlice(&dv.State.Threads)
	sv.UpdateEnd(updt)
}

// TableView returns the tableview
func (sv *ThreadView) TableView() *giv.TableView {
	return sv.ChildByName("threads", 0).(*giv.TableView)
}

// ShowThreads triggers update of view of State.Threads
func (sv *ThreadView) ShowThreads() {
	tv := sv.TableView()
	dv := sv.DebugVw()
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	tv.SetInactive()
	_, idx := gidebug.ThreadByID(dv.State.Threads, dv.State.CurThread)
	if idx >= 0 {
		tv.SelectedIdx = idx
	}
	tv.SetSlice(&dv.State.Threads)
	sv.UpdateEnd(updt)
}

// ThreadViewProps are style properties for DebugView
var ThreadViewProps = ki.Props{
	"EnumType:Flag": gi.KiT_NodeFlags,
	"max-width":     -1,
	"max-height":    -1,
}

//////////////////////////////////////////////////////////////////////////////////////
//  TaskView

// TaskView is a view of the threads
type TaskView struct {
	gi.Layout
}

var KiT_TaskView = kit.Types.AddType(&TaskView{}, TaskViewProps)

func (sv *TaskView) DebugVw() *DebugView {
	dv := sv.ParentByType(KiT_DebugView, ki.Embeds).Embed(KiT_DebugView).(*DebugView)
	return dv
}

func (sv *TaskView) Config(dv *DebugView) {
	sv.Lay = gi.LayoutVert
	config := kit.TypeAndNameList{}
	config.Add(giv.KiT_TableView, "tasks")
	mods, updt := sv.ConfigChildren(config)
	tv := sv.TableView()
	if mods {
		tv.SliceViewSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(giv.SliceViewDoubleClicked) {
				idx := data.(int)
				if dv.Dbg != nil && dv.Dbg.HasTasks() {
					dv.SetThreadIdx(idx)
				}
			}
		})
	} else {
		updt = sv.UpdateStart()
	}
	tv.SetStretchMax()
	tv.SetInactive()
	tv.SetSlice(&dv.State.Tasks)
	sv.UpdateEnd(updt)
}

// TableView returns the tableview
func (sv *TaskView) TableView() *giv.TableView {
	return sv.ChildByName("tasks", 0).(*giv.TableView)
}

// ShowTasks triggers update of view of State.Tasks
func (sv *TaskView) ShowTasks() {
	tv := sv.TableView()
	dv := sv.DebugVw()
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	tv.SetInactive()
	_, idx := gidebug.TaskByID(dv.State.Tasks, dv.State.CurTask)
	if idx >= 0 {
		tv.SelectedIdx = idx
	}
	tv.SetSlice(&dv.State.Tasks)
	sv.UpdateEnd(updt)
}

// TaskViewProps are style properties for DebugView
var TaskViewProps = ki.Props{
	"EnumType:Flag": gi.KiT_NodeFlags,
	"max-width":     -1,
	"max-height":    -1,
}

//////////////////////////////////////////////////////////////////////////////////////
//  VarsView

// VarsView is a view of the variables
type VarsView struct {
	gi.Layout

	// if true, this is global vars, not local ones
	GlobalVars bool `desc:"if true, this is global vars, not local ones"`
}

var KiT_VarsView = kit.Types.AddType(&VarsView{}, VarsViewProps)

func (sv *VarsView) DebugVw() *DebugView {
	dv := sv.ParentByType(KiT_DebugView, ki.Embeds).Embed(KiT_DebugView).(*DebugView)
	return dv
}

func (sv *VarsView) Config(dv *DebugView, globalVars bool) {
	sv.Lay = gi.LayoutVert
	sv.GlobalVars = globalVars
	config := kit.TypeAndNameList{}
	config.Add(giv.KiT_TableView, "vars")
	mods, updt := sv.ConfigChildren(config)
	tv := sv.TableView()
	if mods {
		tv.SliceViewSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(giv.SliceViewDoubleClicked) {
				idx := data.(int)
				if sv.GlobalVars {
					vr := dv.State.GlobalVars[idx]
					dv.ShowVar(vr.Nm)
				} else {
					vr := dv.State.Vars[idx]
					dv.ShowVar(vr.Nm)
				}
			}
		})
	} else {
		updt = sv.UpdateStart()
	}
	tv.SetStretchMax()
	tv.SetInactive()
	if sv.GlobalVars {
		tv.SetSlice(&dv.State.GlobalVars)
	} else {
		tv.SetSlice(&dv.State.Vars)
	}
	sv.UpdateEnd(updt)
}

// TableView returns the tableview
func (sv *VarsView) TableView() *giv.TableView {
	return sv.ChildByName("vars", 0).(*giv.TableView)
}

// ShowVars triggers update of view of State.Vars
func (sv *VarsView) ShowVars() {
	tv := sv.TableView()
	dv := sv.DebugVw()
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	tv.SetInactive()
	if sv.GlobalVars {
		tv.SetSlice(&dv.State.GlobalVars)
	} else {
		tv.SetSlice(&dv.State.Vars)
	}
	sv.UpdateEnd(updt)
}

// VarsViewProps are style properties for DebugView
var VarsViewProps = ki.Props{
	"EnumType:Flag": gi.KiT_NodeFlags,
	"max-width":     -1,
	"max-height":    -1,
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
	Var *gidebug.Variable `desc:"variable being edited"`

	// frame info
	FrameInfo string `desc:"frame info"`

	// parent DebugView
	DbgView *DebugView `json:"-" xml:"-" desc:"parent DebugView"`
}

var KiT_VarView = kit.Types.AddType(&VarView{}, VarViewProps)

// AddNewVarView adds a new gieditor to given parent node, with given name.
func AddNewVarView(parent ki.Ki, name string) *VarView {
	return parent.AddNewChild(KiT_VarView, name).(*VarView)
}

// SetVar sets the source variable and ensures configuration
func (vv *VarView) SetVar(vr *gidebug.Variable, frinfo string) {
	vv.FrameInfo = frinfo
	updt := false
	if vv.Var != vr {
		updt = vv.UpdateStart()
		vv.Var = vr
	}
	vv.Config()
	vv.UpdateEnd(updt)
}

// Config configures the widget
func (vv *VarView) Config() {
	if vv.Var == nil {
		return
	}
	vv.Lay = gi.LayoutVert
	vv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Label, "frame-info")
	// config.Add(gi.KiT_ToolBar, "toolbar")
	config.Add(gi.KiT_SplitView, "splitview")
	mods, updt := vv.ConfigChildren(config)
	vv.SetFrameInfo(vv.FrameInfo)
	vv.ConfigSplitView()
	// vv.ConfigToolBar()
	if mods {
		vv.UpdateEnd(updt)
	}
	return
}

// SplitView returns the main SplitView
func (vv *VarView) SplitView() *gi.SplitView {
	return vv.ChildByName("splitview", 1).(*gi.SplitView)
}

// TreeView returns the main TreeView
func (vv *VarView) TreeView() *giv.TreeView {
	return vv.SplitView().Child(0).Child(0).(*giv.TreeView)
}

// StructView returns the main StructView
func (vv *VarView) StructView() *giv.StructView {
	return vv.SplitView().Child(1).(*giv.StructView)
}

// // ToolBar returns the toolbar widget
// func (vv *VarView) ToolBar() *gi.ToolBar {
// 	return vv.ChildByName("toolbar", 0).(*gi.ToolBar)
// }

// SetFrameInfo sets the frame info
func (vv *VarView) SetFrameInfo(finfo string) {
	lab := vv.ChildByName("frame-info", 0).(*gi.Label)
	lab.Text = finfo
}

// // ConfigToolBar adds a VarView toolbar.
// func (vv *VarView) ConfigToolBar() {
// 	tb := vv.ToolBar()
// 	if tb != nil && tb.HasChildren() {
// 		return
// 	}
// 	tb.SetStretchMaxWidth()
// 	giv.ToolBarView(vv, vv.Viewport, tb)
// }

// ConfigSplitView configures the SplitView.
func (vv *VarView) ConfigSplitView() {
	if vv.Var == nil {
		return
	}
	split := vv.SplitView()
	// split.Dim = mat32.Y
	split.Dim = mat32.X

	if len(split.Kids) == 0 {
		tvfr := gi.AddNewFrame(split, "tvfr", gi.LayoutHoriz)
		tv := giv.AddNewTreeView(tvfr, "tv")
		giv.AddNewStructView(split, "sv")
		tv.TreeViewSig.Connect(vv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if data == nil {
				return
			}
			vve, _ := recv.Embed(KiT_VarView).(*VarView)
			svr := vve.StructView()
			tvn, _ := data.(ki.Ki).Embed(giv.KiT_TreeView).(*giv.TreeView)
			if sig == int64(giv.TreeViewSelected) {
				svr.SetStruct(tvn.SrcNode)
			}
		})
		split.SetSplits(.3, .7)
	}
	tv := vv.TreeView()
	tv.SetRootNode(vv.Var)
	sv := vv.StructView()
	sv.Viewport = vv.Viewport
	sv.SetStruct(vv.Var)
}

// func (ge *VarView) Render2D() {
// 	// ge.ToolBar().UpdateActions()
// 	ge.Frame.Render2D()
// }

var VarViewProps = ki.Props{
	"EnumType:Flag":    gi.KiT_NodeFlags,
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
}

// VarViewDialog opens an interactive editor of the given Ki tree, at its
// root, returns VarView and window
func VarViewDialog(vr *gidebug.Variable, frinfo string, dbgVw *DebugView) *VarView {
	width := 1280
	height := 920
	wnm := "var-view"
	wti := "Var View"
	if vr != nil {
		wnm += "-" + vr.Name()
		wti += ": " + vr.Name()
	}

	win, recyc := gi.RecycleMainWindow(vr, wnm, wti, width, height)
	if recyc {
		mfr, err := win.MainFrame()
		if err == nil {
			vv := mfr.Child(0).(*VarView)
			vv.SetFrameInfo(frinfo)
		}
	}

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	vv := AddNewVarView(mfr, "view")
	vv.Viewport = vp
	vv.DbgView = dbgVw
	vv.SetVar(vr, frinfo)

	// tb := vv.ToolBar()
	// tb.UpdateActions()

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop() // in a separate goroutine
	return vv
}
