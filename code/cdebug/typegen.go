// Code generated by "core generate"; DO NOT EDIT.

package cdebug

import (
	"cogentcore.org/core/parse/syms"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code/cdebug.Variable", IDName: "variable", Doc: "Variable describes a variable.  It is a tree type so that full tree\ncan be visualized.", Embeds: []types.Field{{Name: "NodeBase"}}, Fields: []types.Field{{Name: "Value", Doc: "value of variable -- may be truncated if long"}, {Name: "TypeStr", Doc: "type of variable as a string expression (shortened for display)"}, {Name: "FullTypeStr", Doc: "type of variable as a string expression (full length)"}, {Name: "Kind", Doc: "kind of element"}, {Name: "ElementValue", Doc: "own elemental value of variable (blank for composite types)"}, {Name: "Len", Doc: "length of variable (slices, maps, strings etc)"}, {Name: "Cap", Doc: "capacity of vaiable"}, {Name: "Addr", Doc: "address where variable is located in memory"}, {Name: "Heap", Doc: "if true, the variable is stored in the main memory heap, not the stack"}, {Name: "Loc", Doc: "location where the variable was defined in source"}, {Name: "List", Doc: "if kind is a list type (array, slice), and elements are primitive types, this is the contents"}, {Name: "Map", Doc: "if kind is a map, and elements are primitive types, this is the contents"}, {Name: "MapVar", Doc: "if kind is a map, and elements are not primitive types, this is the contents"}, {Name: "Dbg", Doc: "our debugger -- for getting further variable data"}}})

// NewVariable returns a new [Variable] with the given optional parent:
// Variable describes a variable.  It is a tree type so that full tree
// can be visualized.
func NewVariable(parent ...tree.Node) *Variable { return tree.New[Variable](parent...) }

// SetValue sets the [Variable.Value]:
// value of variable -- may be truncated if long
func (t *Variable) SetValue(v string) *Variable { t.Value = v; return t }

// SetTypeStr sets the [Variable.TypeStr]:
// type of variable as a string expression (shortened for display)
func (t *Variable) SetTypeStr(v string) *Variable { t.TypeStr = v; return t }

// SetFullTypeStr sets the [Variable.FullTypeStr]:
// type of variable as a string expression (full length)
func (t *Variable) SetFullTypeStr(v string) *Variable { t.FullTypeStr = v; return t }

// SetKind sets the [Variable.Kind]:
// kind of element
func (t *Variable) SetKind(v syms.Kinds) *Variable { t.Kind = v; return t }

// SetElementValue sets the [Variable.ElementValue]:
// own elemental value of variable (blank for composite types)
func (t *Variable) SetElementValue(v string) *Variable { t.ElementValue = v; return t }

// SetLen sets the [Variable.Len]:
// length of variable (slices, maps, strings etc)
func (t *Variable) SetLen(v int64) *Variable { t.Len = v; return t }

// SetCap sets the [Variable.Cap]:
// capacity of vaiable
func (t *Variable) SetCap(v int64) *Variable { t.Cap = v; return t }

// SetAddr sets the [Variable.Addr]:
// address where variable is located in memory
func (t *Variable) SetAddr(v uintptr) *Variable { t.Addr = v; return t }

// SetHeap sets the [Variable.Heap]:
// if true, the variable is stored in the main memory heap, not the stack
func (t *Variable) SetHeap(v bool) *Variable { t.Heap = v; return t }

// SetLoc sets the [Variable.Loc]:
// location where the variable was defined in source
func (t *Variable) SetLoc(v Location) *Variable { t.Loc = v; return t }

// SetList sets the [Variable.List]:
// if kind is a list type (array, slice), and elements are primitive types, this is the contents
func (t *Variable) SetList(v ...string) *Variable { t.List = v; return t }

// SetMap sets the [Variable.Map]:
// if kind is a map, and elements are primitive types, this is the contents
func (t *Variable) SetMap(v map[string]string) *Variable { t.Map = v; return t }

// SetMapVar sets the [Variable.MapVar]:
// if kind is a map, and elements are not primitive types, this is the contents
func (t *Variable) SetMapVar(v map[string]*Variable) *Variable { t.MapVar = v; return t }

// SetDbg sets the [Variable.Dbg]:
// our debugger -- for getting further variable data
func (t *Variable) SetDbg(v GiDebug) *Variable { t.Dbg = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/code/cdebug.Params", IDName: "params", Doc: "Params are overall debugger parameters", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Fields: []types.Field{{Name: "Mode", Doc: "mode for running the debugger"}, {Name: "PID", Doc: "process id number to attach to, for Attach mode"}, {Name: "Args", Doc: "optional extra args to pass to the debugger.\nUse -- double-dash and then add args to pass args to the executable\n(double-dash is by itself as a separate arg first).\nFor Debug test, must use -test.run instead of plain -run to specify tests to run."}, {Name: "StatFunc", Doc: "status function for debugger updating status"}, {Name: "VarList", Doc: "parameters for level of detail on overall list of variables"}, {Name: "GetVar", Doc: "parameters for level of detail retrieving a specific variable"}}})
