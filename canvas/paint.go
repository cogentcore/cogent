// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"fmt"
	"image"
	"strings"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
)

// PaintSetter provides setting of basic Stroke and Fill painting parameters
// for selected items
type PaintSetter struct {
	core.Frame

	// Active styles
	PaintStyle styles.Paint

	// paint type for stroke
	StrokeType PaintTypes

	// name of gradient with stops
	StrokeStops string

	// paint type for fill
	FillType PaintTypes

	// name of gradient with stops
	FillStops string

	// the parent [Canvas]
	Canvas *Canvas `copier:"-" json:"-" xml:"-" display:"-"`

	curStrokeType PaintTypes
	curFillType   PaintTypes

	strokeStack *core.Frame
	fillStack   *core.Frame
}

func (pv *PaintSetter) Init() {
	pv.Frame.Init()
	pv.StrokeType = PaintSolid
	pv.FillType = PaintSolid
	pv.curStrokeType = pv.StrokeType
	pv.curFillType = pv.FillType
	pv.PaintStyle = Settings.ShapeStyle

	DashIconsInit()
	MarkerIconsInit()

	pv.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	tree.AddChildAt(pv, "stroke-lab", func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
		})
		tree.AddChild(w, func(w *core.Text) {
			w.SetText("<b>Stroke paint:</b>")
		})
		tree.AddChild(w, func(w *core.Switches) {
			core.Bind(&pv.StrokeType, w)
			w.OnChange(func(e events.Event) {
				if pv.StrokeType == PaintLinear || pv.StrokeType == PaintRadial {
					if pv.StrokeStops == "" {
						pv.StrokeStops = pv.Canvas.DefaultGradient()
					}
					pv.SelectStrokeGrad()
				}
				pv.Canvas.SetStroke(pv.curStrokeType, pv.StrokeType, pv.StrokeStops)
				pv.curStrokeType = pv.StrokeType
				pv.Update()
			})
		})
	})

	tree.AddChildAt(pv, "stroke-width", func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
		})
		tree.AddChild(w, func(w *core.Text) {
			w.SetText("Width:  ").Styler(func(s *styles.Style) {
				s.Align.Items = styles.Center
			})
		})
		tree.AddChild(w, func(w *core.Spinner) {
			core.Bind(&pv.PaintStyle.Stroke.Width.Value, w)
			w.SetMin(0).SetStep(0.05)
			w.OnChange(func(e events.Event) {
				if pv.IsStrokeOn() {
					pv.Canvas.SetStrokeWidth(pv.StrokeWidthProp())
				}
			})
		})

		// uncb.SetCurrentIndex(int(Settings.Size.Units))
		tree.AddChild(w, func(w *core.Chooser) {
			core.Bind(&pv.PaintStyle.Stroke.Width.Unit, w)
			w.OnChange(func(e events.Event) {
				if pv.IsStrokeOn() {
					pv.Canvas.SetStrokeWidth(pv.StrokeWidthProp())
				}
			})
		})

		// core.NewSpace(wr, "sp1").Styler(func(s *styles.Style) {
		// 	s.Min.X.Ch(5)
		// })

		tree.AddChild(w, func(w *core.Chooser) {
			// dshcb.ItemsFromIconList(AllDashIcons, true, 0)
			// dshcb.SetProp("width", units.NewCh(15))
			w.OnChange(func(e events.Event) {
				if pv.IsStrokeOn() {
					pv.Canvas.SetDashProperties(pv.StrokeDashProp())
				}
			})
		})
	})

	tree.AddChildAt(pv, "stroke-markers", func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
		})
		tree.AddChild(w, func(w *core.Chooser) { // start
			// mscb.SetProp("width", units.NewCh(20))
			// mscb.ItemsFromIconList(AllMarkerIcons, true, 0)
			w.OnChange(func(e events.Event) {
				if pv.IsStrokeOn() {
					pv.Canvas.SetMarkerProperties(pv.MarkerProperties())
				}
			})
		})
		tree.AddChild(w, func(w *core.Chooser) { // start-color
			w.SetEnum(MarkerColorsN)
			// mscc.SetProp("width", units.NewCh(5))
			w.OnChange(func(e events.Event) {
				if pv.IsStrokeOn() {
					pv.Canvas.SetMarkerProperties(pv.MarkerProperties())
				}
			})
		})

		tree.AddChild(w, func(w *core.Separator) {})

		tree.AddChild(w, func(w *core.Chooser) { // mid
			// mscb.SetProp("width", units.NewCh(20))
			// mscb.ItemsFromIconList(AllMarkerIcons, true, 0)
			w.OnChange(func(e events.Event) {
				if pv.IsStrokeOn() {
					pv.Canvas.SetMarkerProperties(pv.MarkerProperties())
				}
			})
		})
		tree.AddChild(w, func(w *core.Chooser) { // mid-color
			w.SetEnum(MarkerColorsN)
			// mmcc.SetProp("width", units.NewCh(5))
			w.OnChange(func(e events.Event) {
				if pv.IsStrokeOn() {
					pv.Canvas.SetMarkerProperties(pv.MarkerProperties())
				}
			})
		})

		tree.AddChild(w, func(w *core.Separator) {})

		tree.AddChild(w, func(w *core.Chooser) { // end
			// mscb.SetProp("width", units.NewCh(20))
			// mscb.ItemsFromIconList(AllMarkerIcons, true, 0)
			w.OnChange(func(e events.Event) {
				if pv.IsStrokeOn() {
					pv.Canvas.SetMarkerProperties(pv.MarkerProperties())
				}
			})
		})
		tree.AddChild(w, func(w *core.Chooser) { // end-color
			w.SetEnum(MarkerColorsN)
			// mscc.SetProp("width", units.NewCh(5))
			w.OnChange(func(e events.Event) {
				if pv.IsStrokeOn() {
					pv.Canvas.SetMarkerProperties(pv.MarkerProperties())
				}
			})
		})
	})

	//////// stroke stack

	tree.AddChildAt(pv, "stroke-stack", func(w *core.Frame) {
		pv.strokeStack = w
		w.StackTop = 1
		// ss.StackTopOnly = true
		w.Styler(func(s *styles.Style) {
			s.Display = styles.Stacked
		})
		// ss.StackTopOnly = true
		w.Updater(func() {
			w.StackTop = int(pv.StrokeType)
		})
		tree.AddChild(w, func(w *core.Frame) {}) // "stroke-blank"

		tree.AddChild(w, func(w *core.ColorPicker) {
			core.Bind(&pv.PaintStyle.Stroke.Color, w)
			w.OnChange(func(e events.Event) {
				if pv.StrokeType == PaintSolid {
					pv.Canvas.SetStrokeColor(pv.StrokeProp(), false) // not manip
				}
			})
		})

		tree.AddChild(w, func(w *core.Table) { // "stroke-grad"
			// sg.SetProp("index", true)
			// sg.SetProp("toolbar", true)
			// sg.SelectedIndex = -1
			w.SetSlice(&pv.Canvas.EditState.Gradients)
			// todo: bindselect
			// sg.WidgetSig.Connect(pv.This, func(recv, send tree.Node, sig int64, data any) {
			// 	if sig == int64(core.WidgetSelected) {
			// 		svv, _ := send.(*core.Table)
			// 		if svv.SelectedIndex >= 0 {
			// 			pv.StrokeStops = pv.Vector.EditState.Gradients[svv.SelectedIndex].Name
			// 			pv.Vector.SetStroke(pv.StrokeType, pv.StrokeType, pv.StrokeStops) // handles full updating
			// 		}
			// 	}
			// })
		})
	})

	tree.AddChild(pv, func(w *core.Separator) {})

	tree.AddChildAt(pv, "fill-lab", func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
		})
		tree.AddChild(w, func(w *core.Text) {
			w.SetText("<b>Fill paint:</b>")
		})
		tree.AddChild(w, func(w *core.Switches) {
			core.Bind(&pv.FillType, w)
			w.OnChange(func(e events.Event) {
				if pv.FillType == PaintLinear || pv.FillType == PaintRadial {
					if pv.FillStops == "" {
						pv.FillStops = pv.Canvas.DefaultGradient()
					}
					pv.SelectFillGrad()
				}
				pv.Canvas.SetFill(pv.curFillType, pv.FillType, pv.FillStops)
				pv.curFillType = pv.FillType
				pv.Update()
			})
		})
	})

	tree.AddChildAt(pv, "fill-stack", func(w *core.Frame) {
		pv.fillStack = w
		w.StackTop = 1
		// fs.StackTopOnly = true
		w.Styler(func(s *styles.Style) {
			s.Display = styles.Stacked
		})
		w.Updater(func() {
			w.StackTop = int(pv.FillType)
		})

		tree.AddChild(w, func(w *core.Frame) {}) // "fill-blank"

		tree.AddChild(w, func(w *core.ColorPicker) {
			core.Bind(&pv.PaintStyle.Fill.Color, w)
			w.OnChange(func(e events.Event) {
				if pv.FillType == PaintSolid {
					pv.Canvas.SetFillColor(pv.FillProp(), false) // not manip
				}
			})
		})

		tree.AddChild(w, func(w *core.Table) { // "fill-grad"
			// sg.SetProp("index", true)
			// sg.SetProp("toolbar", true)
			// sg.SelectedIndex = -1
			w.SetSlice(&pv.Canvas.EditState.Gradients)
			// fg.WidgetSig.Connect(pv.This, func(recv, send tree.Node, sig int64, data any) {
			// 	if sig == int64(core.WidgetSelected) {
			// 		svv, _ := send.(*core.Table)
			// 		if svv.SelectedIndex >= 0 {
			// 			pv.FillStops = pv.Vector.EditState.Gradients[svv.SelectedIndex].Name
			// 			pv.Vector.SetFill(pv.FillType, pv.FillType, pv.FillStops) // this handles updating gradients etc to use stops
			// 		}
			// 	}
			// })
			// fg.ListSig.Connect(pv.This, func(recv, send tree.Node, sig int64, data any) {
			// 	// fmt.Printf("svs: %v   %v\n", sig, data)
			// 	// svv, _ := send.(*core.Table)
			// 	if sig == int64(core.ListDeleted) { // not clear what we can do here
			// 	} else {
			// 		pv.Vector.UpdateGradients()
			// 	}
			// })
			// fg.ViewSig.Connect(pv.This, func(recv, send tree.Node, sig int64, data any) {
			// 	// fmt.Printf("vs: %v   %v\n", sig, data)
			// 	// svv, _ := send.(*core.Table)
			// 	pv.Vector.UpdateGradients()
			// })
		})

		tree.AddChild(w, func(w *core.Stretch) {})
	})
}

