// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package cdelve

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cogentcore.org/cogent/code/cdebug"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/num"
	"cogentcore.org/core/text/highlighting"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textcore"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
)

func init() {
	cdebug.Debuggers[fileinfo.Go] = func(path, rootPath string, outbuf *lines.Lines, pars *cdebug.Params) (cdebug.GiDebug, error) {
		return NewGiDelve(path, rootPath, outbuf, pars)
	}
}

// GiDelve is the Delve implementation of the GiDebug interface
type GiDelve struct {
	path          string                   // path to exe
	rootPath      string                   // root path for project
	conn          string                   // connection ip addr and port (127.0.0.1:<port>) -- what we pass to RPCClient
	dlv           *rpc2.RPCClient          // the delve rpc2 client interface
	cmd           *exec.Cmd                // command running delve
	obuf          *textcore.OutputBuffer   // output buffer
	lastEvalScope *api.EvalScope           // last used EvalScope
	statFunc      func(stat cdebug.Status) // status function
	params        cdebug.Params            // local copy of initial params
}

// NewGiDelve creates a new debugger exe and client
// for given path, and project root path
// test = run in test mode, and args are optional additional args to pass
// to the debugger.
func NewGiDelve(path, rootPath string, outbuf *lines.Lines, pars *cdebug.Params) (*GiDelve, error) {
	gd := &GiDelve{}
	err := gd.Start(path, rootPath, outbuf, pars)
	return gd, err
}

func (gd *GiDelve) HasTasks() bool {
	return true
}

func (gd *GiDelve) WriteToConsole(msg string) {
	if gd.obuf == nil {
		log.Println(msg)
		return
	}
	tlns := []rune(msg)
	mlns := rich.NewPlainText(tlns)
	gd.obuf.Lines.AppendTextMarkup([][]rune{tlns}, []rich.Text{mlns})
}

func (gd *GiDelve) LogErr(err error) error {
	if err == nil {
		return err
	}
	gd.WriteToConsole(err.Error() + "\n")
	return err
}

func (gd *GiDelve) SetParams(params *cdebug.Params) {
	gd.params = *params
	if err := gd.StartedCheck(); err != nil {
		return
	}
	lc := gd.toLoadConfig(&gd.params.VarList)
	gd.dlv.SetReturnValuesLoadConfig(lc)
}

// StartedCheck checks that delve client is running properly
func (gd *GiDelve) StartedCheck() error {
	if gd.cmd == nil || gd.dlv == nil {
		err := cdebug.NotStartedErr
		return gd.LogErr(err)
	}
	return nil
}

// Start starts the debugger for a given exe path
func (gd *GiDelve) Start(path, rootPath string, outbuf *lines.Lines, pars *cdebug.Params) error {
	gd.path = path
	gd.rootPath = rootPath
	gd.params = *pars
	gd.statFunc = pars.StatFunc
	switch pars.Mode {
	case cdebug.Exec:
		targs := []string{"debug", "--headless", "--api-version=2"}
		targs = append(targs, gd.params.Args...)
		gd.cmd = exec.Command("dlv", targs...)
	case cdebug.Test:
		targs := []string{"test", "--headless", "--api-version=2"}
		if pars.TestName != "" {
			targs = append(targs, "--", "-test.run", pars.TestName)
		}
		targs = append(targs, gd.params.Args...)
		gd.cmd = exec.Command("dlv", targs...)
	case cdebug.Attach:
		// note: --log here creates huge amounts of messages and doesn't work..
		targs := []string{"attach", fmt.Sprintf("%d", gd.params.PID), "--headless", "--api-version=2"}
		targs = append(targs, gd.params.Args...)
		gd.cmd = exec.Command("dlv", targs...)
	}
	gd.cmd.Dir = filepath.Dir(path)
	stdout, err := gd.cmd.StdoutPipe()
	if err == nil {
		gd.cmd.Stderr = gd.cmd.Stdout
		err = gd.cmd.Start()
		if err == nil {
			gd.obuf = &textcore.OutputBuffer{}
			gd.obuf.SetOutput(stdout).SetLines(outbuf).SetMarkupFunc(gd.monitorOutput)
			go gd.obuf.MonitorOutput()
		}
	}
	if err != nil {
		gd.statFunc(cdebug.Error)
		return gd.LogErr(err)
	}
	return nil
}

