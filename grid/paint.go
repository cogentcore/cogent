// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"fmt"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/girl"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/svg"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// PaintView provides editing of basic Stroke and Fill painting parameters
// for selected items
type PaintView struct {
	gi.Layout
	StrokeType  PaintTypes `desc:"paint type for stroke"`
	StrokeStops string     `desc:"name of gradient with stops"`
	FillType    PaintTypes `desc:"paint type for fill"`
	FillStops   string     `desc:"name of gradient with stops"`
	GridView    *GridView  `copy:"-" json:"-" xml:"-" view:"-" desc:"the parent gridview"`
}

var KiT_PaintView = kit.Types.AddType(&PaintView{}, PaintViewProps)

/////////////////////////////////////////////////////////////////////////
//  Actions

// ManipAction manages all the updating etc associated with performing an action
// that includes an ongoing manipulation with a final non-manip update.
// runs given function to actually do the update.
func (gv *GridView) ManipAction(act, data string, manip bool, fun func(sii svg.NodeSVG)) {
	es := &gv.EditState
	sv := gv.SVG()
	updt := false
	sv.SetFullReRender()
	actStart := false
	finalAct := false
	if !manip && es.InAction() {
		finalAct = true
	}
	if manip && !es.InAction() {
		manip = false
		actStart = true
		es.ActStart(act, data)
		es.ActUnlock()
	}
	if !manip {
		if !finalAct {
			sv.UndoSave(act, data)
		}
		updt = sv.UpdateStart()
	}
	for itm := range es.Selected {
		gv.ManipActionFun(itm, fun)
	}
	if !manip {
		sv.UpdateEnd(updt)
		if !actStart {
			es.ActDone()
		}
	} else {
		sv.ManipUpdate()
	}
}

func (gv *GridView) ManipActionFun(sii svg.NodeSVG, fun func(itm svg.NodeSVG)) {
	if gp, isgp := sii.(*svg.Group); isgp {
		for _, kid := range gp.Kids {
			gv.ManipActionFun(kid.(svg.NodeSVG), fun)
		}
		return
	}
	fun(sii)
}

// SetStrokeNode sets the stroke properties of Node
// based on previous and current PaintType
func (gv *GridView) SetStrokeNode(sii svg.NodeSVG, prev, pt PaintTypes, sp string) {
	if gp, isgp := sii.(*svg.Group); isgp {
		for _, kid := range gp.Kids {
			gv.SetStrokeNode(kid.(svg.NodeSVG), prev, pt, sp)
		}
		return
	}
	switch pt {
	case PaintLinear:
		svg.UpdateNodeGradientProp(sii, "stroke", false, sp)
	case PaintRadial:
		svg.UpdateNodeGradientProp(sii, "stroke", true, sp)
	default:
		if prev == PaintLinear || prev == PaintRadial {
			pstr := kit.ToString(sii.Prop("stroke"))
			svg.DeleteNodeGradient(sii, pstr)
		}
		sii.SetProp("stroke", sp)
	}
	gv.UpdateMarkerColors(sii)
}

// SetStroke sets the stroke properties of selected items
// based on previous and current PaintType
func (gv *GridView) SetStroke(prev, pt PaintTypes, sp string) {
	es := &gv.EditState
	sv := gv.SVG()
	sv.UndoSave("SetStroke", sp)
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	for itm := range es.Selected {
		gv.SetStrokeNode(itm, prev, pt, sp)
	}
	sv.UpdateEnd(updt)
}

// SetStrokeWidthNode sets the stroke width of Node
func (gv *GridView) SetStrokeWidthNode(sii svg.NodeSVG, wp string) {
	if gp, isgp := sii.(*svg.Group); isgp {
		for _, kid := range gp.Kids {
			gv.SetStrokeWidthNode(kid.(svg.NodeSVG), wp)
		}
		return
	}
	g := sii.AsSVGNode()
	if !g.Pnt.StrokeStyle.Color.IsNil() {
		g.SetProp("stroke-width", wp)
	}
}

