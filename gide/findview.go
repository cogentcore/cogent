// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"bytes"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/filecat"
)

// FindLoc corresponds to the search scope
type FindLoc int

const (
	// FindLocAll finds in all open folders in the left file browser
	FindLocAll FindLoc = iota

	// FindLocFile only finds in the current active file
	FindLocFile

	// FindLocDir only finds in the directory of the current active file
	FindLocDir

	// FindLocNotTop finds in all open folders *except* the top-level folder
	FindLocNotTop

	// FindLocN is the number of find locations (scopes)
	FindLocN
)

//go:generate stringer -type=FindLoc

var KiT_FindLoc = kit.Enums.AddEnumAltLower(FindLocN, false, nil, "FindLoc")

// MarshalJSON encodes
func (ev FindLoc) MarshalJSON() ([]byte, error) { return kit.EnumMarshalJSON(ev) }

// UnmarshalJSON decodes
func (ev *FindLoc) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// FindParams are parameters for find / replace
type FindParams struct {
	Find       string              `desc:"find string"`
	Replace    string              `desc:"replace string"`
	IgnoreCase bool                `desc:"ignore case"`
	Langs      []filecat.Supported `desc:"languages for files to search"`
	Loc        FindLoc             `desc:"locations to search in"`
	FindHist   []string            `desc:"history of finds"`
	ReplHist   []string            `desc:"history of replaces"`
}

// FindView is a find / replace widget that displays results in a TextView
// and has a toolbar for controlling find / replace process.
type FindView struct {
	gi.Layout
	Gide   Gide          `json:"-" xml:"-" desc:"parent gide project"`
	LangVV giv.ValueView `desc:"langs value view"`
	Time   time.Time     `desc:"time of last find"`
}

var KiT_FindView = kit.Types.AddType(&FindView{}, FindViewProps)

// Params returns the find params
func (fv *FindView) Params() *FindParams {
	return &fv.Gide.ProjPrefs().Find
}

// SaveFindString saves the given find string to the find params history and current str
func (fv *FindView) SaveFindString(find string) {
	fv.Params().Find = find
	gi.StringsInsertFirstUnique(&fv.Params().FindHist, find, gi.Prefs.SavedPathsMax)
	ftc := fv.FindText()
	if ftc != nil {
		ftc.ItemsFromStringList(fv.Params().FindHist, true, 0)
	}
}

// SaveReplString saves the given replace string to the find params history and current str
func (fv *FindView) SaveReplString(repl string) {
	fv.Params().Replace = repl
	gi.StringsInsertFirstUnique(&fv.Params().ReplHist, repl, gi.Prefs.SavedPathsMax)
	rtc := fv.ReplText()
	if rtc != nil {
		rtc.ItemsFromStringList(fv.Params().ReplHist, true, 0)
	}
}

// FindAction runs a new find with current params
func (fv *FindView) FindAction() {
	fv.SaveFindString(fv.Params().Find)
	fv.Gide.Find(fv.Params().Find, fv.Params().Replace, fv.Params().IgnoreCase, fv.Params().Loc, fv.Params().Langs)
}

// ReplaceAction performs the replace
func (fv *FindView) ReplaceAction() bool {
	winUpdt := fv.Gide.VPort().Win.UpdateStart()
	defer fv.Gide.VPort().Win.UpdateEnd(winUpdt)

	fv.SaveReplString(fv.Params().Replace)
	gi.StringsInsertFirstUnique(&fv.Params().ReplHist, fv.Params().Replace, gi.Prefs.SavedPathsMax)

	ftv := fv.TextView()
	tl, ok := ftv.OpenLinkAt(ftv.CursorPos)
	if !ok {
		ok = ftv.CursorNextLink(false) // no wrap
		if !ok {
			return false
		}
		tl, ok = ftv.OpenLinkAt(ftv.CursorPos)
		if !ok {
			return false
		}
	}
	ge := fv.Gide
	tv, reg, _, _, ok := ge.ParseOpenFindURL(tl.URL, ftv)
	if !ok {
		return false
	}
	if reg.IsNil() {
		ok = ftv.CursorNextLink(false) // no wrap
		if !ok {
			return false
		}
		tl, ok = ftv.OpenLinkAt(ftv.CursorPos)
		if !ok {
			return false
		}
		tv, reg, _, _, ok = ge.ParseOpenFindURL(tl.URL, ftv)
		if !ok || reg.IsNil() {
			return false
		}
	}
	reg.Time.SetTime(fv.Time)
	reg = tv.Buf.AdjustReg(reg)
	if !reg.IsNil() {
		tv.RefreshIfNeeded()
		tbe := tv.Buf.DeleteText(reg.Start, reg.End, true, true)
		tv.Buf.InsertText(tbe.Reg.Start, []byte(fv.Params().Replace), true, true)

		// delete the link for the just done replace
		ftvln := ftv.CursorPos.Ln
		st := giv.TextPos{Ln: ftvln, Ch: 0}
		len := len(ftv.Buf.Lines[ftvln])
		en := giv.TextPos{Ln: ftvln, Ch: len}
		ftv.Buf.DeleteText(st, en, false, true)
		// ftv.NeedsRefresh()
	}

	tv.Highlights = nil
	tv.NeedsRefresh()

	ok = ftv.CursorNextLink(false) // no wrap
	if ok {
		ftv.OpenLinkAt(ftv.CursorPos) // move to next
	}
	return ok
}

