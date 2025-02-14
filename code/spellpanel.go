// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"strings"

	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/spell"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/texteditor/text"
	"cogentcore.org/core/tree"

	"cogentcore.org/core/core"
)

// SpellPanel is a widget that displays results of a spell check.
type SpellPanel struct {
	core.Frame

	// parent code project
	Code *Code `json:"-" xml:"-" copier:"-"`

	// texteditor that we're spell-checking
	Text *TextEditor `json:"-" xml:"-" copier:"-"`

	// current spelling errors
	Errs lexer.Line

	// current line in text we're on
	CurLn int

	// current index in Errs we're on
	CurIndex int

	// current unknown lex token
	UnkLex lexer.Lex

	// current unknown word
	UnkWord string

	// a list of suggestions from spell checker
	Suggest []string

	// last user action (ignore, change, learn)
	LastAction *core.Button
}

func (sv *SpellPanel) Init() {
	sv.Frame.Init()
	sv.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	sv.CheckNext() // todo: on start

	tree.AddChildAt(sv, "spellbar", func(w *core.Toolbar) {
		tree.AddChild(w, func(w *core.Button) {
			w.SetText("Check Current File").
				SetTooltip("spell check the current file").
				OnClick(func(e events.Event) {
					uf := sv.UnknownText()
					uf.SetText("")
					sf := sv.ChangeText()
					sf.SetText("")
					sv.Code.Spell()
				})
		})
		tree.AddChild(w, func(w *core.Button) {
			w.SetText("Train").
				SetTooltip("add additional text to the training corpus").
				OnClick(func(e events.Event) {
					cur := ""
					d := core.NewBody("Select a Text File to Add to Corpus")
					fv := core.NewFilePicker(d).SetFilename(cur).SetExtensions(".txt")
					fv.OnSelect(func(e events.Event) {
						cur = fv.SelectedFile()
					})
					fv.OnDoubleClick(func(e events.Event) {
						cur = fv.SelectedFile()
						d.Close()
					})
					d.AddBottomBar(func(bar *core.Frame) {
						d.AddCancel(bar)
						d.AddOK(bar).OnClick(func(e events.Event) {
							spell.Spell.AddWord(cur)
						})
					})
					d.RunFullDialog(sv)
				})
		})
	})
	tree.AddChildAt(sv, "unknownbar", func(w *core.Toolbar) {
		tree.AddChildAt(w, "unknown-str", func(w *core.TextField) {
			w.SetTooltip("Unknown word")
			w.SetReadOnly(true)
		})
		tree.AddChild(w, func(w *core.Button) {
			w.SetText("Skip").OnClick(func(e events.Event) {
				sv.LastAction = w
				sv.CheckNext()
			})
		})
		tree.AddChild(w, func(w *core.Button) {
			w.SetText("Ignore").OnClick(func(e events.Event) {
				spell.Spell.IgnoreWord(sv.UnkWord)
				sv.LastAction = w
				sv.CheckNext()
			})
		})
		tree.AddChild(w, func(w *core.Button) {
			w.SetText("Learn").OnClick(func(e events.Event) {
				nw := strings.ToLower(sv.UnkWord)
				spell.Spell.AddWord(nw)
				sv.LastAction = w
				sv.CheckNext()
			})
		})
	})
	tree.AddChildAt(sv, "changebar", func(w *core.Toolbar) {
		tree.AddChildAt(w, "change-str", func(w *core.TextField) {
			w.SetTooltip("This string will replace the unknown word in text")
		})
		tree.AddChild(w, func(w *core.Button) {
			w.SetText("Change").SetTooltip("change the unknown word to the selected suggestion").
				OnClick(func(e events.Event) {
					sv.LastAction = w
					sv.Change()
				})
		})
		tree.AddChild(w, func(w *core.Button) {
			w.SetText("Change All").SetTooltip("change all instances of the unknown word in this document").
				OnClick(func(e events.Event) {
					tv := sv.Text
					if tv == nil || tv.Buffer == nil {
						return
					}
					tv.QReplaceStart(sv.UnkWord, sv.ChangeText().Text(), false)
					tv.QReplaceReplaceAll(0)
					sv.LastAction = w
					sv.Errs = tv.Buffer.AdjustedTagsLine(sv.Errs, sv.CurLn) // update tags
					sv.CheckNext()
				})
		})
	})
	tree.AddChildAt(sv, "suggest", func(w *core.List) {
		sv.Suggest = []string{"                                              "}
		w.SetReadOnly(true)
		w.SetProperty("index", false)
		w.SetSlice(&sv.Suggest)
		w.OnSelect(func(e events.Event) {
			sv.AcceptSuggestion(sv.Suggest[w.SelectedIndex])
		})
	})
}

func (sv *SpellPanel) Config(cv *Code, atv *TextEditor) { // TODO(config): better name?
	sv.Code = cv
	sv.Text = atv
	sv.CurLn = 0
	sv.CurIndex = 0
	sv.Errs = nil
}