////////  Actions

// PaintSet manages all the updating etc associated with setting paint
// parameters.
func (cv *Canvas) PaintSet(act Actions, data string, manip bool, fun func(nd svg.Node)) {
	es := &cv.EditState
	sv := cv.SVG
	// update := false
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
			sv.UndoSave(act.String(), data)
		}
	}
	for itm := range es.Selected {
		cv.PaintSetFun(itm, fun)
	}
	if !manip {
		if !actStart {
			es.ActDone()
			cv.ChangeMade()
		}
	} else {
		sv.NeedsRender()
	}
}

func (cv *Canvas) PaintSetFun(nd svg.Node, fun func(itm svg.Node)) {
	if gp, isgp := nd.(*svg.Group); isgp {
		for _, kid := range gp.Children {
			cv.PaintSetFun(kid.(svg.Node), fun)
		}
		return
	}
	fun(nd)
}

// SetPaintPropNode sets paint property on given node,
// using given setter function.
func SetPaintPropNode(nd svg.Node, fun func(g svg.Node)) {
	if gp, isgp := nd.(*svg.Group); isgp {
		for _, kid := range gp.Children {
			SetPaintPropNode(kid.(svg.Node), fun)
		}
		return
	}
	fun(nd)
}

// SetPaintProp sets paint property selected nodes,
// using given setter function.
func (cv *Canvas) SetPaintProp(actName, val string, fun func(g svg.Node)) {
	es := &cv.EditState
	cv.SVG.UndoSave(actName, val)
	for itm := range es.Selected {
		SetPaintPropNode(itm, fun)
	}
	cv.ChangeMade()
}