// ReplaceAllAction performs replace all
func (fv *FindView) ReplaceAllAction() {
	for {
		ok := fv.ReplaceAction()
		if !ok {
			break
		}
	}
}

// NextFind shows next find result
func (fv *FindView) NextFind() {
	ftv := fv.TextView()
	ok := ftv.CursorNextLink(true) // wrap
	if ok {
		ftv.OpenLinkAt(ftv.CursorPos)
	}
}

// PrevFind shows previous find result
func (fv *FindView) PrevFind() {
	ftv := fv.TextView()
	ok := ftv.CursorPrevLink(true) // wrap
	if ok {
		ftv.OpenLinkAt(ftv.CursorPos)
	}
}

// OpenFindURL opens given find:/// url from Find
func (fv *FindView) OpenFindURL(ur string, ftv *giv.TextView) bool {
	ge := fv.Gide
	tv, reg, fbBufStLn, fCount, ok := ge.ParseOpenFindURL(ur, ftv)
	if !ok {
		return false
	}
	reg.Time.SetTime(fv.Time)
	reg = tv.Buf.AdjustReg(reg)
	find := fv.Params().Find
	giv.PrevISearchString = find
	tve := tv.Embed(giv.KiT_TextView).(*giv.TextView)
	fv.HighlightFinds(tve, ftv, fbBufStLn, fCount, find)
	tv.SetNeedsRefresh()
	tv.RefreshIfNeeded()
	tv.SetCursorShow(reg.Start)
	return true
}

// HighlightFinds highlights all the find results in ftv buffer
func (fv *FindView) HighlightFinds(tv, ftv *giv.TextView, fbStLn, fCount int, find string) {
	lnka := []byte(`<a href="`)
	lnkasz := len(lnka)

	fb := ftv.Buf

	if len(tv.Highlights) != fCount { // highlight
		hi := make([]giv.TextRegion, fCount)
		for i := 0; i < fCount; i++ {
			fln := fbStLn + 1 + i
			ltxt := fb.Markup[fln]
			fpi := bytes.Index(ltxt, lnka)
			if fpi < 0 {
				continue
			}
			fpi += lnkasz
			epi := fpi + bytes.Index(ltxt[fpi:], []byte(`"`))
			lnk := string(ltxt[fpi:epi])
			iup, err := url.Parse(lnk)
			if err != nil {
				continue
			}
			ireg := giv.TextRegion{}
			lidx := strings.Index(iup.Fragment, "L")
			ireg.FromString(iup.Fragment[lidx:])
			ireg.Time.SetTime(fv.Time)
			hi[i] = ireg
		}
		tv.Highlights = hi
	}
}

//////////////////////////////////////////////////////////////////////////////////////
//    GUI config

// UpdateView updates view with current settings
func (fv *FindView) UpdateView(ge Gide) {
	fv.Gide = ge
	mods, updt := fv.StdFindConfig()
	fv.ConfigToolbar()
	ft := fv.FindText()
	ft.ItemsFromStringList(fv.Params().FindHist, true, 0)
	ft.SetText(fv.Params().Find)
	rt := fv.ReplText()
	rt.ItemsFromStringList(fv.Params().ReplHist, true, 0)
	rt.SetText(fv.Params().Replace)
	ib := fv.IgnoreBox()
	ib.SetChecked(fv.Params().IgnoreCase)
	cf := fv.LocCombo()
	cf.SetCurIndex(int(fv.Params().Loc))
	tvly := fv.TextViewLay()
	fv.Gide.ConfigOutputTextView(tvly)
	if mods {
		na := fv.FindNextAct()
		na.GrabFocus()
		fv.UpdateEnd(updt)
	}
}

// StdConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (fv *FindView) StdConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "findbar")
	config.Add(gi.KiT_ToolBar, "replbar")
	config.Add(gi.KiT_Layout, "findtext")
	return config
}

