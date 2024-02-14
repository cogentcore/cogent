// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

import (
	"fmt"

	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/units"
)

// PaintView provides editing of basic Stroke and Fill painting parameters
// for selected items
type PaintView struct {
	gi.Layout

	// paint type for stroke
	StrokeType PaintTypes

	// name of gradient with stops
	StrokeStops string

	// paint type for fill
	FillType PaintTypes

	// name of gradient with stops
	FillStops string

	// the parent vectorview
	VectorView *VectorView `copier:"-" json:"-" xml:"-" view:"-"`
}

/////////////////////////////////////////////////////////////////////////
//  Actions

// ManipAction manages all the updating etc associated with performing an action
// that includes an ongoing manipulation with a final non-manip update.
// runs given function to actually do the update.
func (vv *VectorView) ManipAction(act, data string, manip bool, fun func(sii svg.Node)) {
	es := &vv.EditState
	sv := vv.SVG()
	// updt := false
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
		// updt = sv.UpdateStart()
	}
	for itm := range es.Selected {
		vv.ManipActionFun(itm, fun)
	}
	if !manip {
		// sv.UpdateEnd(updt)
		if !actStart {
			es.ActDone()
			vv.ChangeMade()
		}
	} else {
		sv.ManipUpdate()
	}
}

func (vv *VectorView) ManipActionFun(sii svg.Node, fun func(itm svg.Node)) {
	if gp, isgp := sii.(*svg.Group); isgp {
		for _, kid := range gp.Kids {
			vv.ManipActionFun(kid.(svg.Node), fun)
		}
		return
	}
	fun(sii)
}

// SetColorNode sets the color properties of Node
// based on previous and current PaintType
func (vv *VectorView) SetColorNode(sii svg.Node, prop string, prev, pt PaintTypes, sp string) {
	if gp, isgp := sii.(*svg.Group); isgp {
		for _, kid := range gp.Kids {
			vv.SetColorNode(kid.(svg.Node), prop, prev, pt, sp)
		}
		return
	}
	switch pt {
	// case PaintLinear:
	// 	svg.UpdateNodeGradientProp(sii, prop, false, sp)
	// case PaintRadial:
	// 	svg.UpdateNodeGradientProp(sii, prop, true, sp)
	default:
		if prev == PaintLinear || prev == PaintRadial {
			pstr := laser.ToString(sii.Prop(prop))
			_ = pstr
			// svg.DeleteNodeGradient(sii, pstr)
		}
		sii.AsNodeBase().SetColorProps(prop, sp)
	}
	vv.UpdateMarkerColors(sii)
}

// SetStroke sets the stroke properties of selected items
// based on previous and current PaintType
func (vv *VectorView) SetStroke(prev, pt PaintTypes, sp string) {
	es := &vv.EditState
	sv := vv.SVG()
	sv.UndoSave("SetStroke", sp)
	// updt := sv.UpdateStart()
	for itm := range es.Selected {
		vv.SetColorNode(itm, "stroke", prev, pt, sp)
	}
	// sv.UpdateEnd(updt)
	vv.ChangeMade()
}

// SetStrokeWidthNode sets the stroke width of Node
func (vv *VectorView) SetStrokeWidthNode(sii svg.Node, wp string) {
	if gp, isgp := sii.(*svg.Group); isgp {
		for _, kid := range gp.Kids {
			vv.SetStrokeWidthNode(kid.(svg.Node), wp)
		}
		return
	}
	g := sii.AsNodeBase()
	if g.Paint.StrokeStyle.Color != nil {
		g.SetProp("stroke-width", wp)
	}
}

// SetStrokeWidth sets the stroke width property for selected items
// manip means currently being manipulated -- don't save undo.
func (vv *VectorView) SetStrokeWidth(wp string, manip bool) {
	es := &vv.EditState
	sv := vv.SVG()
	// updt := false
	if !manip {
		sv.UndoSave("SetStrokeWidth", wp)
		// updt = sv.UpdateStart()
		// sv.SetFullReRender()
	}
	for itm := range es.Selected {
		vv.SetStrokeWidthNode(itm.(svg.Node), wp)
	}
	if !manip {
		// sv.UpdateEnd(updt)
		vv.ChangeMade()
	} else {
		sv.ManipUpdate()
	}
}

