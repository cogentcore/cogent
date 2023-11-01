// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"log"
	"sort"
	"strings"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/texteditor/textbuf"
	"goki.dev/girl/units"
	"goki.dev/ki/v2"
	"goki.dev/pi/v2/lex"
	"goki.dev/pi/v2/syms"
	"goki.dev/pi/v2/token"
)

// SymbolsParams are parameters for structure view of file or package
type SymbolsParams struct {

	// scope of symbols to list
	Scope SymbolsViewScope
}

// SymbolsView is a widget that displays results of a file or package parse
type SymbolsView struct {
	gi.Layout

	// parent gide project
	Gide Gide `json:"-" xml:"-"`

	// params for structure display
	SymParams SymbolsParams

	// all the symbols for the file or package in a tree
	Syms *SymNode

	// only show symbols that match this string
	Match string
}

// Params returns the symbols params
func (sv *SymbolsView) Params() *SymbolsParams {
	return &sv.Gide.ProjPrefs().Symbols
}

//////////////////////////////////////////////////////////////////////////////////////
//    GUI config

// Config configures the view
func (sv *SymbolsView) Config(ge Gide, sp SymbolsParams) {
	sv.Gide = ge
	sv.SymParams = sp
	sv.Lay = gi.LayoutVert
	sv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := ki.Config{}
	config.Add(gi.ToolbarType, "sym-toolbar")
	config.Add(gi.FrameType, "sym-frame")
	mods, updt := sv.ConfigChildren(config)
	if !mods {
		updt = sv.UpdateStart()
	}
	sv.ConfigToolbar()
	sb := sv.ScopeCombo()
	sb.SetCurIndex(int(sv.Params().Scope))
	sv.ConfigTree(sp.Scope)
	sv.UpdateEnd(updt)
}

// Toolbar returns the symbols toolbar
func (sv *SymbolsView) Toolbar() *gi.Toolbar {
	return sv.ChildByName("sym-toolbar", 0).(*gi.Toolbar)
}

// Toolbar returns the spell toolbar
func (sv *SymbolsView) Frame() *gi.Frame {
	return sv.ChildByName("sym-frame", 0).(*gi.Frame)
}

// ScopeCombo returns the scope ComboBox
func (sv *SymbolsView) ScopeCombo() *gi.Chooser {
	return sv.Toolbar().ChildByName("scope-combo", 5).(*gi.Chooser)
}

// SearchText returns the unknown word textfield from toolbar
func (sv *SymbolsView) SearchText() *gi.TextField {
	return sv.Toolbar().ChildByName("search-str", 1).(*gi.TextField)
}

// ConfigToolbar adds toolbar.
func (sv *SymbolsView) ConfigToolbar() {
	svbar := sv.Toolbar()
	if svbar.HasChildren() {
		return
	}
	svbar.SetStretchMaxWidth()

	svbar.AddAction(gi.ActOpts{Label: "Refresh", Icon: "update", Tooltip: "refresh symbols for current file and scope"},
		sv.This(), func(recv, send ki.Ki, sig int64, data any) {
			svv, _ := recv.Embed(KiT_SymbolsView).(*SymbolsView)
			svv.RefreshAction()
		})
	sl := svbar.NewChild(gi.LabelType, "scope-lbl").(*gi.Label)
	sl.SetText("Scope:")
	sl.Tooltip = "scope symbols to:"
	scb := svbar.NewChild(gi.KiT_ComboBox, "scope-combo").(*gi.Chooser)
	scb.SetText("Scope")
	scb.Tooltip = sl.Tooltip
	scb.ItemsFromEnum(Kit_SymbolsViewScope, false, 0)
	// scb.ComboSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	svv, _ := recv.Embed(KiT_SymbolsView).(*SymbolsView)
	// 	smb := send.(*gi.Chooser)
	// 	eval := smb.CurVal.(kit.EnumValue)
	// 	svv.Params().Scope = SymbolsViewScope(eval.Value)
	// 	sv.ConfigTree(SymbolsViewScope(eval.Value))
	// 	sv.SearchText().GrabFocus()
	// })

	slbl := svbar.NewChild(gi.LabelType, "search-lbl").(*gi.Label)
	slbl.SetText("Search:")
	slbl.Tooltip = "narrow symbols list to symbols containing text you enter here"
	stxt := svbar.NewChild(gi.TextField, "search-str").(*gi.TextField)
	stxt.SetStretchMaxWidth()
	stxt.Tooltip = "narrow symbols list by entering a search string -- case is ignored if string is all lowercase -- otherwise case is matched"
	stxt.SetActiveState(true)
	stxt.TextFieldSig.ConnectOnly(stxt.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(gi.TextFieldInsert) || sig == int64(gi.TextFieldBackspace) || sig == int64(gi.TextFieldDelete) {
			sv.Match = sv.SearchText().Text()
			sv.ConfigTree(sv.Params().Scope)
			stxt.CursorEnd()
			sv.SearchText().GrabFocus()
		}
		if sig == int64(gi.TextFieldCleared) {
			sv.Match = ""
			sv.SearchText().SetText(sv.Match)
			sv.ConfigTree(sv.Params().Scope)
			sv.SearchText().GrabFocus()
		}
	})
}