// SetStrokeWidth sets the stroke width property for selected items
// manip means currently being manipulated -- don't save undo.
func (gv *GridView) SetStrokeWidth(wp string, manip bool) {
	es := &gv.EditState
	sv := gv.SVG()
	updt := false
	if !manip {
		sv.UndoSave("SetStrokeWidth", wp)
		updt = sv.UpdateStart()
		sv.SetFullReRender()
	}
	for itm := range es.Selected {
		gv.SetStrokeWidthNode(itm.(svg.NodeSVG), wp)
	}
	if !manip {
		sv.UpdateEnd(updt)
	} else {
		sv.ManipUpdate()
	}
}

// SetStrokeColor sets the stroke color for selected items.
// manip means currently being manipulated -- don't save undo.
func (gv *GridView) SetStrokeColor(sp string, manip bool) {
	gv.ManipAction("SetStrokeColor", sp, manip,
		func(itm svg.NodeSVG) {
			p := itm.Prop("stroke")
			if p != nil {
				itm.SetProp("stroke", sp)
				gv.UpdateMarkerColors(itm)
			}
		})
}

// SetMarkerNode sets the marker properties of Node.
func (gv *GridView) SetMarkerNode(sii svg.NodeSVG, start, mid, end string, sc, mc, ec MarkerColors) {
	if gp, isgp := sii.(*svg.Group); isgp {
		for _, kid := range gp.Kids {
			gv.SetMarkerNode(kid.(svg.NodeSVG), start, mid, end, sc, mc, ec)
		}
		return
	}
	sv := gv.SVG()
	MarkerSetProp(&sv.SVG, sii, "marker-start", start, sc)
	MarkerSetProp(&sv.SVG, sii, "marker-mid", mid, mc)
	MarkerSetProp(&sv.SVG, sii, "marker-end", end, ec)
}

// SetMarkerProps sets the marker props
func (gv *GridView) SetMarkerProps(start, mid, end string, sc, mc, ec MarkerColors) {
	es := &gv.EditState
	sv := gv.SVG()
	sv.UndoSave("SetMarkerProps", start+" "+mid+" "+end)
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	for itm := range es.Selected {
		gv.SetMarkerNode(itm, start, mid, end, sc, mc, ec)
	}
	sv.UpdateEnd(updt)
}

// UpdateMarkerColors updates the marker colors, when setting fill or stroke
func (gv *GridView) UpdateMarkerColors(sii svg.NodeSVG) {
	if sii == nil {
		return
	}
	sv := gv.SVG()
	MarkerUpdateColorProp(&sv.SVG, sii, "marker-start")
	MarkerUpdateColorProp(&sv.SVG, sii, "marker-mid")
	MarkerUpdateColorProp(&sv.SVG, sii, "marker-end")
}

// SetDashNode sets the stroke-dasharray property of Node.
// multiplies dash values by the line width in dots.
func (gv *GridView) SetDashNode(sii svg.NodeSVG, dary []float64) {
	if gp, isgp := sii.(*svg.Group); isgp {
		for _, kid := range gp.Kids {
			gv.SetDashNode(kid.(svg.NodeSVG), dary)
		}
		return
	}
	if len(dary) == 0 {
		sii.DeleteProp("stroke-dasharray")
		return
	}
	g := sii.AsSVGNode()
	mary := DashMulWidth(float64(g.Pnt.StrokeStyle.Width.Dots), dary)
	ds := DashString(mary)
	sii.SetProp("stroke-dasharray", ds)
}

// SetDashProps sets the dash props
func (gv *GridView) SetDashProps(dary []float64) {
	es := &gv.EditState
	sv := gv.SVG()
	sv.UndoSave("SetDashProps", "")
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	for itm := range es.Selected {
		gv.SetDashNode(itm, dary)
	}
	sv.UpdateEnd(updt)
}