// SetStrokeColor sets the stroke color for selected items.
// manip means currently being manipulated -- don't save undo.
func (vv *VectorView) SetStrokeColor(sp string, manip bool) {
	vv.ManipAction("SetStrokeColor", sp, manip,
		func(itm svg.Node) {
			p := itm.Prop("stroke")
			if p != nil {
				itm.AsNodeBase().SetColorProps("stroke", sp)
				vv.UpdateMarkerColors(itm)
			}
		})
}

// SetMarkerNode sets the marker properties of Node.
func (vv *VectorView) SetMarkerNode(sii svg.Node, start, mid, end string, sc, mc, ec MarkerColors) {
	if gp, isgp := sii.(*svg.Group); isgp {
		for _, kid := range gp.Kids {
			vv.SetMarkerNode(kid.(svg.Node), start, mid, end, sc, mc, ec)
		}
		return
	}
	sv := vv.SVG()
	MarkerSetProp(&sv.SVG, sii, "marker-start", start, sc)
	MarkerSetProp(&sv.SVG, sii, "marker-mid", mid, mc)
	MarkerSetProp(&sv.SVG, sii, "marker-end", end, ec)
}

// SetMarkerProps sets the marker props
func (vv *VectorView) SetMarkerProps(start, mid, end string, sc, mc, ec MarkerColors) {
	es := &vv.EditState
	sv := vv.SVG()
	sv.UndoSave("SetMarkerProps", start+" "+mid+" "+end)
	// updt := sv.UpdateStart()
	// sv.SetFullReRender()
	for itm := range es.Selected {
		vv.SetMarkerNode(itm, start, mid, end, sc, mc, ec)
	}
	// sv.UpdateEnd(updt)
	vv.ChangeMade()
}

// UpdateMarkerColors updates the marker colors, when setting fill or stroke
func (vv *VectorView) UpdateMarkerColors(sii svg.Node) {
	if sii == nil {
		return
	}
	sv := vv.SVG()
	MarkerUpdateColorProp(&sv.SVG, sii, "marker-start")
	MarkerUpdateColorProp(&sv.SVG, sii, "marker-mid")
	MarkerUpdateColorProp(&sv.SVG, sii, "marker-end")
}

// SetDashNode sets the stroke-dasharray property of Node.
// multiplies dash values by the line width in dots.
func (vv *VectorView) SetDashNode(sii svg.Node, dary []float64) {
	if gp, isgp := sii.(*svg.Group); isgp {
		for _, kid := range gp.Kids {
			vv.SetDashNode(kid.(svg.Node), dary)
		}
		return
	}
	if len(dary) == 0 {
		sii.DeleteProp("stroke-dasharray")
		return
	}
	g := sii.AsNodeBase()
	mary := DashMulWidth(float64(g.Paint.StrokeStyle.Width.Dots), dary)
	ds := DashString(mary)
	sii.SetProp("stroke-dasharray", ds)
}

// SetDashProps sets the dash props
func (vv *VectorView) SetDashProps(dary []float64) {
	es := &vv.EditState
	sv := vv.SVG()
	sv.UndoSave("SetDashProps", "")
	// updt := sv.UpdateStart()
	// sv.SetFullReRender()
	for itm := range es.Selected {
		vv.SetDashNode(itm, dary)
	}
	// sv.UpdateEnd(updt)
	vv.ChangeMade()
}

// SetFill sets the fill props of selected items
// based on previous and current PaintType
func (vv *VectorView) SetFill(prev, pt PaintTypes, fp string) {
	es := &vv.EditState
	sv := vv.SVG()
	sv.UndoSave("SetFill", fp)
	// updt := sv.UpdateStart()
	// sv.SetFullReRender()
	for itm := range es.Selected {
		vv.SetColorNode(itm, "fill", prev, pt, fp)
	}
	// sv.UpdateEnd(updt)
	vv.ChangeMade()
}

// SetFillColor sets the fill color for selected items
// manip means currently being manipulated -- don't save undo.
func (vv *VectorView) SetFillColor(fp string, manip bool) {
	vv.ManipAction("SetFillColor", fp, manip,
		func(itm svg.Node) {
			p := itm.Prop("fill")
			if p != nil {
				itm.AsNodeBase().SetColorProps("fill", fp)
				vv.UpdateMarkerColors(itm)
			}
		})
}

