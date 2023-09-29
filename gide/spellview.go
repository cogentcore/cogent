// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"fmt"
	"strings"

	"github.com/goki/gi/gist"
	"github.com/goki/gi/giv/textbuf"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/spell"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// SpellView is a widget that displays results of spell check
type SpellView struct {
	gi.Layout

	// parent gide project
	Gide Gide `json:"-" xml:"-" copy:"-" desc:"parent gide project"`

	// textview that we're spell-checking
	Text *TextView `json:"-" xml:"-" copy:"-" desc:"textview that we're spell-checking"`

	// current spelling errors
	Errs lex.Line `desc:"current spelling errors"`

	// current line in text we're on
	CurLn int `desc:"current line in text we're on"`

	// current index in Errs we're on
	CurIdx int `desc:"current index in Errs we're on"`

	// current unknown lex token
	UnkLex lex.Lex `desc:"current unknown lex token"`

	// current unknown word
	UnkWord string `desc:"current unknown word"`

	// a list of suggestions from spell checker
	Suggest []string `desc:"a list of suggestions from spell checker"`

	// last user action (ignore, change, learn)
	LastAction *gi.Action `desc:"last user action (ignore, change, learn)"`
}

var KiT_SpellView = kit.Types.AddType(&SpellView{}, SpellViewProps)

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
func (sv *SpellView) Config(ge Gide, atv *TextView) {
	sv.Gide = ge
	sv.Text = atv
	sv.CurLn = 0
	sv.CurIdx = 0
	sv.Errs = nil
	sv.Lay = gi.LayoutVert
	sv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := kit.Config{}
	config.Add(gi.KiT_ToolBar, "spellbar")
	config.Add(gi.KiT_ToolBar, "unknownbar")
	config.Add(gi.KiT_ToolBar, "changebar")
	config.Add(giv.KiT_SliceView, "suggest")
	mods, updt := sv.ConfigChildren(config)
	if !mods {
		updt = sv.UpdateStart()
	}
	sv.ConfigToolbar()
	sv.UpdateEnd(updt)
	gi.InitSpell()
	sv.CheckNext()
}

// SpellBar returns the spell toolbar
func (sv *SpellView) SpellBar() *gi.ToolBar {
	return sv.ChildByName("spellbar", 0).(*gi.ToolBar)
}

// UnknownBar returns the toolbar that displays the unknown word
func (sv *SpellView) UnknownBar() *gi.ToolBar {
	return sv.ChildByName("unknownbar", 0).(*gi.ToolBar)
}

// ChangeBar returns the suggest toolbar
func (sv *SpellView) ChangeBar() *gi.ToolBar {
	return sv.ChildByName("changebar", 0).(*gi.ToolBar)
}

// ChangeAct returns the spell change action from toolbar
func (sv *SpellView) ChangeAct() *gi.Action {
	return sv.ChangeBar().ChildByName("change", 3).(*gi.Action)
}

// ChangeAllAct returns the spell change action from toolbar
func (sv *SpellView) ChangeAllAct() *gi.Action {
	return sv.ChangeBar().ChildByName("change-all", 3).(*gi.Action)
}

// SkipAct returns the skip action from toolbar
func (sv *SpellView) SkipAct() *gi.Action {
	return sv.UnknownBar().ChildByName("skip", 3).(*gi.Action)
}

// IgnoreAct returns the ignore action from toolbar
func (sv *SpellView) IgnoreAct() *gi.Action {
	return sv.UnknownBar().ChildByName("ignore", 3).(*gi.Action)
}

