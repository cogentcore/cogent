// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"image"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/grr"
	"goki.dev/pi/v2/lex"
	"goki.dev/pi/v2/token"
)

// TextView is the Gide-specific version of the TextView, with support for
// setting / clearing breakpoints, etc
type TextView struct {
	texteditor.Editor
}

/*
// MakeContextMenu builds the textview context menu
func (tv *TextView) MakeContextMenu(m *gi.Scene) {
	ac := m.AddAction(gi.ActOpts{Label: "Copy", ShortcutKey: keyfun.Copy},
		tv.This(), func(recv, send ki.Ki, sig int64, data any) {
			txf := recv.Embed(KiT_TextView).(*TextView)
			txf.Copy(true)
		})
	ac.SetActiveState(tv.HasSelection())
	if !tv.IsInactive() {
		ac = m.AddAction(gi.ActOpts{Label: "Cut", ShortcutKey: keyfun.Cut},
			tv.This(), func(recv, send ki.Ki, sig int64, data any) {
				txf := recv.Embed(KiT_TextView).(*TextView)
				txf.Cut()
			})
		ac.SetActiveState(tv.HasSelection())
		ac = m.AddAction(gi.ActOpts{Label: "Paste", ShortcutKey: keyfun.Paste},
			tv.This(), func(recv, send ki.Ki, sig int64, data any) {
				txf := recv.Embed(KiT_TextView).(*TextView)
				txf.Paste()
			})
		ac.SetActiveState(tv.HasSelection() && !tv.Buf.InComment(tv.CursorPos))

		m.AddSeparator("sep-clip")

		ac = m.AddAction(gi.ActOpts{Label: "Lookup", ShortcutKey: keyfun.Lookup},
			tv.This(), func(recv, send ki.Ki, sig int64, data any) {
				txf := recv.Embed(KiT_TextView).(*TextView)
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
				txf := recv.Embed(KiT_TextView).(*TextView)
				txf.SetBreakpoint(tv.CursorPos.Ln)
			})
		ac.SetActiveState(hasDbg)
		ac = m.AddAction(gi.ActOpts{Label: "ClearBreakpoint"},
			tv.This(), func(recv, send ki.Ki, sig int64, data any) {
				txf := recv.Embed(KiT_TextView).(*TextView)
				txf.ClearBreakpoint(tv.CursorPos.Ln)
			})
		ac.SetActiveState(hasDbg && tv.HasBreakpoint(tv.CursorPos.Ln))
		ac = m.AddAction(gi.ActOpts{Label: "Debug: Find Frames"},
			tv.This(), func(recv, send ki.Ki, sig int64, data any) {
				txf := recv.Embed(KiT_TextView).(*TextView)
				txf.FindFrames(tv.CursorPos.Ln)
			})
		ac.SetActiveState(hasDbg)
	} else {
		ac = m.AddAction(gi.ActOpts{Label: "Clear"},
			tv.This(), func(recv, send ki.Ki, sig int64, data any) {
				txf := recv.Embed(KiT_TextView).(*TextView)
				txf.Clear()
			})
	}
}
*/

// func (tv *TextView) FocusChanged2D(change gi.FocusChanges) {
// 	tv.TextView.FocusChanged2D(change)
// 	ge, ok := ParentGide(tv)
// 	if ok {
// 		if change == gi.FocusGot || change == gi.FocusActive {
// 			ge.SetActiveTextView(tv)
// 		}
// 	}
// }

// CurDebug returns the current debugger, true if it is present
func (tv *TextView) CurDebug() (*DebugView, bool) {
	if tv.Buf == nil {
		return nil, false
	}
	if ge, ok := ParentGide(tv); ok {
		dbg := ge.CurDebug()
		if dbg != nil {
			return dbg, true
		}
	}
	return nil, false
}

// SetBreakpoint sets breakpoint at given line (e.g., tv.CursorPos.Ln)
func (tv *TextView) SetBreakpoint(ln int) {
	dbg, has := tv.CurDebug()
	if !has {
		return
	}
	// tv.Buf.SetLineIcon(ln, "stop")
	tv.Buf.SetLineColor(ln, grr.Log(colors.FromName(DebugBreakColors[DebugBreakInactive])))
	dbg.AddBreak(string(tv.Buf.Filename), ln+1)
}

func (tv *TextView) ClearBreakpoint(ln int) {
	if tv.Buf == nil {
		return
	}
	// tv.Buf.DeleteLineIcon(ln)
	tv.Buf.DeleteLineColor(ln)
	dbg, has := tv.CurDebug()
	if !has {
		return
	}
	dbg.DeleteBreak(string(tv.Buf.Filename), ln+1)
}

// HasBreakpoint checks if line has a breakpoint
func (tv *TextView) HasBreakpoint(ln int) bool {
	if tv.Buf == nil {
		return false
	}
	_, has := tv.Buf.LineColors[ln]
	return has
}