// DefaultGradient returns the default gradient to use for setting stops
func (vv *VectorView) DefaultGradient() string {
	es := &vv.EditState
	sv := vv.SVG()
	if len(vv.EditState.Gradients) == 0 {
		es.ConfigDefaultGradient()
		sv.UpdateGradients(es.Gradients)
	}
	return es.Gradients[0].Name
}

// UpdateGradients updates gradients from EditState
func (vv *VectorView) UpdateGradients() {
	es := &vv.EditState
	sv := vv.SVG()
	// updt := sv.UpdateStart()
	sv.UpdateGradients(es.Gradients)
	// sv.UpdateEnd(updt)
}

///////////////////////////////////////////////////////////////
//  PaintView

// Update updates the current settings based on the values in the given Paint and
// props from node (node can be nil)
/*
func (pv *PaintView) Update(pc *paint.Paint, kn ki.Ki) {
	updt := pv.UpdateStart()
	defer pv.UpdateEnd(updt)

	pv.StrokeType, pv.StrokeStops = pv.DecodeType(kn, &pc.StrokeStyle.Color, "stroke")
	pv.FillType, pv.FillStops = pv.DecodeType(kn, &pc.FillStyle.Color, "fill")

	es := &pv.VectorView.EditState
	grl := &es.Gradients

	spt := pv.ChildByName("stroke-lab", 0).ChildByName("stroke-type", 1).(*gi.ButtonBox)
	spt.SelectItem(int(pv.StrokeType))

	ss := pv.StrokeStack()

	switch pv.StrokeType {
	case PaintSolid:
		if ss.StackTop != 1 {
			ss.SetFullReRender()
		}
		ss.StackTop = 1
		sc := ss.ChildByName("stroke-clr", 1).(*giv.ColorView)
		sc.SetColor(pc.StrokeStyle.Color.Color)
	case PaintLinear, PaintRadial:
		if ss.StackTop != 2 {
			ss.SetFullReRender()
		}
		ss.StackTop = 2
		sg := ss.ChildByName("stroke-grad", 1).(*giv.TableView)
		sg.SetSlice(grl)
		pv.SelectStrokeGrad()
	default:
		if ss.StackTop != 0 {
			ss.SetFullReRender()
		}
		ss.StackTop = 0
	}

	wr := pv.ChildByName("stroke-width", 2)
	wsb := wr.ChildByName("width", 1).(*gi.Spinner)
	wsb.SetValue(pc.StrokeStyle.Width.Val)
	uncb := wr.ChildByName("width-units", 2).(*gi.Chooser)
	uncb.SetCurrentIndex(int(pc.StrokeStyle.Width.Un))

	dshcb := wr.ChildByName("dashes", 3).(*gi.Chooser)
	nwdsh, dnm := DashMatchArray(float64(pc.StrokeStyle.Width.Dots), pc.StrokeStyle.Dashes)
	if nwdsh {
		dshcb.ItemsFromIconList(AllDashIconNames, false, 0)
	}
	dshcb.SetCurVal(icons.Icon(dnm))

	mkr := pv.ChildByName("stroke-markers", 3)

	ms, _, mc := MarkerFromNodeProp(kn, "marker-start")
	mscb := mkr.ChildByName("marker-start", 0).(*gi.Chooser)
	mscc := mkr.ChildByName("marker-start-color", 1).(*gi.Chooser)
	if ms != "" {
		mscb.SetCurVal(MarkerNameToIcon(ms))
		mscc.SetCurrentIndex(int(mc))
	} else {
		mscb.SetCurrentIndex(0)
		mscc.SetCurrentIndex(0)
	}
	ms, _, mc = MarkerFromNodeProp(kn, "marker-mid")
	mmcb := mkr.ChildByName("marker-mid", 2).(*gi.Chooser)
	mmcc := mkr.ChildByName("marker-mid-color", 3).(*gi.Chooser)
	if ms != "" {
		mmcb.SetCurVal(MarkerNameToIcon(ms))
		mmcc.SetCurrentIndex(int(mc))
	} else {
		mmcb.SetCurrentIndex(0)
		mmcc.SetCurrentIndex(0)
	}
	ms, _, mc = MarkerFromNodeProp(kn, "marker-end")
	mecb := mkr.ChildByName("marker-end", 4).(*gi.Chooser)
	mecc := mkr.ChildByName("marker-end-color", 5).(*gi.Chooser)
	if ms != "" {
		mecb.SetCurVal(MarkerNameToIcon(ms))
		mecc.SetCurrentIndex(int(mc))
	} else {
		mecb.SetCurrentIndex(0)
		mecc.SetCurrentIndex(0)
	}

	fpt := pv.ChildByName("fill-lab", 0).ChildByName("fill-type", 1).(*gi.ButtonBox)
	fpt.SelectItem(int(pv.FillType))

	fs := pv.FillStack()

	switch pv.FillType {
	case PaintSolid:
		if fs.StackTop != 1 {
			fs.SetFullReRender()
		}
		fs.StackTop = 1
		fc := fs.ChildByName("fill-clr", 1).(*giv.ColorView)
		fc.SetColor(pc.FillStyle.Color.Color)
	case PaintLinear, PaintRadial:
		if fs.StackTop != 2 {
			fs.SetFullReRender()
		}
		fs.StackTop = 2
		fg := fs.ChildByName("fill-grad", 1).(*giv.TableView)
		if fg.Slice != grl {
			pv.SetFullReRender()
		}
		fg.SetSlice(grl)
		pv.SelectFillGrad()
	default:
		if fs.StackTop != 0 {
			fs.SetFullReRender()
		}
		fs.StackTop = 0
	}
}
*/

