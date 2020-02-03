// Copyright (c) 2020, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidebug

import (
	"errors"
	"time"

	"github.com/goki/gi/giv"
)

var NotStartedErr = errors.New("debugger not started")

// GiDebug is the interface for all supported debuggers.
// It is based directly on the Delve Client interface.
type GiDebug interface {
	// Start starts the debugger for a given exe path, and sets output of debugger
	// session to given textbuf which is used to monitor output
	Start(path string, outbuf *giv.TextBuf) error

	// IsActive returns true if the debugger is active and ready for commands
	IsActive() bool

	// Returns the pid of the process we are debugging.
	ProcessPid() int

	// LastModified returns the time that the process' executable was modified.
	LastModified() time.Time

	// Detach detaches the debugger, optionally killing the process.
	Detach(killProcess bool) error

	// Restarts program.
	Restart() ([]*DiscardedBreakpoint, error)

	// Restarts program from the specified position.
	RestartFrom(pos string, resetArgs bool, newArgs []string) ([]*DiscardedBreakpoint, error)

	// GetState returns the current debugger state.
	GetState() (*DebuggerState, error)

	// GetStateNonBlocking returns the current debugger state,
	// returning immediately if the target is already running.
	GetStateNonBlocking() (*DebuggerState, error)

	// Continue resumes process execution.
	Continue() <-chan *DebuggerState

	// Rewind resumes process execution backwards.
	Rewind() <-chan *DebuggerState

	// Next continues to the next source line, not entering function calls.
	Next() (*DebuggerState, error)

	// Step continues to the next source line, entering function calls.
	Step() (*DebuggerState, error)

	// StepOut continues to the return address of the current function
	StepOut() (*DebuggerState, error)

	// Call resumes process execution while making a function call.
	Call(goroutineID int, expr string, unsafe bool) (*DebuggerState, error)

	// SingleStep will step a single cpu instruction.
	StepInstruction() (*DebuggerState, error)

	// SwitchThread switches the current thread context.
	SwitchThread(threadID int) (*DebuggerState, error)

	// SwitchGoroutine switches the current goroutine (and the current thread as well)
	SwitchGoroutine(goroutineID int) (*DebuggerState, error)

	// Halt suspends the process.
	Halt() (*DebuggerState, error)

	// GetBreakpoint gets a breakpoint by ID.
	GetBreakpoint(id int) (*Breakpoint, error)

	// GetBreakpointByName gets a breakpoint by name.
	GetBreakpointByName(name string) (*Breakpoint, error)

	// CreateBreakpoint creates a new breakpoint.
	CreateBreakpoint(bp *Breakpoint) (*Breakpoint, error)

	// ListBreakpoints gets all breakpoints.
	ListBreakpoints() ([]*Breakpoint, error)

	// ClearBreakpoint deletes a breakpoint by ID.
	ClearBreakpoint(id int) (*Breakpoint, error)

	// ClearBreakpointByName deletes a breakpoint by name
	ClearBreakpointByName(name string) (*Breakpoint, error)

	// AmmendBreakpoint allows user to update an existing breakpoint for example
	// to change the information retrieved when the breakpoint is hit or to change,
	// add or remove the break condition
	AmendBreakpoint(bp *Breakpoint) error

	// Cancels a Next or Step call that was interrupted by a manual stop or by another breakpoint
	CancelNext() error

	// ListThreads lists all threads.
	ListThreads() ([]*Thread, error)

	// GetThread gets a thread by its ID.
	GetThread(id int) (*Thread, error)

	// ListPackageVariables lists all package variables in the context of the current thread.
	ListPackageVariables(filter string, cfg LoadConfig) ([]*Variable, error)

	// EvalVariable returns a variable in the context of the current thread.
	EvalVariable(scope EvalScope, symbol string, cfg LoadConfig) (*Variable, error)

	// SetVariable sets the value of a variable
	SetVariable(scope EvalScope, symbol, value string) error

	// ListSources lists all source files in the process matching filter.
	ListSources(filter string) ([]string, error)

	// ListFunctions lists all functions in the process matching filter.
	ListFunctions(filter string) ([]string, error)

	// ListTypes lists all types in the process matching filter.
	ListTypes(filter string) ([]string, error)

	// ListLocals lists all local variables in scope.
	ListLocalVariables(scope EvalScope, cfg LoadConfig) ([]*Variable, error)

	// ListFunctionArgs lists all arguments to the current function.
	ListFunctionArgs(scope EvalScope, cfg LoadConfig) ([]*Variable, error)

	// ListRegisters lists registers and their values.
	// ListRegisters(threadID int, includeFp bool) (Registers, error)

	// ListGoroutines lists all goroutines.
	ListGoroutines(start, count int) ([]*Goroutine, int, error)

	// Returns stacktrace
	Stacktrace(goroutineID int, depth int, opts StacktraceOptions, cfg *LoadConfig) ([]*Stackframe, error)

	// Returns whether we attached to a running process or not
	AttachedToExistingProcess() bool

	// Returns concrete location information described by a location expression
	// loc ::= <filename>:<line> | <function>[:<line>] | /<regex>/ | (+|-)<offset> | <line> | *<address>
	// * <filename> can be the full path of a file or just a suffix
	// * <function> ::= <package>.<receiver type>.<name> | <package>.(*<receiver type>).<name> | <receiver type>.<name> | <package>.<name> | (*<receiver type>).<name> | <name>
	// * <function> must be unambiguous
	// * /<regex>/ will return a location for each function matched by regex
	// * +<offset> returns a location for the line that is <offset> lines after the current line
	// * -<offset> returns a location for the line that is <offset> lines before the current line
	// * <line> returns a location for a line in the current file
	// * *<address> returns the location corresponding to the specified address
	// NOTE: this function does not actually set breakpoints.
	// If findInstruction is true FindLocation will only return locations that correspond to instructions.
	FindLocation(scope EvalScope, loc string, findInstruction bool) ([]*Location, error)

	/*
		// Disassemble code between startPC and endPC
		DisassembleRange(scope EvalScope, startPC, endPC uint64, flavour AssemblyFlavour) (AsmInstructions, error)

		// Disassemble code of the function containing PC
		DisassemblePC(scope EvalScope, pc uint64, flavour AssemblyFlavour) (AsmInstructions, error)
	*/

	// SetReturnValuesLoadConfig sets the load configuration for return values.
	SetReturnValuesLoadConfig(cfg *LoadConfig)

	// Disconnect closes the connection to the server without sending a Detach request first.
	// If cont is true a continue command will be sent instead.
	Disconnect(cont bool) error
}
