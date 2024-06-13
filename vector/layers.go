// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"fmt"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/views"
)

// Layer represents one layer group
type Layer struct {
	Name string

	// visiblity toggle
	Vis bool

	// lock toggle
	Lck bool
}

// FromNode copies state / prop values from given node
func (l *Layer) FromNode(k tree.Node) {
	l.Vis = LayerIsVisible(k)
	l.Lck = LayerIsLocked(k)
}

// ToNode copies state / prop values to given node
func (l *Layer) ToNode(n tree.Node) {
	nb := n.AsTree()
	if l.Vis {
		nb.Properties["style"] = ""
		nb.Properties["display"] = "inline"
	} else {
		nb.Properties["style"] = "display:none"
		nb.Properties["display"] = "none"
	}
	nb.Properties["insensitive"] = l.Lck
}

// Layers is the list of all layers
type Layers []*Layer

func (ly *Layers) SyncLayers(sv *SVGView) {
	*ly = make(Layers, 0)
	for _, n := range sv.Root().Children {
		if NodeIsLayer(n) {
			l := &Layer{Name: n.AsTree().Name}
			l.FromNode(n)
			*ly = append(*ly, l)
		}
	}
}

func (ly *Layers) LayersUpdated(sv *SVGView) {
	si := 1 // starting index -- assuming namedview
	for i, l := range *ly {
		kc := sv.Root().ChildByName(l.Name, si+i)
		if kc != nil {
			l.ToNode(kc)
		}
	}
}

func (ly *Layers) LayerIndexByName(nm string) int {
	for i, l := range *ly {
		if l.Name == nm {
			return i
		}
	}
	return -1
}

/////////////////////////////////////////////////////////////////
//  VectorView

// FirstLayerIndex returns index of first layer group in svg
func (vv *VectorView) FirstLayerIndex() int {
	sv := vv.SVG()
	for i, kc := range sv.Root().Children {
		if NodeIsLayer(kc) {
			return i
		}
	}
	return min(1, len(sv.Root().Children))
}

func (vv *VectorView) LayerViewSigs(lyv *views.Table) {
	// es := &gv.EditState
	// sv := gv.SVG()
	// lyv.ViewSig.Connect(gv.This, func(recv, send tree.Node, sig int64, data any) {
	// 	// fmt.Printf("tv viewsig: %v  data: %v  send: %v\n", sig, data, send.Path())
	// 	update := sv.UpdateStart()
	// 	es.Layers.LayersUpdated(sv)
	// 	sv.UpdateEnd(update)
	// 	gv.UpdateLayerView()
	// })

	// lyv.ListSig.Connect(gv.This, func(recv, send tree.Node, sig int64, data any) {
	// 	svs := views.ListSignals(sig)
	// 	idx := data.(int)
	// 	fmt.Printf("tv listsig: %v  data: %v\n", svs.String(), idx)
	// 	switch svs {
	// 	case views.ListInserted:
	// 		si := gv.FirstLayerIndex()
	// 		li := si + idx
	// 		l := es.Layers[idx]
	// 		l.Name = fmt.Sprintf("Layer%d", li)
	// 		l.Vis = true
	// 		sl := sv.InsertNewChild(svg.KiT_Group, li, l.Name)
	// 		sl.SetProp("groupmode"] = "layer"
	// 		// todo: move selected into this new group
	// 		gv.UpdateLayerView()
	// 	case views.ListDeleted:
	// 	}
	// })

	// lyv.WidgetSig.Connect(gv.This, func(recv, send tree.Node, sig int64, data any) {
	// 	fmt.Printf("tv widgetsig: %v  data: %v\n", core.WidgetSignals(sig).String(), data)
	// 	if sig == int64(core.WidgetSelected) {
	// 		idx := data.(int)
	// 		ly := es.Layers[idx]
	// 		gv.SetCurLayer(ly.Name)
	// 	}
	// })
}

func (vv *VectorView) SyncLayers() {
	sv := vv.SVG()
	vv.EditState.Layers.SyncLayers(sv)
}

func (vv *VectorView) UpdateLayerView() {
	vv.SyncLayers()
	es := &vv.EditState
	lys := &es.Layers
	lyv := vv.LayerView()
	lyv.SetSlice(lys)
	nl := len(*lys)
	if nl == 0 {
		return
	}
	ci := lys.LayerIndexByName(es.CurLayer)
	if ci < 0 {
		ci = nl - 1
		es.CurLayer = (*lys)[ci].Name
	}
	// lyv.ClearSelected() // todo
	lyv.SelectIndex(ci)
}

// AddLayer adds a new layer
func (vv *VectorView) AddLayer() { //types:add
	sv := vv.SVG()
	svr := sv.Root()

	lys := &vv.EditState.Layers
	lys.SyncLayers(sv)
	nl := len(*lys)
	si := 1 // starting index -- assuming namedview
	if nl == 0 {
		bg := svr.InsertNewChild(svg.GroupType, si)
		bg.AsTree().SetName("LayerBG")
		bg.AsTree().Properties["groupmode"] = "layer"
		l1 := svr.InsertNewChild(svg.GroupType, si+1)
		l1.AsTree().SetName("Layer1")
		l1.AsTree().Properties["groupmode"] = "layer"
		nk := len(svr.Children)
		for i := nk - 1; i >= 3; i-- {
			kc := svr.Child(i)
			tree.MoveToParent(kc, l1)
		}
		vv.SetCurLayer(l1.AsTree().Name)
	} else {
		l1 := svr.InsertNewChild(svg.GroupType, si+nl)
		l1.AsTree().SetName(fmt.Sprintf("Layer%d", nl))
		l1.AsTree().Properties["groupmode"] = "layer"
		vv.SetCurLayer(l1.AsTree().Name)
	}
	vv.UpdateLayerView()
}

/////////////////////////////////////////////////////////////////
//  Node

// NodeIsLayer returns true if given node is a layer
func NodeIsLayer(kn tree.Node) bool {
	gm := reflectx.ToString(kn.AsTree().Property("groupmode"))
	return gm == "layer"
}

// LayerIsLocked returns true if layer is locked (insensitive = true)
func LayerIsLocked(kn tree.Node) bool {
	b, _ := reflectx.ToBool(kn.AsTree().Property("insensitive"))
	return b
}

// LayerIsVisible returns true if layer is visible
func LayerIsVisible(kn tree.Node) bool {
	cp := reflectx.ToString(kn.AsTree().Property("style"))
	return cp != "display:none"
}

// NodeParentLayer returns the parent group that is a layer -- nil if none
func NodeParentLayer(n tree.Node) tree.Node {
	var parLay tree.Node
	n.AsTree().WalkUp(func(pn tree.Node) bool {
		if NodeIsLayer(pn) {
			parLay = pn
			return tree.Break
		}
		return tree.Continue
	})
	return parLay
}

// IsCurLayer returns true if given layer is the current layer
// for creating items
func (vv *VectorView) IsCurLayer(lay string) bool {
	return vv.EditState.CurLayer == lay
}

// SetCurLayer sets the current layer for creating items to given one
func (vv *VectorView) SetCurLayer(lay string) {
	vv.EditState.CurLayer = lay
	vv.SetStatus("set current layer to: " + lay)
}

// ClearCurLayer clears the current layer for creating items if it
// was set to the given layer name
func (vv *VectorView) ClearCurLayer(lay string) {
	if vv.EditState.CurLayer == lay {
		vv.EditState.CurLayer = ""
		vv.SetStatus("clear current layer from: " + lay)
	}
}
