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

	// value of variable -- may be truncated if long
	Value string `inactive:"-" width:"60" desc:"value of variable -- may be truncated if long"`

	// type of variable as a string expression (shortened for display)
	TypeStr string `inactive:"-" desc:"type of variable as a string expression (shortened for display)"`

	// type of variable as a string expression (full length)
	FullTypeStr string `view:"-" inactive:"-" desc:"type of variable as a string expression (full length)"`

	// kind of element
	Kind syms.Kinds `inactive:"-" desc:"kind of element"`

	// own elemental value of variable (blank for composite types)
	ElValue string `inactive:"-" view:"-" desc:"own elemental value of variable (blank for composite types)"`

	// length of variable (slices, maps, strings etc)
	Len int64 `inactive:"-" desc:"length of variable (slices, maps, strings etc)"`

	// capacity of vaiable
	Cap int64 `inactive:"-" tableview:"-" desc:"capacity of vaiable"`

	// address where variable is located in memory
	Addr uintptr `inactive:"-" desc:"address where variable is located in memory"`

	// if true, the variable is stored in the main memory heap, not the stack
	Heap bool `inactive:"-" desc:"if true, the variable is stored in the main memory heap, not the stack"`

	// location where the variable was defined in source
	Loc Location `inactive:"-" tableview:"-" desc:"location where the variable was defined in source"`

	// if kind is a list type (array, slice), and elements are primitive types, this is the contents
	List []string `tableview:"-" desc:"if kind is a list type (array, slice), and elements are primitive types, this is the contents"`

	// if kind is a map, and elements are primitive types, this is the contents
	Map map[string]string `tableview:"-" desc:"if kind is a map, and elements are primitive types, this is the contents"`

	// if kind is a map, and elements are not primitive types, this is the contents
	MapVar map[string]*Variable `tableview:"-" desc:"if kind is a map, and elements are not primitive types, this is the contents"`

	// our debugger -- for getting further variable data
	Dbg GiDebug `view:"-" desc:"our debugger -- for getting further variable data"`
}

var KiT_Variable = kit.Types.AddType(&Variable{}, nil)

func (vr *Variable) CopyFieldsFrom(frm any) {
	fr := frm.(*Variable)
	vr.Value = fr.Value
	vr.TypeStr = fr.TypeStr
	vr.FullTypeStr = fr.FullTypeStr
	vr.Kind = fr.Kind
	vr.ElValue = fr.ElValue
	vr.Len = fr.Len
	vr.Cap = fr.Cap
	vr.Addr = fr.Addr
	vr.Heap = fr.Heap
	vr.Loc = fr.Loc
	vr.List = fr.List
	vr.Map = fr.Map
	vr.MapVar = fr.MapVar
	vr.Dbg = fr.Dbg
}

func init() {
	kit.Types.SetProps(KiT_Variable, VariableProps)
}

var VariableProps = ki.Props{
	"EnumType:Flag": gi.KiT_NodeFlags,
	"StructViewFields": ki.Props{ // hide in view
		"UniqueNm": `view:"-"`,
		"Flag":     `view:"-"`,
		"Kids":     `view:"-"`,
		"Props":    `view:"-"`,
	},
	"ToolBar": ki.PropSlice{
		{"FollowPtr", ki.Props{
			"desc": "retrieve the contents of this pointer -- child nodes will contain further data",
			"icon": "update",
			"updtfunc": func(vri any, act *gi.Action) {
				vr := vri.(ki.Ki).Embed(KiT_Variable).(*Variable)
				act.SetActiveState(!vr.HasChildren())
			},
		}},
	},
}

// SortVars sorts vars by name
func SortVars(vrs []*Variable) {
	sort.Slice(vrs, func(i, j int) bool {
		return vrs[i].Nm < vrs[j].Nm
	})
}

// Label satisfies the gi.Labeler interface for showing name = value
func (vr *Variable) Label() string {
	val := vr.Value
	sz := len(vr.Value)
	if sz == 0 {
		return vr.Nm
	}
	if sz > 40 {
		val = val[:40] + "..."
	}
	return vr.Nm + " = " + val
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

// FollowPtr retrieves the contents of this pointer and adds it as a child.
func (vr *Variable) FollowPtr() {
	if vr.Dbg == nil {
		return
	}
	updt := vr.UpdateStart()
	vr.Dbg.FollowPtr(vr)
	vr.UpdateEnd(updt)
}

// VarParams are parameters controlling how much detail the debugger reports
// about variables.
type VarParams struct {

	// requests pointers to be automatically dereferenced -- this can be very dangerous in terms of size of variable data returned and is not recommended.
	FollowPointers bool `def:"false" desc:"requests pointers to be automatically dereferenced -- this can be very dangerous in terms of size of variable data returned and is not recommended."`

	// how far to recurse when evaluating nested types.
	MaxRecurse int `desc:"how far to recurse when evaluating nested types."`

	// the maximum number of bytes read from a string
	MaxStringLen int `desc:"the maximum number of bytes read from a string"`

	// the maximum number of elements read from an array, a slice or a map.
	MaxArrayValues int `desc:"the maximum number of elements read from an array, a slice or a map."`

	// the maximum number of fields read from a struct, -1 will read all fields.
	MaxStructFields int `desc:"the maximum number of fields read from a struct, -1 will read all fields."`
}

// Params are overall debugger parameters
type Params struct {

	// mode for running the debugger
	Mode Modes `xml:"-" json:"-" view:"-" desc:"mode for running the debugger"`

	// process id number to attach to, for Attach mode
	PID uint64 `xml:"-" json:"-" view:"-" desc:"process id number to attach to, for Attach mode"`

	// optional extra args to pass to the debugger.  Use double-dash -- and then add args to pass args to the executable (double-dash is by itself as a separate arg first)
	Args []string `desc:"optional extra args to pass to the debugger.  Use double-dash -- and then add args to pass args to the executable (double-dash is by itself as a separate arg first)"`

	// status function for debugger updating status
	StatFunc func(stat Status) `xml:"-" json:"-" view:"-" desc:"status function for debugger updating status"`

	// parameters for level of detail on overall list of variables
	VarList VarParams `desc:"parameters for level of detail on overall list of variables"`

	// parameters for level of detail retrieving a specific variable
	GetVar VarParams `desc:"parameters for level of detail retrieving a specific variable"`
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
