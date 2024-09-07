package main

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
)

// draw renders the graph.
func (gr *Graph) draw(pc *paint.Context) {
	TheGraph.EvalMu.Lock()
	defer TheGraph.EvalMu.Unlock()
	gr.updateCoords()
	gr.drawAxes(pc)
	gr.drawTrackingLines(pc)
	gr.drawLines(pc)
	gr.drawMarbles(pc)
}

func (gr *Graph) updateCoords() {
	if !gr.State.Running || gr.Params.CenterX.Changes || gr.Params.CenterY.Changes {
		sizeFromCenter := math32.Vector2{X: GraphViewBoxSize, Y: GraphViewBoxSize}
		center := math32.Vector2{X: float32(gr.Params.CenterX.Eval(0, 0)), Y: float32(gr.Params.CenterY.Eval(0, 0))}
		gr.Vectors.Min = center.Sub(sizeFromCenter)
		gr.Vectors.Max = center.Add(sizeFromCenter)
		gr.Vectors.Size = sizeFromCenter.MulScalar(2)
	}
}

// canvasCoord converts the given graph coordinate to a normalized 0-1 canvas coordinate.
func (gr *Graph) canvasCoord(v math32.Vector2) math32.Vector2 {
	res := math32.Vector2{}
	res.X = (v.X - gr.Vectors.Min.X) / (gr.Vectors.Max.X - gr.Vectors.Min.X)
	res.Y = (gr.Vectors.Max.Y - v.Y) / (gr.Vectors.Max.Y - gr.Vectors.Min.Y)
	return res
}

func (gr *Graph) drawAxes(pc *paint.Context) {
	pc.StrokeStyle.Color = colors.Scheme.OutlineVariant

	start := gr.canvasCoord(math32.Vec2(gr.Vectors.Min.X, 0))
	end := gr.canvasCoord(math32.Vec2(gr.Vectors.Max.X, 0))
	pc.MoveTo(start.X, start.Y)
	pc.LineTo(end.X, end.Y)
	pc.Stroke()

	start = gr.canvasCoord(math32.Vec2(0, gr.Vectors.Min.Y))
	end = gr.canvasCoord(math32.Vec2(0, gr.Vectors.Max.Y))
	pc.MoveTo(start.X, start.Y)
	pc.LineTo(end.X, end.Y)
	pc.Stroke()
}

func (gr *Graph) drawTrackingLines(pc *paint.Context) {
	for _, m := range gr.Marbles {
		if !m.TrackingInfo.Track {
			continue
		}
		for j, pos := range m.TrackingInfo.History {
			cpos := gr.canvasCoord(pos)
			if j == 0 {
				pc.MoveTo(cpos.X, cpos.Y)
			} else {
				pc.LineTo(cpos.X, cpos.Y)
			}
		}
		pc.StrokeStyle.Color = colors.Uniform(m.Color)
		pc.Stroke()
	}
}

func (gr *Graph) drawLines(pc *paint.Context) {
	for _, ln := range gr.Lines {
		// TODO: this logic doesn't work
		// If the line doesn't change over time then we don't need to keep graphing it while running marbles
		// if !ln.Changes && gr.State.Running && !gr.Params.CenterX.Changes && !gr.Params.CenterY.Changes {
		// 	continue
		// }
		ln.draw(gr, pc)
	}
}

func (ln *Line) draw(gr *Graph, pc *paint.Context) {
	start := true
	skipped := false
	for x := TheGraph.Vectors.Min.X; x < TheGraph.Vectors.Max.X; x += TheGraph.Vectors.Inc.X {
		if TheGraph.State.Error != nil {
			return
		}
		fx := float64(x)
		y := ln.Expr.Eval(fx, TheGraph.State.Time, ln.TimesHit)
		GraphIf := ln.GraphIf.EvalBool(fx, y, TheGraph.State.Time, ln.TimesHit)
		if GraphIf && TheGraph.Vectors.Min.Y < float32(y) && TheGraph.Vectors.Max.Y > float32(y) {
			coord := gr.canvasCoord(math32.Vec2(x, float32(y)))
			if start || skipped {
				pc.MoveTo(coord.X, coord.Y)
				start, skipped = false, false
			} else {
				pc.LineTo(coord.X, coord.Y)
			}
		} else {
			skipped = true
		}
	}
	pc.StrokeStyle.Color = colors.Uniform(ln.Colors.Color)
	pc.StrokeStyle.Width.Dp(4)
	pc.ToDots()
	pc.Stroke()
}

func (gr *Graph) drawMarbles(pc *paint.Context) {
	for i, m := range gr.Marbles {
		pos := gr.canvasCoord(m.Pos)
		if i == gr.State.SelectedMarble {
			pc.DrawCircle(pos.X, pos.Y, 0.02)
			pc.FillStyle.Color = colors.Scheme.Warn.Container
			pc.Fill()
		}
		pc.DrawCircle(pos.X, pos.Y, 0.005)
		pc.FillStyle.Color = colors.Uniform(m.Color)
		pc.Fill()
	}
}
