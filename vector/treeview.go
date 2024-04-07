// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/fi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
)

// TreeView is a TreeView that knows how to operate on FileNode nodes
type TreeView struct {
	giv.TreeView

	// the parent vectorview
	VectorView *VectorView `copier:"-" json:"-" xml:"-" view:"-"`
}

// SelectNodeInTree selects given node in TreeView
func (gv *VectorView) SelectNodeInTree(kn tree.Node, mode events.SelectModes) {
	tv := gv.TreeView()
	tvn := tv.FindSyncNode(kn)
	if tvn != nil {
		tvn.OpenParents()
		tvn.SelectAction(mode)
	}
}

// SelectedAsTreeViews returns the currently-selected items from SVG as TreeView nodes
func (gv *VectorView) SelectedAsTreeViews() []giv.TreeViewer {
	es := &gv.EditState
	sl := es.SelectedList(false)
	if len(sl) == 0 {
		return nil
	}
	tv := gv.TreeView()
	var tvl []giv.TreeViewer
	for _, si := range sl {
		tvn := tv.FindSyncNode(si.This())
		if tvn != nil {
			tvl = append(tvl, tvn)
		}
	}
	return tvl
}

// DuplicateSelected duplicates selected items in SVG view, using TreeView methods
func (gv *VectorView) DuplicateSelected() { //gti:add
	tvl := gv.SelectedAsTreeViews()
	if len(tvl) == 0 {
		gv.SetStatus("Duplicate: no tree items found")
		return
	}
	sv := gv.SVG()
	sv.UndoSave("DuplicateSelected", "")
	// sv.SetFullReRender()
	tv := gv.TreeView()
	// tv.SetFullReRender()
	for _, tvi := range tvl {
		tvi.AsTreeView().DuplicateSync()
	}
	gv.SetStatus("Duplicated selected items")
	tv.ReSync() // todo: should not be needed
	gv.ChangeMade()
}

// CopySelected copies selected items in SVG view, using TreeView methods
func (gv *VectorView) CopySelected() { //gti:add
	tvl := gv.SelectedAsTreeViews()
	if len(tvl) == 0 {
		gv.SetStatus("Copy: no tree items found")
		return
	}
	tv := gv.TreeView()
	tv.SetSelectedViews(tvl)
	tvl[0].Copy(true) // operates on first element in selection
	gv.SetStatus("Copied selected items")
}

// CutSelected cuts selected items in SVG view, using TreeView methods
func (gv *VectorView) CutSelected() { //gti:add
	tvl := gv.SelectedAsTreeViews()
	if len(tvl) == 0 {
		gv.SetStatus("Cut: no tree items found")
		return
	}
	sv := gv.SVG()
	sv.UndoSave("CutSelected", "")
	// sv.SetFullReRender()
	sv.EditState().ResetSelected()
	tv := gv.TreeView()
	// tv.SetFullReRender()
	tv.SetSelectedViews(tvl)
	tvl[0].Cut() // operates on first element in selection
	gv.SetStatus("Cut selected items")
	tv.ReSync() // todo: should not be needed
	sv.UpdateSelSprites()
	gv.ChangeMade()
}

// PasteClip pastes clipboard, using cur layer etc
func (gv *VectorView) PasteClip() { //gti:add
	md := gv.Clipboard().Read([]string{fi.DataJson})
	if md == nil {
		return
	}
	// es := &gv.EditState
	sv := gv.SVG()
	sv.UndoSave("Paste", "")
	// sv.SetFullReRender()
	tv := gv.TreeView()
	// tv.SetFullReRender()
	// parent := tv
	// if es.CurLayer != "" {
	// 	ly := tv.ChildByName("tv_"+es.CurLayer, 1)
	// 	if ly != nil {
	// 		parent = ly.Embed(KiT_TreeView).(*TreeView)
	// 	}
	// }
	// par.PasteChildren(md, dnd.DropCopy)
	gv.SetStatus("Pasted items from clipboard")
	tv.ReSync() // todo: should not be needed
	gv.ChangeMade()
}