// RefreshAction loads symbols for current file and scope
func (sv *SymbolsView) RefreshAction() {
	sv.ConfigTree(SymbolsViewScope(sv.Params().Scope))
	sv.SearchText().GrabFocus()
}

// ConfigTree adds a treeview to the symbolsview
func (sv *SymbolsView) ConfigTree(scope SymbolsViewScope) {
	sfr := sv.Frame()
	updt := sfr.UpdateStart()
	sfr.SetFullReRender()
	var tv *SymTreeView
	if sv.Syms == nil {
		sfr.SetProp("height", units.NewEm(5)) // enables scrolling
		sfr.SetStretchMaxWidth()
		sfr.SetStretchMaxHeight()
		// sfr.SetReRenderAnchor()  // must be off if using SetFullReRender

		sv.Syms = &SymNode{}
		sv.Syms.InitName(sv.Syms, "syms")

		tv = sfr.NewChild(KiT_SymTreeView, "treeview").(*SymTreeView)
		tv.SetRootNode(sv.Syms)
		tv.TreeViewSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if data == nil || sig != int64(giv.TreeViewSelected) {
				return
			}
			tvn, _ := data.(ki.Ki).Embed(KiT_SymTreeView).(*SymTreeView)
			sn := tvn.SymNode()
			if sn != nil {
				sv.SelectSymbol(sn.Symbol)
			}
		})
	} else {
		tv = sfr.Child(0).(*SymTreeView)
	}

	if scope == SymScopePackage {
		sv.OpenPackage()
	} else {
		sv.OpenFile()
	}
	tv.ReSync()

	tv.OpenAll()
	sfr.UpdateEnd(updt)
}

func (sv *SymbolsView) SelectSymbol(ssym syms.Symbol) {
	ge := sv.Gide
	tv := ge.ActiveTextView()
	if tv == nil || string(tv.Buf.Filename) != ssym.Filename {
		var ok = false
		tr := textbuf.NewRegion(ssym.SelectReg.St.Ln, ssym.SelectReg.St.Ch, ssym.SelectReg.Ed.Ln, ssym.SelectReg.Ed.Ch)
		tv, ok = ge.OpenFileAtRegion(gi.FileName(ssym.Filename), tr)
		if ok == false {
			log.Printf("GideView SelectSymbol: OpenFileAtRegion returned false: %v\n", ssym.Filename)
		}
	} else {
		tv.UpdateStart()
		tv.Highlights = tv.Highlights[:0]
		tr := textbuf.NewRegion(ssym.SelectReg.St.Ln, ssym.SelectReg.St.Ch, ssym.SelectReg.Ed.Ln, ssym.SelectReg.Ed.Ch)
		tv.Highlights = append(tv.Highlights, tr)
		tv.UpdateEnd(true)
		tv.RefreshIfNeeded()
		tv.SetCursorShow(tr.Start)
		tv.GrabFocus()
	}
}

// OpenPackage opens package-level symbols for current active textview
func (sv *SymbolsView) OpenPackage() {
	ge := sv.Gide
	tv := ge.ActiveTextView()
	if sv.Syms == nil || tv == nil || tv.Buf == nil || !tv.Buf.Hi.UsingPi() {
		return
	}
	pfs := tv.Buf.PiState.Done()
	if len(pfs.ParseState.Scopes) == 0 {
		gi.PromptDialog(sv.ViewportSafe(), gi.DlgOpts{Title: "Symbols not yet parsed", Prompt: "Symbols not yet parsed -- try again in a few moments"}, gi.AddOk, gi.NoCancel, nil, nil)
		return
	}
	pkg := pfs.ParseState.Scopes[0] // first scope of parse state is the full set of package symbols
	sv.Syms.OpenSyms(pkg, "", sv.Match)
}

