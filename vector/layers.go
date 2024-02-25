// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"fmt"

	"cogentcore.org/core/giv"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/svg"
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
func (l *Layer) FromNode(k ki.Ki) {
	l.Vis = LayerIsVisible(k)
	l.Lck = LayerIsLocked(k)
}

// ToNode copies state / prop values to given node
func (l *Layer) ToNode(k ki.Ki) {
	if l.Vis {
		k.SetProp("style", "")
		k.SetProp("display", "inline")
	} else {
		k.SetProp("style", "display:none")
		k.SetProp("display", "none")
	}
	k.SetProp("insensitive", l.Lck)
}

// Layers is the list of all layers
type Layers []*Layer

func (ly *Layers) SyncLayers(sv *SVGView) {
	*ly = make(Layers, 0)
	for _, kc := range sv.Root().Kids {
		if NodeIsLayer(kc) {
			l := &Layer{Name: kc.Name()}
			l.FromNode(kc)
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

func (ly *Layers) LayerIdxByName(nm string) int {
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
	for i, kc := range sv.Root().Kids {
		if NodeIsLayer(kc) {
			return i
		}
	}
	return min(1, len(sv.Root().Kids))
}

func (vv *VectorView) LayerViewSigs(lyv *giv.TableView) {
	// es := &gv.EditState
	// sv := gv.SVG()
	// lyv.ViewSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	// fmt.Printf("tv viewsig: %v  data: %v  send: %v\n", sig, data, send.Path())
	// 	updt := sv.UpdateStart()
	// 	es.Layers.LayersUpdated(sv)
	// 	sv.UpdateEnd(updt)
	// 	gv.UpdateLayerView()
	// })

	// lyv.SliceViewSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	svs := giv.SliceViewSignals(sig)
	// 	idx := data.(int)
	// 	fmt.Printf("tv sliceviewsig: %v  data: %v\n", svs.String(), idx)
	// 	switch svs {
	// 	case giv.SliceViewInserted:
	// 		si := gv.FirstLayerIndex()
	// 		li := si + idx
	// 		l := es.Layers[idx]
	// 		l.Name = fmt.Sprintf("Layer%d", li)
	// 		l.Vis = true
	// 		sl := sv.InsertNewChild(svg.KiT_Group, li, l.Name)
	// 		sl.SetProp("groupmode", "layer")
	// 		// todo: move selected into this new group
	// 		gv.UpdateLayerView()
	// 	case giv.SliceViewDeleted:
	// 	}
	// })

	// lyv.WidgetSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	fmt.Printf("tv widgetsig: %v  data: %v\n", gi.WidgetSignals(sig).String(), data)
	// 	if sig == int64(gi.WidgetSelected) {
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
	ci := lys.LayerIdxByName(es.CurLayer)
	if ci < 0 {
		ci = nl - 1
		es.CurLayer = (*lys)[ci].Name
	}
	// lyv.ClearSelected() // todo
	lyv.SelectIdx(ci)
}

// AddLayer adds a new layer
func (vv *VectorView) AddLayer() { //gti:add
	sv := vv.SVG()
	svr := sv.Root()

	lys := &vv.EditState.Layers
	lys.SyncLayers(sv)
	nl := len(*lys)
	si := 1 // starting index -- assuming namedview
	if nl == 0 {
		bg := svr.InsertNewChild(svg.GroupType, si, "LayerBG")
		bg.SetProp("groupmode", "layer")
		l1 := svr.InsertNewChild(svg.GroupType, si+1, "Layer1")
		l1.SetProp("groupmode", "layer")
		svr.SetChildAdded()
		nk := len(svr.Kids)
		for i := nk - 1; i >= 3; i-- {
			kc := svr.Child(i)
			ki.MoveToParent(kc, l1)
		}
		vv.SetCurLayer(l1.Name())
	} else {
		l1 := svr.InsertNewChild(svg.GroupType, si+nl, fmt.Sprintf("Layer%d", nl))
		svr.SetChildAdded()
		l1.SetProp("groupmode", "layer")
		vv.SetCurLayer(l1.Name())
	}
	vv.UpdateLayerView()
}

/////////////////////////////////////////////////////////////////
//  Node

// NodeIsLayer returns true if given node is a layer
func NodeIsLayer(kn ki.Ki) bool {
	gm := laser.ToString(kn.Prop("groupmode"))
	return gm == "layer"
}

// LayerIsLocked returns true if layer is locked (insensitive = true)
func LayerIsLocked(kn ki.Ki) bool {
	b, _ := laser.ToBool(kn.Prop("insensitive"))
	return b
}

// LayerIsVisible returns true if layer is visible
func LayerIsVisible(kn ki.Ki) bool {
	cp := laser.ToString(kn.Prop("style"))
	return cp != "display:none"
}

// NodeParentLayer returns the parent group that is a layer -- nil if none
func NodeParentLayer(kn ki.Ki) ki.Ki {
	var parLay ki.Ki
	kn.WalkUp(func(k ki.Ki) bool {
		if NodeIsLayer(k) {
			parLay = k
			return ki.Break
		}
		return ki.Continue
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
