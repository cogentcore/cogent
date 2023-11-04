// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"bytes"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"time"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/gi/v2/texteditor/textbuf"
	"goki.dev/girl/states"
	"goki.dev/goosi/events"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/pi/v2/filecat"
	"goki.dev/pi/v2/lex"
)

// FindLoc corresponds to the search scope
type FindLoc int32 //enums:enum -trim-prefix FindLoc

const (
	// FindLocAll finds in all open folders in the left file browser
	FindLocAll FindLoc = iota

	// FindLocFile only finds in the current active file
	FindLocFile

	// FindLocDir only finds in the directory of the current active file
	FindLocDir

	// FindLocNotTop finds in all open folders *except* the top-level folder
	FindLocNotTop
)

// FindParams are parameters for find / replace
type FindParams struct {

	// find string
	Find string

	// replace string
	Replace string

	// ignore case
	IgnoreCase bool

	// use regexp regular expression search and replace
	Regexp bool

	// languages for files to search
	Langs []filecat.Supported

	// locations to search in
	Loc FindLoc

	// history of finds
	FindHist []string

	// history of replaces
	ReplHist []string
}

// FindView is a find / replace widget that displays results in a TextView
// and has a toolbar for controlling find / replace process.
type FindView struct {
	gi.Layout

	// parent gide project
	Gide Gide `json:"-" xml:"-"`

	// langs value view
	LangVV giv.Value

	// time of last find
	Time time.Time

	// compiled regexp
	Re *regexp.Regexp
}

// Params returns the find params
func (fv *FindView) Params() *FindParams {
	return &fv.Gide.ProjPrefs().Find
}

// ShowResults shows the results in the buffer
func (fv *FindView) ShowResults(res []FileSearchResults) {
	ftv := fv.TextView()
	fbuf := ftv.Buf
	outlns := make([][]byte, 0, 100)
	outmus := make([][]byte, 0, 100) // markups
	for _, fs := range res {
		fp := fs.Node.Info.Path
		fn := fs.Node.MyRelPath()
		fbStLn := len(outlns) // find buf start ln
		lstr := fmt.Sprintf(`%v: %v`, fn, fs.Count)
		outlns = append(outlns, []byte(lstr))
		mstr := fmt.Sprintf(`<b>%v</b>`, lstr)
		outmus = append(outmus, []byte(mstr))
		for _, mt := range fs.Matches {
			txt := bytes.TrimSpace(mt.Text)
			txt = append([]byte{'\t'}, txt...)
			ln := mt.Reg.Start.Ln + 1
			ch := mt.Reg.Start.Ch + 1
			ech := mt.Reg.End.Ch + 1
			fnstr := fmt.Sprintf("%v:%d:%d", fn, ln, ch)
			nomu := bytes.Replace(txt, []byte("<mark>"), nil, -1)
			nomu = bytes.Replace(nomu, []byte("</mark>"), nil, -1)
			nomus := html.EscapeString(string(nomu))
			lstr = fmt.Sprintf(`%v: %s`, fnstr, nomus) // note: has tab embedded at start of lstr

			outlns = append(outlns, []byte(lstr))
			mstr = fmt.Sprintf(`	<a href="find:///%v#R%vN%vL%vC%v-L%vC%v">%v</a>: %s`, fp, fbStLn, fs.Count, ln, ch, ln, ech, fnstr, txt)
			outmus = append(outmus, []byte(mstr))
		}
		outlns = append(outlns, []byte(""))
		outmus = append(outmus, []byte(""))
	}
	ltxt := bytes.Join(outlns, []byte("\n"))
	mtxt := bytes.Join(outmus, []byte("\n"))
	fbuf.SetReadOnly(true)
	fbuf.AppendTextMarkup(ltxt, mtxt, texteditor.EditSignal)
	ftv.CursorStartDoc()
	ok := ftv.CursorNextLink(false) // no wrap
	if ok {
		ftv.OpenLinkAt(ftv.CursorPos)
	}
}

