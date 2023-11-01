// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"reflect"
	"strings"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/laser"
)

// KeyMapsView opens a view of a key maps table
func KeyMapsView(km *KeyMaps) {
	if gi.ActivateExistingMainWindow(km) {
		return
	}
	sc := gi.NewScene("gogi-key-maps")
	sc.Title = "Available Key Maps: Duplicate an existing map (using Ctxt Menu) as starting point for creating a custom map"
	sc.Lay = gi.LayoutVert
	sc.Data = km

	title := gi.NewLabel(sc, "title").SetText(sc.Title).SetType(gi.LabelHeadlineSmall)
	title.Style(func(s *styles.Style) {
		s.Width.Ch(30) // need for wrap
		s.SetStretchMaxWidth()
		s.Text.WhiteSpace = styles.WhiteSpaceNormal // wrap
	})

	tv := NewTableView(sc).SetSlice(km)
	tv.SetStretchMax()

	AvailKeyMapsChanged = false
	tv.OnChange(func(e events.Event) {
		AvailKeyMapsChanged = true
	})

	tb := tv.Toolbar()
	gi.NewSeparator(tb)
	sp := NewFuncButton(tb, km.SavePrefs).SetText("Save to preferences").SetIcon(icons.Save).SetKey(keyfun.Save)
	sp.SetUpdateFunc(func() {
		sp.SetEnabled(AvailMapsChanged && km == &AvailKeyMaps)
	})
	oj := NewFuncButton(tb, km.OpenJSON).SetText("Open from file").SetIcon(icons.Open).SetKey(keyfun.Open)
	oj.Args[0].SetTag("ext", ".json")
	sj := NewFuncButton(tb, km.SaveJSON).SetText("Save to file").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs)
	sj.Args[0].SetTag("ext", ".json")
	gi.NewSeparator(tb)
	vs := NewFuncButton(tb, ViewStdKeyMaps).SetConfirm(true).SetText("View standard").SetIcon(icons.Visibility)
	vs.SetUpdateFunc(func() {
		vs.SetEnabledUpdt(km != &StdKeyMaps)
	})
	rs := NewFuncButton(tb, km.RevertToStd).SetConfirm(true).SetText("Revert to standard").SetIcon(icons.DeviceReset)
	rs.SetUpdateFunc(func() {
		rs.SetEnabledUpdt(km != &StdKeyMaps)
	})
	tb.OverflowMenu().SetMenu(func(m *gi.Scene) {
		NewFuncButton(m, km.OpenPrefs).SetIcon(icons.Open).SetKey(keyfun.OpenAlt1)
	})

	gi.NewWindow(sc).Run()
}

//////////////////////////////////////////////////////////////////////////////////////
//  PrefsView

/*
// PrefsView opens a view of user preferences, returns structview and window
func PrefsView(pf *Preferences) *gi.Window {
	winm := "gide-prefs"
	width := 800
	height := 800
	win, recyc := gi.RecycleMainWindow(pf, winm, "Gide Preferences", width, height)
	if recyc {
		return win
	}

	sc := win.WinViewport2D()
	updt := sc.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	sv := mfr.NewChild(giv.KiT_StructView, "sv").(*giv.StructView)
	sv.Viewport = sc
	sv.SetStruct(pf)
	sv.SetStretchMaxWidth()
	sv.SetStretchMaxHeight()

	// mmen := win.MainMenu
	// giv.MainMenuView(pf, win, mmen)
	//
	// inClosePrompt := false
	// win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
	// 	if !pf.Changed {
	// 		win.Close()
	// 		return
	// 	}
	// 	if inClosePrompt {
	// 		return
	// 	}
	// 	inClosePrompt = true
	// 	gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Save Prefs Before Closing?",
	// 		Prompt: "Do you want to save any changes to preferences before closing?"},
	// 		[]string{"Save and Close", "Discard and Close", "Cancel"},
	// 		win.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 			switch sig {
	// 			case 0:
	// 				pf.Save()
	// 				fmt.Println("Preferences Saved to prefs.json")
	// 				win.Close()
	// 			case 1:
	// 				pf.Open() // if we don't do this, then it actually remains in edited state
	// 				win.Close()
	// 			case 2:
	// 				inClosePrompt = false
	// 				// default is to do nothing, i.e., cancel
	// 			}
	// 		})
	// })
	//
	// win.MainMenuUpdated()

	// if !win.HasGeomPrefs() { // resize to contents
	// 	vpsz := vp.PrefSize(win.OSWin.Screen().PixSize)
	// 	win.SetSize(vpsz)
	// }

	sc.UpdateEndNoSig(updt)
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

	sc := win.WinViewport2D()
	updt := sc.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	title := mfr.NewChild(gi.LabelType, "title").(*gi.Label)
	title.SetText("Project preferences are saved in the project .gide file, along with other current state (open directories, splitter settings, etc) -- do Save Project to save.")
	title.SetProp("width", units.NewValue(30, units.Ch)) // need for wrap
	title.SetStretchMaxWidth()
	title.SetProp("white-space", styles.WhiteSpaceNormal) // wrap

	sv := mfr.NewChild(giv.KiT_StructView, "sv").(*giv.StructView)
	sv.Viewport = sc
	sv.SetStruct(pf)
	sv.SetStretchMaxWidth()
	sv.SetStretchMaxHeight()

	// mmen := win.MainMenu
	// giv.MainMenuView(pf, win, mmen)
	//
	// win.MainMenuUpdated()
	//
	// 	if !win.HasGeomPrefs() { // resize to contents
	// 		vpsz := vp.PrefSize(win.OSWin.Screen().PixSize)
	// 		win.SetSize(vpsz)
	// 	}

	sc.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
	return sv, win
}

*/

