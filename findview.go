// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/goki/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// FindParams are parameters for find / replace
type FindParams struct {
	Find       string    `desc:"find string"`
	Replace    string    `desc:"replace string"`
	IgnoreCase bool      `desc:"ignore case"`
	Langs      LangNames `desc:"languages for files to search"`
	CurFile    bool      `desc:"only process current active file"`
}

// FindView is a find / replace widget that displays results in a TextView
// and has a toolbar for controlling find / replace process.
type FindView struct {
	gi.Layout
	Gide   *Gide         `json:"-" xml:"-" desc:"parent gide project"`
	Find   FindParams    `desc:"params for find / replace"`
	LangVV giv.ValueView `desc:"langs value view"`
}

var KiT_FindView = kit.Types.AddType(&FindView{}, FindViewProps)

// FindAction runs a new find with current params
func (fv *FindView) FindAction() {
	fv.Gide.Prefs.Find = fv.Find
	fv.Gide.Find(fv.Find.Find, fv.Find.Replace, fv.Find.IgnoreCase, fv.Find.CurFile, fv.Find.Langs)
}

// ReplaceAction performs the replace
func (fv *FindView) ReplaceAction() bool {
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
	er := giv.TextRegion{}
	ge := fv.Gide
	tv, reg, _, _, ok := ge.ParseOpenFindURL(tl.URL, ftv)
	if !ok {
		return false
	}
	if reg == er { // nil
		ok = ftv.CursorNextLink(false) // no wrap
		if !ok {
			return false
		}
		tl, ok = ftv.OpenLinkAt(ftv.CursorPos)
		tv, reg, _, _, ok = ge.ParseOpenFindURL(tl.URL, ftv)
		if !ok || reg == er {
			return false
		}
	}
	tv.RefreshIfNeeded()
	tbe := tv.Buf.DeleteText(reg.Start, reg.End, true, true)
	tv.Buf.InsertText(tbe.Reg.Start, []byte(fv.Find.Replace), true, true)
	offset := len(fv.Find.Replace) - (reg.End.Ch - reg.Start.Ch) // current offset + new length - old length

	regexstart, err := regexp.Compile(`L[0-9]+C[0-9]+-`)
	if err != nil {
		fmt.Printf("error %s", err)
	}

	// on what line was the replace
	var curline = -1
	url, err := url.Parse(tl.URL)
	if err == nil {
		s := regexstart.FindString(url.Fragment)
		s = strings.TrimSuffix(s, "-")
		line, _ := fv.UrlPos(s)
		curline = line - 1 // -1 because url is 1 based and text is 0 based
	}

	ftv.UpdateStart()
	// update the find link for the just done replace
	var curlinkidx int
	for i, r := range ftv.Renders {
		if r.Links != nil {
			if r.Links[0].URL == tl.URL {
				ur, urup := fv.UpdateUrl(r.Links[0].URL, 0, offset)
				if urup {
					r.Links[0].URL = ur
					curlinkidx = i
					mu := string(ftv.Buf.Markup[curlinkidx])
					umu, umuup := fv.UpdateMarkup(mu, 0, offset, true, fv.Find.Replace)
					if umuup {
						bmu := []byte(umu)
						ftv.Buf.Markup[curlinkidx] = bmu
					}
					break
				}
			}
		}
	}

	// update the find links for any others for the same line of text
	for linkidx := curlinkidx + 1; linkidx < len(ftv.Renders); linkidx++ {
		r := ftv.Renders[linkidx]
		if r.Links != nil {
			if r.Links[0].Label != tl.Label { // another file
				break
			}
			urlstr := r.Links[0].URL
			url, err := url.Parse(urlstr)
			if err == nil {
				s := regexstart.FindString(url.Fragment)
				s = strings.TrimSuffix(s, "-")
				line, _ := fv.UrlPos(s)
				if (line - 1) == curline {
					ur, urup := fv.UpdateUrl(urlstr, offset, offset)
					if urup {
						r.Links[0].URL = ur
						mu := string(ftv.Buf.Markup[linkidx])
						umu, umuup := fv.UpdateMarkup(mu, offset, offset, false, fv.Find.Replace)
						if umuup {
							bmu := []byte(umu)
							ftv.Buf.Markup[linkidx] = bmu
						}
					}
				}
			}
		}
	}

	// delete the link for the just done replace
	ftvln := ftv.CursorPos.Ln
	st := giv.TextPos{Ln: ftvln, Ch: 0}
	len := len(ftv.Buf.Lines[ftvln])
	en := giv.TextPos{Ln: ftvln, Ch: len}
	ftv.Buf.DeleteText(st, en, false, true)
	ftv.UpdateEnd(true)
	ftv.NeedsRefresh()

	tv.Highlights = tv.Highlights[:0]
	tv.NeedsRefresh()

	ok = ftv.CursorNextLink(false) // no wrap
	if ok {
		ftv.OpenLinkAt(ftv.CursorPos) // move to next
	}
	return ok
}

