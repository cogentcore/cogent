// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

//////////////////////////////////////////////////////////////////////////////////////
//  PrefsView

// PrefsView opens a view of user preferences, returns structview and window
func PrefsView(pf *Preferences) *gi.Window {
	winm := "gide-prefs"
	width := 800
	height := 800
	win, recyc := gi.RecycleMainWindow(pf, winm, "Gide Preferences", width, height)
	if recyc {
		return win
	}

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	sv := mfr.AddNewChild(giv.KiT_StructView, "sv").(*giv.StructView)
	sv.Viewport = vp
	sv.SetStruct(pf)
	sv.SetStretchMaxWidth()
	sv.SetStretchMaxHeight()

	mmen := win.MainMenu
	giv.MainMenuView(pf, win, mmen)

	inClosePrompt := false
	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		if !pf.Changed {
			win.Close()
			return
		}
		if inClosePrompt {
			return
		}
		inClosePrompt = true
		gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Save Prefs Before Closing?",
			Prompt: "Do you want to save any changes to preferences before closing?"},
			[]string{"Save and Close", "Discard and Close", "Cancel"},
			win.This(), func(recv, send ki.Ki, sig int64, data any) {
				switch sig {
				case 0:
					pf.Save()
					fmt.Println("Preferences Saved to prefs.json")
					win.Close()
				case 1:
					pf.Open() // if we don't do this, then it actually remains in edited state
					win.Close()
				case 2:
					inClosePrompt = false
					// default is to do nothing, i.e., cancel
				}
			})
	})

	win.MainMenuUpdated()

	if !win.HasGeomPrefs() { // resize to contents
		vpsz := vp.PrefSize(win.OSWin.Screen().PixSize)
		win.SetSize(vpsz)
	}

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
	return win
}

//////////////////////////////////////////////////////////////////////////////////////
//  ProjPrefsView

// ProjPrefsView opens a view of project preferences (settings), returns structview and window
func ProjPrefsView(pf *ProjPrefs) (*giv.StructView, *gi.Window) {
	winm := "gide-proj-prefs"

	width := 800
	height := 800
	win, recyc := gi.RecycleMainWindow(pf, winm, "Gide Project Preferences", width, height)
	if recyc {
		mfr, _ := win.MainFrame()
		sv := mfr.Child(1).(*giv.StructView)
		return sv, win
	}

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	title := mfr.AddNewChild(gi.KiT_Label, "title").(*gi.Label)
	title.SetText("Project preferences are saved in the project .gide file, along with other current state (open directories, splitter settings, etc) -- do Save Project to save.")
	title.SetProp("width", units.NewValue(30, units.Ch)) // need for wrap
	title.SetStretchMaxWidth()
	title.SetProp("white-space", gist.WhiteSpaceNormal) // wrap

	sv := mfr.AddNewChild(giv.KiT_StructView, "sv").(*giv.StructView)
	sv.Viewport = vp
	sv.SetStruct(pf)
	sv.SetStretchMaxWidth()
	sv.SetStretchMaxHeight()

	mmen := win.MainMenu
	giv.MainMenuView(pf, win, mmen)

	win.MainMenuUpdated()

	if !win.HasGeomPrefs() { // resize to contents
		vpsz := vp.PrefSize(win.OSWin.Screen().PixSize)
		win.SetSize(vpsz)
	}

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
	return sv, win
}

//////////////////////////////////////////////////////////////////////////////////////
//  KeyMapsView