func (gd *GiDelve) monitorOutput(buf *lines.Lines, out []rune) rich.Text {
	sty := buf.FontStyle()
	mu := rich.NewText(sty, out)
	if gd.conn != "" {
		return mu
	}
	sout := string(out)
	flds := strings.Fields(sout)
	if len(flds) == 0 {
		return mu
	}
	if flds[0] == "API" && flds[1] == "server" && flds[2] == "listening" && flds[3] == "at:" {
		gd.conn = flds[4]
		gd.dlv = rpc2.NewClient(gd.conn)
		gd.SetParams(&gd.params)
		if gd.statFunc != nil {
			gd.statFunc(cdebug.Ready)
		}
		return mu
	}
	if flds[0] == "exit" && flds[1] == "status" {
		if gd.statFunc != nil {
			gd.statFunc(cdebug.Error)
		}
		return mu
	}
	return highlighting.MarkupPathsAsLinks(out, mu, 2) // only first 2 fields
}

// IsActive returns whether debugger is active and ready for commands
func (gd *GiDelve) IsActive() bool {
	return gd.cmd != nil && gd.dlv != nil
}

// Returns the pid of the process we are debugging.
func (gd *GiDelve) ProcessPid() int {
	if err := gd.StartedCheck(); err != nil {
		log.Println(cdebug.NotStartedErr)
		return -1
	}
	return gd.dlv.ProcessPid()
}

// LastModified returns the time that the process' executable was modified.
func (gd *GiDelve) LastModified() time.Time {
	if err := gd.StartedCheck(); err != nil {
		log.Println(cdebug.NotStartedErr)
		return time.Time{}
	}
	return gd.dlv.LastModified()
}

// Detach detaches the debugger, optionally killing the process.
func (gd *GiDelve) Detach(killProcess bool) error {
	var err error
	if gd.dlv != nil {
		err = gd.dlv.Detach(killProcess)
		gd.dlv = nil
	}
	if gd.cmd != nil && gd.cmd.Process != nil { // make sure it dies!
		err = gd.cmd.Process.Kill()
	}
	return err
}

// Disconnect closes the connection to the server without sending a Detach request first.
// If cont is true a continue command will be sent instead.
func (gd *GiDelve) Disconnect(cont bool) error {
	if err := gd.StartedCheck(); err != nil {
		return err
	}
	gd.dlv.Disconnect(cont)
	return nil
}

// Restarts program.
func (gd *GiDelve) Restart() error {
	if err := gd.StartedCheck(); err != nil {
		return err
	}
	_, err := gd.dlv.Restart(true)
	return gd.LogErr(err)
}

// Restarts program from the specified position.
func (gd *GiDelve) RestartFrom(pos string, resetArgs bool, newArgs []string) error {
	if err := gd.StartedCheck(); err != nil {
		return err
	}
	// note: [3]string is new redirects, which are files that can be used to redirect input,
	// output and error streams.  Introduced in 1.5.1
	_, err := gd.dlv.RestartFrom(false, pos, resetArgs, newArgs, [3]string{"", "", ""}, true)
	return gd.LogErr(err)
}

// GetState returns the current debugger state.
// This will return immediately -- if the target is running then
// the Running flag will be set and a Stop bus be called to
// get any further information about the target.
func (gd *GiDelve) GetState() (*cdebug.State, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ds, err := gd.dlv.GetStateNonBlocking() // using non-blocking!
	gd.LogErr(err)
	return gd.cvtState(ds), err
}

