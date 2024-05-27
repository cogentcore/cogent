// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"strings"

	"cogentcore.org/core/events"
	"cogentcore.org/core/parse/lexer"
	"cogentcore.org/core/spell"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/texteditor/textbuf"

	"cogentcore.org/core/core"
	"cogentcore.org/core/views"
)

// SpellView is a widget that displays results of spell check
type SpellView struct {
	core.Frame

	// parent code project
	Code *CodeView `json:"-" xml:"-" copier:"-"`

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

// SpellAction runs a new spell check with current params
func (sv *SpellView) SpellAction() {
	uf := sv.UnknownText()
	uf.SetText("")

	sf := sv.ChangeText()
	sf.SetText("")

	sv.Code.Spell()
}

func (sv *SpellView) OnInit() {
	sv.Frame.OnInit()
	sv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	texteditor.InitSpell()
	sv.CheckNext() // todo: on start

	sv.Maker(func(p *core.Plan) {
		core.AddAt(p, "spellbar", func(w *core.Toolbar) {
			w.Maker(func(p *core.Plan) {
				core.Add(p, func(w *core.Button) {
					w.SetText("Check Current File").
						SetTooltip("spell check the current file").
						OnClick(func(e events.Event) {
							sv.SpellAction()
						})
				})
				core.Add(p, func(w *core.Button) {
					w.SetText("Train").
						SetTooltip("add additional text to the training corpus").
						OnClick(func(e events.Event) {
							sv.TrainAction()
						})
				})
			})
		})
		core.AddAt(p, "unknownbar", func(w *core.Toolbar) {
			w.Maker(func(p *core.Plan) {
				core.AddAt(p, "unknown-str", func(w *core.TextField) {
					w.SetTooltip("Unknown word")
					w.SetReadOnly(true)
				})
				core.Add(p, func(w *core.Button) {
					w.SetText("Skip").
						OnClick(func(e events.Event) {
							sv.SkipAction()
						})
				})
				core.Add(p, func(w *core.Button) {
					w.SetText("Ignore").
						OnClick(func(e events.Event) {
							sv.IgnoreAction()
						})
				})
				core.Add(p, func(w *core.Button) {
					w.SetText("Learn").
						OnClick(func(e events.Event) {
							sv.LearnAction()
						})
				})
			})
		})
		core.AddAt(p, "changebar", func(w *core.Toolbar) {
			w.Maker(func(p *core.Plan) {
				core.AddAt(p, "change-str", func(w *core.TextField) {
					w.SetTooltip("This string will replace the unknown word in text")
				})
				core.Add(p, func(w *core.Button) {
					w.SetText("Change").
						SetTooltip("change the unknown word to the selected suggestion").
						OnClick(func(e events.Event) {
							sv.ChangeAction()
						})
				})
				core.Add(p, func(w *core.Button) {
					w.SetText("Change All").
						SetTooltip("change all instances of the unknown word in this document").
						OnClick(func(e events.Event) {
							sv.ChangeAllAction()
						})
				})
			})
		})
		core.AddAt(p, "suggest", func(w *views.SliceView) {
			sv.Suggest = []string{"                                              "}
			w.SetReadOnly(true)
			w.SetProperty("index", false)
			w.SetSlice(&sv.Suggest)
			// w.SliceViewSig.Connect(suggest, func(recv, send tree.Node, sig int64, data any) {
			// 	svv := recv.Embed(views.KiT_SliceView).(*views.SliceView)
			// 	idx := svv.SelectedIndex
			// 	if idx >= 0 && idx < len(sv.Suggest) {
			// 		sv.AcceptSuggestion(sv.Suggest[svv.SelectedIndex])
			// 	}
			// })
		})
	})
}

func (sv *SpellView) Config(cv *CodeView, atv *TextEditor) { // TODO(config): better name?
	sv.Code = cv
	sv.Text = atv
	sv.CurLn = 0
	sv.CurIndex = 0
	sv.Errs = nil
}

// SpellBar returns the spell toolbar
func (sv *SpellView) SpellBar() *core.Toolbar {
	return sv.ChildByName("spellbar", 0).(*core.Toolbar)
}

// UnknownBar returns the toolbar that displays the unknown word
func (sv *SpellView) UnknownBar() *core.Toolbar {
	return sv.ChildByName("unknownbar", 0).(*core.Toolbar)
}

// ChangeBar returns the suggest toolbar
func (sv *SpellView) ChangeBar() *core.Toolbar {
	return sv.ChildByName("changebar", 0).(*core.Toolbar)
}

// ChangeAct returns the spell change action from toolbar
func (sv *SpellView) ChangeAct() *core.Button {
	return sv.ChangeBar().ChildByName("change", 3).(*core.Button)
}

// ChangeAllAct returns the spell change action from toolbar
func (sv *SpellView) ChangeAllAct() *core.Button {
	return sv.ChangeBar().ChildByName("change-all", 3).(*core.Button)
}

// SkipAct returns the skip action from toolbar
func (sv *SpellView) SkipAct() *core.Button {
	return sv.UnknownBar().ChildByName("skip", 3).(*core.Button)
}

// IgnoreAct returns the ignore action from toolbar
func (sv *SpellView) IgnoreAct() *core.Button {
	return sv.UnknownBar().ChildByName("ignore", 3).(*core.Button)
}

// LearnAct returns the learn action from toolbar
func (sv *SpellView) LearnAct() *core.Button {
	return sv.UnknownBar().ChildByName("learn", 3).(*core.Button)
}

// UnknownText returns the unknown word textfield from toolbar
func (sv *SpellView) UnknownText() *core.TextField {
	return sv.UnknownBar().ChildByName("unknown-str", 1).(*core.TextField)
}

// ChangeText returns the unknown word textfield from toolbar
func (sv *SpellView) ChangeText() *core.TextField {
	return sv.ChangeBar().ChildByName("change-str", 1).(*core.TextField)
}

// SuggestView returns the view for the list of suggestions
func (sv *SpellView) SuggestView() *views.SliceView {
	return sv.ChildByName("suggest", 1).(*views.SliceView)
}

// CheckNext will find the next misspelled/unknown word and get suggestions for replacing it
func (sv *SpellView) CheckNext() {
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
			word := string(lx.Src(tv.Buffer.Lines[sv.CurLn]))
			_, known := spell.CheckWord(word) // could have been fixed by now..
			if known {
				sv.CurIndex++
				continue
			}
			break
		} else {
			sv.CurLn++
			if sv.CurLn >= tv.NLines {
				done = true
				break
			}
			sv.CurIndex = 0
			sv.Errs = tv.Buffer.SpellCheckLineErrs(sv.CurLn)
		}
	}
	if done {
		tv.ClearHighlights()
		core.MessageSnackbar(sv, "End of file, spelling check complete")
		return
	}
	sv.UnkLex = sv.Errs[sv.CurIndex]
	sv.CurIndex++
	sv.UnkWord = string(sv.UnkLex.Src(tv.Buffer.Lines[sv.CurLn]))
	sv.Suggest, _ = spell.CheckWord(sv.UnkWord)

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
	hr := textbuf.Region{Start: st, End: en}
	hr.TimeNow()
	tv.Highlights = append(tv.Highlights, hr)
	if sv.LastAction == nil {
		sv.SetFocusEvent()
	} else {
		sv.LastAction.SetFocusEvent()
	}
	tv.NeedsRender()
}