// OpenFile opens file-level symbols for current active textview
func (sv *SymbolsView) OpenFile() {
	ge := sv.Gide
	tv := ge.ActiveTextView()
	if sv.Syms == nil || tv == nil || tv.Buf == nil || !tv.Buf.Hi.UsingPi() {
		return
	}
	pfs := tv.Buf.PiState.Done()
	if len(pfs.ParseState.Scopes) == 0 {
		gi.PromptDialog(sv.ViewportSafe(), gi.DlgOpts{Title: "Symbols not yet parsed", Prompt: "Symbols not yet parsed -- try again in a few moments"}, gi.AddOk, gi.NoCancel, nil, nil)
		return
	}
	pkg := pfs.ParseState.Scopes[0] // first scope of parse state is the full set of package symbols
	sv.Syms.OpenSyms(pkg, string(tv.Buf.Filename), sv.Match)
}

func symMatch(str, match string, ignoreCase bool) bool {
	if match == "" {
		return true
	}
	if ignoreCase {
		return strings.Contains(strings.ToLower(str), match)
	}
	return strings.Contains(str, match)
}

// OpenSyms opens symbols from given symbol map (assumed to be package-level symbols)
// filtered by filename and match -- called on root node of tree.
func (sn *SymNode) OpenSyms(pkg *syms.Symbol, fname, match string) {
	sn.DeleteChildren(ki.DestroyKids)

	gvars := []syms.Symbol{} // collect and list global vars first
	funcs := []syms.Symbol{} // collect and add functions (no receiver) to end

	ignoreCase := !lex.HasUpperCase(match)

	sls := pkg.Children.Slice(true)
	for _, sy := range sls {
		if fname != "" {
			if sy.Filename != fname { // this is what restricts to single file
				continue
			}
		}
		if sy.Name == "" || sy.Name[0] == '_' {
			continue
		}
		switch {
		case sy.Kind.SubCat() == token.NameFunction:
			if symMatch(sy.Name, match, ignoreCase) {
				funcs = append(funcs, *sy)
			}
		case sy.Kind.SubCat() == token.NameVar:
			if symMatch(sy.Name, match, ignoreCase) {
				gvars = append(gvars, *sy)
			}
		case sy.Kind.SubCat() == token.NameType:
			var methods []syms.Symbol
			var fields []syms.Symbol
			for _, x := range sy.Children {
				if symMatch(x.Name, match, ignoreCase) {
					if x.Kind == token.NameMethod {
						methods = append(methods, *x)
					} else if x.Kind == token.NameField {
						fields = append(fields, *x)
					}
				}
			}
			if symMatch(sy.Name, match, ignoreCase) || len(methods) > 0 || len(fields) > 0 {
				kn := sn.NewChild(KiT_SymNode, sy.Name).(*SymNode)
				kn.Symbol = *sy
				sort.Slice(fields, func(i, j int) bool {
					return fields[i].Name < fields[j].Name
				})
				sort.Slice(methods, func(i, j int) bool {
					return methods[i].Name < methods[j].Name
				})
				for _, fld := range fields {
					dnm := fld.Label()
					fn := kn.NewChild(KiT_SymNode, dnm).(*SymNode)
					fn.Symbol = fld
				}
				for _, mth := range methods {
					dnm := mth.Label()
					mn := kn.NewChild(KiT_SymNode, dnm).(*SymNode)
					mn.Symbol = mth
				}
			}
		}
	}
	for _, fn := range funcs {
		dnm := fn.Label()
		fk := sn.NewChild(KiT_SymNode, dnm).(*SymNode)
		fk.Symbol = fn
	}
	for _, vr := range gvars {
		dnm := vr.Label()
		vk := sn.NewChild(KiT_SymNode, dnm).(*SymNode)
		vk.Symbol = vr
	}
}

// SymbolsViewProps are style properties for SymbolsView
// var SymbolsViewProps = ki.Props{
// 	"EnumType:Flag":    gi.KiT_NodeFlags,
// 	"background-color": &gi.Prefs.Colors.Background,
// 	"color":            &gi.Prefs.Colors.Font,
// 	"max-width":        -1,
// 	"max-height":       -1,
// }

