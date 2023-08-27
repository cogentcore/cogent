// Copyright (c) 2020, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidelve

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/go-delve/delve/service/api"
	"github.com/goki/gi/giv"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/syms"
	"goki.dev/gide/gidebug"
)

func (gd *GiDelve) cvtState(ds *api.DebuggerState) *gidebug.State {
	if ds == nil {
		return nil
	}
	st := &gidebug.State{}
	st.Running = ds.Running
	th := gd.cvtThread(ds.CurrentThread)
	if th != nil {
		st.Thread = *th
	}
	gr := gd.cvtTask(ds.SelectedGoroutine)
	if gr != nil {
		st.Task = *gr
	}
	st.NextUp = ds.NextInProgress
	st.Exited = ds.Exited
	st.ExitStatus = ds.ExitStatus
	st.Err = ds.Err
	return st
}

func (gd *GiDelve) cvtThread(ds *api.Thread) *gidebug.Thread {
	if ds == nil {
		return nil
	}
	th := &gidebug.Thread{}
	th.ID = ds.ID
	th.PC = ds.PC
	th.File = giv.RelFilePath(ds.File, gd.rootPath)
	th.Line = ds.Line
	th.FPath = ds.File
	if ds.Function != nil {
		th.Func = ds.Function.Name_
	}
	th.Task = int(ds.GoroutineID)
	return th
}

func (gd *GiDelve) cvtThreads(ds []*api.Thread) []*gidebug.Thread {
	if ds == nil || len(ds) == 0 {
		return nil
	}
	nd := len(ds)
	th := make([]*gidebug.Thread, nd)
	for i, dt := range ds {
		th[i] = gd.cvtThread(dt)
	}
	return th
}

func (gd *GiDelve) cvtTask(ds *api.Goroutine) *gidebug.Task {
	if ds == nil {
		return nil
	}
	gr := &gidebug.Task{}
	gr.ID = int(ds.ID)
	gr.PC = ds.UserCurrentLoc.PC
	gr.File = giv.RelFilePath(ds.UserCurrentLoc.File, gd.rootPath)
	gr.Line = ds.UserCurrentLoc.Line
	gr.FPath = ds.UserCurrentLoc.File
	if ds.UserCurrentLoc.Function != nil {
		gr.Func = ds.UserCurrentLoc.Function.Name_
	}
	gr.Thread = ds.ThreadID
	gr.LaunchLoc = *gd.cvtLocation(&ds.GoStatementLoc)
	gr.StartLoc = *gd.cvtLocation(&ds.StartLoc)
	return gr
}

func (gd *GiDelve) cvtTasks(ds []*api.Goroutine) []*gidebug.Task {
	if ds == nil || len(ds) == 0 {
		return nil
	}
	nd := len(ds)
	th := make([]*gidebug.Task, nd)
	for i, dt := range ds {
		th[i] = gd.cvtTask(dt)
	}
	return th
}

func (gd *GiDelve) cvtLocation(ds *api.Location) *gidebug.Location {
	if ds == nil {
		return nil
	}
	lc := &gidebug.Location{}
	lc.PC = ds.PC
	lc.File = giv.RelFilePath(ds.File, gd.rootPath)
	lc.Line = ds.Line
	lc.FPath = ds.File
	if ds.Function != nil {
		lc.Func = ds.Function.Name_
	}
	return lc
}

func (gd *GiDelve) cvtBreak(ds *api.Breakpoint) *gidebug.Break {
	if ds == nil {
		return nil
	}
	bp := &gidebug.Break{}
	bp.On = true // if we're converting, it is on..
	bp.ID = ds.ID
	bp.PC = ds.Addr
	bp.File = giv.RelFilePath(ds.File, gd.rootPath)
	bp.FPath = ds.File
	bp.Line = ds.Line
	bp.Func = ds.FunctionName
	bp.Cond = ds.Cond
	bp.Trace = ds.Tracepoint
	return bp
}