////////////////////////////////////////////////////////////////////////////////////////
//  KeyMapValueView

// ValueView registers KeyMapValueView as the viewer of KeyMapName
func (kn KeyMapName) ValueView() giv.Value {
	vv := &KeyMapValueView{}
	ki.InitNode(vv)
	return vv
}

// KeyMapValueView presents an action for displaying an KeyMapName and selecting
// from KeyMapChooserDialog
type KeyMapValueView struct {
	giv.ValueBase
}

func (vv *KeyMapValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *KeyMapValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none -- click to set)"
	}
	ac.SetText(txt)
}

func (vv *KeyMapValueView) ConfigWidget(widg gi.Widget) {
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

func (vv *KeyMapValueView) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	if vv.IsInactive() {
		return
	}
	cur := laser.ToString(vv.Value.Interface())
	_, curRow, _ := AvailKeyMaps.MapByName(KeyMapName(cur))
	desc, _ := vv.Tag("desc")
	giv.TableViewSelectDialog(sc, &AvailKeyMaps, giv.DlgOpts{Title: "Select a KeyMap", Prompt: desc}, curRow, nil,
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

	sc := win.WinViewport2D()
	updt := sc.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	title := mfr.NewChild(gi.LabelType, "title").(*gi.Label)
	title.SetText("Available Language Opts: Add or modify entries to customize options for language / file types")
	title.SetProp("width", units.NewValue(30, units.Ch)) // need for wrap
	title.SetStretchMaxWidth()
	title.SetProp("white-space", styles.WhiteSpaceNormal) // wrap

	mv := mfr.NewChild(giv.KiT_MapView, "mv").(*giv.MapView)
	mv.Viewport = sc
	mv.SetMap(pt)
	mv.SetStretchMaxWidth()
	mv.SetStretchMaxHeight()

	AvailLangsChanged = false
	mv.ViewSig.Connect(mfr.This(), func(recv, send ki.Ki, sig int64, data any) {
		AvailLangsChanged = true
	})

	// mmen := win.MainMenu
	// giv.MainMenuView(pt, win, mmen)
	//
	// inClosePrompt := false
	// win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
	// 	if !AvailLangsChanged || pt != &AvailLangs { // only for main avail map..
	// 		win.Close()
	// 		return
	// 	}
	// 	if inClosePrompt {
	// 		return
	// 	}
	// 	inClosePrompt = true
	// 	gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Save Lang Opts Before Closing?",
	// 		Prompt: "Do you want to save any changes to preferences language options file before closing, or Cancel the close and do a Save to a different file?"},
	// 		[]string{"Save and Close", "Discard and Close", "Cancel"},
	// 		win.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 			switch sig {
	// 			case 0:
	// 				pt.SavePrefs()
	// 				fmt.Printf("Preferences Saved to %v\n", PrefsLangsFileName)
	// 				win.Close()
	// 			case 1:
	// 				pt.OpenPrefs() // revert
	// 				win.Close()
	// 			case 2:
	// 				inClosePrompt = true
	// 				// default is to do nothing, i.e., cancel
	// 			}
	// 		})
	// })
	//
	// win.MainMenuUpdated()
	//
	// if !win.HasGeomPrefs() { // resize to contents
	// 	vpsz := vp.PrefSize(win.OSWin.Screen().PixSize)
	// 	win.SetSize(vpsz)
	// }

	sc.UpdateEndNoSig(updt)
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

	sc := win.WinViewport2D()
	updt := sc.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	title := mfr.NewChild(gi.LabelType, "title").(*gi.Label)
	title.SetText("Gide Commands")
	title.SetProp("width", units.NewValue(30, units.Ch)) // need for wrap
	title.SetStretchMaxWidth()
	title.SetProp("white-space", styles.WhiteSpaceNormal) // wrap

	tv := mfr.NewChild(giv.KiT_TableView, "tv").(*giv.TableView)
	tv.Viewport = sc
	tv.SetSlice(pt)
	tv.SetStretchMaxWidth()
	tv.SetStretchMaxHeight()

	CustomCmdsChanged = false
	tv.ViewSig.Connect(mfr.This(), func(recv, send ki.Ki, sig int64, data any) {
		CustomCmdsChanged = true
	})

	// mmen := win.MainMenu
	// giv.MainMenuView(pt, win, mmen)
	//
	// inClosePrompt := false
	// win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
	// 	if !CustomCmdsChanged || pt != &CustomCmds { // only for main avail map..
	// 		win.Close()
	// 		return
	// 	}
	// 	if inClosePrompt {
	// 		return
	// 	}
	// 	inClosePrompt = true
	// 	gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Save Commands Before Closing?",
	// 		Prompt: "Do you want to save any changes to custom commands file before closing, or Cancel the close and do a Save to a different file?"},
	// 		[]string{"Save and Close", "Discard and Close", "Cancel"},
	// 		win.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 			switch sig {
	// 			case 0:
	// 				pt.SavePrefs()
	// 				fmt.Printf("Preferences Saved to %v\n", PrefsCmdsFileName)
	// 				win.Close()
	// 			case 1:
	// 				pt.OpenPrefs() // revert
	// 				win.Close()
	// 			case 2:
	// 				inClosePrompt = false
	// 				// default is to do nothing, i.e., cancel
	// 			}
	// 		})
	// })
	//
	// win.MainMenuUpdated()
	//
	// if !win.HasGeomPrefs() { // resize to contents
	// 	vpsz := vp.PrefSize(win.OSWin.Screen().PixSize)
	// 	win.SetSize(vpsz)
	// }

	sc.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
}

////////////////////////////////////////////////////////////////////////////////////////
//  CmdValueView

// ValueView registers CmdValueView as the viewer of CmdName
func (kn CmdName) ValueView() giv.Value {
	vv := &CmdValueView{}
	ki.InitNode(vv)
	return vv
}

// CmdValueView presents an action for displaying an CmdName and selecting
type CmdValueView struct {
	giv.ValueBase
}

func (vv *CmdValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *CmdValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none)"
	}
	ac.SetText(txt)
}