// KeyMapsView opens a view of a key maps table
func KeyMapsView(km *KeyMaps) {
	winm := "gide-key-maps"
	width := 800
	height := 800
	win, recycle := gi.RecycleMainWindow(km, winm, "Gide Key Maps", width, height)
	if recycle {
		return
	}

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	title := mfr.AddNewChild(gi.KiT_Label, "title").(*gi.Label)
	title.SetText("Available Key Maps: Duplicate an existing map (using Ctxt Menu) as starting point for creating a custom map")
	title.SetProp("width", units.NewValue(30, units.Ch)) // need for wrap
	title.SetStretchMaxWidth()
	title.SetProp("white-space", gist.WhiteSpaceNormal) // wrap

	tv := mfr.AddNewChild(giv.KiT_TableView, "tv").(*giv.TableView)
	tv.Viewport = vp
	tv.SetSlice(km)
	tv.SetStretchMaxWidth()
	tv.SetStretchMaxHeight()

	AvailKeyMapsChanged = false
	tv.ViewSig.Connect(mfr.This(), func(recv, send ki.Ki, sig int64, data any) {
		AvailKeyMapsChanged = true
	})

	mmen := win.MainMenu
	giv.MainMenuView(km, win, mmen)

	inClosePrompt := false
	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		if !AvailKeyMapsChanged || km != &AvailKeyMaps { // only for main avail map..
			win.Close()
			return
		}
		if inClosePrompt {
			return
		}
		inClosePrompt = true
		gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Save KeyMaps Before Closing?",
			Prompt: "Do you want to save any changes to preferences keymaps file before closing, or Cancel the close and do a Save to a different file?"},
			[]string{"Save and Close", "Discard and Close", "Cancel"},
			win.This(), func(recv, send ki.Ki, sig int64, data any) {
				switch sig {
				case 0:
					km.SavePrefs()
					fmt.Printf("Preferences Saved to %v\n", PrefsKeyMapsFileName)
					win.Close()
				case 1:
					km.OpenPrefs() // revert
					win.Close()
				case 2:
					inClosePrompt = false
					// default is to do nothing, i.e., cancel
				}
			})
	})

	win.MainMenuUpdated()

	if !win.HasGeomPrefs() { // resize to contents
		vpsz := vp.PrefSize(win.OSWin.Screen().PixSize)
		win.SetSize(vpsz)
	}

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
}

////////////////////////////////////////////////////////////////////////////////////////
//  KeyMapValueView

// ValueView registers KeyMapValueView as the viewer of KeyMapName
func (kn KeyMapName) ValueView() giv.ValueView {
	vv := &KeyMapValueView{}
	ki.InitNode(vv)
	return vv
}

// KeyMapValueView presents an action for displaying an KeyMapName and selecting
// from KeyMapChooserDialog
type KeyMapValueView struct {
	giv.ValueViewBase
}

var KiT_KeyMapValueView = kit.Types.AddType(&KeyMapValueView{}, nil)

func (vv *KeyMapValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_Action
	return vv.WidgetTyp
}

func (vv *KeyMapValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Button)
	txt := kit.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none -- click to set)"
	}
	ac.SetText(txt)
}

func (vv *KeyMapValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	ac := vv.Widget.(*gi.Button)
	ac.SetProp("border-radius", units.NewValue(4, units.Px))
	ac.ActionSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data any) {
		vvv, _ := recv.Embed(KiT_KeyMapValueView).(*KeyMapValueView)
		ac := vvv.Widget.(*gi.Button)
		vvv.Activate(ac.Viewport, nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *KeyMapValueView) HasAction() bool {
	return true
}

func (vv *KeyMapValueView) Activate(vp *gi.Viewport2D, dlgRecv ki.Ki, dlgFunc ki.RecvFunc) {
	if vv.IsInactive() {
		return
	}
	cur := kit.ToString(vv.Value.Interface())
	_, curRow, _ := AvailKeyMaps.MapByName(KeyMapName(cur))
	desc, _ := vv.Tag("desc")
	giv.TableViewSelectDialog(vp, &AvailKeyMaps, giv.DlgOpts{Title: "Select a KeyMap", Prompt: desc}, curRow, nil,
		vv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.DialogAccepted) {
				ddlg, _ := send.(*gi.Dialog)
				si := giv.TableViewSelectDialogValue(ddlg)
				if si >= 0 {
					km := AvailKeyMaps[si]
					vv.SetValue(km.Name)
					vv.UpdateWidget()
				}
			}
			if dlgRecv != nil && dlgFunc != nil {
				dlgFunc(dlgRecv, send, sig, data)
			}
		})
}

