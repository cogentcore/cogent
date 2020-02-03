// Copyright (c) 2020, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidelve

import (
	"github.com/go-delve/delve/service/api"
	"github.com/goki/gide/gidebug"
)

func CvtDebuggerState(ds *api.DebuggerState) *gidebug.DebuggerState {
	if ds == nil {
		return nil
	}
	st := &gidebug.DebuggerState{}
	st.Running = ds.Running
	st.CurrentThread = CvtThread(ds.CurrentThread)
	st.SelectedGoroutine = CvtGoroutine(ds.SelectedGoroutine)
	st.Threads = CvtThreads(ds.Threads)
	st.NextInProgress = ds.NextInProgress
	st.Exited = ds.Exited
	st.ExitStatus = ds.ExitStatus
	st.When = ds.When
	st.Err = ds.Err
	return st
}

func CvtThread(ds *api.Thread) *gidebug.Thread {
	if ds == nil {
		return nil
	}
	th := &gidebug.Thread{}
	th.ID = ds.ID
	th.PC = ds.PC
	th.File = ds.File
	th.Line = ds.Line
	th.Function = CvtFunction(ds.Function)
	th.GoroutineID = ds.GoroutineID
	th.Breakpoint = CvtBreakpoint(ds.Breakpoint)
	th.BreakpointInfo = CvtBreakpointInfo(ds.BreakpointInfo)
	th.ReturnValues = CvtVariables(ds.ReturnValues)
	return th
}

func CvtThreads(ds []*api.Thread) []*gidebug.Thread {
	if ds == nil || len(ds) == 0 {
		return nil
	}
	nd := len(ds)
	th := make([]*gidebug.Thread, nd)
	for i, dt := range ds {
		th[i] = CvtThread(dt)
	}
	return th
}

func CvtGoroutine(ds *api.Goroutine) *gidebug.Goroutine {
	if ds == nil {
		return nil
	}
	gr := &gidebug.Goroutine{}
	gr.ID = ds.ID
	gr.CurrentLoc = *CvtLocation(&ds.CurrentLoc)
	gr.UserCurrentLoc = *CvtLocation(&ds.UserCurrentLoc)
	gr.GoStatementLoc = *CvtLocation(&ds.GoStatementLoc)
	gr.StartLoc = *CvtLocation(&ds.StartLoc)
	gr.ThreadID = ds.ThreadID
	gr.Unreadable = ds.Unreadable
	return gr
}

func CvtGoroutines(ds []*api.Goroutine) []*gidebug.Goroutine {
	if ds == nil || len(ds) == 0 {
		return nil
	}
	nd := len(ds)
	th := make([]*gidebug.Goroutine, nd)
	for i, dt := range ds {
		th[i] = CvtGoroutine(dt)
	}
	return th
}

func CvtLocation(ds *api.Location) *gidebug.Location {
	if ds == nil {
		return nil
	}
	lc := &gidebug.Location{}
	lc.PC = ds.PC
	lc.File = ds.File
	lc.Line = ds.Line
	lc.Function = CvtFunction(ds.Function)
	lc.PCs = ds.PCs
	return lc
}

func CvtLocations(ds []api.Location) []*gidebug.Location {
	if ds == nil || len(ds) == 0 {
		return nil
	}
	nd := len(ds)
	th := make([]*gidebug.Location, nd)
	for i := range ds {
		th[i] = CvtLocation(&ds[i])
	}
	return th
}

func CvtFunction(ds *api.Function) *gidebug.Function {
	if ds == nil {
		return nil
	}
	fc := &gidebug.Function{}
	fc.Name_ = ds.Name_
	fc.Value = ds.Value
	fc.Type = ds.Type
	fc.GoType = ds.GoType
	fc.Optimized = ds.Optimized
	return fc
}

func CvtBreakpoint(ds *api.Breakpoint) *gidebug.Breakpoint {
	if ds == nil {
		return nil
	}
	bp := &gidebug.Breakpoint{}
	bp.ID = ds.ID
	bp.Name = ds.Name
	bp.Addr = ds.Addr
	bp.Addrs = ds.Addrs
	bp.File = ds.File
	bp.Line = ds.Line
	bp.FunctionName = ds.FunctionName
	bp.Cond = ds.Cond
	bp.Tracepoint = ds.Tracepoint
	bp.TraceReturn = ds.TraceReturn
	bp.Goroutine = ds.Goroutine
	bp.Stacktrace = ds.Stacktrace
	bp.Variables = ds.Variables
	bp.LoadArgs = CvtLoadConfig(ds.LoadArgs)
	bp.LoadLocals = CvtLoadConfig(ds.LoadLocals)
	bp.HitCount = ds.HitCount
	bp.TotalHitCount = ds.TotalHitCount
	return bp
}

