// Copyright (c) 2023, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidev

import (
	"fmt"

	"goki.dev/gi/v2/filetree"
	"goki.dev/gi/v2/gi"
	"goki.dev/gide/v2/gide"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// NTextViews is the number of text views to create -- to keep things simple
// and consistent (e.g., splitter settings always have the same number of
// values), we fix this degree of freedom, and have flexibility in the
// splitter settings for what to actually show.
const NTextViews = 2

// These are then the fixed indices of the different elements in the splitview
const (
	FileTreeIdx = iota
	TextView1Idx
	TextView2Idx
	TabsIdx
)

func (ge *GideView) ConfigWidget(sc *gi.Scene) {
	ge.ConfigGideView(sc)
}

// Config configures the view
func (ge *GideView) ConfigGideView(sc *gi.Scene) {
	if ge.HasChildren() {
		return
	}

	updt := ge.UpdateStart()
	// sc.TopAppBar = ge.TopAppBar

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

	ge.UpdateEndLayout(updt)
}

// IsConfiged returns true if the view is configured
func (ge *GideView) IsConfiged() bool {
	return ge.HasChildren()
}

// Splits returns the main Splits
func (ge *GideView) Splits() *gi.Splits {
	return ge.ChildByName("splitview", 2).(*gi.Splits)
}

// TextViewButtonByIndex returns the top textview menu button by index (0 or 1)
func (ge *GideView) TextViewButtonByIndex(idx int) *gi.Button {
	return ge.Splits().Child(TextView1Idx + idx).Child(0).(*gi.Button)
}

// TextViewByIndex returns the TextView by index (0 or 1), nil if not found
func (ge *GideView) TextViewByIndex(idx int) *gide.TextView {
	return ge.Splits().Child(TextView1Idx + idx).Child(1).(*gide.TextView)
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
	split.Dim = mat32.X
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

	for i := 0; i < NTextViews; i++ {
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
			ge.TextViewButtonMenu(i, m)
		}
		txbut.OnClick(func(e events.Event) {
			ge.SetActiveTextViewIdx(i)
		})

		ted := gide.NewTextView(txly, "textview-"+txnm)
		ted.Gide = ge
		ted.Style(func(s *styles.Style) {
			s.Grow.Set(1, 1)
			s.Min.X.Ch(80)
			s.Min.Y.Em(40)
			if ge.Prefs.Editor.WordWrap {
				s.Text.WhiteSpace = styles.WhiteSpacePreWrap
			} else {
				s.Text.WhiteSpace = styles.WhiteSpacePre
			}
			s.Text.TabSize = ge.Prefs.Editor.TabSize
			s.Font.Family = string(gi.Prefs.MonoFont)
		})
		ted.OnFocus(func(e events.Event) {
			ge.SetActiveTextViewIdx(i)
		})
		// get updates on cursor movement and qreplace
		ted.OnInput(func(e events.Event) {
			ge.UpdateStatusLabel()
		})
	}

	ge.UpdateTextButtons()

	mtab := gi.NewTabs(split, "tabs")
	mtab.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})

	// mtab.OnChange(func(e events.Event) {
	// todo: need to monitor deleted
	// gee.TabDeleted(data.(string))
	// if data == "Find" {
	// 	ge.ActiveTextView().ClearHighlights()
	// }
	// })

	split.SetSplits(ge.Prefs.Splits...)
}

// ConfigStatusBar configures statusbar with label
func (ge *GideView) ConfigStatusBar() {
	sb := ge.StatusBar()
	sb.Style(func(s *styles.Style) {
		s.Grow.Set(1, 0)
		s.Min.Y.Em(1.2)
		s.Margin.Zero()
		s.Padding.Zero()
	})
	lbl := gi.NewLabel(sb, "sb-lbl")
	lbl.Style(func(s *styles.Style) {
		s.Grow.Set(1, 0)
		s.Min.X.Ch(100)
		s.Min.Y.Em(1.1)
		s.Margin.Zero()
		s.Padding.Set(units.Dp(4))
		s.Text.TabSize = 4
	})
}
