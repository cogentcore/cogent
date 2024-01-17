// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"image"

	"cogentcore.org/core/abilities"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/pi/lex"
	"cogentcore.org/core/pi/token"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/texteditor"
)

// TextEditor is the Code-specific version of the TextEditor, with support for
// setting / clearing breakpoints, etc
type TextEditor struct {
	texteditor.Editor

	Code Code
}

func (ed *TextEditor) OnInit() {
	ed.Editor.OnInit()
	ed.HandleEvents()
	ed.SetStyles()
	ed.AddContextMenu(ed.ContextMenu)
}

func (ed *TextEditor) SetStyles() {
	ed.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.LongHoverable)
	})
}

// HandleEvents sets connections between mouse and key events and actions
func (ed *TextEditor) HandleEvents() {
	ed.On(events.Focus, func(e events.Event) {
		ed.Code.SetActiveTextEditor(ed)
	})
	ed.OnDoubleClick(func(e events.Event) {
		pt := ed.PointToRelPos(e.Pos())
		tpos := ed.PixelToCursor(pt)
		if ed.Buf != nil && pt.X >= 0 && ed.Buf.IsValidLine(tpos.Ln) {
			if pt.X < int(ed.LineNoOff) {
				e.SetHandled()
				ed.LineNoDoubleClick(tpos)
				return
			}
			ed.HandleDebugDoubleClick(e, tpos)
		}
	})
	ed.On(events.LongHoverStart, func(e events.Event) {
		tt := ""
		vv := ed.DebugVarValueAtPos(e.Pos())
		if vv != "" {
			tt = vv
		}
		// todo: look for documentation on symbols here -- we don't actually have this
		// in pi so we need lsp to make this work
		if tt != "" {
			e.SetHandled()
			pos := e.Pos()
			pos.X += 20
			pos.Y += 20
			gi.NewTooltipText(ed, tt).SetPos(pos).Run()
		}
	})
}

// CurDebug returns the current debugger, true if it is present
func (ed *TextEditor) CurDebug() (*DebugView, bool) {
	if ed.Buf == nil {
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

// SetBreakpoint sets breakpoint at given line (e.g., tv.CursorPos.Ln)
func (ed *TextEditor) SetBreakpoint(ln int) {
	dbg, has := ed.CurDebug()
	if !has {
		return
	}
	// tv.Buf.SetLineIcon(ln, "stop")
	ed.Buf.SetLineColor(ln, grr.Log1(colors.FromName(DebugBreakColors[DebugBreakInactive])))
	dbg.AddBreak(string(ed.Buf.Filename), ln+1)
}

func (ed *TextEditor) ClearBreakpoint(ln int) {
	if ed.Buf == nil {
		return
	}
	// tv.Buf.DeleteLineIcon(ln)
	ed.Buf.DeleteLineColor(ln)
	dbg, has := ed.CurDebug()
	if !has {
		return
	}
	dbg.DeleteBreak(string(ed.Buf.Filename), ln+1)
}

// HasBreakpoint checks if line has a breakpoint
func (ed *TextEditor) HasBreakpoint(ln int) bool {
	if ed.Buf == nil {
		return false
	}
	_, has := ed.Buf.LineColors[ln]
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
	lx, _ := ed.Buf.HiTagAtPos(tpos)
	if lx == nil {
		return ""
	}
	if !lx.Tok.Tok.InCat(token.Name) {
		return ""
	}
	varNm := ed.Buf.LexObjPathString(tpos.Ln, lx) // get full path
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
	dbg.FindFrames(string(ed.Buf.Filename), ln+1)
}

// DoubleClickEvent processes double-clicks NOT on the line-number section
func (ed *TextEditor) HandleDebugDoubleClick(e events.Event, tpos lex.Pos) {
	dbg, has := ed.CurDebug()
	lx, _ := ed.Buf.HiTagAtPos(tpos)
	if has && lx != nil && lx.Tok.Tok.InCat(token.Name) {
		varNm := ed.Buf.LexObjPathString(tpos.Ln, lx)
		err := dbg.ShowVar(varNm)
		if err == nil {
			e.SetHandled()
			return
		}
	}
	// todo: could do e.g., lookup here, but messes with normal select..
}

// LineNoDoubleClick processes double-clicks on the line-number section
func (ed *TextEditor) LineNoDoubleClick(tpos lex.Pos) {
	ln := tpos.Ln
	ed.ToggleBreakpoint(ln)
	ed.SetNeedsRender(true)
}

// ConfigOutputTextEditor configures a command-output textview within given parent layout
func ConfigOutputTextEditor(ed *texteditor.Editor) {
	ed.Style(func(s *styles.Style) {
		s.Text.WhiteSpace = styles.WhiteSpacePreWrap
		s.Text.TabSize = 8
		s.Font.Family = string(gi.AppearanceSettings.MonoFont)
		s.Min.X.Ch(20)
		s.Min.Y.Em(20)
		s.Grow.Set(1, 1)
		if ed.Buf != nil {
			ed.Buf.Opts.LineNos = false
		}
	})
	ed.SetReadOnly(true)
}

// ConfigEditorTextEditor configures an editor texteditor
func ConfigEditorTextEditor(ed *texteditor.Editor) {
	ed.Style(func(s *styles.Style) {
		s.Min.X.Ch(80)
		s.Min.Y.Em(40)
		s.Font.Family = string(gi.AppearanceSettings.MonoFont)
	})
}

// ContextMenu builds the text editor context menu
func (ed *TextEditor) ContextMenu(m *gi.Scene) {
	gi.NewButton(m).SetText("Copy").SetIcon(icons.ContentCopy).
		SetKey(keyfun.Copy).SetState(!ed.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			ed.Copy(true)
		})
	if ed.IsReadOnly() {
		gi.NewButton(m).SetText("Clear").SetIcon(icons.ClearAll).
			OnClick(func(e events.Event) {
				ed.Clear()
			})
		return
	}

	gi.NewButton(m).SetText("Cut").SetIcon(icons.ContentCopy).
		SetKey(keyfun.Cut).SetState(!ed.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			ed.Cut()
		})
	gi.NewButton(m).SetText("Paste").SetIcon(icons.ContentPaste).
		SetKey(keyfun.Paste).SetState(ed.Clipboard().IsEmpty(), states.Disabled).
		OnClick(func(e events.Event) {
			ed.Paste()
		})

	gi.NewSeparator(m)
	giv.NewFuncButton(m, ed.Lookup).SetIcon(icons.Search)

	fn := ed.Code.FileNodeForFile(string(ed.Buf.Filename), false)
	if fn != nil {
		fn.SelectAction(events.SelectOne)
		fn.VCSContextMenu(m)
	}

	if ed.Code.CurDebug() != nil {
		gi.NewSeparator(m)

		gi.NewButton(m).SetText("Set breakpoint").SetIcon(icons.StopCircle).
			SetTooltip("debugger will stop here").OnClick(func(e events.Event) {
			ed.SetBreakpoint(ed.CursorPos.Ln)
		})
		if ed.HasBreakpoint(ed.CursorPos.Ln) {
			gi.NewButton(m).SetText("Clear breakpoint").SetIcon(icons.Cancel).
				OnClick(func(e events.Event) {
					ed.ClearBreakpoint(ed.CursorPos.Ln)
				})
		}
		gi.NewButton(m).SetText("Debug: Find frames").SetIcon(icons.Cancel).
			SetTooltip("Finds stack frames in the debugger containing this file and line").
			OnClick(func(e events.Event) {
				ed.FindFrames(ed.CursorPos.Ln)
			})
	}
}
