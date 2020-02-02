// Copyright (c) 2020, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidelve

import (
	"log"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/go-delve/delve/service/rpc2"
	"github.com/goki/gide/gidebug"
)

// GiDelve is the Delve implementation of the GiDebug interface
type GiDelve struct {
	dlv *rpc2.RPCClient // the delve rpc2 client interface
	cmd *exec.Cmd       // command running delve
}

// Start starts the debugger for a given exe path
func (gd *GiDelve) Start(path string) error {
	gd.cmd = exec.Command("dlv", "debug", "--headless", "--api-version=2", "--log", "--listen=127.0.0.1:8181")
	gd.cmd.Dir = filepath.Dir(path)
	err := gd.cmd.Start()
	if err != nil {
		log.Println(err)
		return err
	}
	gd.dlv = rpc2.NewClient("127.0.0.1:8181")
	return nil
}

// Returns the pid of the process we are debugging.
func (gd *GiDelve) ProcessPid() int {
	if gd.cmd == nil || gd.dlv == nil {
		log.Println(gidebug.NotStartedErr)
		return -1
	}
	return gd.dlv.ProcessPid()
}

// LastModified returns the time that the process' executable was modified.
func (gd *GiDelve) LastModified() time.Time {
	if gd.cmd == nil || gd.dlv == nil {
		log.Println(gidebug.NotStartedErr)
		return time.Time{}
	}
	return gd.dlv.LastModified()
}

// Detach detaches the debugger, optionally killing the process.
func (gd *GiDelve) Detach(killProcess bool) error {
	if gd.cmd == nil || gd.dlv == nil {
		err := gidebug.NotStartedErr
		log.Println(err)
		return err
	}
	return gd.dlv.Detach(killProcess)
}

// Restarts program.
func (gd *GiDelve) Restart() ([]gidebug.DiscardedBreakpoint, error) {
	if gd.cmd == nil || gd.dlv == nil {
		err := gidebug.NotStartedErr
		log.Println(err)
		return nil, err
	}
	db, err := gd.dlv.Restart()
	_ = db
	return nil, err
}

// Restarts program from the specified position.
func (gd *GiDelve) RestartFrom(pos string, resetArgs bool, newArgs []string) ([]gidebug.DiscardedBreakpoint, error) {
	if gd.cmd == nil || gd.dlv == nil {
		err := gidebug.NotStartedErr
		log.Println(err)
		return nil, err
	}
	db, err := gd.dlv.RestartFrom(pos, resetArgs, newArgs)
	_ = db
	return nil, err
}

// GetState returns the current debugger state.
func (gd *GiDelve) GetState() (*gidebug.DebuggerState, error) {
	if gd.cmd == nil || gd.dlv == nil {
		err := gidebug.NotStartedErr
		log.Println(err)
		return nil, err
	}
	ds, err := gd.dlv.GetState()
	return CvtDebuggerState(ds), err
}

// GetStateNonBlocking returns the current debugger state,
// returning immediately if the target is already running.
func (gd *GiDelve) GetStateNonBlocking() (*gidebug.DebuggerState, error) {
	if gd.cmd == nil || gd.dlv == nil {
		err := gidebug.NotStartedErr
		log.Println(err)
		return nil, err
	}
	ds, err := gd.dlv.GetStateNonBlocking()
	return CvtDebuggerState(ds), err
}

// Continue resumes process execution.
func (gd *GiDelve) Continue() <-chan *gidebug.DebuggerState {
	return nil
}

// Rewind resumes process execution backwards.
func (gd *GiDelve) Rewind() <-chan *gidebug.DebuggerState {
	return nil
}

// Next continues to the next source line, not entering function calls.
func (gd *GiDelve) Next() (*gidebug.DebuggerState, error) {
	if gd.cmd == nil || gd.dlv == nil {
		err := gidebug.NotStartedErr
		log.Println(err)
		return nil, err
	}
	ds, err := gd.dlv.Next()
	return CvtDebuggerState(ds), err
}