// DeleteSelected deletes selected items in SVG view, using TreeView methods
func (gv *VectorView) DeleteSelected() {
	tvl := gv.SelectedAsTreeViews()
	if len(tvl) == 0 {
		gv.SetStatus("Delete: no tree items found")
		return
	}
	sv := gv.SVG()
	sv.UndoSave("DeleteSelected", "")
	sv.EditState().ResetSelected()
	// sv.SetFullReRender()
	tv := gv.TreeView()
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
//  TreeView

/*
// TreeViewIsLayerFunc is an ActionUpdateFunc that activates if node is a Layer
var TreeViewIsLayerFunc = giv.ActionUpdateFunc(func(fni any, act *gi.Button) {
	tv := fni.(tree.Node).Embed(KiT_TreeView).(*TreeView)
	sn := tv.SrcNode
	if sn != nil {
		act.SetInactiveState(!NodeIsLayer(sn))
	}
})

// ParVectorView returns the parent VectorView
func (tv *TreeView) ParVectorView() *VectorView {
	rtv := tv.RootView.Embed(KiT_TreeView).(*TreeView)
	return rtv.VectorView
}
*/

// SelectSVG
func (tv *TreeView) SelectSVG() {
	gv := tv.VectorView
	if gv != nil {
		gv.SelectNodeInSVG(tv.SyncNode, events.SelectOne)
	}
}

// LayerIsCurrent returns true if layer is the current active one for creating
func (tv *TreeView) LayerIsCurrent() bool {
	gv := tv.VectorView
	if gv != nil {
		return gv.IsCurLayer(tv.SyncNode.Name())
	}
	return false
}

// LayerSetCurrent sets this layer as the current layer name
func (tv *TreeView) LayerSetCurrent() {
	sn := tv.SyncNode
	gv := tv.VectorView
	if gv != nil {
		cur := gv.EditState.CurLayer
		if cur != "" {
			cli := tv.Par.ChildByName("tv_"+cur, 0)
			if cli != nil {
				cl := cli.(*TreeView)
				cl.LayerClearCurrent()
			}
		}
		if LayerIsLocked(sn) {
			tv.LayerToggleLock()
		}
		if !LayerIsVisible(sn) {
			tv.LayerToggleVis()
		}
		gv.SetCurLayer(sn.Name())
		// tv.SetFullReRender() // needed for icon updating
		// tv.UpdateSig()
	}
}

// LayerClearCurrent clears this layer as the current layer if it was set as such.
func (tv *TreeView) LayerClearCurrent() {
	gv := tv.VectorView
	if gv != nil {
		gv.ClearCurLayer(tv.SyncNode.Name())
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
func (tv *TreeView) LayerToggleLock() {
	sn := tv.SyncNode
	np := ""
	if LayerIsLocked(sn) {
		np = "false"
	} else {
		tv.LayerClearCurrent()
		np = "true"
	}
	sn.SetProp("insensitive", np)
	// tv.SetFullReRenderIconLabel()
	// tv.UpdateSig()
}

// LayerToggleVis toggles visibility of the layer
func (tv *TreeView) LayerToggleVis() {
	sn := tv.SyncNode
	np := ""
	if LayerIsVisible(sn) {
		np = "display:none"
		tv.LayerClearCurrent()
	} else {
		np = "display:inline"
	}
	sn.SetProp("style", np)
	// tv.UpdateSig()
}

/*
var TreeViewProps = ki.Props{
	".svgnode": ki.Props{
		"font-weight": gist.WeightNormal,
		"font-style":  gist.FontNormal,
	},
	".layer": ki.Props{
		"font-weight": gist.WeightBold,
	},
	".invisible": ki.Props{
		"font-style": gist.FontItalic,
	},
	".locked": ki.Props{
		"color": "#ff4252",
	},
	giv.TreeViewSelectors[giv.TreeViewActive]: ki.Props{},
	giv.TreeViewSelectors[giv.TreeViewSel]: ki.Props{
		"background-color": &gi.Settings.Colors.Select,
	},
	giv.TreeViewSelectors[giv.TreeViewFocus]: ki.Props{
		"background-color": &gi.Settings.Colors.Control,
	},
	"CtxtMenuActive": ki.PropSlice{
		{"SrcEdit", ki.Props{
			"label": "Edit",
		}},
		{"SelectSVG", ki.Props{
			"label": "Select",
		}},
		{"sep-edit", ki.BlankProp{}},
		{"SrcDuplicate", ki.Props{
			"label":    "Duplicate",
			"shortcut": keyfun.Duplicate,
		}},
		{"Copy", ki.Props{
			"shortcut": keyfun.Copy,
			"Args": ki.PropSlice{
				{"reset", ki.Props{
					"value": true,
				}},
			},
		}},
		{"Cut", ki.Props{
			"shortcut": keyfun.Cut,
			"updtfunc": giv.ActionUpdateFunc(func(tvi any, act *gi.Button) {
				tv := tvi.(tree.Node).Embed(KiT_TreeView).(*TreeView)
				act.SetInactiveState(tv.IsRootOrField(""))
			}),
		}},
		{"Paste", ki.Props{
			"shortcut": keyfun.Paste,
		}},
		{"sep-layer", ki.BlankProp{}},
		{"LayerSetCurrent", ki.Props{
			"label":    "Layer: Set Current",
			"updtfunc": TreeViewIsLayerFunc,
		}},
		{"LayerToggleLock", ki.Props{
			"label":    "Layer: Toggle Lock",
			"updtfunc": TreeViewIsLayerFunc,
		}},
		{"LayerToggleVis", ki.Props{
			"label":    "Layer: Toggle Visible",
			"updtfunc": TreeViewIsLayerFunc,
		}},
		{"sep-open", ki.BlankProp{}},
		{"OpenAll", ki.Props{}},
		{"CloseAll", ki.Props{}},
	},
}

func (tv *TreeView) Style2D() {
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
		tv.StyleTreeView()
		tv.LayState.SetFromStyle(&tv.Sty.Layout) // also does reset
	}
}
*/
