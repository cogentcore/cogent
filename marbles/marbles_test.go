// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"math"
	"path/filepath"
	"testing"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newGraph returns the global graph, newly initialized in a new body with
// the default settings and a toolbar, and a temporary home directory for
// the autosave file.
func newGraph(t *testing.T) (*Graph, *core.Body) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	TheSettings.Defaults()
	TheGraph = Graph{}
	b := core.NewBody()
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(TheGraph.MakeToolbar)
	})
	TheGraph.Init(b)
	return &TheGraph, b
}

// setLines sets the expressions of the lines of the graph and graphs it.
func setLines(gr *Graph, exprs ...string) {
	gr.Lines = nil
	for _, e := range exprs {
		gr.Lines = append(gr.Lines, &Line{Expr: Expr{Expr: e}})
	}
	gr.Graph()
}

// eval compiles and evaluates the given expression at x, with the functions
// of the graph.
func eval(t *testing.T, expr string, x float64) float64 {
	t.Helper()
	ex := Expr{Expr: expr}
	require.NoError(t, ex.Compile(), expr)
	return ex.Eval(x, 0, 0)
}

// TestExpressions tests the syntax of the expressions: implicit
// multiplication, function calls without parentheses, symbols, and the
// default functions.
func TestExpressions(t *testing.T) {
	newGraph(t)
	tests := []struct {
		expr string
		x    float64
		want float64
	}{
		{"2x+1", 3, 7},
		{"x^2", 3, 9},
		{"x**2", 3, 9},
		{"sinx", math.Pi / 2, 1},
		{"2sin(x)", math.Pi / 2, 2},
		{"sqrt(x)", 16, 4},
		{"√x", 16, 4},
		{"pi", 0, math.Pi},
		{"2pi", 0, 2 * math.Pi},
		{"(x+1)(x-1)", 3, 8},
		{"x-.5", 1, 0.5},
		{"-x", 2, -2},
		{"if(x>0, 1, 2)", -1, 2},
		{"if(true, 1, 2)", 0, 1},
		{"max(1, x, 3)", 5, 5},
		{"min(1, x, 3)", 5, 1},
		{"avg(1, x, 3)", 5, 3},
		{"abs(x)", -2, 2},
		{"log(x, 2)", 8, 3},
		{"ln(e)", 0, 1},
		{"floor(x)", 2.5, 2},
		{"ceil(x)", 2.5, 3},
		{"round(x)", 2.4, 2},
		{"fact(3)", 0, 6},
		{"mod(x, 3)", 7, 1},
		{"pow(x, 3)", 2, 8},
		{"exp(0)", 0, 1},
		{"cbrt(x)", 27, 3},
		{"tan(0)+sec(0)+cos(0)", 0, 2},
		{"arctan(1)", 0, math.Pi / 4},
		{"arccot(x)", -1, 3 * math.Pi / 4},
		{"cosh(0)+sinh(0)+tanh(0)+sech(0)", 0, 2},
		{"arcsinh(0)+arccosh(1)+arctanh(0)", 0, 0},
		{"nmarbles", 0, float64(TheGraph.Params.NMarbles)},
		{"inf > 1000", 0, 0},
	}
	for _, tt := range tests {
		if tt.expr == "inf > 1000" {
			ex := Expr{Expr: tt.expr}
			require.NoError(t, ex.Compile())
			assert.True(t, ex.EvalBool(0, 0, 0, 0), tt.expr)
			continue
		}
		assert.InDelta(t, tt.want, eval(t, tt.expr, tt.x), 1e-9, tt.expr)
	}
	r := eval(t, "rand", 0)
	assert.True(t, r >= 0 && r < 1, "rand in [0,1)")

	ex := Expr{Expr: "x(t"}
	assert.Error(t, ex.Compile(), "invalid syntax")
	assert.Nil(t, ex.Val)
	ex = Expr{Expr: "x > 1"}
	require.NoError(t, ex.Compile())
	assert.Equal(t, 0.0, ex.Eval(2, 0, 0), "a bool is not a number")
	ex = Expr{Expr: "x + 1"}
	require.NoError(t, ex.Compile())
	assert.False(t, ex.EvalBool(0, 0, 0, 0), "a number is not a bool")
	assert.Equal(t, 0.0, (&Expr{}).Eval(1, 0, 0), "empty")
	assert.True(t, (&Expr{}).EvalBool(1, 0, 0, 0), "empty")
	ex = Expr{Expr: "sqrt(true)"}
	require.NoError(t, ex.Compile())
	assert.Equal(t, 0.0, ex.Eval(0, 0, 0), "wrong argument type")
}

