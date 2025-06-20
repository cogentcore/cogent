// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
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
	pv.PaintStyle.Defaults()
	pv.PaintStyle.Stroke.Color = colors.Uniform(color.Black) // default is off
	pv.PaintStyle.Stroke.Width.Px(1)                         // dp is not understood by svg..

	DashIconsInit()
	MarkerIconsInit()

	pstyle := &pv.PaintStyle

	pv.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(0, 1)
	})

	tree.AddChildAt(pv, "stroke-lab", func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
		})
		tree.AddChild(w, func(w *core.Text) {
			w.SetText("<b>Stroke:</b>")
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
					pv.SetStroke(pv.curStrokeType, pv.StrokeType, pv.StrokeStops)
				} else {
					pv.SetStroke(pv.curStrokeType, pv.StrokeType, pv.StrokeProp())
				}
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
				s.SetTextWrap(false)
				s.Align.Items = styles.Center
			})
		})
		tree.AddChild(w, func(w *core.Spinner) {
			core.Bind(&pstyle.Stroke.Width.Value, w)
			w.SetMin(0).SetStep(0.05)
			w.OnChange(func(e events.Event) {
				pstyle.ToDots()
				if pv.IsStrokeOn() {
					pv.Canvas.SetStrokeProperty("stroke-width", pstyle.Stroke.Width.String())
				}
			})
		})

		tree.AddChild(w, func(w *core.Chooser) {
			core.Bind(&pstyle.Stroke.Width.Unit, w)
			w.OnChange(func(e events.Event) {
				if pv.IsStrokeOn() {
					pv.Canvas.SetStrokeProperty("stroke-width", pstyle.Stroke.Width.String())
				}
			})
		})

		tree.AddChild(w, func(w *core.Chooser) {
			pv.dashes = w
			w.Styler(func(s *styles.Style) {
				s.IconSize.X.Em(3)
			})
			w.SetItems(AllDashItems...)
			w.OnChange(func(e events.Event) {
				if pv.IsStrokeOn() {
					pv.Canvas.SetDashProperties(pv.StrokeDashProp())
				}
			})
		})

		tree.AddChild(w, func(w *core.Chooser) {
			core.Bind(&pstyle.Stroke.Cap, w)
			w.OnChange(func(e events.Event) {
				if pv.IsStrokeOn() {
					pv.Canvas.SetStrokeProperty("stroke-linecap", pstyle.Stroke.Cap.String())
				}
			})
		})

		tree.AddChild(w, func(w *core.Chooser) {
			core.Bind(&pstyle.Stroke.Join, w)
			w.OnChange(func(e events.Event) {
				if pv.IsStrokeOn() {
					pv.Canvas.SetStrokeProperty("stroke-linejoin", pstyle.Stroke.Join.String())
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
				s.IconSize.Set(units.Em(3), units.Em(2))
			})
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
				s.IconSize.Set(units.Em(3), units.Em(2))
			})
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
				s.IconSize.Set(units.Em(3), units.Em(2))
			})
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
			core.Bind(&pstyle.Stroke.Color, w)
			w.HandleValueOnInput()
			w.OnInput(func(e events.Event) {
				if pv.StrokeType == PaintSolid {
					pv.UpdateStrokeOpacity()
					pv.Canvas.SetStrokeColor(pv.StrokeProp(), false) // not final
				}
			})
			w.OnChange(func(e events.Event) {
				if pv.StrokeType == PaintSolid {
					pv.UpdateStrokeOpacity()
					pv.Canvas.SetStrokeColor(pv.StrokeProp(), true) // final
				}
			})
		})

		tree.AddChild(w, func(w *core.Table) {
			pv.strokeGrads = w
			w.SetSlice(&pv.Canvas.EditState.Gradients)
			w.OnSelect(func(e events.Event) {
				pv.StrokeStops = pv.Canvas.EditState.Gradients[w.SelectedIndex].Name
				pv.SetStroke(pv.StrokeType, pv.StrokeType, pv.StrokeStops)
			})
			w.OnChange(func(e events.Event) {
				pv.Canvas.UpdateGradients()
				if w.SelectedIndex >= 0 {
					pv.StrokeStops = pv.Canvas.EditState.Gradients[w.SelectedIndex].Name
					pv.SetStroke(pv.StrokeType, pv.StrokeType, pv.StrokeStops)
				}
				w.Update()
			})
		})
	})

	tree.AddChild(pv, func(w *core.Separator) {})

	tree.AddChildAt(pv, "fill-lab", func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.CenterAll()
		})
		tree.AddChild(w, func(w *core.Text) {
			w.SetText("<b>Fill:</b>")
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
					pv.SetFill(pv.curFillType, pv.FillType, pv.FillStops)
				} else {
					pv.SetFill(pv.curFillType, pv.FillType, pv.FillProp())
				}
				pv.curFillType = pv.FillType
				pv.Update()
			})
		})
		tree.AddChild(w, func(w *core.Chooser) {
			core.Bind(&pstyle.Fill.Rule, w)
			w.OnChange(func(e events.Event) {
				if pv.IsFillOn() {
					pv.Canvas.SetFillProperty("fill-rule", pstyle.Fill.Rule.String())
				}
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
			core.Bind(&pstyle.Fill.Color, w)
			w.HandleValueOnInput()
			w.OnInput(func(e events.Event) {
				if pv.FillType == PaintSolid {
					pv.UpdateFillOpacity()
					pv.Canvas.SetFillColor(pv.FillProp(), false) // not final
				}
			})
			w.OnChange(func(e events.Event) {
				if pv.FillType == PaintSolid {
					pv.UpdateFillOpacity()
					pv.Canvas.SetFillColor(pv.FillProp(), true) // final
				}
			})
		})

		tree.AddChild(w, func(w *core.Table) {
			pv.fillGrads = w
			w.SetSlice(&pv.Canvas.EditState.Gradients)
			w.OnSelect(func(e events.Event) {
				pv.FillStops = pv.Canvas.EditState.Gradients[w.SelectedIndex].Name
				pv.SetFill(pv.FillType, pv.FillType, pv.FillStops)
			})
			w.OnChange(func(e events.Event) {
				pv.Canvas.UpdateGradients()
				if w.SelectedIndex >= 0 {
					pv.FillStops = pv.Canvas.EditState.Gradients[w.SelectedIndex].Name
					pv.SetFill(pv.FillType, pv.FillType, pv.FillStops)
				}
				w.Update()
			})
		})

		tree.AddChild(w, func(w *core.Stretch) {})
	})
	tree.AddChildAt(pv, "opacity", func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
		})
		tree.AddChild(w, func(w *core.Text) {
			w.SetText("Opacity: ").Styler(func(s *styles.Style) {
				s.SetTextWrap(false)
			})
		})
		tree.AddChild(w, func(w *core.Slider) {
			core.Bind(&pstyle.Opacity, w)
			w.SetMin(0).SetMax(1).SetStep(0.05)
			w.SetTooltip("Global opacity applied to both stroke and fill")
			w.HandleValueOnInput()
			w.OnInput(func(e events.Event) {
				pv.Canvas.SetOpacity(fmt.Sprintf("%g", pstyle.Opacity), false) // not final
			})
			w.OnChange(func(e events.Event) {
				pv.Canvas.SetOpacity(fmt.Sprintf("%g", pstyle.Opacity), true) // final
			})
			w.Styler(func(s *styles.Style) {
				w.ValueColor = nil
				g := gradient.NewLinear()
				for c := float32(0); c <= 1; c += 0.05 {
					gc := colors.WithAF32(colors.Black, c)
					g.AddStop(gc, c)
				}
				s.Background = g
			})
		})
	})

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

