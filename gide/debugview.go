// Copyright (c) 2020, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"fmt"
	"log"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/mat32"
	"github.com/goki/gide/gidebug"
	"github.com/goki/gide/gidebug/gidelve"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/filecat"
)

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
		err := fmt.Errorf("Gi Debug: File type %v not supported\n", sup)
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
	Sup     filecat.Supported `desc:"supported file type to determine debugger"`
	ExePath string            `desc:"path to executable / dir to debug"`
	Dbg     gidebug.GiDebug   `json:"-" xml:"-" desc:"the debugger"`
	State   gidebug.AllState  `json:"-" xml:"-" desc:"all relevant debug state info"`
	OutBuf  *giv.TextBuf      `json:"-" xml:"-" desc:"output from the debugger"`
	Gide    Gide              `json:"-" xml:"-" desc:"parent gide project"`
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
}

// DeleteAllBreaks deletes all breakpoints
func (dv *DebugView) DeleteAllBreaks() {
	if dv.Gide == nil || dv.Gide.IsDeleted() {
		return
	}
	for _, bk := range dv.State.Breaks {
		tb := dv.Gide.TextBufForFile(bk.FPath, false)
		if tb != nil {
			tb.DeleteLineColor(bk.Line - 1)
			tb.DeleteLineIcon(bk.Line - 1)
			tb.Refresh()
		}
	}
}

// Start starts the debuger
func (dv *DebugView) Start() {
	if dv.Gide == nil {
		return
	}
	if dv.Dbg == nil {
		rootPath := string(dv.Gide.ProjPrefs().ProjRoot)
		pars := &dv.Gide.ProjPrefs().Debug
		dv.State.Mode = pars.Mode
		pars.StatFunc = func(stat gidebug.Status) {
			if stat == gidebug.Ready && dv.State.Mode == gidebug.Attach {
				dv.UpdateFmState()
			}
			dv.SetStatus(stat)
		}
		dbg, err := NewDebugger(dv.Sup, dv.ExePath, rootPath, dv.OutBuf, pars)
		if err == nil {
			dv.Dbg = dbg
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
	dv.Dbg.UpdateBreaks(&dv.State.Breaks)
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
	if !dv.DbgIsAvail() {
		dv.State.DeleteBreakByFile(fpath, line) // already doing this!
		dv.ShowBreaks(true)
		return
	}
	bk, _ := dv.State.BreakByFile(fpath, line)
	if bk != nil {
		dv.Dbg.ClearBreak(bk.ID)
		dv.State.DeleteBreakByID(bk.ID)
	}
	dv.ShowBreaks(true)
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
	dv.ShowBreaks(false)
	dv.ShowStack(false)
	dv.ShowVars(false)
	dv.ShowThreads(false)
	if dv.Dbg.HasTasks() {
		dv.ShowTasks(false)
	}
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

// ShowFile shows the file name in gide
func (dv *DebugView) ShowFile(fname string, ln int) {
	if fname == "" || fname == "?" {
		return
	}
	// fmt.Printf("File: %s:%d\n", fname, ln)
	dv.Gide.ShowFile(fname, ln)
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

// ShowVar shows info on a given variable within the current frame scope in a text view dialog
// todo: replace with a treeview!
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
	dv.State.Status = stat
	cb := dv.CtrlBar()
	stl := cb.ChildByName("status", 1).(*gi.Label)
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
}

//////////////////////////////////////////////////////////////////////////////////////
//    GUI config

// Config configures the view -- parameters for the job must have
// already been set in ge.ProjParams.Debug.
func (dv *DebugView) Config(ge Gide, sup filecat.Supported, exePath string) {
	dv.Gide = ge
	dv.Sup = sup
	dv.ExePath = exePath
	dv.State.BlankState()
	dv.OutBuf = &giv.TextBuf{}
	dv.OutBuf.InitName(dv.OutBuf, "debug-outbuf")
	dv.Lay = gi.LayoutVert
	dv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "ctrlbar")
	config.Add(gi.KiT_TabView, "tabs")
	mods, updt := dv.ConfigChildren(config, ki.UniqueNames)
	if !mods {
		updt = dv.UpdateStart()
	}
	dv.ConfigToolbar()
	dv.ConfigTabs()
	dv.State.Breaks = nil // get rid of dummy
	dv.Start()
	dv.SetFullReRender()
	dv.UpdateEnd(updt)
}

// CtrlBar returns the find toolbar
func (dv *DebugView) CtrlBar() *gi.ToolBar {
	return dv.ChildByName("ctrlbar", 0).(*gi.ToolBar)
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

// VarVw returns the thread view from tabs
func (dv DebugView) VarVw() *VarsView {
	tv := dv.Tabs()
	return tv.TabByName("Vars").(*VarsView)
}

// TaskVw returns the thread view from tabs
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
	vv.Config(dv)
	if dv.Sup == filecat.Go { // dv.Dbg.HasTasks() { // todo: not avail here yet
		ta := tb.RecycleTab("Tasks", KiT_TaskView, false).(*TaskView)
		ta.Config(dv)
	}
	th := tb.RecycleTab("Threads", KiT_ThreadView, false).(*ThreadView)
	th.Config(dv)
	ff := tb.RecycleTab("Find Frames", KiT_StackView, false).(*StackView)
	ff.Config(dv, true) // find frames
}

// ActionActivate is the update function for actions that depend on the debugger being avail
// for input commands
func (dv *DebugView) ActionActivate(act *gi.Action) {
	act.SetActiveStateUpdt(dv.DbgIsAvail())
}

func (dv *DebugView) ConfigToolbar() {
	cb := dv.CtrlBar()
	if cb.HasChildren() {
		return
	}
	cb.SetStretchMaxWidth()

	// rb := dv.ReplBar()
	// rb.SetStretchMaxWidth()

	// cb.AddAction(gi.ActOpts{Label: "Updt", Icon: "update", Tooltip: "update current state"}, dv.This(),
	// 	func(recv, send ki.Ki, sig int64, data interface{}) {
	// 		dvv := recv.Embed(KiT_DebugView).(*DebugView)
	// 		dvv.UpdateView()
	// 		cb.UpdateActions()
	// 	})
	stl := gi.AddNewLabel(cb, "status", "Building..   ")
	stl.Redrawable = true
	stl.CurBgColor.SetString("yellow", nil)
	if gi.Prefs.IsDarkMode() {
		stl.CurBgColor = stl.CurBgColor.Darker(75)
	}
	cb.AddAction(gi.ActOpts{Label: "Restart", Icon: "update", Tooltip: "(re)start the debugger on exe:" + dv.ExePath}, dv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			dvv.Start()
			cb.UpdateActions()
		})
	cb.AddAction(gi.ActOpts{Label: "Cont", Icon: "play", Tooltip: "continue execution from current point", UpdateFunc: dv.ActionActivate}, dv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			go dvv.Continue()
			cb.UpdateActions()
		})
	gi.AddNewLabel(cb, "step", "Step: ")
	cb.AddAction(gi.ActOpts{Label: "Over", Icon: "step-over", Tooltip: "continues to the next source line, not entering function calls", UpdateFunc: dv.ActionActivate}, dv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			dvv.StepOver()
			cb.UpdateActions()
		})
	cb.AddAction(gi.ActOpts{Label: "Into", Icon: "step-into", Tooltip: "continues to the next source line, entering into function calls", UpdateFunc: dv.ActionActivate}, dv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			dvv.StepInto()
			cb.UpdateActions()
		})
	cb.AddAction(gi.ActOpts{Label: "Out", Icon: "step-out", Tooltip: "continues to the return point of the current function", UpdateFunc: dv.ActionActivate}, dv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			dvv.StepOut()
			cb.UpdateActions()
		})
	cb.AddAction(gi.ActOpts{Label: "Single", Icon: "step-fwd", Tooltip: "steps a single CPU instruction", UpdateFunc: dv.ActionActivate}, dv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			dvv.StepOut()
			cb.UpdateActions()
		})
	cb.AddAction(gi.ActOpts{Label: "Stop", Icon: "stop", Tooltip: "stop execution"}, dv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			dvv.Stop()
			cb.UpdateActions()
		})

}

