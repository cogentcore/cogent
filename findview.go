// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/goki/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// FindParams are parameters for find / replace
type FindParams struct {
	Find       string    `desc:"find string"`
	Replace    string    `desc:"replace string"`
	IgnoreCase bool      `desc:"ignore case"`
	Langs      LangNames `desc:"languages for files to search"`
}

// FindView is a find / replace widget that displays results in a TextView
// and has a toolbar for controlling find / replace process.
type FindView struct {
	gi.Layout
	Gide *Gide      `json:"-" xml:"-" desc:"parent gide project"`
	Find FindParams `desc:"params for find / replace"`
}

var KiT_FindView = kit.Types.AddType(&FindView{}, FindViewProps)

// FindAction runs a new find with current params
func (fv *FindView) FindAction() {
	fv.Gide.Find(fv.Find.Find, fv.Find.IgnoreCase, fv.Find.Langs)
}

// NextFind shows next find result
func (fv *FindView) NextFind() {
	tv := fv.TextView()
	tv.CursorNextLink()
	tv.OpenLinkAt(tv.CursorPos)
}

// PrevFind shows previous find result
func (fv *FindView) PrevFind() {
	tv := fv.TextView()
	tv.CursorPrevLink()
	tv.OpenLinkAt(tv.CursorPos)
}

// OpenFindURL opens given find:/// url from Find
func (fv *FindView) OpenFindURL(ur string, ftv *giv.TextView) bool {
	ge := fv.Gide
	up, err := url.Parse(ur)
	if err != nil {
		log.Printf("FindView OpenFindURL parse err: %v\n", err)
		return false
	}
	fpath := up.Path[1:] // has double //
	pos := up.Fragment
	tv, _, ok := ge.NextViewFile(gi.FileName(fpath))
	if !ok {
		gi.PromptDialog(fv.Viewport, gi.DlgOpts{Title: "Couldn't Open File at Link", Prompt: fmt.Sprintf("Could not find or open file path in project: %v", fpath)}, true, false, nil, nil)
		return false
	}
	if pos == "" {
		return true
	}

	reg := giv.TextRegion{}
	var fbStLn, fCount int
	lidx := strings.Index(pos, "L")
	if lidx > 0 {
		reg.FromString(pos[lidx:])
		pos = pos[:lidx]
	}
	fmt.Sscanf(pos, "R%dN%d", &fbStLn, &fCount)

	lnka := []byte(`<a href="`)
	lnkasz := len(lnka)

	fb := ftv.Buf
	find := fv.Find.Find
	ignoreCase := fv.Find.IgnoreCase
	tv.PrevISearchString = find
	tv.PrevISearchCase = !ignoreCase

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
			hi[i] = ireg
		}
		tv.Highlights = hi
	}
	tv.SetCursorShow(reg.Start)
	return true
}

//////////////////////////////////////////////////////////////////////////////////////
//    GUI config

// UpdateView updates view with current settings
func (fv *FindView) UpdateView(ge *Gide, fp FindParams) {
	fv.Gide = ge
	fv.Find = fp
	mods, updt := fv.StdFindConfig()
	fv.ConfigToolbar()
	ft := fv.FindText()
	ft.SetText(fv.Find.Find)
	ib := fv.IgnoreBox()
	ib.SetChecked(fv.Find.IgnoreCase)
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

// ToolBar returns the find toolbar
func (fv *FindView) ToolBar() *gi.ToolBar {
	tbi, ok := fv.ChildByName("findbar", 0)
	if !ok {
		return nil
	}
	return tbi.(*gi.ToolBar)
}

// FindText returns the find textfield in toolbar
func (fv *FindView) FindText() *gi.TextField {
	tb := fv.ToolBar()
	if tb == nil {
		return nil
	}
	tfi, ok := tb.ChildByName("find-str", 1)
	if !ok {
		return nil
	}
	return tfi.(*gi.TextField)
}

// IgnoreBox returns the ignore case checkbox in toolbar
func (fv *FindView) IgnoreBox() *gi.CheckBox {
	tb := fv.ToolBar()
	if tb == nil {
		return nil
	}
	tfi, ok := tb.ChildByName("ignore-case", 2)
	if !ok {
		return nil
	}
	return tfi.(*gi.CheckBox)
}

// FindNextAct returns the find next action in toolbar -- selected first
func (fv *FindView) FindNextAct() *gi.Action {
	tb := fv.ToolBar()
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
	tv := tvly.KnownChild(0).(*giv.TextView)
	return tv
}

// ConfigToolbar adds toolbar.
func (fv *FindView) ConfigToolbar() {
	tb := fv.ToolBar()
	if tb.HasChildren() {
		return
	}
	tb.SetStretchMaxWidth()

	finda := tb.AddNewChild(gi.KiT_Action, "find-act").(*gi.Action)
	finda.SetText("Find:")
	finda.ActionSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.Embed(KiT_FindView).(*FindView)
		fvv.FindAction()
	})

	finds := tb.AddNewChild(gi.KiT_TextField, "find-str").(*gi.TextField)
	finds.SetMinPrefWidth(units.NewValue(80, units.Ch))
	finds.TextFieldSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) {
			fvv, _ := recv.Embed(KiT_FindView).(*FindView)
			tf := send.(*gi.TextField)
			fvv.Find.Find = tf.Text()
			fvv.FindAction()
		}
	})

	ic := tb.AddNewChild(gi.KiT_CheckBox, "ignore-case").(*gi.CheckBox)
	ic.SetText("Ignore Case")
	ic.ButtonSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonToggled) {
			fvv, _ := recv.Embed(KiT_FindView).(*FindView)
			cb := send.(*gi.CheckBox)
			fvv.Find.IgnoreCase = cb.IsChecked()
		}
	})

	next := tb.AddNewChild(gi.KiT_Action, "next").(*gi.Action)
	next.SetIcon("widget-wedge-down")
	next.Tooltip = "go to next result"
	next.ActionSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.Embed(KiT_FindView).(*FindView)
		fvv.NextFind()
	})

	prev := tb.AddNewChild(gi.KiT_Action, "prev").(*gi.Action)
	prev.SetIcon("widget-wedge-up")
	prev.Tooltip = "go to previous result"
	prev.ActionSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.Embed(KiT_FindView).(*FindView)
		fvv.PrevFind()
	})

}

var FindViewProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
}
