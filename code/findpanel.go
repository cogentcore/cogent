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

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/stringsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/tree"
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
	Loc filetree.FindLocation

	// languages for files to search
	Languages []fileinfo.Known

	// history of finds
	FindHist []string

	// history of replaces
	ReplHist []string
}

// FindPanel is a find / replace widget that displays results in a [TextEditor]
// and has a toolbar for controlling find / replace process.
type FindPanel struct {
	core.Frame

	// parent code project
	Code *Code `json:"-" xml:"-"`

	// time of last find
	Time time.Time

	// compiled regexp
	Re *regexp.Regexp
}

func (fv *FindPanel) Init() {
	fv.Frame.Init()
	fv.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})

	tree.AddChildAt(fv, "findbar", func(w *core.Frame) {
		core.ToolbarStyles(w)
		w.Maker(fv.makeFindToolbar)
	})
	tree.AddChildAt(fv, "replbar", func(w *core.Frame) {
		core.ToolbarStyles(w)
		w.Maker(fv.makeReplToolbar)
	})
	tree.AddChildAt(fv, "findtext", func(w *textcore.Editor) {
		ConfigOutputTextEditor(w)
		w.LinkHandler = func(tl *paint.TextLink) {
			fv.OpenFindURL(tl.URL, w)
		}
	})
}

func (fv *FindPanel) OnAdd() {
	fv.Frame.OnAdd()
	fv.Code, _ = ParentCode(fv)
}

// Params returns the find params
func (fv *FindPanel) Params() *FindParams {
	return &fv.Code.Settings.Find
}

// ShowResults shows the results in the buffer
func (fv *FindPanel) ShowResults(res []filetree.SearchResults) {
	ftv := fv.TextEditor()
	fbuf := ftv.Lines
	fbuf.Options.LineNumbers = false
	outlns := make([][]byte, 0, 100)
	outmus := make([][]byte, 0, 100) // markups
	for _, fs := range res {
		fp := fs.Node.Info.Path
		fn := fs.Node.RelativePath()
		fbStLn := len(outlns) // find buf start ln
		lstr := fmt.Sprintf(`%v: %v`, fn, fs.Count)
		outlns = append(outlns, []byte(lstr))
		mstr := fmt.Sprintf(`<b>%v</b>`, lstr)
		outmus = append(outmus, []byte(mstr))
		for _, mt := range fs.Matches {
			txt := bytes.TrimSpace(mt.Text)
			txt = append([]byte{'\t'}, txt...)
			ln := mt.Reg.Start.Line + 1
			ch := mt.Reg.Start.Char + 1
			ech := mt.Reg.End.Char + 1
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
	fbuf.AppendTextMarkup(ltxt, mtxt)
	ftv.CursorStartDoc()

	fv.Update()
	ftv.SetCursorShow(textpos.Pos{Ln: 0})
	ftv.NeedsLayout()
	ok := ftv.CursorNextLink(false) // no wrap
	if ok {
		ftv.OpenLinkAt(ftv.CursorPos)
	}
}

// FindAction runs a new find with current params
func (fv *FindPanel) FindAction() {
	fp := fv.Params()
	if !fv.CompileRegexp() {
		return
	}
	fv.Code.Find(fp.Find, fp.Replace, fp.IgnoreCase, fp.Regexp, fp.Loc, fp.Languages)
}

// CheckValidRegexp returns false if using regexp and it is not valid
func (fv *FindPanel) CheckValidRegexp() bool {
	fp := fv.Params()
	if !fp.Regexp {
		return true
	}
	if fv.Re == nil {
		return fv.CompileRegexp()
	}
	return true
}

// ReplaceAction performs the replace -- if using regexp mode, regexp must be compiled in advance
func (fv *FindPanel) ReplaceAction() bool {
	if !fv.CheckValidRegexp() {
		return false
	}
	fp := fv.Params()
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
	reg = tv.Lines.AdjustRegion(reg)
	if !reg.IsNil() {
		if fp.Regexp {
			rg := tv.Lines.Region(reg.Start, reg.End)
			b := rg.ToBytes()
			rb := fv.Re.ReplaceAll(b, []byte(fp.Replace))
			tv.Lines.ReplaceText(reg.Start, reg.End, reg.Start, string(rb), false)
		} else {
			// MatchCase only if doing IgnoreCase
			tv.Lines.ReplaceText(reg.Start, reg.End, reg.Start, fp.Replace, fp.IgnoreCase)
		}

		// delete the link for the just done replace
		ftvln := ftv.CursorPos.Line
		st := textpos.Pos{Ln: ftvln, Ch: 0}
		len := ftv.Lines.LineLen(ftvln)
		en := textpos.Pos{Ln: ftvln, Ch: len}
		ftv.Lines.DeleteText(st, en)
	}

	tv.ClearHighlights()

	ok = ftv.CursorNextLink(false) // no wrap
	if ok {
		ftv.OpenLinkAt(ftv.CursorPos) // move to next
	}
	fv.NeedsRender()
	return ok
}

// ReplaceAllAction performs replace all, prompting before proceeding
func (fv *FindPanel) ReplaceAllAction() {
	d := core.NewBody("Confirm replace all")
	core.NewText(d).SetType(core.TextSupporting).SetText("Are you sure you want to replace all?")
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Replace all").OnClick(func(e events.Event) {
			fv.ReplaceAll()
		})
	})
	d.RunDialog(fv)
}