// UrlPos returns the line and char from a url fragment "LxxCxx"
func (fv *FindView) UrlPos(lnch string) (int, int) {
	scidx := strings.Index(lnch, "C")
	if scidx == -1 { // url does not have TextPos
		return -1, -1
	}
	ln := lnch[0:scidx]
	ln = strings.TrimPrefix(ln, "L")
	ch := lnch[scidx:]
	ch = strings.TrimPrefix(ch, "C")

	lni, lnerr := strconv.Atoi(ln)
	chi, cherr := strconv.Atoi(ch)
	if lnerr == nil && cherr == nil {
		return lni, chi
	}
	return -1, -1
}

// UpdateUrlTextPos updates the link character position (e.g. when replace string is different length than original)
func (fv *FindView) UpdateUrl(url string, stoff, enoff int) (string, bool) {
	//fmt.Println("old:", url)
	regextr, err := regexp.Compile(`L[0-9]+C[0-9]+-L[0-9]+C[0-9]+`)
	if err != nil {
		fmt.Printf("error %s", err)
	}
	trstr := regextr.FindString(url)
	tr := giv.TextRegion{}
	tr.FromString(trstr)
	update := fmt.Sprintf("L%dC%d-L%dC%d", tr.Start.Ln+1, tr.Start.Ch+1+stoff, tr.End.Ln+1, tr.End.Ch+1+enoff)
	url = strings.Replace(url, trstr, update, 1)
	//fmt.Println("new:", url)
	return url, true
}

