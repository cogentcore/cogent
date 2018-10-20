// Copyright (c"strings") 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"fmt"
	"github.com/goki/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/spell"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
	"strings"
)

// SpellParams
type SpellParams struct {
}

// SpellView is a widget that displays results of spell check
type SpellView struct {
	gi.Layout
	Gide         *Gide          `json:"-" xml:"-" desc:"parent gide project"`
	Spell        SpellParams    `desc:"params for spelling"`
	Unknown      spell.TextWord `desc:"current unknown/misspelled word"`
	Suggestions  []string       `desc:"a list of suggestions from spell checker"`
	ChangeOffset int            `desc:"compensation for change word length different than original word"`
	PreviousLine int            `desc:"line of previous unknown word"`
	CurrentLine  int            `desc:"line of current unknown word"`
}

var KiT_SpellView = kit.Types.AddType(&SpellView{}, SpellViewProps)

// SpellAction runs a new find with current params
func (sv *SpellView) SpellAction() {
	sv.Gide.Prefs.Spell = sv.Spell

	uf := sv.FindUnknownText()
	uf.SetText("")

	sf := sv.FindChangeText()
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

// SpellNextAct returns the spell next action in toolbar
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

// FindCheckAct returns the spell check action in toolbar
func (sv *SpellView) FindCheckAct() *gi.Action {
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

// FindChangeAct returns the spell change action in toolbar
func (sv *SpellView) FindChangeAct() *gi.Action {
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

// FindIgnoreAct returns the ignore action in toolbar
func (sv *SpellView) FindIgnoreAct() *gi.Action {
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

// FindLearnAct returns the learn action in toolbar
func (sv *SpellView) FindLearnAct() *gi.Action {
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

// FindUnknownText returns the unknown word textfield in toolbar
func (sv *SpellView) FindUnknownText() *gi.TextField {
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

// FindChangeText returns the unknown word textfield in toolbar
func (sv *SpellView) FindChangeText() *gi.TextField {
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

// FindSuggestView returns the view for the list of suggestions
func (sv *SpellView) FindSuggestView() *giv.SliceView {
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

	// spell toolbar
	check := spbar.AddNewChild(gi.KiT_Action, "check").(*gi.Action)
	check.SetText("Check Current File")
	check.Tooltip = "spell check the current file"
	check.ActionSig.Connect(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
		svv.SpellAction()
	})

	// unknown toolbar
	unknown := unknbar.AddNewChild(gi.KiT_TextField, "unknown-str").(*gi.TextField)
	unknown.SetStretchMaxWidth()
	unknown.Tooltip = "Unknown word"
	unknown.TextFieldSig.Connect(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
	})
	tf := sv.FindUnknownText()
	if tf != nil {
		tf.SetInactive()
	}

	ignore := unknbar.AddNewChild(gi.KiT_Action, "ignore").(*gi.Action)
	ignore.SetText("Ignore")
	ignore.Tooltip = "ignore all instances of the unknown word"
	ignore.ActionSig.Connect(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
		svv.IgnoreAction()
	})

	learn := unknbar.AddNewChild(gi.KiT_Action, "learn").(*gi.Action)
	learn.SetText("Learn")
	learn.Tooltip = "add the unknown word to my personal dictionary"
	learn.ActionSig.Connect(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
		svv.LearnAction()
	})

	// change toolbar
	changestr := chgbar.AddNewChild(gi.KiT_TextField, "change-str").(*gi.TextField)
	changestr.SetStretchMaxWidth()
	changestr.Tooltip = "This string will replace the unknown word in text"
	changestr.TextFieldSig.Connect(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
	})

	change := chgbar.AddNewChild(gi.KiT_Action, "change").(*gi.Action)
	change.SetText("Change")
	change.Tooltip = "change the unknown word to the selected suggestion"
	change.ActionSig.Connect(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
		svv.ChangeAction()
	})

	// suggest toolbar
	suggest := sugbar.AddNewChild(giv.KiT_SliceView, "suggestions").(*giv.SliceView)
	suggest.SetSlice(&sv.Suggestions, nil)
	suggest.SetStretchMaxWidth()
	suggest.SetStretchMaxHeight()

}

// CheckNext will find the next misspelled/unknown word
func (sv *SpellView) CheckNext() {
	tw, suggests := spell.NextUnknownWord()
	if tw.Word == "" {
		gi.PromptDialog(sv.Viewport, gi.DlgOpts{Title: "Spelling Check Complete", Prompt: fmt.Sprintf("End of file, spelling check complete")}, true, false, nil, nil)
		return
	}

	if tw.Word != "" {
		sv.SetUnknownAndSuggest(tw, suggests)
	}
}

// SetUnknownAndSuggest
func (sv *SpellView) SetUnknownAndSuggest(unknown spell.TextWord, suggests []string) {
	uf := sv.FindUnknownText()
	uf.SetText(unknown.Word)
	sv.Unknown = unknown
	sv.Suggestions = suggests
	sv.PreviousLine = sv.CurrentLine
	sv.CurrentLine = unknown.Line

	cf := sv.FindChangeText()
	if len(suggests) == 0 {
		cf.SetText("")
	} else {
		cf.SetText(suggests[0])
		sugview := sv.FindSuggestView()
		sugview.IsArray = true
		sugview.UpdateFromSlice()
	}

}

// ChangeAction replaces the known word with the selected suggested word
// and call CheckNextAction
func (sv *SpellView) ChangeAction() {
	tv := sv.Gide.ActiveTextView()
	if tv == nil {
		return
	}
	if sv.CurrentLine != sv.PreviousLine {
		sv.ChangeOffset = 0
	}

	st := giv.TextPos{Ln: sv.Unknown.Line, Ch: sv.Unknown.StartPos + sv.ChangeOffset}
	en := giv.TextPos{Ln: sv.Unknown.Line, Ch: sv.Unknown.EndPos + sv.ChangeOffset}
	tbe := tv.Buf.DeleteText(st, en, true, true)
	ct := sv.FindChangeText()
	bs := []byte(string(ct.EditTxt))
	tv.Buf.InsertText(tbe.Reg.Start, bs, true, true)
	sv.ChangeOffset = len(bs) - (en.Ch - st.Ch) // new length - old length
	sv.CheckNext()
}

// IgnoreAction will skip the current misspelled/unknown word
// and call CheckNextAction
func (sv *SpellView) IgnoreAction() {
	sv.CheckNext()
}

// LearnAction will add the current unknown word to corpus
// and call CheckNext
func (sv *SpellView) LearnAction() {
	new := strings.ToLower(sv.Unknown.Word)
	spell.LearnWord(new)
	sv.CheckNext()
}

var SpellViewProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
}
