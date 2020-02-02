// Copyright (c) 2020, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidebug

// DebuggerCommand is a command which changes the debugger's execution state.
type DebuggerCommand struct {
	// Name is the command to run.
	Name string `json:"name"`
	// ThreadID is used to specify which thread to use with the SwitchThread
	// command.
	ThreadID int `json:"threadID,omitempty"`
	// GoroutineID is used to specify which thread to use with the SwitchGoroutine
	// command.
	GoroutineID int `json:"goroutineID,omitempty"`
	// When ReturnInfoLoadConfig is not nil it will be used to load the value
	// of any return variables.
	ReturnInfoLoadConfig *LoadConfig
	// Expr is the expression argument for a Call command
	Expr string `json:"expr,omitempty"`
	// UnsafeCall disabled parameter escape checking for function calls
	UnsafeCall bool `json:"unsafeCall,omitempty"`
}

// EvalScope is the scope a command should
// be evaluated in. Describes the goroutine and frame number.
type EvalScope struct {
	GoroutineID  int
	Frame        int
	DeferredCall int // when DeferredCall is n > 0 this eval scope is relative to the n-th deferred call in the current frame
}

const (
	// Continue resumes process execution.
	Continue = "continue"
	// Rewind resumes process execution backwards (target must be a recording).
	Rewind = "rewind"
	// Step continues to next source line, entering function calls.
	Step = "step"
	// StepOut continues to the return address of the current function
	StepOut = "stepOut"
	// StepInstruction continues for exactly 1 cpu instruction.
	StepInstruction = "stepInstruction"
	// Next continues to the next source line, not entering function calls.
	Next = "next"
	// SwitchThread switches the debugger's current thread context.
	SwitchThread = "switchThread"
	// SwitchGoroutine switches the debugger's current thread context to the thread running the specified goroutine
	SwitchGoroutine = "switchGoroutine"
	// Halt suspends the process.
	Halt = "halt"
	// Call resumes process execution injecting a function call.
	Call = "call"
)