// Continue resumes process execution.
func (gd *GiDelve) Continue(all *cdebug.AllState) <-chan *cdebug.State {
	if err := gd.StartedCheck(); err != nil {
		return nil
	}
	dsc := gd.dlv.Continue()
	sc := make(chan *cdebug.State)
	go func() {
		for nv := range dsc {
			if nv.Err != nil {
				gd.LogErr(nv.Err)
			}
			ds := gd.cvtState(nv)
			if !ds.Exited {
				bk, _ := cdebug.BreakByFile(all.Breaks, ds.Task.FPath, ds.Task.Line)
				if bk != nil && bk.Trace {
					ds.CurTrace = bk.ID
					gd.WriteToConsole(fmt.Sprintf("Trace: %d File: %s:%d\n", bk.ID, ds.Task.File, ds.Task.Line))
					continue
				}
			}
			// fmt.Printf("sending %s\n", ds.String())
			sc <- ds
		}
		close(sc)
	}()
	return sc
}

// // Rewind resumes process execution backwards.
// func (gd *GiDelve) Rewind() <-chan *cdebug.State {
// 	if err := gd.StartedCheck(); err != nil {
// 		return nil
// 	}
// 	ds := gd.dlv.Rewind()
// 	return gd.cvtStateChan(ds)
// }

// StepOver continues to the next source line, not entering function calls.
func (gd *GiDelve) StepOver() (*cdebug.State, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ds, err := gd.dlv.Next()
	gd.LogErr(err)
	return gd.cvtState(ds), err
}

// StepInto continues to the next source line, entering function calls.
func (gd *GiDelve) StepInto() (*cdebug.State, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ds, err := gd.dlv.Step()
	gd.LogErr(err)
	return gd.cvtState(ds), err
}

// StepOut continues to the return address of the current function
func (gd *GiDelve) StepOut() (*cdebug.State, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ds, err := gd.dlv.StepOut()
	gd.LogErr(err)
	return gd.cvtState(ds), err
}

// StepSingle steps a single cpu instruction.
func (gd *GiDelve) StepSingle() (*cdebug.State, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ds, err := gd.dlv.StepInstruction()
	gd.LogErr(err)
	return gd.cvtState(ds), err
}

// Call resumes process execution while making a function call.
func (gd *GiDelve) Call(goroutineID int, expr string, unsafe bool) (*cdebug.State, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ds, err := gd.dlv.Call(int64(goroutineID), expr, unsafe)
	gd.LogErr(err)
	return gd.cvtState(ds), err
}

// SwitchThread switches the current thread context.
func (gd *GiDelve) SwitchThread(threadID int) (*cdebug.State, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ds, err := gd.dlv.SwitchThread(threadID)
	gd.LogErr(err)
	return gd.cvtState(ds), err
}

// SwitchTask switches the current goroutine (and the current thread as well)
func (gd *GiDelve) SwitchTask(goroutineID int) (*cdebug.State, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ds, err := gd.dlv.SwitchGoroutine(int64(goroutineID))
	gd.LogErr(err)
	return gd.cvtState(ds), err
}

// Stop suspends the process.
func (gd *GiDelve) Stop() (*cdebug.State, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ds, err := gd.dlv.Halt()
	gd.LogErr(err)
	return gd.cvtState(ds), err
}

// GetBreak gets a breakpoint by ID.
func (gd *GiDelve) GetBreak(id int) (*cdebug.Break, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ds, err := gd.dlv.GetBreakpoint(id)
	gd.LogErr(err)
	return gd.cvtBreak(ds), err
}

// GetBreakByName gets a breakpoint by name.
func (gd *GiDelve) GetBreakByName(name string) (*cdebug.Break, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ds, err := gd.dlv.GetBreakpointByName(name)
	gd.LogErr(err)
	return gd.cvtBreak(ds), err
}

// SetBreak sets a new breakpoint at given file and line number
func (gd *GiDelve) SetBreak(fname string, line int) (*cdebug.Break, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	bp := &api.Breakpoint{}
	bp.File = fname
	bp.Line = line
	ds, err := gd.dlv.CreateBreakpoint(bp)
	gd.LogErr(err)
	return gd.cvtBreak(ds), err
}

