// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
)

// Tree is a [core.Tree] that interacts properly with [Canvas].
type Tree struct {
	core.Tree

	// the parent [Canvas]
	Canvas *Canvas `copier:"-" json:"-" xml:"-" display:"-"`
}

// SelectNodeInTree selects given node in Tree
func (cv *Canvas) SelectNodeInTree(kn tree.Node, mode events.SelectModes) {
	tv := cv.tree
	tvn := tv.FindSyncNode(kn)
	if tvn != nil {
		tvn.OpenParents()
		tvn.SelectEvent(mode)
	}
}

// SelectedAsTrees returns the currently selected items from SVG as Tree nodes
func (cv *Canvas) SelectedAsTrees() []core.Treer {
	es := &cv.EditState
	sl := es.SelectedList(false)
	if len(sl) == 0 {
		return nil
	}
	tv := cv.tree
	var tvl []core.Treer
	for _, si := range sl {
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
	// sv.SetFullReRender()
	tv := cv.tree
	// tv.SetFullReRender()
	for _, tr := range tvl {
		tr.AsCoreTree().Duplicate()
	}
	cv.SetStatus("Duplicated selected items")
	tv.Resync() // todo: should not be needed
	cv.ChangeMade()
}

// CopySelected copies selected items in SVG view, using Tree methods
func (cv *Canvas) CopySelected() { //types:add
	tvl := cv.SelectedAsTrees()
	if len(tvl) == 0 {
		cv.SetStatus("Copy: no tree items found")
		return
	}
	tv := cv.tree
	tv.SetSelectedNodes(tvl)
	tvl[0].Copy() // operates on first element in selection
	cv.SetStatus("Copied selected items")
}

// CutSelected cuts selected items in SVG view, using Tree methods
func (cv *Canvas) CutSelected() { //types:add
	tvl := cv.SelectedAsTrees()
	if len(tvl) == 0 {
		cv.SetStatus("Cut: no tree items found")
		return
	}
	sv := cv.SVG
	sv.UndoSave("CutSelected", "")
	// sv.SetFullReRender()
	sv.EditState().ResetSelected()
	tv := cv.tree
	// tv.SetFullReRender()
	tv.SetSelectedNodes(tvl)
	tvl[0].Cut() // operates on first element in selection
	cv.SetStatus("Cut selected items")
	tv.Resync() // todo: should not be needed
	sv.UpdateSelSprites()
	cv.ChangeMade()
}

// PasteClip pastes clipboard, using cur layer etc
func (cv *Canvas) PasteClip() { //types:add
	md := cv.Clipboard().Read([]string{fileinfo.DataJson})
	if md == nil {
		return
	}
	// es := &gv.EditState
	sv := cv.SVG
	sv.UndoSave("Paste", "")
	// sv.SetFullReRender()
	tv := cv.tree
	// tv.SetFullReRender()
	// parent := tv
	// if es.CurLayer != "" {
	// 	ly := tv.ChildByName("tv_"+es.CurLayer, 1)
	// 	if ly != nil {
	// 		parent = ly.Embed(KiT_Tree).(*Tree)
	// 	}
	// }
	// par.PasteChildren(md, dnd.DropCopy)
	cv.SetStatus("Pasted items from clipboard")
	tv.Resync() // todo: should not be needed
	cv.ChangeMade()
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
	// sv.SetFullReRender()
	tv := cv.tree
	// tv.SetFullReRender()
	// for _, tvi := range tvl {
	// 	tvi.SrcDelete()
	// }
	cv.SetStatus("Deleted selected items")
	tv.Resync() // todo: should not be needed
	sv.UpdateSelSprites()
	cv.ChangeMade()
}

///////////////////////////////////////////////
//  Tree

/*
// TreeIsLayerFunc is an ActionUpdateFunc that activates if node is a Layer
var TreeIsLayerFunc = core.ActionUpdateFunc(func(fni any, act *core.Button) {
	tv := fni.(tree.Node).Embed(KiT_Tree).(*Tree)
	sn := tv.SrcNode
	if sn != nil {
		act.SetInactiveState(!NodeIsLayer(sn))
	}
})
*/

func (tv *Tree) Init() {
	tv.Tree.Init()
	tree.AddChildInit(tv.Parts, "text", func(w *core.Text) {
		w.Styler(func(s *styles.Style) {
			sn := tv.SyncNode
			switch {
			case NodeIsLayer(sn):
				s.Font.Weight = rich.Bold
			case LayerIsLocked(sn):
				s.Color = colors.Scheme.Error.Base
			case !LayerIsVisible(sn):
				s.Font.Slant = rich.Italic
			}
		})
	})
	tv.Updater(func() {
		sn := tv.SyncNode
		tv.Icon = icons.Blank
		if NodeIsLayer(sn) {
			switch {
			case tv.LayerIsCurrent():
				tv.Icon = icons.Check
			case LayerIsLocked(sn):
				tv.Icon = icons.Lock
			case !LayerIsVisible(sn):
				tv.Icon = icons.Close
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
				tv.Icon = "tool-text"
			}
		}
	})
}

// SelectSVG
func (tv *Tree) SelectSVG() {
	cv := tv.Canvas
	if cv != nil {
		cv.SelectNodeInSVG(tv.SyncNode, events.SelectOne)
	}
}

// LayerIsCurrent returns true if layer is the current active one for creating
func (tv *Tree) LayerIsCurrent() bool {
	cv := tv.Canvas
	if cv != nil {
		return cv.IsCurLayer(tv.SyncNode.AsTree().Name)
	}
	return false
}

// LayerSetCurrent sets this layer as the current layer name
func (tv *Tree) LayerSetCurrent() {
	sn := tv.SyncNode
	cv := tv.Canvas
	if cv != nil {
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
		// tv.SetFullReRender() // needed for icon updating
		// tv.UpdateSig()
	}
}

// LayerClearCurrent clears this layer as the current layer if it was set as such.
func (tv *Tree) LayerClearCurrent() {
	cv := tv.Canvas
	if cv != nil {
		cv.ClearCurLayer(tv.SyncNode.AsTree().Name)
		// tv.SetFullReRender() // needed for icon updating
		// tv.UpdateSig()
	}
}

// NodeIsMetaData returns true if given node is a MetaData
func NodeIsMetaData(kn tree.Node) bool {
	_, ismd := kn.(*svg.MetaData)
	return ismd
}

// LayerToggleLock toggles whether layer is locked or not
func (tv *Tree) LayerToggleLock() {
	sn := tv.SyncNode
	np := ""
	if LayerIsLocked(sn) {
		np = "false"
	} else {
		tv.LayerClearCurrent()
		np = "true"
	}
	sn.AsTree().Properties["insensitive"] = np
	// tv.SetFullReRenderIconLabel()
	// tv.UpdateSig()
}

// LayerToggleVis toggles visibility of the layer
func (tv *Tree) LayerToggleVis() {
	sn := tv.SyncNode
	np := ""
	if LayerIsVisible(sn) {
		np = "display:none"
		tv.LayerClearCurrent()
	} else {
		np = "display:inline"
	}
	sn.AsTree().Properties["style"] = np
	// tv.UpdateSig()
}

/*
var TreeProperties = tree.Properties{
	"CtxtMenuActive": tree.Propertieslice{
		{"SelectSVG", tree.Properties{
			"label": "Select",
		}},
		{"sep-layer", tree.BlankProp{}},
		{"LayerSetCurrent", tree.Properties{
			"label":    "Layer: Set Current",
			"updatefunc": TreeIsLayerFunc,
		}},
		{"LayerToggleLock", tree.Properties{
			"label":    "Layer: Toggle Lock",
			"updatefunc": TreeIsLayerFunc,
		}},
		{"LayerToggleVis", tree.Properties{
			"label":    "Layer: Toggle Visible",
			"updatefunc": TreeIsLayerFunc,
		}},
	},
}

*/