/////////////////////////////////////////////////////////////////////////////
// SymNode

// SymScope corresponds to the search scope
type SymbolsViewScope int32 //enums:enum -trim-prefix SymScope

const (
	// SymScopePackage scopes list of symbols to the package of the active file
	SymScopePackage SymbolsViewScope = iota

	// SymScopeFile restricts the list of symbols to the active file
	SymScopeFile

	// SymScopeN is the number of symbol scopes
	SymScopeN
)

// SymNode represents a language symbol -- the name of the node is
// the name of the symbol. Some symbols, e.g. type have children
type SymNode struct {
	ki.Node

	// the symbol
	Symbol syms.Symbol
}

/////////////////////////////////////////////////////////////////////////////
// SymTreeView

// SymTreeView is a TreeView that knows how to operate on FileNode nodes
type SymTreeView struct {
	giv.TreeView
}

// SymNode returns the SrcNode as a *gide* SymNode
func (st *SymTreeView) SymNode() *SymNode {
	sn := st.SrcNode.Embed(KiT_SymNode)
	if sn == nil {
		return nil
	}
	return sn.(*SymNode)
}

/*
var SymTreeViewProps = ki.Props{
	"EnumType:Flag":    giv.KiT_TreeViewFlags,
	"indent":           units.NewValue(2, units.Ch),
	"spacing":          units.NewValue(.5, units.Ch),
	"border-width":     units.NewValue(0, units.Px),
	"border-radius":    units.NewValue(0, units.Px),
	"padding":          units.NewValue(0, units.Px),
	"margin":           units.NewValue(1, units.Px),
	"text-align":       gist.AlignLeft,
	"vertical-align":   gist.AlignTop,
	"color":            &gi.Prefs.Colors.Font,
	"background-color": "inherit",
	".exec": ki.Props{
		"font-weight": gist.WeightBold,
	},
	".open": ki.Props{
		"font-style": gist.FontItalic,
	},
	"#icon": ki.Props{
		"width":   units.NewValue(1, units.Em),
		"height":  units.NewValue(1, units.Em),
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
		"fill":    &gi.Prefs.Colors.Icon,
		"stroke":  &gi.Prefs.Colors.Font,
	},
	"#branch": ki.Props{
		"icon":             "wedge-down",
		"icon-off":         "wedge-right",
		"margin":           units.NewValue(0, units.Px),
		"padding":          units.NewValue(0, units.Px),
		"background-color": color.Transparent,
		"max-width":        units.NewValue(.8, units.Em),
		"max-height":       units.NewValue(.8, units.Em),
	},
	"#space": ki.Props{
		"width": units.NewValue(.5, units.Em),
	},
	"#label": ki.Props{
		"margin":    units.NewValue(0, units.Px),
		"padding":   units.NewValue(0, units.Px),
		"min-width": units.NewValue(16, units.Ch),
	},
	"#menu": ki.Props{
		"indicator": "none",
	},
	giv.TreeViewSelectors[giv.TreeViewActive]: ki.Props{},
	giv.TreeViewSelectors[giv.TreeViewSel]: ki.Props{
		"background-color": &gi.Prefs.Colors.Select,
	},
	giv.TreeViewSelectors[giv.TreeViewFocus]: ki.Props{
		"background-color": &gi.Prefs.Colors.Control,
	},
	"CtxtMenuActive": ki.PropSlice{},
}

func (st *SymTreeView) Style2D() {
	sn := st.SymNode()
	st.Class = ""
	if sn != nil {
		if sn.Symbol.Kind == token.NameType {
			st.Icon = gi.IconName("type")
		} else if sn.Symbol.Kind == token.NameVar || sn.Symbol.Kind == token.NameVarGlobal {
			st.Icon = gi.IconName("var")
		} else if sn.Symbol.Kind == token.NameMethod {
			st.Icon = gi.IconName("method")
		} else if sn.Symbol.Kind == token.NameFunction {
			st.Icon = gi.IconName("function")
		} else if sn.Symbol.Kind == token.NameField {
			st.Icon = gi.IconName("field")
		}
	}
	st.StyleTreeView()
	st.LayState.SetFromStyle(&st.Sty.Layout) // also does reset
}
*/