//////////////////////////////////////////////////////////////////////////////////////
//  LangsView

// LangsView opens a view of a languages options map
func LangsView(pt *Langs) {
	winm := "gide-langs"
	width := 800
	height := 800
	win, recyc := gi.RecycleMainWindow(pt, winm, "Gide Languages", width, height)
	if recyc {
		return
	}

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	title := mfr.AddNewChild(gi.KiT_Label, "title").(*gi.Label)
	title.SetText("Available Language Opts: Add or modify entries to customize options for language / file types")
	title.SetProp("width", units.NewValue(30, units.Ch)) // need for wrap
	title.SetStretchMaxWidth()
	title.SetProp("white-space", gist.WhiteSpaceNormal) // wrap

	mv := mfr.AddNewChild(giv.KiT_MapView, "mv").(*giv.MapView)
	mv.Viewport = vp
	mv.SetMap(pt)
	mv.SetStretchMaxWidth()
	mv.SetStretchMaxHeight()

	AvailLangsChanged = false
	mv.ViewSig.Connect(mfr.This(), func(recv, send ki.Ki, sig int64, data any) {
		AvailLangsChanged = true
	})

	mmen := win.MainMenu
	giv.MainMenuView(pt, win, mmen)

	inClosePrompt := false
	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		if !AvailLangsChanged || pt != &AvailLangs { // only for main avail map..
			win.Close()
			return
		}
		if inClosePrompt {
			return
		}
		inClosePrompt = true
		gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Save Lang Opts Before Closing?",
			Prompt: "Do you want to save any changes to preferences language options file before closing, or Cancel the close and do a Save to a different file?"},
			[]string{"Save and Close", "Discard and Close", "Cancel"},
			win.This(), func(recv, send ki.Ki, sig int64, data any) {
				switch sig {
				case 0:
					pt.SavePrefs()
					fmt.Printf("Preferences Saved to %v\n", PrefsLangsFileName)
					win.Close()
				case 1:
					pt.OpenPrefs() // revert
					win.Close()
				case 2:
					inClosePrompt = true
					// default is to do nothing, i.e., cancel
				}
			})
	})

	win.MainMenuUpdated()

	if !win.HasGeomPrefs() { // resize to contents
		vpsz := vp.PrefSize(win.OSWin.Screen().PixSize)
		win.SetSize(vpsz)
	}

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
}

//////////////////////////////////////////////////////////////////////////////////////
//  CmdsView

// CmdsView opens a view of a commands table
func CmdsView(pt *Commands) {
	winm := "gide-commands"
	width := 800
	height := 800
	win, recyc := gi.RecycleMainWindow(pt, winm, "Gide Commands", width, height)
	if recyc {
		return
	}

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	title := mfr.AddNewChild(gi.KiT_Label, "title").(*gi.Label)
	title.SetText("Gide Commands")
	title.SetProp("width", units.NewValue(30, units.Ch)) // need for wrap
	title.SetStretchMaxWidth()
	title.SetProp("white-space", gist.WhiteSpaceNormal) // wrap

	tv := mfr.AddNewChild(giv.KiT_TableView, "tv").(*giv.TableView)
	tv.Viewport = vp
	tv.SetSlice(pt)
	tv.SetStretchMaxWidth()
	tv.SetStretchMaxHeight()

	CustomCmdsChanged = false
	tv.ViewSig.Connect(mfr.This(), func(recv, send ki.Ki, sig int64, data any) {
		CustomCmdsChanged = true
	})

	mmen := win.MainMenu
	giv.MainMenuView(pt, win, mmen)

	inClosePrompt := false
	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		if !CustomCmdsChanged || pt != &CustomCmds { // only for main avail map..
			win.Close()
			return
		}
		if inClosePrompt {
			return
		}
		inClosePrompt = true
		gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Save Commands Before Closing?",
			Prompt: "Do you want to save any changes to custom commands file before closing, or Cancel the close and do a Save to a different file?"},
			[]string{"Save and Close", "Discard and Close", "Cancel"},
			win.This(), func(recv, send ki.Ki, sig int64, data any) {
				switch sig {
				case 0:
					pt.SavePrefs()
					fmt.Printf("Preferences Saved to %v\n", PrefsCmdsFileName)
					win.Close()
				case 1:
					pt.OpenPrefs() // revert
					win.Close()
				case 2:
					inClosePrompt = false
					// default is to do nothing, i.e., cancel
				}
			})
	})

	win.MainMenuUpdated()

	if !win.HasGeomPrefs() { // resize to contents
		vpsz := vp.PrefSize(win.OSWin.Screen().PixSize)
		win.SetSize(vpsz)
	}

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
}