// SetColorNode sets the color properties of Node
// based on previous and current PaintType
func (cv *Canvas) SetColorNode(nd svg.Node, prop string, prev, pt PaintTypes, sp string) {
	if gp, isgp := nd.(*svg.Group); isgp {
		for _, kid := range gp.Children {
			cv.SetColorNode(kid.(svg.Node), prop, prev, pt, sp)
		}
		return
	}
	switch pt {
	// case PaintLinear:
	// 	svg.UpdateNodeGradientProp(nd, prop, false, sp)
	// case PaintRadial:
	// 	svg.UpdateNodeGradientProp(nd, prop, true, sp)
	default:
		if prev == PaintLinear || prev == PaintRadial {
			pstr := reflectx.ToString(nd.AsTree().Properties[prop])
			_ = pstr
			// svg.DeleteNodeGradient(nd, pstr)
		}
		nd.AsNodeBase().SetColorProperties(prop, sp)
	}
	cv.UpdateMarkerColors(nd)
}

// SetStroke sets the stroke properties of selected items
// based on previous and current PaintType
func (cv *Canvas) SetStroke(prev, pt PaintTypes, sp string) {
	cv.SetPaintProp("SetStroke", sp, func(nd svg.Node) {
		cv.SetColorNode(nd, "stroke", prev, pt, sp)
	})
}

