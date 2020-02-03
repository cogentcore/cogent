// Copyright (c) 2020, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidebug

import "reflect"

// VariableFlags is the type of the Flags field of Variable.
type VariableFlags uint16

const (
	// VariableEscaped is set for local variables that escaped to the heap
	//
	// The compiler performs escape analysis on local variables, the variables
	// that may outlive the stack frame are allocated on the heap instead and
	// only the address is recorded on the stack. These variables will be
	// marked with this flag.
	VariableEscaped = (1 << iota)

	// VariableShadowed is set for local variables that are shadowed by a
	// variable with the same name in another scope
	VariableShadowed

	// VariableConstant means this variable is a constant value
	VariableConstant

	// VariableArgument means this variable is a function argument
	VariableArgument

	// VariableReturnArgument means this variable is a function return value
	VariableReturnArgument
)

// Variable describes a variable.
type Variable struct {
	// Name of the variable or struct member
	Name string `json:"name"`
	// Address of the variable or struct member
	Addr uintptr `json:"addr"`
	// Only the address field is filled (result of evaluating expressions like &<expr>)
	OnlyAddr bool `json:"onlyAddr"`
	// Go type of the variable
	Type string `json:"type"`
	// Type of the variable after resolving any typedefs
	RealType string `json:"realType"`

	Flags VariableFlags `json:"flags"`

	Kind reflect.Kind `json:"kind"`

	//Strings have their length capped at proc.maxArrayValues, use Len for the real length of a string
	//Function variables will store the name of the function in this field
	Value string `json:"value"`

	// Number of elements in an array or a slice, number of keys for a map, number of struct members for a struct, length of strings
	Len int64 `json:"len"`
	// Cap value for slices
	Cap int64 `json:"cap"`

	// Array and slice elements, member fields of structs, key/value pairs of maps, value of complex numbers
	// The Name field in this slice will always be the empty string except for structs (when it will be the field name) and for complex numbers (when it will be "real" and "imaginary")
	// For maps each map entry will have to items in this slice, even numbered items will represent map keys and odd numbered items will represent their values
	// This field's length is capped at proc.maxArrayValues for slices and arrays and 2*proc.maxArrayValues for maps, in the circumstances where the cap takes effect len(Children) != Len
	// The other length cap applied to this field is related to maximum recursion depth, when the maximum recursion depth is reached this field is left empty, contrary to the previous one this cap also applies to structs (otherwise structs will always have all their member fields returned)
	Children []*Variable `json:"children"`

	// Base address of arrays, Base address of the backing array for slices (0 for nil slices)
	// Base address of the backing byte array for strings
	// address of the struct backing chan and map variables
	// address of the function entry point for function variables (0 for nil function pointers)
	Base uintptr `json:"base"`

	// Unreadable addresses will have this field set
	Unreadable string `json:"unreadable"`

	// LocationExpr describes the location expression of this variable's address
	LocationExpr string
	// DeclLine is the line number of this variable's declaration
	DeclLine int64
}

// LoadConfig describes how to load values from target's memory
type LoadConfig struct {
	// FollowPointers requests pointers to be automatically dereferenced.
	FollowPointers bool
	// MaxVariableRecurse is how far to recurse when evaluating nested types.
	MaxVariableRecurse int
	// MaxStringLen is the maximum number of bytes read from a string
	MaxStringLen int
	// MaxArrayValues is the maximum number of elements read from an array, a slice or a map.
	MaxArrayValues int
	// MaxStructFields is the maximum number of fields read from a struct, -1 will read all fields.
	MaxStructFields int
}

// DefaultConfig can be used by default
var DefaultConfig = LoadConfig{
	FollowPointers:     true,
	MaxVariableRecurse: 2,
	MaxStringLen:       200,
	MaxArrayValues:     100,
	MaxStructFields:    -1,
}
