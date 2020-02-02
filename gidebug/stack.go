// Copyright (c) 2020, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidebug

// Location holds program location information.
type Location struct {
	PC       uint64    `json:"pc"`
	File     string    `json:"file"`
	Line     int       `json:"line"`
	Function *Function `json:"function,omitempty"`
}

// Stackframe describes one frame in a stack trace.
type Stackframe struct {
	Location
	Locals    []Variable
	Arguments []Variable

	FrameOffset        int64
	FramePointerOffset int64

	Defers []Defer

	Bottom bool `json:"Bottom,omitempty"` // Bottom is true if this is the bottom frame of the stack

	Err string
}

// Defer describes a deferred function.
type Defer struct {
	DeferredLoc Location // deferred function
	DeferLoc    Location // location of the defer statement
	SP          uint64   // value of SP when the function was deferred
	Unreadable  string
}

// Var will return the variable described by 'name' within
// this stack frame.
func (frame *Stackframe) Var(name string) *Variable {
	for i := range frame.Locals {
		if frame.Locals[i].Name == name {
			return &frame.Locals[i]
		}
	}
	for i := range frame.Arguments {
		if frame.Arguments[i].Name == name {
			return &frame.Arguments[i]
		}
	}
	return nil
}

// Function represents thread-scoped function information.
type Function struct {
	// Name is the function name.
	Name_  string `json:"name"`
	Value  uint64 `json:"value"`
	Type   byte   `json:"type"`
	GoType uint64 `json:"goType"`
	// Optimized is true if the function was optimized
	Optimized bool `json:"optimized"`
}

// Name will return the function name.
func (fn *Function) Name() string {
	if fn == nil {
		return "???"
	}
	return fn.Name_
}
