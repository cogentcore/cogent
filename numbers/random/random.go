// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package random plots histograms of random distributions.
package random

//go:generate core generate

import (
	"strconv"

	"cogentcore.org/core/base/randx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32/minmax"
	"cogentcore.org/core/plot/plotcore"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/histogram"
	"cogentcore.org/core/tensor/table"
	"cogentcore.org/core/tree"
)

// Random is the random distribution plotter widget.
type Random struct {
	core.Frame
	Data
}

// Data contains the random distribution plotter data and options.
type Data struct { //types:add
	// random params
	Dist randx.RandParams `display:"add-fields"`

	// number of samples
	NumSamples int

	// number of bins in the histogram
	NumBins int

	// range for histogram
	Range minmax.F64

	// table for raw data
	Table *table.Table `display:"no-inline"`

	// histogram of data
	Histogram *table.Table `display:"no-inline"`

	// the plot
	plot *plotcore.PlotEditor `display:"-"`
}

// logPrec is the precision for saving float values in logs.
const logPrec = 4

func (rd *Random) Init() {
	rd.Frame.Init()

	rd.Dist.Defaults()
	rd.Dist.Dist = randx.Gaussian
	rd.Dist.Mean = 0.5
	rd.Dist.Var = 0.15
	rd.NumSamples = 1000000
	rd.NumBins = 100
	rd.Range.Set(0, 1)
	rd.Table = &table.Table{}
	rd.Histogram = &table.Table{}
	rd.ConfigTable(rd.Table)
	rd.Plot()

	rd.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	tree.AddChild(rd, func(w *core.Splits) {
		w.SetSplits(0.3, 0.7)
		tree.AddChild(w, func(w *core.Form) {
			w.SetStruct(&rd.Data)
			w.OnChange(func(e events.Event) {
				rd.Plot()
			})
		})
		tree.AddChild(w, func(w *plotcore.PlotEditor) {
			rd.plot = rd.ConfigPlot(w, rd.Histogram)
		})
	})
}

// Plot generates the data and plots a histogram of results.
func (rd *Random) Plot() { //types:add
	dt := rd.Table

	dt.SetNumRows(rd.NumSamples)
	for vi := 0; vi < rd.NumSamples; vi++ {
		vl := rd.Dist.Gen()
		dt.SetFloat("Value", vi, float64(vl))
	}

	histogram.F64Table(rd.Histogram, dt.Columns[0].(*tensor.Float64).Values, rd.NumBins, rd.Range.Min, rd.Range.Max)
	if rd.plot != nil {
		rd.plot.UpdatePlot()
	}
}

func (rd *Random) ConfigTable(dt *table.Table) {
	dt.SetMetaData("name", "Data")
	dt.SetMetaData("read-only", "true")
	dt.SetMetaData("precision", strconv.Itoa(logPrec))

	dt.AddFloat64Column("Value")
}

func (rd *Random) ConfigPlot(plt *plotcore.PlotEditor, dt *table.Table) *plotcore.PlotEditor {
	plt.Params.Title = "Random distribution histogram"
	plt.Params.XAxisColumn = "Value"
	plt.Params.Type = plotcore.Bar
	plt.Params.XAxisRotation = 45
	plt.SetTable(dt)
	plt.SetColParams("Value", plotcore.Off, plotcore.FloatMin, 0, plotcore.FloatMax, 0)
	plt.SetColParams("Count", plotcore.On, plotcore.FixMin, 0, plotcore.FloatMax, 0)
	return plt
}

func (rd *Random) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(rd.Plot).SetIcon(icons.ScatterPlot)
	})
	tree.Add(p, func(w *core.Separator) {})
	if rd.plot != nil {
		rd.plot.MakeToolbar(p)
	}
}
