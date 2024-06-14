// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
)

// Tree is a [core.Tree] that interacts properly with [Vector].
type Tree struct {
	core.Tree

	// the parent vector
	Vector *Vector `copier:"-" json:"-" xml:"-" display:"-"`
}

// SelectNodeInTree selects given node in Tree
func (gv *Vector) SelectNodeInTree(kn tree.Node, mode events.SelectModes) {
	tv := gv.Tree()
	tvn := tv.FindSyncNode(kn)
	if tvn != nil {
		tvn.OpenParents()
		tvn.SelectAction(mode)
	}
}

// SelectedAsTrees returns the currently selected items from SVG as Tree nodes
func (gv *Vector) SelectedAsTrees() []core.Treer {
	es := &gv.EditState
	sl := es.SelectedList(false)
	if len(sl) == 0 {
		return nil
	}
	tv := gv.Tree()
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
func (gv *Vector) DuplicateSelected() { //types:add
	tvl := gv.SelectedAsTrees()
	if len(tvl) == 0 {
		gv.SetStatus("Duplicate: no tree items found")
		return
	}
	sv := gv.SVG()
	sv.UndoSave("DuplicateSelected", "")
	// sv.SetFullReRender()
	tv := gv.Tree()
	// tv.SetFullReRender()
	for _, tvi := range tvl {
		tvi.AsCoreTree().DuplicateSync()
	}
	gv.SetStatus("Duplicated selected items")
	tv.ReSync() // todo: should not be needed
	gv.ChangeMade()
}

// CopySelected copies selected items in SVG view, using Tree methods
func (gv *Vector) CopySelected() { //types:add
	tvl := gv.SelectedAsTrees()
	if len(tvl) == 0 {
		gv.SetStatus("Copy: no tree items found")
		return
	}
	tv := gv.Tree()
	tv.SetSelectedViews(tvl)
	tvl[0].Copy(true) // operates on first element in selection
	gv.SetStatus("Copied selected items")
}

// CutSelected cuts selected items in SVG view, using Tree methods
func (gv *Vector) CutSelected() { //types:add
	tvl := gv.SelectedAsTrees()
	if len(tvl) == 0 {
		gv.SetStatus("Cut: no tree items found")
		return
	}
	sv := gv.SVG()
	sv.UndoSave("CutSelected", "")
	// sv.SetFullReRender()
	sv.EditState().ResetSelected()
	tv := gv.Tree()
	// tv.SetFullReRender()
	tv.SetSelectedViews(tvl)
	tvl[0].Cut() // operates on first element in selection
	gv.SetStatus("Cut selected items")
	tv.ReSync() // todo: should not be needed
	sv.UpdateSelSprites()
	gv.ChangeMade()
}

// PasteClip pastes clipboard, using cur layer etc
func (gv *Vector) PasteClip() { //types:add
	md := gv.Clipboard().Read([]string{fileinfo.DataJson})
	if md == nil {
		return
	}
	// es := &gv.EditState
	sv := gv.SVG()
	sv.UndoSave("Paste", "")
	// sv.SetFullReRender()
	tv := gv.Tree()
	// tv.SetFullReRender()
	// parent := tv
	// if es.CurLayer != "" {
	// 	ly := tv.ChildByName("tv_"+es.CurLayer, 1)
	// 	if ly != nil {
	// 		parent = ly.Embed(KiT_Tree).(*Tree)
	// 	}
	// }
	// par.PasteChildren(md, dnd.DropCopy)
	gv.SetStatus("Pasted items from clipboard")
	tv.ReSync() // todo: should not be needed
	gv.ChangeMade()
}

// DeleteSelected deletes selected items in SVG view, using Tree methods
func (gv *Vector) DeleteSelected() {
	tvl := gv.SelectedAsTrees()
	if len(tvl) == 0 {
		gv.SetStatus("Delete: no tree items found")
		return
	}
	sv := gv.SVG()
	sv.UndoSave("DeleteSelected", "")
	sv.EditState().ResetSelected()
	// sv.SetFullReRender()
	tv := gv.Tree()
	// tv.SetFullReRender()
	// for _, tvi := range tvl {
	// 	tvi.SrcDelete()
	// }
	gv.SetStatus("Deleted selected items")
	tv.ReSync() // todo: should not be needed
	sv.UpdateSelSprites()
	gv.ChangeMade()
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

// ParVector returns the parent Vector
func (tv *Tree) ParVector() *Vector {
	rtv := tv.RootView.Embed(KiT_Tree).(*Tree)
	return rtv.Vector
}
*/

// SelectSVG
func (tv *Tree) SelectSVG() {
	gv := tv.Vector
	if gv != nil {
		gv.SelectNodeInSVG(tv.SyncNode, events.SelectOne)
	}
}

// LayerIsCurrent returns true if layer is the current active one for creating
func (tv *Tree) LayerIsCurrent() bool {
	gv := tv.Vector
	if gv != nil {
		return gv.IsCurLayer(tv.SyncNode.AsTree().Name)
	}
	return false
}

// LayerSetCurrent sets this layer as the current layer name
func (tv *Tree) LayerSetCurrent() {
	sn := tv.SyncNode
	gv := tv.Vector
	if gv != nil {
		cur := gv.EditState.CurLayer
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
		gv.SetCurLayer(sn.AsTree().Name)
		// tv.SetFullReRender() // needed for icon updating
		// tv.UpdateSig()
	}
}

// LayerClearCurrent clears this layer as the current layer if it was set as such.
func (tv *Tree) LayerClearCurrent() {
	gv := tv.Vector
	if gv != nil {
		gv.ClearCurLayer(tv.SyncNode.AsTree().Name)
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
	".svgnode": tree.Properties{
		"font-weight": gist.WeightNormal,
		"font-style":  gist.FontNormal,
	},
	".layer": tree.Properties{
		"font-weight": gist.WeightBold,
	},
	".invisible": tree.Properties{
		"font-style": gist.FontItalic,
	},
	".locked": tree.Properties{
		"color": "#ff4252",
	},
	core.TreeSelectors[core.TreeActive]: tree.Properties{},
	core.TreeSelectors[core.TreeSel]: tree.Properties{
		"background-color": &core.Settings.Colors.Select,
	},
	core.TreeSelectors[core.TreeFocus]: tree.Properties{
		"background-color": &core.Settings.Colors.Control,
	},
	"CtxtMenuActive": tree.Propertieslice{
		{"SrcEdit", tree.Properties{
			"label": "Edit",
		}},
		{"SelectSVG", tree.Properties{
			"label": "Select",
		}},
		{"sep-edit", tree.BlankProp{}},
		{"SrcDuplicate", tree.Properties{
			"label":    "Duplicate",
			"shortcut": keymap.Duplicate,
		}},
		{"Copy", tree.Properties{
			"shortcut": keymap.Copy,
			"Args": tree.Propertieslice{
				{"reset", tree.Properties{
					"value": true,
				}},
			},
		}},
		{"Cut", tree.Properties{
			"shortcut": keymap.Cut,
			"updatefunc": core.ActionUpdateFunc(func(tvi any, act *core.Button) {
				tv := tvi.(tree.Node).Embed(KiT_Tree).(*Tree)
				act.SetInactiveState(tv.IsRootOrField(""))
			}),
		}},
		{"Paste", tree.Properties{
			"shortcut": keymap.Paste,
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
		{"sep-open", tree.BlankProp{}},
		{"OpenAll", tree.Properties{}},
		{"CloseAll", tree.Properties{}},
	},
}

func (tv *Tree) Style2D() {
	sn := tv.SrcNode
	tv.Class = ""
	if sn != nil {
		if NodeIsLayer(sn) {
			tv.Icon = icons.Icon("blank")
			tv.AddClass("layer")
			if tv.LayerIsCurrent() {
				tv.Icon = icons.Icon("checkmark")
			}
			if LayerIsLocked(sn) {
				tv.AddClass("locked")
				tv.Icon = icons.Icon("close")
			}
			if !LayerIsVisible(sn) {
				tv.AddClass("invisible")
				tv.Icon = icons.Icon("close")
			}
			// todo: visibility and locked flags
		} else {
			tv.AddClass("svgnode")
			switch sn.(type) {
			case *svg.Circle:
				tv.Icon = icons.Icon("circlebutton-off")
			case *svg.Ellipse:
				tv.Icon = icons.Icon("circlebutton-off")
			case *svg.Rect:
				tv.Icon = icons.Icon("stop")
			case *svg.Path:
				tv.Icon = icons.Icon("color")
			case *svg.Image:
				tv.Icon = icons.Icon("file-image") // todo: image
			case *svg.Text:
				tv.Icon = icons.Icon("file-doc") // todo: A = text
			}
		}
		tv.StyleTree()
		tv.LayState.SetFromStyle(&tv.Sty.Layout) // also does reset
	}
}
*/