// OpacityFromColor extracts the opacity as a 0-1 float from given uniform color-as-image.
func OpacityFromColor(img image.Image) float32 {
	return float32(colors.ToUniform(img).A) / 255
}

func (pv *PaintSetter) UpdateStrokeOpacity() {
	pv.PaintStyle.Stroke.Opacity = OpacityFromColor(pv.PaintStyle.Stroke.Color)
}

func (pv *PaintSetter) UpdateFillOpacity() {
	pv.PaintStyle.Fill.Opacity = OpacityFromColor(pv.PaintStyle.Fill.Color)
}

// SetStrokeOpacity sets stroke opacity
func (pv *PaintSetter) SetStrokeOpacity(nd svg.Node) {
	nd.AsNodeBase().SetProperty("stroke-opacity", pv.PaintStyle.Stroke.Opacity)
}

// SetStrokeOthers sets opacity and stroke width properties
func (pv *PaintSetter) SetStrokeOthers(nd svg.Node) {
	if !pv.IsStrokeOn() {
		return
	}
	nb := nd.AsNodeBase()
	pv.SetStrokeOpacity(nd)
	nb.SetProperty("strodke-width", pv.PaintStyle.Stroke.Width.String())
	nb.SetProperty("stroke-linecap", pv.PaintStyle.Stroke.Cap.String())
	nb.SetProperty("stroke-linejoin", pv.PaintStyle.Stroke.Join.String())
}

