// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cdebug

import (
	"strings"
	"testing"

	"cogentcore.org/core/text/parse/syms"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBreaks tests adding, finding, deleting, and merging breakpoints.
func TestBreaks(t *testing.T) {
	var as AllState
	as.BlankState()
	assert.Equal(t, NotInit, as.Status)
	b1 := as.AddBreak("/p/a.go", 10)
	assert.True(t, b1.On)
	assert.Equal(t, "p/a.go", b1.File)
	b1.On = false
	assert.Same(t, b1, as.AddBreak("/p/a.go", 10), "existing")
	assert.True(t, b1.On, "turned on again")
	b2 := as.AddBreak("/p/b.go", 3)
	b1.ID, b2.ID = 2, 1

	br, i := as.BreakByFile("/p/b.go", 3)
	assert.Same(t, b2, br)
	assert.Equal(t, 1, i)
	_, i = as.BreakByFile("/p/b.go", 4)
	assert.Equal(t, -1, i)
	br, i = as.BreakByID(2)
	assert.Same(t, b1, br)
	assert.Equal(t, 0, i)
	_, i = as.BreakByID(5)
	assert.Equal(t, -1, i)

	// the breaks of the debugger: one updated, one new, one without an ID
	as.CurBreaks = []*Break{{ID: 2, FPath: "/p/a.go", Line: 10, Cond: "x > 1"}, {ID: 3, FPath: "/p/c.go", Line: 1}, {ID: 0}}
	as.MergeBreaks()
	require.Len(t, as.Breaks, 3)
	assert.Equal(t, []int{1, 2, 3}, []int{as.Breaks[0].ID, as.Breaks[1].ID, as.Breaks[2].ID}, "sorted")
	assert.False(t, as.Breaks[0].On, "not in the debugger")
	assert.True(t, as.Breaks[1].On)
	assert.Equal(t, "x > 1", as.Breaks[1].Cond)
	assert.True(t, as.Breaks[2].On)

	assert.True(t, as.DeleteBreakByID(3))
	assert.False(t, as.DeleteBreakByID(3))
	assert.True(t, as.DeleteBreakByFile("/p/b.go", 3))
	assert.False(t, as.DeleteBreakByFile("/p/b.go", 3))
	assert.Len(t, as.Breaks, 1)
}

// TestStateLookups tests finding threads, tasks, frames, and variables.
func TestStateLookups(t *testing.T) {
	thrs := []*Thread{{ID: 1, File: "a.go", Line: 3, Func: "main"}, {ID: 7}}
	th, i := ThreadByID(thrs, 7)
	assert.Same(t, thrs[1], th)
	assert.Equal(t, 1, i)
	_, i = ThreadByID(thrs, 9)
	assert.Equal(t, -1, i)
	assert.Contains(t, thrs[0].String(), "a.go:3")
	tks := []*Task{{ID: 2, File: "b.go", Line: 4, Func: "f"}}
	tk, i := TaskByID(tks, 2)
	assert.Same(t, tks[0], tk)
	assert.Equal(t, 0, i)
	_, i = TaskByID(tks, 3)
	assert.Equal(t, -1, i)
	st := State{Thread: *thrs[0], Task: *tks[0], Running: true}
	assert.Contains(t, st.String(), "Run: true")

	var as AllState
	as.Stack = []*Frame{{Depth: 0}, {Depth: 1}}
	assert.Equal(t, 1, as.StackFrame(1).Depth)
	assert.Nil(t, as.StackFrame(2))
	assert.Nil(t, as.StackFrame(-1))

	newVar := func(name, value string) *Variable {
		v := NewVariable()
		v.SetName(name)
		v.Value = value
		return v
	}
	x, g, h := newVar("x", "1"), newVar("main.g", "2"), newVar("pkg.sub.h", "3")
	as.Vars = []*Variable{x}
	as.GlobalVars = []*Variable{g, h}
	assert.Same(t, x, as.VarByName("x"))
	assert.Same(t, g, as.VarByName("g"), "without the package")
	assert.Same(t, g, as.VarByName("main.g"))
	assert.Same(t, h, as.VarByName("pkg.sub.h"))
	assert.Nil(t, as.VarByName("none"))

	vrs := []*Variable{h, x, g}
	SortVars(vrs)
	assert.Equal(t, []*Variable{g, h, x}, vrs)
}

// TestVariableStrings tests the strings of variables, with their values
// and elements.
func TestVariableStrings(t *testing.T) {
	newVar := func(name, typ string) *Variable {
		v := NewVariable()
		v.SetName(name)
		v.TypeStr = typ
		return v
	}
	v := newVar("v", "int")
	assert.Equal(t, "v", v.Label())
	v.Value = "3"
	assert.Equal(t, "v = 3", v.Label())
	v.Value = strings.Repeat("x", 50)
	assert.Equal(t, "v = "+strings.Repeat("x", 40)+"...", v.Label())
	v.Value = ""
	v.ElementValue = "4"
	assert.Equal(t, "4", v.ValueString(false, 0, 3, 100, true))

	assert.Contains(t, v.TypeInfo(true), "Name: v\nType: int")
	assert.Contains(t, v.TypeInfo(false), "Name: v\tType: int")

	// a pointer to a struct with fields
	s := newVar("s", "T")
	f := newVar("f", "int")
	f.Value = "1"
	tree.MoveToParent(f, s)
	l := newVar("l", "[]int")
	l.List = []string{"1", "2"}
	tree.MoveToParent(l, s)
	m := newVar("m", "map[string]int")
	m.Map = map[string]string{"a": "1"}
	tree.MoveToParent(m, s)
	mv := newVar("mv", "map[string]T")
	e := newVar("", "T")
	e.Value = "{}"
	mv.MapVar = map[string]*Variable{"k": e}
	tree.MoveToParent(mv, s)
	p := newVar("p", "*T")
	p.Kind = syms.Ptr
	tree.MoveToParent(s, p)
	assert.Equal(t, "*T {f: 1, l: []int {0: 1, 1: 2}, m: map[string]int {a: 1, }, mv: map[string]T {k: {}, }, }", p.ValueString(false, 0, 3, 1000, true))
	nl := p.ValueString(true, 0, 3, 1000, true)
	assert.Contains(t, nl, "\n  f: 1")
	assert.Contains(t, p.ValueString(false, 0, 0, 1000, true), "{...}", "beyond the depth")
	assert.Contains(t, p.ValueString(false, 0, 3, 10, true), "...", "beyond the length")
	p.FollowPtr() // no debugger
}
