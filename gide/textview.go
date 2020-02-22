package gide

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// TextView is the Gide-specific version of the TextView, with support for
// setting / clearing breakpoints, etc
type TextView struct {
	giv.TextView
}

var KiT_TextView = kit.Types.AddType(&TextView{}, giv.TextViewProps)

// MakeContextMenu builds the textview context menu
func (tv *TextView) MakeContextMenu(m *gi.Menu) {
	ac := m.AddAction(gi.ActOpts{Label: "Copy", ShortcutKey: gi.KeyFunCopy},
		tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			txf := recv.Embed(KiT_TextView).(*TextView)
			txf.Copy(true)
		})
	ac.SetActiveState(tv.HasSelection())
	if !tv.IsInactive() {
		ac = m.AddAction(gi.ActOpts{Label: "Cut", ShortcutKey: gi.KeyFunCut},
			tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				txf := recv.Embed(KiT_TextView).(*TextView)
				txf.Cut()
			})
		ac.SetActiveState(tv.HasSelection())
		ac = m.AddAction(gi.ActOpts{Label: "Paste", ShortcutKey: gi.KeyFunPaste},
			tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				txf := recv.Embed(KiT_TextView).(*TextView)
				txf.Paste()
			})
		m.AddSeparator("sep-tvmenu")
		ac.SetActiveState(tv.HasSelection() && !tv.Buf.InComment(tv.CursorPos))
		hasDbg := false
		if ge, ok := ParentGide(tv); ok {
			if ge.CurDebug() != nil {
				hasDbg = true
			}
		}
		ac = m.AddAction(gi.ActOpts{Label: "SetBreakpoint"},
			tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				txf := recv.Embed(KiT_TextView).(*TextView)
				txf.SetBreakpoint(tv.CursorPos.Ln)
			})
		ac.SetActiveState(hasDbg)
		ac = m.AddAction(gi.ActOpts{Label: "ClearBreakpoint"},
			tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				txf := recv.Embed(KiT_TextView).(*TextView)
				txf.ClearBreakpoint(tv.CursorPos.Ln)
			})
		ac.SetActiveState(hasDbg && tv.HasBreakpoint(tv.CursorPos.Ln))
		ac = m.AddAction(gi.ActOpts{Label: "Debug: Find Frames"},
			tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				txf := recv.Embed(KiT_TextView).(*TextView)
				txf.FindFrames(tv.CursorPos.Ln)
			})
		ac.SetActiveState(hasDbg)
	} else {
		ac = m.AddAction(gi.ActOpts{Label: "Clear"},
			tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				txf := recv.Embed(KiT_TextView).(*TextView)
				txf.Clear()
			})
	}
}

func (tv *TextView) FocusChanged2D(change gi.FocusChanges) {
	tv.TextView.FocusChanged2D(change)
	ge, ok := ParentGide(tv)
	if ok {
		if change == gi.FocusGot || change == gi.FocusActive {
			ge.SetActiveTextView(tv)
		}
	}
}

// CurDebug returns the current debugger, true if it is present
func (tv *TextView) CurDebug() (*DebugView, bool) {
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
	tv.Buf.SetLineIcon(ln, "stop")
	tv.Buf.SetLineColor(ln, "red")
	dbg.AddBreak(string(tv.Buf.Filename), ln+1)
}

func (tv *TextView) ClearBreakpoint(ln int) {
	tv.Buf.DeleteLineIcon(ln)
	tv.Buf.DeleteLineColor(ln)
	dbg, has := tv.CurDebug()
	if !has {
		return
	}
	dbg.DeleteBreak(string(tv.Buf.Filename), ln+1)
}

// HasBreakpoint checks if line has a breakpoint
func (tv *TextView) HasBreakpoint(ln int) bool {
	// todo: use debugger instead of local info..
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

// FindFrames finds stack frames in the debugger containing this file and line
func (tv *TextView) FindFrames(ln int) {
	dbg, has := tv.CurDebug()
	if !has {
		return
	}
	dbg.FindFrames(string(tv.Buf.Filename), ln+1)
}

// MouseEvent handles the mouse.Event
func (tv *TextView) MouseEvent(me *mouse.Event) {
	if me.Button != mouse.Left || me.Action != mouse.DoubleClick {
		tv.TextView.MouseEvent(me)
		return
	}
	pt := tv.PointToRelPos(me.Pos())
	if pt.X >= 0 && pt.X < int(tv.LineNoOff) {
		newPos := tv.PixelToCursor(pt)
		ln := newPos.Ln
		tv.ToggleBreakpoint(ln)
		tv.RenderLines(ln, ln)
		me.SetProcessed()
		return
	}
	tv.TextView.MouseEvent(me)
}

// TextViewEvents sets connections between mouse and key events and actions
func (tv *TextView) TextViewEvents() {
	tv.HoverTooltipEvent()
	tv.MouseMoveEvent()
	tv.MouseDragEvent()
	tv.ConnectEvent(oswin.MouseEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		txf := recv.Embed(KiT_TextView).(*TextView)
		me := d.(*mouse.Event)
		txf.MouseEvent(me) // gets our new one
	})
	tv.MouseFocusEvent()
	tv.ConnectEvent(oswin.KeyChordEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		txf := recv.Embed(KiT_TextView).(*TextView)
		kt := d.(*key.ChordEvent)
		txf.KeyInput(kt)
	})
}

// ConnectEvents2D indirectly sets connections between mouse and key events and actions
func (tv *TextView) ConnectEvents2D() {
	tv.TextViewEvents()
}

// ConfigOutputTextView configures a command-output textview within given parent layout
func ConfigOutputTextView(ly *gi.Layout) *giv.TextView {
	ly.Lay = gi.LayoutVert
	ly.SetStretchMaxWidth()
	ly.SetStretchMaxHeight()
	ly.SetMinPrefWidth(units.NewValue(20, units.Ch))
	ly.SetMinPrefHeight(units.NewValue(10, units.Ch))
	var tv *giv.TextView
	if ly.HasChildren() {
		tv = ly.Child(0).Embed(giv.KiT_TextView).(*giv.TextView)
	} else {
		tv = ly.AddNewChild(giv.KiT_TextView, ly.Nm).(*giv.TextView)
	}
	tv.SetProp("line-nos", false)
	// if ge.Prefs.Editor.WordWrap {
	tv.SetProp("white-space", gi.WhiteSpacePreWrap)
	// } else {
	// 	tv.SetProp("white-space", gi.WhiteSpacePre)
	// }
	tv.SetProp("tab-size", 8) // std for output
	tv.SetProp("font-family", gi.Prefs.MonoFont)
	tv.SetInactive()
	return tv
}
