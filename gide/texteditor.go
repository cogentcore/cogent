// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"fmt"
	"image"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/grr"
	"goki.dev/pi/v2/lex"
	"goki.dev/pi/v2/token"
)

// TextEditor is the Gide-specific version of the TextEditor, with support for
// setting / clearing breakpoints, etc
type TextEditor struct {
	texteditor.Editor

	Gide Gide
}

func (ed *TextEditor) OnInit() {
	ed.HandleGideEvents()
	ed.EditorStyles()
}

// TextEditorEvents sets connections between mouse and key events and actions
func (ed *TextEditor) HandleGideEvents() {
	ed.HandleEditorEvents()
	ed.HandleGideDoubleClick()
	ed.HandleGideDebugHover()
}

// func (tv *TextEditor) FocusChanged2D(change gi.FocusChanges) {
// 	tv.TextEditor.FocusChanged2D(change)
// 	ge, ok := ParentGide(tv)
// 	if ok {
// 		if change == gi.FocusGot || change == gi.FocusActive {
// 			ge.SetActiveTextEditor(tv)
// 		}
// 	}
// }

// CurDebug returns the current debugger, true if it is present
func (ed *TextEditor) CurDebug() (*DebugView, bool) {
	if ed.Buf == nil {
		return nil, false
	}
	if ge, ok := ParentGide(ed); ok {
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
	ed.RenderLines(ln, ln)
}

func (ed *TextEditor) HandleGideDoubleClick() {
	fmt.Println("setting double click")
	ed.OnDoubleClick(func(e events.Event) {
		pt := ed.PointToRelPos(e.LocalPos())
		tpos := ed.PixelToCursor(pt)
		fmt.Println("in double click", tpos)
		if pt.X >= 0 && ed.Buf.IsValidLine(tpos.Ln) {
			if pt.X < int(ed.LineNoOff) {
				e.SetHandled()
				ed.LineNoDoubleClick(tpos)
				return
			}
			ed.HandleDebugDoubleClick(e, tpos)
		}
	})
}

func (ed *TextEditor) HandleGideDebugHover() {
	ed.On(events.LongHoverStart, func(e events.Event) {
		tt := ""
		vv := ed.DebugVarValueAtPos(e.LocalPos())
		if vv != "" {
			tt = vv
		}
		if tt != "" {
			e.SetHandled()
			pos := e.LocalPos()
			pos.X += 20
			pos.Y += 20
			gi.NewTooltipText(ed, tt, ed.WinBBox().Min).Run()
		}
	})
}

// ConfigOutputTextEditor configures a command-output textview within given parent layout
func ConfigOutputTextEditor(ed *texteditor.Editor) {
	ed.Style(func(s *styles.Style) {
		s.Text.WhiteSpace = styles.WhiteSpacePreWrap
		s.Text.TabSize = 8
		s.Font.Family = string(gi.Prefs.MonoFont)
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
		s.Font.Family = string(gi.Prefs.MonoFont)
	})
}

/*
// MakeContextMenu builds the textview context menu
func (tv *TextEditor) MakeContextMenu(m *gi.Scene) {
	ac := m.AddAction(gi.ActOpts{Label: "Copy", ShortcutKey: keyfun.Copy},
		tv.This(), func(recv, send ki.Ki, sig int64, data any) {
			txf := recv.Embed(KiT_TextEditor).(*TextEditor)
			txf.Copy(true)
		})
	ac.SetActiveState(tv.HasSelection())
	if !tv.IsInactive() {
		ac = m.AddAction(gi.ActOpts{Label: "Cut", ShortcutKey: keyfun.Cut},
			tv.This(), func(recv, send ki.Ki, sig int64, data any) {
				txf := recv.Embed(KiT_TextEditor).(*TextEditor)
				txf.Cut()
			})
		ac.SetActiveState(tv.HasSelection())
		ac = m.AddAction(gi.ActOpts{Label: "Paste", ShortcutKey: keyfun.Paste},
			tv.This(), func(recv, send ki.Ki, sig int64, data any) {
				txf := recv.Embed(KiT_TextEditor).(*TextEditor)
				txf.Paste()
			})
		ac.SetActiveState(tv.HasSelection() && !tv.Buf.InComment(tv.CursorPos))

		m.AddSeparator("sep-clip")

		ac = m.AddAction(gi.ActOpts{Label: "Lookup", ShortcutKey: keyfun.Lookup},
			tv.This(), func(recv, send ki.Ki, sig int64, data any) {
				txf := recv.Embed(KiT_TextEditor).(*TextEditor)
				txf.Lookup()
			})

		m.AddSeparator("sep-dbg")
		hasDbg := false
		if ge, ok := ParentGide(tv); ok {
			if ge.CurDebug() != nil {
				hasDbg = true
			}
		}
		ac = m.AddAction(gi.ActOpts{Label: "SetBreakpoint"},
			tv.This(), func(recv, send ki.Ki, sig int64, data any) {
				txf := recv.Embed(KiT_TextEditor).(*TextEditor)
				txf.SetBreakpoint(tv.CursorPos.Ln)
			})
		ac.SetActiveState(hasDbg)
		ac = m.AddAction(gi.ActOpts{Label: "ClearBreakpoint"},
			tv.This(), func(recv, send ki.Ki, sig int64, data any) {
				txf := recv.Embed(KiT_TextEditor).(*TextEditor)
				txf.ClearBreakpoint(tv.CursorPos.Ln)
			})
		ac.SetActiveState(hasDbg && tv.HasBreakpoint(tv.CursorPos.Ln))
		ac = m.AddAction(gi.ActOpts{Label: "Debug: Find Frames"},
			tv.This(), func(recv, send ki.Ki, sig int64, data any) {
				txf := recv.Embed(KiT_TextEditor).(*TextEditor)
				txf.FindFrames(tv.CursorPos.Ln)
			})
		ac.SetActiveState(hasDbg)
	} else {
		ac = m.AddAction(gi.ActOpts{Label: "Clear"},
			tv.This(), func(recv, send ki.Ki, sig int64, data any) {
				txf := recv.Embed(KiT_TextEditor).(*TextEditor)
				txf.Clear()
			})
	}
}
*/