// DebugViewProps are style properties for DebugView
var DebugViewProps = ki.Props{
	"EnumType:Flag": gi.KiT_NodeFlags,
	"max-width":     -1,
	"max-height":    -1,
}

//////////////////////////////////////////////////////////////////////////////////////
//  StackView

// StackView is a view of the stack trace
type StackView struct {
	gi.Layout
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
	mods, updt := sv.ConfigChildren(config, ki.UniqueNames)
	tv := sv.TableView()
	if mods {
		tv.SliceViewSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
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
	mods, updt := sv.ConfigChildren(config, ki.UniqueNames)
	tv := sv.TableView()
	if mods {
		tv.SliceViewSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(giv.SliceViewDoubleClicked) {
				idx := data.(int)
				dv.ShowBreakFile(idx)
			}
		})
	} else {
		updt = sv.UpdateStart()
	}
	tv.SetStretchMax()
	tv.NoAdd = true
	tv.NoDelete = true
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
	mods, updt := sv.ConfigChildren(config, ki.UniqueNames)
	tv := sv.TableView()
	if mods {
		tv.SliceViewSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
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
	mods, updt := sv.ConfigChildren(config, ki.UniqueNames)
	tv := sv.TableView()
	if mods {
		tv.SliceViewSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
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
}

var KiT_VarsView = kit.Types.AddType(&VarsView{}, VarsViewProps)

func (sv *VarsView) DebugVw() *DebugView {
	dv := sv.ParentByType(KiT_DebugView, ki.Embeds).Embed(KiT_DebugView).(*DebugView)
	return dv
}

func (sv *VarsView) Config(dv *DebugView) {
	sv.Lay = gi.LayoutVert
	config := kit.TypeAndNameList{}
	config.Add(giv.KiT_TableView, "vars")
	mods, updt := sv.ConfigChildren(config, ki.UniqueNames)
	tv := sv.TableView()
	if mods {
		tv.SliceViewSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(giv.SliceViewDoubleClicked) {
				idx := data.(int)
				vr := dv.State.Vars[idx]
				dv.ShowVar(vr.Nm)
			}
		})
	} else {
		updt = sv.UpdateStart()
	}
	tv.SetStretchMax()
	tv.SetInactive()
	tv.SetSlice(&dv.State.Vars)
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
	tv.SetSlice(&dv.State.Vars)
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
	Var       *gidebug.Variable `desc:"variable being edited"`
	FrameInfo string            `desc:"frame info"`
	DbgView   *DebugView        `json:"-" xml:"-" desc:"parent DebugView"`
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
	mods, updt := vv.ConfigChildren(config, ki.UniqueNames)
	vv.SetFrameInfo(vv.FrameInfo)
	vv.ConfigSplitView()
	// vv.ConfigToolbar()
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

// // ConfigToolbar adds a VarView toolbar.
// func (vv *VarView) ConfigToolbar() {
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
		tv.TreeViewSig.Connect(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
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
