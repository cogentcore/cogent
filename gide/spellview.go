// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"fmt"
	"strings"

	"github.com/goki/gi/spell"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// SpellParams are parameters for spell check and correction
type SpellParams struct {
}

// SpellView is a widget that displays results of spell check
type SpellView struct {
	gi.Layout
	Gide         Gide        `json:"-" xml:"-" desc:"parent gide project"`
	Spell        SpellParams `desc:"params for spelling"`
	Unknown      gi.TextWord `desc:"current unknown/misspelled word"`
	Suggestions  []string    `desc:"a list of suggestions from spell checker"`
	ChangeOffset int         `desc:"compensation for change word length different than original word"`
	PreviousLine int         `desc:"line of previous unknown word"`
	CurrentLine  int         `desc:"line of current unknown word"`
	LastAction   *gi.Action  `desc:"last user action (ignore, change, learn)"`
}

var KiT_SpellView = kit.Types.AddType(&SpellView{}, SpellViewProps)

// SpellAction runs a new spell check with current params
func (sv *SpellView) SpellAction() {
	sv.Gide.ProjPrefs().Spell = sv.Spell

	uf := sv.UnknownText()
	uf.SetText("")

	sf := sv.ChangeText()
	sf.SetText("")

	sv.Gide.Spell()
}

// OpenSpellURL opens given spell:/// url from Find
func (sv *SpellView) OpenSpellURL(ur string, ftv *giv.TextView) bool {
	ge := sv.Gide
	tv, reg, _, _, ok := ge.ParseOpenFindURL(ur, ftv)
	if !ok {
		return false
	}
	tv.RefreshIfNeeded()
	tv.SetCursorShow(reg.Start)
	return true
}

//////////////////////////////////////////////////////////////////////////////////////
//    GUI config

// Config configures the view
func (sv *SpellView) Config(ge Gide, sp SpellParams) {
	sv.Gide = ge
	sv.Spell = sp
	sv.Lay = gi.LayoutVert
	sv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "spellbar")
	config.Add(gi.KiT_ToolBar, "unknownbar")
	config.Add(gi.KiT_ToolBar, "changebar")
	config.Add(gi.KiT_ToolBar, "suggestbar")
	config.Add(gi.KiT_Layout, "spelltext")
	mods, updt := sv.ConfigChildren(config, false)
	if !mods {
		updt = sv.UpdateStart()
	}
	sv.ConfigToolbar()
	tvly := sv.TextViewLay()
	sv.Gide.ConfigOutputTextView(tvly)
	sv.UpdateEnd(updt)
}

// TextViewLay returns the spell check results TextView layout
func (sv *SpellView) TextViewLay() *gi.Layout {
	return sv.ChildByName("spelltext", 1).(*gi.Layout)
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

// SuggestBar returns the suggest toolbar
func (sv *SpellView) SuggestBar() *gi.ToolBar {
	return sv.ChildByName("suggestbar", 0).(*gi.ToolBar)
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

// TextView returns the spell check results TextView
func (sv *SpellView) TextView() *giv.TextView {
	return sv.TextViewLay().Child(0).Embed(giv.KiT_TextView).(*giv.TextView)
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
	return sv.SuggestBar().ChildByName("suggestions", 1).(*giv.SliceView)
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

	sugbar := sv.SuggestBar()
	if sugbar.HasChildren() {
		return
	}
	sugbar.SetStretchMaxWidth()
	sugbar.SetMinPrefHeight(units.NewValue(50, units.Ch))

	// spell toolbar
	spbar.AddAction(gi.ActOpts{Label: "Check Current File", Tooltip: "spell check the current file"},
		sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
			svv.SpellAction()
		})

	train := spbar.AddAction(gi.ActOpts{Label: "Train", Tooltip: "add additional text to the training corpus"}, sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
		svv.TrainAction()
	})
	train.SetProp("horizontal-align", gi.AlignRight)

	// unknown toolbar

	unknown := unknbar.AddNewChild(gi.KiT_TextField, "unknown-str").(*gi.TextField)
	unknown.SetStretchMaxWidth()
	unknown.Tooltip = "Unknown word"
	unknown.TextFieldSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
	})
	tf := sv.UnknownText()
	if tf != nil {
		tf.SetInactive()
	}

	unknbar.AddAction(gi.ActOpts{Name: "skip", Label: "Skip"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
			svv.SkipAction()
		})

	unknbar.AddAction(gi.ActOpts{Name: "ignore", Label: "Ignore"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
			svv.IgnoreAction()
		})

	unknbar.AddAction(gi.ActOpts{Name: "learn", Label: "Learn"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
			svv.LearnAction()
		})

	// change toolbar
	changestr := chgbar.AddNewChild(gi.KiT_TextField, "change-str").(*gi.TextField)
	changestr.SetStretchMaxWidth()
	changestr.Tooltip = "This string will replace the unknown word in text"
	changestr.TextFieldSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
	})

	chgbar.AddAction(gi.ActOpts{Name: "change", Label: "Change", Tooltip: "change the unknown word to the selected suggestion"}, sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
		svv.ChangeAction()
	})

	chgbar.AddAction(gi.ActOpts{Name: "change-all", Label: "Change All", Tooltip: "change all instances of the unknown word in this document"}, sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
		svv.ChangeAllAction()
	})

	// suggest toolbar
	suggest := sugbar.AddNewChild(giv.KiT_SliceView, "suggestions").(*giv.SliceView)
	suggest.SetInactive()
	suggest.SetProp("index", false)
	suggest.SetSlice(&sv.Suggestions)
	suggest.SetStretchMaxWidth()
	suggest.SetStretchMaxHeight()
	suggest.SliceViewSig.Connect(suggest, func(recv, send ki.Ki, sig int64, data interface{}) {
		svv := recv.Embed(giv.KiT_SliceView).(*giv.SliceView)
		idx := svv.SelectedIdx
		if idx >= 0 && idx < len(sv.Suggestions) {
			sv.AcceptSuggestion(sv.Suggestions[svv.SelectedIdx])
		}
	})
}

