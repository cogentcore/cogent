// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/syms"
	"github.com/goki/pi/token"
	"reflect"
	"sort"
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
	Gide      Gide          `json:"-" xml:"-" desc:"parent gide project"`
	SymParams SymbolsParams `desc:"params for structure display"`
	SymsTree  SymTree       `desc:"all the syms for the file or package in a tree"`
}

var KiT_SymbolsView = kit.Types.AddType(&SymbolsView{}, SymbolsViewProps)

// SymbolsAction runs a new parse with current params
func (sv *SymbolsView) SymbolsAction() {
	sv.Gide.ProjPrefs().Symbols = sv.SymParams
	sv.Gide.Symbols()
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
	sv.SymParams = sp
	_, updt := sv.StdSymbolsConfig()
	sv.ConfigToolbar()
	sv.ConfigTree()
	sv.UpdateEnd(updt)
}

// StdConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (sv *SymbolsView) StdConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "symbolsbar")
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
	sv.SymsTree.OpenTree(sv)
	//if !svtree.HasChildren() {
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
	//}
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
func (st *SymTree) OpenTree(view *SymbolsView) {
	ge := view.Gide
	tv := ge.ActiveTextView()
	if tv == nil || tv.Buf == nil {
		return
	}

	fs := &tv.Buf.PiState

	st.SRoot = st // we are our own root..
	if st.NodeType == nil {
		st.NodeType = KiT_SymNode
	}
	st.SRoot.View = view

	funcs := []syms.Symbol{} // collect and add functions (no receiver) to end
	for _, v := range fs.Syms {
		if v.Kind != token.NamePackage { // note: package symbol filename won't always corresp.
			continue
		}
		for _, w := range v.Children {
			if w.Filename != fs.Src.Filename {
				continue
			}
			switch w.Kind {
			case token.NameFunction:
				funcs = append(funcs, *w)
			case token.NameStruct, token.NameMap, token.NameArray:
				kid := st.AddNewChild(nil, w.Name)
				kn := kid.Embed(KiT_SymNode).(*SymNode)
				kn.SRoot = st.SRoot
				kn.Symbol = *w
				var temp []syms.Symbol
				for _, x := range w.Children {
					if x.Kind == token.NameMethod {
						temp = append(temp, *x)
					}
				}
				sort.Slice(temp, func(i, j int) bool {
					return temp[i].Name < temp[j].Name
				})
				for i, _ := range temp {
					skid := kid.AddNewChild(nil, temp[i].Name)
					kn := skid.Embed(KiT_SymNode).(*SymNode)
					kn.SRoot = st.SRoot
					kn.Symbol = temp[i]
				}
			}
		}
	}
	for i, _ := range funcs {
		skid := st.AddNewChild(nil, funcs[i].Name)
		kn := skid.Embed(KiT_SymNode).(*SymNode)
		kn.SRoot = st.SRoot
		kn.Symbol = funcs[i]
	}
}
