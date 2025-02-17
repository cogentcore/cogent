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
		ed.Code.SetActiveTextEditor(ed)
	})
	ed.OnDoubleClick(func(e events.Event) {
		pt := ed.PointToRelPos(e.Pos())
		tpos := ed.PixelToCursor(pt)
		if ed.Buffer != nil && pt.X >= 0 && ed.Buffer.IsValidLine(tpos.Line) {
			if pt.X < int(ed.LineNumberOffset) {
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
	// todo: look for documentation on symbols here; we don't actually have this
	// in parse so we need lsp to make this work
	return ed.DebugVarValueAtPos(pos), pos
}

// CurDebug returns the current debugger, true if it is present
func (ed *TextEditor) CurDebug() (*DebugPanel, bool) {
	if ed.Buffer == nil {
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
	ed.Buffer.SetLineColor(ln, DebugBreakColors[DebugBreakInactive])
	dbg.AddBreak(string(ed.Buffer.Filename), ln+1)
}

func (ed *TextEditor) ClearBreakpoint(ln int) {
	if ed.Buffer == nil {
		return
	}
	// tv.Buf.DeleteLineIcon(ln)
	ed.Buffer.DeleteLineColor(ln)
	dbg, has := ed.CurDebug()
	if !has {
		return
	}
	dbg.DeleteBreak(string(ed.Buffer.Filename), ln+1)
}

// HasBreakpoint checks if line has a breakpoint
func (ed *TextEditor) HasBreakpoint(ln int) bool {
	if ed.Buffer == nil {
		return false
	}
	_, has := ed.Buffer.LineColors[ln]
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
	lx, _ := ed.Buffer.HiTagAtPos(tpos)
	if lx == nil {
		return ""
	}
	if !lx.Token.Token.InCat(token.Name) {
		return ""
	}
	varNm := ed.Buffer.LexObjPathString(tpos.Line, lx) // get full path
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
	dbg.FindFrames(string(ed.Buffer.Filename), ln+1)
}

// DoubleClickEvent processes double-clicks NOT on the line-number section
func (ed *TextEditor) HandleDebugDoubleClick(e events.Event, tpos textpos.Pos) {
	dbg, has := ed.CurDebug()
	lx, _ := ed.Buffer.HiTagAtPos(tpos)
	if has && lx != nil && lx.Token.Token.InCat(token.Name) {
		varNm := ed.Buffer.LexObjPathString(tpos.Line, lx)
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
		s.Text.WhiteSpace = styles.WhiteSpacePreWrap
		s.Text.TabSize = 8
		s.Min.X.Ch(20)
		s.Min.Y.Em(5)
		s.Grow.Set(1, 1)
		if ed.Buffer != nil {
			ed.Buffer.Options.LineNumbers = false
		}
	})
}

// ContextMenu builds the text editor context menu
func (ed *TextEditor) ContextMenu(m *core.Scene) {
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
	core.NewFuncButton(m).SetFunc(ed.Lookup).SetIcon(icons.Search)

	fn := ed.Code.FileNodeForFile(string(ed.Buffer.Filename), false)
	if fn != nil {
		fn.SelectEvent(events.SelectOne)
		fn.VCSContextMenu(m)
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