func (pv *PaintSetter) SetFillOthers(nd svg.Node) {
	if !pv.IsFillOn() {
		return
	}
	nd.AsNodeBase().SetProperty("fill-opacity", pv.PaintStyle.Fill.Opacity)
	nd.AsNodeBase().SetProperty("fill-rule", pv.PaintStyle.Fill.Rule.String())
}

// SetProperties sets the properties for given node according to current settings
func (pv *PaintSetter) SetProperties(nd svg.Node) {
	cv := pv.Canvas
	pv.SetColorNode(nd, "stroke", pv.StrokeType, pv.StrokeType, pv.StrokeProp())
	if pv.IsStrokeOn() {
		cv.SetMarkerProperties(pv.MarkerProperties())
		pv.SetStrokeOthers(nd)
	}
	pv.SetColorNode(nd, "fill", pv.FillType, pv.FillType, pv.FillProp())
	pv.SetFillOthers(nd)
}

func (pv *PaintSetter) SetStrokeStack(pt PaintTypes) {
	pv.strokeStack.StackTop = pv.PaintTypeStack(pt)
}

func (pv *PaintSetter) SetFillStack(pt PaintTypes) {
	pv.fillStack.StackTop = pv.PaintTypeStack(pt)
}

// SetColorNode sets the color properties of Node
// based on previous and current PaintType
func (pv *PaintSetter) SetColorNode(nd svg.Node, prop string, prev, pt PaintTypes, sp string) {
	cv := pv.Canvas
	sv := cv.SSVG()

	others := func() {
		if prop == "stroke" {
			pv.SetStrokeOthers(nd)
		} else {
			pv.SetFillOthers(nd)
		}
	}

	setPropsNode(nd, func(nd svg.Node) {
		switch pt {
		case PaintLinear:
			sv.GradientUpdateNodeProp(nd, prop, false, sp)
			others()
		case PaintRadial:
			sv.GradientUpdateNodeProp(nd, prop, true, sp)
			others()
		default:
			if prev == PaintLinear || prev == PaintRadial {
				pstr := reflectx.ToString(nd.AsTree().Properties[prop])
				sv.GradientDeleteForNode(nd, pstr)
			}
			if pt == PaintOff {
				nd.AsNodeBase().SetColorProperties(prop, "none")
			} else {
				nd.AsNodeBase().SetColorProperties(prop, sp)
				others()
			}
		}
	})
	cv.UpdateMarkerColors(nd)
	cv.UpdateTree()
}

// SetStroke sets the stroke properties of selected items
// based on previous and current PaintType
func (pv *PaintSetter) SetStroke(prev, pt PaintTypes, sp string) {
	cv := pv.Canvas
	cv.setPropsOnSelected("SetStroke", sp, func(nd svg.Node) {
		pv.SetColorNode(nd, "stroke", prev, pt, sp)
		if pv.StrokeType == PaintSolid {
			pv.SetStrokeOpacity(nd)
		}
	})
}

// SetStrokeColor sets the stroke color for selected items.
// which can be done dynamically (for [events.Input] events, final = false,
// followed by a final [events.Change] event (final = true)
func (cv *Canvas) SetStrokeColor(sp string, final bool) {
	cv.setPropsOnSelectedInput(SetStrokeColor, sp, final,
		func(nd svg.Node) {
			p := nd.AsTree().Properties["stroke"]
			if p != nil {
				nd.AsNodeBase().SetColorProperties("stroke", sp)
				cv.PaintSetter().SetStrokeOpacity(nd)
				cv.UpdateMarkerColors(nd)
			}
		})
}

// SetStrokeProperty sets given property only if stroke.color is non-nil.
func (cv *Canvas) SetStrokeProperty(prop, wp string) {
	cv.setPropsOnSelected(prop, wp, func(nd svg.Node) {
		g := nd.AsNodeBase()
		if g.Paint.Stroke.Color != nil {
			g.SetProperty(prop, wp)
		}
	})
}