// SetFillNode sets the fill props of given node
// based on previous and current PaintType
func (gv *GridView) SetFillNode(sii svg.NodeSVG, prev, pt PaintTypes, fp string) {
	if gp, isgp := sii.(*svg.Group); isgp {
		for _, kid := range gp.Kids {
			gv.SetFillNode(kid.(svg.NodeSVG), prev, pt, fp)
		}
		return
	}
	switch pt {
	case PaintLinear:
		svg.UpdateNodeGradientProp(sii, "fill", false, fp)
	case PaintRadial:
		svg.UpdateNodeGradientProp(sii, "fill", true, fp)
	default:
		if prev == PaintLinear || prev == PaintRadial {
			pstr := kit.ToString(sii.Prop("fill"))
			svg.DeleteNodeGradient(sii, pstr)
		}
		sii.SetProp("fill", fp)
	}
	gv.UpdateMarkerColors(sii)
}

// SetFill sets the fill props of selected items
// based on previous and current PaintType
func (gv *GridView) SetFill(prev, pt PaintTypes, fp string) {
	es := &gv.EditState
	sv := gv.SVG()
	sv.UndoSave("SetFill", fp)
	updt := sv.UpdateStart()
	sv.SetFullReRender()
	for itm := range es.Selected {
		gv.SetFillNode(itm, prev, pt, fp)
	}
	sv.UpdateEnd(updt)
}

// SetFillColor sets the fill color for selected items
// manip means currently being manipulated -- don't save undo.
func (gv *GridView) SetFillColor(fp string, manip bool) {
	gv.ManipAction("SetFillColor", fp, manip,
		func(itm svg.NodeSVG) {
			p := itm.Prop("fill")
			if p != nil {
				itm.SetProp("fill", fp)
				gv.UpdateMarkerColors(itm)
			}
		})
}

// DefaultGradient returns the default gradient to use for setting stops
func (gv *GridView) DefaultGradient() string {
	es := &gv.EditState
	sv := gv.SVG()
	if len(gv.EditState.Gradients) == 0 {
		es.ConfigDefaultGradient()
		sv.UpdateGradients(es.Gradients)
	}
	return es.Gradients[0].Name
}

// UpdateGradients updates gradients from EditState
func (gv *GridView) UpdateGradients() {
	es := &gv.EditState
	sv := gv.SVG()
	updt := sv.UpdateStart()
	sv.UpdateGradients(es.Gradients)
	sv.UpdateEnd(updt)
}

///////////////////////////////////////////////////////////////
//  PaintView

