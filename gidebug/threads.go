// Copyright (c) 2020, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidebug

// Thread is a thread within the debugged process.
type Thread struct {
	// ID is a unique identifier for the thread.
	ID int `json:"id"`
	// PC is the current program counter for the thread.
	PC uint64 `json:"pc"`
	// File is the file for the program counter.
	File string `json:"file"`
	// Line is the line number for the program counter.
	Line int `json:"line"`
	// Function is function information at the program counter. May be nil.
	Function *Function `json:"function,omitempty"`

	// ID of the goroutine running on this thread
	GoroutineID int `json:"goroutineID"`

	// Breakpoint this thread is stopped at
	Breakpoint *Breakpoint `json:"breakPoint,omitempty"`
	// Informations requested by the current breakpoint
	BreakpointInfo *BreakpointInfo `json:"breakPointInfo,omitempty"`

	// ReturnValues contains the return values of the function we just stepped out of
	ReturnValues []Variable
}

// Goroutine represents the information relevant to Delve from the runtime's
// internal G structure.
type Goroutine struct {
	// ID is a unique identifier for the goroutine.
	ID int `json:"id"`
	// Current location of the goroutine
	CurrentLoc Location `json:"currentLoc"`
	// Current location of the goroutine, excluding calls inside runtime
	UserCurrentLoc Location `json:"userCurrentLoc"`
	// Location of the go instruction that started this goroutine
	GoStatementLoc Location `json:"goStatementLoc"`
	// Location of the starting function
	StartLoc Location `json:"startLoc"`
	// ID of the associated thread for running goroutines
	ThreadID   int    `json:"threadID"`
	Unreadable string `json:"unreadable"`
}
