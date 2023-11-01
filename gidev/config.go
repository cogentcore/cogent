// Copyright (c) 2023, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidev

import (
	"fmt"

	"goki.dev/gi/v2/filetree"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/gide/v2/gide"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

func (ge *GideView) ConfigWidget(sc *gi.Scene) {
	ge.ConfigGideView()
}

// Config configures the view
func (ge *GideView) ConfigGideView() {
	if ge.HasChildren() {
		return
	}
	updt := ge.UpdateStart()
	ge.Lay = gi.LayoutVert
	// ge.SetProp("spacing", gi.StdDialogVSpaceUnits)
	gi.NewToolbar(ge, "toolbar")
	gi.NewSplits(ge, "splitview")
	gi.NewFrame(ge, "statusbar").SetLayout(gi.LayoutHoriz)

	ge.UpdateFiles()
	ge.ConfigSplits()
	ge.ConfigToolbar()
	ge.ConfigStatusBar()

	ge.SetStatus("just updated")

	ge.OpenConsoleTab()

	ge.UpdateEndLayout(updt)
}

// IsConfiged returns true if the view is fully configured
func (ge *GideView) IsConfiged() bool {
	if !ge.HasChildren() {
		return false
	}
	sv := ge.Splits()
	if !sv.HasChildren() {
		return false
	}
	return true
}

// Splits returns the main Splits
func (ge *GideView) Splits() *gi.Splits {
	spi := ge.ChildByName("splitview", 2)
	if spi == nil {
		return nil
	}
	return spi.(*gi.Splits)
}

// TextViewByIndex returns the TextView by index (0 or 1), nil if not found
func (ge *GideView) TextViewByIndex(idx int) *gide.TextView {
	split := ge.Splits()
	svk := split.Child(TextView1Idx + idx).Child(0).Child(1)
	return svk.(*gide.TextView)
}

// TextViewButtonByIndex returns the top textview menu button by index (0 or 1)
func (ge *GideView) TextViewButtonByIndex(idx int) *gi.Button {
	split := ge.Splits()
	svk := split.Child(TextView1Idx + idx).Child(0).Child(0)
	return svk.(*gi.Button)
}

// Tabs returns the main TabView
func (ge *GideView) Tabs() *gi.Tabs {
	split := ge.Splits()
	if split == nil {
		return nil
	}
	tv := split.Child(TabsIdx).(*gi.Tabs)
	return tv
}

// Toolbar returns the main toolbar
func (ge *GideView) Toolbar() *gi.Toolbar {
	tbk := ge.ChildByName("toolbar", 2)
	if tbk == nil {
		return nil
	}
	return tbk.(*gi.Toolbar)
}

// StatusBar returns the statusbar widget
func (ge *GideView) StatusBar() *gi.Frame {
	if ge.This() == nil || ge.Is(ki.Deleted) || !ge.HasChildren() {
		return nil
	}
	return ge.ChildByName("statusbar", 2).(*gi.Frame)
}

// StatusLabel returns the statusbar label widget
func (ge *GideView) StatusLabel() *gi.Label {
	return ge.StatusBar().Child(0).(*gi.Label)
}

// ConfigStatusBar configures statusbar with label
func (ge *GideView) ConfigStatusBar() {
	sb := ge.StatusBar()
	if sb == nil || sb.HasChildren() {
		return
	}
	sb.Style(func(s *styles.Style) {
		sb.SetStretchMaxWidth()
		sb.SetMinPrefHeight(units.NewValue(1.2, units.Em))
		sb.SetProp("overflow", "hidden") // no scrollbars!
		sb.SetProp("margin", 0)
		sb.SetProp("padding", 0)
	})
	lbl := sb.NewChild(gi.LabelType, "sb-lbl").(*gi.Label)
	lbl.SetStretchMaxWidth()
	lbl.SetMinPrefHeight(units.NewValue(1, units.Em))
	lbl.SetProp("vertical-align", styles.AlignTop)
	lbl.SetProp("margin", 0)
	lbl.SetProp("padding", 0)
	lbl.SetProp("tab-size", 4)
}

// ConfigToolbar adds a GideView toolbar.
func (ge *GideView) ConfigToolbar() {
	tb := ge.Toolbar()
	if tb.HasChildren() {
		return
	}
	tb.SetStretchMaxWidth()
	giv.ToolbarView(ge, ge.Viewport, tb)
	gi.NewSeparator(tb, "sepmod")
	sm := tb.NewChild(gi.SwitchType, "go-mod").(*gi.Switch)
	sm.SetChecked(ge.Prefs.GoMod)
	sm.SetText("Go Mod")
	sm.Tooltip = "Toggles the use of go modules -- saved with project -- if off, uses old school GOPATH mode"
	sm.ButtonSig.Connect(ge.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(gi.ButtonToggled) {
			cb := send.(*gi.Switch)
			ge.Prefs.GoMod = cb.IsChecked()
			gide.SetGoMod(ge.Prefs.GoMod)
		}
	})
}

