// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"image"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
)

// TextEditor is the Code-specific version of the TextEditor, with support for
// setting / clearing breakpoints, etc
type TextEditor struct {
	textcore.Editor

	Code *Code
}

func (ed *TextEditor) Init() {
	ed.Editor.Init()
	ed.AddContextMenu(ed.ContextMenu)
	ed.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.LongHoverable)
	})

	ed.On(events.Focus, func(e events.Event) {
		ed.Code.SetActiveEditor(ed)
	})
	// clicks move the cursor without an Input event or a key event,
	// which otherwise drive the cursor-dependent language features;
	// this runs before the editor moves the cursor, but the update is
	// deferred until after it.
	ed.OnClick(func(e events.Event) {
		if ed.Code != nil {
			ed.Code.LangSvc().ScheduleSigHelp(ed)
		}
	})
	ed.OnDoubleClick(func(e events.Event) {
		pt := ed.PointToRelPos(e.Pos())
		tpos := ed.PixelToCursor(pt)
		if ed.Lines != nil && pt.X >= 0 && ed.Lines.IsValidLine(tpos.Line) {
			if pt.X < int(ed.LineNumberPixels()) {
				e.SetHandled()
				ed.LineNumberDoubleClick(tpos)
				return
			}
			ed.HandleDebugDoubleClick(e, tpos)
		}
	})
}

func (ed *TextEditor) WidgetTooltip(pos image.Point) (string, image.Point) {
	if pos == image.Pt(-1, -1) {
		return "_", image.Point{}
	}
	if dbg := ed.DebugVarValueAtPos(pos); dbg != "" {
		return dbg, pos
	}
	// look for documentation on symbols via the language backend (gopls etc):
	if ed.Lines != nil && ed.Code != nil {
		pt := ed.PointToRelPos(pos)
		tpos := ed.PixelToCursor(pt)
		if ed.Lines.IsValidLine(tpos.Line) {
			w := ed.Lines.WordAt(tpos)
			hpos := tpos
			if w.Start.Line == tpos.Line && w.End.Char > 0 {
				hpos = w.End
			}
			if docs := ed.Code.LangSvc().HoverAt(ed, hpos); docs != "" {
				return docs, pos
			}
		}
	}
	return "", pos
}

// CurDebug returns the current debugger, true if it is present
func (ed *TextEditor) CurDebug() (*DebugPanel, bool) {
	if ed.Lines == nil {
		return nil, false
	}
	if ge, ok := ParentCode(ed); ok {
		dbg := ge.CurDebug()
		if dbg != nil {
			return dbg, true
		}
	}
	return nil, false
}

// SetBreakpoint sets breakpoint at given line (e.g., tv.CursorPos.Line)
func (ed *TextEditor) SetBreakpoint(ln int) {
	dbg, has := ed.CurDebug()
	if !has {
		return
	}
	ed.Lines.SetLineColor(ln, DebugBreakColors[DebugBreakInactive])
	dbg.AddBreak(ed.Lines.Filename(), ln+1)
}

func (ed *TextEditor) ClearBreakpoint(ln int) {
	if ed.Lines == nil {
		return
	}
	// tv.Buf.DeleteLineIcon(ln)
	ed.Lines.DeleteLineColor(ln)
	dbg, has := ed.CurDebug()
	if !has {
		return
	}
	dbg.DeleteBreak(ed.Lines.Filename(), ln+1)
}

// HasBreakpoint checks if line has a breakpoint
func (ed *TextEditor) HasBreakpoint(ln int) bool {
	if ed.Lines == nil {
		return false
	}
	_, has := ed.Lines.LineColor(ln)
	return has
}

func (ed *TextEditor) ToggleBreakpoint(ln int) {
	if ed.HasBreakpoint(ln) {
		ed.ClearBreakpoint(ln)
	} else {
		ed.SetBreakpoint(ln)
	}
}

// DebugVarValueAtPos returns debugger variable value for given mouse position
func (ed *TextEditor) DebugVarValueAtPos(pos image.Point) string {
	dbg, has := ed.CurDebug()
	if !has {
		return ""
	}
	pt := ed.PointToRelPos(pos)
	tpos := ed.PixelToCursor(pt)
	lx, _ := ed.Lines.HiTagAtPos(tpos)
	if lx == nil {
		return ""
	}
	if !lx.Token.Token.InCat(token.Name) {
		return ""
	}
	varNm := ed.Lines.LexObjPathString(tpos.Line, lx) // get full path
	val := dbg.VarValue(varNm)
	if val != "" {
		return varNm + " = " + val
	}
	return ""
}

// FindFrames finds stack frames in the debugger containing this file and line
func (ed *TextEditor) FindFrames(ln int) {
	dbg, has := ed.CurDebug()
	if !has {
		return
	}
	dbg.FindFrames(ed.Lines.Filename(), ln+1)
}

// DoubleClickEvent processes double-clicks NOT on the line-number section
func (ed *TextEditor) HandleDebugDoubleClick(e events.Event, tpos textpos.Pos) {
	dbg, has := ed.CurDebug()
	lx, _ := ed.Lines.HiTagAtPos(tpos)
	if has && lx != nil && lx.Token.Token.InCat(token.Name) {
		varNm := ed.Lines.LexObjPathString(tpos.Line, lx)
		err := dbg.ShowVar(varNm)
		if err == nil {
			e.SetHandled()
			return
		}
	}
	// todo: could do e.g., lookup here, but messes with normal select..
}

