// Copyright (c) 2020, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidebug

// Breakpoint addresses a location at which process execution may be
// suspended.
type Breakpoint struct {
	// ID is a unique identifier for the breakpoint.
	ID int `json:"id"`

	// User defined name of the breakpoint
	Name string `json:"name"`

	// Addr is the address of the breakpoint.
	Addr uint64 `json:"addr"`

	// Addrs is the list of addresses for this breakpoint.
	Addrs []uint64 `json:"addrs"`

	// File is the source file for the breakpoint.
	File string `json:"file"`

	// Line is a line in File for the breakpoint.
	Line int `json:"line"`

	// FunctionName is the name of the function at the current breakpoint, and
	// may not always be available.
	FunctionName string `json:"functionName,omitempty"`

	// Breakpoint condition
	Cond string

	// Tracepoint flag, signifying this is a tracepoint.
	Tracepoint bool `json:"continue"`

	// TraceReturn flag signifying this is a breakpoint set at a return
	// statement in a traced function.
	TraceReturn bool `json:"traceReturn"`

	// retrieve goroutine information
	Goroutine bool `json:"goroutine"`

	// number of stack frames to retrieve
	Stacktrace int `json:"stacktrace"`

	// expressions to evaluate
	Variables []string `json:"variables,omitempty"`

	// LoadArgs requests loading function arguments when the breakpoint is hit
	LoadArgs *LoadConfig

	// LoadLocals requests loading function locals when the breakpoint is hit
	LoadLocals *LoadConfig

	// number of times a breakpoint has been reached in a certain goroutine
	HitCount map[string]uint64 `json:"hitCount"`

	// number of times a breakpoint has been reached
	TotalHitCount uint64 `json:"totalHitCount"`
}

// BreakpointInfo contains informations about the current breakpoint
type BreakpointInfo struct {
	Stacktrace []Stackframe `json:"stacktrace,omitempty"`
	Goroutine  *Goroutine   `json:"goroutine,omitempty"`
	Variables  []Variable   `json:"variables,omitempty"`
	Arguments  []Variable   `json:"arguments,omitempty"`
	Locals     []Variable   `json:"locals,omitempty"`
}

// DiscardedBreakpoint is a breakpoint that is not
// reinstated during a restart.
type DiscardedBreakpoint struct {
	Breakpoint *Breakpoint
	Reason     string
}