// LearnAct returns the learn action from toolbar
func (sv *SpellView) LearnAct() *gi.Action {
	return sv.UnknownBar().ChildByName("learn", 3).(*gi.Action)
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
	if spbar.HasChildren() {
		return
	}
	spbar.SetStretchMaxWidth()

	unknbar := sv.UnknownBar()
	if unknbar.HasChildren() {
		return
	}
	unknbar.SetStretchMaxWidth()

	chgbar := sv.ChangeBar()
	if chgbar.HasChildren() {
		return
	}
	chgbar.SetStretchMaxWidth()

	// spell toolbar
	spbar.AddAction(gi.ActOpts{Label: "Check Current File", Tooltip: "spell check the current file"},
		sv.This(), func(recv, send ki.Ki, sig int64, data any) {
			svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
			svv.SpellAction()
		})

	train := spbar.AddAction(gi.ActOpts{Label: "Train", Tooltip: "add additional text to the training corpus"}, sv.This(), func(recv, send ki.Ki, sig int64, data any) {
		svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
		svv.TrainAction()
	})
	train.SetProp("horizontal-align", gist.AlignRight)

	// unknown toolbar

	unknown := unknbar.AddNewChild(gi.KiT_TextField, "unknown-str").(*gi.TextField)
	unknown.SetStretchMaxWidth()
	unknown.Tooltip = "Unknown word"
	unknown.TextFieldSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data any) {
	})
	tf := sv.UnknownText()
	if tf != nil {
		tf.SetInactive()
	}

	unknbar.AddAction(gi.ActOpts{Name: "skip", Label: "Skip"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data any) {
			svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
			svv.SkipAction()
		})

	unknbar.AddAction(gi.ActOpts{Name: "ignore", Label: "Ignore"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data any) {
			svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
			svv.IgnoreAction()
		})

	unknbar.AddAction(gi.ActOpts{Name: "learn", Label: "Learn"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data any) {
			svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
			svv.LearnAction()
		})

	// change toolbar
	changestr := chgbar.AddNewChild(gi.KiT_TextField, "change-str").(*gi.TextField)
	changestr.SetStretchMaxWidth()
	changestr.Tooltip = "This string will replace the unknown word in text"
	changestr.TextFieldSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data any) {
	})

	chgbar.AddAction(gi.ActOpts{Name: "change", Label: "Change", Tooltip: "change the unknown word to the selected suggestion"}, sv.This(), func(recv, send ki.Ki, sig int64, data any) {
		svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
		svv.ChangeAction()
	})

	chgbar.AddAction(gi.ActOpts{Name: "change-all", Label: "Change All", Tooltip: "change all instances of the unknown word in this document"}, sv.This(), func(recv, send ki.Ki, sig int64, data any) {
		svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
		svv.ChangeAllAction()
	})

	// suggest toolbar
	suggest := sv.SuggestView()
	sv.Suggest = []string{"                                              "}
	suggest.SetInactive()
	suggest.SetProp("index", false)
	suggest.SetSlice(&sv.Suggest)
	suggest.SetStretchMaxWidth()
	suggest.SetStretchMaxHeight()
	suggest.SliceViewSig.Connect(suggest, func(recv, send ki.Ki, sig int64, data any) {
		svv := recv.Embed(giv.KiT_SliceView).(*giv.SliceView)
		idx := svv.SelectedIdx
		if idx >= 0 && idx < len(sv.Suggest) {
			sv.AcceptSuggestion(sv.Suggest[svv.SelectedIdx])
		}
	})
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
		gi.PromptDialog(sv.Viewport, gi.DlgOpts{Title: "Spelling Check Complete", Prompt: fmt.Sprintf("End of file, spelling check complete")}, gi.AddOk, gi.NoCancel, nil, nil)
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
	sugview.SetFullReRender()
	sugview.Slice = &sv.Suggest
	sugview.Update()

	st := sv.UnkStartPos()
	en := sv.UnkEndPos()
	tv.UpdateStart()
	tv.Highlights = tv.Highlights[:0]
	tv.SetCursorShow(st)
	hr := textbuf.Region{Start: st, End: en}
	hr.TimeNow()
	tv.Highlights = append(tv.Highlights, hr)
	tv.UpdateEnd(true)
	if sv.LastAction == nil {
		sv.GrabFocus()
	} else {
		sv.LastAction.GrabFocus()
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
	tv.Buf.ReplaceText(st, en, st, ct.Text(), giv.EditSignal, giv.ReplaceNoMatchCase)
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
	vp := sv.Viewport
	giv.FileViewDialog(vp, "", ".txt", giv.DlgOpts{Title: "Select a Text File to Add to Corpus"}, nil,
		vp.Win, func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.DialogAccepted) {
				dlg, _ := send.(*gi.Dialog)
				filepath := giv.FileViewDialogValue(dlg)
				gi.AddToSpellModel(filepath)
			}
		})
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
	if tv == nil || tv.Buf == nil || tv.IsDeleted() || tv.IsDestroyed() {
		return
	}
	tv.ClearHighlights()
}

// SpellViewProps are style properties for SpellView
var SpellViewProps = ki.Props{
	"EnumType:Flag":    gi.KiT_NodeFlags,
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
}