// Update updates the current settings based on the values in the given Paint and
// props from node (node can be nil)
func (pv *PaintView) Update(pc *girl.Paint, kn ki.Ki) {
	updt := pv.UpdateStart()
	defer pv.UpdateEnd(updt)

	pv.StrokeType, pv.StrokeStops = pv.DecodeType(kn, &pc.StrokeStyle.Color, "stroke")
	pv.FillType, pv.FillStops = pv.DecodeType(kn, &pc.FillStyle.Color, "fill")

	es := &pv.GridView.EditState
	grl := &es.Gradients

	spt := pv.ChildByName("stroke-lab", 0).ChildByName("stroke-type", 1).(*gi.ButtonBox)
	spt.SelectItem(int(pv.StrokeType))

	ss := pv.ChildByName("stroke-stack", 1).(*gi.Frame)
	ss.SetFullReRender()

	switch pv.StrokeType {
	case PaintSolid:
		ss.StackTop = 1
		sc := ss.ChildByName("stroke-clr", 1).(*giv.ColorView)
		sc.SetColor(pc.StrokeStyle.Color.Color)
	case PaintLinear, PaintRadial:
		ss.StackTop = 2
		sg := ss.ChildByName("stroke-grad", 1).(*giv.TableView)
		sg.SetSlice(grl)
		pv.SelectStrokeGrad()
	default:
		ss.StackTop = 0
	}

	wr := pv.ChildByName("stroke-width", 2)
	wsb := wr.ChildByName("width", 1).(*gi.SpinBox)
	wsb.SetValue(pc.StrokeStyle.Width.Val)
	uncb := wr.ChildByName("width-units", 2).(*gi.ComboBox)
	uncb.SetCurIndex(int(pc.StrokeStyle.Width.Un))

	dshcb := wr.ChildByName("dashes", 3).(*gi.ComboBox)
	nwdsh, dnm := DashMatchArray(float64(pc.StrokeStyle.Width.Dots), pc.StrokeStyle.Dashes)
	if nwdsh {
		dshcb.ItemsFromIconList(AllDashIconNames, false, 0)
	}
	dshcb.SetCurVal(gi.IconName(dnm))

	mkr := pv.ChildByName("stroke-markers", 3)

	ms, _, mc := MarkerFromNodeProp(kn, "marker-start")
	mscb := mkr.ChildByName("marker-start", 0).(*gi.ComboBox)
	mscc := mkr.ChildByName("marker-start-color", 1).(*gi.ComboBox)
	if ms != "" {
		mscb.SetCurVal(MarkerNameToIcon(ms))
		mscc.SetCurIndex(int(mc))
	} else {
		mscb.SetCurIndex(0)
		mscc.SetCurIndex(0)
	}
	ms, _, mc = MarkerFromNodeProp(kn, "marker-mid")
	mmcb := mkr.ChildByName("marker-mid", 2).(*gi.ComboBox)
	mmcc := mkr.ChildByName("marker-mid-color", 3).(*gi.ComboBox)
	if ms != "" {
		mmcb.SetCurVal(MarkerNameToIcon(ms))
		mmcc.SetCurIndex(int(mc))
	} else {
		mmcb.SetCurIndex(0)
		mmcc.SetCurIndex(0)
	}
	ms, _, mc = MarkerFromNodeProp(kn, "marker-end")
	mecb := mkr.ChildByName("marker-end", 4).(*gi.ComboBox)
	mecc := mkr.ChildByName("marker-end-color", 5).(*gi.ComboBox)
	if ms != "" {
		mecb.SetCurVal(MarkerNameToIcon(ms))
		mecc.SetCurIndex(int(mc))
	} else {
		mecb.SetCurIndex(0)
		mecc.SetCurIndex(0)
	}

	fpt := pv.ChildByName("fill-lab", 0).ChildByName("fill-type", 1).(*gi.ButtonBox)
	fpt.SelectItem(int(pv.FillType))

	fs := pv.ChildByName("fill-stack", 1).(*gi.Frame)
	fs.SetFullReRender()

	switch pv.FillType {
	case PaintSolid:
		fs.StackTop = 1
		fc := fs.ChildByName("fill-clr", 1).(*giv.ColorView)
		fc.SetColor(pc.FillStyle.Color.Color)
	case PaintLinear, PaintRadial:
		fs.StackTop = 2
		fg := fs.ChildByName("fill-grad", 1).(*giv.TableView)
		fg.SetSlice(grl)
		pv.SelectFillGrad()
	default:
		fs.StackTop = 0
	}
}

// GradStopsName returns the stopsname for gradient from url
func (pv *PaintView) GradStopsName(gii gi.Node2D, url string) string {
	gr := svg.GradientByName(gii, url)
	if gr == nil {
		return ""
	}
	if gr.StopsName != "" {
		return gr.StopsName
	}
	return gr.Nm
}

// DecodeType decodes the paint type from paint and props
// also returns the name of the gradient if using one.
func (pv *PaintView) DecodeType(kn ki.Ki, cs *gist.ColorSpec, prop string) (PaintTypes, string) {
	pstr := ""
	var gii gi.Node2D
	if kn != nil {
		pstr = kit.ToString(kn.Prop(prop))
		gii = kn.(gi.Node2D)
	}
	ptyp := PaintSolid
	grnm := ""
	switch {
	case pstr == "inherit":
		ptyp = PaintInherit
	case pstr == "none" || cs.IsNil():
		ptyp = PaintOff
	case strings.HasPrefix(pstr, "url(#linear") || (cs.Gradient != nil && !cs.Gradient.IsRadial):
		ptyp = PaintLinear
		grnm = pv.GradStopsName(gii, pstr)
	case strings.HasPrefix(pstr, "url(#radial") || (cs.Gradient != nil && cs.Gradient.IsRadial):
		ptyp = PaintRadial
		grnm = pv.GradStopsName(gii, pstr)
	default:
		ptyp = PaintSolid
	}
	if grnm == "" {
		if prop == "fill" {
			grnm = pv.FillStops
		} else {
			grnm = pv.StrokeStops
		}
	}
	return ptyp, grnm
}

