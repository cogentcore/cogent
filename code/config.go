// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/filetree"
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
