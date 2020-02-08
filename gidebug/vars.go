// Copyright (c) 2020, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidebug

import (
	"fmt"
	"sort"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/ki/indent"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/syms"
)

// Variable describes a variable.  It is a Ki tree type so that full tree
// can be visualized.
type Variable struct {
	ki.Node
	Value       string               `inactive:"-" width:"60" desc:"value of variable -- may be truncated if long"`
	TypeStr     string               `inactive:"-" desc:"type of variable as a string expression (shortened for display)"`
	FullTypeStr string               `view:"-" inactive:"-" desc:"type of variable as a string expression (full length)"`
	Kind        syms.Kinds           `inactive:"-" desc:"kind of element"`
	ElValue     string               `inactive:"-" view:"-" desc:"own elemental value of variable (blank for composite types)"`
	Len         int64                `inactive:"-" desc:"length of variable (slices, maps, strings etc)"`
	Cap         int64                `inactive:"-" tableview:"-" desc:"capacity of vaiable"`
	Addr        uintptr              `inactive:"-" desc:"address where variable is located in memory"`
	Heap        bool                 `inactive:"-" desc:"if true, the variable is stored in the main memory heap, not the stack"`
	Loc         Location             `inactive:"-" tableview:"-" desc:"location where the variable was defined in source"`
	List        []string             `tableview:"-" desc:"if kind is a list type (array, slice), and elements are primitive types, this is the contents"`
	Map         map[string]string    `tableview:"-" desc:"if kind is a map, and elements are primitive types, this is the contents"`
	MapVar      map[string]*Variable `tableview:"-" desc:"if kind is a map, and elements are not primitive types, this is the contents"`
}

var KiT_Variable = kit.Types.AddType(&Variable{}, VariableProps)

var VariableProps = ki.Props{
	"EnumType:Flag": gi.KiT_NodeFlags,
	"StructViewFields": ki.Props{ // hide in view
		"UniqueNm": `view:"-"`,
		"Flag":     `view:"-"`,
		"Kids":     `view:"-"`,
		"Props":    `view:"-"`,
	},
}

// SortVars sorts vars by name
func SortVars(vrs []*Variable) {
	sort.Slice(vrs, func(i, j int) bool {
		return vrs[i].Nm < vrs[j].Nm
	})
}

// ValueString returns the value of the variable, integrating over sub-elements
// if newlines, each element is separated by a new line, and indented.
// Generally this should be used to set the Value field after getting new data.
// The maxdepth and maxlen parameters provide constraints on the detail
// provided by this string.  outType indicates whether to output type name
func (vr *Variable) ValueString(newlines bool, ident int, maxdepth, maxlen int, outType bool) string {
	if vr.Value != "" {
		return vr.Value
	}
	if vr.ElValue != "" {
		return vr.ElValue
	}
	nkids := len(vr.Kids)
	if vr.Kind.IsPtr() && nkids == 1 {
		return "*" + (vr.Kids[0].(*Variable)).ValueString(newlines, ident, maxdepth, maxlen, true)
	}
	tabSz := 2
	ichr := indent.Space
	var b strings.Builder
	if outType {
		b.WriteString(vr.TypeStr)
	}
	b.WriteString(" {")
	if ident > maxdepth {
		b.WriteString("...")
		if newlines {
			b.WriteString("\n")
			b.WriteString(indent.String(ichr, ident, tabSz))
		}
		b.WriteString("}")
		return b.String()
	}
	lln := len(vr.List)
	if lln > 0 {
		for i, el := range vr.List {
			b.WriteString(fmt.Sprintf("%d: %s", i, el))
			if i < lln-1 {
				b.WriteString(", ")
			}
		}
	}
	lln = len(vr.Map)
	if lln > 0 {
		for k, v := range vr.Map {
			b.WriteString(fmt.Sprintf("%s: %s, ", k, v))
		}
	}
	lln = len(vr.MapVar)
	if lln > 0 {
		for k, ve := range vr.MapVar {
			if newlines {
				b.WriteString("\n")
				b.WriteString(indent.String(ichr, ident+1, tabSz))
			}
			b.WriteString(k + ": ")
			b.WriteString(ve.ValueString(newlines, ident+1, maxdepth, maxlen, false))
			if b.Len() > maxlen {
				b.WriteString("...")
				break
			} else if !newlines {
				b.WriteString(", ")
			}
		}
	}
	for _, vek := range vr.Kids {
		ve := vek.(*Variable)
		if newlines {
			b.WriteString("\n")
			b.WriteString(indent.String(ichr, ident+1, tabSz))
		}
		if ve.Nm != "" {
			b.WriteString(ve.Nm + ": ")
		}
		b.WriteString(ve.ValueString(newlines, ident+1, maxdepth, maxlen, true))
		if b.Len() > maxlen {
			b.WriteString("...")
			break
		} else if !newlines {
			b.WriteString(", ")
		}
	}
	if newlines {
		b.WriteString("\n")
		b.WriteString(indent.String(ichr, ident, tabSz))
	}
	b.WriteString("}")
	return b.String()
}

// TypeInfo returns a string of type information -- if newlines, then
// include newlines between each item (else tabs)
func (vr *Variable) TypeInfo(newlines bool) string {
	sep := "\t"
	if newlines {
		sep = "\n"
	}
	info := []string{"Name: " + vr.Nm, "Type: " + vr.TypeStr, fmt.Sprintf("Len:  %d", vr.Len), fmt.Sprintf("Cap:  %d", vr.Cap), fmt.Sprintf("Addr: %x", vr.Addr), fmt.Sprintf("Heap: %v", vr.Heap)}
	return strings.Join(info, sep)
}