// TestTimeParams tests the time parameters t and a, the y of EvalWithY,
// and the times hit h.
func TestTimeParams(t *testing.T) {
	newGraph(t)
	ex := Expr{Expr: "t+a"}
	require.NoError(t, ex.Compile())
	assert.InDelta(t, 2+10*math.Sin(2), ex.Eval(0, 2, 0), 1e-9)
	ex = Expr{Expr: "y+h"}
	require.NoError(t, ex.Compile())
	assert.Equal(t, 5.0, ex.EvalWithY(0, 0, 2, 3))
}

// TestLineFunctions tests calling other lines as functions: the value,
// the derivatives, integrals, sums, products, and times hit.
func TestLineFunctions(t *testing.T) {
	gr, _ := newGraph(t)
	setLines(gr, "x^2", "f(x)+1", "f'(x)", `f"(x)`, "F(x)", "fint(1, x)", "fsum(1, x)", "fpsum(1, x)", "fh(x)")
	require.NoError(t, gr.State.Error)
	line := func(i int, x float64) float64 {
		return gr.Lines[i].Expr.Eval(x, 0, 0)
	}
	assert.InDelta(t, 10, line(1, 3), 1e-9)
	assert.InDelta(t, 6, line(2, 3), 1e-4, "derivative")
	assert.InDelta(t, 2, line(3, 3), 1e-3, "second derivative")
	assert.InDelta(t, 9, line(4, 3), 1e-6, "integral from 0")
	assert.InDelta(t, 26.0/3, line(5, 3), 1e-6, "integral from 1")
	assert.InDelta(t, -26.0/3, gr.Lines[0].Expr.Integrate(3, 1, 0), 1e-6, "reversed")
	assert.Equal(t, 0.0, gr.Lines[0].Expr.Integrate(1, 1, 0))
	assert.InDelta(t, 14, line(6, 3), 1e-9, "1+4+9")
	assert.InDelta(t, 36, line(7, 3), 1e-9, "1*4*9")
	gr.Lines[0].TimesHit = 2
	assert.InDelta(t, 6, line(8, 3), 1e-9, "times hit * x")

	assert.Error(t, CheckArgs("f", []any{1.0, 2.0}, "float64"))
	assert.Error(t, CheckArgs("f", []any{true}, "float64"))
	for _, name := range []string{"f", "f'", `f"`, "F", "fh"} {
		_, err := gr.Functions[name](true)
		assert.Error(t, err, name)
	}
	for _, name := range []string{"fint", "fsum", "fpsum"} {
		_, err := gr.Functions[name](1.0)
		assert.Error(t, err, name)
	}
}

// TestFunctionArgs tests the errors of the wrappers of the functions.
func TestFunctionArgs(t *testing.T) {
	_, err := NewFunc0(func() float64 { return 0 })(1.0)
	assert.Error(t, err)
	_, err = NewFunc1(math.Sin)()
	assert.Error(t, err)
	_, err = NewFunc2(math.Pow)(1.0)
	assert.Error(t, err)
	_, err = NewFunc2(math.Pow)(true, 1.0)
	assert.Error(t, err)
	_, err = NewFunc2(math.Pow)(1.0, true)
	assert.Error(t, err)
	if3 := DefaultFunctions["if"]
	_, err = if3(true)
	assert.Error(t, err)
	_, err = if3(1.0, 1.0, 1.0)
	assert.Error(t, err)
	_, err = NewFuncV(func(v ...float64) float64 { return 0 })(1.0, true)
	assert.Error(t, err)
	for _, name := range []string{"min", "max", "avg"} {
		v, err := DefaultFunctions[name]()
		assert.NoError(t, err)
		assert.Equal(t, 0, v, name)
	}
}