// CheckNext will find the next misspelled/unknown word and get suggestions for replacing it
func (sv *SpellView) CheckNext() {
	tw, suggests, _ := gi.NextUnknownWord()
	if tw.Word == "" {
		gi.PromptDialog(sv.Viewport, gi.DlgOpts{Title: "Spelling Check Complete", Prompt: fmt.Sprintf("End of file, spelling check complete")}, true, false, nil, nil)
		return
	}
	sv.SetUnknownAndSuggest(tw, suggests)
}

// SetUnknownAndSuggest sets the textfield unknown with the best suggestion
// and fills the suggestions with all suggestions up to the max
func (sv *SpellView) SetUnknownAndSuggest(unknown gi.TextWord, suggests []string) {
	uf := sv.UnknownText()
	uf.SetText(unknown.Word)
	sv.Unknown = unknown
	sv.Suggestions = suggests
	sv.PreviousLine = sv.CurrentLine
	sv.CurrentLine = unknown.Line

	cf := sv.ChangeText()
	if len(suggests) == 0 {
		cf.SetText("")
	} else {
		cf.SetText(suggests[0])
	}
	sugview := sv.SuggestView()
	sugview.Config()
	tv := sv.Gide.ActiveTextView()
	st := sv.UnknownStartPos()
	en := sv.UnknownEndPos()
	tv.UpdateStart()
	tv.Highlights = tv.Highlights[:0]
	tv.SetCursorShow(st)
	tv.Highlights = append(tv.Highlights, giv.TextRegion{Start: st, End: en})
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
	tv := sv.Gide.ActiveTextView()
	if tv == nil {
		return
	}
	st := sv.UnknownStartPos()
	en := sv.UnknownEndPos()
	tbe := tv.Buf.DeleteText(st, en, true, true)
	ct := sv.ChangeText()
	bs := []byte(string(ct.EditTxt))
	tv.Buf.InsertText(tbe.Reg.Start, bs, true, true)
	sv.ChangeOffset = sv.ChangeOffset + len(bs) - (en.Ch - st.Ch) // new length - old length
	sv.LastAction = sv.ChangeAct()
	sv.CheckNext()
}

// ChangeAllAction replaces the known word with the selected suggested word
// and call CheckNextAction
func (sv *SpellView) ChangeAllAction() {
	tv := sv.Gide.ActiveTextView()
	if tv == nil {
		return
	}
	tv.QReplaceStart(sv.Unknown.Word, sv.ChangeText().Txt)
	tv.QReplaceReplaceAll(0)
	sv.LastAction = sv.ChangeAllAct()
	sv.CheckNext()
}

// TrainAction allows you to train on additional text files and also to rebuild the spell model
func (sv *SpellView) TrainAction() {
	vp := sv.Viewport
	giv.FileViewDialog(vp, "", ".txt", giv.DlgOpts{Title: "Select a Text File to Add to Corpus"}, nil,
		vp.Win, func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				dlg, _ := send.(*gi.Dialog)
				filepath := giv.FileViewDialogValue(dlg)
				gi.AddToSpellModel(filepath)
			}
		})
}

// UnknownStartPos returns the start position of the current unknown word adjusted for any prior replacement text
func (sv *SpellView) UnknownStartPos() giv.TextPos {
	pos := giv.TextPos{Ln: sv.Unknown.Line, Ch: sv.Unknown.StartPos}
	pos = sv.AdjustTextPos(pos)
	return pos
}

// UnknownEndPos returns the end position of the current unknown word adjusted for any prior replacement text
func (sv *SpellView) UnknownEndPos() giv.TextPos {
	pos := giv.TextPos{Ln: sv.Unknown.Line, Ch: sv.Unknown.EndPos}
	pos = sv.AdjustTextPos(pos)
	return pos
}

// AdjustTextPos adjust the character position to compensate for replacement text being different length than original text
func (sv *SpellView) AdjustTextPos(tp giv.TextPos) giv.TextPos {
	if sv.CurrentLine != sv.PreviousLine {
		sv.ChangeOffset = 0
		return tp
	}
	tp.Ch += sv.ChangeOffset
	return tp
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
	spell.IgnoreWord(sv.Unknown.Word)
	sv.LastAction = sv.IgnoreAct()
	sv.CheckNext()
}

// LearnAction will add the current unknown word to corpus
// and call CheckNext
func (sv *SpellView) LearnAction() {
	nw := strings.ToLower(sv.Unknown.Word)
	gi.LearnWord(nw)
	sv.LastAction = sv.LearnAct()
	sv.CheckNext()
}

// AcceptSuggestion replaces the misspelled word with the word in the ChangeText field
func (sv *SpellView) AcceptSuggestion(s string) {
	ct := sv.ChangeText()
	ct.SetText(s)
	sv.ChangeAction()
}

// SpellViewProps are style properties for SpellView
var SpellViewProps = ki.Props{
	"EnumType:Flag":    gi.KiT_NodeFlags,
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
}