// SpellBar returns the spell toolbar
func (sv *SpellPanel) SpellBar() *core.Toolbar {
	return sv.ChildByName("spellbar", 0).(*core.Toolbar)
}

// UnknownBar returns the toolbar that displays the unknown word
func (sv *SpellPanel) UnknownBar() *core.Toolbar {
	return sv.ChildByName("unknownbar", 0).(*core.Toolbar)
}

// ChangeBar returns the suggest toolbar
func (sv *SpellPanel) ChangeBar() *core.Toolbar {
	return sv.ChildByName("changebar", 0).(*core.Toolbar)
}

// UnknownText returns the unknown word textfield from toolbar
func (sv *SpellPanel) UnknownText() *core.TextField {
	return sv.UnknownBar().ChildByName("unknown-str", 1).(*core.TextField)
}

// ChangeText returns the unknown word textfield from toolbar
func (sv *SpellPanel) ChangeText() *core.TextField {
	return sv.ChangeBar().ChildByName("change-str", 1).(*core.TextField)
}

// SuggestView returns the view for the list of suggestions
func (sv *SpellPanel) SuggestView() *core.List {
	return sv.ChildByName("suggest", 1).(*core.List)
}

// CheckNext will find the next misspelled/unknown word and get suggestions for replacing it
func (sv *SpellPanel) CheckNext() {
	tv := sv.Text
	if tv == nil || tv.Buffer == nil {
		return
	}
	if sv.CurLn == 0 && sv.Errs == nil {
		sv.CurLn = -1
	}
	done := false
	for {
		if sv.CurIndex < len(sv.Errs) {
			lx := sv.Errs[sv.CurIndex]
			word := string(lx.Src(tv.Buffer.Line(sv.CurLn)))
			_, known := spell.Spell.CheckWord(word) // could have been fixed by now..
			if known {
				sv.CurIndex++
				continue
			}
			break
		} else {
			sv.CurLn++
			if sv.CurLn >= tv.NumLines {
				done = true
				break
			}
			sv.CurIndex = 0
			sv.Errs = tv.Buffer.SpellCheckLineErrors(sv.CurLn)
		}
	}
	if done {
		tv.ClearHighlights()
		core.MessageSnackbar(sv, "End of file, spelling check complete")
		return
	}
	sv.UnkLex = sv.Errs[sv.CurIndex]
	sv.CurIndex++
	sv.UnkWord = string(sv.UnkLex.Src(tv.Buffer.Line(sv.CurLn)))
	sv.Suggest, _ = spell.Spell.CheckWord(sv.UnkWord)

	uf := sv.UnknownText()
	uf.SetText(sv.UnkWord)

	cf := sv.ChangeText()
	if len(sv.Suggest) == 0 {
		cf.SetText("")
	} else {
		cf.SetText(sv.Suggest[0])
	}

	sugview := sv.SuggestView()
	sugview.Slice = &sv.Suggest
	sugview.Update()

	st := sv.UnkStartPos()
	en := sv.UnkEndPos()

	tv.Highlights = tv.Highlights[:0]
	tv.SetCursorTarget(st)
	hr := text.Region{Start: st, End: en}
	hr.TimeNow()
	tv.Highlights = append(tv.Highlights, hr)
	if sv.LastAction == nil {
		sv.SetFocus()
	} else {
		sv.LastAction.SetFocus()
	}
	tv.NeedsRender()
}

// UnkStartPos returns the start position of the current unknown word
func (sv *SpellPanel) UnkStartPos() textpos.Pos {
	pos := textpos.Pos{Ln: sv.CurLn, Ch: sv.UnkLex.St}
	return pos
}

// UnkEndPos returns the end position of the current unknown word
func (sv *SpellPanel) UnkEndPos() textpos.Pos {
	pos := textpos.Pos{Ln: sv.CurLn, Ch: sv.UnkLex.Ed}
	return pos
}

func (sv *SpellPanel) Change() {
	tv := sv.Text
	if tv == nil || tv.Buffer == nil {
		return
	}
	st := sv.UnkStartPos()
	en := sv.UnkEndPos()
	ct := sv.ChangeText()
	tv.Buffer.ReplaceText(st, en, st, ct.Text(), texteditor.EditSignal, texteditor.ReplaceNoMatchCase)
	nwrs := tv.Buffer.AdjustedTagsLine(sv.Errs, sv.CurLn) // update tags
	if len(nwrs) == len(sv.Errs)-1 && sv.CurIndex > 0 {   // Adjust got rid of changed one..
		sv.CurIndex--
	}
	sv.Errs = nwrs
	sv.CheckNext()
}

// AcceptSuggestion replaces the misspelled word with the word in the ChangeText field
func (sv *SpellPanel) AcceptSuggestion(s string) {
	ct := sv.ChangeText()
	ct.SetText(s)
	sv.Change()
}

func (sv *SpellPanel) Destroy() {
	tv := sv.Text
	if tv == nil || tv.Buffer == nil || tv.This == nil {
		return
	}
	tv.ClearHighlights()
}