func (pv *PaintView) SelectStrokeGrad() {
	es := &pv.GridView.EditState
	grl := &es.Gradients
	ss := pv.ChildByName("stroke-stack", 1).(*gi.Frame)
	sg := ss.ChildByName("stroke-grad", 1).(*giv.TableView)
	sg.UnselectAllIdxs()
	for i, g := range *grl {
		if g.Name == pv.StrokeStops {
			sg.SelectIdx(i)
			break
		}
	}
}

func (pv *PaintView) SelectFillGrad() {
	es := &pv.GridView.EditState
	grl := &es.Gradients
	fs := pv.ChildByName("fill-stack", 1).(*gi.Frame)
	fg := fs.ChildByName("fill-grad", 1).(*giv.TableView)
	fg.UnselectAllIdxs()
	for i, g := range *grl {
		if g.Name == pv.FillStops {
			fg.SelectIdx(i)
			break
		}
	}
}

func (pv *PaintView) Config(gv *GridView) {
	if pv.HasChildren() {
		return
	}
	updt := pv.UpdateStart()
	pv.StrokeType = PaintSolid
	pv.FillType = PaintSolid

	DashIconsInit()
	MarkerIconsInit()

	pv.GridView = gv
	pv.Lay = gi.LayoutVert
	pv.SetProp("spacing", gi.StdDialogVSpaceUnits)

	spl := gi.AddNewLayout(pv, "stroke-lab", gi.LayoutHoriz)
	gi.AddNewLabel(spl, "stroke-pnt-lab", "<b>Stroke Paint:  </b>")
	spt := gi.AddNewButtonBox(spl, "stroke-type")
	spt.ItemsFromStringList(PaintTypeNames)
	spt.SelectItem(int(pv.StrokeType))
	spt.Mutex = true

	wr := gi.AddNewLayout(pv, "stroke-width", gi.LayoutHoriz)
	gi.AddNewLabel(wr, "width-lab", "Width:  ").SetProp("vertical-align", gist.AlignMiddle)

	wsb := gi.AddNewSpinBox(wr, "width")
	wsb.SetProp("min", 0)
	wsb.SetProp("step", 0.05)
	wsb.SetValue(Prefs.Style.StrokeStyle.Width.Val)
	wsb.SpinBoxSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.IsStrokeOn() {
			pv.GridView.SetStrokeWidth(pv.StrokeWidthProp(), false)
		}
	})

	uncb := gi.AddNewComboBox(wr, "width-units")
	uncb.ItemsFromEnum(units.KiT_Units, true, 0)
	uncb.ComboSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.IsStrokeOn() {
			pv.GridView.SetStrokeWidth(pv.StrokeWidthProp(), false)
		}
	})

	gi.AddNewSpace(wr, "sp1").SetProp("width", units.NewCh(5))

	dshcb := gi.AddNewComboBox(wr, "dashes")
	dshcb.SetProp("width", units.NewCh(15))
	dshcb.ItemsFromIconList(AllDashIconNames, true, 0)
	dshcb.ComboSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.IsStrokeOn() {
			pv.GridView.SetDashProps(pv.StrokeDashProp())
		}
	})

	mkr := gi.AddNewLayout(pv, "stroke-markers", gi.LayoutHoriz)

	mscb := gi.AddNewComboBox(mkr, "marker-start")
	// mscb.SetProp("width", units.NewCh(20))
	mscb.ItemsFromIconList(AllMarkerIconNames, true, 0)
	mscb.ComboSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.IsStrokeOn() {
			pv.GridView.SetMarkerProps(pv.MarkerProps())
		}
	})
	mscc := gi.AddNewComboBox(mkr, "marker-start-color")
	mscc.SetProp("width", units.NewCh(5))
	mscc.ItemsFromStringList(MarkerColorNames, true, 0)
	mscc.ComboSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.IsStrokeOn() {
			pv.GridView.SetMarkerProps(pv.MarkerProps())
		}
	})

	gi.AddNewSeparator(mkr, "sp1", false)

	mmcb := gi.AddNewComboBox(mkr, "marker-mid")
	// mmcb.SetProp("width", units.NewCh(20))
	mmcb.ItemsFromIconList(AllMarkerIconNames, true, 0)
	mmcb.ComboSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.IsStrokeOn() {
			pv.GridView.SetMarkerProps(pv.MarkerProps())
		}
	})
	mmcc := gi.AddNewComboBox(mkr, "marker-mid-color")
	mmcc.SetProp("width", units.NewCh(5))
	mmcc.ItemsFromStringList(MarkerColorNames, true, 0)
	mmcc.ComboSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.IsStrokeOn() {
			pv.GridView.SetMarkerProps(pv.MarkerProps())
		}
	})

	gi.AddNewSeparator(mkr, "sp1", false)

	mecb := gi.AddNewComboBox(mkr, "marker-end")
	// mecb.SetProp("width", units.NewCh(20))
	mecb.ItemsFromIconList(AllMarkerIconNames, true, 0)
	mecb.ComboSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.IsStrokeOn() {
			pv.GridView.SetMarkerProps(pv.MarkerProps())
		}
	})
	mecc := gi.AddNewComboBox(mkr, "marker-end-color")
	mecc.SetProp("width", units.NewCh(5))
	mecc.ItemsFromStringList(MarkerColorNames, true, 0)
	mecc.ComboSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.IsStrokeOn() {
			pv.GridView.SetMarkerProps(pv.MarkerProps())
		}
	})

	////////////////////////////////
	// stroke stack

	ss := gi.AddNewFrame(pv, "stroke-stack", gi.LayoutStacked)
	ss.StackTop = 1
	ss.SetStretchMax()
	ss.SetReRenderAnchor()
	// ss.StackTopOnly = true

	gi.AddNewFrame(ss, "stroke-blank", gi.LayoutHoriz) // nothing

	spt.ButtonSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		prev := pv.StrokeType
		pv.StrokeType = PaintTypes(sig)
		updt := ss.UpdateStart()
		ss.SetFullReRender()
		sp := pv.StrokeProp()
		switch pv.StrokeType {
		case PaintOff, PaintInherit:
			ss.StackTop = 0
		case PaintSolid:
			ss.StackTop = 1
		case PaintLinear, PaintRadial:
			if pv.StrokeStops == "" {
				pv.StrokeStops = pv.GridView.DefaultGradient()
			}
			sp = pv.StrokeStops
			ss.StackTop = 2
			pv.SelectStrokeGrad()
		}
		ss.UpdateEnd(updt)
		pv.GridView.SetStroke(prev, pv.StrokeType, sp)
	})

	sc := giv.AddNewColorView(ss, "stroke-clr")
	sc.SetProp("vertical-align", gist.AlignTop)
	sc.Config()
	sc.SetColor(Prefs.Style.StrokeStyle.Color.Color)
	sc.ViewSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.StrokeType == PaintSolid {
			pv.GridView.SetStrokeColor(pv.StrokeProp(), false) // not manip
		}
	})
	sc.ManipSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.StrokeType == PaintSolid {
			pv.GridView.SetStrokeColor(pv.StrokeProp(), true) // manip
		}
	})

	sg := giv.AddNewTableView(ss, "stroke-grad")
	sg.SetProp("index", true)
	sg.SetProp("toolbar", true)
	sg.SelectedIdx = -1
	sg.SetSlice(&pv.GridView.EditState.Gradients)
	sg.WidgetSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.WidgetSelected) {
			svv, _ := send.(*giv.TableView)
			if svv.SelectedIdx >= 0 {
				pv.StrokeStops = pv.GridView.EditState.Gradients[svv.SelectedIdx].Name
				pv.GridView.SetStroke(pv.StrokeType, pv.StrokeType, pv.StrokeStops) // handles full updating
			}
		}
	})

	gi.AddNewSeparator(pv, "fill-sep", true)

	fpl := gi.AddNewLayout(pv, "fill-lab", gi.LayoutHoriz)
	gi.AddNewLabel(fpl, "fill-pnt-lab", "<b>Fill Paint:  </b>")

	fpt := gi.AddNewButtonBox(fpl, "fill-type")
	fpt.ItemsFromStringList(PaintTypeNames)
	fpt.SelectItem(int(pv.FillType))
	fpt.Mutex = true

	fs := gi.AddNewFrame(pv, "fill-stack", gi.LayoutStacked)
	fs.SetStretchMax()
	fs.StackTop = 1
	fs.SetReRenderAnchor()
	fs.StackTopOnly = true

	gi.AddNewFrame(fs, "fill-blank", gi.LayoutHoriz) // nothing

	fpt.ButtonSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		prev := pv.FillType
		pv.FillType = PaintTypes(sig)
		updt := fs.UpdateStart()
		fs.SetFullReRender()
		fp := pv.FillProp()
		switch pv.FillType {
		case PaintOff, PaintInherit:
			fs.StackTop = 0
		case PaintSolid:
			fs.StackTop = 1
		case PaintLinear, PaintRadial:
			if pv.FillStops == "" {
				pv.FillStops = pv.GridView.DefaultGradient()
			}
			fp = pv.FillStops
			fs.StackTop = 2
			pv.SelectFillGrad()
		}
		fs.UpdateEnd(updt)
		pv.GridView.SetFill(prev, pv.FillType, fp)
	})

	fc := giv.AddNewColorView(fs, "fill-clr")
	fc.SetProp("vertical-align", gist.AlignTop)
	fc.Config()
	fc.SetColor(Prefs.Style.FillStyle.Color.Color)
	fc.ViewSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.FillType == PaintSolid {
			pv.GridView.SetFillColor(pv.FillProp(), false)
		}
	})
	fc.ManipSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if pv.FillType == PaintSolid {
			pv.GridView.SetFillColor(pv.FillProp(), true) // manip
		}
	})

	fg := giv.AddNewTableView(fs, "fill-grad")
	fg.SetProp("index", true)
	fg.SetProp("toolbar", true)
	fg.SelectedIdx = -1
	fg.SetSlice(&pv.GridView.EditState.Gradients)
	fg.WidgetSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.WidgetSelected) {
			svv, _ := send.(*giv.TableView)
			if svv.SelectedIdx >= 0 {
				pv.FillStops = pv.GridView.EditState.Gradients[svv.SelectedIdx].Name
				pv.GridView.SetFill(pv.FillType, pv.FillType, pv.FillStops) // this handles updating gradients etc to use stops
			}
		}
	})
	fg.SliceViewSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		// svv, _ := send.(*giv.TableView)
		if sig == int64(giv.SliceViewDeleted) { // not clear what we can do here
		} else {
			pv.GridView.UpdateGradients()
		}
	})
	fg.ViewSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		// svv, _ := send.(*giv.TableView)
		pv.GridView.UpdateGradients()
	})

	gi.AddNewStretch(pv, "endstr")

	pv.UpdateEnd(updt)
}