func (gd *GiDelve) cvtBreaks(ds []*api.Breakpoint) []*gidebug.Break {
	if ds == nil || len(ds) == 0 {
		return nil
	}
	nd := len(ds)
	vr := make([]*gidebug.Break, nd)
	for i := range ds {
		vr[i] = gd.cvtBreak(ds[i])
	}
	return vr
}

func (gd *GiDelve) cvtFrame(ds *api.Stackframe, taskID int) *gidebug.Frame {
	if ds == nil {
		return nil
	}
	fr := &gidebug.Frame{}
	fr.ThreadID = taskID
	fr.PC = ds.Location.PC
	fr.File = giv.RelFilePath(ds.Location.File, gd.rootPath)
	fr.Line = ds.Location.Line
	fr.FPath = ds.Location.File
	if ds.Location.Function != nil {
		fr.Func = ds.Location.Function.Name_
	}
	fr.Vars = gd.cvtVars(ds.Locals)
	fr.Args = gd.cvtVars(ds.Arguments)
	return fr
}

func (gd *GiDelve) cvtStack(ds []api.Stackframe, taskID int) []*gidebug.Frame {
	if ds == nil || len(ds) == 0 {
		return nil
	}
	nd := len(ds)
	vr := make([]*gidebug.Frame, nd)
	for i := range ds {
		vr[i] = gd.cvtFrame(&ds[i], taskID)
		vr[i].Depth = i
	}
	return vr
}

func ShortType(typ string) string {
	si := strings.Index(typ, "/")
	if si < 0 {
		return typ
	}
	tnm := lex.TrimLeftToAlpha(typ)
	tsi := strings.Index(typ, tnm)
	fdir, fnm := filepath.Split(tnm)
	fdd := filepath.Base(fdir)
	return typ[:tsi] + fdd + "/" + fnm
}

func (gd *GiDelve) cvtVar(ds *api.Variable) *gidebug.Variable {
	if ds == nil {
		return nil
	}
	vr := &gidebug.Variable{}
	vr.InitName(vr, ds.Name)
	vr.Addr = uintptr(ds.Addr)
	vr.FullTypeStr = ds.RealType
	vr.TypeStr = ShortType(ds.RealType)
	if ds.Flags&api.VariableEscaped != 0 {
		vr.Heap = true
	}
	vr.Kind = syms.ReflectKindMap[ds.Kind]
	vr.ElValue = ds.Value
	vr.Value = ds.Value // note: NOT calling vr.ValueString(false, 0)
	vr.Len = ds.Len
	vr.Cap = ds.Cap
	vr.Loc.Line = int(ds.DeclLine)
	vr.Loc.FPath = ds.LocationExpr
	vr.Dbg = gd
	nkids := len(ds.Children)
	switch {
	case nkids == 1 && vr.Kind.IsPtr():
		el := &ds.Children[0]
		if el.Name == "" {
			el.Name = "*" + ds.Name
		}
	case nkids > 0 && vr.Kind.SubCat() == syms.List:
		el := &ds.Children[0]
		elk := syms.ReflectKindMap[el.Kind]
		if elk.IsPrimitiveNonPtr() {
			vr.List = make([]string, nkids)
			for i := range ds.Children {
				vr.List[i] = ds.Children[i].Value
			}
			return vr
		}
	case nkids > 1 && vr.Kind.SubCat() == syms.Map:
		mapn := nkids / 2
		el := &ds.Children[1] // alternates key / value
		elk := syms.ReflectKindMap[el.Kind]
		if elk.IsPrimitiveNonPtr() {
			vr.Map = make(map[string]string, mapn)
			for i := 0; i < mapn; i++ {
				k := &ds.Children[2*i]
				el = &ds.Children[2*i+1]
				vr.Map[k.Value] = el.Value
			}
			return vr
		}
		// object map
		vr.MapVar = make(map[string]*gidebug.Variable, mapn)
		for i := 0; i < mapn; i++ {
			k := &ds.Children[2*i]
			el = &ds.Children[2*i+1]
			vr.MapVar[k.Value] = gd.cvtVar(el)
		}
		return vr
	case nkids > 0 && nkids < 10 && vr.Kind.SubCat() == syms.Struct:
		allPrim := true
		for i := range ds.Children {
			el := &ds.Children[i]
			elk := syms.ReflectKindMap[el.Kind]
			if !elk.IsPrimitiveNonPtr() {
				allPrim = false
				break
			}
		}
		if allPrim {
			vstr := ""
			for i := range ds.Children {
				el := &ds.Children[i]
				vstr += el.Name + ": " + el.Value
				if i < nkids-1 {
					vstr += ", "
				}
			}
			vr.Value = vstr
			vr.ElValue = vstr
			return vr
		}
	}
	for i := range ds.Children {
		el := &ds.Children[i]
		nkv := gd.cvtVar(el)
		if nkv.Nm == "" {
			nkv.SetName(fmt.Sprintf("[%d]", i))
		}
		vr.AddChild(nkv)
	}
	return vr
}

