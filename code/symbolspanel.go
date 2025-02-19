// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"log"
	"sort"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/parse/syms"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/text/token"
	"cogentcore.org/core/tree"
)

// SymbolsParams are parameters for structure view of file or package
type SymbolsParams struct {

	// scope of symbols to list
	Scope SymScopes
}

// SymbolsPanel is a widget that displays results of a file or package parse of symbols.
type SymbolsPanel struct {
	core.Frame

	// parent code project
	Code *Code `json:"-" xml:"-"`

	// params for structure display
	SymParams SymbolsParams

	// all the symbols for the file or package in a tree
	Syms *SymNode

	// only show symbols that match this string
	Match string
}

// Params returns the symbols params
func (sv *SymbolsPanel) Params() *SymbolsParams {
	return &sv.Code.Settings.Symbols
}

func (sv *SymbolsPanel) Init() {
	sv.Frame.Init()
	sv.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})

	scope := sv.SymParams.Scope

	tree.AddChildAt(sv, "sym-toolbar", func(w *core.Toolbar) {
		w.Maker(sv.makeToolbar)
	})
	tree.AddChildAt(sv, "sym-frame", func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 1)
			s.Overflow.Set(styles.OverflowAuto)
		})
		tree.AddChildAt(w, "syms", func(w *SymTree) {
			sv.Syms = NewSymNode()
			sv.Syms.SetName("syms")
			if scope == SymScopePackage {
				sv.OpenPackage()
			} else {
				sv.OpenFile()
			}
			w.SyncTree(sv.Syms)
			w.OnSelect(func(e events.Event) {
				if len(w.SelectedNodes) == 0 {
					return
				}
				sn := w.SelectedNodes[0].AsCoreTree().SyncNode.(*SymNode)
				if sn != nil {
					SelectSymbol(sv.Code, sn.Symbol)
				}
			})
		})
	})
}

func (sv *SymbolsPanel) Config(cv *Code, sp SymbolsParams) { // TODO(config): better name?
	sv.Code = cv
	sv.SymParams = sp
}

func (sv *SymbolsPanel) UpdateSymbols() {
	scope := sv.SymParams.Scope
	tv := sv.FindPath("sym-frame/syms").(*SymTree)
	if scope == SymScopePackage {
		sv.OpenPackage()
	} else {
		sv.OpenFile()
	}
	tv.Resync()
	tv.OpenAll()
}

// Toolbar returns the symbols toolbar
func (sv *SymbolsPanel) Toolbar() *core.Toolbar {
	return sv.ChildByName("sym-toolbar", 0).(*core.Toolbar)
}

// FrameWidget returns the frame widget.
func (sv *SymbolsPanel) FrameWidget() *core.Frame {
	return sv.ChildByName("sym-frame", 0).(*core.Frame)
}

// ScopeChooser returns the scope Chooser
func (sv *SymbolsPanel) ScopeChooser() *core.Chooser {
	return sv.Toolbar().ChildByName("scope-chooser", 5).(*core.Chooser)
}

// SearchText returns the unknown word textfield from toolbar
func (sv *SymbolsPanel) SearchText() *core.TextField {
	return sv.Toolbar().ChildByName("search-str", 1).(*core.TextField)
}

// makeToolbar adds toolbar.
func (sv *SymbolsPanel) makeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.Button) {
		w.SetText("Refresh").SetIcon(icons.Update).
			SetTooltip("refresh symbols for current file and scope").
			OnClick(func(e events.Event) {
				sv.RefreshAction()
			})
	})

	tree.Add(p, func(w *core.Text) {
		w.SetText("Scope:").SetTooltip("scope symbols to:")
	})

	tree.AddAt(p, "scope-chooser", func(w *core.Chooser) {
		w.SetEnum(sv.Params().Scope)
		w.SetTooltip("scope symbols to:")
		w.OnChange(func(e events.Event) {
			sv.Params().Scope = w.CurrentItem.Value.(SymScopes)
			sv.UpdateSymbols()
			sv.SearchText().SetFocus()
		})
		w.SetCurrentValue(sv.Params().Scope) // todo: also update?
	})

	tree.Add(p, func(w *core.Text) {
		w.SetText("Search:").
			SetTooltip("narrow symbols list to symbols containing text you enter here")
	})

	tree.AddAt(p, "search-str", func(w *core.TextField) {
		w.SetTooltip("narrow symbols list by entering a search string -- case is ignored if string is all lowercase -- otherwise case is matched")
		w.OnChange(func(e events.Event) {
			sv.Match = w.Text()
			sv.UpdateSymbols()
			// TODO: why do this?
			// w.CursorEnd()
			// w.SetFocusEvent()
		})
	})
}