// StrokeProp returns the stroke property string according to current settings
func (pv *PaintView) StrokeProp() string {
	ss := pv.ChildByName("stroke-stack", 1).(*gi.Frame)
	switch pv.StrokeType {
	case PaintOff:
		return "none"
	case PaintSolid:
		sc := ss.ChildByName("stroke-clr", 1).(*giv.ColorView)
		return sc.Color.HexString()
	case PaintLinear:
		return pv.StrokeStops
	case PaintRadial:
		return pv.StrokeStops
	case PaintInherit:
		return "inherit"
	}
	return "none"
}

// MarkerProp returns the marker property string according to current settings
// along with color type to set.
func (pv *PaintView) MarkerProps() (start, mid, end string, sc, mc, ec MarkerColors) {
	mkr := pv.ChildByName("stroke-markers", 3)

	mscb := mkr.ChildByName("marker-start", 0).(*gi.ComboBox)
	mscc := mkr.ChildByName("marker-start-color", 1).(*gi.ComboBox)
	start = IconToMarkerName(mscb.CurVal)
	sc = MarkerColors(mscc.CurIndex)

	mmcb := mkr.ChildByName("marker-mid", 2).(*gi.ComboBox)
	mmcc := mkr.ChildByName("marker-mid-color", 3).(*gi.ComboBox)
	mid = IconToMarkerName(mmcb.CurVal)
	mc = MarkerColors(mmcc.CurIndex)

	mecb := mkr.ChildByName("marker-end", 4).(*gi.ComboBox)
	mecc := mkr.ChildByName("marker-end-color", 5).(*gi.ComboBox)
	end = IconToMarkerName(mecb.CurVal)
	ec = MarkerColors(mecc.CurIndex)

	return
}

