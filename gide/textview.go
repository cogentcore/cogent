package gide

import (
	"fmt"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

type TextView struct {
	giv.TextView
}

var KiT_TextView = kit.Types.AddType(&TextView{}, TextViewProps)

var TextViewProps = ki.Props{
	"EnumType:Flag":    giv.KiT_TextViewFlags,
	"white-space":      gi.WhiteSpacePreWrap,
	"font-family":      "Go Mono",
	"border-width":     0, // don't render our own border
	"cursor-width":     units.NewValue(3, units.Px),
	"border-color":     &gi.Prefs.Colors.Border,
	"border-style":     gi.BorderSolid,
	"padding":          units.NewValue(2, units.Px),
	"margin":           units.NewValue(2, units.Px),
	"vertical-align":   gi.AlignTop,
	"text-align":       gi.AlignLeft,
	"tab-size":         4,
	"color":            &gi.Prefs.Colors.Font,
	"background-color": &gi.Prefs.Colors.Background,
	giv.TextViewSelectors[giv.TextViewActive]: ki.Props{
		"background-color": "highlight-10",
	},
	giv.TextViewSelectors[giv.TextViewFocus]: ki.Props{
		"background-color": "lighter-0",
	},
	giv.TextViewSelectors[giv.TextViewInactive]: ki.Props{
		"background-color": "highlight-20",
	},
	giv.TextViewSelectors[giv.TextViewSel]: ki.Props{
		"background-color": &gi.Prefs.Colors.Select,
	},
	giv.TextViewSelectors[giv.TextViewHighlight]: ki.Props{
		"background-color": &gi.Prefs.Colors.Highlight,
	},
}

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
		ac = m.AddAction(gi.ActOpts{Label: "Declaration"},
			tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				txf := recv.Embed(KiT_TextView).(*TextView)
				txf.Declaration()
			})
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
				txf.SetBreakpoint()
			})
		ac.SetActiveState(tv.HasSelection() && hasDbg)
		ac = m.AddAction(gi.ActOpts{Label: "ClearBreakpoint"},
			tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				txf := recv.Embed(KiT_TextView).(*TextView)
				txf.ClearBreakpoint()
			})
		ac.SetActiveState(tv.HasSelection() && hasDbg)
	} else {
		ac = m.AddAction(gi.ActOpts{Label: "Clear"},
			tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				txf := recv.Embed(KiT_TextView).(*TextView)
				txf.Clear()
			})
	}
}

func (tv *TextView) Declaration() {
	fmt.Println("Go to Declaration: not yet implemented")
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
	// if ge.Prefs.Editor.WordWrap {
	tv.SetProp("white-space", gi.WhiteSpacePreWrap)
	// } else {
	// 	tv.SetProp("white-space", gi.WhiteSpacePre)
	// }
	tv.SetProp("tab-size", 8) // std for output
	tv.SetProp("font-family", Prefs.FontFamily)
	tv.SetInactive()
	return tv
}

func (tv *TextView) SetBreakpoint() {
	ge, ok := ParentGide(tv)
	if !ok {
		return
	}
	dbg := ge.CurDebug()
	if dbg == nil {
		return
	}
	sel := tv.Selection()
	ln := sel.Reg.Start.Ln
	tv.Buf.SetLineIcon(ln, "stop")
	dbg.SetBreak(string(tv.Buf.Filename), ln+1)
}

func (tv *TextView) ClearBreakpoint() {
}
