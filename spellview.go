// Copyright (c"strings") 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"fmt"
	"strings"

	"github.com/goki/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// SpellParams
type SpellParams struct {
}

// SpellView is a widget that displays results of spell check
type SpellView struct {
	gi.Layout
	Gide         *Gide       `json:"-" xml:"-" desc:"parent gide project"`
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
	sv.Gide.Prefs.Spell = sv.Spell

	uf := sv.UnknownText()
	uf.SetText("")

	sf := sv.ChangeText()
	sf.SetText("")

	sv.Gide.Spell()
}

// OpenFindURL opens given spell:/// url from Find
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

// UpdateView updates view with current settings
func (sv *SpellView) UpdateView(ge *Gide, sp SpellParams) {
	sv.Gide = ge
	sv.Spell = sp
	_, updt := sv.StdSpellConfig()
	sv.ConfigToolbar()
	tvly := sv.TextViewLay()
	sv.Gide.ConfigOutputTextView(tvly)
	sv.UpdateEnd(updt)
}

// StdConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (sv *SpellView) StdConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "spellbar")
	config.Add(gi.KiT_ToolBar, "unknownbar")
	config.Add(gi.KiT_ToolBar, "changebar")
	config.Add(gi.KiT_ToolBar, "suggestbar")
	config.Add(gi.KiT_Layout, "spelltext")
	return config
}

// StdSpellConfig configures a standard setup of the overall layout -- returns
// mods, updt from ConfigChildren and does NOT call UpdateEnd
func (sv *SpellView) StdSpellConfig() (mods, updt bool) {
	sv.Lay = gi.LayoutVert
	sv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := sv.StdConfig()
	mods, updt = sv.ConfigChildren(config, false)
	return
}

// TextViewLay returns the spell check results TextView layout
func (sv *SpellView) TextViewLay() *gi.Layout {
	tvi, ok := sv.ChildByName("spelltext", 1)
	if !ok {
		return nil
	}
	return tvi.(*gi.Layout)
}

// SpellBar returns the spell toolbar
func (sv *SpellView) SpellBar() *gi.ToolBar {
	tbi, ok := sv.ChildByName("spellbar", 0)
	if !ok {
		return nil
	}
	return tbi.(*gi.ToolBar)
}

// UnknownBar returns the toolbar that displays the unknown word
func (sv *SpellView) UnknownBar() *gi.ToolBar {
	tbi, ok := sv.ChildByName("unknownbar", 0)
	if !ok {
		return nil
	}
	return tbi.(*gi.ToolBar)
}

// ChangeBar returns the suggest toolbar
func (sv *SpellView) ChangeBar() *gi.ToolBar {
	tbi, ok := sv.ChildByName("changebar", 0)
	if !ok {
		return nil
	}
	return tbi.(*gi.ToolBar)
}

// SuggestBar returns the suggest toolbar
func (sv *SpellView) SuggestBar() *gi.ToolBar {
	tbi, ok := sv.ChildByName("suggestbar", 0)
	if !ok {
		return nil
	}
	return tbi.(*gi.ToolBar)
}

// SpellNextAct returns the spell next action from toolbar
func (sv *SpellView) SpellNextAct() *gi.Action {
	tb := sv.UnknownBar()
	if tb == nil {
		return nil
	}
	tfi, ok := tb.ChildByName("next", 3)
	if !ok {
		return nil
	}
	return tfi.(*gi.Action)
}

// CheckAct returns the spell check action from toolbar
func (sv *SpellView) CheckAct() *gi.Action {
	tb := sv.UnknownBar()
	if tb == nil {
		return nil
	}
	tfi, ok := tb.ChildByName("check", 3)
	if !ok {
		return nil
	}
	return tfi.(*gi.Action)
}

// ChangeAct returns the spell change action from toolbar
func (sv *SpellView) ChangeAct() *gi.Action {
	tb := sv.UnknownBar()
	if tb == nil {
		return nil
	}
	tfi, ok := tb.ChildByName("change", 3)
	if !ok {
		return nil
	}
	return tfi.(*gi.Action)
}

// SkitpAct returns the skip action from toolbar
func (sv *SpellView) SkipAct() *gi.Action {
	tb := sv.UnknownBar()
	if tb == nil {
		return nil
	}
	tfi, ok := tb.ChildByName("skip", 3)
	if !ok {
		return nil
	}
	return tfi.(*gi.Action)
}