// CompileRegexp compiles the regexp if necessary -- returns false if it is invalid
func (fv *FindPanel) CompileRegexp() bool {
	fp := fv.Params()
	if !fp.Regexp {
		fv.Re = nil
		return true
	}
	var err error
	fv.Re, err = regexp.Compile(fp.Find)
	if err != nil {
		core.ErrorSnackbar(fv, err, "The regular expression was invalid")
		return false
	}
	return true
}

// ReplaceAll performs replace all
func (fv *FindPanel) ReplaceAll() {
	if !fv.CheckValidRegexp() {
		return
	}
	wasGoMod := fv.Code.Settings.GoMod
	fv.Code.Settings.GoMod = false
	SetGoMod(false) // much faster without
	go func() {
		for {
			sc := fv.Code.AsWidget().Scene
			sc.AsyncLock()
			ok := fv.ReplaceAction()
			sc.NeedsLayout()
			sc.AsyncUnlock()
			if !ok {
				break
			}
		}
		if wasGoMod {
			fv.Code.Settings.GoMod = true
			SetGoMod(true)
		}
	}()
}

// NextFind shows next find result
func (fv *FindPanel) NextFind() {
	ftv := fv.TextEditor()
	ok := ftv.CursorNextLink(true) // wrap
	if ok {
		ftv.OpenLinkAt(ftv.CursorPos)
	}
}

// PrevFind shows previous find result
func (fv *FindPanel) PrevFind() {
	ftv := fv.TextEditor()
	ok := ftv.CursorPrevLink(true) // wrap
	if ok {
		ftv.OpenLinkAt(ftv.CursorPos)
	}
}

// OpenFindURL opens given find:/// url from Find
func (fv *FindPanel) OpenFindURL(ur string, ftv *textcore.Editor) bool {
	ge := fv.Code
	tv, reg, fbBufStLn, fCount, ok := ge.ParseOpenFindURL(ur, ftv)
	if !ok {
		return false
	}
	reg.Time.SetTime(fv.Time)
	reg = tv.Lines.AdjustRegion(reg)
	find := fv.Params().Find
	textcore.PrevISearchString = find
	tve := textcore.AsEditor(tv)
	fv.HighlightFinds(tve, ftv, fbBufStLn, fCount, find)
	tv.SetCursorTarget(reg.Start)
	tv.NeedsLayout()
	return true
}