// ListBreaks gets all breakpoints.
func (gd *GiDelve) ListBreaks() ([]*cdebug.Break, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ds, err := gd.dlv.ListBreakpoints(true) // true = all
	gd.LogErr(err)
	return gd.cvtBreaks(ds), err
}

// ClearBreak deletes a breakpoint by ID.
func (gd *GiDelve) ClearBreak(id int) error {
	if err := gd.StartedCheck(); err != nil {
		return err
	}
	_, err := gd.dlv.ClearBreakpoint(id)
	return gd.LogErr(err)
}

// ClearBreakByName deletes a breakpoint by name
func (gd *GiDelve) ClearBreakByName(name string) (*cdebug.Break, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ds, err := gd.dlv.ClearBreakpointByName(name)
	gd.LogErr(err)
	return gd.cvtBreak(ds), err
}

// AmmendBreak allows user to update an existing breakpoint for example
// to change the information retrieved when the breakpoint is hit or to change,
// add or remove the break condition
func (gd *GiDelve) AmendBreak(id int, fname string, line int, cond string, trace bool) error {
	if err := gd.StartedCheck(); err != nil {
		return err
	}
	bp := &api.Breakpoint{}
	bp.ID = id
	bp.File = fname
	bp.Line = line
	bp.Cond = cond
	bp.Tracepoint = trace
	err := gd.dlv.AmendBreakpoint(bp)
	return gd.LogErr(err)
}

// UpdateBreaks updates current breakpoints based on given list of breakpoints.
// first gets the current list, and does actions to ensure that the list is set.
func (gd *GiDelve) UpdateBreaks(brk *[]*cdebug.Break) error {
	if err := gd.StartedCheck(); err != nil {
		return err
	}
	cb, err := gd.ListBreaks()
	if err != nil {
		return gd.LogErr(err)
	}
	for itr := 0; itr < 2; itr++ {
		update := false
		for _, b := range *brk {
			c, ci := cdebug.BreakByFile(cb, b.FPath, b.Line)
			if c != nil && c.ID > 0 {
				if !b.On {
					cb = append(cb[:ci], cb[ci+1:]...) // remove from cb
					gd.ClearBreak(c.ID)                // remove from list
					continue
				}
				bc := b.Cond
				bt := b.Trace
				if bc != c.Cond || bt != c.Trace {
					gd.AmendBreak(c.ID, c.File, c.Line, b.Cond, b.Trace)
				}
				*b = *c
				b.Cond = bc
				b.Trace = bt
				cb = append(cb[:ci], cb[ci+1:]...) // remove from cb
			} else { // set but not found
				if b.On {
					update = true // need another iter
					gd.SetBreak(b.FPath, b.Line)
				}
			}
		}
		for _, c := range cb { // any we didn't get
			if c.ID <= 0 {
				continue
			}
			*brk = append(*brk, c)
		}
		if update {
			cb, err = gd.ListBreaks()
		} else {
			break
		}
	}
	cdebug.SortBreaks(*brk)
	return nil
}

// Cancels a Next or Step call that was interrupted by a manual stop or by another breakpoint
func (gd *GiDelve) CancelNext() error {
	if err := gd.StartedCheck(); err != nil {
		return err
	}
	err := gd.dlv.CancelNext()
	return gd.LogErr(err)
}

// InitAllState initializes the given AllState with relevant info for
// current state of things.  Does Not get AllVars
func (gd *GiDelve) InitAllState(all *cdebug.AllState) error {
	all.CurThread = all.State.Thread.ID
	all.CurTask = all.State.Task.ID
	all.CurFrame = 0
	st, err := gd.ListThreads()
	if err != nil {
		return err
	}
	all.Threads = st
	th, err := gd.ListTasks()
	if err != nil {
		return err
	}
	all.Tasks = th
	sf, err := gd.Stack(all.CurTask, 100)
	if err != nil {
		return err
	}
	all.Stack = sf
	tsk, _ := cdebug.TaskByID(all.Tasks, all.CurTask)
	if tsk != nil && tsk.Func != "" {
		vr, err := gd.ListVars(all.CurTask, 0)
		if err != nil {
			return err
		}
		all.Vars = vr
	}

	all.CurBreak = 0
	cf := all.StackFrame(0)
	if cf != nil {
		bk, _ := cdebug.BreakByFile(all.Breaks, cf.FPath, cf.Line)
		if bk != nil {
			all.CurBreak = bk.ID
		}
	}

	return nil
}

