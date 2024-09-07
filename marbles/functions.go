package main

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/Knetic/govaluate"
	"gonum.org/v1/gonum/diff/fd"
)

// Functions are a map of named expression functions
type Functions map[string]govaluate.ExpressionFunction

// NewFuncV makes a function that can be used in expressions from a function that takes a variadic input and returns a single value.
func NewFuncV[I, O any](f func(...I) O) govaluate.ExpressionFunction {
	return func(args ...any) (any, error) {
		newArgs := []I{}
		for i, arg := range args {
			a, ok := arg.(I)
			if !ok {
				return nil, fmt.Errorf("evaluation error: function of type %T does not accept input type %T for argument %v", f, arg, i)
			}
			newArgs = append(newArgs, a)
		}
		res := f(newArgs...)
		return res, nil
	}
}

// NewFunc0 makes a function that can be used in expressions from a function that takes no arguments and returns a single value.
// IMPORTANT: zero arg functions must be added to [ZeroArgFunctions].
func NewFunc0[O any](f func() O) govaluate.ExpressionFunction {
	return func(args ...any) (any, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("evaluation error: function of type %T wants 0 arguments, not %v arguments", f, len(args))
		}
		res := f()
		return res, nil
	}
}

// NewFunc1 makes a function that can be used in expressions from a function that takes a single argument and returns a single value.
func NewFunc1[I, O any](f func(I) O) govaluate.ExpressionFunction {
	return func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("evaluation error: function of type %T wants 1 argument, not %v arguments", f, len(args))
		}
		arg0, ok := args[0].(I)
		if !ok {
			return nil, fmt.Errorf("evaluation error: function of type %T does not accept input type %T", f, args[0])
		}
		res := f(arg0)
		return res, nil
	}
}

// NewFunc2 makes a function that can be used in expressions from a function that takes two arguments and returns a single value.
func NewFunc2[I1, I2, O any](f func(I1, I2) O) govaluate.ExpressionFunction {
	return func(args ...any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("evaluation error: function of type %T wants 2 arguments, not %v arguments", f, len(args))
		}
		arg0, ok := args[0].(I1)
		if !ok {
			return nil, fmt.Errorf("evaluation error: function of type %T does not accept input type %T for argument 0", f, args[0])
		}
		arg1, ok := args[1].(I2)
		if !ok {
			return nil, fmt.Errorf("evaluation error: function of type %T does not accept input type %T for argument 1", f, args[1])
		}
		res := f(arg0, arg1)
		return res, nil
	}
}

// NewFunc3 makes a function that can be used in expressions from a function that takes three arguments and returns a single value.
func NewFunc3[I1, I2, I3, O any](f func(I1, I2, I3) O) govaluate.ExpressionFunction {
	return func(args ...any) (any, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("evaluation error: function of type %T wants 3 arguments, not %v arguments", f, len(args))
		}
		arg0, ok := args[0].(I1)
		if !ok {
			return nil, fmt.Errorf("evaluation error: function of type %T does not accept input type %T for argument 0", f, args[0])
		}
		arg1, ok := args[1].(I2)
		if !ok {
			return nil, fmt.Errorf("evaluation error: function of type %T does not accept input type %T for argument 1", f, args[1])
		}
		arg2, ok := args[2].(I3)
		if !ok {
			return nil, fmt.Errorf("evaluation error: function of type %T does not accept input type %T for argument 2", f, args[2])
		}
		res := f(arg0, arg1, arg2)
		return res, nil
	}
}

