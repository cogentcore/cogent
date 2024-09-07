package main

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/Knetic/govaluate"
)

// EquationChange type has the string that needs to be replaced and what to replace it with
type EquationChange struct {
	Old string
	New string
}

// UnreadableChangeSlice is all of the strings that should change before compiling, but the user shouldn't see
var UnreadableChangeSlice = []EquationChange{
	{"^", "**"},
	{"√", "sqrt"},
	{"∞", "inf()"},
	{"∫", "int"},
	{"∏", "psum"},
	{"Σ", "sum"},
	{")(", ")*("},
}

// EquationChangeSlice is all of the strings that should be changed
var EquationChangeSlice = []EquationChange{
	{"''", `"`},
	{"**", "^"},
	{"sqrt", "√"},
	{"pi", "π"},
	{"inf", "∞"},
	{"int", "∫"},
	{"psum", "∏"},
	{"sum", "Σ"},
	{`\`, ""},
}

// ZeroArgFunctions are the functions that do not take any arguments.
var ZeroArgFunctions = []string{"rand", "nmarbles", "inf"}

// PrepareExpr prepares an expression by looping both equation change slices
func (ex *Expr) PrepareExpr(functionsArg map[string]govaluate.ExpressionFunction) (string, map[string]govaluate.ExpressionFunction) {
	functions := make(map[string]govaluate.ExpressionFunction)
	for name, function := range functionsArg {
		functions[name] = function
	}
	ex.LoopEquationChangeSlice()
	params := []string{"π", "e", "x", "a", "t", "h", "y", "n"}
	symbols := []string{"+", "-", "*", "/", "^", ">", "<", "=", "(", ")"}
	expr := LoopUnreadableChangeSlice(ex.Expr)
	expr = strings.ReplaceAll(expr, "true", "(0==0)") // prevent true and false from being interpreted as functions
	expr = strings.ReplaceAll(expr, "false", "(0!=0)")
	for _, s := range symbols {
		expr = strings.ReplaceAll(expr, s+"-", s+" -")
		expr = strings.ReplaceAll(expr, s+".", s+"0.")
	}
	for _, p := range params {
		expr = strings.ReplaceAll(expr, strings.ToUpper(p), p)
	}
	i := 0
	functionsToDelete := []string{}
	functionsToAdd := make(map[string]govaluate.ExpressionFunction)
	// sort functions by length so that functions that contain other functions don't cause problems
	functionKeys := []string{}
	for k := range functions {
		functionKeys = append(functionKeys, k)
	}
	slices.SortFunc(functionKeys, func(a, b string) int {
		return cmp.Compare(len(b), len(a))
	})
	isZeroArg := map[string]bool{}
	for _, name := range functionKeys { // to prevent issues with the equation, all functions are turned into zfunctionindexz. z is just a letter that isn't used in anything else.
		function := functions[name]
		newName := fmt.Sprintf("z%vz", i)
		expr = strings.ReplaceAll(expr, name, newName)
		functionsToAdd[newName] = function
		functionsToDelete = append(functionsToDelete, name)
		isZeroArg[newName] = slices.Contains(ZeroArgFunctions, name)
		i++
	}
	for name, function := range functionsToAdd {
		functions[name] = function
	}
	for _, name := range functionsToDelete {
		delete(functions, name)
	}
	for fname := range functions { // if there is a function name and no parentheses after, put parentheses around the next character, or directly after it if it is a zero-arg function
		if isZeroArg[fname] {
			expr = strings.ReplaceAll(expr, fname, fname+"()")
			expr = strings.ReplaceAll(expr, fname+"()()", fname+"()")
		}
		for _, pname := range params {
			expr = strings.ReplaceAll(expr, fname+pname, fname+"("+pname+")")
		}
		for n := 0; n < 10; n++ {
			ns := strconv.Itoa(n)
			expr = strings.ReplaceAll(expr, fname+ns, fname+"("+ns+")")
		}
	}
	for n := 0; n < 10; n++ { // if the expression contains a number and then a parameter or a function right after, then change it to multiply the number and the parameter/function. Also ()number changes to ()*number
		ns := strconv.Itoa(n)
		for _, pname := range params {
			expr = strings.ReplaceAll(expr, ns+pname, ns+"*"+pname)
			expr = strings.ReplaceAll(expr, ns+"("+pname, ns+"*("+pname)
		}
		for fname := range functions {
			expr = strings.ReplaceAll(expr, ns+fname, ns+"*"+fname)
			expr = strings.ReplaceAll(expr, ns+"("+fname, ns+"*("+fname)
		}
		expr = strings.ReplaceAll(expr, ")"+ns, ")*"+ns)
	}
	for _, pname := range params { // if the expression contains a parameter before another parameter or a function, make it multiply. Also ()parameter changes to ()*parameter
		for _, pname1 := range params {
			for strings.Contains(expr, pname+pname1) || strings.Contains(expr, pname+"("+pname1) {
				expr = strings.ReplaceAll(expr, pname+pname1, pname+"*"+pname1)
				expr = strings.ReplaceAll(expr, pname+"("+pname1, pname+"*("+pname1)
			}
		}
		for fname := range functions {
			expr = strings.ReplaceAll(expr, pname+fname, pname+"*"+fname)
			expr = strings.ReplaceAll(expr, pname+"("+fname, pname+"*("+fname)
		}
		expr = strings.ReplaceAll(expr, ")"+pname, ")*"+pname)
		expr = strings.ReplaceAll(expr, pname+"(", pname+"*(")
	}
	for fname := range functions { // replace ()fname() with ()*fname()
		expr = strings.ReplaceAll(expr, ")"+fname, ")*"+fname)
	}

	return expr, functions
}

// LoopEquationChangeSlice loops over the Equation Change slice and makes the replacements
func (ex *Expr) LoopEquationChangeSlice() {
	for _, d := range EquationChangeSlice {
		ex.Expr = strings.ReplaceAll(ex.Expr, d.Old, d.New)
	}
}

// LoopUnreadableChangeSlice loops over the unreadable Change slice and makes the replacements
func LoopUnreadableChangeSlice(expr string) string {
	for _, d := range UnreadableChangeSlice {
		expr = strings.ReplaceAll(expr, d.Old, d.New)
	}
	return expr
}