// HighlightFinds highlights all the find results in ftv buffer
func (fv *FindPanel) HighlightFinds(tv, ftv *textcore.Editor, fbStLn, fCount int, find string) {
	lnka := []byte(`<a href="`)
	lnkasz := len(lnka)

	fb := ftv.Lines

	if len(tv.Highlights) != fCount { // highlight
		hi := make([]text.Region, fCount)
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
			ireg := text.Region{}
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

// TextEditorLay returns the find results TextEditor
func (fv *FindPanel) TextEditor() *textcore.Editor {
	return textcore.AsEditor(fv.ChildByName("findtext", 1))
}

// makeFindToolbar
func (fv *FindPanel) makeFindToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.Button) {
		w.SetText("Find:").SetTooltip("Find given string in project files. Only open folders in file browser will be searched -- adjust those to scope the search").
			OnClick(func(e events.Event) {
				fv.FindAction()
			})
	})
	tree.AddAt(p, "find-str", func(w *core.Chooser) {
		w.SetEditable(true).SetDefaultNew(true).
			SetTooltip("String to find -- hit enter or tab to update search -- click for history")
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Max.X.Zero()
		})
		w.OnChange(func(e events.Event) {
			find := w.CurrentItem.Value.(string)
			fv.Params().Find = find
			if find == "" {
				tv := fv.Code.ActiveTextEditor()
				if tv != nil {
					tv.ClearHighlights()
				}
				fvtv := fv.TextEditor()
				if fvtv != nil {
					fvtv.Lines.SetText(nil)
				}
			} else {
				stringsx.InsertFirstUnique(&fv.Params().FindHist, find, core.SystemSettings.SavedPathsMax)
				fv.FindAction()
			}
		})
		w.Updater(func() {
			w.SetCurrentValue(fv.Params().Find)
		})
	})

	tree.AddAt(p, "ignore-case", func(w *core.Switch) {
		w.SetText("Ignore Case")
		w.OnChange(func(e events.Event) {
			fv.Params().IgnoreCase = w.StateIs(states.Checked)
		})
		w.Updater(func() {
			w.SetChecked(fv.Params().IgnoreCase)
		})
	})
	tree.AddAt(p, "regexp", func(w *core.Switch) {
		w.SetText("Regexp").
			SetTooltip("use regular expression for search and replace -- see https://github.com/google/re2/wiki/Syntax")
		w.OnChange(func(e events.Event) {
			fv.Params().Regexp = w.StateIs(states.Checked)
		})
		w.Updater(func() {
			w.SetChecked(fv.Params().Regexp)
		})
	})

	ttxt := "location to find in: all = all open folders in browser; file = current active file; dir = directory of current active file; nottop = all except the top-level in browser"
	tree.Add(p, func(w *core.Text) {
		w.SetText("Loc:").
			SetTooltip(ttxt)
	})

	tree.AddAt(p, "loc", func(w *core.Chooser) {
		w.SetTooltip(ttxt)
		w.SetEnum(fv.Params().Loc)
		w.OnChange(func(e events.Event) {
			if eval, ok := w.CurrentItem.Value.(filetree.FindLocation); ok {
				fv.Params().Loc = eval
			}
		})
		w.Updater(func() {
			w.SetCurrentValue(fv.Params().Loc)
		})
	})
}

func (fv *FindPanel) makeReplToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.KeyboardArrowUp).SetTooltip("go to previous result").
			OnClick(func(e events.Event) {
				fv.PrevFind()
			})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.KeyboardArrowDown).SetTooltip("go to next result").
			OnClick(func(e events.Event) {
				fv.NextFind()
			})
		w.StartFocus()
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("Replace:").SetTooltip("Replace find string with replace string for currently selected find result").
			OnClick(func(e events.Event) {
				fv.ReplaceAction()
			})
	})

	tree.AddAt(p, "repl-str", func(w *core.Chooser) {
		w.SetEditable(true).SetDefaultNew(true).
			SetTooltip("String to replace find string -- click for history -- use ${n} for regexp submatch where n = 1 for first submatch, etc")
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Max.X.Zero()
		})
		w.OnChange(func(e events.Event) {
			repl := w.CurrentItem.Value.(string)
			fv.Params().Replace = repl
			stringsx.InsertFirstUnique(&fv.Params().ReplHist, repl, core.SystemSettings.SavedPathsMax)
		})
		w.Updater(func() {
			w.SetCurrentValue(fv.Params().Replace)
		})
	})

	tree.Add(p, func(w *core.Button) {
		w.SetText("All").SetTooltip("replace all find strings with replace string").
			OnClick(func(e events.Event) {
				fv.ReplaceAllAction()
			})
	})

	tree.Add(p, func(w *core.Text) {
		w.SetText("Lang:").SetTooltip("Language(s) to restrict search / replace to")
	})

	tree.AddAt(p, "languages", func(w *core.InlineList) {
		w.SetSlice(&fv.Params().Languages)
		w.SetTooltip("Language(s) to restrict search / replace to")
	})
}