func (vv *CmdValueView) ConfigWidget(widg gi.Widget) {
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

func (vv *CmdValueView) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	if vv.IsInactive() {
		return
	}
	cur := laser.ToString(vv.Value.Interface())
	curRow := -1
	if cur != "" {
		_, curRow, _ = AvailCmds.CmdByName(CmdName(cur), false)
	}
	desc, _ := vv.Tag("desc")
	giv.TableViewSelectDialog(sc, &AvailCmds, giv.DlgOpts{Title: "Select a Command", Prompt: desc}, curRow, nil,
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

	sc := win.WinViewport2D()
	updt := sc.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	title := mfr.NewChild(gi.LabelType, "title").(*gi.Label)
	title.SetText("Available Splitter Settings: Can duplicate an existing (using Ctxt Menu) as starting point for new one")
	title.SetProp("width", units.NewValue(30, units.Ch)) // need for wrap
	title.SetStretchMaxWidth()
	title.SetProp("white-space", styles.WhiteSpaceNormal) // wrap

	tv := mfr.NewChild(giv.KiT_TableView, "tv").(*giv.TableView)
	tv.Viewport = sc
	tv.SetSlice(pt)
	tv.SetStretchMaxWidth()
	tv.SetStretchMaxHeight()

	AvailSplitsChanged = false
	tv.ViewSig.Connect(mfr.This(), func(recv, send ki.Ki, sig int64, data any) {
		AvailSplitsChanged = true
	})

	// mmen := win.MainMenu
	// giv.MainMenuView(pt, win, mmen)
	//
	// inClosePrompt := false
	// win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
	// 	if !AvailSplitsChanged || pt != &AvailSplits { // only for main avail map..
	// 		win.Close()
	// 		return
	// 	}
	// 	if inClosePrompt {
	// 		return
	// 	}
	// 	inClosePrompt = true
	// 	gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Save Splits Before Closing?",
	// 		Prompt: "Do you want to save any changes to custom splitter settings file before closing, or Cancel the close and do a Save to a different file?"},
	// 		[]string{"Save and Close", "Discard and Close", "Cancel"},
	// 		win.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 			switch sig {
	// 			case 0:
	// 				pt.SavePrefs()
	// 				fmt.Printf("Preferences Saved to %v\n", PrefsSplitsFileName)
	// 				win.Close()
	// 			case 1:
	// 				pt.OpenPrefs() // revert
	// 				win.Close()
	// 			case 2:
	// 				inClosePrompt = false
	// 				// default is to do nothing, i.e., cancel
	// 			}
	// 		})
	// })
	//
	// win.MainMenuUpdated()
	//
	// if !win.HasGeomPrefs() { // resize to contents
	// 	vpsz := vp.PrefSize(win.OSWin.Screen().PixSize)
	// 	win.SetSize(vpsz)
	// }

	sc.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
}

////////////////////////////////////////////////////////////////////////////////////////
//  SplitValueView

// ValueView registers SplitValueView as the viewer of SplitName
func (kn SplitName) ValueView() giv.Value {
	vv := &SplitValueView{}
	ki.InitNode(vv)
	return vv
}

// SplitValueView presents an action for displaying an SplitName and selecting
type SplitValueView struct {
	giv.ValueBase
}

func (vv *SplitValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *SplitValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none)"
	}
	ac.SetText(txt)
}

