// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"bytes"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/fi"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/pi/lex"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/texteditor/textbuf"
)

// FindLoc corresponds to the search scope
type FindLoc int32 //enums:enum -trim-prefix FindLoc

const (
	// FindLocAll finds in all directories under the root path
	FindLocAll FindLoc = iota

	// FindOpen finds in all open folders in the left file browser
	FindLocOpen

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

	// locations to search in
	Loc FindLoc

	// languages for files to search
	Langs []fi.Known

	// history of finds
	FindHist []string

	// history of replaces
	ReplHist []string
}

// FindView is a find / replace widget that displays results in a TextEditor
// and has a toolbar for controlling find / replace process.
type FindView struct {
	gi.Layout

	// parent code project
	Code Code `json:"-" xml:"-"`

	// langs value view
	LangVV giv.Value

	// time of last find
	Time time.Time

	// compiled regexp
	Re *regexp.Regexp
}

// Params returns the find params
func (fv *FindView) Params() *FindParams {
	return &fv.Code.ProjPrefs().Find
}

// ShowResults shows the results in the buffer
func (fv *FindView) ShowResults(res []FileSearchResults) {
	ftv := fv.TextEditor()
	fbuf := ftv.Buf
	fbuf.Opts.LineNos = false
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

	fv.Update()
	ftv.SetCursorShow(lex.Pos{Ln: 0})
	ok := ftv.CursorNextLink(false) // no wrap
	if ok {
		ftv.OpenLinkAt(ftv.CursorPos)
	}
}

// SaveFindString saves the given find string to the find params history and current str
func (fv *FindView) SaveFindString(find string) {
	fv.Params().Find = find
	gi.StringsInsertFirstUnique(&fv.Params().FindHist, find, gi.SystemSettings.SavedPathsMax)
	ftc := fv.FindText()
	if ftc != nil {
		ftc.SetStrings(fv.Params().FindHist, true)
	}
}

// SaveReplString saves the given replace string to the find params history and current str
func (fv *FindView) SaveReplString(repl string) {
	fv.Params().Replace = repl
	gi.StringsInsertFirstUnique(&fv.Params().ReplHist, repl, gi.SystemSettings.SavedPathsMax)
	rtc := fv.ReplText()
	if rtc != nil {
		rtc.SetStrings(fv.Params().ReplHist, true)
	}
}

// FindAction runs a new find with current params
func (fv *FindView) FindAction() {
	fp := fv.Params()
	fv.SaveFindString(fp.Find)
	if !fv.CompileRegexp() {
		return
	}
	fv.Code.Find(fp.Find, fp.Replace, fp.IgnoreCase, fp.Regexp, fp.Loc, fp.Langs)
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
	updt := fv.UpdateStart()
	defer fv.UpdateEndRender(updt)

	fp := fv.Params()
	fv.SaveReplString(fp.Replace)
	gi.StringsInsertFirstUnique(&fp.ReplHist, fp.Replace, gi.SystemSettings.SavedPathsMax)

	ftv := fv.TextEditor()
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
	ge := fv.Code
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
	d := gi.NewBody().AddTitle("Confirm Replace All").
		AddText("Are you sure you want to Replace All?")
	d.AddBottomBar(func(pw gi.Widget) {
		d.AddCancel(pw)
		d.AddOk(pw).SetText("Replace All").OnClick(func(e events.Event) {
			fv.ReplaceAll()
		})
	})
	d.NewDialog(fv).Run()
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
		gi.MessageSnackbar(fv, fmt.Sprintf("The regular expression was invalid: %v", err))
		return false
	}
	return true
}

// ReplaceAll performs replace all
func (fv *FindView) ReplaceAll() {
	if !fv.CheckValidRegexp() {
		return
	}
	go func() {
		for {
			updt := fv.Code.AsWidget().Scene.UpdateStartAsync()
			ok := fv.ReplaceAction()
			fv.Code.AsWidget().Scene.UpdateEndAsyncLayout(updt)
			if !ok {
				break
			}
		}
	}()
}

// NextFind shows next find result
func (fv *FindView) NextFind() {
	ftv := fv.TextEditor()
	ok := ftv.CursorNextLink(true) // wrap
	if ok {
		ftv.OpenLinkAt(ftv.CursorPos)
	}
}

// PrevFind shows previous find result
func (fv *FindView) PrevFind() {
	ftv := fv.TextEditor()
	ok := ftv.CursorPrevLink(true) // wrap
	if ok {
		ftv.OpenLinkAt(ftv.CursorPos)
	}
}

