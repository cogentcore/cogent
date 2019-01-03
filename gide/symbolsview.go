// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/syms"
	"github.com/goki/pi/token"
)

// SymNode represents a language symbol -- the name of the node is
// the name of the symbol. Some symbols, e.g. type have children
type SymNode struct {
	ki.Node
	Symbol syms.Symbol `desc:"a string"`
	SRoot  *SymTree    `json:"-" xml:"-" desc:"root of the tree -- has global state"`
}

var KiT_SymNode = kit.Types.AddType(&SymNode{}, nil)

// SymbolsParams are parameters for structure view of file or package
type SymbolsParams struct {
}

// SymbolsView is a widget that displays results of a file or package parse
type SymbolsView struct {
	gi.Layout
	Gide     Gide          `json:"-" xml:"-" desc:"parent gide project"`
	Symbols  SymbolsParams `desc:"params for structure display"`
	SymsTree SymTree       `desc:"all the syms for the file or package in a tree"`
	Syms     []syms.Symbol `desc:"the slice of symbols to display"`
}

var KiT_SymbolsView = kit.Types.AddType(&SymbolsView{}, SymbolsViewProps)

// SymbolsAction runs a new parse with current params
func (sv *SymbolsView) SymbolsAction() {
	sv.Gide.ProjPrefs().Symbols = sv.Symbols
	sv.Gide.Symbols()
}

// SetSymbols sets the slice of symbols for the SymbolsView
func (sv *SymbolsView) SetSymbols(filesyms []syms.Symbol) {
	sv.Syms = filesyms
}

// Display appends the results of the parse to textview of the symbols tab
func (sv *SymbolsView) Display() {
	outlns := make([][]byte, 0, 100)
	outmus := make([][]byte, 0, 100) // markups
	lstr := ""
	mstr := ""
	var f syms.Symbol
	for i := range sv.Syms {
		sbStLn := len(outlns) // find buf start ln
		f = sv.Syms[i]
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
	sv.ConfigTree()

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
	config.Add(gi.KiT_Frame, "symbolstree")
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

// SymbolsBar returns the spell toolbar
func (sv *SymbolsView) SymbolsTree() *gi.Frame {
	tvi, ok := sv.ChildByName("symbolstree", 0)
	if !ok {
		return nil
	}
	return tvi.(*gi.Frame)
}

// ConfigToolbar adds toolbar.
func (sv *SymbolsView) ConfigToolbar() {
	svbar := sv.SymbolsBar()
	if svbar.HasChildren() {
		return
	}
	svbar.SetStretchMaxWidth()

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

// ConfigTree adds a treeview to the symbolsview
func (sv *SymbolsView) ConfigTree() {
	svtree := sv.SymbolsTree()
	svtree.SetStretchMaxWidth()
	sv.SymsTree.OpenTree(sv, sv.Syms)
	if !svtree.HasChildren() {
		svt := svtree.AddNewChild(giv.KiT_TreeView, "symtree").(*giv.TreeView)
		svt.SetRootNode(&sv.SymsTree)
		svt.TreeViewSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if data == nil {
				return
			}
			tvn, _ := data.(ki.Ki).Embed(giv.KiT_TreeView).(*giv.TreeView)
			//sve, _ := recv.Embed(KiT_SymbolsView).(*SymbolsView)
			if tvn.SrcNode.Ptr != nil {
				sn := tvn.SrcNode.Ptr.Embed(KiT_SymNode).(*SymNode)
				switch sig {
				case int64(giv.TreeViewSelected):
					sv.SelectSymbol(sn.Symbol)
					//sve.FileNodeSelected(fn, tvn)
					//case int64(giv.TreeViewOpened):
					//	sve.FileNodeOpened(fn, tvn)
					//case int64(giv.TreeViewClosed):
					//	sve.FileNodeClosed(fn, tvn)
				}
			}
		})
	}
}

func (sv *SymbolsView) SelectSymbol(ssym syms.Symbol) {
	ge := sv.Gide
	tv := ge.ActiveTextView()
	if tv == nil {
		return
	}
	tv.UpdateStart()
	tv.Highlights = tv.Highlights[:0]
	tr := giv.NewTextRegion(ssym.SelectReg.St.Ln, ssym.SelectReg.St.Ch, ssym.SelectReg.Ed.Ln, ssym.SelectReg.Ed.Ch)
	tv.Highlights = append(tv.Highlights, tr)
	tv.UpdateEnd(true)
	tv.RefreshIfNeeded()
	tv.SetCursorShow(tr.Start)
	tv.GrabFocus()
}

// SymbolsViewProps are style properties for SymbolsView
var SymbolsViewProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
}

// SymTree is the root of a tree representing symbols of a package or file
type SymTree struct {
	SymNode
	NodeType reflect.Type `view:"-" json:"-" qxml:"-" desc:"type of node to create -- defaults to giv.FileNode but can use custom node types"`
	View     *SymbolsView
}

var KiT_SymTree = kit.Types.AddType(&SymTree{}, SymTreeProps)

var SymTreeProps = ki.Props{}

// OpenTree opens a SymTree of symbols from a file or package parse
func (st *SymTree) OpenTree(view *SymbolsView, fsyms []syms.Symbol) {
	st.SRoot = st // we are our own root..
	if st.NodeType == nil {
		st.NodeType = KiT_SymNode
	}
	st.SRoot.View = view
	st.ReadSyms(fsyms)
}

// ReadSyms adds the symbols resulting from a file(s) parse into this SymNode
func (sn *SymNode) ReadSyms(fsyms []syms.Symbol) {
	sv := sn.SRoot.View
	if len(sv.Syms) > 0 {
		config := sn.ConfigOfSyms(fsyms)
		mods, updt := sn.ConfigChildren(config, false) // NOT unique names
		for i, snk := range sn.Kids {
			k := snk.Embed(KiT_SymNode).(*SymNode)
			k.SRoot = sn.SRoot
			k.Symbol = fsyms[i]
		}
		if mods {
			sn.UpdateEnd(updt)
		}
	}
}

// ConfigOfSyms returns a type-and-name list for configuring nodes based on
// files in the list of parsed symbols passed to function
func (sn *SymNode) ConfigOfSyms(fsyms []syms.Symbol) kit.TypeAndNameList {
	config1 := kit.TypeAndNameList{}
	typ := sn.SRoot.NodeType

	for i := range fsyms {
		config1.Add(typ, fsyms[i].Name)
	}
	return config1
}
