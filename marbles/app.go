package main

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

func (gr *Graph) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(gr.Graph).SetIcon(icons.ShowChart)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(gr.Run).SetIcon(icons.PlayArrow)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(gr.Stop).SetIcon(icons.Stop)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(gr.Step).SetIcon(icons.Step)
	})

	tree.Add(p, func(w *core.Separator) {})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(gr.AddLine).SetIcon(icons.Add)
	})

	tree.Add(p, func(w *core.Separator) {})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(gr.SelectNextMarble).SetText("Next marble").SetIcon(icons.ArrowForward)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(gr.StopSelecting).SetText("Unselect").SetIcon(icons.Close)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(gr.TrackSelectedMarble).SetText("Track").SetIcon(icons.PinDrop)
	})

	tree.Add(p, func(w *core.Separator) {})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(gr.OpenJSON).SetText("Open").SetIcon(icons.Open).SetKey(keymap.Open)
		w.Args[0].SetTag(`extension:".json"`)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(gr.SaveLast).SetText("Save").SetIcon(icons.Save).SetKey(keymap.Save)
		w.Updater(func() {
			w.SetEnabled(gr.State.File != "")
		})
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(gr.SaveJSON).SetText("Save as").SetIcon(icons.SaveAs).SetKey(keymap.SaveAs)
		w.Args[0].SetTag(`extension:".json"`)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(gr.Reset).SetIcon(icons.Reset)
	})
}

func (gr *Graph) MakeBasicElements(b *core.Body) {
	sp := core.NewSplits(b).SetTiles(core.TileSecondLong)
	sp.Styler(func(s *styles.Style) {
		if sp.SizeClass() == core.SizeExpanded {
			s.Direction = styles.Column
		} else {
			s.Direction = styles.Row
		}
	})

	gr.Objects.LinesTable = core.NewTable(sp).SetSlice(&gr.Lines)
	gr.Objects.LinesTable.OnChange(func(e events.Event) {
		gr.Graph()
	})

	gr.Objects.ParamsForm = core.NewForm(sp).SetStruct(&gr.Params)
	gr.Objects.ParamsForm.OnChange(func(e events.Event) {
		gr.Graph()
	})

	gr.Objects.Graph = core.NewCanvas(sp).SetDraw(gr.draw)

	gr.Vectors.Min = math32.Vector2{X: -GraphViewBoxSize, Y: -GraphViewBoxSize}
	gr.Vectors.Max = math32.Vector2{X: GraphViewBoxSize, Y: GraphViewBoxSize}
	gr.Vectors.Size = gr.Vectors.Max.Sub(gr.Vectors.Min)
	var n float32 = 1.0 / float32(TheSettings.GraphInc)
	gr.Vectors.Inc = math32.Vector2{X: n, Y: n}

	statusText := core.NewText(b)
	statusText.Updater(func() {
		if gr.State.File == "" {
			statusText.SetText("Welcome to Cogent Marbles!")
		} else {
			statusText.SetText("<b>" + string(gr.State.File) + "</b>")
		}
	})
}