// DefaultFunctions are the default functions that can be used in expressions
var DefaultFunctions = Functions{
	"sin": NewFunc1(math.Sin),
	"cos": NewFunc1(math.Cos),
	"tan": NewFunc1(math.Tan),
	"sec": NewFunc1(func(x float64) float64 {
		return 1 / math.Cos(x)
	}),
	"csc": NewFunc1(func(x float64) float64 {
		return 1 / math.Sin(x)
	}),
	"cot": NewFunc1(func(x float64) float64 {
		return 1 / math.Tan(x)
	}),
	"arcsin": NewFunc1(math.Asin),
	"arccos": NewFunc1(math.Acos),
	"arctan": NewFunc1(math.Atan),
	"arcsec": NewFunc1(func(x float64) float64 {
		return math.Acos(1 / x)
	}),
	"arccsc": NewFunc1(func(x float64) float64 {
		return math.Asin(1 / x)
	}),
	"arccot": NewFunc1(func(x float64) float64 {
		y := math.Atan(1 / x)
		if x < 0 {
			y += math.Pi
		}
		return y
	}),
	"sinh": NewFunc1(math.Sinh),
	"cosh": NewFunc1(math.Cosh),
	"tanh": NewFunc1(math.Tanh),
	"sech": NewFunc1(func(x float64) float64 {
		return 1 / math.Cosh(x)
	}),
	"csch": NewFunc1(func(x float64) float64 {
		return 1 / math.Sinh(x)
	}),
	"coth": NewFunc1(func(x float64) float64 {
		return 1 / math.Tanh(x)
	}),
	"arcsinh": NewFunc1(math.Asinh),
	"arccosh": NewFunc1(math.Acosh),
	"arctanh": NewFunc1(math.Atanh),
	"arcsech": NewFunc1(func(x float64) float64 {
		return math.Acosh(1 / x)
	}),
	"arccsch": NewFunc1(func(x float64) float64 {
		return math.Asinh(1 / x)
	}),
	"arccoth": NewFunc1(func(x float64) float64 {
		return math.Atanh(1 / x)
	}),
	"ln": NewFunc1(math.Log),
	"log": NewFunc2(func(x, base float64) float64 {
		return math.Log(x) / math.Log(base)
	}),
	"abs": NewFunc1(math.Abs),
	"pow": NewFunc2(math.Pow),
	"exp": NewFunc1(math.Exp),
	"mod": NewFunc2(math.Mod),
	"fact": NewFunc1(func(x float64) float64 {
		return math.Gamma(x + 1)
	}),
	"floor": NewFunc1(math.Floor),
	"ceil":  NewFunc1(math.Ceil),
	"round": NewFunc1(math.Round),
	"sqrt":  NewFunc1(math.Sqrt),
	"cbrt":  NewFunc1(math.Cbrt),
	"min": NewFuncV(func(v ...float64) any {
		if len(v) == 0 {
			return 0
		}
		min := v[0]
		for i := 1; i < len(v); i++ {
			x := v[i]
			if x < min {
				min = x
			}
		}
		return min
	}),
	"max": NewFuncV(func(v ...float64) any {
		if len(v) == 0 {
			return 0
		}
		max := v[0]
		for i := 1; i < len(v); i++ {
			x := v[i]
			if x > max {
				max = x
			}
		}
		return max
	}),
	"avg": NewFuncV(func(v ...float64) any {
		if len(v) == 0 {
			return 0
		}
		var total float64
		for _, x := range v {
			total += x
		}
		return total / float64(len(v))
	}),
	"if": NewFunc3(func(condition bool, val1, val2 any) any {
		if condition {
			return val1
		}
		return val2
	}),
	// IMPORTANT: zero arg functions must be added to [ZeroArgFunctions].
	"rand": NewFunc0(rand.Float64),
	"nmarbles": NewFunc0(func() float64 {
		return float64(TheGraph.Params.NMarbles)
	}),
	"inf": NewFunc0(func() float64 {
		return math.Inf(1)
	}),
}

// CheckArgs checks if a function is passed the right number of arguments, and the right type of arguments.
func CheckArgs(name string, have []any, want ...string) error {
	if len(have) != len(want) {
		return fmt.Errorf("function %v needs %v arguments, not %v arguments", name, len(want), len(have))
	}
	for i, d := range want {
		if d != fmt.Sprintf("%T", have[i]) {
			return fmt.Errorf("function %v needs %v. %v does not work", name, want, have)
		}
	}
	return nil
}

