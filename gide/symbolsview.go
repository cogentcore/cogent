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
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/pi/v2/lex"
	"goki.dev/pi/v2/syms"
	"goki.dev/pi/v2/token"
)

// SymbolsParams are parameters for structure view of file or package
type SymbolsParams struct {

	// scope of symbols to list
	Scope SymScopes
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
func (sv *SymbolsView) ConfigSymbolsView(ge Gide, sp SymbolsParams) {
	sv.Gide = ge
	sv.SymParams = sp
	sv.Style(func(s *styles.Style) {
		s.Direction = styles.Col
	})
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

	gi.NewButton(svbar).SetText("Refresh").SetIcon(icons.Update).SetTooltip("refresh symbols for current file and scope").
		OnClick(func(e events.Event) {
			sv.RefreshAction()
		})

	sl := gi.NewLabel(svbar).SetText("Scope:").SetTooltip("scope symbols to:")
	scb := gi.NewChooser(svbar).SetTooltip(sl.Tooltip)
	scb.SetEnum(sv.Params().Scope, false, 0)
	scb.OnChange(func(e events.Event) {
		sv.Params().Scope = scb.CurVal.(SymScopes)
		sv.ConfigTree(sv.Params().Scope)
		sv.SearchText().GrabFocus()
	})

	gi.NewLabel(svbar).SetText("Search:").SetTooltip("narrow symbols list to symbols containing text you enter here")
	stxt := gi.NewTextField(svbar, "search-str").SetTooltip("narrow symbols list by entering a search string -- case is ignored if string is all lowercase -- otherwise case is matched")
	stxt.OnChange(func(e events.Event) {
		sv.Match = stxt.Text()
		sv.ConfigTree(sv.Params().Scope)
		stxt.CursorEnd()
		stxt.GrabFocus()
	})
}

// RefreshAction loads symbols for current file and scope
func (sv *SymbolsView) RefreshAction() {
	sv.ConfigTree(SymScopes(sv.Params().Scope))
	sv.SearchText().GrabFocus()
}

// ConfigTree adds a treeview to the symbolsview
func (sv *SymbolsView) ConfigTree(scope SymScopes) {
	sfr := sv.Frame()
	updt := sfr.UpdateStart()
	var tv *SymTreeView
	if sv.Syms == nil {
		// sfr.SetProp("height", units.NewEm(5)) // enables scrolling

		sv.Syms = &SymNode{}
		sv.Syms.InitName(sv.Syms, "syms")

		tv = NewSymTreeView(sfr)
		tv.SyncRootNode(sv.Syms)
		tv.OnSelect(func(e events.Event) {
			sn := tv.SymNode()
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
		gi.NewDialog(sv).Title("Symbols not yet parsed").
			Prompt("Symbols not yet parsed -- try again in a few moments").Modal(true).Ok().Run()
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
		gi.NewDialog(sv).Title("Symbols not yet parsed").
			Prompt("Symbols not yet parsed -- try again in a few moments").Modal(true).Ok().Run()
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
				kn := NewSymNode(sn, sy.Name).SetSymbol(*sy)
				sort.Slice(fields, func(i, j int) bool {
					return fields[i].Name < fields[j].Name
				})
				sort.Slice(methods, func(i, j int) bool {
					return methods[i].Name < methods[j].Name
				})
				for _, fld := range fields {
					NewSymNode(kn, fld.Label()).SetSymbol(fld)
				}
				for _, mth := range methods {
					NewSymNode(kn, mth.Label()).SetSymbol(mth)
				}
			}
		}
	}
	for _, fn := range funcs {
		NewSymNode(sn, fn.Label()).SetSymbol(fn)
	}
	for _, vr := range gvars {
		NewSymNode(sn, vr.Label()).SetSymbol(vr)
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

// SymScopes corresponds to the search scope
type SymScopes int32 //enums:enum -trim-prefix SymScope

const (
	// SymScopePackage scopes list of symbols to the package of the active file
	SymScopePackage SymScopes = iota

	// SymScopeFile restricts the list of symbols to the active file
	SymScopeFile
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
	return st.SyncNode.(*SymNode)
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