// UpdateAllState updates the state for given threadId and
// frame number (only info different from current results is updated).
// For given thread (lowest-level supported by language,
// e.g., Task if supported, else Thread), and frame number.
func (gd *GiDelve) UpdateAllState(all *cdebug.AllState, threadID int, frame int) error {
	update := false
	if threadID != all.CurTask {
		update = true
		all.CurTask = threadID
		sf, err := gd.Stack(all.CurTask, 100)
		if err != nil {
			return err
		}
		all.Stack = sf
	}
	if update || all.CurFrame != frame {
		all.CurFrame = frame
		tsk, _ := cdebug.TaskByID(all.Tasks, all.CurTask)
		if tsk != nil && tsk.Func != "" {
			vr, err := gd.ListVars(all.CurTask, all.CurFrame)
			if err != nil {
				return err
			}
			all.Vars = vr
		}

	}
	return nil
}

// FindFrames looks through the Stacks of all Tasks / Threads
// for the closest Stack Frame to given file and line number.
// File name search uses Contains to allow for paths to be searched.
// Results are sorted by line number proximity to given line.
func (gd *GiDelve) FindFrames(all *cdebug.AllState, fname string, line int) ([]*cdebug.Frame, error) {
	var err error
	var fr []*cdebug.Frame
	for _, tsk := range all.Tasks {
		sf, err := gd.Stack(tsk.ID, 100)
		if err != nil {
			break
		}
		for _, f := range sf {
			if !strings.Contains(f.FPath, fname) {
				continue
			}
			fr = append(fr, f)
			break
		}
	}
	sort.Slice(fr, func(i, j int) bool {
		dsti := num.Abs(fr[i].Line - line)
		dstj := num.Abs(fr[j].Line - line)
		return dsti < dstj
	})
	return fr, err
}

// CurThreadID returns the proper current threadID (task or thread)
// based on debugger, from given state.
func (gd *GiDelve) CurThreadID(all *cdebug.AllState) int {
	return all.CurTask
}

// ListThreads lists all threads.
func (gd *GiDelve) ListThreads() ([]*cdebug.Thread, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ds, err := gd.dlv.ListThreads()
	gd.LogErr(err)
	return gd.cvtThreads(ds), err
}

// GetThread gets a thread by its ID.
func (gd *GiDelve) GetThread(id int) (*cdebug.Thread, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ds, err := gd.dlv.GetThread(id)
	gd.LogErr(err)
	return gd.cvtThread(ds), err
}

// ListTasks lists all goroutines.
func (gd *GiDelve) ListTasks() ([]*cdebug.Task, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ds, _, err := gd.dlv.ListGoroutines(0, 1000)
	gd.LogErr(err)
	return gd.cvtTasks(ds), err
}

// Stack returns stacktrace
func (gd *GiDelve) Stack(goroutineID int, depth int) ([]*cdebug.Frame, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ds, err := gd.dlv.Stacktrace(int64(goroutineID), depth, api.StacktraceSimple, nil)
	gd.LogErr(err)
	return gd.cvtStack(ds, goroutineID), err
}

// ListGlobalVars lists all package variables in the context of the current thread.
func (gd *GiDelve) ListGlobalVars(filter string) ([]*cdebug.Variable, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	lc := gd.toLoadConfig(&gd.params.VarList)
	ds, err := gd.dlv.ListPackageVariables(filter, *lc)
	gd.LogErr(err)
	cv := gd.cvtVars(ds)
	// now we have to fill in the pointers here
	gd.fixVarList(cv, gd.lastEvalScope, lc)
	gd.LogErr(err)
	return cv, err
}