// SetFunctionsTo sets the functions of the graph to another set of functions
func (gr *Graph) SetFunctionsTo(functions Functions) {
	gr.Functions = make(Functions)
	for k, d := range functions {
		gr.Functions[k] = d
	}
}

// AddLineFunctions adds all of the line functions
func (gr *Graph) AddLineFunctions() {
	for k, ln := range gr.Lines {
		ln.SetFunctionName(k)
	}
}

// SetFunctionName sets the function name for a line and adds the function to the functions
func (ln *Line) SetFunctionName(k int) {
	if k >= len(FunctionNames) {
		// ln.FuncName = "unassigned"
		return
	}
	functionName := FunctionNames[k]
	// ln.FuncName = functionName + "(x)="
	TheGraph.Functions[functionName] = func(args ...any) (any, error) {
		err := CheckArgs(functionName, args, "float64")
		if err != nil {
			return 0, err
		}
		val := float64(ln.Expr.Eval(args[0].(float64), TheGraph.State.Time, ln.TimesHit))
		return val, nil
	}
	TheGraph.Functions[functionName+"'"] = func(args ...any) (any, error) {
		err := CheckArgs(functionName+"'", args, "float64")
		if err != nil {
			return 0, err
		}
		val := fd.Derivative(func(x float64) float64 {
			return ln.Expr.Eval(x, TheGraph.State.Time, ln.TimesHit)
		}, args[0].(float64), &fd.Settings{
			Formula: fd.Central,
		})
		return val, nil
	}
	TheGraph.Functions[functionName+`"`] = func(args ...any) (any, error) {
		err := CheckArgs(functionName+`"`, args, "float64")
		if err != nil {
			return 0, err
		}
		val := fd.Derivative(func(x float64) float64 {
			return ln.Expr.Eval(x, TheGraph.State.Time, ln.TimesHit)
		}, args[0].(float64), &fd.Settings{
			Formula: fd.Central2nd,
		})
		return val, nil
	}
	capitalName := strings.ToUpper(functionName)
	TheGraph.Functions[capitalName] = func(args ...any) (any, error) {
		err := CheckArgs(capitalName, args, "float64")
		if err != nil {
			return 0, err
		}
		val := ln.Expr.Integrate(0, args[0].(float64), ln.TimesHit)
		return val, nil
	}
	TheGraph.Functions[functionName+"int"] = func(args ...any) (any, error) {
		err := CheckArgs(functionName+"int", args, "float64", "float64")
		if err != nil {
			return 0, err
		}
		min := args[0].(float64)
		max := args[1].(float64)
		val := ln.Expr.Integrate(min, max, ln.TimesHit)
		return val, nil
	}
	TheGraph.Functions[functionName+"h"] = func(args ...any) (any, error) {
		err := CheckArgs(functionName+"h", args, "float64")
		if err != nil {
			return 0, err
		}
		return float64(ln.TimesHit) * args[0].(float64), nil
	}
	TheGraph.Functions[functionName+"sum"] = func(args ...any) (any, error) {
		err := CheckArgs(functionName+"sum", args, "float64", "float64")
		if err != nil {
			return 0, err
		}
		total := 0.0
		for i := args[0].(float64); i <= args[1].(float64); i++ {
			total += (ln.Expr.Eval(i, TheGraph.State.Time, ln.TimesHit))
		}
		return total, nil
	}
	TheGraph.Functions[functionName+"psum"] = func(args ...any) (any, error) {
		err := CheckArgs(functionName+"psum", args, "float64", "float64")
		if err != nil {
			return 0, err
		}
		total := 1.0
		for i := args[0].(float64); i <= args[1].(float64); i++ {
			total *= (ln.Expr.Eval(i, TheGraph.State.Time, ln.TimesHit))
		}
		return total, nil
	}
}
