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

var Debuggers = map[filecat.Supported]func(path, rootPath string, outbuf *giv.TextBuf) (gidebug.GiDebug, error){
	filecat.Go: func(path, rootPath string, outbuf *giv.TextBuf) (gidebug.GiDebug, error) {
		return gidelve.NewGiDelve(path, rootPath, outbuf)
	},
}

func NewDebugger(sup filecat.Supported, path, rootPath string, outbuf *giv.TextBuf) (gidebug.GiDebug, error) {
	df, ok := Debuggers[sup]
	if !ok {
		err := fmt.Errorf("Gi Debug: File type %v not supported\n", sup)
		log.Println(err)
		return nil, err
	}
	dbg, err := df(path, rootPath, outbuf)
	if err != nil {
		log.Println(err)
	}
	return dbg, err
}

// DebugParams are parameters for the debugger
type DebugParams struct {
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

// Detatch debugger on our death..
func (dv *DebugView) Destroy() {
	if dv.Dbg != nil {
		dv.Dbg.Detach(true)
	}
	dv.Layout.Destroy()
}

// Start starts the debuger
func (dv *DebugView) Start() {
	if dv.Dbg == nil {
		rootPath := ""
		if dv.Gide != nil {
			rootPath = string(dv.Gide.ProjPrefs().ProjRoot)
		}
		dbg, err := NewDebugger(dv.Sup, dv.ExePath, rootPath, dv.OutBuf)
		if err == nil {
			dv.Dbg = dbg
		}
	} else {
		dv.Dbg.Restart()
		go dv.Continue()
	}
}

// Continue continues running from current point
func (dv *DebugView) Continue() {
	ds := <-dv.Dbg.Continue()
	dv.UpdateFmState(ds)
}

// Next step
func (dv *DebugView) Next() {
	ds, err := dv.Dbg.Next()
	if err != nil {
		return
	}
	dv.UpdateFmState(ds)
}

// Step step
func (dv *DebugView) Step() {
	ds, err := dv.Dbg.Step()
	if err != nil {
		return
	}
	dv.UpdateFmState(ds)
}

// Stop
func (dv *DebugView) Stop() {
	ds, err := dv.Dbg.Stop()
	if err != nil {
		return
	}
	dv.UpdateFmState(ds)
}

// SetBreak
func (dv *DebugView) SetBreak(fname string, line int) {
	// fmt.Printf("set bp: %v:%v\n", fname, line)
	_, err := dv.Dbg.SetBreak(fname, line)
	if err != nil {
		return
	}
	// dv.UpdateFmState(ds)
}

// ClearBreak
func (dv *DebugView) ClearBreak(fname string, line int) {
	// fmt.Printf("set bp: %v:%v\n", fname, line)
	// _, err := dv.Dbg.ClearBreakpoint(fname, line)
	// if err != nil {
	// 	return
	// }
	// dv.UpdateFmState(ds)
}

// UpdateFmState updates the View from given debug state
func (dv *DebugView) UpdateFmState(ds *gidebug.State) {
	if ds.Running {
		return
	}
	err := dv.Dbg.InitAllState(&dv.State)
	if err == gidebug.IsRunningErr {
		return
	}
	cf := dv.State.CurStackFrame()
	if cf != nil {
		dv.ShowFile(cf.Loc.FPath, cf.Loc.Line)
		dv.ShowStack()
	}
}

// ShowFile shows the file name in gide
func (dv *DebugView) ShowFile(fname string, ln int) {
	// fmt.Printf("File: %s:%d\n", fname, ln)
	dv.Gide.ShowFile(fname, ln)
}

// ShowStack shows the current stack
func (dv *DebugView) ShowStack() {
	sv := dv.StackVw()
	sv.ShowStack()
}

//////////////////////////////////////////////////////////////////////////////////////
//    GUI config

// Config configures the view
func (dv *DebugView) Config(ge Gide, sup filecat.Supported, exePath string) {
	dv.Gide = ge
	dv.Sup = sup
	dv.ExePath = exePath
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
	dv.Start()
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

// StackView returns the stack view from tabs
func (dv DebugView) StackVw() *StackView {
	tv := dv.Tabs()
	return tv.TabByName("Stack").(*StackView)
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
	cv := tb.RecycleTab("Console", gi.KiT_Layout, false).(*gi.Layout)
	otv := ConfigOutputTextView(cv)
	otv.SetBuf(dv.OutBuf)
	sv := tb.RecycleTab("Stack", KiT_StackView, false).(*StackView)
	sv.Config()
}

// DebuggerIsActive is used for updating active state of toolbar
func (dv *DebugView) DebuggerIsActive() bool {
	if dv.Dbg != nil && dv.Dbg.IsActive() {
		return true
	}
	return false
}

// ActionActivate is the update function for actions that depend on the debugger being running
func (dv *DebugView) ActionActivate(act *gi.Action) {
	act.SetActiveStateUpdt(dv.DebuggerIsActive())
}

func (dv *DebugView) ConfigToolbar() {
	cb := dv.CtrlBar()
	if cb.HasChildren() {
		return
	}
	cb.SetStretchMaxWidth()

	// rb := dv.ReplBar()
	// rb.SetStretchMaxWidth()

	cb.AddAction(gi.ActOpts{Icon: "update", Tooltip: "(re)start the debugger on exe:" + dv.ExePath}, dv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			dvv.Start()
			cb.UpdateActions()
		})
	cb.AddAction(gi.ActOpts{Icon: "play", Tooltip: "continue execution from current point"}, dv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			dvv.Continue()
			cb.UpdateActions()
		})
	cb.AddAction(gi.ActOpts{Icon: "fast-fwd", Tooltip: "step to next source line, skipping over methods", UpdateFunc: dv.ActionActivate}, dv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			dvv.Next()
			cb.UpdateActions()
		})
	cb.AddAction(gi.ActOpts{Icon: "step-fwd", Tooltip: "step to next instruction", UpdateFunc: dv.ActionActivate}, dv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			dvv.Step()
			cb.UpdateActions()
		})
	cb.AddAction(gi.ActOpts{Icon: "stop", Tooltip: "stop execution", UpdateFunc: dv.ActionActivate}, dv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			dvv.Stop()
			cb.UpdateActions()
		})

}