// ListVars lists all local variables in scope, including args
func (gd *GiDelve) ListVars(threadID int, frame int) ([]*cdebug.Variable, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ec := gd.toEvalScope(threadID, frame)
	gd.lastEvalScope = ec
	lc := gd.toLoadConfig(&gd.params.VarList)
	vs, err := gd.dlv.ListLocalVariables(*ec, *lc)
	gd.LogErr(err)
	as, err := gd.dlv.ListFunctionArgs(*ec, *lc)
	gd.LogErr(err)
	cv := gd.cvtVars(vs)
	ca := gd.cvtVars(as)
	cv = append(cv, ca...)
	cdebug.SortVars(cv)
	// now we have to fill in the pointers here
	gd.fixVarList(cv, ec, lc)
	gd.LogErr(err)
	return cv, err
}

// GetVariable returns a variable based on expression in the context of the current thread.
func (gd *GiDelve) GetVar(expr string, threadID int, frame int) (*cdebug.Variable, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ec := gd.toEvalScope(threadID, frame)
	gd.lastEvalScope = ec
	lc := gd.toLoadConfig(&gd.params.GetVar)
	expr = quotePkgPaths(expr)
	ds, err := gd.dlv.EvalVariable(*ec, expr, *lc)
	gd.LogErr(err)
	if err != nil {
		return nil, err
	}
	vr := gd.cvtVar(ds)
	gd.fixVar(vr, ec, lc)
	return vr, err
}

// FollowPtr fills in the Child of given Variable
// with retrieved value.
func (gd *GiDelve) FollowPtr(vr *cdebug.Variable) error {
	if err := gd.StartedCheck(); err != nil {
		return err
	}
	if gd.lastEvalScope == nil {
		return fmt.Errorf("FollowPtr: no previous eval scope")
	}
	expr := ""
	addr := vr.Addr
	if vr.FullTypeStr[0] != '*' {
		expr = fmt.Sprintf("(*%q)(%#x)", vr.FullTypeStr, vr.Addr)
	} else {
		expr = fmt.Sprintf("(%q)(%#x)", vr.FullTypeStr, vr.Addr)
	}
	// fmt.Printf("expr: %s\n", expr)
	ch, err := gd.GetVar(expr, int(gd.lastEvalScope.GoroutineID), gd.lastEvalScope.Frame)
	if err == nil {
		vr.CopyFrom(ch)
		vr.Addr = addr
	} else {
		gd.LogErr(err)
	}
	return err
}

// SetVar sets the value of a variable
func (gd *GiDelve) SetVar(name, value string, threadID int, frame int) error {
	if err := gd.StartedCheck(); err != nil {
		return err
	}
	ec := gd.toEvalScope(threadID, frame)
	err := gd.dlv.SetVariable(*ec, name, value)
	return gd.LogErr(err)
}

// ListSources lists all source files in the process matching filter.
func (gd *GiDelve) ListSources(filter string) ([]string, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ds, err := gd.dlv.ListSources(filter)
	gd.LogErr(err)
	return ds, err
}

// ListFuncs lists all functions in the process matching filter.
func (gd *GiDelve) ListFuncs(filter string) ([]string, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ds, err := gd.dlv.ListFunctions(filter)
	gd.LogErr(err)
	return ds, err
}

// ListTypes lists all types in the process matching filter.
func (gd *GiDelve) ListTypes(filter string) ([]string, error) {
	if err := gd.StartedCheck(); err != nil {
		return nil, err
	}
	ds, err := gd.dlv.ListTypes(filter)
	gd.LogErr(err)
	return ds, err
}

// Returns whether we attached to a running process or not
func (gd *GiDelve) AttachedToExistingProcess() bool {
	if err := gd.StartedCheck(); err != nil {
		return false
	}
	ds := gd.dlv.AttachedToExistingProcess()
	return ds
}