// IsStrokeOn returns true if stroke is active
func (pv *PaintView) IsStrokeOn() bool {
	return pv.StrokeType >= PaintSolid && pv.StrokeType < PaintInherit
}

// StrokeWidthProp returns stroke-width property
func (pv *PaintView) StrokeWidthProp() string {
	wr := pv.ChildByName("stroke-width", 2)
	wsb := wr.ChildByName("width", 1).(*gi.SpinBox)
	uncb := wr.ChildByName("width-units", 2).(*gi.ComboBox)
	unnm := "px"
	if uncb.CurIndex > 0 {
		unvl := units.Units(uncb.CurIndex)
		unnm = unvl.String()
	}
	return fmt.Sprintf("%g%s", wsb.Value, unnm)
}

// StrokeDashProp returns stroke-dasharray property as an array (nil = none)
// these values need to be multiplied by line widths for each item.
func (pv *PaintView) StrokeDashProp() []float64 {
	wr := pv.ChildByName("stroke-width", 2)
	dshcb := wr.ChildByName("dashes", 3).(*gi.ComboBox)
	if dshcb.CurIndex == 0 {
		return nil
	}
	dnm := kit.ToString(dshcb.CurVal)
	if dnm == "" {
		return nil
	}
	dary, ok := AllDashesMap[dnm]
	if !ok {
		return nil
	}
	return dary
}