// StdFindConfig configures a standard setup of the overall layout -- returns
// mods, updt from ConfigChildren and does NOT call UpdateEnd
func (fv *FindView) StdFindConfig() (mods, updt bool) {
	fv.Lay = gi.LayoutVert
	fv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := fv.StdConfig()
	mods, updt = fv.ConfigChildren(config, false)
	return
}

// FindBar returns the find toolbar
func (fv *FindView) FindBar() *gi.ToolBar {
	tbi, ok := fv.ChildByName("findbar", 0)
	if !ok {
		return nil
	}
	return tbi.(*gi.ToolBar)
}

// ReplBar returns the replace toolbar
func (fv *FindView) ReplBar() *gi.ToolBar {
	tbi, ok := fv.ChildByName("replbar", 1)
	if !ok {
		return nil
	}
	return tbi.(*gi.ToolBar)
}

// FindText returns the find textfield in toolbar
func (fv *FindView) FindText() *gi.ComboBox {
	tb := fv.FindBar()
	if tb == nil {
		return nil
	}
	tfi, ok := tb.ChildByName("find-str", 1)
	if !ok {
		return nil
	}
	return tfi.(*gi.ComboBox)
}

// ReplText returns the replace textfield in toolbar
func (fv *FindView) ReplText() *gi.ComboBox {
	tb := fv.ReplBar()
	if tb == nil {
		return nil
	}
	tfi, ok := tb.ChildByName("repl-str", 1)
	if !ok {
		return nil
	}
	return tfi.(*gi.ComboBox)
}

// IgnoreBox returns the ignore case checkbox in toolbar
func (fv *FindView) IgnoreBox() *gi.CheckBox {
	tb := fv.FindBar()
	if tb == nil {
		return nil
	}
	tfi, ok := tb.ChildByName("ignore-case", 2)
	if !ok {
		return nil
	}
	return tfi.(*gi.CheckBox)
}

// LocCombo returns the loc combobox
func (fv *FindView) LocCombo() *gi.ComboBox {
	tb := fv.ReplBar()
	if tb == nil {
		return nil
	}
	tfi, ok := tb.ChildByName("loc", 5)
	if !ok {
		return nil
	}
	return tfi.(*gi.ComboBox)
}

// CurDirBox returns the cur file checkbox in toolbar
func (fv *FindView) CurDirBox() *gi.CheckBox {
	tb := fv.ReplBar()
	if tb == nil {
		return nil
	}
	tfi, ok := tb.ChildByName("cur-dir", 6)
	if !ok {
		return nil
	}
	return tfi.(*gi.CheckBox)
}

// FindNextAct returns the find next action in toolbar -- selected first
func (fv *FindView) FindNextAct() *gi.Action {
	tb := fv.FindBar()
	if tb == nil {
		return nil
	}
	tfi, ok := tb.ChildByName("next", 3)
	if !ok {
		return nil
	}
	return tfi.(*gi.Action)
}

// TextViewLay returns the find results TextView layout
func (fv *FindView) TextViewLay() *gi.Layout {
	tvi, ok := fv.ChildByName("findtext", 1)
	if !ok {
		return nil
	}
	return tvi.(*gi.Layout)
}

// TextView returns the find results TextView
func (fv *FindView) TextView() *giv.TextView {
	tvly := fv.TextViewLay()
	if tvly == nil {
		return nil
	}
	tv := tvly.KnownChild(0).Embed(giv.KiT_TextView).(*giv.TextView)
	return tv
}

