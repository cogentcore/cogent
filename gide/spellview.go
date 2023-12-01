// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"strings"

	"goki.dev/gi/v2/texteditor"
	"goki.dev/gi/v2/texteditor/textbuf"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/pi/v2/lex"
	"goki.dev/spell"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/ki/v2"
)

// SpellView is a widget that displays results of spell check
type SpellView struct {
	gi.Layout

	// parent gide project
	Gide Gide `json:"-" xml:"-" copy:"-"`

	// textview that we're spell-checking
	Text *TextEditor `json:"-" xml:"-" copy:"-"`

	// current spelling errors
	Errs lex.Line

	// current line in text we're on
	CurLn int

	// current index in Errs we're on
	CurIdx int

	// current unknown lex token
	UnkLex lex.Lex

	// current unknown word
	UnkWord string

	// a list of suggestions from spell checker
	Suggest []string

	// last user action (ignore, change, learn)
	LastAction *gi.Button
}

// SpellAction runs a new spell check with current params
func (sv *SpellView) SpellAction() {
	uf := sv.UnknownText()
	uf.SetText("")

	sf := sv.ChangeText()
	sf.SetText("")

	sv.Gide.Spell()
}

//////////////////////////////////////////////////////////////////////////////////////
//    GUI config

// Config configures the view
func (sv *SpellView) ConfigSpellView(ge Gide, atv *TextEditor) {
	sv.Gide = ge
	sv.Text = atv
	sv.CurLn = 0
	sv.CurIdx = 0
	sv.Errs = nil
	if sv.HasChildren() {
		return
	}
	sv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	gi.NewToolbar(sv, "spellbar")
	gi.NewToolbar(sv, "unknownbar")
	gi.NewToolbar(sv, "changebar")
	giv.NewSliceView(sv, "suggest")
	sv.ConfigToolbar()
	gi.InitSpell()
	sv.CheckNext()
}

// SpellBar returns the spell toolbar
func (sv *SpellView) SpellBar() *gi.Toolbar {
	return sv.ChildByName("spellbar", 0).(*gi.Toolbar)
}

// UnknownBar returns the toolbar that displays the unknown word
func (sv *SpellView) UnknownBar() *gi.Toolbar {
	return sv.ChildByName("unknownbar", 0).(*gi.Toolbar)
}

// ChangeBar returns the suggest toolbar
func (sv *SpellView) ChangeBar() *gi.Toolbar {
	return sv.ChildByName("changebar", 0).(*gi.Toolbar)
}

// ChangeAct returns the spell change action from toolbar
func (sv *SpellView) ChangeAct() *gi.Button {
	return sv.ChangeBar().ChildByName("change", 3).(*gi.Button)
}

// ChangeAllAct returns the spell change action from toolbar
func (sv *SpellView) ChangeAllAct() *gi.Button {
	return sv.ChangeBar().ChildByName("change-all", 3).(*gi.Button)
}

// SkipAct returns the skip action from toolbar
func (sv *SpellView) SkipAct() *gi.Button {
	return sv.UnknownBar().ChildByName("skip", 3).(*gi.Button)
}

// IgnoreAct returns the ignore action from toolbar
func (sv *SpellView) IgnoreAct() *gi.Button {
	return sv.UnknownBar().ChildByName("ignore", 3).(*gi.Button)
}

// LearnAct returns the learn action from toolbar
func (sv *SpellView) LearnAct() *gi.Button {
	return sv.UnknownBar().ChildByName("learn", 3).(*gi.Button)
}

// UnknownText returns the unknown word textfield from toolbar
func (sv *SpellView) UnknownText() *gi.TextField {
	return sv.UnknownBar().ChildByName("unknown-str", 1).(*gi.TextField)
}

// ChangeText returns the unknown word textfield from toolbar
func (sv *SpellView) ChangeText() *gi.TextField {
	return sv.ChangeBar().ChildByName("change-str", 1).(*gi.TextField)
}

// SuggestView returns the view for the list of suggestions
func (sv *SpellView) SuggestView() *giv.SliceView {
	return sv.ChildByName("suggest", 1).(*giv.SliceView)
}

// ConfigToolbar adds toolbar.
func (sv *SpellView) ConfigToolbar() {
	spbar := sv.SpellBar()
	unknbar := sv.UnknownBar()
	chgbar := sv.ChangeBar()

	// spell toolbar
	gi.NewButton(spbar).SetText("Check Current File").
		SetTooltip("spell check the current file").
		OnClick(func(e events.Event) {
			sv.SpellAction()
		})

	gi.NewButton(spbar).SetText("Train").
		SetTooltip("add additional text to the training corpus").
		OnClick(func(e events.Event) {
			sv.TrainAction()
		})
	// todo:
	// train.SetProp("horizontal-align", styles.AlignRight)

	// unknown toolbar

	gi.NewTextField(unknbar, "unknown-str").SetTooltip("Unknown word")
	tf := sv.UnknownText()
	if tf != nil {
		tf.SetReadOnly(true)
	}

	gi.NewButton(unknbar, "skip").SetText("Skip").
		OnClick(func(e events.Event) {
			sv.SkipAction()
		})

	gi.NewButton(unknbar, "ignore").SetText("Ignore").
		OnClick(func(e events.Event) {
			sv.IgnoreAction()
		})

	gi.NewButton(unknbar, "learn").SetText("Learn").
		OnClick(func(e events.Event) {
			sv.LearnAction()
		})

	// change toolbar
	gi.NewTextField(chgbar, "change-str").
		SetTooltip("This string will replace the unknown word in text")

	gi.NewButton(chgbar, "change").SetText("Change").
		SetTooltip("change the unknown word to the selected suggestion").
		OnClick(func(e events.Event) {
			sv.ChangeAction()
		})

	gi.NewButton(chgbar, "change-all").SetText("Change All").
		SetTooltip("change all instances of the unknown word in this document").
		OnClick(func(e events.Event) {
			sv.ChangeAllAction()
		})

	// suggest toolbar
	suggest := sv.SuggestView()
	sv.Suggest = []string{"                                              "}
	suggest.SetReadOnly(true)
	suggest.SetProp("index", false)
	suggest.SetSlice(&sv.Suggest)
	// suggest.SliceViewSig.Connect(suggest, func(recv, send ki.Ki, sig int64, data any) {
	// 	svv := recv.Embed(giv.KiT_SliceView).(*giv.SliceView)
	// 	idx := svv.SelectedIdx
	// 	if idx >= 0 && idx < len(sv.Suggest) {
	// 		sv.AcceptSuggestion(sv.Suggest[svv.SelectedIdx])
	// 	}
	// })
}

