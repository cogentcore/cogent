// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/syms"
	"github.com/goki/pi/token"
)

// SymbolsParams are parameters for structure view of file or package
type SymbolsParams struct {
}

// SymbolsView is a widget that displays results of a file parse
type SymbolsView struct {
	gi.Layout
	Gide    Gide          `json:"-" xml:"-" desc:"parent gide project"`
	Symbols SymbolsParams `desc:"params for structure display"`
}

var KiT_SymbolsView = kit.Types.AddType(&SymbolsView{}, SymbolsViewProps)

// SymbolsAction runs a new parse with current params
func (sv *SymbolsView) SymbolsAction() {
	sv.Gide.ProjPrefs().Symbols = sv.Symbols
	sv.Gide.Symbols()
}

// Display appends the results of the parse to textview of the symbols tab
func (sv *SymbolsView) Display(funcs []syms.Symbol) {
	outlns := make([][]byte, 0, 100)
	outmus := make([][]byte, 0, 100) // markups
	lstr := ""
	mstr := ""
	var f syms.Symbol
	for i := range funcs {
		sbStLn := len(outlns) // find buf start ln
		f = funcs[i]
		ln := f.SelectReg.St.Ln + 1
		ch := f.SelectReg.St.Ch + 1
		ech := f.SelectReg.Ed.Ch + 1
		if f.Kind == token.NameFunction || f.Kind == token.NameMethod {
			d := f.Detail
			s1 := strings.SplitAfterN(d, "(", 2)
			s0 := strings.SplitAfterN(s1[1], ")", 2)
			sd := "(" + s0[0]
			if len(sd) == 0 {
				sd = "()"
			}
			lead := ""
			if f.Kind == token.NameMethod {
				lead = "\t"
			}
			lstr = fmt.Sprintf(`%s%v%v`, lead, f.Name, sd)
		} else {
			lstr = f.Name
		}
		outlns = append(outlns, []byte(lstr))
		mstr = fmt.Sprintf(`<a href="symbols:///%v#R%vL%vC%v-L%vC%v">%v</a>`, f.Filename, sbStLn, ln, ch, ln, ech, lstr)
		outmus = append(outmus, []byte(mstr))
		outlns = append(outlns, []byte(""))
		outmus = append(outmus, []byte(""))
	}
	ltxt := bytes.Join(outlns, []byte("\n"))
	mtxt := bytes.Join(outmus, []byte("\n"))
	sv.TextView().Buf.AppendTextMarkup(ltxt, mtxt, false, true) // no save undo, yes signal
}

// OpenSymbolsURL opens given symbols:/// url from Find
func (sv *SymbolsView) OpenSymbolsURL(ur string, ftv *giv.TextView) bool {
	ge := sv.Gide
	tv, reg, _, _, ok := ge.ParseOpenFindURL(ur, ftv)
	if !ok {
		return false
	}
	tv.UpdateStart()
	tv.Highlights = tv.Highlights[:0]
	tv.Highlights = append(tv.Highlights, reg)
	tv.UpdateEnd(true)
	tv.RefreshIfNeeded()
	tv.SetCursorShow(reg.Start)
	tv.GrabFocus()
	return true
}

//////////////////////////////////////////////////////////////////////////////////////
//    GUI config

// UpdateView updates view with current settings
func (sv *SymbolsView) UpdateView(ge Gide, sp SymbolsParams) {
	sv.Gide = ge
	sv.Symbols = sp
	_, updt := sv.StdSymbolsConfig()
	sv.ConfigToolbar()
	tvly := sv.TextViewLay()
	sv.Gide.ConfigOutputTextView(tvly)
	sv.UpdateEnd(updt)
}

// StdConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (sv *SymbolsView) StdConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "symbolsbar")
	config.Add(gi.KiT_Layout, "symbolstext")
	return config
}

// StdSymbolsConfig configures a standard setup of the overall layout -- returns
// mods, updt from ConfigChildren and does NOT call UpdateEnd
func (sv *SymbolsView) StdSymbolsConfig() (mods, updt bool) {
	sv.Lay = gi.LayoutVert
	sv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := sv.StdConfig()
	mods, updt = sv.ConfigChildren(config, false)
	return
}

// TextViewLay returns the symbols view TextView layout
func (sv *SymbolsView) TextViewLay() *gi.Layout {
	tvi, ok := sv.ChildByName("symbolstext", 1)
	if !ok {
		return nil
	}
	return tvi.(*gi.Layout)
}

// TextView returns the symbols parse results
func (sv *SymbolsView) TextView() *giv.TextView {
	tvly := sv.TextViewLay()
	if tvly == nil {
		return nil
	}
	tv := tvly.KnownChild(0).(*giv.TextView)
	return tv
}

// SymbolsBar returns the spell toolbar
func (sv *SymbolsView) SymbolsBar() *gi.ToolBar {
	tbi, ok := sv.ChildByName("symbolsbar", 0)
	if !ok {
		return nil
	}
	return tbi.(*gi.ToolBar)
}

// ConfigToolbar adds toolbar.
func (sv *SymbolsView) ConfigToolbar() {
	stbar := sv.SymbolsBar()
	if stbar.HasChildren() {
		return
	}
	stbar.SetStretchMaxWidth()

	//// symbols toolbar
	//pkg := stbar.AddNewChild(gi.KiT_Action, "package").(*gi.Action)
	//pkg.SetText("Package Symbols")
	//pkg.Tooltip = "show the symbols of the entire package"
	////check.ActionSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
	////	svv, _ := recv.Embed(KiT_SpellView).(*SymbolsView)
	//	//svv.PackageAction
	////})
	//
	//vars := stbar.AddNewChild(gi.KiT_Action, "vars").(*gi.Action)
	//vars.SetProp("horizontal-align", gi.AlignRight)
	//vars.SetText("Vars")
	//vars.Tooltip = "show variables as well as functions"
	////train.ActionSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
	////	svv, _ := recv.Embed(KiT_SpellView).(*SpellView)
	////	svv.VarAction()
	////})
	//
	//checkbox := stbar.AddNewChild(gi.KiT_CheckBox, "checkbox").(*gi.CheckBox)
	//checkbox.SetProp("horizontal-align", gi.AlignRight)
	//
}

// SymbolsViewProps are style properties for SymbolsView
var SymbolsViewProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
}
