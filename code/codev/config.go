// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codev

import (
	"fmt"

	"cogentcore.org/cogent/code/code"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// NTextEditors is the number of text views to create -- to keep things simple
// and consistent (e.g., splitter settings always have the same number of
// values), we fix this degree of freedom, and have flexibility in the
// splitter settings for what to actually show.
const NTextEditors = 2

// These are then the fixed indices of the different elements in the splitview
const (
	FileTreeIndex = iota
	TextEditor1Index
	TextEditor2Index
	TabsIndex
)

func (ge *CodeView) Config() {
	ge.ConfigCodeView()
}

// Config configures the view
func (ge *CodeView) ConfigCodeView() {
	if ge.HasChildren() {
		return
	}

	ge.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	// ge.SetProp("spacing", gi.StdDialogVSpaceUnits)
	core.NewSplits(ge, "splitview")
	core.NewFrame(ge, "statusbar")

	ge.ConfigSplits()
	ge.ConfigStatusBar()

	ge.SetStatus("just updated")

	ge.OpenConsoleTab()
	ge.UpdateFiles()
	ge.NeedsLayout()
}

// IsConfiged returns true if the view is configured
func (ge *CodeView) IsConfiged() bool {
	return ge.HasChildren()
}

// Splits returns the main Splits
func (ge *CodeView) Splits() *core.Splits {
	return ge.ChildByName("splitview", 2).(*core.Splits)
}

// TextEditorButtonByIndex returns the top texteditor menu button by index (0 or 1)
func (ge *CodeView) TextEditorButtonByIndex(idx int) *core.Button {
	return ge.Splits().Child(TextEditor1Index + idx).Child(0).(*core.Button)
}

// TextEditorByIndex returns the TextEditor by index (0 or 1), nil if not found
func (ge *CodeView) TextEditorByIndex(idx int) *code.TextEditor {
	return ge.Splits().Child(TextEditor1Index + idx).Child(1).(*code.TextEditor)
}

// Tabs returns the main TabView
func (ge *CodeView) Tabs() *core.Tabs {
	return ge.Splits().Child(TabsIndex).(*core.Tabs)
}

// StatusBar returns the statusbar widget
func (ge *CodeView) StatusBar() *core.Frame {
	if ge.This() == nil || !ge.HasChildren() {
		return nil
	}
	return ge.ChildByName("statusbar", 2).(*core.Frame)
}

// StatusLabel returns the statusbar label widget
func (ge *CodeView) StatusLabel() *core.Label {
	return ge.StatusBar().Child(0).(*core.Label)
}

// SelectedFileNode returns currently selected file tree node as a *filetree.Node
// could be nil.
func (ge *CodeView) SelectedFileNode() *filetree.Node {
	n := len(ge.Files.SelectedNodes)
	if n == 0 {
		return nil
	}
	return filetree.AsNode(ge.Files.SelectedNodes[n-1].This())
}

// ConfigSplits configures the Splits.
func (ge *CodeView) ConfigSplits() {
	// note: covered by global update
	split := ge.Splits().SetSplits(ge.Settings.Splits...)
	ftfr := core.NewFrame(split, "filetree")
	ftfr.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Overflow.Set(styles.OverflowAuto)
	})
	ft := filetree.NewTree(ftfr, "filetree")
	ft.OpenDepth = 4
	ge.Files = ft
	ft.FileNodeType = code.FileNodeType

	ge.Files.OnSelect(func(e events.Event) {
		e.SetHandled()
		sn := ge.SelectedFileNode()
		if sn != nil {
			ge.FileNodeSelected(sn)
		}
	})

	for i := 0; i < NTextEditors; i++ {
		i := i
		txnm := fmt.Sprintf("%d", i)
		txly := core.NewLayout(split, "textlay-"+txnm)
		txly.Style(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Grow.Set(1, 1)
		})
		txbut := core.NewButton(txly, "textbut-"+txnm).SetText("texteditor: " + txnm)
		txbut.Type = core.ButtonAction
		txbut.Style(func(s *styles.Style) {
			s.Grow.Set(1, 0)
		})
		txbut.Menu = func(m *core.Scene) {
			ge.TextEditorButtonMenu(i, m)
		}
		txbut.OnClick(func(e events.Event) {
			ge.SetActiveTextEditorIndex(i)
		})

		ted := code.NewTextEditor(txly, "texteditor-"+txnm)
		ted.Code = ge
		code.ConfigEditorTextEditor(&ted.Editor)
		ted.OnFocus(func(e events.Event) {
			ge.ActiveTextEditorIndex = i
		})
		// get updates on cursor movement and qreplace
		ted.OnInput(func(e events.Event) {
			ge.UpdateStatusLabel()
		})
	}

	ge.UpdateTextButtons()

	mtab := core.NewTabs(split, "tabs").SetType(core.FunctionalTabs)
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
}

// ConfigStatusBar configures statusbar with label
func (ge *CodeView) ConfigStatusBar() {
	sb := ge.StatusBar()
	sb.Style(func(s *styles.Style) {
		s.Grow.Set(1, 0)
		s.Min.Y.Em(1.0)
		s.Margin.Zero()
		s.Padding.Set(units.Dp(4))
	})
	lbl := core.NewLabel(sb, "sb-lbl").SetText("This is the status bar initial configuration.  Welcome to code!")
	lbl.Style(func(s *styles.Style) {
		s.Min.X.Ch(100)
		s.Min.Y.Em(1.0)
		s.Margin.Zero()
		s.Padding.Zero()
		s.Text.TabSize = 4
	})
}
