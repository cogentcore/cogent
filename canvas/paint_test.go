// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"slices"
	"strings"
	"testing"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// prop returns the given property of the node with the given name as a string.
func prop(t *testing.T, cv *Canvas, name, prop string) string {
	t.Helper()
	p := node(t, cv, name).AsTree().Property(prop)
	if p == nil {
		return ""
	}
	return reflectx.ToString(p)
}

// paintChild returns the widget at the given path of child names and
// indexes in the paint setter.
func paintSwitches(pv *PaintSetter, frame string) *core.Switches {
	return pv.ChildByName(frame).AsTree().Child(1).(*core.Switches)
}

// TestPaintColors tests setting the stroke and fill colors and the
// opacity of the selection, as the color pickers and slider do.
func TestPaintColors(t *testing.T) {
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		pv := cv.PaintSetter()
		selectNodes(t, cv, "rect59")
		cv.UpdateTabs()
		assert.Equal(t, PaintSolid, pv.FillType)
		assert.Equal(t, "#ff00ff", strings.ToLower(pv.FillProp()), "from the selection")

		pv.PaintStyle.Fill.Color = colors.Uniform(colors.Red)
		pv.UpdateFillOpacity()
		cv.SetFillColor(pv.FillProp(), false) // input
		cv.SetFillColor(pv.FillProp(), true)  // change
		assert.Equal(t, "#ff0000", strings.ToLower(prop(t, cv, "rect59", "fill")))
		assert.Equal(t, "SetFillColor", cv.Undo(), "one undo for the input and change")
		assert.NotEqual(t, "#ff0000", strings.ToLower(prop(t, cv, "rect59", "fill")))
	}, func(cv *Canvas) { // after the undone drawing is rendered (styled)
		pv := cv.PaintSetter()
		selectNodes(t, cv, "rect59")
		pv.PaintStyle.Stroke.Color = colors.Uniform(colors.Blue)
		pv.UpdateStrokeOpacity()
		cv.SetStrokeColor(pv.StrokeProp(), true)
		assert.Equal(t, "#0000ff", strings.ToLower(prop(t, cv, "rect59", "stroke")))
		cv.SetOpacity("0.5", false)
		cv.SetOpacity("0.5", true)
		assert.Equal(t, "0.5", prop(t, cv, "rect59", "opacity"))
		cv.SetStrokeProperty("stroke-width", "3px")
		assert.Equal(t, "3px", prop(t, cv, "rect59", "stroke-width"))
		cv.SetFillProperty("fill-rule", "evenodd")
		assert.Equal(t, "evenodd", prop(t, cv, "rect59", "fill-rule"))
		cv.SetDashProperties([]float64{2, 1})
		assert.NotEmpty(t, prop(t, cv, "rect59", "stroke-dasharray"))
		cv.SetDashProperties(nil)
		assert.Empty(t, prop(t, cv, "rect59", "stroke-dasharray"))
		assert.InDelta(t, 0.5, OpacityFromColor(colors.Uniform(colors.WithAF32(colors.Red, 0.5))), 0.01)
	})
}

// TestPaintTypes tests changing the paint type of the stroke and fill
// of the selection with the switches, including gradients.
func TestPaintTypes(t *testing.T) {
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		pv := cv.PaintSetter()
		selectNodes(t, cv, "rect59")
		cv.UpdateTabs()
		set := func(frame string, pt PaintTypes, field *PaintTypes) {
			*field = pt
			sw := paintSwitches(pv, frame)
			sw.Update()
			sw.SendChange()
		}
		set("fill-lab", PaintOff, &pv.FillType)
		assert.Equal(t, "none", prop(t, cv, "rect59", "fill"))
		set("fill-lab", PaintLinear, &pv.FillType)
		assert.True(t, strings.HasPrefix(prop(t, cv, "rect59", "fill"), "url(#linear"), prop(t, cv, "rect59", "fill"))
		require.NotEmpty(t, cv.EditState.Gradients, "the default gradient")
		cv.UpdateTabs()
		assert.Equal(t, PaintLinear, pv.FillType, "from the node")
		set("fill-lab", PaintRadial, &pv.FillType)
		assert.True(t, strings.HasPrefix(prop(t, cv, "rect59", "fill"), "url(#radial"), prop(t, cv, "rect59", "fill"))
		set("fill-lab", PaintSolid, &pv.FillType)
		assert.True(t, strings.HasPrefix(prop(t, cv, "rect59", "fill"), "#"), "solid again")
		set("fill-lab", PaintInherit, &pv.FillType)
		assert.Equal(t, "inherit", pv.FillProp())

		set("stroke-lab", PaintOff, &pv.StrokeType)
		assert.Equal(t, "none", prop(t, cv, "rect59", "stroke"))
		assert.False(t, pv.IsStrokeOn())
		set("stroke-lab", PaintLinear, &pv.StrokeType)
		assert.True(t, strings.HasPrefix(prop(t, cv, "rect59", "stroke"), "url(#linear"))
		cv.UpdateTabs()
		assert.Equal(t, PaintLinear, pv.StrokeType)
		set("stroke-lab", PaintRadial, &pv.StrokeType)
		set("stroke-lab", PaintSolid, &pv.StrokeType)
		assert.True(t, pv.IsStrokeOn())
		set("stroke-lab", PaintInherit, &pv.StrokeType)
		assert.Equal(t, "inherit", pv.StrokeProp())
	})
}

