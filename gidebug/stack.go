// Copyright (c) 2020, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidebug

import (
	"path/filepath"
	"strings"
)

// Location holds program location information.
// In most cases a Location object will represent a physical location, with
// a single PC address held in the PC field.
// FindLocations however returns logical locations that can either have
// multiple PC addresses each (due to inlining) or no PC address at all.
type Location struct {
	PC       uint64    `desc:"program counter (address)"`
	File     string    `desc:"file"`
	Line     int       `desc"line within file"`
	Function *Function `desc:"the function"`
	PCs      []uint64  `desc:"multiple possible PCs possible due to inlining"`
}

// Stackframe describes one frame in a stack trace.
type Stackframe struct {
	Location
	Locals             []*Variable `tableview:"-" desc:"local variables"`
	Arguments          []*Variable `tableview:"-" desc:"local function args"`
	Bottom             bool        `tableview:"-" desc:"this is true if last row"`
	FrameOffset        int64       `tableview:"-" desc:"?"`
	FramePointerOffset int64       `tableview:"-" desc:"?"`
	Defers             []Defer     `tableview:"-" desc:"?"`
	Err                string      `tableview:"-" desc:"err?"`
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
			return frame.Locals[i]
		}
	}
	for i := range frame.Arguments {
		if frame.Arguments[i].Name == name {
			return frame.Arguments[i]
		}
	}
	return nil
}

// Function represents thread-scoped function information.
type Function struct {
	Name      string
	Value     uint64
	Type      byte   `tableview:"-"`
	GoType    uint64 `tableview:"-"`
	Optimized bool   `tableview:"-" desc:"Optimized is true if the function was optimized"`
}

// StacktraceOptions is the type of the Opts field of StacktraceIn that
// configures the stacktrace.
// Tracks proc.StacktraceOptions
type StacktraceOptions uint16

const (
	// StacktraceReadDefers requests a stacktrace decorated with deferred calls
	// for each frame.
	StacktraceReadDefers StacktraceOptions = 1 << iota

	// StacktraceSimple requests a stacktrace where no stack switches will be
	// attempted.
	StacktraceSimple

	// StacktraceG requests a stacktrace starting with the register
	// values saved in the runtime.g structure.
	StacktraceG
)

// ImportPathToDirectoryPath maps an import path to a directory path.
type PackageBuildInfo struct {
	ImportPath    string
	DirectoryPath string
	Files         []string
}

// RelFile sets the file name relative to given base file path
func RelFile(file, base string) string {
	nf, err := filepath.Rel(base, file)
	if err == nil && !strings.HasPrefix(nf, "..") {
		file = nf
	}
	return file
}

// DispStackframe is the stack frame for display purposes
type DispStackframe struct {
	PC       uint64 `desc:"program counter (address)"`
	File     string `desc:"file"`
	Line     int    `desc"line within file"`
	Function string `desc:"the function name"`
}

// StackToDisp translates a stackframe into a more cleanly displayable format
func StackToDisp(sf []*Stackframe, basePath string) []*DispStackframe {
	df := make([]*DispStackframe, len(sf))
	for i, f := range sf {
		ds := &DispStackframe{}
		ds.PC = f.PC
		ds.File = RelFile(f.File, basePath)
		ds.Line = f.Line
		if f.Function != nil && f.Function.Name != "" {
			ds.Function = f.Function.Name
		}
		df[i] = ds
	}
	return df
}
