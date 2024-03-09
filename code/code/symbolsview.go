// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"log"
	"sort"
	"strings"

	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/pi/lex"
	"cogentcore.org/core/pi/syms"
	"cogentcore.org/core/pi/token"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/texteditor/textbuf"
)

// SymbolsParams are parameters for structure view of file or package
type SymbolsParams struct {

	// scope of symbols to list
	Scope SymScopes
}

// SymbolsView is a widget that displays results of a file or package parse
type SymbolsView struct {
	gi.Layout

	// parent code project
	Code Code `json:"-" xml:"-"`

	// params for structure display
	SymParams SymbolsParams

	// all the symbols for the file or package in a tree
	Syms *SymNode

	// only show symbols that match this string
	Match string
}

// Params returns the symbols params
func (sv *SymbolsView) Params() *SymbolsParams {
	return &sv.Code.ProjSettings().Symbols
}

//////////////////////////////////////////////////////////////////////////////////////
//    GUI config

// Config configures the view
func (sv *SymbolsView) ConfigSymbolsView(ge Code, sp SymbolsParams) {
	sv.Code = ge
	sv.SymParams = sp
	if sv.HasChildren() {
		return
	}
	sv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	gi.NewToolbar(sv, "sym-toolbar")
	svfr := gi.NewFrame(sv, "sym-frame")
	svfr.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Overflow.Set(styles.OverflowAuto)
	})
	sv.ConfigToolbar()
	sb := sv.ScopeChooser()
	sb.SetCurrentIndex(int(sv.Params().Scope))
	sv.ConfigTree(sp.Scope)
}

// Toolbar returns the symbols toolbar
func (sv *SymbolsView) Toolbar() *gi.Toolbar {
	return sv.ChildByName("sym-toolbar", 0).(*gi.Toolbar)
}

// Toolbar returns the spell toolbar
func (sv *SymbolsView) Frame() *gi.Frame {
	return sv.ChildByName("sym-frame", 0).(*gi.Frame)
}

// ScopeChooser returns the scope Chooser
func (sv *SymbolsView) ScopeChooser() *gi.Chooser {
	return sv.Toolbar().ChildByName("scope-chooser", 5).(*gi.Chooser)
}

// SearchText returns the unknown word textfield from toolbar
func (sv *SymbolsView) SearchText() *gi.TextField {
	return sv.Toolbar().ChildByName("search-str", 1).(*gi.TextField)
}

// ConfigToolbar adds toolbar.
func (sv *SymbolsView) ConfigToolbar() {
	tb := sv.Toolbar()
	if tb.HasChildren() {
		return
	}

	gi.NewButton(tb).SetText("Refresh").SetIcon(icons.Update).
		SetTooltip("refresh symbols for current file and scope").
		OnClick(func(e events.Event) {
			sv.RefreshAction()
		})

	sl := gi.NewLabel(tb).SetText("Scope:").SetTooltip("scope symbols to:")

	ch := gi.NewChooser(tb, "scope-chooser").SetEnum(sv.Params().Scope)
	ch.SetTooltip(sl.Tooltip)
	ch.OnChange(func(e events.Event) {
		sv.Params().Scope = ch.CurrentItem.Value.(SymScopes)
		sv.ConfigTree(sv.Params().Scope)
		sv.SearchText().SetFocusEvent()
	})
	ch.SetCurrentValue(sv.Params().Scope)

	gi.NewLabel(tb).SetText("Search:").
		SetTooltip("narrow symbols list to symbols containing text you enter here")

	tf := gi.NewTextField(tb, "search-str")
	tf.SetTooltip("narrow symbols list by entering a search string -- case is ignored if string is all lowercase -- otherwise case is matched")
	tf.OnChange(func(e events.Event) {
		sv.Match = tf.Text()
		sv.ConfigTree(sv.Params().Scope)
		tf.CursorEnd()
		tf.SetFocusEvent()
	})
}

// RefreshAction loads symbols for current file and scope
func (sv *SymbolsView) RefreshAction() {
	sv.ConfigTree(SymScopes(sv.Params().Scope))
	sv.SearchText().SetFocusEvent()
}

