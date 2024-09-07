package main

import (
	"fmt"
	"math"

	"github.com/Knetic/govaluate"
	"gonum.org/v1/gonum/integrate"
)

// Expr is an expression
type Expr struct {
	// Equation: use x for the x value, t for the time passed since the marbles were ran (incremented by TimeStep), and a for 10*sin(t) (swinging back and forth version of t)
	Expr string `width:"30" label:""`

	Val *govaluate.EvaluableExpression `display:"-" json:"-"`

	Params map[string]any `display:"-" json:"-"`
}

// Integrate returns the integral of an expression
func (ex *Expr) Integrate(min, max float64, h int) float64 {
	var vals []float64
	sign := float64(1)
	diff := max - min
	if diff == 0 {
		return 0
	}
	if diff < 0 {
		diff = -diff
		sign = -1
		min, max = max, min
	}
	accuracy := 16
	dx := diff / float64(accuracy)
	for x := min; x <= max; x += dx {
		vals = append(vals, ex.Eval(x, TheGraph.State.Time, h))
	}
	if len(vals) != accuracy+1 {
		vals = append(vals, ex.Eval(max, TheGraph.State.Time, h))
	}
	val := integrate.Romberg(vals, dx)
	return sign * val
}

// Compile gets an expression ready for evaluation.
func (ex *Expr) Compile() error {
	expr, functions := ex.PrepareExpr(TheGraph.Functions)
	var err error
	ex.Val, err = govaluate.NewEvaluableExpressionWithFunctions(expr, functions)
	if HandleError(err) {
		ex.Val = nil
		return err
	}
	if ex.Params == nil {
		ex.Params = make(map[string]any, 2)
	}
	ex.Params["π"] = math.Pi
	ex.Params["e"] = math.E
	return err
}

// Eval corees the y value of the function for given x, t and h value
func (ex *Expr) Eval(x, t float64, h int) float64 {
	if ex.Expr == "" {
		return 0
	}
	ex.Params["x"] = x
	ex.Params["t"] = t
	ex.Params["a"] = 10 * math.Sin(t)
	ex.Params["h"] = h
	yi, err := ex.Val.Evaluate(ex.Params)
	if HandleError(err) {
		return 0
	}
	switch y := yi.(type) {
	case float64:
		return y
	default:
		TheGraph.Stop()
		HandleError(fmt.Errorf("expression %v is invalid, it is a %T value, should be a float64 value", ex.Expr, yi))
		return 0
	}
}

// EvalWithY calls eval but with a y value set
func (ex *Expr) EvalWithY(x, t float64, h int, y float64) float64 {
	ex.Params["y"] = y
	return ex.Eval(x, t, h)
}

// EvalBool checks if a statement is true based on the x, y, t and h values
func (ex *Expr) EvalBool(x, y, t float64, h int) bool {
	if ex.Expr == "" {
		return true
	}
	ex.Params["x"] = x
	ex.Params["t"] = t
	ex.Params["a"] = 10 * math.Sin(t)
	ex.Params["h"] = h
	ex.Params["y"] = y
	ri, err := ex.Val.Evaluate(ex.Params)
	if HandleError(err) {
		return true
	}
	switch r := ri.(type) {
	case bool:
		return r
	default:
		TheGraph.Stop()
		HandleError(fmt.Errorf("expression %v is invalid, it is a %T value, should be a bool value", ex.Expr, ri))
		return false
	}
}