func (tv *TextView) ToggleBreakpoint(ln int) {
	if tv.HasBreakpoint(ln) {
		tv.ClearBreakpoint(ln)
	} else {
		tv.SetBreakpoint(ln)
	}
}

// DebugVarValueAtPos returns debugger variable value for given mouse position
func (tv *TextView) DebugVarValueAtPos(pos image.Point) string {
	dbg, has := tv.CurDebug()
	if !has {
		return ""
	}
	pt := tv.PointToRelPos(pos)
	tpos := tv.PixelToCursor(pt)
	lx, _ := tv.Buf.HiTagAtPos(tpos)
	if lx == nil {
		return ""
	}
	if !lx.Tok.Tok.InCat(token.Name) {
		return ""
	}
	varNm := tv.Buf.LexObjPathString(tpos.Ln, lx) // get full path
	val := dbg.VarValue(varNm)
	if val != "" {
		return varNm + " = " + val
	}
	return ""
}

// FindFrames finds stack frames in the debugger containing this file and line
func (tv *TextView) FindFrames(ln int) {
	dbg, has := tv.CurDebug()
	if !has {
		return
	}
	dbg.FindFrames(string(tv.Buf.Filename), ln+1)
}

// LineNoDoubleClick processes double-clicks on the line-number section
func (tv *TextView) LineNoDoubleClick(tpos lex.Pos) {
	ln := tpos.Ln
	tv.ToggleBreakpoint(ln)
	tv.RenderLines(ln, ln)
}

// DoubleClickEvent processes double-clicks NOT on the line-number section
func (tv *TextView) DoubleClickEvent(tpos lex.Pos) {
	dbg, has := tv.CurDebug()
	lx, _ := tv.Buf.HiTagAtPos(tpos)
	if has && lx != nil && lx.Tok.Tok.InCat(token.Name) {
		varNm := tv.Buf.LexObjPathString(tpos.Ln, lx)
		err := dbg.ShowVar(varNm)
		if err == nil {
			return
		}
	}
	// todo: could do e.g., lookup here, but messes with normal select..
}

/*
// MouseEvent handles the mouse.Event
func (tv *TextView) MouseEvent(me *mouse.Event) {
	if me.Button != mouse.Left || me.Action != mouse.DoubleClick {
		tv.TextView.MouseEvent(me)
		return
	}
	if tv.Buf == nil {
		return
	}
	pt := tv.PointToRelPos(me.Pos())
	tpos := tv.PixelToCursor(pt)
	if pt.X >= 0 && tv.Buf.IsValidLine(tpos.Ln) {
		if pt.X < int(tv.LineNoOff) {
			me.SetProcessed()
			tv.LineNoDoubleClick(tpos)
			return
		}
		me.SetProcessed()
		tv.DoubleClickEvent(tpos)
	}
	tv.TextView.MouseEvent(me)
}

func (tv *TextView) HoverEvent() {
	tv.ConnectEvent(oswin.MouseHoverEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.HoverEvent)
		txf := recv.Embed(KiT_TextView).(*TextView)
		tt := ""
		vv := tv.DebugVarValueAtPos(me.Pos())
		if vv != "" {
			tt = vv
		}
		if tt != "" {
			me.SetProcessed()
			pos := me.Pos()
			pos.X += 20
			pos.Y += 20
			gi.PopupTooltip(tt, pos.X, pos.Y, txf.Viewport, txf.Nm)
		}
	})
}
*/

// TextViewEvents sets connections between mouse and key events and actions
func (tv *TextView) TextViewEvents() {
	/*
		tv.HoverEvent()
		tv.MouseMoveEvent()
		tv.MouseDragEvent()
			tv.ConnectEvent(oswin.MouseEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
				txf := recv.Embed(KiT_TextView).(*TextView)
				me := d.(*mouse.Event)
				txf.MouseEvent(me) // gets our new one
			})
			tv.MouseFocusEvent()
			tv.ConnectEvent(oswin.KeyChordEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
				txf := recv.Embed(KiT_TextView).(*TextView)
				kt := d.(*key.ChordEvent)
				txf.KeyInput(kt)
			})
	*/
}

// ConnectEvents2D indirectly sets connections between mouse and key events and actions
func (tv *TextView) ConnectEvents2D() {
	tv.TextViewEvents()
}

// ConfigOutputTextView configures a command-output textview within given parent layout
func ConfigOutputTextView(tv *texteditor.Editor) {
	tv.SetMinPrefWidth(units.Ch(20))
	tv.SetMinPrefHeight(units.Ch(10))
	tv.SetFlag(false, texteditor.EditorHasLineNos)
	tv.Style(func(s *styles.Style) {
		s.Text.WhiteSpace = styles.WhiteSpacePreWrap
		s.Text.TabSize = 8
		s.Font.Family = string(gi.Prefs.MonoFont)
	})
	tv.SetState(true, states.ReadOnly)

}