// LineNumberDoubleClick processes double-clicks on the line-number section
func (ed *TextEditor) LineNumberDoubleClick(tpos textpos.Pos) {
	ln := tpos.Line
	ed.ToggleBreakpoint(ln)
	ed.NeedsRender()
}

// ConfigOutputTextEditor configures a command-output texteditor within given parent layout
func ConfigOutputTextEditor(ed *textcore.Editor) {
	ed.SetReadOnly(true)
	ed.Styler(func(s *styles.Style) {
		ed.SetReadOnly(true)
		s.SetAbilities(true, abilities.ScrollableUnattended)
		ed.AutoscrollOnInput = true
		s.Text.WhiteSpace = text.WrapNever
		s.Text.TabSize = 8
		s.Min.X.Ch(20)
		s.Min.Y.Em(5)
		s.Grow.Set(1, 1)
		if ed.Lines != nil {
			ed.Lines.Settings.LineNumbers = false
		}
	})
}

// ContextMenu builds the text editor context menu
// ShowContextMenu shows the context menu, first moving the cursor to the
// position of the click if there is no selection, so that the language
// actions in the menu (lookup, code actions, etc) apply there.
func (ed *TextEditor) ShowContextMenu(e events.Event) {
	if ed.Lines != nil && !ed.HasSelection() {
		tpos := ed.PixelToCursor(ed.PointToRelPos(e.Pos()))
		if tpos != textpos.PosErr && ed.Lines.IsValidLine(tpos.Line) {
			ed.SetCursorShow(tpos)
		}
	}
	ed.Editor.ShowContextMenu(e)
}

func (ed *TextEditor) ContextMenu(m *core.Scene, pos image.Point) {
	core.NewButton(m).SetText("Copy").SetIcon(icons.ContentCopy).
		SetKey(keymap.Copy).SetState(!ed.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			ed.Copy(true)
		})
	if ed.IsReadOnly() {
		core.NewButton(m).SetText("Clear").SetIcon(icons.ClearAll).
			OnClick(func(e events.Event) {
				ed.Clear()
			})
		return
	}

	core.NewButton(m).SetText("Cut").SetIcon(icons.ContentCopy).
		SetKey(keymap.Cut).SetState(!ed.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			ed.Cut()
		})
	core.NewButton(m).SetText("Paste").SetIcon(icons.ContentPaste).
		SetKey(keymap.Paste).SetState(ed.Clipboard().IsEmpty(), states.Disabled).
		OnClick(func(e events.Event) {
			ed.Paste()
		})

	core.NewSeparator(m)
	if ed.Lines != nil && ed.Code != nil && ed.Code.LangSvc().LSPActive(ed.Lines) {
		core.NewFuncButton(m).SetFunc(ed.Code.Lookup).SetIcon(icons.Search).SetShortcut(KeyQuickFix.Chord())
		core.NewButton(m).SetText("Code actions").SetIcon(icons.Bolt).SetMenu(func(m *core.Scene, pos image.Point) {
			ed.Code.LangSvc().codeActionsMenu(m, ed)
		})
		core.NewFuncButton(m).SetFunc(ed.Code.GotoDefinition).SetText("Go to definition").SetIcon(icons.Directions)
		core.NewFuncButton(m).SetFunc(ed.Code.GotoTypeDefinition).SetText("Go to type").SetIcon(icons.Title)
		core.NewFuncButton(m).SetFunc(ed.Code.FindImplementations).SetText("Find implementations").SetIcon(icons.SearchCheck)
		core.NewFuncButton(m).SetFunc(ed.Code.FindReferences).SetText("Find references").SetIcon(icons.ManageSearch)
		core.NewFuncButton(m).SetFunc(ed.Code.ShowProblems).SetText("Show problems").SetIcon(icons.BugReport)
		core.NewFuncButton(m).SetFunc(ed.Code.RenameSymbol).SetText("Rename symbol...").SetIcon(icons.SwapHoriz)
		core.NewFuncButton(m).SetFunc(ed.Code.LangFormat).SetText("Format file").SetIcon(icons.TextFormat)
		// core.NewFuncButton(m).SetFunc(ed.Code.ShowInlayHints).SetText("Show inlay hints")
	} else {
		core.NewFuncButton(m).SetFunc(ed.Lookup).SetIcon(icons.Search)
	}

	fn := ed.Code.FileNodeForFile(ed.Lines.Filename())
	if fn != nil {
		core.NewSeparator(m)
		fn.SelectEvent(events.SelectOne)
		fn.VCSContextMenu(m, pos)
	}

	if ed.Code.CurDebug() != nil {
		core.NewSeparator(m)

		core.NewButton(m).SetText("Set breakpoint").SetIcon(icons.StopCircle).
			SetTooltip("debugger will stop here").OnClick(func(e events.Event) {
			ed.SetBreakpoint(ed.CursorPos.Line)
		})
		if ed.HasBreakpoint(ed.CursorPos.Line) {
			core.NewButton(m).SetText("Clear breakpoint").SetIcon(icons.Cancel).
				OnClick(func(e events.Event) {
					ed.ClearBreakpoint(ed.CursorPos.Line)
				})
		}
		core.NewButton(m).SetText("Debug: Find frames").SetIcon(icons.Cancel).
			SetTooltip("Finds stack frames in the debugger containing this file and line").
			OnClick(func(e events.Event) {
				ed.FindFrames(ed.CursorPos.Line)
			})
	}
}
