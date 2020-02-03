// Copyright (c) 2020, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"fmt"
	"log"

	"github.com/goki/gi/gi"
	"github.com/goki/gide/gidebug"
	"github.com/goki/gide/gidebug/gidelve"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/filecat"
)

var Debuggers = map[filecat.Supported]func(path string) (gidebug.GiDebug, error){
	filecat.Go: func(path string) (gidebug.GiDebug, error) {
		return gidelve.NewGiDelve(path)
	},
}

func NewDebugger(sup filecat.Supported, path string) (gidebug.GiDebug, error) {
	df, ok := Debuggers[sup]
	if !ok {
		err := fmt.Errorf("Gi Debug: File type %v not supported\n", sup)
		log.Println(err)
		return nil, err
	}
	dbg, err := df(path)
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
	Gide    Gide              `json:"-" xml:"-" desc:"parent gide project"`
}

var KiT_DebugView = kit.Types.AddType(&DebugView{}, DebugViewProps)

// Start starts the debuger
func (dv *DebugView) Start() {
	if dv.Dbg == nil {
		dbg, err := NewDebugger(dv.Sup, dv.ExePath)
		if err == nil {
			dv.Dbg = dbg
		}
	} else {
		dv.Dbg.Restart()
	}
}

// Continue continues running from current point
func (dv *DebugView) Continue() {
	ds := <-dv.Dbg.Continue()
	fmt.Printf("%v\n", ds)
}

// Next step
func (dv *DebugView) Next() {
	dv.Dbg.Next()
}

// Step step
func (dv *DebugView) Step() {
	dv.Dbg.Step()
}

// Stop
func (dv *DebugView) Stop() {
	dv.Dbg.Halt()
}

//////////////////////////////////////////////////////////////////////////////////////
//    GUI config

// Config configures the view
func (dv *DebugView) Config(ge Gide, sup filecat.Supported, exePath string) {
	dv.Gide = ge
	dv.Sup = sup
	dv.ExePath = exePath
	dv.Start()
	dv.Lay = gi.LayoutVert
	dv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "ctrlbar")
	// config.Add(gi.KiT_ToolBar, "replbar")
	config.Add(gi.KiT_TabView, "tabs")
	mods, updt := dv.ConfigChildren(config, ki.NonUniqueNames)
	if !mods {
		updt = dv.UpdateStart()
	}
	dv.ConfigToolbar()
	// ft := dv.DebugText()
	// // ft.ItemsFromStringList(dv.Params().DebugHist, true, 0)
	// // ft.SetText(dv.Params().Debug)
	// rt := dv.ReplText()
	// // rt.ItemsFromStringList(dv.Params().ReplHist, true, 0)
	// // rt.SetText(dv.Params().Replace)
	// ib := dv.IgnoreBox()
	// // ib.SetChecked(dv.Params().IgnoreCase)
	// cf := dv.LocCombo()
	// // cf.SetCurIndex(int(dv.Params().Loc))
	dv.UpdateEnd(updt)
}

// CtrlBar returns the find toolbar
func (dv *DebugView) CtrlBar() *gi.ToolBar {
	return dv.ChildByName("ctrlbar", 0).(*gi.ToolBar)
}

// ConfigToolbar adds toolbar.
func (dv *DebugView) ConfigToolbar() {
	cb := dv.CtrlBar()
	if cb.HasChildren() {
		return
	}
	cb.SetStretchMaxWidth()

	// rb := dv.ReplBar()
	// rb.SetStretchMaxWidth()

	cb.AddAction(gi.ActOpts{Icon: "play", Tooltip: "(re)start the debugger on" + dv.ExePath}, dv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			dvv.Start()
			go dvv.Continue()
		})
	cb.AddAction(gi.ActOpts{Icon: "fast-fwd", Tooltip: "step to next source line, skipping over methods"}, dv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			dvv.Next()
		})
	cb.AddAction(gi.ActOpts{Icon: "step-fwd", Tooltip: "step to next instruction"}, dv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			dvv.Step()
		})
	cb.AddAction(gi.ActOpts{Icon: "stop", Tooltip: "stop execution"}, dv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			dvv := recv.Embed(KiT_DebugView).(*DebugView)
			dvv.Stop()
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
