// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"fmt"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
)

// Tree is a [core.Tree] that interacts properly with [Canvas].
type Tree struct { //types:add
	core.Tree

	// the parent [Canvas]
	Canvas *Canvas `copier:"-" json:"-" xml:"-" display:"-"`
}

func (tv *Tree) Init() {
	tv.Tree.Init()
	tv.ContextMenus = nil
	tv.AddContextMenu(tv.contextMenu)
	tv.Styler(func(s *styles.Style) {
		s.IconSize.Set(units.Em(1))
	})
	tv.Parts.OnDoubleClick(func(e events.Event) {
		e.SetHandled()
		if tv.HasChildren() {
			tv.ToggleClose()
		} else {
			tv.SelectSVG()
			tv.EditNode()
		}
	})
	tree.AddChildInit(tv.Parts, "text", func(w *core.Text) {
		w.Styler(func(s *styles.Style) {
			sn := tv.SyncNode
			if !NodeIsLayer(sn) {
				return
			}
			s.Font.Weight = rich.Bold
			if LayerIsLocked(sn) {
				s.Color = colors.Scheme.Error.Base
			}
			if !LayerIsVisible(sn) {
				s.Opacity = 0.5
			}
		})
	})
	tv.Updater(func() {
		sn := tv.SyncNode
		tv.Icon = icons.Blank
		if sn != nil && NodeIsLayer(sn) {
			switch {
			case tv.LayerIsCurrent():
				tv.Icon = icons.Check
			case LayerIsLocked(sn):
				tv.Icon = icons.Lock
			}
		} else {
			switch sn.(type) {
			case *svg.Circle:
				tv.Icon = icons.Circle
			case *svg.Ellipse:
				tv.Icon = icons.Circle
			case *svg.Rect:
				tv.Icon = icons.Rectangle
			case *svg.Path:
				tv.Icon = icons.LineCurve
			case *svg.Image:
				tv.Icon = icons.Image
			case *svg.Text:
				tv.Icon = icons.TextAd
			case *svg.Group:
				tv.Icon = icons.Folder
			default:
				tv.Icon = icons.ChartData
			}
		}
	})
}

// FileRoot returns the Root node as a [Tree].
func (tv *Tree) CanvasRoot() *Tree {
	return tv.Root.(*Tree)
}

// NewGroup makes a new group.
func (tv *Tree) NewGroup() { //types:add
	cv := tv.CanvasRoot().Canvas
	NewSVGElement[svg.Group](cv.SVG, true)
}

// SelectSVG selects node in SVG
func (tv *Tree) SelectSVG() { //types:add
	cv := tv.CanvasRoot().Canvas
	cv.SelectNodeInSVG(tv.SyncNode, events.SelectOne)
}

// LayerIsCurrent returns true if layer is the current active one for creating.
func (tv *Tree) LayerIsCurrent() bool { //types:add
	cv := tv.CanvasRoot().Canvas
	return cv.IsCurLayer(tv.SyncNode.AsTree().Name)
}

// LayerSetCurrent sets this layer as the current layer name
func (tv *Tree) LayerSetCurrent() { //types:add
	cv := tv.CanvasRoot().Canvas
	sn := tv.SyncNode
	cur := cv.EditState.CurLayer
	if cur != "" {
		cli := tv.Parent.AsTree().ChildByName("tv_"+cur, 0)
		if cli != nil {
			cl := cli.(*Tree)
			cl.LayerClearCurrent()
		}
	}
	if LayerIsLocked(sn) {
		tv.LayerToggleLock()
	}
	if !LayerIsVisible(sn) {
		tv.LayerToggleVis()
	}
	cv.SetCurLayer(sn.AsTree().Name)
	cv.UpdateLayers()
}

// LayerClearCurrent clears this layer as the current layer if it was set as such.
func (tv *Tree) LayerClearCurrent() { //types:add
	cv := tv.CanvasRoot().Canvas
	cv.ClearCurLayer(tv.SyncNode.AsTree().Name)
}

// NodeIsMetaData returns true if given node is a MetaData
func NodeIsMetaData(kn tree.Node) bool {
	_, ismd := kn.(*svg.MetaData)
	return ismd
}