func ToBreakpoint(ds *gidebug.Breakpoint) *api.Breakpoint {
	if ds == nil {
		return nil
	}
	bp := &api.Breakpoint{}
	bp.ID = ds.ID
	bp.Name = ds.Name
	bp.Addr = ds.Addr
	bp.Addrs = ds.Addrs
	bp.File = ds.File
	bp.Line = ds.Line
	bp.FunctionName = ds.FunctionName
	bp.Cond = ds.Cond
	bp.Tracepoint = ds.Tracepoint
	bp.TraceReturn = ds.TraceReturn
	bp.Goroutine = ds.Goroutine
	return bp
}

func CvtBreakpoints(ds []*api.Breakpoint) []*gidebug.Breakpoint {
	if ds == nil || len(ds) == 0 {
		return nil
	}
	nd := len(ds)
	vr := make([]*gidebug.Breakpoint, nd)
	for i := range ds {
		vr[i] = CvtBreakpoint(ds[i])
	}
	return vr
}

func CvtBreakpointInfo(ds *api.BreakpointInfo) *gidebug.BreakpointInfo {
	if ds == nil {
		return nil
	}
	bp := &gidebug.BreakpointInfo{}
	bp.Stacktrace = CvtStackframes(ds.Stacktrace)
	bp.Goroutine = CvtGoroutine(ds.Goroutine)
	bp.Variables = CvtVariables(ds.Variables)
	bp.Arguments = CvtVariables(ds.Arguments)
	bp.Locals = CvtVariables(ds.Locals)
	return bp
}

func CvtStackframe(ds *api.Stackframe) *gidebug.Stackframe {
	if ds == nil {
		return nil
	}
	fr := &gidebug.Stackframe{}
	fr.Location = *CvtLocation(&ds.Location)
	fr.Locals = CvtVariables(ds.Locals)
	fr.Arguments = CvtVariables(ds.Arguments)
	fr.FrameOffset = ds.FrameOffset
	fr.FramePointerOffset = ds.FramePointerOffset
	//	fr.Defers = CvtDefers(ds.Defers)
	fr.Bottom = ds.Bottom
	fr.Err = ds.Err
	return fr
}

func CvtStackframes(ds []api.Stackframe) []*gidebug.Stackframe {
	if ds == nil || len(ds) == 0 {
		return nil
	}
	nd := len(ds)
	vr := make([]*gidebug.Stackframe, nd)
	for i := range ds {
		vr[i] = CvtStackframe(&ds[i])
	}
	return vr
}

func CvtVariable(ds *api.Variable) *gidebug.Variable {
	if ds == nil {
		return nil
	}
	vr := &gidebug.Variable{}
	vr.Name = ds.Name
	vr.Addr = ds.Addr
	vr.OnlyAddr = ds.OnlyAddr
	vr.Type = ds.Type
	vr.RealType = ds.RealType
	vr.Flags = gidebug.VariableFlags(ds.Flags)
	vr.Kind = ds.Kind
	vr.Value = ds.Value
	vr.Len = ds.Len
	vr.Cap = ds.Cap
	vr.Children = CvtVariables(ds.Children)
	vr.Base = ds.Base
	vr.Unreadable = ds.Unreadable
	vr.LocationExpr = ds.LocationExpr
	vr.DeclLine = ds.DeclLine
	return vr
}

func CvtVariables(ds []api.Variable) []*gidebug.Variable {
	if ds == nil || len(ds) == 0 {
		return nil
	}
	nd := len(ds)
	vr := make([]*gidebug.Variable, nd)
	for i := range ds {
		vr[i] = CvtVariable(&ds[i])
	}
	return vr
}

func CvtLoadConfig(ds *api.LoadConfig) *gidebug.LoadConfig {
	if ds == nil {
		return nil
	}
	lc := &gidebug.LoadConfig{}
	lc.FollowPointers = ds.FollowPointers
	lc.MaxVariableRecurse = ds.MaxVariableRecurse
	lc.MaxStringLen = ds.MaxStringLen
	lc.MaxArrayValues = ds.MaxArrayValues
	lc.MaxStructFields = ds.MaxStructFields
	return lc
}

func ToLoadConfig(ds *gidebug.LoadConfig) *api.LoadConfig {
	if ds == nil {
		return nil
	}
	lc := &api.LoadConfig{}
	lc.FollowPointers = ds.FollowPointers
	lc.MaxVariableRecurse = ds.MaxVariableRecurse
	lc.MaxStringLen = ds.MaxStringLen
	lc.MaxArrayValues = ds.MaxArrayValues
	lc.MaxStructFields = ds.MaxStructFields
	return lc
}

func CvtDebuggerStateChan(in <-chan *api.DebuggerState) <-chan *gidebug.DebuggerState {
	sc := make(chan *gidebug.DebuggerState)
	go func() {
		nv := <-in
		sc <- CvtDebuggerState(nv)
	}()
	return sc
}

func ToEvalScope(ds *gidebug.EvalScope) *api.EvalScope {
	if ds == nil {
		return nil
	}
	es := &api.EvalScope{}
	es.GoroutineID = ds.GoroutineID
	es.Frame = ds.Frame
	es.DeferredCall = ds.DeferredCall
	return es
}