// SetMarkerProperties sets the marker properties
func (cv *Canvas) SetMarkerProperties(start, mid, end string, sc, mc, ec MarkerColors) {
	sv := cv.SVG.SVG
	cv.setPropsOnSelected("SetMarkerProperties", start+" "+mid+" "+end, func(nd svg.Node) {
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
	cv.setPropsOnSelected("SetDashProperties", "", func(nd svg.Node) {
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
func (pv *PaintSetter) SetFill(prev, pt PaintTypes, fp string) {
	cv := pv.Canvas
	cv.setPropsOnSelected("SetFill", fp, func(nd svg.Node) {
		pv.SetColorNode(nd, "fill", prev, pt, fp)
	})
}

// SetFillColor sets the fill color for selected items,
// which can be done dynamically (for [events.Input] events, final = false,
// followed by a final [events.Change] event (final = true)
func (cv *Canvas) SetFillColor(fp string, final bool) {
	cv.setPropsOnSelectedInput(SetFillColor, fp, final,
		func(nd svg.Node) {
			p := nd.AsTree().Properties["fill"]
			if p != nil {
				nd.AsNodeBase().SetColorProperties("fill", fp)
				cv.PaintSetter().SetFillOthers(nd)
			}
		})
}

// SetFillProperty sets given property only if fill.color is non-nil.
func (cv *Canvas) SetFillProperty(prop, wp string) {
	cv.setPropsOnSelected(prop, wp, func(nd svg.Node) {
		g := nd.AsNodeBase()
		if g.Paint.Fill.Color != nil {
			g.SetProperty(prop, wp)
		}
	})
}

// SetOpacity sets the global opacity for selected items.
// which can be done dynamically (for [events.Input] events, final = false,
// followed by a final [events.Change] event (final = true)
func (cv *Canvas) SetOpacity(sp string, final bool) {
	cv.setPropsOnSelectedInput(SetOpacity, sp, final,
		func(nd svg.Node) {
			nd.AsNodeBase().Properties["opacity"] = sp
		})
}

// UpdateFromNode updates the current settings based on the values in the given Paint
// Style and properties from node (node can be nil)
func (pv *PaintSetter) UpdateFromNode(ps *styles.Paint, nd svg.Node) {
	pv.StrokeType, pv.StrokeStops = pv.GetPaintType(nd, ps.Stroke.Color, "stroke")
	pv.FillType, pv.FillStops = pv.GetPaintType(nd, ps.Fill.Color, "fill")

	pv.SetStrokeStack(pv.StrokeType)
	switch pv.StrokeType {
	case PaintSolid:
		pv.PaintStyle.Stroke.Color = gradient.ApplyOpacity(ps.Stroke.Color, ps.Stroke.Opacity)
		pv.PaintStyle.Stroke.Opacity = ps.Stroke.Opacity
	case PaintLinear, PaintRadial:
		pv.SelectStrokeGrad()
	}

	pv.PaintStyle.Stroke.Width = ps.Stroke.Width
	pv.PaintStyle.Stroke.Dashes = ps.Stroke.Dashes
	pv.PaintStyle.Stroke.Cap = ps.Stroke.Cap
	pv.PaintStyle.Stroke.Join = ps.Stroke.Join

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
		pv.PaintStyle.Fill.Color = gradient.ApplyOpacity(ps.Fill.Color, ps.Fill.Opacity)
		pv.PaintStyle.Fill.Opacity = ps.Fill.Opacity
		pv.PaintStyle.Fill.Rule = ps.Fill.Rule
	case PaintLinear, PaintRadial:
		pv.SelectFillGrad()
	}
	pv.PaintStyle.Opacity = ps.Opacity
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
	switch pv.StrokeType {
	case PaintOff:
		return "none"
	case PaintSolid:
		// opacity handled separately: always report colors as pure
		return colors.AsHex(colors.WithA(colors.ToUniform(pv.PaintStyle.Stroke.Color), 255))
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
		// opacity handled separately: always report colors as pure
		return colors.AsHex(colors.WithA(colors.ToUniform(pv.PaintStyle.Fill.Color), 255))
	case PaintLinear:
		return pv.FillStops
	case PaintRadial:
		return pv.FillStops
	case PaintInherit:
		return "inherit"
	}
	return "none"
}

type PaintTypes int32 //enums:enum -trim-prefix Paint

const (
	PaintOff PaintTypes = iota
	PaintSolid
	PaintLinear
	PaintRadial
	PaintInherit
)