func (fv *FindView) UpdateMarkup(url string, stoff, enoff int, dorep bool, rep string) (string, bool) {
	//fmt.Println("old:", url)
	regextr, err := regexp.Compile(`L[0-9]+C[0-9]+-L[0-9]+C[0-9]+`)
	if err != nil {
		fmt.Printf("error %s", err)
	}
	trstr := regextr.FindString(url)
	tr := giv.TextRegion{}
	tr.FromString(trstr)
	// text links are 1 based - compensate for the -1 done in FromString()
	update := fmt.Sprintf("L%dC%d-L%dC%d", tr.Start.Ln+1, tr.Start.Ch+1+stoff, tr.End.Ln+1, tr.End.Ch+1+enoff)
	url = strings.Replace(url, trstr, update, 1)

	if dorep {
		regexspan, err := regexp.Compile(`<mark>.*</mark>`)
		if err != nil {
			fmt.Printf("error %s", err)
		}
		url = regexspan.ReplaceAllString(url, rep)
	}
	//fmt.Println("new:", url)
	return url, true
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
	find := fv.Find.Find
	giv.PrevISearchString = find
	ge.HighlightFinds(tv, ftv, fbBufStLn, fCount, find)
	tv.SetNeedsRefresh()
	tv.RefreshIfNeeded()
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
	rt := fv.ReplText()
	rt.SetText(fv.Find.Replace)
	ib := fv.IgnoreBox()
	ib.SetChecked(fv.Find.IgnoreCase)
	cf := fv.CurFileBox()
	cf.SetChecked(fv.Find.CurFile)
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
func (fv *FindView) FindText() *gi.TextField {
	tb := fv.FindBar()
	if tb == nil {
		return nil
	}
	tfi, ok := tb.ChildByName("find-str", 1)
	if !ok {
		return nil
	}
	return tfi.(*gi.TextField)
}

// ReplText returns the replace textfield in toolbar
func (fv *FindView) ReplText() *gi.TextField {
	tb := fv.ReplBar()
	if tb == nil {
		return nil
	}
	tfi, ok := tb.ChildByName("repl-str", 1)
	if !ok {
		return nil
	}
	return tfi.(*gi.TextField)
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

// CurFileBox returns the cur file checkbox in toolbar
func (fv *FindView) CurFileBox() *gi.CheckBox {
	tb := fv.ReplBar()
	if tb == nil {
		return nil
	}
	tfi, ok := tb.ChildByName("cur-file", 5)
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
	tv := tvly.KnownChild(0).(*giv.TextView)
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
	finda.ActionSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.Embed(KiT_FindView).(*FindView)
		fvv.FindAction()
	})

	finds := fb.AddNewChild(gi.KiT_TextField, "find-str").(*gi.TextField)
	finds.SetStretchMaxWidth()
	finds.Tooltip = "String to find -- hit enter or tab to update search"
	finds.TextFieldSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) {
			fvv, _ := recv.Embed(KiT_FindView).(*FindView)
			tf := send.(*gi.TextField)
			fvv.Find.Find = tf.Text()
			fvv.FindAction()
		}
	})

	ic := fb.AddNewChild(gi.KiT_CheckBox, "ignore-case").(*gi.CheckBox)
	ic.SetText("Ignore Case")
	ic.ButtonSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonToggled) {
			fvv, _ := recv.Embed(KiT_FindView).(*FindView)
			cb := send.(*gi.CheckBox)
			fvv.Find.IgnoreCase = cb.IsChecked()
		}
	})

	next := fb.AddNewChild(gi.KiT_Action, "next").(*gi.Action)
	next.SetIcon("widget-wedge-down")
	next.Tooltip = "go to next result"
	next.ActionSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.Embed(KiT_FindView).(*FindView)
		fvv.NextFind()
	})

	prev := fb.AddNewChild(gi.KiT_Action, "prev").(*gi.Action)
	prev.SetIcon("widget-wedge-up")
	prev.Tooltip = "go to previous result"
	prev.ActionSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.Embed(KiT_FindView).(*FindView)
		fvv.PrevFind()
	})

	repla := rb.AddNewChild(gi.KiT_Action, "repl-act").(*gi.Action)
	repla.SetText("Replace:")
	repla.Tooltip = "Replace find string with replace string for currently-selected find result"
	repla.ActionSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.Embed(KiT_FindView).(*FindView)
		fvv.ReplaceAction()
	})

	repls := rb.AddNewChild(gi.KiT_TextField, "repl-str").(*gi.TextField)
	repls.SetStretchMaxWidth()
	repls.Tooltip = "String to replace find string"
	repls.TextFieldSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) {
			fvv, _ := recv.Embed(KiT_FindView).(*FindView)
			tf := send.(*gi.TextField)
			fvv.Find.Replace = tf.Text()
		}
	})

	repall := rb.AddNewChild(gi.KiT_Action, "repl-all").(*gi.Action)
	repall.SetText("All")
	repall.Tooltip = "replace all find strings with replace string"
	repall.ActionSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.Embed(KiT_FindView).(*FindView)
		fvv.ReplaceAllAction()
	})

	langl := rb.AddNewChild(gi.KiT_Label, "lang-lbl").(*gi.Label)
	langl.SetText("Lang:")

	fv.LangVV = giv.ToValueView(&fv.Find.Langs, "")
	fv.LangVV.SetStandaloneValue(reflect.ValueOf(&fv.Find.Langs))
	vtyp := fv.LangVV.WidgetType()
	langw := rb.AddNewChild(vtyp, "langs").(gi.Node2D)
	fv.LangVV.ConfigWidget(langw)
	langw.AsWidget().Tooltip = "Language(s) to restrict search / replace to"
	//	vvb := vv.AsValueViewBase()
	//	vvb.ViewSig.ConnectOnly(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
	//		fvv, _ := recv.Embed(KiT_FindView).(*FindView)
	// hmm, langs updated..
	//	})

	cf := rb.AddNewChild(gi.KiT_CheckBox, "cur-file").(*gi.CheckBox)
	cf.SetText("Cur File")
	cf.Tooltip = "Only in current active file"
	cf.ButtonSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonToggled) {
			fvv, _ := recv.Embed(KiT_FindView).(*FindView)
			cb := send.(*gi.CheckBox)
			fvv.Find.CurFile = cb.IsChecked()
		}
	})

}

var FindViewProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
}