func (vv *SplitValueView) ConfigWidget(widg gi.Widget) {
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

func (vv *SplitValueView) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	if vv.IsInactive() {
		return
	}
	cur := laser.ToString(vv.Value.Interface())
	curRow := -1
	if cur != "" {
		_, curRow, _ = AvailSplits.SplitByName(SplitName(cur))
	}
	desc, _ := vv.Tag("desc")
	giv.TableViewSelectDialog(sc, &AvailSplits, giv.DlgOpts{Title: "Select a Named Splitter Config", Prompt: desc}, curRow, nil,
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

	sc := win.WinViewport2D()
	updt := sc.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	title := mfr.NewChild(gi.LabelType, "title").(*gi.Label)
	title.SetText("Available Registers: Can duplicate an existing (using Ctxt Menu) as starting point for new one")
	title.SetProp("width", units.NewValue(30, units.Ch)) // need for wrap
	title.SetStretchMaxWidth()
	title.SetProp("white-space", styles.WhiteSpaceNormal) // wrap

	tv := mfr.NewChild(giv.KiT_MapView, "tv").(*giv.MapView)
	tv.Viewport = sc
	tv.SetMap(pt)
	tv.SetStretchMaxWidth()
	tv.SetStretchMaxHeight()

	AvailRegistersChanged = false
	tv.ViewSig.Connect(mfr.This(), func(recv, send ki.Ki, sig int64, data any) {
		AvailRegistersChanged = true
	})

	// mmen := win.MainMenu
	// giv.MainMenuView(pt, win, mmen)
	//
	// inClosePrompt := false
	// win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
	// 	if !AvailRegistersChanged || pt != &AvailRegisters { // only for main avail map..
	// 		win.Close()
	// 		return
	// 	}
	// 	if inClosePrompt {
	// 		return
	// 	}
	// 	inClosePrompt = true
	// 	gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Save Registers Before Closing?",
	// 		Prompt: "Do you want to save any changes to custom register file before closing, or Cancel the close and do a Save to a different file?"},
	// 		[]string{"Save and Close", "Discard and Close", "Cancel"},
	// 		win.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 			switch sig {
	// 			case 0:
	// 				pt.SavePrefs()
	// 				fmt.Printf("Preferences Saved to %v\n", PrefsRegistersFileName)
	// 				win.Close()
	// 			case 1:
	// 				pt.OpenPrefs() // revert
	// 				win.Close()
	// 			case 2:
	// 				inClosePrompt = false
	// 				// default is to do nothing, i.e., cancel
	// 			}
	// 		})
	// })
	//
	// win.MainMenuUpdated()
	//
	// if !win.HasGeomPrefs() { // resize to contents
	// 	vpsz := vp.PrefSize(win.OSWin.Screen().PixSize)
	// 	win.SetSize(vpsz)
	// }

	sc.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
}

////////////////////////////////////////////////////////////////////////////////////////
//  RegisterValueView

// ValueView registers RegisterValueView as the viewer of RegisterName
func (kn RegisterName) ValueView() giv.Value {
	vv := &RegisterValueView{}
	ki.InitNode(vv)
	return vv
}

// RegisterValueView presents an action for displaying an RegisterName and selecting
type RegisterValueView struct {
	giv.ValueBase
}

func (vv *RegisterValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *RegisterValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none)"
	}
	ac.SetText(txt)
}

func (vv *RegisterValueView) ConfigWidget(widg gi.Widget) {
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

func (vv *RegisterValueView) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	if vv.IsInactive() {
		return
	}
	cur := laser.ToString(vv.Value.Interface())
	var recv gi.Widget
	if vv.Widget != nil {
		recv = vv.Widget
	} else {
		recv = sc.This().(gi.Widget)
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
