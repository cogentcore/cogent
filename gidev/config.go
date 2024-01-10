// Copyright (c) 2023, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidev

import (
	"fmt"

	"goki.dev/gi/v2/gi"
	"goki.dev/gide/v2/gide"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/gix/filetree"
	"goki.dev/goosi/events"
	"goki.dev/ki/v2"
)

// NTextEditors is the number of text views to create -- to keep things simple
// and consistent (e.g., splitter settings always have the same number of
// values), we fix this degree of freedom, and have flexibility in the
// splitter settings for what to actually show.
const NTextEditors = 2

// These are then the fixed indices of the different elements in the splitview
const (
	FileTreeIdx = iota
	TextEditor1Idx
	TextEditor2Idx
	TabsIdx
)

func (ge *GideView) ConfigWidget() {
	ge.ConfigGideView()
}

// Config configures the view
func (ge *GideView) ConfigGideView() {
	if ge.HasChildren() {
		return
	}

	updt := ge.UpdateStart()
	defer ge.UpdateEndLayout(updt)

	ge.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	// ge.SetProp("spacing", gi.StdDialogVSpaceUnits)
	gi.NewSplits(ge, "splitview")
	gi.NewFrame(ge, "statusbar")

	ge.ConfigSplits()
	ge.ConfigStatusBar()

	ge.SetStatus("just updated")

	ge.OpenConsoleTab()
	ge.UpdateFiles()
}

// IsConfiged returns true if the view is configured
func (ge *GideView) IsConfiged() bool {
	return ge.HasChildren()
}

// Splits returns the main Splits
func (ge *GideView) Splits() *gi.Splits {
	return ge.ChildByName("splitview", 2).(*gi.Splits)
}

// TextEditorButtonByIndex returns the top textview menu button by index (0 or 1)
func (ge *GideView) TextEditorButtonByIndex(idx int) *gi.Button {
	return ge.Splits().Child(TextEditor1Idx + idx).Child(0).(*gi.Button)
}

// TextEditorByIndex returns the TextEditor by index (0 or 1), nil if not found
func (ge *GideView) TextEditorByIndex(idx int) *gide.TextEditor {
	return ge.Splits().Child(TextEditor1Idx + idx).Child(1).(*gide.TextEditor)
}

// Tabs returns the main TabView
func (ge *GideView) Tabs() *gi.Tabs {
	return ge.Splits().Child(TabsIdx).(*gi.Tabs)
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

// SelectedFileNode returns currently selected file tree node as a *filetree.Node
// could be nil.
func (ge *GideView) SelectedFileNode() *filetree.Node {
	n := len(ge.Files.SelectedNodes)
	if n == 0 {
		return nil
	}
	return filetree.AsNode(ge.Files.SelectedNodes[n-1].This())
}

// ConfigSplits configures the Splits.
func (ge *GideView) ConfigSplits() {
	// note: covered by global update
	split := ge.Splits()
	ftfr := gi.NewFrame(split, "filetree")
	ftfr.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Overflow.Set(styles.OverflowAuto)
	})
	ft := filetree.NewTree(ftfr, "filetree")
	ft.OpenDepth = 4
	ge.Files = ft
	ft.NodeType = gide.FileNodeType

	ge.Files.OnSelect(func(e events.Event) {
		e.SetHandled()
		sn := ge.SelectedFileNode()
		if sn != nil {
			ge.FileNodeSelected(sn)
		}
	})
	ge.Files.OnDoubleClick(func(e events.Event) {
		e.SetHandled()
		sn := ge.SelectedFileNode()
		if sn != nil {
			ge.FileNodeOpened(sn)
		}
	})

	for i := 0; i < NTextEditors; i++ {
		i := i
		txnm := fmt.Sprintf("%d", i)
		txly := gi.NewLayout(split, "textlay-"+txnm)
		txly.Style(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Grow.Set(1, 1)
		})
		txbut := gi.NewButton(txly, "textbut-"+txnm).SetText("textview: " + txnm)
		txbut.Type = gi.ButtonAction
		txbut.Style(func(s *styles.Style) {
			s.Grow.Set(1, 0)
		})
		txbut.Menu = func(m *gi.Scene) {
			ge.TextEditorButtonMenu(i, m)
		}
		txbut.OnClick(func(e events.Event) {
			ge.SetActiveTextEditorIdx(i)
		})

		ted := gide.NewTextEditor(txly, "textview-"+txnm)
		ted.Gide = ge
		gide.ConfigEditorTextEditor(&ted.Editor)
		ted.OnFocus(func(e events.Event) {
			ge.ActiveTextEditorIdx = i
		})
		// get updates on cursor movement and qreplace
		ted.OnInput(func(e events.Event) {
			ge.UpdateStatusLabel()
		})
	}

	ge.UpdateTextButtons()

	mtab := gi.NewTabs(split, "tabs").SetType(gi.FunctionalTabs)
	mtab.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})

	// mtab.OnChange(func(e events.Event) {
	// todo: need to monitor deleted
	// gee.TabDeleted(data.(string))
	// if data == "Find" {
	// 	ge.ActiveTextEditor().ClearHighlights()
	// }
	// })

	split.SetSplits(ge.Prefs.Splits...)
}

// ConfigStatusBar configures statusbar with label
func (ge *GideView) ConfigStatusBar() {
	sb := ge.StatusBar()
	sb.Style(func(s *styles.Style) {
		s.Grow.Set(1, 0)
		s.Min.Y.Em(1.0)
		s.Margin.Zero()
		s.Padding.Set(units.Dp(4))
	})
	lbl := gi.NewLabel(sb, "sb-lbl").SetText("This is the status bar initial configuration.  Welcome to gide!")
	lbl.Style(func(s *styles.Style) {
		s.Min.X.Ch(100)
		s.Min.Y.Em(1.0)
		s.Margin.Zero()
		s.Padding.Zero()
		s.Text.TabSize = 4
	})
}
