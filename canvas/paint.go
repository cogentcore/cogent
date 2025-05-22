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

	markerStart, markerMid, markerEnd                *core.Chooser
	markerStartColor, markerMidColor, markerEndColor *core.Chooser

	dashes *core.Chooser

	strokeGrads, fillGrads *core.Table
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
		s.Grow.Set(0, 1)
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
				pv.SetStrokeStack(pv.StrokeType)
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
				pv.PaintStyle.ToDots()
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
			pv.dashes = w
			w.Styler(func(s *styles.Style) {
				s.IconSize.X.Em(3)
			})
			// tree.AddChildInit(w, "icon", func(w *core.Icon) {
			// 	w.Styler(func(s *styles.Style) {
			// 		s.Min.X.Em(3)
			// 	})
			// })
			w.SetItems(AllDashItems...)
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
			pv.markerStart = w
			w.Styler(func(s *styles.Style) {
				s.IconSize.X.Em(3)
				s.IconSize.Y.Em(2)
			})
			// tree.AddChildInit(w, "icon", func(w *core.Icon) {
			// 	w.Styler(func(s *styles.Style) {
			// 		s.Min.Set(units.Em(2))
			// 	})
			// })
			w.SetItems(AllMarkerItems...)
			w.OnChange(func(e events.Event) {
				if pv.IsStrokeOn() {
					pv.Canvas.SetMarkerProperties(pv.MarkerProperties())
				}
			})
		})
		tree.AddChild(w, func(w *core.Chooser) { // start-color
			pv.markerStartColor = w
			w.SetEnum(MarkerColorsN)
			w.OnChange(func(e events.Event) {
				if pv.IsStrokeOn() {
					pv.Canvas.SetMarkerProperties(pv.MarkerProperties())
				}
			})
		})

		tree.AddChild(w, func(w *core.Separator) {})

		tree.AddChild(w, func(w *core.Chooser) { // mid
			pv.markerMid = w
			w.Styler(func(s *styles.Style) {
				s.IconSize.X.Em(3)
				s.IconSize.Y.Em(2)
			})
			// tree.AddChildInit(w, "icon", func(w *core.Icon) {
			// 	w.Styler(func(s *styles.Style) {
			// 		s.Min.Set(units.Em(2))
			// 	})
			// })
			w.SetItems(AllMarkerItems...)
			w.OnChange(func(e events.Event) {
				if pv.IsStrokeOn() {
					pv.Canvas.SetMarkerProperties(pv.MarkerProperties())
				}
			})
		})
		tree.AddChild(w, func(w *core.Chooser) { // mid-color
			pv.markerMidColor = w
			w.SetEnum(MarkerColorsN)
			w.OnChange(func(e events.Event) {
				if pv.IsStrokeOn() {
					pv.Canvas.SetMarkerProperties(pv.MarkerProperties())
				}
			})
		})

		tree.AddChild(w, func(w *core.Separator) {})

		tree.AddChild(w, func(w *core.Chooser) { // end
			pv.markerEnd = w
			w.Styler(func(s *styles.Style) {
				s.IconSize.X.Em(3)
				s.IconSize.Y.Em(2)
			})
			// tree.AddChildInit(w, "icon", func(w *core.Icon) {
			// 	w.Styler(func(s *styles.Style) {
			// 		s.Min.Set(units.Em(2))
			// 	})
			// })
			w.SetItems(AllMarkerItems...)
			w.OnChange(func(e events.Event) {
				if pv.IsStrokeOn() {
					pv.Canvas.SetMarkerProperties(pv.MarkerProperties())
				}
			})
		})
		tree.AddChild(w, func(w *core.Chooser) { // end-color
			pv.markerEndColor = w
			w.SetEnum(MarkerColorsN)
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
		w.Styler(func(s *styles.Style) {
			s.Display = styles.Stacked
			s.Grow.Set(0, 1)
		})
		tree.AddChild(w, func(w *core.Frame) {}) // "stroke-blank"

		tree.AddChild(w, func(w *core.ColorPicker) {
			core.Bind(&pv.PaintStyle.Stroke.Color, w)
			w.HandleValueOnInput()
			w.OnInput(func(e events.Event) {
				if pv.StrokeType == PaintSolid {
					pv.Canvas.SetStrokeColor(pv.StrokeProp(), false) // not final
				}
			})
			w.OnChange(func(e events.Event) {
				if pv.StrokeType == PaintSolid {
					pv.Canvas.SetStrokeColor(pv.StrokeProp(), true) // final
				}
			})
		})

		tree.AddChild(w, func(w *core.Table) {
			pv.strokeGrads = w
			w.SetSlice(&pv.Canvas.EditState.Gradients)
			w.OnSelect(func(e events.Event) {
				pv.StrokeStops = pv.Canvas.EditState.Gradients[w.SelectedIndex].Name
				pv.Canvas.SetStroke(pv.StrokeType, pv.StrokeType, pv.StrokeStops)
			})
			w.OnChange(func(e events.Event) {
				pv.Canvas.UpdateGradients()
				if w.SelectedIndex >= 0 {
					pv.StrokeStops = pv.Canvas.EditState.Gradients[w.SelectedIndex].Name
					pv.Canvas.SetStroke(pv.StrokeType, pv.StrokeType, pv.StrokeStops)
				}
				w.Update()
			})
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
				pv.SetFillStack(pv.FillType)
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
		w.Styler(func(s *styles.Style) {
			s.Display = styles.Stacked
			s.Grow.Set(0, 1)
		})

		tree.AddChild(w, func(w *core.Frame) {})

		tree.AddChild(w, func(w *core.ColorPicker) {
			core.Bind(&pv.PaintStyle.Fill.Color, w)
			w.HandleValueOnInput()
			w.OnInput(func(e events.Event) {
				if pv.FillType == PaintSolid {
					pv.Canvas.SetFillColor(pv.FillProp(), false) // not final
				}
			})
			w.OnChange(func(e events.Event) {
				if pv.FillType == PaintSolid {
					pv.Canvas.SetFillColor(pv.FillProp(), true) // final
				}
			})
		})

		tree.AddChild(w, func(w *core.Table) {
			pv.fillGrads = w
			w.SetSlice(&pv.Canvas.EditState.Gradients)
			w.OnSelect(func(e events.Event) {
				pv.FillStops = pv.Canvas.EditState.Gradients[w.SelectedIndex].Name
				pv.Canvas.SetFill(pv.FillType, pv.FillType, pv.FillStops)
			})
			w.OnChange(func(e events.Event) {
				pv.Canvas.UpdateGradients()
				if w.SelectedIndex >= 0 {
					pv.FillStops = pv.Canvas.EditState.Gradients[w.SelectedIndex].Name
					pv.Canvas.SetFill(pv.FillType, pv.FillType, pv.FillStops)
				}
				w.Update()
			})
		})

		tree.AddChild(w, func(w *core.Stretch) {})
	})
}

////////  setPaintProp

// setPaintPropNode sets a paint property on given node,
// using given setter function.
func setPaintPropNode(nd svg.Node, fun func(g svg.Node)) {
	if gp, isgp := nd.(*svg.Group); isgp {
		for _, kid := range gp.Children {
			setPaintPropNode(kid.(svg.Node), fun)
		}
		return
	}
	fun(nd)
}

// setPaintPropInput sets paint property from a slider-based input that
// sends continuous [events.Input] events, followed by a final [events.Change]
// event, which should have the final = true flag set. This uses the
// [Action] framework to manage the undo saving dynamics involved.
func (cv *Canvas) setPaintPropInput(act Actions, data string, final bool, fun func(nd svg.Node)) {
	es := &cv.EditState
	sv := cv.SVG
	actStart := false
	finalAct := false
	if final && es.InAction() {
		finalAct = true
	}
	if !final && !es.InAction() {
		final = true
		actStart = true
		es.ActStart(act, data)
		es.ActUnlock()
	}
	if final {
		if !finalAct { // was already saved earlier otherwise
			sv.UndoSave(act.String(), data)
		}
	}
	for nd := range es.Selected {
		setPaintPropNode(nd, fun)
	}
	if final {
		if !actStart {
			es.ActDone()
			cv.ChangeMade()
			sv.NeedsRender()
		}
	} else {
		sv.NeedsRender()
	}
}

// setPaintProp sets paint property on selected nodes,
// using given setter function.
func (cv *Canvas) setPaintProp(actName, val string, fun func(g svg.Node)) {
	es := &cv.EditState
	cv.SVG.UndoSave(actName, val)
	for itm := range es.Selected {
		setPaintPropNode(itm, fun)
	}
	cv.ChangeMade()
	cv.SVG.NeedsRender()
}

func (pv *PaintSetter) PaintTypeStack(pt PaintTypes) int {
	st := 0
	switch pt {
	case PaintSolid:
		st = 1
	case PaintLinear, PaintRadial:
		st = 2
	}
	return st
}

func (pv *PaintSetter) SetStrokeStack(pt PaintTypes) {
	pv.strokeStack.StackTop = pv.PaintTypeStack(pt)
}

func (pv *PaintSetter) SetFillStack(pt PaintTypes) {
	pv.fillStack.StackTop = pv.PaintTypeStack(pt)
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
	sv := cv.SSVG()
	switch pt {
	case PaintLinear:
		sv.GradientUpdateNodeProp(nd, prop, false, sp)
	case PaintRadial:
		sv.GradientUpdateNodeProp(nd, prop, true, sp)
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
	cv.setPaintProp("SetStroke", sp, func(nd svg.Node) {
		cv.SetColorNode(nd, "stroke", prev, pt, sp)
	})
}

// SetStrokeColor sets the stroke color for selected items.
// which can be done dynamically (for [events.Input] events, final = false,
// followed by a final [events.Change] event (final = true)
func (cv *Canvas) SetStrokeColor(sp string, final bool) {
	cv.setPaintPropInput(SetStrokeColor, sp, final,
		func(itm svg.Node) {
			p := itm.AsTree().Properties["stroke"]
			if p != nil {
				itm.AsNodeBase().SetColorProperties("stroke", sp)
				cv.UpdateMarkerColors(itm)
			}
		})
}

func (cv *Canvas) SetStrokeWidth(wp string) {
	cv.setPaintProp("SetStrokeWidth", wp, func(nd svg.Node) {
		g := nd.AsNodeBase()
		if g.Paint.Stroke.Color != nil {
			g.SetProperty("stroke-width", wp)
		}
	})
}

// SetMarkerProperties sets the marker properties
func (cv *Canvas) SetMarkerProperties(start, mid, end string, sc, mc, ec MarkerColors) {
	sv := cv.SVG.SVG
	cv.setPaintProp("SetMarkerProperties", start+" "+mid+" "+end, func(nd svg.Node) {
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
func (cv *Canvas) SetDashProperties(dary []float64) {
	cv.setPaintProp("SetDashProperties", "", func(nd svg.Node) {
		g := nd.AsNodeBase()
		if len(dary) == 0 {
			delete(g.Properties, "stroke-dasharray")
			return
		}
		mary := DashMulWidth(float64(g.Paint.Stroke.Width.Dots), dary)
		ds := DashString(mary)
		g.Properties["stroke-dasharray"] = ds
	})
}

// SetFill sets the fill properties of selected items
// based on previous and current PaintType
func (cv *Canvas) SetFill(prev, pt PaintTypes, fp string) {
	cv.setPaintProp("SetFill", fp, func(nd svg.Node) {
		cv.SetColorNode(nd, "fill", prev, pt, fp)
	})
}

// SetFillColor sets the fill color for selected items,
// which can be done dynamically (for [events.Input] events, final = false,
// followed by a final [events.Change] event (final = true)
func (cv *Canvas) SetFillColor(fp string, final bool) {
	cv.setPaintPropInput(SetFillColor, fp, final,
		func(nd svg.Node) {
			p := nd.AsTree().Properties["fill"]
			if p != nil {
				nd.AsNodeBase().SetColorProperties("fill", fp)
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
	sv.UpdateGradients(es.Gradients)
}

////////  PaintSetter

// UpdateFromNode updates the current settings based on the values in the given Paint
// Style and properties from node (node can be nil)
func (pv *PaintSetter) UpdateFromNode(ps *styles.Paint, nd svg.Node) {
	pv.StrokeType, pv.StrokeStops = pv.GetPaintType(nd, ps.Stroke.Color, "stroke")
	pv.FillType, pv.FillStops = pv.GetPaintType(nd, ps.Fill.Color, "fill")

	// es := &pv.Canvas.EditState
	// grl := &es.Gradients

	pv.SetStrokeStack(pv.StrokeType)
	switch pv.StrokeType {
	case PaintSolid:
		pv.PaintStyle.Stroke.Color = ps.Stroke.Color
	case PaintLinear, PaintRadial:
		// sg.SetSlice(grl)
		pv.SelectStrokeGrad()
	}

	pv.PaintStyle.Stroke.Width = ps.Stroke.Width
	pv.PaintStyle.Stroke.Dashes = ps.Stroke.Dashes

	setMarker := func(ic, cc *core.Chooser, ms string, mc MarkerColors) {
		if ms != "" {
			ic.SetCurrentValue(ms)
		} else {
			ic.SetCurrentIndex(0)
		}
		cc.SetCurrentValue(mc)
	}

	ms, _, mc := MarkerFromNodeProp(nd, "marker-start")
	setMarker(pv.markerStart, pv.markerStartColor, ms, mc)
	ms, _, mc = MarkerFromNodeProp(nd, "marker-mid")
	setMarker(pv.markerMid, pv.markerMidColor, ms, mc)
	ms, _, mc = MarkerFromNodeProp(nd, "marker-end")
	setMarker(pv.markerEnd, pv.markerEndColor, ms, mc)

	pv.SetFillStack(pv.FillType)
	switch pv.FillType {
	case PaintSolid:
		pv.PaintStyle.Fill.Color = ps.Fill.Color
	case PaintLinear, PaintRadial:
		pv.SelectFillGrad()
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
	es := &pv.Canvas.EditState
	grl := &es.Gradients
	sg := pv.strokeGrads
	sg.ResetSelectedIndexes()
	for i, g := range *grl {
		if g.Name == pv.StrokeStops {
			sg.SelectIndex(i)
			break
		}
	}
}

func (pv *PaintSetter) SelectFillGrad() {
	es := &pv.Canvas.EditState
	grl := &es.Gradients
	sg := pv.fillGrads
	sg.ResetSelectedIndexes()
	for i, g := range *grl {
		if g.Name == pv.FillStops {
			sg.SelectIndex(i)
			break
		}
	}
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
	start = pv.markerStart.CurrentItem.Value.(string)
	sc = pv.markerStartColor.CurrentItem.Value.(MarkerColors)
	mid = pv.markerMid.CurrentItem.Value.(string)
	mc = pv.markerMidColor.CurrentItem.Value.(MarkerColors)
	end = pv.markerEnd.CurrentItem.Value.(string)
	ec = pv.markerEndColor.CurrentItem.Value.(MarkerColors)
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
	if pv.dashes.CurrentIndex == 0 {
		return nil
	}
	dnm := pv.dashes.CurrentItem.Value.(string)
	dary, ok := AllDashesMap[dnm]
	if !ok {
		return nil
	}
	return dary
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