// SetStrokeColor sets the stroke color for selected items.
// manip means currently being manipulated -- don't save undo.
func (cv *Canvas) SetStrokeColor(sp string, manip bool) {
	cv.PaintSet(SetStrokeColor, sp, manip,
		func(itm svg.Node) {
			p := itm.AsTree().Properties["stroke"]
			if p != nil {
				itm.AsNodeBase().SetColorProperties("stroke", sp)
				cv.UpdateMarkerColors(itm)
			}
		})
}

func (cv *Canvas) SetStrokeWidth(wp string) {
	cv.SetPaintProp("SetStrokeWidth", wp, func(nd svg.Node) {
		g := nd.AsNodeBase()
		if g.Paint.Stroke.Color != nil {
			g.SetProperty("stroke-width", wp)
		}
	})
}

// SetMarkerProperties sets the marker properties
func (cv *Canvas) SetMarkerProperties(start, mid, end string, sc, mc, ec MarkerColors) {
	sv := cv.SVG.SVG
	cv.SetPaintProp("SetMarkerProperties", start+" "+mid+" "+end, func(nd svg.Node) {
		MarkerSetProp(sv, nd, "marker-start", start, sc)
		MarkerSetProp(sv, nd, "marker-mid", mid, mc)
		MarkerSetProp(sv, nd, "marker-end", end, ec)
	})
}

// UpdateMarkerColors updates the marker colors, when setting fill or stroke
func (cv *Canvas) UpdateMarkerColors(nd svg.Node) {
	if nd == nil {
		return
	}
	sv := cv.SVG
	MarkerUpdateColorProp(sv.SVG, nd, "marker-start")
	MarkerUpdateColorProp(sv.SVG, nd, "marker-mid")
	MarkerUpdateColorProp(sv.SVG, nd, "marker-end")
}

// SetDashNode sets the stroke-dasharray property of Node.
// multiplies dash values by the line width in dots.
func (cv *Canvas) SetDashNode(nd svg.Node, dary []float64) {
	if gp, isgp := nd.(*svg.Group); isgp {
		for _, kid := range gp.Children {
			cv.SetDashNode(kid.(svg.Node), dary)
		}
		return
	}
	if len(dary) == 0 {
		delete(nd.AsTree().Properties, "stroke-dasharray")
		return
	}
	g := nd.AsNodeBase()
	mary := DashMulWidth(float64(g.Paint.Stroke.Width.Dots), dary)
	ds := DashString(mary)
	nd.AsTree().Properties["stroke-dasharray"] = ds
}