// CheckNext will find the next misspelled/unknown word and get suggestions for replacing it
func (sv *SpellView) CheckNext() {
	tv := sv.Text
	if tv == nil || tv.Buf == nil {
		return
	}
	if sv.CurLn == 0 && sv.Errs == nil {
		sv.CurLn = -1
	}
	done := false
	for {
		if sv.CurIdx < len(sv.Errs) {
			lx := sv.Errs[sv.CurIdx]
			word := string(lx.Src(tv.Buf.Lines[sv.CurLn]))
			_, known := spell.CheckWord(word) // could have been fixed by now..
			if known {
				sv.CurIdx++
				continue
			}
			break
		} else {
			sv.CurLn++
			if sv.CurLn >= tv.NLines {
				done = true
				break
			}
			sv.CurIdx = 0
			sv.Errs = tv.Buf.SpellCheckLineErrs(sv.CurLn)
		}
	}
	if done {
		tv.ClearHighlights()
		gi.NewBody().AddTitle("Spelling Check Complete").
			AddText("End of file, spelling check complete").
			AddOkOnly().NewDialog(sv).Run()
		return
	}
	sv.UnkLex = sv.Errs[sv.CurIdx]
	sv.CurIdx++
	sv.UnkWord = string(sv.UnkLex.Src(tv.Buf.Lines[sv.CurLn]))
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
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)

	tv.Highlights = tv.Highlights[:0]
	tv.SetCursorShow(st)
	hr := textbuf.Region{Start: st, End: en}
	hr.TimeNow()
	tv.Highlights = append(tv.Highlights, hr)
	if sv.LastAction == nil {
		sv.SetFocusEvent()
	} else {
		sv.LastAction.SetFocusEvent()
	}
}

// ChangeAction replaces the known word with the selected suggested word
// and call CheckNextAction
func (sv *SpellView) ChangeAction() {
	tv := sv.Text
	if tv == nil || tv.Buf == nil {
		return
	}
	st := sv.UnkStartPos()
	en := sv.UnkEndPos()
	ct := sv.ChangeText()
	tv.Buf.ReplaceText(st, en, st, ct.Text(), texteditor.EditSignal, texteditor.ReplaceNoMatchCase)
	nwrs := tv.Buf.AdjustedTagsImpl(sv.Errs, sv.CurLn) // update tags
	if len(nwrs) == len(sv.Errs)-1 && sv.CurIdx > 0 {  // Adjust got rid of changed one..
		sv.CurIdx--
	}
	sv.Errs = nwrs
	sv.LastAction = sv.ChangeAct()
	sv.CheckNext()
}

// ChangeAllAction replaces the known word with the selected suggested word
// and call CheckNextAction
func (sv *SpellView) ChangeAllAction() {
	tv := sv.Text
	if tv == nil || tv.Buf == nil {
		return
	}
	tv.QReplaceStart(sv.UnkWord, sv.ChangeText().Txt, false)
	tv.QReplaceReplaceAll(0)
	sv.LastAction = sv.ChangeAllAct()
	sv.Errs = tv.Buf.AdjustedTagsImpl(sv.Errs, sv.CurLn) // update tags
	sv.CheckNext()
}

// TrainAction allows you to train on additional text files and also to rebuild the spell model
func (sv *SpellView) TrainAction() {
	cur := ""
	d := gi.NewBody().AddTitle("Select a Text File to Add to Corpus")
	fv := giv.NewFileView(d).SetFilename(cur, ".txt")
	fv.OnSelect(func(e events.Event) {
		cur = fv.SelectedFile()
	}).OnDoubleClick(func(e events.Event) {
		cur = fv.SelectedFile()
		d.Close()
	})
	d.AddBottomBar(func(pw gi.Widget) {
		d.AddCancel(pw)
		d.AddOk(pw).OnClick(func(e events.Event) {
			gi.AddToSpellModel(cur)
		})
	})
	d.NewFullDialog(sv).Run()
}

// UnkStartPos returns the start position of the current unknown word
func (sv *SpellView) UnkStartPos() lex.Pos {
	pos := lex.Pos{Ln: sv.CurLn, Ch: sv.UnkLex.St}
	return pos
}

// UnkEndPos returns the end position of the current unknown word
func (sv *SpellView) UnkEndPos() lex.Pos {
	pos := lex.Pos{Ln: sv.CurLn, Ch: sv.UnkLex.Ed}
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
	if tv == nil || tv.Buf == nil || tv.Is(ki.Deleted) {
		return
	}
	tv.ClearHighlights()
}

// SpellViewProps are style properties for SpellView
// var SpellViewProps = ki.Props{
// 	"EnumType:Flag":    gi.KiT_NodeFlags,
// 	"background-color": &gi.Prefs.Colors.Background,
// 	"color":            &gi.Prefs.Colors.Font,
// 	"max-width":        -1,
// 	"max-height":       -1,
// }
