// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"bytes"
	"fmt"
	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/syms"
)

// StructureParams are parameters for structure view of file or package
type StructureParams struct {
}

// StructureView is a widget that displays results of a file parse
type StructureView struct {
	gi.Layout
	Gide      Gide            `json:"-" xml:"-" desc:"parent gide project"`
	Structure StructureParams `desc:"params for structure display"`
}

var KiT_StructureView = kit.Types.AddType(&StructureView{}, StructureViewProps)

// StructureAction runs a new parse with current params
func (sv *StructureView) StructureAction() {
	sv.Gide.ProjPrefs().Structure = sv.Structure
	sv.Gide.Structure()
}

// Display appends the results of the parse to textview of the structure tab
func (sv *StructureView) Display(funcs []syms.Symbol) {
	outlns := make([][]byte, 0, 100)
	outmus := make([][]byte, 0, 100) // markups
	lstr := ""
	mstr := ""
	var f syms.Symbol
	for i := range funcs {
		sbStLn := len(outlns) // find buf start ln
		f = funcs[i]
		ln := f.Region.St.Ln + 1
		ch := f.Region.St.Ch + 1
		ech := f.Region.Ed.Ch + 1
		d := f.Detail
		if len(d) == 0 {
			d = "()"
		}
		lstr = fmt.Sprintf(`%v%v`, f.Name, d)
		outlns = append(outlns, []byte(lstr))
		mstr = fmt.Sprintf(`	<a href="structure:///%v#R%vL%vC%v-L%vC%v">%v</a>`, f.Filename, sbStLn, ln, ch, ln, ech, lstr)
		outmus = append(outmus, []byte(mstr))
		outlns = append(outlns, []byte(""))
		outmus = append(outmus, []byte(""))
	}
	ltxt := bytes.Join(outlns, []byte("\n"))
	mtxt := bytes.Join(outmus, []byte("\n"))
	sv.TextView().Buf.AppendTextMarkup(ltxt, mtxt, false, true) // no save undo, yes signal
}

// OpenStructureURL opens given structure:/// url from Find
func (sv *StructureView) OpenStructureURL(ur string, ftv *giv.TextView) bool {
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
func (sv *StructureView) UpdateView(ge Gide, sp StructureParams) {
	sv.Gide = ge
	sv.Structure = sp
	_, updt := sv.StdStructureConfig()
	sv.ConfigToolbar()
	tvly := sv.TextViewLay()
	sv.Gide.ConfigOutputTextView(tvly)
	sv.UpdateEnd(updt)
}

// StdConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (sv *StructureView) StdConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "structurebar")
	config.Add(gi.KiT_Layout, "structuretext")
	return config
}

// StdStructureConfig configures a standard setup of the overall layout -- returns
// mods, updt from ConfigChildren and does NOT call UpdateEnd
func (sv *StructureView) StdStructureConfig() (mods, updt bool) {
	sv.Lay = gi.LayoutVert
	sv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := sv.StdConfig()
	mods, updt = sv.ConfigChildren(config, false)
	return
}

// ConfigToolbar adds toolbar.
func (sv *StructureView) ConfigToolbar() {
}

// TextViewLay returns the structure view TextView layout
func (sv *StructureView) TextViewLay() *gi.Layout {
	tvi, ok := sv.ChildByName("structuretext", 1)
	if !ok {
		return nil
	}
	return tvi.(*gi.Layout)
}

// TextView returns the structure parse results
func (sv *StructureView) TextView() *giv.TextView {
	tvly := sv.TextViewLay()
	if tvly == nil {
		return nil
	}
	tv := tvly.KnownChild(0).(*giv.TextView)
	return tv
}

// StructureViewProps are style properties for StructureView
var StructureViewProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
}