// SetDashProperties sets the dash properties
func (cv *Canvas) SetDashProperties(dary []float64) {
	es := &cv.EditState
	sv := cv.SVG
	sv.UndoSave("SetDashProperties", "")
	// update := sv.UpdateStart()
	// sv.SetFullReRender()
	for itm := range es.Selected {
		cv.SetDashNode(itm, dary)
	}
	// sv.UpdateEnd(update)
	cv.ChangeMade()
}

// SetFill sets the fill properties of selected items
// based on previous and current PaintType
func (cv *Canvas) SetFill(prev, pt PaintTypes, fp string) {
	es := &cv.EditState
	sv := cv.SVG
	sv.UndoSave("SetFill", fp)
	// update := sv.UpdateStart()
	// sv.SetFullReRender()
	for itm := range es.Selected {
		cv.SetColorNode(itm, "fill", prev, pt, fp)
	}
	// sv.UpdateEnd(update)
	cv.ChangeMade()
}

// SetFillColor sets the fill color for selected items
// manip means currently being manipulated -- don't save undo.
func (cv *Canvas) SetFillColor(fp string, manip bool) {
	cv.PaintSet(SetFillColor, fp, manip,
		func(itm svg.Node) {
			p := itm.AsTree().Properties["fill"]
			if p != nil {
				itm.AsNodeBase().SetColorProperties("fill", fp)
				cv.UpdateMarkerColors(itm)
			}
		})
}

// DefaultGradient returns the default gradient to use for setting stops
func (cv *Canvas) DefaultGradient() string {
	es := &cv.EditState
	sv := cv.SVG
	if len(cv.EditState.Gradients) == 0 {
		es.ConfigDefaultGradient()
		sv.UpdateGradients(es.Gradients)
	}
	return es.Gradients[0].Name
}

// UpdateGradients updates gradients from EditState
func (cv *Canvas) UpdateGradients() {
	es := &cv.EditState
	sv := cv.SVG
	// update := sv.UpdateStart()
	sv.UpdateGradients(es.Gradients)
	// sv.UpdateEnd(update)
}

////////  PaintSetter

// Update updates the current settings based on the values in the given Paint and
// properties from node (node can be nil)
func (pv *PaintSetter) UpdateFromNode(ps *styles.Paint, nd svg.Node) {
	pv.StrokeType, pv.StrokeStops = pv.GetPaintType(nd, ps.Stroke.Color, "stroke")
	pv.FillType, pv.FillStops = pv.GetPaintType(nd, ps.Fill.Color, "fill")

	// es := &pv.Canvas.EditState
	// grl := &es.Gradients

	switch pv.StrokeType {
	case PaintSolid:
		pv.strokeStack.StackTop = 1
		pv.PaintStyle.Stroke.Color = ps.Stroke.Color
	case PaintLinear, PaintRadial:
		pv.strokeStack.StackTop = 2
		// sg := ss.ChildByName("stroke-grad", 1).(*core.Table)
		// sg.SetSlice(grl)
		// pv.SelectStrokeGrad()
	default:
		pv.strokeStack.StackTop = 0
	}

	pv.PaintStyle.Stroke.Width = ps.Stroke.Width
	pv.PaintStyle.Stroke.Dashes = ps.Stroke.Dashes

	// ms, _, mc := MarkerFromNodeProp(nd, "marker-start")
	// mscb := mkr.ChildByName("marker-start", 0).(*core.Chooser)
	// mscc := mkr.ChildByName("marker-start-color", 1).(*core.Chooser)
	// if ms != "" {
	// 	mscb.SetCurVal(MarkerNameToIcon(ms))
	// 	mscc.SetCurrentIndex(int(mc))
	// } else {
	// 	mscb.SetCurrentIndex(0)
	// 	mscc.SetCurrentIndex(0)
	// }
	// ms, _, mc = MarkerFromNodeProp(nd, "marker-mid")
	// mmcb := mkr.ChildByName("marker-mid", 2).(*core.Chooser)
	// mmcc := mkr.ChildByName("marker-mid-color", 3).(*core.Chooser)
	// if ms != "" {
	// 	mmcb.SetCurVal(MarkerNameToIcon(ms))
	// 	mmcc.SetCurrentIndex(int(mc))
	// } else {
	// 	mmcb.SetCurrentIndex(0)
	// 	mmcc.SetCurrentIndex(0)
	// }
	// ms, _, mc = MarkerFromNodeProp(nd, "marker-end")
	// mecb := mkr.ChildByName("marker-end", 4).(*core.Chooser)
	// mecc := mkr.ChildByName("marker-end-color", 5).(*core.Chooser)
	// if ms != "" {
	// 	mecb.SetCurVal(MarkerNameToIcon(ms))
	// 	mecc.SetCurrentIndex(int(mc))
	// } else {
	// 	mecb.SetCurrentIndex(0)
	// 	mecc.SetCurrentIndex(0)
	// }

	switch pv.FillType {
	case PaintSolid:
		pv.fillStack.StackTop = 1
		pv.PaintStyle.Fill.Color = ps.Fill.Color
	case PaintLinear, PaintRadial:
		pv.fillStack.StackTop = 2
		// fg := fs.ChildByName("fill-grad", 1).(*core.Table)
		// if fg.Slice != grl {
		// 	pv.SetFullReRender()
		// }
		// fg.SetSlice(grl)
		// pv.SelectFillGrad()
	default:
		pv.fillStack.StackTop = 0
	}
	pv.Update()
}

