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
	//sv.ConfigToolbar()
	tvly := sv.TextViewLay()
	sv.Gide.ConfigOutputTextView(tvly)
	sv.UpdateEnd(updt)
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

// StdConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (sv *SpellView) StdConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	//config.Add(gi.KiT_ToolBar, "findbar")
	//config.Add(gi.KiT_ToolBar, "replbar")
	config.Add(gi.KiT_Layout, "spelltext")
	return config
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

var SpellViewProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
}