func (gd *GiDelve) cvtVars(ds []api.Variable) []*gidebug.Variable {
	if ds == nil || len(ds) == 0 {
		return nil
	}
	nd := len(ds)
	vr := make([]*gidebug.Variable, nd)
	for i := range ds {
		vr[i] = gd.cvtVar(&ds[i])
	}
	return vr
}

func (gd *GiDelve) fixVarList(cv []*gidebug.Variable, ec *api.EvalScope, lc *api.LoadConfig) {
	for _, vr := range cv {
		gd.fixVar(vr, ec, lc)
	}
}

// trimLeftToAlpha returns string without any leading non-alpha runes
func trimLeftToAlpha(nm string) string {
	return strings.TrimLeftFunc(nm, func(r rune) bool {
		return !unicode.IsLetter(r)
	})
}

// quotePkgPaths puts quotes around a package path
func quotePkgPaths(vnm string) string {
	if strings.Contains(vnm, "/") && !strings.Contains(vnm, `"`) { // unquoted path
		pstr := trimLeftToAlpha(vnm)
		pi := strings.Index(vnm, pstr)
		segs := strings.Split(pstr, "/")
		lseg := segs[len(segs)-1]
		di := strings.Index(lseg, ".")
		post := ""
		if di > 0 {
			post = lseg[di:]
			lseg = lseg[:di]
		}
		pstr = strings.Join(segs[:len(segs)-1], "/")
		pstr += "/" + lseg
		vnm = vnm[:pi] + `"` + pstr + `"` + post
	}
	return vnm
}

func (gd *GiDelve) fixVar(vr *gidebug.Variable, ec *api.EvalScope, lc *api.LoadConfig) {
	if vr.Kind.IsPtr() && vr.NumChildren() == 1 && vr.Nm != "" {
		vrk := vr.Child(0).(*gidebug.Variable)
		if vrk.NumChildren() == 0 && !vrk.Kind.IsPrimitiveNonPtr() {
			vnm := "*" + vr.Nm
			dss, err := gd.dlv.EvalVariable(*ec, vnm, *lc)
			if err == nil {
				vrkr := gd.cvtVar(dss)
				if vrkr.Nm == "" {
					vrkr.SetName(vnm)
				}
				vr.DeleteChildAtIndex(0, true)
				vr.AddChild(vrkr)
			}
		}
	}
	vr.Value = vr.ValueString(false, 0, gd.params.VarList.MaxRecurse, 256, false) // max depth, max len -- short for summary -- no type
}

func (gd *GiDelve) toLoadConfig(ds *gidebug.VarParams) *api.LoadConfig {
	if ds == nil {
		return nil
	}
	lc := &api.LoadConfig{}
	lc.FollowPointers = ds.FollowPointers
	lc.MaxVariableRecurse = ds.MaxRecurse
	lc.MaxStringLen = ds.MaxStringLen
	lc.MaxArrayValues = ds.MaxArrayValues
	lc.MaxStructFields = ds.MaxStructFields
	return lc
}

func (gd *GiDelve) toEvalScope(threadID int, frame int) *api.EvalScope {
	es := &api.EvalScope{}
	es.GoroutineID = int64(threadID)
	es.Frame = frame
	// es.DeferredCall
	return es
}
