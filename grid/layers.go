// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"fmt"

	"github.com/goki/gi/svg"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// Layer represents one layer group
type Layer struct {
	Name string
	Vis  bool `desc:"visiblity toggle"`
	Lck  bool `desc:"lock toggle"`
}

// Layers is the list of all layers
type Layers []*Layer

func (ly *Layers) SyncLayers(svg *SVGView) {
	*ly = make(Layers, 0)
	for _, kc := range svg.Kids {
		if NodeIsLayer(kc) {
			l := &Layer{Name: kc.Name(), Vis: LayerIsVisible(kc), Lck: LayerIsLocked(kc)}
			*ly = append(*ly, l)
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
//  GridView

func (gv *GridView) SyncLayers() {
	sv := gv.SVG()
	gv.EditState.Layers.SyncLayers(sv)
}

func (gv *GridView) UpdateLayerView() {
	gv.SyncLayers()
	es := &gv.EditState
	lys := &es.Layers
	lyv := gv.LayerView()
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
	lyv.ClearSelected()
	lyv.SelectIdx(ci)
}

func (gv *GridView) AddLayer() {
	sv := gv.SVG()
	updt := sv.UpdateStart()
	defer sv.UpdateEnd(updt)

	lys := &gv.EditState.Layers
	lys.SyncLayers(sv)
	nl := len(*lys)
	si := 1 // starting index -- assuming namedview
	if nl == 0 {
		bg := sv.InsertNewChild(svg.KiT_Group, si, "LayerBG")
		bg.SetProp("groupmode", "layer")
		l1 := sv.InsertNewChild(svg.KiT_Group, si+1, "Layer1")
		l1.SetProp("groupmode", "layer")
		nk := len(sv.Kids)
		for i := nk - 1; i >= 3; i-- {
			kc := sv.Child(i)
			ki.MoveToParent(kc, l1)
		}
		gv.SetCurLayer(l1.Name())
	} else {
		l1 := sv.InsertNewChild(svg.KiT_Group, si+nl, fmt.Sprintf("Layer%d", nl))
		l1.SetProp("groupmode", "layer")
		gv.SetCurLayer(l1.Name())
	}
	gv.UpdateLayerView()
}

/////////////////////////////////////////////////////////////////
//  Node

// NodeIsLayer returns true if given node is a layer
func NodeIsLayer(kn ki.Ki) bool {
	gm := kit.ToString(kn.Prop("groupmode"))
	return gm == "layer"
}

// LayerIsLocked returns true if layer is locked (insensitive = true)
func LayerIsLocked(kn ki.Ki) bool {
	cp := kit.ToString(kn.Prop("insensitive"))
	return cp == "true"
}

// LayerIsVisible returns true if layer is visible
func LayerIsVisible(kn ki.Ki) bool {
	cp := kit.ToString(kn.Prop("style"))
	return cp != "display:none"
}

// NodeParentLayer returns the parent group that is a layer -- nil if none
func NodeParentLayer(kn ki.Ki) ki.Ki {
	var parLay ki.Ki
	kn.FuncUp(0, kn, func(k ki.Ki, level int, d interface{}) bool {
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
func (gv *GridView) IsCurLayer(lay string) bool {
	return gv.EditState.CurLayer == lay
}

// SetCurLayer sets the current layer for creating items to given one
func (gv *GridView) SetCurLayer(lay string) {
	gv.EditState.CurLayer = lay
	gv.SetStatus("set current layer to: " + lay)
}

// ClearCurLayer clears the current layer for creating items if it
// was set to the given layer name
func (gv *GridView) ClearCurLayer(lay string) {
	if gv.EditState.CurLayer == lay {
		gv.EditState.CurLayer = ""
		gv.SetStatus("clear current layer from: " + lay)
	}
}
