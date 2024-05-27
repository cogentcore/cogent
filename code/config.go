// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
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

func (ge *CodeView) Make(p *core.Plan) {
	splits := core.AddAt(p, "splitview", func(w *core.Splits) {
		w.SetSplits(ge.Settings.Splits...)
	})
	sb := core.AddAt(p, "statusbar", func(w *core.Frame) {
		w.Style(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Min.Y.Em(1.0)
			s.Margin.Zero()
			s.Padding.Set(units.Dp(4))
		})
	})
	core.AddAt(sb, "sb-text", func(w *core.Text) {
		w.SetText("Welcome to Cogent Code!" + strings.Repeat(" ", 80))
		w.Style(func(s *styles.Style) {
			s.Min.X.Ch(100)
			s.Min.Y.Em(1.0)
			s.Margin.Zero()
			s.Padding.Zero()
			s.Text.TabSize = 4
		})
	})

	ftfr := core.AddAt(splits, "filetree", func(w *core.Frame) {
		w.Style(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Overflow.Set(styles.OverflowAuto)
		})
	})
	core.AddAt(ftfr, "filetree", func(w *filetree.Tree) {
		w.OpenDepth = 4
		ge.Files = w
		w.FileNodeType = FileNodeType

		w.OnSelect(func(e events.Event) {
			e.SetHandled()
			sn := ge.SelectedFileNode()
			if sn != nil {
				ge.FileNodeSelected(sn)
			}
		})
	})

	for i := 0; i < NTextEditors; i++ {
		txnm := fmt.Sprintf("%d", i)
		txfr := core.AddAt(splits, "textframe-"+txnm, func(w *core.Frame) {
			w.Style(func(s *styles.Style) {
				s.Direction = styles.Column
				s.Grow.Set(1, 1)
			})
		})
		core.AddAt(txfr, "textbut-"+txnm, func(w *core.Button) {
			w.SetText("texteditor: " + txnm)
			w.Type = core.ButtonAction
			w.Style(func(s *styles.Style) {
				s.Grow.Set(1, 0)
			})
			w.Menu = func(m *core.Scene) {
				ge.TextEditorButtonMenu(i, m)
			}
			w.OnClick(func(e events.Event) {
				ge.SetActiveTextEditorIndex(i)
			})
			// todo: update
			// ge.UpdateTextButtons()
		})
		core.AddAt(txfr, "texteditor-"+txnm, func(w *TextEditor) {
			w.Code = ge
			ConfigEditorTextEditor(&w.Editor)
			w.OnFocus(func(e events.Event) {
				ge.ActiveTextEditorIndex = i
			})
			// get updates on cursor movement and qreplace
			w.OnInput(func(e events.Event) {
				ge.UpdateStatusText()
			})
		})
	}

	core.AddAt(splits, "tabs", func(w *core.Tabs) {
		w.SetType(core.FunctionalTabs)
		w.Style(func(s *styles.Style) {
			s.Grow.Set(1, 1)
		})
	})

	// todo: builder function
	// ge.OpenConsoleTab()
	// ge.UpdateFiles()

	// todo: need this still:
	// mtab.OnChange(func(e events.Event) {
	// todo: need to monitor deleted
	// gee.TabDeleted(data.(string))
	// if data == "Find" {
	// 	ge.ActiveTextEditor().ClearHighlights()
	// }
	// })
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
func (ge *CodeView) TextEditorByIndex(idx int) *TextEditor {
	return ge.Splits().Child(TextEditor1Index + idx).Child(1).(*TextEditor)
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

// StatusText returns the status bar text widget
func (ge *CodeView) StatusText() *core.Text {
	return ge.StatusBar().Child(0).(*core.Text)
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