// RefreshAction loads symbols for current file and scope
func (sv *SymbolsPanel) RefreshAction() {
	sv.UpdateSymbols()
	sv.SearchText().SetFocus()
}

func SelectSymbol(cv *Code, ssym syms.Symbol) {
	tv := cv.ActiveTextEditor()
	if tv == nil || tv.Buffer == nil || string(tv.Buffer.Filename) != ssym.Filename {
		tr := text.NewRegion(ssym.SelectReg.St.Line, ssym.SelectReg.St.Ch, ssym.SelectReg.Ed.Line, ssym.SelectReg.Ed.Ch)
		_, ok := cv.OpenFileAtRegion(core.Filename(ssym.Filename), tr)
		if !ok {
			log.Printf("Code SelectSymbol: OpenFileAtRegion returned false: %v\n", ssym.Filename)
		}
		return
	}

	tv.Highlights = tv.Highlights[:0]
	tr := text.NewRegion(ssym.SelectReg.St.Line, ssym.SelectReg.St.Ch, ssym.SelectReg.Ed.Line, ssym.SelectReg.Ed.Ch)
	tv.Highlights = append(tv.Highlights, tr)
	tv.SetCursorTarget(tr.Start)
	tv.SetFocus()
	cv.FocusOnTabs()
	tv.NeedsLayout()
}

// OpenPackage opens package-level symbols for current active texteditor
func (sv *SymbolsPanel) OpenPackage() {
	cv := sv.Code
	tv := cv.ActiveTextEditor()
	if sv.Syms == nil || tv == nil || tv.Buffer == nil || !tv.Buffer.Highlighter.UsingParse() {
		return
	}
	pfs := tv.Buffer.ParseState.Done()
	if len(pfs.ParseState.Scopes) == 0 {
		core.MessageSnackbar(sv, "Symbols not yet parsed -- try again in a few moments")
		return
	}
	pkg := pfs.ParseState.Scopes[0] // first scope of parse state is the full set of package symbols
	sv.Syms.OpenSyms(pkg, "", sv.Match)
}

// OpenFile opens file-level symbols for current active texteditor
func (sv *SymbolsPanel) OpenFile() {
	cv := sv.Code
	tv := cv.ActiveTextEditor()
	if sv.Syms == nil || tv == nil || tv.Buffer == nil || !tv.Buffer.Highlighter.UsingParse() {
		return
	}
	pfs := tv.Buffer.ParseState.Done()
	if len(pfs.ParseState.Scopes) == 0 {
		core.MessageSnackbar(sv, "Symbols not yet parsed -- try again in a few moments")
		return
	}
	pkg := pfs.ParseState.Scopes[0] // first scope of parse state is the full set of package symbols
	sv.Syms.OpenSyms(pkg, string(tv.Buffer.Filename), sv.Match)
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

	ignoreCase := !lexer.HasUpperCase(match)

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
				kn := NewSymNode(sn).SetSymbol(*sy)
				kn.SetName(sy.Name)
				sort.Slice(fields, func(i, j int) bool {
					return fields[i].Name < fields[j].Name
				})
				sort.Slice(methods, func(i, j int) bool {
					return methods[i].Name < methods[j].Name
				})
				for _, fld := range fields {
					fn := NewSymNode(kn).SetSymbol(fld)
					fn.SetName(fld.Label())
				}
				for _, mth := range methods {
					mn := NewSymNode(kn).SetSymbol(mth)
					mn.SetName(mth.Label())
				}
			}
		}
	}
	for _, fn := range funcs {
		n := NewSymNode(sn).SetSymbol(fn)
		n.SetName(fn.Label())
	}
	for _, vr := range gvars {
		n := NewSymNode(sn).SetSymbol(vr)
		n.SetName(vr.Label())
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
	tree.NodeBase

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
// SymTree

// SymTree is a Tree that knows how to operate on FileNode nodes
type SymTree struct {
	core.Tree
}

// SymNode returns the SrcNode as a *code* SymNode
func (st *SymTree) SymNode() *SymNode {
	return st.SyncNode.(*SymNode)
}

func (st *SymTree) Init() {
	st.Tree.Init()

	st.Parts.Styler(func(s *styles.Style) {
		s.Gap.X.Em(0.4)
	})
	tree.AddChildInit(st.Parts, "branch", func(w *core.Switch) {
		w.SetIconOn(st.IconOpen).SetIconOff(st.IconClosed).SetIconIndeterminate(st.SymNode().GetIcon())
		tree.AddChildInit(w, "stack", func(w *core.Frame) {
			tree.AddChildInit(w, "icon-indeterminate", func(w *core.Icon) {
				w.Styler(func(s *styles.Style) {
					s.Min.Set(units.Em(1))
				})
			})
		})
	})
}
