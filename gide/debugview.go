// Copyright (c) 2020, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"time"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/filecat"
)

// DebugParams are parameters for find / replace
type DebugParams struct {
	Debug      string              `desc:"find string"`
	Replace    string              `desc:"replace string"`
	IgnoreCase bool                `desc:"ignore case"`
	Langs      []filecat.Supported `desc:"languages for files to search"`
	Loc        FindLoc             `desc:"locations to search in"`
	DebugHist  []string            `desc:"history of finds"`
	ReplHist   []string            `desc:"history of replaces"`
}

// DebugView is a find / replace widget that displays results in a TextView
// and has a toolbar for controlling find / replace process.
type DebugView struct {
	gi.Layout
	Gide   Gide          `json:"-" xml:"-" desc:"parent gide project"`
	LangVV giv.ValueView `desc:"langs value view"`
	Time   time.Time     `desc:"time of last find"`
}

var KiT_DebugView = kit.Types.AddType(&DebugView{}, DebugViewProps)

// DebugAction runs a new find with current params
func (fv *DebugView) DebugAction() {
}

// ReplaceAction performs the replace
func (fv *DebugView) ReplaceAction() bool {
	return false
}

// ReplaceAllAction performs replace all
func (fv *DebugView) ReplaceAllAction() {
}

// NextDebug shows next find result
func (fv *DebugView) NextDebug() {
}

// PrevDebug shows previous find result
func (fv *DebugView) PrevDebug() {
}

// OpenDebugURL opens given find:/// url from Debug
func (fv *DebugView) OpenDebugURL(ur string, ftv *giv.TextView) bool {
	return false
}

// HighlightDebugs highlights all the find results in ftv buffer
func (fv *DebugView) HighlightDebugs(tv, ftv *giv.TextView, fbStLn, fCount int, find string) {
}

//////////////////////////////////////////////////////////////////////////////////////
//    GUI config

// Config configures the view
func (fv *DebugView) Config(ge Gide) {
	fv.Gide = ge
	fv.Lay = gi.LayoutVert
	fv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "findbar")
	config.Add(gi.KiT_ToolBar, "replbar")
	config.Add(gi.KiT_Layout, "findtext")
	mods, updt := fv.ConfigChildren(config, ki.NonUniqueNames)
	if !mods {
		updt = fv.UpdateStart()
	}
	fv.ConfigToolbar()
	// ft := fv.DebugText()
	// // ft.ItemsFromStringList(fv.Params().DebugHist, true, 0)
	// // ft.SetText(fv.Params().Debug)
	// rt := fv.ReplText()
	// // rt.ItemsFromStringList(fv.Params().ReplHist, true, 0)
	// // rt.SetText(fv.Params().Replace)
	// ib := fv.IgnoreBox()
	// // ib.SetChecked(fv.Params().IgnoreCase)
	// cf := fv.LocCombo()
	// // cf.SetCurIndex(int(fv.Params().Loc))
	tvly := fv.TextViewLay()
	fv.Gide.ConfigOutputTextView(tvly)
	if mods {
		na := fv.DebugNextAct()
		na.GrabFocus()
	}
	fv.UpdateEnd(updt)
}

// DebugBar returns the find toolbar
func (fv *DebugView) DebugBar() *gi.ToolBar {
	return fv.ChildByName("findbar", 0).(*gi.ToolBar)
}

// ReplBar returns the replace toolbar
func (fv *DebugView) ReplBar() *gi.ToolBar {
	return fv.ChildByName("replbar", 1).(*gi.ToolBar)
}

// DebugText returns the find textfield in toolbar
func (fv *DebugView) DebugText() *gi.ComboBox {
	return fv.DebugBar().ChildByName("find-str", 1).(*gi.ComboBox)
}

// ReplText returns the replace textfield in toolbar
func (fv *DebugView) ReplText() *gi.ComboBox {
	return fv.ReplBar().ChildByName("repl-str", 1).(*gi.ComboBox)
}

// IgnoreBox returns the ignore case checkbox in toolbar
func (fv *DebugView) IgnoreBox() *gi.CheckBox {
	return fv.DebugBar().ChildByName("ignore-case", 2).(*gi.CheckBox)
}

// LocCombo returns the loc combobox
func (fv *DebugView) LocCombo() *gi.ComboBox {
	return fv.ReplBar().ChildByName("loc", 5).(*gi.ComboBox)
}

// CurDirBox returns the cur file checkbox in toolbar
func (fv *DebugView) CurDirBox() *gi.CheckBox {
	return fv.ReplBar().ChildByName("cur-dir", 6).(*gi.CheckBox)
}

// DebugNextAct returns the find next action in toolbar -- selected first
func (fv *DebugView) DebugNextAct() *gi.Action {
	return fv.DebugBar().ChildByName("next", 3).(*gi.Action)
}

// TextViewLay returns the find results TextView layout
func (fv *DebugView) TextViewLay() *gi.Layout {
	return fv.ChildByName("findtext", 1).(*gi.Layout)
}

// TextView returns the find results TextView
func (fv *DebugView) TextView() *giv.TextView {
	return fv.TextViewLay().Child(0).Embed(giv.KiT_TextView).(*giv.TextView)
}