// ConfigTree adds a treeview to the symbolsview.
// This is called for refresh action.
func (sv *SymbolsView) ConfigTree(scope SymScopes) {
	sfr := sv.Frame()
	var tv *SymTreeView
	if sv.Syms == nil {
		sv.Syms = &SymNode{}
		sv.Syms.InitName(sv.Syms, "syms")
		tv = NewSymTreeView(sfr)
		tv.SyncRootNode(sv.Syms)
		tv.OnSelect(func(e events.Event) {
			if len(tv.SelectedNodes) == 0 {
				return
			}
			sn := tv.SelectedNodes[0].AsTreeView().SyncNode.(*SymNode)
			if sn != nil {
				SelectSymbol(sv.Code, sn.Symbol)
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
	sfr.NeedsLayout()
}

func SelectSymbol(ge Code, ssym syms.Symbol) {
	tv := ge.ActiveTextEditor()
	if tv == nil || tv.Buf == nil || string(tv.Buf.Filename) != ssym.Filename {
		var ok = false
		tr := textbuf.NewRegion(ssym.SelectReg.St.Ln, ssym.SelectReg.St.Ch, ssym.SelectReg.Ed.Ln, ssym.SelectReg.Ed.Ch)
		tv, ok = ge.OpenFileAtRegion(gi.Filename(ssym.Filename), tr)
		if !ok {
			log.Printf("CodeView SelectSymbol: OpenFileAtRegion returned false: %v\n", ssym.Filename)
		}
		return
	}

	tv.Highlights = tv.Highlights[:0]
	tr := textbuf.NewRegion(ssym.SelectReg.St.Ln, ssym.SelectReg.St.Ch, ssym.SelectReg.Ed.Ln, ssym.SelectReg.Ed.Ch)
	tv.Highlights = append(tv.Highlights, tr)
	tv.SetCursorTarget(tr.Start)
	tv.SetFocusEvent()
	ge.FocusOnTabs()
	tv.NeedsLayout()
}

// OpenPackage opens package-level symbols for current active texteditor
func (sv *SymbolsView) OpenPackage() {
	ge := sv.Code
	tv := ge.ActiveTextEditor()
	if sv.Syms == nil || tv == nil || tv.Buf == nil || !tv.Buf.Hi.UsingPi() {
		return
	}
	pfs := tv.Buf.PiState.Done()
	if len(pfs.ParseState.Scopes) == 0 {
		gi.MessageSnackbar(sv, "Symbols not yet parsed -- try again in a few moments")
		return
	}
	pkg := pfs.ParseState.Scopes[0] // first scope of parse state is the full set of package symbols
	sv.Syms.OpenSyms(pkg, "", sv.Match)
}

// OpenFile opens file-level symbols for current active texteditor
func (sv *SymbolsView) OpenFile() {
	ge := sv.Code
	tv := ge.ActiveTextEditor()
	if sv.Syms == nil || tv == nil || tv.Buf == nil || !tv.Buf.Hi.UsingPi() {
		return
	}
	pfs := tv.Buf.PiState.Done()
	if len(pfs.ParseState.Scopes) == 0 {
		gi.MessageSnackbar(sv, "Symbols not yet parsed -- try again in a few moments")
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
	sn.DeleteChildren()

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

// GetIcon returns the appropriate Icon for this symbol type
func (sy *SymNode) GetIcon() icons.Icon {
	ic := icons.Blank
	switch sy.Symbol.Kind {
	case token.NameType:
		ic = icons.Title
	case token.NameVar, token.NameVarGlobal:
		ic = icons.Variable
	case token.NameMethod:
		ic = icons.Method
	case token.NameFunction:
		ic = icons.Function
	case token.NameField:
		ic = icons.Field
	case token.NameConstant:
		ic = icons.Constant
	}
	return ic
}

/////////////////////////////////////////////////////////////////////////////
// SymTreeView

// SymTreeView is a TreeView that knows how to operate on FileNode nodes
type SymTreeView struct {
	giv.TreeView
}

// SymNode returns the SrcNode as a *code* SymNode
func (st *SymTreeView) SymNode() *SymNode {
	return st.SyncNode.(*SymNode)
}

func (st *SymTreeView) OnInit() {
	st.TreeView.OnInit()
}

func (st *SymTreeView) UpdateBranchIcons() {
	st.SetSymIcon()
}

func (st *SymTreeView) SetSymIcon() {
	ic := st.SymNode().GetIcon()
	if _, ok := st.BranchPart(); !ok {
		st.Update()
	}
	if bp, ok := st.BranchPart(); ok {
		if bp.IconIndeterminate != ic {
			bp.IconIndeterminate = ic
			bp.Update()
			st.NeedsRender()
		}
	}
}