// ChangeAction replaces the known word with the selected suggested word
// and call CheckNextAction
func (sv *SpellView) ChangeAction() {
	tv := sv.Text
	if tv == nil || tv.Buffer == nil {
		return
	}
	st := sv.UnkStartPos()
	en := sv.UnkEndPos()
	ct := sv.ChangeText()
	tv.Buffer.ReplaceText(st, en, st, ct.Text(), texteditor.EditSignal, texteditor.ReplaceNoMatchCase)
	nwrs := tv.Buffer.AdjustedTagsImpl(sv.Errs, sv.CurLn) // update tags
	if len(nwrs) == len(sv.Errs)-1 && sv.CurIndex > 0 {   // Adjust got rid of changed one..
		sv.CurIndex--
	}
	sv.Errs = nwrs
	sv.LastAction = sv.ChangeAct()
	sv.CheckNext()
}

// ChangeAllAction replaces the known word with the selected suggested word
// and call CheckNextAction
func (sv *SpellView) ChangeAllAction() {
	tv := sv.Text
	if tv == nil || tv.Buffer == nil {
		return
	}
	tv.QReplaceStart(sv.UnkWord, sv.ChangeText().Txt, false)
	tv.QReplaceReplaceAll(0)
	sv.LastAction = sv.ChangeAllAct()
	sv.Errs = tv.Buffer.AdjustedTagsImpl(sv.Errs, sv.CurLn) // update tags
	sv.CheckNext()
}

// TrainAction allows you to train on additional text files and also to rebuild the spell model
func (sv *SpellView) TrainAction() {
	cur := ""
	d := core.NewBody().AddTitle("Select a Text File to Add to Corpus")
	fv := views.NewFileView(d).SetFilename(cur, ".txt")
	fv.OnSelect(func(e events.Event) {
		cur = fv.SelectedFile()
	}).OnDoubleClick(func(e events.Event) {
		cur = fv.SelectedFile()
		d.Close()
	})
	d.AddBottomBar(func(parent core.Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).OnClick(func(e events.Event) {
			texteditor.AddToSpellModel(cur)
		})
	})
	d.RunFullDialog(sv)
}

// UnkStartPos returns the start position of the current unknown word
func (sv *SpellView) UnkStartPos() lexer.Pos {
	pos := lexer.Pos{Ln: sv.CurLn, Ch: sv.UnkLex.St}
	return pos
}

// UnkEndPos returns the end position of the current unknown word
func (sv *SpellView) UnkEndPos() lexer.Pos {
	pos := lexer.Pos{Ln: sv.CurLn, Ch: sv.UnkLex.Ed}
	return pos
}

// SkipAction will skip this single instance of misspelled/unknown word
// and call CheckNextAction
func (sv *SpellView) SkipAction() {
	sv.LastAction = sv.SkipAct()
	sv.CheckNext()
}

// IgnoreAction will skip this and future instances of misspelled/unknown word
// and call CheckNextAction
func (sv *SpellView) IgnoreAction() {
	spell.IgnoreWord(sv.UnkWord)
	sv.LastAction = sv.IgnoreAct()
	sv.CheckNext()
}

// LearnAction will add the current unknown word to corpus
// and call CheckNext
func (sv *SpellView) LearnAction() {
	nw := strings.ToLower(sv.UnkWord)
	spell.LearnWord(nw)
	sv.LastAction = sv.LearnAct()
	sv.CheckNext()
}

// AcceptSuggestion replaces the misspelled word with the word in the ChangeText field
func (sv *SpellView) AcceptSuggestion(s string) {
	ct := sv.ChangeText()
	ct.SetText(s)
	sv.ChangeAction()
}

func (sv *SpellView) Destroy() {
	tv := sv.Text
	if tv == nil || tv.Buffer == nil || tv.This() == nil {
		return
	}
	tv.ClearHighlights()
}