// TestMarkers tests setting markers on a path, with each kind of color.
func TestMarkers(t *testing.T) {
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		pv := cv.PaintSetter()
		path := cv.SVG.Root().Child(4).AsTree().Name // the id of path81 is not unique, so it is renamed
		selectNodes(t, cv, path)
		cv.UpdateTabs()
		require.NotEmpty(t, AllMarkerNames)
		name := AllMarkerNames[1]
		set := func(ch *core.Chooser, v any) {
			ch.SetCurrentValue(v)
			ch.SendChange()
		}
		set(pv.markerStart, name)
		mp := prop(t, cv, path, "marker-start")
		assert.Contains(t, mp, name)
		nm, _, mc := MarkerFromNodeProp(node(t, cv, path), "marker-start")
		assert.Equal(t, name, nm)
		assert.Equal(t, MarkerDef, mc)

		set(pv.markerStartColor, MarkerCopy)
		_, id, mc := MarkerFromNodeProp(node(t, cv, path), "marker-start")
		assert.Equal(t, MarkerCopy, mc, "a copy for the node")
		assert.NotZero(t, id)
		cv.SetStrokeColor("#ff0000", true) // updates the copied marker color
		set(pv.markerMid, name)
		set(pv.markerMidColor, MarkerCust)
		_, _, mc = MarkerFromNodeProp(node(t, cv, path), "marker-mid")
		assert.Equal(t, MarkerCust, mc)
		set(pv.markerEnd, name)
		set(pv.markerEndColor, MarkerDef)
		cv.UpdateTabs()
		assert.Equal(t, name, pv.markerEnd.CurrentItem.Value, "from the node")

		set(pv.markerStart, "-")
		assert.Empty(t, prop(t, cv, path, "marker-start"), "removed")
		assert.Equal(t, "-", IconToMarkerName(MarkerNameToIcon("-")))

		MarkerSetProp(cv.SSVG(), node(t, cv, path), "marker-end", "no-such-marker", MarkerDef)
	})
}

// TestDashes tests matching and adding dash patterns.
func TestDashes(t *testing.T) {
	isNew, nm := DashMatchArray(2, nil)
	assert.False(t, isNew)
	assert.Equal(t, "dash-solid", nm)
	dary := AllDashesMap[AllDashNames[2]]
	isNew, nm = DashMatchArray(2, DashMulWidth(2, dary))
	assert.False(t, isNew)
	assert.Equal(t, AllDashNames[2], nm)
	isNew, nm = DashMatchArray(0, []float64{17, 3, 11})
	assert.True(t, isNew)
	assert.Equal(t, "custom-17-3-11", nm)
	assert.Equal(t, "1,2.5", DashString([]float64{1, 2.5}))
	DashIconsInit()
	assert.Contains(t, AllDashNames, nm)
}

// TestText tests setting the text and its style from the text form.
func TestText(t *testing.T) {
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		selectNodes(t, cv, "text511")
		cv.UpdateTabs()
		es := &cv.EditState
		assert.True(t, es.SelectIsText)
		assert.Equal(t, "Text", es.Text.String, "from the node")
		es.Text.String = "Hello"
		es.Text.Font.Size.Px(48)
		es.Text.Update()
		assert.Equal(t, "Hello", node(t, cv, "tspan162").(*svg.Text).Text)
		assert.Equal(t, "48px", prop(t, cv, "text511", "font-size"))
		cv.SetText("") // nothing
		cv.SetTool(TextTool)
		cv.UpdateModalToolbar()
		tf := tree.ChildByType[*core.TextField](cv.modalTools)
		require.NotNil(t, tf, "the text toolbar")
		es.Text.String = "World"
		tf.Update()
		tf.SendChange()
		assert.Equal(t, "World", node(t, cv, "tspan162").(*svg.Text).Text)
		sp := tree.ChildByType[*core.Spinner](cv.modalTools)
		sp.SetValue(24)
		sp.SendChange()
		ch := tree.ChildByType[*core.Chooser](cv.modalTools)
		ch.SendChange()
		assert.Equal(t, "24px", prop(t, cv, "text511", "font-size"))

		selectNodes(t, cv, "rect59", "rect318")
		cv.SetText("several") // only one
	})
}

// TestGradients tests the list of gradients and making their icons.
func TestGradients(t *testing.T) {
	runCanvas(t, "gradientgp.svg", func(cv *Canvas) {
		es := &cv.EditState
		require.NotEmpty(t, es.Gradients)
		gr := es.Gradients[0]
		assert.NotEmpty(t, gr.Icon)
		assert.NotEmpty(t, gr.Stops)
		gr.Stops[0].Color = colors.Red
		cv.UpdateGradients()
		g := cv.SSVG().FindDefByName(gr.Name).(*svg.Gradient)
		assert.Equal(t, colors.Red, g.Grad.AsBase().Stops[0].Color)
		assert.Equal(t, gr.Name, cv.DefaultGradient())

		dup := &Gradient{Name: gr.Name} // duplicate name, no stops
		es.Gradients = append(es.Gradients, dup)
		cv.UpdateGradients()
		assert.NotEqual(t, gr.Name, dup.Name, "renamed")
		assert.Len(t, dup.Stops, 2, "default stops")
		z := &Gradient{Stops: slices.Clone(gr.Stops)}
		for i := range z.Stops {
			z.Stops[i].Pos = 0
		}
		z.Validate()
		assert.Equal(t, float32(1), z.Stops[1].Pos, "spread out")
	})
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		assert.Empty(t, cv.EditState.Gradients)
		assert.NotEmpty(t, cv.DefaultGradient(), "made")
	})
}