// TestLineNamesInFunctions tests that the names of the lines (f, g, ...)
// within the names of the functions, as in floor, and the letters of the
// time parameters within them, as in tan, are not taken as references.
func TestLineNamesInFunctions(t *testing.T) {
	gr, _ := newGraph(t)
	setLines(gr, "floor(x)", "tan(x)")
	assert.NoError(t, gr.State.Error)
	assert.False(t, CheckCircular("floor(x)", 0), "floor is not f")
	assert.True(t, CheckCircular("f(x)", 0))
	assert.False(t, gr.Lines[1].Changes, "tan does not change over time")
	assert.True(t, CheckIfChanges("x+t"))
	assert.False(t, CheckIfChanges("g(x)+1"), "g does not change")
}

// TestMarbles tests running the marbles: they fall with gravity and bounce
// off a line, which changes their color and counts the hit.
func TestMarbles(t *testing.T) {
	gr, b := newGraph(t)
	gr.Params.NMarbles = 3
	gr.Params.MarbleStartX.Expr = "n-1"
	gr.Params.MarbleStartY.Expr = "5"
	setLines(gr, "0")
	require.Len(t, gr.Marbles, 3)
	assert.Equal(t, math32.Vec2(-1, 5), gr.Marbles[0].Pos)
	assert.Equal(t, math32.Vec2(1, 5), gr.Marbles[2].Pos)
	ln := gr.Lines[0]
	ln.Colors.ColorSwitch = colors.Red
	minY := float32(5)
	for range 200 {
		gr.Step()
		minY = min(minY, gr.Marbles[1].Pos.Y)
	}
	assert.Greater(t, ln.TimesHit, 0, "the line is hit")
	assert.Greater(t, minY, float32(-0.5), "the marble bounces off the line")
	assert.Equal(t, ln.Colors.ColorSwitch, gr.Marbles[1].Color)
	assert.Greater(t, gr.State.Time, 0.0)

	gr.Run()
	assert.True(t, gr.State.Running)
	gr.Run() // again: no change
	step := gr.State.Time
	gr.Step() // not while running
	assert.Equal(t, step, gr.State.Time)
	gr.RunTick(16)
	assert.Equal(t, 1, gr.State.Step)
	assert.Greater(t, gr.State.Time, step)
	gr.Graph() // stops and resets
	assert.False(t, gr.State.Running)
	assert.Equal(t, 0.0, gr.State.Time)
	gr.RunTick(16) // not running
	assert.Equal(t, 0.0, gr.State.Time)
	gr.State.Running = true
	gr.State.Error = errors.New("error")
	gr.RunTick(16)
	assert.False(t, gr.State.Running, "an error stops it")
	core.RenderImage(b)
}

// TestSelectAndTrack tests selecting marbles in bounds, and tracking their
// positions.
func TestSelectAndTrack(t *testing.T) {
	gr, b := newGraph(t)
	gr.Params.NMarbles = 3
	gr.Params.MarbleStartX.Expr = "20n-20" // -20, 0, 20: only 0 is in bounds
	gr.Params.MarbleStartY.Expr = "5"
	gr.Params.TrackingSettings.Accuracy = 100
	gr.Params.TrackingSettings.NTrackingFrames = 3
	setLines(gr, "-100")
	assert.Equal(t, -1, gr.State.SelectedMarble)
	gr.TrackSelectedMarble() // nothing selected
	gr.SelectNextMarble()
	assert.Equal(t, 1, gr.State.SelectedMarble, "the one in bounds")
	gr.SelectNextMarble()
	assert.Equal(t, 1, gr.State.SelectedMarble, "wraps around to the one in bounds")
	gr.TrackSelectedMarble()
	m := gr.Marbles[1]
	assert.True(t, m.TrackingInfo.Track)
	for range 20 {
		gr.Step()
		gr.State.Step++
	}
	assert.Greater(t, len(m.TrackingInfo.History), 1)
	assert.LessOrEqual(t, len(m.TrackingInfo.History), 11)
	core.RenderImage(b)
	gr.StopSelecting()
	assert.Equal(t, -1, gr.State.SelectedMarble)

	for _, m := range gr.Marbles {
		m.Pos.X = 100
	}
	gr.State.SelectedMarble = -1
	gr.SelectNextMarble() // none in bounds
	assert.Equal(t, 0, gr.State.SelectedMarble)
}