////////////////////////////////////////////////////////////////////////////////////////
//  CmdValueView

// ValueView registers CmdValueView as the viewer of CmdName
func (kn CmdName) ValueView() giv.ValueView {
	vv := &CmdValueView{}
	ki.InitNode(vv)
	return vv
}

// CmdValueView presents an action for displaying an CmdName and selecting
type CmdValueView struct {
	giv.ValueViewBase
}

var KiT_CmdValueView = kit.Types.AddType(&CmdValueView{}, nil)

func (vv *CmdValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_Action
	return vv.WidgetTyp
}

func (vv *CmdValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Button)
	txt := kit.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none)"
	}
	ac.SetText(txt)
}

func (vv *CmdValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	ac := vv.Widget.(*gi.Button)
	ac.SetProp("border-radius", units.NewValue(4, units.Px))
	ac.ActionSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data any) {
		vvv, _ := recv.Embed(KiT_CmdValueView).(*CmdValueView)
		ac := vvv.Widget.(*gi.Button)
		vvv.Activate(ac.Viewport, nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *CmdValueView) HasAction() bool {
	return true
}

func (vv *CmdValueView) Activate(vp *gi.Viewport2D, dlgRecv ki.Ki, dlgFunc ki.RecvFunc) {
	if vv.IsInactive() {
		return
	}
	cur := kit.ToString(vv.Value.Interface())
	curRow := -1
	if cur != "" {
		_, curRow, _ = AvailCmds.CmdByName(CmdName(cur), false)
	}
	desc, _ := vv.Tag("desc")
	giv.TableViewSelectDialog(vp, &AvailCmds, giv.DlgOpts{Title: "Select a Command", Prompt: desc}, curRow, nil,
		vv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.DialogAccepted) {
				ddlg, _ := send.(*gi.Dialog)
				si := giv.TableViewSelectDialogValue(ddlg)
				if si >= 0 {
					pt := AvailCmds[si]
					vv.SetValue(CommandName(pt.Cat, pt.Name))
					vv.UpdateWidget()
				}
			}
			if dlgRecv != nil && dlgFunc != nil {
				dlgFunc(dlgRecv, send, sig, data)
			}
		})

}

//////////////////////////////////////////////////////////////////////////////////////
//  SplitsView

