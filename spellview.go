// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"github.com/goki/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// SpellParams
type SpellParams struct {
}

// SpellView is a widget that displays results of spell check
type SpellView struct {
	gi.Layout
	Gide  *Gide       `json:"-" xml:"-" desc:"parent gide project"`
	Spell SpellParams `desc:"params for spelling"`
}

var KiT_SpellView = kit.Types.AddType(&SpellView{}, SpellViewProps)

// SpellAction runs a new find with current params
func (sv *SpellView) SpellAction() {
	sv.Gide.Prefs.Spell = sv.Spell
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

// SpellCheckAct returns the spell check action in toolbar
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

// SpellIgnoreAct returns the ignore action in toolbar
func (sv *SpellView) SpellIgnoreAct() *gi.Action {
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

// SpellLearnAct returns the learn action in toolbar
func (sv *SpellView) SpellLearnAct() *gi.Action {
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

// FindSuggestText returns the unknown word textfield in toolbar
func (sv *SpellView) FindSuggestText() *gi.TextField {
	tb := sv.UnknownBar()
	if tb == nil {
		return nil
	}
	tfi, ok := tb.ChildByName("suggest-str", 1)
	if !ok {
		return nil
	}
	return tfi.(*gi.TextField)
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

	sugbar := sv.SuggestBar()
	if sugbar.HasChildren() {
		return
	}
	sugbar.SetStretchMaxWidth()

	//lbl := spbar.AddNewChild(gi.KiT_Label, "check-lbl").(*gi.Label)
	//lbl.SetStretchMaxWidth()
	//lbl.SetText("Spell check current file")

	check := spbar.AddNewChild(gi.KiT_Action, "check").(*gi.Action)
	check.SetText("Check Current File")
	check.Tooltip = "spell check the current file"
	check.ActionSig.Connect(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
		svv.ChangeAction()
	})

	unknown := unknbar.AddNewChild(gi.KiT_TextField, "unknown-str").(*gi.TextField)
	unknown.SetStretchMaxWidth()
	unknown.Tooltip = "Word not found in dictionary"
	unknown.TextFieldSig.Connect(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		//if sig == int64(gi.TextFieldDone) {
		//	svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
		//	tf := send.(*gi.TextField)
		//	svv.Spell.Check = tf.Text()
		//}
	})

	suggest := sugbar.AddNewChild(gi.KiT_TextField, "suggest-str").(*gi.TextField)
	suggest.SetStretchMaxWidth()
	suggest.Tooltip = "Suggestion"
	suggest.TextFieldSig.Connect(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		//if sig == int64(gi.TextFieldDone) {
		//	svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
		//	tf := send.(*gi.TextField)
		//	svv.Find.Replace = tf.Text()
		//}
	})
	tf := sv.FindSuggestText()
	if tf != nil {
		tf.SetInactive()
	}

	change := sugbar.AddNewChild(gi.KiT_Action, "change").(*gi.Action)
	change.SetText("Change")
	change.Tooltip = "change the unknown word to the selected suggestion"
	change.ActionSig.Connect(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
		svv.ChangeAction()
	})

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
}

// ChangeAction replaces the known word with the selected suggested word
func (sv *SpellView) CheckAction() {
}

// ChangeAction replaces the known word with the selected suggested word
func (sv *SpellView) ChangeAction() {
	// todo: borrow code from findview replace action
}

func (sv *SpellView) IgnoreAction() {

}

func (sv *SpellView) LearnAction() {

}

var SpellViewProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
}