// LayerToggleLock toggles whether layer is locked or not
func (tv *Tree) LayerToggleLock() { //types:add
	cv := tv.CanvasRoot().Canvas
	sn := tv.SyncNode
	np := ""
	if LayerIsLocked(sn) {
		np = "false"
	} else {
		tv.LayerClearCurrent()
		np = "true"
	}
	sn.AsTree().Properties["insensitive"] = np
	cv.UpdateLayers()
}

// LayerToggleVis toggles visibility of the layer
func (tv *Tree) LayerToggleVis() { //types:add
	cv := tv.CanvasRoot().Canvas
	sn := tv.SyncNode
	np := ""
	if LayerIsVisible(sn) {
		np = "display:none"
		tv.LayerClearCurrent()
	} else {
		np = "display:inline"
	}
	sn.AsTree().Properties["style"] = np
	cv.UpdateLayers()
}

func (tv *Tree) contextMenu(m *core.Scene) {
	sn := tv.SyncNode
	tri := tv.This.(core.Treer)
	isLay := NodeIsLayer(sn)

	core.NewFuncButton(m).SetFunc(tv.EditNode).SetText("Edit").
		SetIcon(icons.Edit).SetEnabled(tv.HasSelection())
	core.NewFuncButton(m).SetFunc(tv.SelectSVG).SetText("Select").
		SetIcon(icons.Select)
	core.NewFuncButton(m).SetFunc(tv.NewGroup).SetText("New group").
		SetIcon(icons.NewLabel)

	if isLay {
		core.NewSeparator(m)

		core.NewFuncButton(m).SetFunc(tv.LayerSetCurrent).SetText("Set as current").
			SetIcon(icons.SwitchFill)
		core.NewFuncButton(m).SetFunc(tv.LayerToggleLock).SetText("Toggle lock").
			SetIcon(icons.SwitchFill)
		core.NewFuncButton(m).SetFunc(tv.LayerToggleVis).SetText("Toggle vis").
			SetIcon(icons.SwitchFill)
		core.NewSeparator(m)
	}

	core.NewFuncButton(m).SetFunc(tv.Duplicate).SetIcon(icons.ContentCopy).SetEnabled(tv.HasSelection())
	core.NewFuncButton(m).SetFunc(tv.DeleteNode).SetText("Delete").SetIcon(icons.Delete).
		SetEnabled(tv.HasSelection())
	core.NewFuncButton(m).SetFunc(tri.Copy).SetIcon(icons.Copy).SetKey(keymap.Copy).SetEnabled(tv.HasSelection())
	core.NewFuncButton(m).SetFunc(tri.Cut).SetIcon(icons.Cut).SetKey(keymap.Cut).SetEnabled(tv.HasSelection())
	paste := core.NewFuncButton(m).SetFunc(tri.Paste).SetIcon(icons.Paste).SetKey(keymap.Paste)
	cb := tv.Scene.Events.Clipboard()
	if cb != nil {
		paste.SetState(cb.IsEmpty(), states.Disabled)
	}
	core.NewSeparator(m)
	core.NewFuncButton(m).SetFunc(tv.OpenAll).SetIcon(icons.KeyboardArrowDown).SetEnabled(tv.HasSelection())
	core.NewFuncButton(m).SetFunc(tv.CloseAll).SetIcon(icons.KeyboardArrowRight).SetEnabled(tv.HasSelection())
}

//////// Canvas methods

// SelectNodeInTree selects given node in Tree
func (cv *Canvas) SelectNodeInTree(nd tree.Node, mode events.SelectModes) {
	tv := cv.tree
	tvn := tv.FindSyncNode(nd)
	if tvn != nil {
		tvn.OpenParents()
		tvn.SelectEvent(mode)
	}
}

// AnySelectedNodes returns svg nodes that are selected in the svg tree (first)
// or selected in the SVG drawing (second), as svg Nodes.
// This is useful for contextualizing tree actions (e.g., NewGroup).
func (cv *Canvas) AnySelectedNodes() []svg.Node {
	sl := cv.tree.GetSelectedNodes()
	if len(sl) > 0 {
		nl := make([]svg.Node, len(sl))
		for i := range sl {
			sn := sl[i].AsCoreTree().SyncNode.(svg.Node)
			nl[i] = sn
		}
		return nl
	}
	return cv.EditState.SelectedList(false)
}