/*
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
*/

/*
// DecodeType decodes the paint type from paint and props
// also returns the name of the gradient if using one.
func (pv *PaintView) DecodeType(kn ki.Ki, cs *style.ColorSpec, prop string) (PaintTypes, string) {
	pstr := ""
	var gii gi.Node2D
	if kn != nil {
		pstr = laser.ToString(kn.Prop(prop))
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
		if gii != nil {
			grnm = pv.GradStopsName(gii, pstr)
		}
	case strings.HasPrefix(pstr, "url(#radial") || (cs.Gradient != nil && cs.Gradient.IsRadial):
		ptyp = PaintRadial
		if gii != nil {
			grnm = pv.GradStopsName(gii, pstr)
		}
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
*/

func (pv *PaintView) SelectStrokeGrad() {
	es := &pv.VectorView.EditState
	grl := &es.Gradients
	ss := pv.StrokeStack()
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
	es := &pv.VectorView.EditState
	grl := &es.Gradients
	fs := pv.FillStack()
	fg := fs.ChildByName("fill-grad", 1).(*giv.TableView)
	fg.UnselectAllIdxs()
	for i, g := range *grl {
		if g.Name == pv.FillStops {
			fg.SelectIdx(i)
			break
		}
	}
}

func (pv *PaintView) Config(vv *VectorView) {
	if pv.HasChildren() {
		return
	}
	updt := pv.UpdateStart()
	pv.StrokeType = PaintSolid
	pv.FillType = PaintSolid

	DashIconsInit()
	MarkerIconsInit()

	pv.VectorView = vv
	pv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	sty := &Prefs.ShapeStyle

	spl := gi.NewLayout(pv, "stroke-lab")
	gi.NewLabel(spl).SetText("<b>Stroke Paint:  </b>")
	spt := gi.NewChooser(spl) // .SetStrings(PaintTypeNames)
	spt.SelectItem(int(pv.StrokeType))
	// spt.Mutex = true

	wr := gi.NewLayout(pv, "stroke-width")
	gi.NewLabel(wr).SetText("Width:  ").Style(func(s *styles.Style) {
		s.Align.Items = styles.Center
	})

	gi.NewSpinner(wr, "width").SetMin(0).SetStep(0.05).
		SetValue(sty.StrokeStyle.Width.Val).OnChange(func(e events.Event) {
		if pv.IsStrokeOn() {
			pv.VectorView.SetStrokeWidth(pv.StrokeWidthProp(), false)
		}
	})

	// uncb.SetCurrentIndex(int(Prefs.Size.Units))
	gi.NewChooser(wr, "width-units").SetEnum(units.UnitsN).OnChange(func(e events.Event) {
		if pv.IsStrokeOn() {
			pv.VectorView.SetStrokeWidth(pv.StrokeWidthProp(), false)
		}
	})

	gi.NewSpace(wr, "sp1").Style(func(s *styles.Style) {
		s.Min.X.Ch(5)
	})

	dshcb := gi.NewChooser(wr, "dashes")
	// dshcb.ItemsFromIconList(AllDashIconNames, true, 0)
	// dshcb.SetProp("width", units.NewCh(15))
	dshcb.OnChange(func(e events.Event) {
		if pv.IsStrokeOn() {
			pv.VectorView.SetDashProps(pv.StrokeDashProp())
		}
	})

	mkr := gi.NewLayout(pv, "stroke-markers")

	mscb := gi.NewChooser(mkr, "marker-start")
	// mscb.SetProp("width", units.NewCh(20))
	// mscb.ItemsFromIconList(AllMarkerIconNames, true, 0)
	mscb.OnChange(func(e events.Event) {
		if pv.IsStrokeOn() {
			pv.VectorView.SetMarkerProps(pv.MarkerProps())
		}
	})
	mscc := gi.NewChooser(mkr, "marker-start-color").SetEnum(MarkerColorsN)
	// mscc.SetProp("width", units.NewCh(5))
	mscc.OnChange(func(e events.Event) {
		if pv.IsStrokeOn() {
			pv.VectorView.SetMarkerProps(pv.MarkerProps())
		}
	})

	gi.NewSeparator(mkr)

	mmcb := gi.NewChooser(mkr, "marker-mid")
	// mmcb.SetProp("width", units.NewCh(20))
	// mmcb.ItemsFromIconList(AllMarkerIconNames, true, 0)
	mmcb.OnChange(func(e events.Event) {
		if pv.IsStrokeOn() {
			pv.VectorView.SetMarkerProps(pv.MarkerProps())
		}
	})
	mmcc := gi.NewChooser(mkr, "marker-mid-color").SetEnum(MarkerColorsN)
	// mmcc.SetProp("width", units.NewCh(5))
	mmcc.OnChange(func(e events.Event) {
		if pv.IsStrokeOn() {
			pv.VectorView.SetMarkerProps(pv.MarkerProps())
		}
	})

	gi.NewSeparator(mkr)

	mecb := gi.NewChooser(mkr, "marker-end")
	// mecb.SetProp("width", units.NewCh(20))
	// mecb.ItemsFromIconList(AllMarkerIconNames, true, 0)
	mecb.OnChange(func(e events.Event) {
		if pv.IsStrokeOn() {
			pv.VectorView.SetMarkerProps(pv.MarkerProps())
		}
	})
	mecc := gi.NewChooser(mkr, "marker-end-color").SetEnum(MarkerColorsN)
	// mecc.SetProp("width", units.NewCh(5))
	mecc.OnChange(func(e events.Event) {
		if pv.IsStrokeOn() {
			pv.VectorView.SetMarkerProps(pv.MarkerProps())
		}
	})

	////////////////////////////////
	// stroke stack

	ss := gi.NewFrame(pv, "stroke-stack")
	ss.Styles.Display = styles.Stacked
	ss.StackTop = 1
	// ss.StackTopOnly = true

	gi.NewFrame(ss, "stroke-blank") // nothing

	// spt.ButtonSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	pvv := recv.Embed(KiT_PaintView).(*PaintView)
	// 	sss := pvv.StrokeStack()
	// 	prev := pv.StrokeType
	// 	pvv.StrokeType = PaintTypes(sig)
	// 	updt := pvv.UpdateStart()
	// 	pvv.SetFullReRender()
	// 	sp := pvv.StrokeProp()
	// 	switch pvv.StrokeType {
	// 	case PaintOff, PaintInherit:
	// 		sss.StackTop = 0
	// 	case PaintSolid:
	// 		sss.StackTop = 1
	// 	case PaintLinear, PaintRadial:
	// 		if pvv.StrokeStops == "" {
	// 			pvv.StrokeStops = pvv.VectorView.DefaultGradient()
	// 		}
	// 		sp = pvv.StrokeStops
	// 		sss.StackTop = 2
	// 		pvv.SelectStrokeGrad()
	// 	}
	// 	pvv.UpdateEnd(updt)
	// 	pvv.VectorView.SetStroke(prev, pvv.StrokeType, sp)
	// })

	sc := giv.NewColorView(ss, "stroke-clr")
	// sc.SetProp("vertical-align", styles.AlignTop)
	sc.Config()
	// sc.SetColor(sty.StrokeStyle.Color)
	// sc.ViewSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	if pv.StrokeType == PaintSolid {
	// 		pv.VectorView.SetStrokeColor(pv.StrokeProp(), false) // not manip
	// 	}
	// })
	// sc.ManipSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	if pv.StrokeType == PaintSolid {
	// 		pv.VectorView.SetStrokeColor(pv.StrokeProp(), true) // manip
	// 	}
	// })

	sg := giv.NewTableView(ss, "stroke-grad")
	// sg.SetProp("index", true)
	// sg.SetProp("toolbar", true)
	// sg.SelectedIdx = -1
	sg.SetSlice(&pv.VectorView.EditState.Gradients)
	// sg.WidgetSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	if sig == int64(gi.WidgetSelected) {
	// 		svv, _ := send.(*giv.TableView)
	// 		if svv.SelectedIdx >= 0 {
	// 			pv.StrokeStops = pv.VectorView.EditState.Gradients[svv.SelectedIdx].Name
	// 			pv.VectorView.SetStroke(pv.StrokeType, pv.StrokeType, pv.StrokeStops) // handles full updating
	// 		}
	// 	}
	// })

	gi.NewSeparator(pv)

	fpl := gi.NewLayout(pv, "fill-lab")
	gi.NewLabel(fpl).SetText("<b>Fill Paint:  </b>")

	fpt := gi.NewChooser(fpl, "fill-type").SetEnum(PaintTypesN)
	fpt.SelectItem(int(pv.FillType))
	// fpt.Mutex = true

	fs := gi.NewFrame(pv, "fill-stack")
	fs.Styles.Display = styles.Stacked
	fs.StackTop = 1
	// fs.StackTopOnly = true

	gi.NewFrame(fs, "fill-blank")

	// fpt.ButtonSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	pvv := recv.Embed(KiT_PaintView).(*PaintView)
	// 	fss := pvv.FillStack()
	// 	prev := pvv.FillType
	// 	pvv.FillType = PaintTypes(sig)
	// 	updt := fss.UpdateStart()
	// 	fss.SetFullReRender()
	// 	fp := pvv.FillProp()
	// 	switch pvv.FillType {
	// 	case PaintOff, PaintInherit:
	// 		fss.StackTop = 0
	// 	case PaintSolid:
	// 		fss.StackTop = 1
	// 	case PaintLinear, PaintRadial:
	// 		if pvv.FillStops == "" {
	// 			pvv.FillStops = pvv.VectorView.DefaultGradient()
	// 		}
	// 		fp = pvv.FillStops
	// 		fss.StackTop = 2
	// 		pvv.SelectFillGrad()
	// 	}
	// 	pvv.UpdateEnd(updt)
	// 	pvv.VectorView.SetFill(prev, pvv.FillType, fp)
	// })

	giv.NewColorView(fs, "fill-clr")
	// fc.SetProp("vertical-align", styles.AlignTop)
	// fc.Config()
	// fc.SetColor(sty.FillStyle.Color.Color)
	// fc.ViewSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	if pv.FillType == PaintSolid {
	// 		pv.VectorView.SetFillColor(pv.FillProp(), false)
	// 	}
	// })
	// fc.ManipSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	if pv.FillType == PaintSolid {
	// 		pv.VectorView.SetFillColor(pv.FillProp(), true) // manip
	// 	}
	// })

	fg := giv.NewTableView(fs, "fill-grad")
	// fg.SetProp("index", true)
	// fg.SetProp("toolbar", true)
	// fg.SelectedIdx = -1
	fg.SetSlice(&pv.VectorView.EditState.Gradients)
	// fg.WidgetSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	if sig == int64(gi.WidgetSelected) {
	// 		svv, _ := send.(*giv.TableView)
	// 		if svv.SelectedIdx >= 0 {
	// 			pv.FillStops = pv.VectorView.EditState.Gradients[svv.SelectedIdx].Name
	// 			pv.VectorView.SetFill(pv.FillType, pv.FillType, pv.FillStops) // this handles updating gradients etc to use stops
	// 		}
	// 	}
	// })
	// fg.SliceViewSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	// fmt.Printf("svs: %v   %v\n", sig, data)
	// 	// svv, _ := send.(*giv.TableView)
	// 	if sig == int64(giv.SliceViewDeleted) { // not clear what we can do here
	// 	} else {
	// 		pv.VectorView.UpdateGradients()
	// 	}
	// })
	// fg.ViewSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	// fmt.Printf("vs: %v   %v\n", sig, data)
	// 	// svv, _ := send.(*giv.TableView)
	// 	pv.VectorView.UpdateGradients()
	// })

	gi.NewStretch(pv)

	pv.UpdateEnd(updt)
}