var fnFolderProps = ki.Props{
	"icon":     "folder-open",
	"icon-off": "folder",
}

// ConfigSplits configures the Splits.
func (ge *GideView) ConfigSplits() {
	split := ge.Splits()
	split.Dim = mat32.X
	if split.HasChildren() {
		return
	}
	updt := split.UpdateStart()
	ftfr := gi.NewFrame(split, "filetree", gi.LayoutVert)
	ftfr.SetReRenderAnchor()
	ft := ftfr.NewChild(gide.KiT_FileTreeView, "filetree").(*gide.FileTreeView)
	ft.SetFlag(int(giv.TreeViewFlagUpdtRoot)) // filetree needs this
	ft.OpenDepth = 4
	ge.FilesView = ft
	ft.SetRootNode(&ge.Files)
	ft.TreeViewSig.Connect(ge.This(), func(recv, send ki.Ki, sig int64, data any) {
		if data == nil {
			return
		}
		tvn, _ := data.(ki.Ki).Embed(gide.KiT_FileTreeView).(*gide.FileTreeView)
		gee, _ := recv.Embed(KiT_GideView).(*GideView)
		if tvn.SrcNode != nil {
			fn := tvn.SrcNode.Embed(giv.KiT_FileNode).(*filetree.Node)
			switch sig {
			case int64(giv.TreeViewSelected):
				gee.FileNodeSelected(fn, tvn)
			case int64(giv.TreeViewOpened):
				gee.FileNodeOpened(fn, tvn)
			case int64(giv.TreeViewClosed):
				gee.FileNodeClosed(fn, tvn)
			}
		}
	})

	for i := 0; i < NTextViews; i++ {
		txnm := fmt.Sprintf("%d", i)
		txly := gi.NewLayout(split, "textlay-"+txnm, gi.LayoutVert)
		txly.SetStretchMaxWidth()
		txly.SetStretchMaxHeight()
		txly.SetReRenderAnchor() // anchor here: Splits will only anchor Frame, but we just have layout

		// need to sandbox the button in its own layer to isolate FullReRender issues
		txbly := gi.NewLayout(txly, "butlay-"+txnm, gi.LayoutVert)
		txbly.SetProp("spacing", units.NewEm(0))
		txbly.SetStretchMaxWidth()
		txbly.SetReRenderAnchor() // anchor here!

		txbut := gi.NewMenuButton(txbly, "textbut-"+txnm)
		txbut.SetStretchMaxWidth()
		txbut.SetText("textview: " + txnm)
		txbut.MakeMenuFunc = ge.TextViewButtonMenu
		txbut.ButtonSig.Connect(ge.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.ButtonClicked) {
				gee, _ := recv.Embed(KiT_GideView).(*GideView)
				idx := 0
				nm := send.Name()
				nln := len(nm)
				if nm[nln-1] == '1' {
					idx = 1
				}
				gee.SetActiveTextViewIdx(idx)
			}
		})

		txily := gi.NewLayout(txly, "textilay-"+txnm, gi.LayoutVert)
		txily.SetStretchMaxWidth()
		txily.SetStretchMaxHeight()
		txily.SetMinPrefWidth(units.NewCh(80))
		txily.SetMinPrefHeight(units.NewEm(40))

		ted := gide.NewTextView(txily, "textview-"+txnm)
		ted.TextViewSig.Connect(ge.This(), func(recv, send ki.Ki, sig int64, data any) {
			gee, _ := recv.Embed(KiT_GideView).(*GideView)
			tee := send.Embed(gide.KiT_TextView).(*gide.TextView)
			gee.TextViewSig(tee, texteditor.EditorSignals(sig))
		})
	}

	ge.ConfigTextViews()
	ge.UpdateTextButtons()

	mtab := gi.NewTabView(split, "tabs")
	mtab.TabViewSig.Connect(ge.This(), func(recv, send ki.Ki, sig int64, data any) {
		gee, _ := recv.Embed(KiT_GideView).(*GideView)
		tvsig := gi.TabsSignals(sig)
		switch tvsig {
		case gi.TabDeleted:
			gee.TabDeleted(data.(string))
			if data == "Find" {
				ge.ActiveTextView().ClearHighlights()
			}
		}
	})

	split.SetSplits(ge.Prefs.Splits...)
	split.UpdateEnd(updt)
}

// ConfigTextViews configures text views according to current settings
func (ge *GideView) ConfigTextViews() {
	for i := 0; i < NTextViews; i++ {
		tv := ge.TextViewByIndex(i)
		if ge.Prefs.Editor.WordWrap {
			tv.SetProp("white-space", styles.WhiteSpacePreWrap)
		} else {
			tv.SetProp("white-space", styles.WhiteSpacePre)
		}
		tv.SetProp("tab-size", ge.Prefs.Editor.TabSize)
		tv.SetProp("font-family", gi.Prefs.MonoFont)
	}
}