// ConfigToolbar adds toolbar.
func (fv *FindView) ConfigToolbar() {
	fb := fv.FindBar()
	if fb.HasChildren() {
		return
	}
	fb.SetStretchMaxWidth()

	rb := fv.ReplBar()
	rb.SetStretchMaxWidth()

	finda := fb.AddNewChild(gi.KiT_Action, "find-act").(*gi.Action)
	finda.SetText("Find:")
	finda.Tooltip = "Find given string in project files. Only open folders in file browser will be searched -- adjust those to scope the search"
	finda.ActionSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.Embed(KiT_FindView).(*FindView)
		fvv.FindAction()
	})

	finds := fb.AddNewChild(gi.KiT_ComboBox, "find-str").(*gi.ComboBox)
	finds.Editable = true
	finds.SetStretchMaxWidth()
	finds.Tooltip = "String to find -- hit enter or tab to update search -- click for history"
	finds.ConfigParts()
	finds.ItemsFromStringList(fv.Params().FindHist, true, 0)
	ftf, _ := finds.TextField()
	finds.ComboSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.Embed(KiT_FindView).(*FindView)
		cb := send.(*gi.ComboBox)
		fvv.Params().Find = cb.CurVal.(string)
		fvv.FindAction()
	})
	ftf.TextFieldSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) || sig == int64(gi.TextFieldDeFocused) {
			fvv, _ := recv.Embed(KiT_FindView).(*FindView)
			tf := send.(*gi.TextField)
			fvv.Params().Find = tf.Text()
			fvv.FindAction()
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
			fvv, _ := recv.Embed(KiT_FindView).(*FindView)
			cb := send.(*gi.CheckBox)
			fvv.Params().IgnoreCase = cb.IsChecked()
		}
	})

	next := fb.AddNewChild(gi.KiT_Action, "next").(*gi.Action)
	next.SetIcon("widget-wedge-down")
	next.Tooltip = "go to next result"
	next.ActionSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.Embed(KiT_FindView).(*FindView)
		fvv.NextFind()
	})

	prev := fb.AddNewChild(gi.KiT_Action, "prev").(*gi.Action)
	prev.SetIcon("widget-wedge-up")
	prev.Tooltip = "go to previous result"
	prev.ActionSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.Embed(KiT_FindView).(*FindView)
		fvv.PrevFind()
	})

	repla := rb.AddNewChild(gi.KiT_Action, "repl-act").(*gi.Action)
	repla.SetText("Replace:")
	repla.Tooltip = "Replace find string with replace string for currently-selected find result"
	repla.ActionSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.Embed(KiT_FindView).(*FindView)
		fvv.ReplaceAction()
	})

	repls := rb.AddNewChild(gi.KiT_ComboBox, "repl-str").(*gi.ComboBox)
	repls.Editable = true
	repls.SetStretchMaxWidth()
	repls.Tooltip = "String to replace find string -- click for history"
	repls.ConfigParts()
	repls.ItemsFromStringList(fv.Params().ReplHist, true, 0)
	rtf, _ := repls.TextField()
	repls.ComboSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.Embed(KiT_FindView).(*FindView)
		cb := send.(*gi.ComboBox)
		fvv.Params().Replace = cb.CurVal.(string)
	})
	rtf.TextFieldSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) {
			fvv, _ := recv.Embed(KiT_FindView).(*FindView)
			tf := send.(*gi.TextField)
			fvv.Params().Replace = tf.Text()
		}
	})

	repall := rb.AddNewChild(gi.KiT_Action, "repl-all").(*gi.Action)
	repall.SetText("All")
	repall.Tooltip = "replace all find strings with replace string"
	repall.ActionSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.Embed(KiT_FindView).(*FindView)
		fvv.ReplaceAllAction()
	})

	locl := rb.AddNewChild(gi.KiT_Label, "loc-lbl").(*gi.Label)
	locl.SetText("Loc:")
	locl.Tooltip = "location to find in: all = all open folders in browser; file = current active file; dir = directory of current active file; nottop = all except the top-level in browser"
	// locl.SetProp("vertical-align", gi.AlignMiddle)

	cf := rb.AddNewChild(gi.KiT_ComboBox, "loc").(*gi.ComboBox)
	cf.SetText("Loc")
	cf.Tooltip = locl.Tooltip
	cf.ItemsFromEnum(KiT_FindLoc, false, 0)
	cf.ComboSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.Embed(KiT_FindView).(*FindView)
		cb := send.(*gi.ComboBox)
		eval := cb.CurVal.(kit.EnumValue)
		fvv.Params().Loc = FindLoc(eval.Value)
	})

	langl := rb.AddNewChild(gi.KiT_Label, "lang-lbl").(*gi.Label)
	langl.SetText("Lang:")
	langl.Tooltip = "Language(s) to restrict search / replace to"
	// langl.SetProp("vertical-align", gi.AlignMiddle)

	fv.LangVV = giv.ToValueView(&fv.Params().Langs, "")
	fv.LangVV.SetStandaloneValue(reflect.ValueOf(&fv.Params().Langs))
	vtyp := fv.LangVV.WidgetType()
	langw := rb.AddNewChild(vtyp, "langs").(gi.Node2D)
	fv.LangVV.ConfigWidget(langw)
	langw.AsWidget().Tooltip = langl.Tooltip
	//	vvb := vv.AsValueViewBase()
	//	vvb.ViewSig.ConnectOnly(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
	//		fvv, _ := recv.Embed(KiT_FindView).(*FindView)
	// hmm, langs updated..
	//	})

}

// FindViewProps are style properties for FindView
var FindViewProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
}