// ConfigToolbar adds toolbar.
func (fv *DebugView) ConfigToolbar() {
	fb := fv.DebugBar()
	if fb.HasChildren() {
		return
	}
	fb.SetStretchMaxWidth()

	rb := fv.ReplBar()
	rb.SetStretchMaxWidth()

	fb.AddAction(gi.ActOpts{Label: "Debug:", Tooltip: "Debug given string in project files. Only open folders in file browser will be searched -- adjust those to scope the search"},
		fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			fvv, _ := recv.Embed(KiT_DebugView).(*DebugView)
			fvv.DebugAction()
		})

	finds := fb.AddNewChild(gi.KiT_ComboBox, "find-str").(*gi.ComboBox)
	finds.Editable = true
	finds.SetStretchMaxWidth()
	finds.Tooltip = "String to find -- hit enter or tab to update search -- click for history"
	finds.ConfigParts()
	// finds.ItemsFromStringList(fv.Params().DebugHist, true, 0)
	ftf, _ := finds.TextField()
	finds.ComboSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.Embed(KiT_DebugView).(*DebugView)
		// cb := send.(*gi.ComboBox)
		// fvv.Params().Debug = cb.CurVal.(string)
		fvv.DebugAction()
	})
	ftf.TextFieldSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) || sig == int64(gi.TextFieldDeFocused) {
			fvv, _ := recv.Embed(KiT_DebugView).(*DebugView)
			// tf := send.(*gi.TextField)
			// fvv.Params().Debug = tf.Text()
			fvv.DebugAction()
		} else if sig == int64(gi.TextFieldCleared) {
			tv := fv.Gide.ActiveTextView()
			if tv != nil {
				tv.ClearHighlights()
			}
			fvtv := fv.TextView()
			if fvtv != nil {
				fvtv.Buf.New(0)
			}
		}
	})

	ic := fb.AddNewChild(gi.KiT_CheckBox, "ignore-case").(*gi.CheckBox)
	ic.SetText("Ignore Case")
	ic.ButtonSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonToggled) {
			// fvv, _ := recv.Embed(KiT_DebugView).(*DebugView)
			// cb := send.(*gi.CheckBox)
			// fvv.Params().IgnoreCase = cb.IsChecked()
		}
	})

	fb.AddAction(gi.ActOpts{Name: "next", Icon: "wedge-down", Tooltip: "go to next result"},
		fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			fvv, _ := recv.Embed(KiT_DebugView).(*DebugView)
			fvv.NextDebug()
		})

	fb.AddAction(gi.ActOpts{Name: "prev", Icon: "wedge-up", Tooltip: "go to previous result"},
		fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			fvv, _ := recv.Embed(KiT_DebugView).(*DebugView)
			fvv.PrevDebug()
		})

	rb.AddAction(gi.ActOpts{Label: "Replace:", Tooltip: "Replace find string with replace string for currently-selected find result"}, fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.Embed(KiT_DebugView).(*DebugView)
		fvv.ReplaceAction()
	})

	repls := rb.AddNewChild(gi.KiT_ComboBox, "repl-str").(*gi.ComboBox)
	repls.Editable = true
	repls.SetStretchMaxWidth()
	repls.Tooltip = "String to replace find string -- click for history"
	repls.ConfigParts()
	// repls.ItemsFromStringList(fv.Params().ReplHist, true, 0)
	rtf, _ := repls.TextField()
	repls.ComboSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		// fvv, _ := recv.Embed(KiT_DebugView).(*DebugView)
		// cb := send.(*gi.ComboBox)
		// fvv.Params().Replace = cb.CurVal.(string)
	})
	rtf.TextFieldSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) {
			// fvv, _ := recv.Embed(KiT_DebugView).(*DebugView)
			// tf := send.(*gi.TextField)
			// fvv.Params().Replace = tf.Text()
		}
	})

	rb.AddAction(gi.ActOpts{Label: "All", Tooltip: "replace all find strings with replace string"},
		fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			fvv, _ := recv.Embed(KiT_DebugView).(*DebugView)
			fvv.ReplaceAllAction()
		})

	locl := rb.AddNewChild(gi.KiT_Label, "loc-lbl").(*gi.Label)
	locl.SetText("Loc:")
	locl.Tooltip = "location to find in: all = all open folders in browser; file = current active file; dir = directory of current active file; nottop = all except the top-level in browser"
	// locl.SetProp("vertical-align", gi.AlignMiddle)

	cf := rb.AddNewChild(gi.KiT_ComboBox, "loc").(*gi.ComboBox)
	cf.SetText("Loc")
	cf.Tooltip = locl.Tooltip
	// cf.ItemsFromEnum(KiT_DebugLoc, false, 0)
	cf.ComboSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		// fvv, _ := recv.Embed(KiT_DebugView).(*DebugView)
		// cb := send.(*gi.ComboBox)
		// eval := cb.CurVal.(kit.EnumValue)
		// fvv.Params().Loc = DebugLoc(eval.Value)
	})

	langl := rb.AddNewChild(gi.KiT_Label, "lang-lbl").(*gi.Label)
	langl.SetText("Lang:")
	langl.Tooltip = "Language(s) to restrict search / replace to"
	// langl.SetProp("vertical-align", gi.AlignMiddle)

	// fv.LangVV = giv.ToValueView(&fv.Params().Langs, "")
	// fv.LangVV.SetSoloValue(reflect.ValueOf(&fv.Params().Langs))
	vtyp := fv.LangVV.WidgetType()
	langw := rb.AddNewChild(vtyp, "langs").(gi.Node2D)
	fv.LangVV.ConfigWidget(langw)
	langw.AsWidget().Tooltip = langl.Tooltip
	//	vvb := vv.AsValueViewBase()
	//	vvb.ViewSig.ConnectOnly(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
	//		fvv, _ := recv.Embed(KiT_DebugView).(*DebugView)
	// hmm, langs updated..
	//	})

}

// DebugViewProps are style properties for DebugView
var DebugViewProps = ki.Props{
	"EnumType:Flag":    gi.KiT_NodeFlags,
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
}