// StrokeStack returns the stroke stack frame
func (pv *PaintView) StrokeStack() *gi.Frame {
	return pv.ChildByName("stroke-stack", 1).(*gi.Frame)
}

// FillStack returns the fill stack frame
func (pv *PaintView) FillStack() *gi.Frame {
	return pv.ChildByName("fill-stack", 4).(*gi.Frame)
}

// StrokeProp returns the stroke property string according to current settings
func (pv *PaintView) StrokeProp() string {
	// ss := pv.StrokeStack()
	switch pv.StrokeType {
	case PaintOff:
		return "none"
	case PaintSolid:
		// sc := ss.ChildByName("stroke-clr", 1).(*giv.ColorView)
		// return sc.Color.HexString()
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
	// mkr := pv.ChildByName("stroke-markers", 3)
	//
	// mscb := mkr.ChildByName("marker-start", 0).(*gi.Chooser)
	// mscc := mkr.ChildByName("marker-start-color", 1).(*gi.Chooser)
	// start = IconToMarkerName(mscb.CurVal)
	// sc = MarkerColors(mscc.CurrentIndex)
	//
	// mmcb := mkr.ChildByName("marker-mid", 2).(*gi.Chooser)
	// mmcc := mkr.ChildByName("marker-mid-color", 3).(*gi.Chooser)
	// mid = IconToMarkerName(mmcb.CurVal)
	// mc = MarkerColors(mmcc.CurrentIndex)
	//
	// mecb := mkr.ChildByName("marker-end", 4).(*gi.Chooser)
	// mecc := mkr.ChildByName("marker-end-color", 5).(*gi.Chooser)
	// end = IconToMarkerName(mecb.CurVal)
	// ec = MarkerColors(mecc.CurrentIndex)

	return
}

// IsStrokeOn returns true if stroke is active
func (pv *PaintView) IsStrokeOn() bool {
	return pv.StrokeType >= PaintSolid && pv.StrokeType < PaintInherit
}

// StrokeWidthProp returns stroke-width property
func (pv *PaintView) StrokeWidthProp() string {
	wr := pv.ChildByName("stroke-width", 2)
	wsb := wr.ChildByName("width", 1).(*gi.Spinner)
	uncb := wr.ChildByName("width-units", 2).(*gi.Chooser)
	unnm := "px"
	if uncb.CurrentIndex > 0 {
		unvl := units.Units(uncb.CurrentIndex)
		unnm = unvl.String()
	}
	return fmt.Sprintf("%g%s", wsb.Value, unnm)
}

// StrokeDashProp returns stroke-dasharray property as an array (nil = none)
// these values need to be multiplied by line widths for each item.
func (pv *PaintView) StrokeDashProp() []float64 {
	wr := pv.ChildByName("stroke-width", 2)
	dshcb := wr.ChildByName("dashes", 3).(*gi.Chooser)
	if dshcb.CurrentIndex == 0 {
		return nil
	}
	dnm := laser.ToString(dshcb.CurrentItem.Value)
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
	// fs := pv.FillStack()
	switch pv.FillType {
	case PaintOff:
		return "none"
	case PaintSolid:
		// sc := fs.ChildByName("fill-clr", 1).(*giv.ColorView)
		// return sc.Color.HexString()
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
func (pv *PaintView) SetProps(sii svg.Node) {
	pv.VectorView.SetColorNode(sii, "stroke", pv.StrokeType, pv.StrokeType, pv.StrokeProp())
	if pv.IsStrokeOn() {
		sii.SetProp("stroke-width", pv.StrokeWidthProp())
		start, mid, end, sc, mc, ec := pv.MarkerProps()
		pv.VectorView.SetMarkerNode(sii, start, mid, end, sc, mc, ec)
	}
	pv.VectorView.SetColorNode(sii, "fill", pv.FillType, pv.FillType, pv.FillProp())
}

type PaintTypes int32 //enums:enum -trim-prefix Paint

const (
	PaintOff PaintTypes = iota
	PaintSolid
	PaintLinear
	PaintRadial
	PaintInherit
)