// SplitsView opens a view of a commands table
func SplitsView(pt *Splits) {
	winm := "gide-splits"
	width := 800
	height := 800
	win, recyc := gi.RecycleMainWindow(pt, winm, "Gide Splitter Settings", width, height)
	if recyc {
		return
	}

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	title := mfr.AddNewChild(gi.KiT_Label, "title").(*gi.Label)
	title.SetText("Available Splitter Settings: Can duplicate an existing (using Ctxt Menu) as starting point for new one")
	title.SetProp("width", units.NewValue(30, units.Ch)) // need for wrap
	title.SetStretchMaxWidth()
	title.SetProp("white-space", gist.WhiteSpaceNormal) // wrap

	tv := mfr.AddNewChild(giv.KiT_TableView, "tv").(*giv.TableView)
	tv.Viewport = vp
	tv.SetSlice(pt)
	tv.SetStretchMaxWidth()
	tv.SetStretchMaxHeight()

	AvailSplitsChanged = false
	tv.ViewSig.Connect(mfr.This(), func(recv, send ki.Ki, sig int64, data any) {
		AvailSplitsChanged = true
	})

	mmen := win.MainMenu
	giv.MainMenuView(pt, win, mmen)

	inClosePrompt := false
	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		if !AvailSplitsChanged || pt != &AvailSplits { // only for main avail map..
			win.Close()
			return
		}
		if inClosePrompt {
			return
		}
		inClosePrompt = true
		gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Save Splits Before Closing?",
			Prompt: "Do you want to save any changes to custom splitter settings file before closing, or Cancel the close and do a Save to a different file?"},
			[]string{"Save and Close", "Discard and Close", "Cancel"},
			win.This(), func(recv, send ki.Ki, sig int64, data any) {
				switch sig {
				case 0:
					pt.SavePrefs()
					fmt.Printf("Preferences Saved to %v\n", PrefsSplitsFileName)
					win.Close()
				case 1:
					pt.OpenPrefs() // revert
					win.Close()
				case 2:
					inClosePrompt = false
					// default is to do nothing, i.e., cancel
				}
			})
	})

	win.MainMenuUpdated()

	if !win.HasGeomPrefs() { // resize to contents
		vpsz := vp.PrefSize(win.OSWin.Screen().PixSize)
		win.SetSize(vpsz)
	}

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
}

////////////////////////////////////////////////////////////////////////////////////////
//  SplitValueView

// ValueView registers SplitValueView as the viewer of SplitName
func (kn SplitName) ValueView() giv.ValueView {
	vv := &SplitValueView{}
	ki.InitNode(vv)
	return vv
}

// SplitValueView presents an action for displaying an SplitName and selecting
type SplitValueView struct {
	giv.ValueViewBase
}

var KiT_SplitValueView = kit.Types.AddType(&SplitValueView{}, nil)

func (vv *SplitValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_Action
	return vv.WidgetTyp
}

func (vv *SplitValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Button)
	txt := kit.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none)"
	}
	ac.SetText(txt)
}

func (vv *SplitValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	ac := vv.Widget.(*gi.Button)
	ac.SetProp("border-radius", units.NewValue(4, units.Px))
	ac.ActionSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data any) {
		vvv, _ := recv.Embed(KiT_SplitValueView).(*SplitValueView)
		ac := vvv.Widget.(*gi.Button)
		vvv.Activate(ac.Viewport, nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *SplitValueView) HasAction() bool {
	return true
}

func (vv *SplitValueView) Activate(vp *gi.Viewport2D, dlgRecv ki.Ki, dlgFunc ki.RecvFunc) {
	if vv.IsInactive() {
		return
	}
	cur := kit.ToString(vv.Value.Interface())
	curRow := -1
	if cur != "" {
		_, curRow, _ = AvailSplits.SplitByName(SplitName(cur))
	}
	desc, _ := vv.Tag("desc")
	giv.TableViewSelectDialog(vp, &AvailSplits, giv.DlgOpts{Title: "Select a Named Splitter Config", Prompt: desc}, curRow, nil,
		vv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.DialogAccepted) {
				ddlg, _ := send.(*gi.Dialog)
				si := giv.TableViewSelectDialogValue(ddlg)
				if si >= 0 {
					pt := AvailSplits[si]
					vv.SetValue(pt.Name)
					vv.UpdateWidget()
				}
			}
			if dlgRecv != nil && dlgFunc != nil {
				dlgFunc(dlgRecv, send, sig, data)
			}
		})

}

//////////////////////////////////////////////////////////////////////////////////////
//  RegistersView