// SaveFindString saves the given find string to the find params history and current str
func (fv *FindView) SaveFindString(find string) {
	fv.Params().Find = find
	gi.StringsInsertFirstUnique(&fv.Params().FindHist, find, gi.Prefs.Params.SavedPathsMax)
	ftc := fv.FindText()
	if ftc != nil {
		ftc.SetStrings(fv.Params().FindHist, true, 0)
	}
}

// SaveReplString saves the given replace string to the find params history and current str
func (fv *FindView) SaveReplString(repl string) {
	fv.Params().Replace = repl
	gi.StringsInsertFirstUnique(&fv.Params().ReplHist, repl, gi.Prefs.Params.SavedPathsMax)
	rtc := fv.ReplText()
	if rtc != nil {
		rtc.SetStrings(fv.Params().ReplHist, true, 0)
	}
}

// FindAction runs a new find with current params
func (fv *FindView) FindAction() {
	fp := fv.Params()
	fv.SaveFindString(fp.Find)
	if !fv.CompileRegexp() {
		return
	}
	fv.Gide.Find(fp.Find, fp.Replace, fp.IgnoreCase, fp.Regexp, fp.Loc, fp.Langs)
}

// CheckValidRegexp returns false if using regexp and it is not valid
func (fv *FindView) CheckValidRegexp() bool {
	fp := fv.Params()
	if !fp.Regexp {
		return true
	}
	if fv.Re == nil {
		return false
	}
	return true
}

// ReplaceAction performs the replace -- if using regexp mode, regexp must be compiled in advance
func (fv *FindView) ReplaceAction() bool {
	if !fv.CheckValidRegexp() {
		return false
	}
	wupdt := fv.UpdateStart()
	defer fv.UpdateEnd(wupdt)

	fp := fv.Params()
	fv.SaveReplString(fp.Replace)
	gi.StringsInsertFirstUnique(&fp.ReplHist, fp.Replace, gi.Prefs.Params.SavedPathsMax)

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
		if fp.Regexp {
			rg := tv.Buf.Region(reg.Start, reg.End)
			b := rg.ToBytes()
			rb := fv.Re.ReplaceAll(b, []byte(fp.Replace))
			tv.Buf.ReplaceText(reg.Start, reg.End, reg.Start, string(rb), texteditor.EditSignal, false)
		} else {
			// MatchCase only if doing IgnoreCase
			tv.Buf.ReplaceText(reg.Start, reg.End, reg.Start, fp.Replace, texteditor.EditSignal, fp.IgnoreCase)
		}

		// delete the link for the just done replace
		ftvln := ftv.CursorPos.Ln
		st := lex.Pos{Ln: ftvln, Ch: 0}
		len := len(ftv.Buf.Lines[ftvln])
		en := lex.Pos{Ln: ftvln, Ch: len}
		ftv.Buf.DeleteText(st, en, texteditor.EditSignal)
	}

	tv.ClearHighlights()

	ok = ftv.CursorNextLink(false) // no wrap
	if ok {
		ftv.OpenLinkAt(ftv.CursorPos) // move to next
	}
	return ok
}

// ReplaceAllAction performs replace all, prompting before proceeding
func (fv *FindView) ReplaceAllAction() {
	d := gi.NewDialog(fv).Title("Confirm Replace All").
		Prompt("Are you sure you want to Replace All?").
		Modal(true).Cancel().Ok("Replace All")
	d.Run()
	d.OnAccept(func(e events.Event) {
		fv.ReplaceAll()
	})
}

// CompileRegexp compiles the regexp if necessary -- returns false if it is invalid
func (fv *FindView) CompileRegexp() bool {
	fp := fv.Params()
	if !fp.Regexp {
		fv.Re = nil
		return true
	}
	var err error
	fv.Re, err = regexp.Compile(fp.Find)
	if err != nil {
		gi.NewDialog(fv).Title("Regexp is Invalid").
			Prompt(fmt.Sprintf("The regular expression was invalid: %v", err)).Modal(true).Ok().Run()
		return false
	}
	return true
}

