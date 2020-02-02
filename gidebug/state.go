// Copyright (c) 2020, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidebug

// DebuggerState represents the current context of the debugger.
type DebuggerState struct {
	// Running is true if the process is running and no other information can be collected.
	Running bool
	// CurrentThread is the currently selected debugger thread.
	CurrentThread *Thread `json:"currentThread,omitempty"`
	// SelectedGoroutine is the currently selected goroutine
	SelectedGoroutine *Goroutine `json:"currentGoroutine,omitempty"`
	// List of all the process threads
	Threads []*Thread
	// NextInProgress indicates that a next or step operation was interrupted by another breakpoint
	// or a manual stop and is waiting to complete.
	// While NextInProgress is set further requests for next or step may be rejected.
	// Either execute continue until NextInProgress is false or call CancelNext
	NextInProgress bool
	// Exited indicates whether the debugged process has exited.
	Exited     bool `json:"exited"`
	ExitStatus int  `json:"exitStatus"`
	// When contains a description of the current position in a recording
	When string
	// Filled by RPCClient.Continue, indicates an error
	Err error `json:"-"`
}