// Step continues to the next source line, entering function calls.
func (gd *GiDelve) Step() (*gidebug.DebuggerState, error) {
	if gd.cmd == nil || gd.dlv == nil {
		err := gidebug.NotStartedErr
		log.Println(err)
		return nil, err
	}
	ds, err := gd.dlv.Step()
	return CvtDebuggerState(ds), err
}

// StepOut continues to the return address of the current function
func (gd *GiDelve) StepOut() (*gidebug.DebuggerState, error) {
	if gd.cmd == nil || gd.dlv == nil {
		err := gidebug.NotStartedErr
		log.Println(err)
		return nil, err
	}
	ds, err := gd.dlv.StepOut()
	return CvtDebuggerState(ds), err
}

// Call resumes process execution while making a function call.
func (gd *GiDelve) Call(expr string, unsafe bool) (*gidebug.DebuggerState, error) {
	if gd.cmd == nil || gd.dlv == nil {
		err := gidebug.NotStartedErr
		log.Println(err)
		return nil, err
	}
	ds, err := gd.dlv.Call(expr, unsafe)
	return CvtDebuggerState(ds), err
}

// SingleStep will step a single cpu instruction.
func (gd *GiDelve) StepInstruction() (*gidebug.DebuggerState, error) {
	if gd.cmd == nil || gd.dlv == nil {
		err := gidebug.NotStartedErr
		log.Println(err)
		return nil, err
	}
	ds, err := gd.dlv.StepInstruction()
	return CvtDebuggerState(ds), err
}

// SwitchThread switches the current thread context.
func (gd *GiDelve) SwitchThread(threadID int) (*gidebug.DebuggerState, error) {
	if gd.cmd == nil || gd.dlv == nil {
		err := gidebug.NotStartedErr
		log.Println(err)
		return nil, err
	}
	ds, err := gd.dlv.SwitchThread(threadID)
	return CvtDebuggerState(ds), err
}

// SwitchGoroutine switches the current goroutine (and the current thread as well)
func (gd *GiDelve) SwitchGoroutine(goroutineID int) (*gidebug.DebuggerState, error) {
	if gd.cmd == nil || gd.dlv == nil {
		err := gidebug.NotStartedErr
		log.Println(err)
		return nil, err
	}
	ds, err := gd.dlv.SwitchGoroutine(goroutineID)
	return CvtDebuggerState(ds), err
}

// Halt suspends the process.
func (gd *GiDelve) Halt() (*gidebug.DebuggerState, error) {
	if gd.cmd == nil || gd.dlv == nil {
		err := gidebug.NotStartedErr
		log.Println(err)
		return nil, err
	}
	ds, err := gd.dlv.Halt()
	return CvtDebuggerState(ds), err
}

// GetBreakpoint gets a breakpoint by ID.
func (gd *GiDelve) GetBreakpoint(id int) (*gidebug.Breakpoint, error) {
	if gd.cmd == nil || gd.dlv == nil {
		err := gidebug.NotStartedErr
		log.Println(err)
		return nil, err
	}
	ds, err := gd.dlv.GetBreakpoint(id)
	return CvtBreakpoint(ds), err
}

// GetBreakpointByName gets a breakpoint by name.
func (gd *GiDelve) GetBreakpointByName(name string) (*gidebug.Breakpoint, error) {
	if gd.cmd == nil || gd.dlv == nil {
		err := gidebug.NotStartedErr
		log.Println(err)
		return nil, err
	}
	ds, err := gd.dlv.GetBreakpointByName(name)
	return CvtBreakpoint(ds), err
}

// CreateBreakpoint creates a new breakpoint.
func (gd *GiDelve) CreateBreakpoint(*gidebug.Breakpoint) (*gidebug.Breakpoint, error) {
	if gd.cmd == nil || gd.dlv == nil {
		err := gidebug.NotStartedErr
		log.Println(err)
		return nil, err
	}
	ds, err := gd.dlv.CreateBreakpoint(nil) // todo: need to cvt the other way!
	return CvtBreakpoint(ds), err
}

// ListBreakpoints gets all breakpoints.
func (gd *GiDelve) ListBreakpoints() ([]*gidebug.Breakpoint, error) {
	return nil, nil
}