// ReplaceAll performs replace all
func (fv *FindView) ReplaceAll() {
	if !fv.CheckValidRegexp() {
		return
	}
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
func (fv *FindView) OpenFindURL(ur string, ftv *texteditor.Editor) bool {
	ge := fv.Gide
	tv, reg, fbBufStLn, fCount, ok := ge.ParseOpenFindURL(ur, ftv)
	if !ok {
		return false
	}
	reg.Time.SetTime(fv.Time)
	reg = tv.Buf.AdjustReg(reg)
	find := fv.Params().Find
	texteditor.PrevISearchString = find
	tve := texteditor.AsEditor(tv)
	fv.HighlightFinds(tve, ftv, fbBufStLn, fCount, find)
	tv.SetNeedsRender()
	tv.SetCursorShow(reg.Start)
	return true
}

// HighlightFinds highlights all the find results in ftv buffer
func (fv *FindView) HighlightFinds(tv, ftv *texteditor.Editor, fbStLn, fCount int, find string) {
	lnka := []byte(`<a href="`)
	lnkasz := len(lnka)

	fb := ftv.Buf

	if len(tv.Highlights) != fCount { // highlight
		hi := make([]textbuf.Region, fCount)
		for i := 0; i < fCount; i++ {
			fln := fbStLn + 1 + i
			if fln >= len(fb.Markup) {
				continue
			}
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
			ireg := textbuf.Region{}
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

func (fv *FindView) ConfigWidget(sc *gi.Scene) {
	// fv.ConfigFindView()
}

// Config configures the view
func (fv *FindView) ConfigFindView(ge Gide) {
	fv.Gide = ge
	fv.Lay = gi.LayoutVert
	config := ki.Config{}
	config.Add(gi.ToolbarType, "findbar")
	config.Add(gi.ToolbarType, "replbar")
	config.Add(gi.LayoutType, "findtext")
	mods, updt := fv.ConfigChildren(config)
	if !mods {
		updt = fv.UpdateStart()
	}
	fp := fv.Params()
	fv.ConfigToolbar()
	ft := fv.FindText()
	ft.SetStrings(fp.FindHist, true, 0)
	ft.SetCurVal(fp.Find)
	rt := fv.ReplText()
	rt.SetStrings(fp.ReplHist, true, 0)
	rt.SetCurVal(fp.Replace)
	ib := fv.IgnoreBox()
	ib.SetChecked(fp.IgnoreCase)
	rb := fv.RegexpBox()
	rb.SetChecked(fp.Regexp)
	cf := fv.LocCombo()
	cf.SetCurIndex(int(fp.Loc))
	tv := fv.TextView()
	ConfigOutputTextView(tv)
	if mods {
		na := fv.FindNextAct()
		na.GrabFocus()
	}
	fv.UpdateEnd(updt)
}

// FindBar returns the find toolbar
func (fv *FindView) FindBar() *gi.Toolbar {
	return fv.ChildByName("findbar", 0).(*gi.Toolbar)
}

// ReplBar returns the replace toolbar
func (fv *FindView) ReplBar() *gi.Toolbar {
	return fv.ChildByName("replbar", 1).(*gi.Toolbar)
}

// FindText returns the find textfield in toolbar
func (fv *FindView) FindText() *gi.Chooser {
	return fv.FindBar().ChildByName("find-str", 1).(*gi.Chooser)
}

// ReplText returns the replace textfield in toolbar
func (fv *FindView) ReplText() *gi.Chooser {
	return fv.ReplBar().ChildByName("repl-str", 1).(*gi.Chooser)
}

// IgnoreBox returns the ignore case checkbox in toolbar
func (fv *FindView) IgnoreBox() *gi.Switch {
	return fv.FindBar().ChildByName("ignore-case", 2).(*gi.Switch)
}

// RegexpBox returns the regexp checkbox in toolbar
func (fv *FindView) RegexpBox() *gi.Switch {
	return fv.FindBar().ChildByName("regexp", 3).(*gi.Switch)
}

// LocCombo returns the loc combobox
func (fv *FindView) LocCombo() *gi.Chooser {
	return fv.FindBar().ChildByName("loc", 5).(*gi.Chooser)
}

// FindNextAct returns the find next action in toolbar -- selected first
func (fv *FindView) FindNextAct() *gi.Button {
	return fv.ReplBar().ChildByName("next", 3).(*gi.Button)
}

// TextViewLay returns the find results TextView
func (fv *FindView) TextView() *texteditor.Editor {
	return texteditor.AsEditor(fv.ChildByName("findtext", 1))
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

	gi.NewButton(fb).SetText("Find:").SetTooltip("Find given string in project files. Only open folders in file browser will be searched -- adjust those to scope the search").OnClick(func(e events.Event) {
		fv.FindAction()
	})
	finds := gi.NewChooser(fb).SetEditable(true).
		SetTooltip("String to find -- hit enter or tab to update search -- click for history")
	finds.SetStretchMaxWidth()
	finds.SetStrings(fv.Params().FindHist, true, 0)
	finds.OnChange(func(e events.Event) {
		fv.Params().Find = finds.CurVal.(string)
		if fv.Params().Find == "" {
			tv := fv.Gide.ActiveTextView()
			if tv != nil {
				tv.ClearHighlights()
			}
			fvtv := fv.TextView()
			if fvtv != nil {
				fvtv.Buf.NewBuf(0)
			}
		} else {
			fv.FindAction()
		}
	})

	ic := gi.NewSwitch(fb).SetText("Ignore Case")
	ic.OnClick(func(e events.Event) {
		fv.Params().IgnoreCase = ic.StateIs(states.Checked)
	})
	rx := gi.NewSwitch(fb).SetText("Regexp").
		SetTooltip("use regular expression for search and replace -- see https://github.com/google/re2/wiki/Syntax")
	rx.OnClick(func(e events.Event) {
		fv.Params().Regexp = rx.StateIs(states.Checked)
	})

	locl := gi.NewLabel(fb).SetText("Loc:").
		SetTooltip("location to find in: all = all open folders in browser; file = current active file; dir = directory of current active file; nottop = all except the top-level in browser")

	cf := gi.NewChooser(fb).SetTooltip(locl.Tooltip)
	cf.SetEnum(fv.Params().Loc, false, 0)
	cf.OnClick(func(e events.Event) {
		eval := cf.CurVal.(FindLoc)
		fv.Params().Loc = eval
	})

	//////////////// ReplBar

	gi.NewButton(rb).SetIcon(icons.KeyboardArrowUp).SetTooltip("go to previous result").
		OnClick(func(e events.Event) {
			fv.PrevFind()
		})
	gi.NewButton(rb).SetIcon(icons.KeyboardArrowDown).SetTooltip("go to next result").
		OnClick(func(e events.Event) {
			fv.NextFind()
		})
	gi.NewButton(rb).SetText("Replace:").SetTooltip("Replace find string with replace string for currently-selected find result").
		OnClick(func(e events.Event) {
			fv.ReplaceAction()
		})

	repls := gi.NewChooser(rb).SetEditable(true).SetTooltip("String to replace find string -- click for history -- use ${n} for regexp submatch where n = 1 for first submatch, etc")
	repls.SetStretchMaxWidth()
	repls.SetStrings(fv.Params().ReplHist, true, 0)
	repls.OnChange(func(e events.Event) {
		fv.Params().Replace = repls.CurVal.(string)
	})

	gi.NewButton(rb).SetText("All").SetTooltip("replace all find strings with replace string").
		OnClick(func(e events.Event) {
			fv.ReplaceAllAction()
		})

	langl := gi.NewLabel(rb).SetText("Lang:").SetTooltip("Language(s) to restrict search / replace to")

	fv.LangVV = giv.NewValue(rb, &fv.Params().Langs)
	fv.LangVV.AsWidgetBase().SetTooltip(langl.Tooltip)
}

// // FindViewProps are style properties for FindView
// var FindViewProps = ki.Props{
// 	"EnumType:Flag":    gi.KiT_NodeFlags,
// 	"background-color": &gi.Prefs.Colors.Background,
// 	"color":            &gi.Prefs.Colors.Font,
// 	"max-width":        -1,
// 	"max-height":       -1,
// }
