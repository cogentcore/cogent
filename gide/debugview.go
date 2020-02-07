// Copyright (c) 2020, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"fmt"
	"log"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gide/gidebug"
	"github.com/goki/gide/gidebug/gidelve"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/filecat"
)

// todo:
// * select current depth, task, thread based on cur vals.
// * support proper thread / task abstractions -- currently using Task baked in.
// * breakpoints not updating properly, etc
// * highlight at end not working.
// * use a treeview for single-var view -- need a ki node version of variable
//   and potentially do incremental fetching, esp of pointers which seem like
//   the major issue..
// * also need to map to pi.syms.Kind so we know what kind of thing it is, and
//   customize display based on that
// * still getting weird focus issues.  how to do a complete reset?  why is it happening?
// * add button / filenode menu for choosing exe target

// Debuggers is the list of supported debuggers
var Debuggers = map[filecat.Supported]func(path, rootPath string, outbuf *giv.TextBuf, test bool, pars *gidebug.Params, startFunc func()) (gidebug.GiDebug, error){
	filecat.Go: func(path, rootPath string, outbuf *giv.TextBuf, test bool, pars *gidebug.Params, startFunc func()) (gidebug.GiDebug, error) {
		return gidelve.NewGiDelve(path, rootPath, outbuf, test, pars, startFunc)
	},
}