// OpenFindURL opens given find:/// url from Find
func (fv *FindView) OpenFindURL(ur string, ftv *texteditor.Editor) bool {
	ge := fv.Code
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
	tv.SetCursorTarget(reg.Start)
	tv.SetNeedsLayout(true)
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

func (fv *FindView) ConfigWidget() {
	fv.ConfigFindView()
}

// Config configures the view
func (fv *FindView) ConfigFindView() {
	if fv.HasChildren() {
		return
	}
	fv.Code, _ = ParentCode(fv)

	fv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	updt := fv.UpdateStart()
	defer fv.UpdateEndLayout(updt)

	fb := gi.NewBasicBar(fv, "findbar")
	rb := gi.NewBasicBar(fv, "replbar")
	tv := texteditor.NewEditor(fv, "findtext")
	ConfigOutputTextEditor(tv)
	tv.LinkHandler = func(tl *paint.TextLink) {
		fv.OpenFindURL(tl.URL, tv)
	}
	fv.ConfigToolbars(fb, rb)
	na := fv.FindNextAct()
	na.SetFocusEvent()
	fv.Update()
}

// FindBar returns the find toolbar
func (fv *FindView) FindBar() *gi.BasicBar {
	return fv.ChildByName("findbar", 0).(*gi.BasicBar)
}

// ReplBar returns the replace toolbar
func (fv *FindView) ReplBar() *gi.BasicBar {
	return fv.ChildByName("replbar", 1).(*gi.BasicBar)
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

// TextEditorLay returns the find results TextEditor
func (fv *FindView) TextEditor() *texteditor.Editor {
	return texteditor.AsEditor(fv.ChildByName("findtext", 1))
}

// UpdateFromParams is called in Find function with new params
func (fv *FindView) UpdateFromParams() {
	fp := fv.Params()
	ft := fv.FindText()
	ft.SetCurVal(fp.Find)
	rt := fv.ReplText()
	rt.SetCurVal(fp.Replace)
	ib := fv.IgnoreBox()
	ib.SetChecked(fp.IgnoreCase)
	rb := fv.RegexpBox()
	rb.SetChecked(fp.Regexp)
	cf := fv.LocCombo()
	cf.SetCurVal(fp.Loc)
	// langs auto-updates from param
}

// ConfigToolbars
func (fv *FindView) ConfigToolbars(fb, rb *gi.BasicBar) {
	gi.NewButton(fb).SetText("Find:").SetTooltip("Find given string in project files. Only open folders in file browser will be searched -- adjust those to scope the search").OnClick(func(e events.Event) {
		fv.FindAction()
	})
	finds := gi.NewChooser(fb, "find-str").SetEditable(true).SetAllowNew(true).
		SetTooltip("String to find -- hit enter or tab to update search -- click for history")
	finds.Style(func(s *styles.Style) {
		s.Grow.Set(1, 0)
	})
	finds.OnChange(func(e events.Event) {
		fv.Params().Find = finds.CurVal.(string)
		if fv.Params().Find == "" {
			tv := fv.Code.ActiveTextEditor()
			if tv != nil {
				tv.ClearHighlights()
			}
			fvtv := fv.TextEditor()
			if fvtv != nil {
				fvtv.Buf.NewBuf(0)
			}
		} else {
			fv.FindAction()
		}
	})

	ic := gi.NewSwitch(fb, "ignore-case").SetText("Ignore Case")
	ic.OnChange(func(e events.Event) {
		fv.Params().IgnoreCase = ic.StateIs(states.Checked)
	})
	rx := gi.NewSwitch(fb, "regexp").SetText("Regexp").
		SetTooltip("use regular expression for search and replace -- see https://github.com/google/re2/wiki/Syntax")
	rx.OnChange(func(e events.Event) {
		fv.Params().Regexp = rx.StateIs(states.Checked)
	})

	locl := gi.NewLabel(fb).SetText("Loc:").
		SetTooltip("location to find in: all = all open folders in browser; file = current active file; dir = directory of current active file; nottop = all except the top-level in browser")

	cf := gi.NewChooser(fb, "loc").SetTooltip(locl.Tooltip)
	cf.SetEnum(fv.Params().Loc, false)
	cf.SetCurVal(fv.Params().Loc)
	cf.OnChange(func(e events.Event) {
		if eval, ok := cf.CurVal.(FindLoc); ok {
			fv.Params().Loc = eval
		}
	})

	//////////////// ReplBar

	gi.NewButton(rb).SetIcon(icons.KeyboardArrowUp).SetTooltip("go to previous result").
		OnClick(func(e events.Event) {
			fv.PrevFind()
		})
	gi.NewButton(rb, "next").SetIcon(icons.KeyboardArrowDown).SetTooltip("go to next result").
		OnClick(func(e events.Event) {
			fv.NextFind()
		})
	gi.NewButton(rb).SetText("Replace:").SetTooltip("Replace find string with replace string for currently-selected find result").
		OnClick(func(e events.Event) {
			fv.ReplaceAction()
		})

	repls := gi.NewChooser(rb, "repl-str").SetEditable(true).SetAllowNew(true).
		SetTooltip("String to replace find string -- click for history -- use ${n} for regexp submatch where n = 1 for first submatch, etc")
	repls.Style(func(s *styles.Style) {
		s.Grow.Set(1, 0)
	})
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
