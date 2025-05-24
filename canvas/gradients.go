// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"fmt"
	"strings"

	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/svg"
)

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

// Gradients returns the currently defined gradients with stops
// that are shared among obj-specific ones
func (sv *SVG) Gradients() []*Gradient {
	gl := make([]*Gradient, 0)
	for _, nd := range sv.SVG.Defs.Children {
		g, ok := nd.(*svg.Gradient)
		if !ok {
			continue
		}
		if g.StopsName != "" {
			continue
		}
		gr := &Gradient{}
		gr.UpdateFromGrad(g)
		gl = append(gl, gr)
	}
	return gl
}

// UpdateGradients update SVG gradients from given gradient list.
func (sv *SVG) UpdateGradients(gl []*Gradient) {
	nms := make(map[string]bool)
	for _, gr := range gl {
		if _, has := nms[gr.Name]; has {
			id := sv.SVG.NewUniqueID()
			gr.Name = fmt.Sprintf("%d", id)
		}
		nms[gr.Name] = true
	}

	for _, gr := range gl {
		radial := false
		if strings.HasPrefix(gr.Name, "radial") {
			radial = true
		}
		var g *svg.Gradient
		gg := sv.SVG.FindDefByName(gr.Name)
		if gg == nil {
			g, _ = sv.SVG.GradientNew(radial)
		} else {
			g = gg.(*svg.Gradient)
		}
		gr.UpdateToGrad(g)
	}
	sv.SVG.GradientUpdateAllStops()
}

// Gradient represents a single gradient that defines stops
// (referenced in StopName of other gradients).
type Gradient struct { //types:add

	// Icon of gradient, auto generated to display each gradient.
	Icon icons.Icon `edit:"-" table:"no-header" icon-width:"4"`

	// Id is the name of gradient.
	Id string `edit:"-" width:"6"`

	// Name is the full name of gradient as SVG element.
	Name string `display:"-"`

	// Stops are the gradient stops.
	Stops []gradient.Stop `new-window:"+"`
}

func (gr *Gradient) Validate() {
	if gr.Stops == nil {
		gr.ConfigDefaultStops()
	}
	all0 := true
	for i := range gr.Stops {
		if gr.Stops[i].Pos != 0 {
			all0 = false
		}
	}
	if !all0 {
		return
	}
	for i := range gr.Stops {
		gr.Stops[i].Pos = float32(i)
	}
}

// Updates our gradient from svg gradient
func (gr *Gradient) UpdateFromGrad(g *svg.Gradient) {
	_, id := svg.SplitNameIDDig(g.Name)
	gr.Id = fmt.Sprintf("%d", id)
	gr.Name = g.Name
	if g.Grad == nil {
		gr.Stops = nil
		return
	}
	gb := g.Grad.AsBase()
	gr.Stops = slicesx.CopyFrom(gr.Stops, gb.Stops)
	gr.UpdateIcon()
}

// UpdateToGrad updates svg gradient from our gradient.
func (gr *Gradient) UpdateToGrad(g *svg.Gradient) {
	gr.Validate()
	_, id := svg.SplitNameIDDig(g.Name) // we always need to sync to id & name though
	gr.Id = fmt.Sprintf("%d", id)
	gr.Name = g.Name
	if g.Grad == nil {
		if strings.HasPrefix(gr.Name, "radial") {
			g.Grad = gradient.NewRadial()
		} else {
			g.Grad = gradient.NewLinear()
		}
	}
	gb := g.Grad.AsBase()
	gb.Stops = slicesx.CopyFrom(gb.Stops, gr.Stops)
	gr.UpdateIcon()
}

// ConfigDefaultGradient configures a new default gradient
func (es *EditState) ConfigDefaultGradient() {
	es.Gradients = make([]*Gradient, 1)
	gr := &Gradient{}
	es.Gradients[0] = gr
	// gr.ConfigDefaultGradientStops()
	gr.UpdateIcon()
}

// ConfigDefaultStops configures a new default gradient stops.
func (gr *Gradient) ConfigDefaultStops() {
	gr.Stops = make([]gradient.Stop, 2)
	gr.Stops[0] = gradient.Stop{Color: colors.White, Opacity: 1, Pos: 0}
	gr.Stops[1] = gradient.Stop{Color: colors.Blue, Opacity: 1, Pos: 1}
}

// UpdateIcon updates icon
func (gr *Gradient) UpdateIcon() {
	sv := svg.NewSVG(math32.Vec2(128, 128))
	sv.Root.ViewBox.Size = math32.Vec2(32, 32)

	nst := len(gr.Stops)
	px := 30 / float32(nst)
	for i := range gr.Stops {
		bx := svg.NewRect(sv.Root)
		bx.Pos.X = 1 + float32(i)*px
		bx.Pos.Y = 1
		bx.Size.X = px
		bx.Size.Y = 31
		bx.SetProperty("stroke", "none")
		bx.SetProperty("fill", colors.AsHex(gr.Stops[i].Color))
	}
	gr.Icon = icons.Icon(sv.XMLString())
}