// NewDebugger returns a new debugger for given supported file type
func NewDebugger(sup filecat.Supported, path, rootPath string, outbuf *giv.TextBuf, test bool, pars *gidebug.Params, startFunc func()) (gidebug.GiDebug, error) {
	df, ok := Debuggers[sup]
	if !ok {
		err := fmt.Errorf("Gi Debug: File type %v not supported\n", sup)
		log.Println(err)
		return nil, err
	}
	dbg, err := df(path, rootPath, outbuf, test, pars, startFunc)
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
	Test    bool              `desc:"if true, run debugger on tests instead of an executable"`
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

// Detatch debugger on our death..
func (dv *DebugView) Destroy() {
	if dv.Dbg != nil {
		dv.Dbg.Detach(true)
	}
	dv.DeleteAllBreaks()
	dv.Layout.Destroy()
}

// DeleteAllBreaks deletes all breakpoints
func (dv *DebugView) DeleteAllBreaks() {
	if dv.Gide == nil || dv.Gide.IsDeleted() {
		return
	}
	for _, bk := range dv.State.Breaks {
		tb := dv.Gide.TextBufForFile(bk.FPath, false)
		if tb != nil {
			tb.DeleteLineColor(bk.Line)
			tb.DeleteLineIcon(bk.Line)
			tb.Refresh()
		}
	}
}

// Start starts the debuger
func (dv *DebugView) Start() {
	if dv.Dbg == nil {
		rootPath := ""
		var pars *gidebug.Params
		if dv.Gide != nil {
			rootPath = string(dv.Gide.ProjPrefs().ProjRoot)
			pars = &dv.Gide.ProjPrefs().Debug
		}
		dbg, err := NewDebugger(dv.Sup, dv.ExePath, rootPath, dv.OutBuf, dv.Test, pars, func() {
			dv.SetStatus("Ready")
		})
		if err == nil {
			dv.Dbg = dbg
		}
	} else {
		dv.Dbg.Restart()
		dv.SetStatus("Ready")
	}
}

// Continue continues running from current point -- this MUST be called
// in a separate goroutine!
func (dv *DebugView) Continue() {
	if !dv.DbgIsAvail() {
		return
	}
	dv.SetBreaks()
	dv.State.State.Running = true
	dv.SetStatus("Running")
	ds := <-dv.Dbg.Continue() // we wait here until it returns
	updt := dv.UpdateStart()
	dv.InitState(ds) // todo: do we need a mutex for this?  probably not
	dv.ShowStack(true)
	dv.UpdateEnd(updt)
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
		dv.State.DeleteBreakByFile(fpath, line)
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
	dv.SetStatus("Stopped")
	err := dv.Dbg.InitAllState(&dv.State)
	if err == gidebug.IsRunningErr {
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

// ShowFile shows the file name in gide
func (dv *DebugView) ShowFile(fname string, ln int) {
	if fname == "" || fname == "?" {
		return
	}
	// fmt.Printf("File: %s:%d\n", fname, ln)
	dv.Gide.ShowFile(fname, ln)
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
	// tinfo := vv.TypeInfo(true)
	// tinfo += "\n" + vv.ValueString(true, 0, 20, 4096)
	// prompt := ""
	// cf := dv.State.StackFrame(dv.State.CurFrame)
	// if cf != nil {
	// 	prompt = "at: " + cf.FPath + fmt.Sprintf(":%d  Thread: %d  Depth: %d", cf.Line, dv.State.CurTask, dv.State.CurFrame)
	// }
	// giv.TextViewDialog(dv.Viewport, []byte(tinfo), giv.DlgOpts{Title: "Variable: " + name, Prompt: prompt, Ok: true})
	ged := giv.GoGiEditorDialog(vv)
	ged.TreeView().CloseAll()
	return nil
}

func (dv *DebugView) SetStatus(stat string) {
	cb := dv.CtrlBar()
	stl := cb.Child(0).(*gi.Label)
	stl.SetText(stat)
}

//////////////////////////////////////////////////////////////////////////////////////
//    GUI config

// Config configures the view -- test = run in test mode
func (dv *DebugView) Config(ge Gide, sup filecat.Supported, exePath string, test bool) {
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

// BreakView returns the break view from tabs
func (dv DebugView) BreakVw() *BreakView {
	tv := dv.Tabs()
	return tv.TabByName("Breaks").(*BreakView)
}

// StackView returns the stack view from tabs
func (dv DebugView) StackVw() *StackView {
	tv := dv.Tabs()
	return tv.TabByName("Stack").(*StackView)
}

// VarView returns the thread view from tabs
func (dv DebugView) VarVw() *VarsView {
	tv := dv.Tabs()
	return tv.TabByName("Vars").(*VarsView)
}

// TaskView returns the thread view from tabs
func (dv DebugView) TaskVw() *TaskView {
	tv := dv.Tabs()
	return tv.TabByName("Tasks").(*TaskView)
}

// ThreadView returns the thread view from tabs
func (dv DebugView) ThreadVw() *ThreadView {
	tv := dv.Tabs()
	return tv.TabByName("Threads").(*ThreadView)
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
	// todo: set tabs as non-closable
	cv := tb.RecycleTab("Console", gi.KiT_Layout, false).(*gi.Layout)
	otv := ConfigOutputTextView(cv)
	otv.SetBuf(dv.OutBuf)
	bv := tb.RecycleTab("Breaks", KiT_BreakView, false).(*BreakView)
	bv.Config(dv)
	sv := tb.RecycleTab("Stack", KiT_StackView, false).(*StackView)
	sv.Config(dv)
	vv := tb.RecycleTab("Vars", KiT_VarsView, false).(*VarsView)
	vv.Config(dv)
	if dv.Sup == filecat.Go { // dv.Dbg.HasTasks() { // todo: not avail here yet
		ta := tb.RecycleTab("Tasks", KiT_TaskView, false).(*TaskView)
		ta.Config(dv)
	}
	th := tb.RecycleTab("Threads", KiT_ThreadView, false).(*ThreadView)
	th.Config(dv)
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

	stl := gi.AddNewLabel(cb, "status", "Building..   ")
	stl.Redrawable = true
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
}

var KiT_StackView = kit.Types.AddType(&StackView{}, StackViewProps)

func (sv *StackView) DebugVw() *DebugView {
	dv := sv.ParentByType(KiT_DebugView, ki.Embeds).Embed(KiT_DebugView).(*DebugView)
	return dv
}

func (sv *StackView) Config(dv *DebugView) {
	sv.Lay = gi.LayoutVert
	config := kit.TypeAndNameList{}
	config.Add(giv.KiT_TableView, "stack")
	mods, updt := sv.ConfigChildren(config, ki.UniqueNames)
	tv := sv.TableView()
	if mods {
		tv.SliceViewSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(giv.SliceViewDoubleClicked) {
				idx := data.(int)
				dv.SetFrame(idx)
			}
		})
	} else {
		updt = sv.UpdateStart()
	}
	tv.SetStretchMax()
	tv.SetInactive()
	tv.SetSlice(&dv.State.Stack)
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
	tv.SetSlice(&dv.State.Stack)
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
				bk := dv.State.Breaks[idx]
				dv.ShowFile(bk.FPath, bk.Line)
			}
		})
	} else {
		updt = sv.UpdateStart()
	}
	tv.SetStretchMax()
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
					th := dv.State.Threads[idx]
					dv.SetThread(th.ID)
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
				th := dv.State.Tasks[idx]
				dv.SetThread(th.ID)
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