// ClearBreakpoint deletes a breakpoint by ID.
func (gd *GiDelve) ClearBreakpoint(id int) (*gidebug.Breakpoint, error) {
	return nil, nil
}

// ClearBreakpointByName deletes a breakpoint by name
func (gd *GiDelve) ClearBreakpointByName(name string) (*gidebug.Breakpoint, error) {
	return nil, nil
}

// Allows user to update an existing breakpoint for example to change the information
// retrieved when the breakpoint is hit or to change, add or remove the break condition
func (gd *GiDelve) AmendBreakpoint(*gidebug.Breakpoint) error {
	return nil
}

// Cancels a Next or Step call that was interrupted by a manual stop or by another breakpoint
func (gd *GiDelve) CancelNext() error {
	return nil
}

// ListThreads lists all threads.
func (gd *GiDelve) ListThreads() ([]*gidebug.Thread, error) {
	return nil, nil
}

// GetThread gets a thread by its ID.
func (gd *GiDelve) GetThread(id int) (*gidebug.Thread, error) {
	return nil, nil
}

// ListPackageVariables lists all package variables in the context of the current thread.
func (gd *GiDelve) ListPackageVariables(filter string, cfg gidebug.LoadConfig) ([]gidebug.Variable, error) {
	return nil, nil
}

// EvalVariable returns a variable in the context of the current thread.
func (gd *GiDelve) EvalVariable(scope gidebug.EvalScope, symbol string, cfg gidebug.LoadConfig) (*gidebug.Variable, error) {
	return nil, nil
}

// SetVariable sets the value of a variable
func (gd *GiDelve) SetVariable(scope gidebug.EvalScope, symbol, value string) error {
	return nil
}

// ListSources lists all source files in the process matching filter.
func (gd *GiDelve) ListSources(filter string) ([]string, error) {
	return nil, nil
}

// ListFunctions lists all functions in the process matching filter.
func (gd *GiDelve) ListFunctions(filter string) ([]string, error) {
	return nil, nil
}

// ListTypes lists all types in the process matching filter.
func (gd *GiDelve) ListTypes(filter string) ([]string, error) {
	return nil, nil
}

// ListLocals lists all local variables in scope.
func (gd *GiDelve) ListLocalVariables(scope gidebug.EvalScope, cfg gidebug.LoadConfig) ([]gidebug.Variable, error) {
	return nil, nil
}

// ListFunctionArgs lists all arguments to the current function.
func (gd *GiDelve) ListFunctionArgs(scope gidebug.EvalScope, cfg gidebug.LoadConfig) ([]gidebug.Variable, error) {
	return nil, nil
}

// ListRegisters lists registers and their values.
// func (gd *GiDelve) ListRegisters(threadID int, includeFp bool) (Registers, error)

// ListGoroutines lists all goroutines.
func (gd *GiDelve) ListGoroutines(start, count int) ([]*gidebug.Goroutine, int, error) {
	return nil, 0, nil
}

// Returns stacktrace
func (gd *GiDelve) Stacktrace(goroutineID int, depth int, readDefers bool, cfg *gidebug.LoadConfig) ([]gidebug.Stackframe, error) {
	return nil, nil
}

// Returns whether we attached to a running process or not
func (gd *GiDelve) AttachedToExistingProcess() bool {
	return false
}

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
func (gd *GiDelve) FindLocation(scope gidebug.EvalScope, loc string) ([]gidebug.Location, error) {
	return nil, nil
}

/*
	// Disassemble code between startPC and endPC
	DisassembleRange(scope gidebug.EvalScope, startPC, endPC uint64, flavour AssemblyFlavour) (AsmInstructions, error)

	// Disassemble code of the function containing PC
	DisassemblePC(scope gidebug.EvalScope, pc uint64, flavour AssemblyFlavour) (AsmInstructions, error)
*/

// Disconnect closes the connection to the server without sending a Detach request first.
// If cont is true a continue command will be sent instead.
func (gd *GiDelve) Disconnect(cont bool) error {
	return nil
}
