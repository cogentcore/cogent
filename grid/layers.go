// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"fmt"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/svg"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// Layer represents one layer group
type Layer struct {
	Name string

	// visiblity toggle
	Vis bool `desc:"visiblity toggle"`

	// lock toggle
	Lck bool `desc:"lock toggle"`
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
	for _, kc := range sv.Kids {
		if NodeIsLayer(kc) {
			l := &Layer{Name: kc.Name()}
			l.FromNode(kc)
			*ly = append(*ly, l)
		}
	}
}

func (ly *Layers) LayersUpdated(svg *SVGView) {
	si := 1 // starting index -- assuming namedview
	for i, l := range *ly {
		kc := svg.ChildByName(l.Name, si+i)
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
//  GridView

// FirstLayerIndex returns index of first layer group in svg
func (gv *GridView) FirstLayerIndex() int {
	sv := gv.SVG()
	for i, kc := range sv.Kids {
		if NodeIsLayer(kc) {
			return i
		}
	}
	return ints.MinInt(1, len(sv.Kids))
}

func (gv *GridView) LayerViewSigs(lyv *giv.TableView) {
	es := &gv.EditState
	sv := gv.SVG()
	lyv.ViewSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
		// fmt.Printf("tv viewsig: %v  data: %v  send: %v\n", sig, data, send.Path())
		updt := sv.UpdateStart()
		es.Layers.LayersUpdated(sv)
		sv.UpdateEnd(updt)
		gv.UpdateLayerView()
	})

	lyv.SliceViewSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
		svs := giv.SliceViewSignals(sig)
		idx := data.(int)
		fmt.Printf("tv sliceviewsig: %v  data: %v\n", svs.String(), idx)
		switch svs {
		case giv.SliceViewInserted:
			si := gv.FirstLayerIndex()
			li := si + idx
			l := es.Layers[idx]
			l.Name = fmt.Sprintf("Layer%d", li)
			l.Vis = true
			sl := sv.InsertNewChild(svg.KiT_Group, li, l.Name)
			sl.SetProp("groupmode", "layer")
			// todo: move selected into this new group
			gv.UpdateLayerView()
		case giv.SliceViewDeleted:
		}
	})

	lyv.WidgetSig.Connect(gv.This(), func(recv, send ki.Ki, sig int64, data any) {
		fmt.Printf("tv widgetsig: %v  data: %v\n", gi.WidgetSignals(sig).String(), data)
		if sig == int64(gi.WidgetSelected) {
			idx := data.(int)
			ly := es.Layers[idx]
			gv.SetCurLayer(ly.Name)
		}
	})
}

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
		sv.SetChildAdded()
		nk := len(sv.Kids)
		for i := nk - 1; i >= 3; i-- {
			kc := sv.Child(i)
			ki.MoveToParent(kc, l1)
		}
		gv.SetCurLayer(l1.Name())
	} else {
		l1 := sv.InsertNewChild(svg.KiT_Group, si+nl, fmt.Sprintf("Layer%d", nl))
		sv.SetChildAdded()
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
	kn.FuncUp(0, kn, func(k ki.Ki, level int, d any) bool {
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