// SelectedAsTrees returns the currently selected items from SVG as Tree nodes.
func (cv *Canvas) SelectedAsTrees() []core.Treer {
	es := &cv.EditState
	sl := es.SelectedList(false)
	if len(sl) == 0 {
		return nil
	}
	return cv.ItemsAsTrees(sl...)
}

// ItemsAsTrees returns the list of SVG items as Tree nodes.
func (cv *Canvas) ItemsAsTrees(nd ...svg.Node) []core.Treer {
	tv := cv.tree
	var tvl []core.Treer
	for _, si := range nd {
		tvn := tv.FindSyncNode(si.AsTree().This)
		if tvn != nil {
			tvl = append(tvl, tvn)
		}
	}
	return tvl
}

// DuplicateSelected duplicates selected items in SVG view, using Tree methods
func (cv *Canvas) DuplicateSelected() { //types:add
	tvl := cv.SelectedAsTrees()
	if len(tvl) == 0 {
		cv.SetStatus("Duplicate: no tree items found")
		return
	}
	sv := cv.SVG
	sv.UndoSave("DuplicateSelected", "")
	for _, tr := range tvl {
		ctr := tr.AsCoreTree()
		ctr.Duplicate()
		idx := ctr.IndexInParent()
		nw := ctr.Parent.AsTree().Child(idx + 1).(core.Treer).AsCoreTree()
		sv.SVG.GradientDuplicateNode(nw.SyncNode.(svg.Node))
	}
	cv.SetStatus("Duplicated selected items")
	cv.ChangeMade()
	sv.UpdateView()
}

// CopySelected copies selected items in SVG view, using Tree methods.
func (cv *Canvas) CopySelected() { //types:add
	tvl := cv.SelectedAsTrees()
	if len(tvl) == 0 {
		cv.SetStatus("Copy: no tree items found")
		return
	}
	cv.tree.SetSelectedNodes(tvl)
	tvl[0].Copy() // must be called on first node
	cv.SetStatus("Copied selected items")
}

// CutSelected cuts selected items in SVG view, using Tree methods.
func (cv *Canvas) CutSelected() { //types:add
	tvl := cv.SelectedAsTrees()
	if len(tvl) == 0 {
		cv.SetStatus("Cut: no tree items found")
		return
	}
	sv := cv.SVG
	sv.UndoSave("CutSelected", "")
	sv.EditState().ResetSelected()
	cv.tree.SetSelectedNodes(tvl)
	tvl[0].Cut() // must be called on first node
	cv.SetStatus("Cut selected items")
	cv.ChangeMade()
	cv.UpdateSVG()
}

// PasteClip pastes clipboard, using cur layer etc
func (cv *Canvas) PasteClip() { //types:add
	md := cv.Clipboard().Read([]string{fileinfo.DataJson})
	if md == nil {
		return
	}
	sv := cv.SVG
	sv.UndoSave("Paste", "")
	tv := cv.tree
	parent := sv.NewParent(false)
	pn := tv.FindSyncNode(parent)
	fmt.Println("c paste:", pn)
	pn.PasteChildren(md, events.DropCopy)
	cv.SetStatus("Pasted items from clipboard")
	cv.ChangeMade()
	cv.UpdateSVG()
}

// DeleteSelected deletes selected items in SVG view, using Tree methods
func (cv *Canvas) DeleteSelected() {
	tvl := cv.SelectedAsTrees()
	if len(tvl) == 0 {
		cv.SetStatus("Delete: no tree items found")
		return
	}
	sv := cv.SVG
	sv.UndoSave("DeleteSelected", "")
	sv.EditState().ResetSelected()
	cv.tree.SetSelectedNodes(tvl)
	tvl[0].DeleteSelected() // must be called on first node
	cv.SetStatus("Deleted selected items")
	cv.ChangeMade()
	cv.UpdateSVG()
}

// DeleteItems deletes the given svg.Node item(s) using Tree methods.
func (cv *Canvas) DeleteItems(nd ...svg.Node) {
	sv := cv.SVG
	sv.UndoSave("DeleteItems", "")
	tvl := cv.ItemsAsTrees(nd...)
	cv.tree.SetSelectedNodes(tvl)
	tvl[0].DeleteSelected() // must be called on first node
	cv.SetStatus("Deleted items")
	cv.ChangeMade()
	cv.UpdateSVG()
}