// TestOpenSave tests saving a graph and opening it, the autosave, adding
// lines, and resetting the graph.
func TestOpenSave(t *testing.T) {
	gr, _ := newGraph(t)
	setLines(gr, "x", "2x")
	gr.AddLine()
	assert.Len(t, gr.Lines, 3)
	fn := core.Filename(filepath.Join(t.TempDir(), "graph.json"))
	require.NoError(t, gr.SaveJSON(fn))
	assert.Equal(t, fn, gr.State.File)
	gr.SaveLast()
	TheSettings.PrettyJSON = true
	require.NoError(t, gr.SaveJSON(fn))
	require.NoError(t, gr.AutoSave())

	gr.Reset()
	assert.Len(t, gr.Lines, 1)
	assert.Equal(t, core.Filename(""), gr.State.File)
	gr.SaveLast() // no file

	require.NoError(t, gr.OpenJSON(fn))
	require.Len(t, gr.Lines, 3)
	assert.Equal(t, "2x", gr.Lines[1].Expr.Expr)
	assert.Error(t, gr.OpenJSON(core.Filename(filepath.Join(t.TempDir(), "none.json"))))

	gr.Reset()
	require.NoError(t, gr.OpenAutoSave())
	assert.Len(t, gr.Lines, 3, "the autosave")
	TheSettings.PrettyJSON = false
	require.NoError(t, gr.AutoSave())
	t.Setenv("HOME", t.TempDir())
	assert.NoError(t, gr.OpenAutoSave(), "no autosave")
}

// TestGraphFiles tests opening the example graphs.
func TestGraphFiles(t *testing.T) {
	gr, b := newGraph(t)
	files, err := filepath.Glob("graphs/*.json")
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, f := range files {
		require.NoError(t, gr.OpenJSON(core.Filename(f)), f)
		assert.NoError(t, gr.State.Error, f)
		for range 5 {
			gr.Step()
		}
	}
	core.RenderImage(b)
}

// TestComplete tests completing the names of the functions in expressions.
func TestComplete(t *testing.T) {
	gr, _ := newGraph(t)
	gr.Graph()
	md := ExprComplete(nil, "2+arcs", 0, 0)
	assert.Equal(t, "arcs", md.Seed)
	var names []string
	for _, m := range md.Matches {
		names = append(names, m.Text)
	}
	assert.Contains(t, names, "arcsin")
	assert.Contains(t, names, "arcsinh")
	assert.NotContains(t, names, "sin")
	md = ExprComplete(nil, "xsq", 0, 0)
	assert.Equal(t, "sq", md.Seed, "x is a variable")
	ed := ExprCompleteEdit(nil, "xsq", 3, md.Matches[0], "sq")
	assert.Equal(t, "sqrt", ed.NewText)
}

// TestInvalidLine tests that a line with invalid syntax or circular logic
// is an error of the graph, which draws without it.
func TestInvalidLine(t *testing.T) {
	gr, b := newGraph(t)
	setLines(gr, "x(", "x")
	core.RenderImage(b)
	gr.Step()

	gr, b = newGraph(t)
	setLines(gr, "g(x)", "f(x)")
	assert.Error(t, gr.State.Error, "circular")
	core.RenderImage(b)
	gr.Step()
}
