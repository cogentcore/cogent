// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cdebug

import (
	"fmt"
	"sort"
	"strings"

	"cogentcore.org/core/base/indent"
	"cogentcore.org/core/parse/syms"
	"cogentcore.org/core/tree"
)

// Variable describes a variable.  It is a tree type so that full tree
// can be visualized.
type Variable struct {
	tree.NodeBase

	// value of variable -- may be truncated if long
	Value string `edit:"-" width:"60"`

	// type of variable as a string expression (shortened for display)
	TypeStr string `edit:"-"`

	// type of variable as a string expression (full length)
	FullTypeStr string `view:"-" edit:"-"`

	// kind of element
	Kind syms.Kinds `edit:"-"`

	// own elemental value of variable (blank for composite types)
	ElementValue string `edit:"-" view:"-"`

	// length of variable (slices, maps, strings etc)
	Len int64 `edit:"-"`

	// capacity of vaiable
	Cap int64 `edit:"-" table:"-"`

	// address where variable is located in memory
	Addr uintptr `edit:"-"`

	// if true, the variable is stored in the main memory heap, not the stack
	Heap bool `edit:"-"`

	// location where the variable was defined in source
	Loc Location `edit:"-" table:"-"`

	// if kind is a list type (array, slice), and elements are primitive types, this is the contents
	List []string `table:"-"`

	// if kind is a map, and elements are primitive types, this is the contents
	Map map[string]string `table:"-"`

	// if kind is a map, and elements are not primitive types, this is the contents
	MapVar map[string]*Variable `table:"-"`

	// our debugger -- for getting further variable data
	Dbg GiDebug `view:"-"`
}

// SortVars sorts vars by name
func SortVars(vrs []*Variable) {
	sort.Slice(vrs, func(i, j int) bool {
		return vrs[i].Name < vrs[j].Name
	})
}

// Label satisfies the core.Labeler interface for showing name = value
func (vr *Variable) Label() string {
	val := vr.Value
	sz := len(vr.Value)
	if sz == 0 {
		return vr.Name
	}
	if sz > 40 {
		val = val[:40] + "..."
	}
	return vr.Name + " = " + val
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
	if vr.ElementValue != "" {
		return vr.ElementValue
	}
	nkids := len(vr.Children)
	if vr.Kind.IsPtr() && nkids == 1 {
		return "*" + (vr.Children[0].(*Variable)).ValueString(newlines, ident, maxdepth, maxlen, true)
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
	for _, vek := range vr.Children {
		ve := vek.(*Variable)
		if newlines {
			b.WriteString("\n")
			b.WriteString(indent.String(ichr, ident+1, tabSz))
		}
		if ve.Name != "" {
			b.WriteString(ve.Name + ": ")
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
	info := []string{"Name: " + vr.Name, "Type: " + vr.TypeStr, fmt.Sprintf("Len:  %d", vr.Len), fmt.Sprintf("Cap:  %d", vr.Cap), fmt.Sprintf("Addr: %x", vr.Addr), fmt.Sprintf("Heap: %v", vr.Heap)}
	return strings.Join(info, sep)
}

// FollowPtr retrieves the contents of this pointer and adds it as a child.
func (vr *Variable) FollowPtr() {
	if vr.Dbg == nil {
		return
	}
	vr.Dbg.FollowPtr(vr)
}

// VarParams are parameters controlling how much detail the debugger reports
// about variables.
type VarParams struct {

	// requests pointers to be automatically dereferenced -- this can be very dangerous in terms of size of variable data returned and is not recommended.
	FollowPointers bool `default:"false"`

	// how far to recurse when evaluating nested types.
	MaxRecurse int

	// the maximum number of bytes read from a string
	MaxStringLen int

	// the maximum number of elements read from an array, a slice or a map.
	MaxArrayValues int

	// the maximum number of fields read from a struct, -1 will read all fields.
	MaxStructFields int
}

// Params are overall debugger parameters
type Params struct { //types:add

	// mode for running the debugger
	Mode Modes `xml:"-" toml:"-" json:"-" view:"-"`

	// process id number to attach to, for Attach mode
	PID uint64 `xml:"-" toml:"-" json:"-" view:"-"`

	// optional extra args to pass to the debugger.
	// Use -- double-dash and then add args to pass args to the executable
	// (double-dash is by itself as a separate arg first).
	// For Debug test, must use -test.run instead of plain -run to specify tests to run.
	Args []string

	// status function for debugger updating status
	StatFunc func(stat Status) `xml:"-" toml:"-" json:"-" view:"-"`

	// parameters for level of detail on overall list of variables
	VarList VarParams

	// parameters for level of detail retrieving a specific variable
	GetVar VarParams
}

// DefaultParams are default parameter values
var DefaultParams = Params{
	VarList: VarParams{
		FollowPointers:  false,
		MaxRecurse:      4,
		MaxStringLen:    100,
		MaxArrayValues:  10,
		MaxStructFields: -1,
	},
	GetVar: VarParams{
		FollowPointers:  false,
		MaxRecurse:      10,
		MaxStringLen:    1024,
		MaxArrayValues:  1024,
		MaxStructFields: -1,
	},
}