// GradStopsName returns the stopsname for gradient from url
func (pv *PaintSetter) GradStopsName(nd svg.Node, url string) string {
	gr := pv.Canvas.SSVG().GradientByName(nd, url)
	if gr == nil {
		return ""
	}
	if gr.StopsName != "" {
		return gr.StopsName
	}
	return gr.Name
}

// GetPaintType decodes the paint type from paint and properties
// also returns the name of the gradient if using one.
func (pv *PaintSetter) GetPaintType(nd svg.Node, clr image.Image, prop string) (PaintTypes, string) {
	pstr := ""
	if nd != nil {
		pv := nd.AsNodeBase().Properties[prop]
		pstr = reflectx.ToString(pv)
	}
	ptyp := PaintSolid
	grnm := ""
	lg, islg := clr.(*gradient.Linear)
	rg, isrg := clr.(*gradient.Radial)
	switch {
	case pstr == "inherit":
		ptyp = PaintInherit
	case pstr == "none" || clr == nil:
		ptyp = PaintOff
	case strings.HasPrefix(pstr, "url(#linear") || (islg && lg != nil):
		ptyp = PaintLinear
		if nd != nil {
			grnm = pv.GradStopsName(nd, pstr)
		}
	case strings.HasPrefix(pstr, "url(#radial") || (isrg && rg != nil):
		ptyp = PaintRadial
		if nd != nil {
			grnm = pv.GradStopsName(nd, pstr)
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

func (pv *PaintSetter) SelectStrokeGrad() {
	// todo:
	// es := &pv.Vector.EditState
	// grl := &es.Gradients
	// ss := pv.StrokeStack()
	// sg := ss.ChildByName("stroke-grad", 1).(*core.Table)
	// sg.UnselectAllIndexes()
	// for i, g := range *grl {
	// 	if g.Name == pv.StrokeStops {
	// 		sg.SelectIndex(i)
	// 		break
	// 	}
	// }
}

func (pv *PaintSetter) SelectFillGrad() {
	// todo:
	// es := &pv.Vector.EditState
	// grl := &es.Gradients
	// fs := pv.FillStack()
	// fg := fs.ChildByName("fill-grad", 1).(*core.Table)
	// fg.UnselectAllIndexes()
	// for i, g := range *grl {
	// 	if g.Name == pv.FillStops {
	// 		fg.SelectIndex(i)
	// 		break
	// 	}
	// }
}

// StrokeProp returns the stroke property string according to current settings
func (pv *PaintSetter) StrokeProp() string {
	// ss := pv.StrokeStack()
	switch pv.StrokeType {
	case PaintOff:
		return "none"
	case PaintSolid:
		return colors.AsHex(colors.ToUniform(pv.PaintStyle.Stroke.Color))
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
func (pv *PaintSetter) MarkerProperties() (start, mid, end string, sc, mc, ec MarkerColors) {
	// mkr := pv.ChildByName("stroke-markers", 3)
	//
	// mscb := mkr.ChildByName("marker-start", 0).(*core.Chooser)
	// mscc := mkr.ChildByName("marker-start-color", 1).(*core.Chooser)
	// start = IconToMarkerName(mscb.CurVal)
	// sc = MarkerColors(mscc.CurrentIndex)
	//
	// mmcb := mkr.ChildByName("marker-mid", 2).(*core.Chooser)
	// mmcc := mkr.ChildByName("marker-mid-color", 3).(*core.Chooser)
	// mid = IconToMarkerName(mmcb.CurVal)
	// mc = MarkerColors(mmcc.CurrentIndex)
	//
	// mecb := mkr.ChildByName("marker-end", 4).(*core.Chooser)
	// mecc := mkr.ChildByName("marker-end-color", 5).(*core.Chooser)
	// end = IconToMarkerName(mecb.CurVal)
	// ec = MarkerColors(mecc.CurrentIndex)

	return
}

// IsStrokeOn returns true if stroke is active
func (pv *PaintSetter) IsStrokeOn() bool {
	return pv.StrokeType >= PaintSolid && pv.StrokeType < PaintInherit
}

// StrokeWidthProp returns stroke-width property
func (pv *PaintSetter) StrokeWidthProp() string {
	unnm := pv.PaintStyle.Stroke.Width.Unit.String()
	return fmt.Sprintf("%g%s", pv.PaintStyle.Stroke.Width.Value, unnm)
}

// StrokeDashProp returns stroke-dasharray property as an array (nil = none)
// these values need to be multiplied by line widths for each item.
func (pv *PaintSetter) StrokeDashProp() []float64 {
	// todo: need type for dashes
	// wr := pv.ChildByName("stroke-width", 2)
	// dshcb := wr.AsTree().ChildByName("dashes", 3).(*core.Chooser)
	// if dshcb.CurrentIndex == 0 {
	// 	return nil
	// }
	// dnm := reflectx.ToString(dshcb.CurrentItem.Value)
	// if dnm == "" {
	// 	return nil
	// }
	// dary, ok := AllDashesMap[dnm]
	// if !ok {
	// 	return nil
	// }
	// return dary
	return nil
}

// IsFillOn returns true if Fill is active
func (pv *PaintSetter) IsFillOn() bool {
	return pv.FillType >= PaintSolid && pv.FillType < PaintInherit
}

// FillProp returns the fill property string according to current settings
func (pv *PaintSetter) FillProp() string {
	switch pv.FillType {
	case PaintOff:
		return "none"
	case PaintSolid:
		return colors.AsHex(colors.ToUniform(pv.PaintStyle.Fill.Color))
	case PaintLinear:
		return pv.FillStops
	case PaintRadial:
		return pv.FillStops
	case PaintInherit:
		return "inherit"
	}
	return "none"
}

// SetProperties sets the properties for given node according to current settings
func (pv *PaintSetter) SetProperties(nd svg.Node) {
	cv := pv.Canvas
	cv.SetColorNode(nd, "stroke", pv.StrokeType, pv.StrokeType, pv.StrokeProp())
	if pv.IsStrokeOn() {
		nd.AsTree().Properties["stroke-width"] = pv.StrokeWidthProp()
		cv.SetMarkerProperties(pv.MarkerProperties())
	}
	cv.SetColorNode(nd, "fill", pv.FillType, pv.FillType, pv.FillProp())
}

type PaintTypes int32 //enums:enum -trim-prefix Paint

const (
	PaintOff PaintTypes = iota
	PaintSolid
	PaintLinear
	PaintRadial
	PaintInherit
)