// RegistersView opens a view of a commands table
func RegistersView(pt *Registers) {
	winm := "gide-registers"
	width := 800
	height := 800
	win, recyc := gi.RecycleMainWindow(pt, winm, "Gide Registers", width, height)
	if recyc {
		return
	}

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	title := mfr.AddNewChild(gi.KiT_Label, "title").(*gi.Label)
	title.SetText("Available Registers: Can duplicate an existing (using Ctxt Menu) as starting point for new one")
	title.SetProp("width", units.NewValue(30, units.Ch)) // need for wrap
	title.SetStretchMaxWidth()
	title.SetProp("white-space", gist.WhiteSpaceNormal) // wrap

	tv := mfr.AddNewChild(giv.KiT_MapView, "tv").(*giv.MapView)
	tv.Viewport = vp
	tv.SetMap(pt)
	tv.SetStretchMaxWidth()
	tv.SetStretchMaxHeight()

	AvailRegistersChanged = false
	tv.ViewSig.Connect(mfr.This(), func(recv, send ki.Ki, sig int64, data any) {
		AvailRegistersChanged = true
	})

	mmen := win.MainMenu
	giv.MainMenuView(pt, win, mmen)

	inClosePrompt := false
	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		if !AvailRegistersChanged || pt != &AvailRegisters { // only for main avail map..
			win.Close()
			return
		}
		if inClosePrompt {
			return
		}
		inClosePrompt = true
		gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Save Registers Before Closing?",
			Prompt: "Do you want to save any changes to custom register file before closing, or Cancel the close and do a Save to a different file?"},
			[]string{"Save and Close", "Discard and Close", "Cancel"},
			win.This(), func(recv, send ki.Ki, sig int64, data any) {
				switch sig {
				case 0:
					pt.SavePrefs()
					fmt.Printf("Preferences Saved to %v\n", PrefsRegistersFileName)
					win.Close()
				case 1:
					pt.OpenPrefs() // revert
					win.Close()
				case 2:
					inClosePrompt = false
					// default is to do nothing, i.e., cancel
				}
			})
	})

	win.MainMenuUpdated()

	if !win.HasGeomPrefs() { // resize to contents
		vpsz := vp.PrefSize(win.OSWin.Screen().PixSize)
		win.SetSize(vpsz)
	}

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
}

////////////////////////////////////////////////////////////////////////////////////////
//  RegisterValueView

// ValueView registers RegisterValueView as the viewer of RegisterName
func (kn RegisterName) ValueView() giv.ValueView {
	vv := &RegisterValueView{}
	ki.InitNode(vv)
	return vv
}

// RegisterValueView presents an action for displaying an RegisterName and selecting
type RegisterValueView struct {
	giv.ValueViewBase
}

var KiT_RegisterValueView = kit.Types.AddType(&RegisterValueView{}, nil)

func (vv *RegisterValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_Action
	return vv.WidgetTyp
}

func (vv *RegisterValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Button)
	txt := kit.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none)"
	}
	ac.SetText(txt)
}

func (vv *RegisterValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	ac := vv.Widget.(*gi.Button)
	ac.SetProp("border-radius", units.NewValue(4, units.Px))
	ac.ActionSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data any) {
		vvv, _ := recv.Embed(KiT_RegisterValueView).(*RegisterValueView)
		ac := vvv.Widget.(*gi.Button)
		vvv.Activate(ac.Viewport, nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *RegisterValueView) HasAction() bool {
	return true
}

func (vv *RegisterValueView) Activate(vp *gi.Viewport2D, dlgRecv ki.Ki, dlgFunc ki.RecvFunc) {
	if vv.IsInactive() {
		return
	}
	cur := kit.ToString(vv.Value.Interface())
	var recv gi.Node2D
	if vv.Widget != nil {
		recv = vv.Widget
	} else {
		recv = vp.This().(gi.Node2D)
	}
	gi.StringsChooserPopup(AvailRegisterNames, cur, recv, func(recv, send ki.Ki, sig int64, data any) {
		ac := send.(*gi.Button)
		rnm := ac.Text
		if ci := strings.Index(rnm, ":"); ci > 0 {
			rnm = rnm[:ci]
		}
		vv.SetValue(rnm)
		vv.UpdateWidget()
		if dlgRecv != nil && dlgFunc != nil {
			dlgFunc(dlgRecv, send, int64(gi.DialogAccepted), data)
		}
	})

}