// IgnoreAct returns the ignore action from toolbar
func (sv *SpellView) IgnoreAct() *gi.Action {
	tb := sv.UnknownBar()
	if tb == nil {
		return nil
	}
	tfi, ok := tb.ChildByName("ignore", 3)
	if !ok {
		return nil
	}
	return tfi.(*gi.Action)
}

// LearnAct returns the learn action from toolbar
func (sv *SpellView) LearnAct() *gi.Action {
	tb := sv.UnknownBar()
	if tb == nil {
		return nil
	}
	tfi, ok := tb.ChildByName("learn", 3)
	if !ok {
		return nil
	}
	return tfi.(*gi.Action)
}

// TextView returns the spell check results TextView
func (sv *SpellView) TextView() *giv.TextView {
	tvly := sv.TextViewLay()
	if tvly == nil {
		return nil
	}
	tv := tvly.KnownChild(0).(*giv.TextView)
	return tv
}

// UnknownText returns the unknown word textfield from toolbar
func (sv *SpellView) UnknownText() *gi.TextField {
	tb := sv.UnknownBar()
	if tb == nil {
		return nil
	}
	tfi, ok := tb.ChildByName("unknown-str", 1)
	if !ok {
		return nil
	}
	return tfi.(*gi.TextField)
}

// ChangeText returns the unknown word textfield from toolbar
func (sv *SpellView) ChangeText() *gi.TextField {
	tb := sv.ChangeBar()
	if tb == nil {
		return nil
	}
	tfi, ok := tb.ChildByName("change-str", 1)
	if !ok {
		return nil
	}
	return tfi.(*gi.TextField)
}

// SuggestView returns the view for the list of suggestions
func (sv *SpellView) SuggestView() *giv.SliceView {
	sb := sv.SuggestBar()
	if sb == nil {
		return nil
	}
	slv, ok := sb.ChildByName("suggestions", 1)
	if !ok {
		return nil
	}
	return slv.(*giv.SliceView)
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
	check := spbar.AddNewChild(gi.KiT_Action, "check").(*gi.Action)
	check.SetText("Check Current File")
	check.Tooltip = "spell check the current file"
	check.ActionSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
		svv.SpellAction()
	})

	train := spbar.AddNewChild(gi.KiT_Action, "train").(*gi.Action)
	train.SetProp("horizontal-align", gi.AlignRight)
	train.SetText("Train")
	train.Tooltip = "add additional text to the training corpus"
	train.ActionSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
		svv.TrainAction()
	})

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

	skip := unknbar.AddNewChild(gi.KiT_Action, "skip").(*gi.Action)
	skip.SetText("Skip")
	skip.ActionSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
		svv.SkipAction()
	})

	ignore := unknbar.AddNewChild(gi.KiT_Action, "ignore").(*gi.Action)
	ignore.SetText("Ignore")
	ignore.ActionSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
		svv.IgnoreAction()
	})

	learn := unknbar.AddNewChild(gi.KiT_Action, "learn").(*gi.Action)
	learn.SetText("Learn")
	learn.ActionSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
		svv.LearnAction()
	})

	// change toolbar
	changestr := chgbar.AddNewChild(gi.KiT_TextField, "change-str").(*gi.TextField)
	changestr.SetStretchMaxWidth()
	changestr.Tooltip = "This string will replace the unknown word in text"
	changestr.TextFieldSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
	})

	change := chgbar.AddNewChild(gi.KiT_Action, "change").(*gi.Action)
	change.SetText("Change")
	change.Tooltip = "change the unknown word to the selected suggestion"
	change.ActionSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
		svv.ChangeAction()
	})

	// suggest toolbar
	suggest := sugbar.AddNewChild(giv.KiT_SliceView, "suggestions").(*giv.SliceView)
	suggest.SetInactive()
	suggest.SetProp("index", false)
	suggest.SetSlice(&sv.Suggestions, nil)
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

// SetUnknownAndSuggest
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
	sugview.UpdateFromSlice()
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
	gi.IgnoreWord(sv.Unknown.Word)
	sv.LastAction = sv.IgnoreAct()
	sv.CheckNext()
}

// LearnAction will add the current unknown word to corpus
// and call CheckNext
func (sv *SpellView) LearnAction() {
	new := strings.ToLower(sv.Unknown.Word)
	gi.LearnWord(new)
	sv.LastAction = sv.LearnAct()
	sv.CheckNext()
}

func (sv *SpellView) AcceptSuggestion(s string) {
	ct := sv.ChangeText()
	ct.SetText(s)
	sv.ChangeAction()
}

var SpellViewProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
}