// DebugViewProps are style properties for DebugView
var DebugViewProps = ki.Props{
	"EnumType:Flag":    gi.KiT_NodeFlags,
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
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

func (sv *StackView) Config() {
	sv.Lay = gi.LayoutVert
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "toolbar")
	config.Add(giv.KiT_TableView, "stack")
	mods, updt := sv.ConfigChildren(config, ki.UniqueNames)
	if !mods {
		updt = sv.UpdateStart()
	}
	// sv.ConfigToolbar()
	tv := sv.TableView()
	sv.SetProp("inactive", true)
	tv.SetStretchMax()
	tv.SetInactive()
	dv := sv.DebugVw()
	tv.SetSlice(&dv.State.Stack)
	tv.SetInactive()
	sv.UpdateEnd(updt)
}

// CtrlBar returns the find toolbar
func (sv *StackView) ToolBar() *gi.ToolBar {
	return sv.ChildByName("toolbar", 0).(*gi.ToolBar)
}

// TableView returns the tableview
func (sv *StackView) TableView() *giv.TableView {
	return sv.ChildByName("stack", 1).(*giv.TableView)
}

// ShowStack triggers update of view of State.Stack
func (sv *StackView) ShowStack() {
	tv := sv.TableView()
	dv := sv.DebugVw()
	dv.SetFullReRender()
	updt := dv.UpdateStart()
	tv.SetInactive()
	tv.SetSlice(&dv.State.Stack)
	tv.SetInactive()
	dv.UpdateEnd(updt)
}

// StackViewProps are style properties for DebugView
var StackViewProps = ki.Props{
	"EnumType:Flag":    gi.KiT_NodeFlags,
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
}