// IsFillOn returns true if Fill is active
func (pv *PaintView) IsFillOn() bool {
	return pv.FillType >= PaintSolid && pv.FillType < PaintInherit
}

// FillProp returns the fill property string according to current settings
func (pv *PaintView) FillProp() string {
	fs := pv.ChildByName("fill-stack", 1).(*gi.Frame)
	switch pv.FillType {
	case PaintOff:
		return "none"
	case PaintSolid:
		sc := fs.ChildByName("fill-clr", 1).(*giv.ColorView)
		return sc.Color.HexString()
	case PaintLinear:
		return pv.FillStops
	case PaintRadial:
		return pv.FillStops
	case PaintInherit:
		return "inherit"
	}
	return "none"
}

// SetProps sets the props for given node according to current settings
func (pv *PaintView) SetProps(sii svg.NodeSVG) {
	pv.GridView.SetStrokeNode(sii, pv.StrokeType, pv.StrokeType, pv.StrokeProp())
	if pv.IsStrokeOn() {
		sii.SetProp("stroke-width", pv.StrokeWidthProp())
	}
	pv.GridView.SetFillNode(sii, pv.FillType, pv.FillType, pv.FillProp())
}

var PaintViewProps = ki.Props{
	"EnumType:Flag":    gi.KiT_VpFlags,
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
}

type PaintTypes int

const (
	PaintOff PaintTypes = iota
	PaintSolid
	PaintLinear
	PaintRadial
	PaintInherit
	PaintTypesN
)

var PaintTypeNames = []string{"Off", "Solid", "Linear", "Radial", "Inherit"}

//go:generate stringer -type=PaintTypes

var KiT_PaintTypes = kit.Enums.AddEnum(PaintTypesN, kit.NotBitFlag, nil)

func (ev PaintTypes) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *PaintTypes) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }
