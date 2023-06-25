// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"fmt"
	"reflect"

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
	winm := "grid-prefs"
	width := 800
	height := 800
	win, recyc := gi.RecycleMainWindow(pf, winm, "Grid Preferences", width, height)
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
//  SplitsView

// SplitsView opens a view of a commands table
func SplitsView(pt *Splits) {
	winm := "grid-splits"
	width := 800
	height := 800
	win, recyc := gi.RecycleMainWindow(pt, winm, "Grid Splitter Settings", width, height)
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
	ac := vv.Widget.(*gi.Action)
	txt := kit.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none)"
	}
	ac.SetText(txt)
}

func (vv *SplitValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	ac := vv.Widget.(*gi.Action)
	ac.SetProp("border-radius", units.NewValue(4, units.Px))
	ac.ActionSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data any) {
		vvv, _ := recv.Embed(KiT_SplitValueView).(*SplitValueView)
		ac := vvv.Widget.(*gi.Action)
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
