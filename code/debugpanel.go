// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"image"
	"log"
	"slices"
	"strings"
	"time"

	"cogentcore.org/cogent/code/cdebug"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/cam/hct"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/tree"
)

// DebugBreakStatus represents the status of a certain breakpoint.
type DebugBreakStatus int32 //enums:enum -trim-prefix Debug

const (
	// DebugBreakInactive is an inactive break point
	DebugBreakInactive DebugBreakStatus = iota

	// DebugBreakActive is an active break point
	DebugBreakActive

	// DebugBreakCurrent is the current break point
	DebugBreakCurrent

	// DebugPCCurrent is the current program execution point,
	// updated for every ShowFile action
	DebugPCCurrent
)

const (
	DebugTabConsole = "Console"
	DebugTabBreaks  = "Breaks"
	DebugTabStack   = "Stack"
	DebugTabTasks   = "Tasks"
	DebugTabThreads = "Threads"
	DebugTabVars    = "Vars"
	DebugTabFrames  = "Find Frames"
	DebugTabGlobals = "Global Vars"
)

var (
	// DebugBreakColors are the colors indicating different breakpoint statuses.
	DebugBreakColors = map[DebugBreakStatus]image.Image{
		DebugBreakInactive: colors.Scheme.Warn.Base,
		DebugBreakActive:   colors.Scheme.Error.Base,
		DebugBreakCurrent:  colors.Scheme.Success.Base,
		DebugPCCurrent:     colors.Scheme.Primary.Base,
	}
)

// NewDebugger returns a new debugger for given supported file type
func NewDebugger(sup fileinfo.Known, path, rootPath string, outbuf *lines.Lines, pars *cdebug.Params) (cdebug.GiDebug, error) {
	df, ok := cdebug.Debuggers[sup]
	if !ok {
		err := fmt.Errorf("Code Debug: File type %v not supported -- change the MainLang in File/Project Settings.. to a supported language (Go only option so far)", sup)
		log.Println(err)
		return nil, err
	}
	dbg, err := df(path, rootPath, outbuf, pars)
	if err != nil {
		log.Println(err)
	}
	return dbg, err
}

// DebugPanel is the debugger panel.
type DebugPanel struct {
	core.Frame

	// known file type to determine debugger
	Known fileinfo.Known

	// path to executable / dir to debug
	ExePath string

	// time when dbg was last restarted
	DbgTime time.Time

	// the debugger
	Dbg cdebug.GiDebug `set:"-" json:"-" xml:"-"`

	// all relevant debug state info
	State cdebug.AllState `set:"-" json:"-" xml:"-"`

	// current ShowFile location -- cleared before next one or run
	CurFileLoc cdebug.Location `set:"-" json:"-" xml:"-"`

	// backup breakpoints list -- to track deletes
	BBreaks []*cdebug.Break `set:"-" json:"-" xml:"-"`

	// output from the debugger
	OutputBuffer *lines.Lines `set:"-" json:"-" xml:"-"`

	// parent code project
	Code *Code `set:"-" json:"-" xml:"-"`
}

// Config sets parameters that must be set for a new view
func (dv *DebugPanel) Config(cv *Code, known fileinfo.Known, exePath string) {
	dv.Code = cv
	dv.Known = known
	dv.ExePath = exePath
}

func (dv *DebugPanel) Init() {
	dv.Frame.Init()
	dv.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	tree.AddChildAt(dv, "toolbar", func(w *core.Frame) {
		core.ToolbarStyles(w)
		w.Maker(dv.MakeToolbar)
	})
	tree.AddChildAt(dv, "tabs", func(w *core.Tabs) {
		dv.Updater(dv.InitTabs) // note: this is necessary to allow config to happen first
	})
}

func (dv *DebugPanel) InitTabs() {
	w := dv.Tabs()
	if w.NumTabs() > 0 {
		return
	}
	dtab, _ := w.NewTab(DebugTabConsole)
	ctv := textcore.NewEditor(dtab)
	ctv.SetName("dbg-console")
	ConfigOutputTextEditor(ctv)
	dv.State.BlankState()
	dv.OutputBuffer = lines.NewLines()
	dv.OutputBuffer.SetFilename("debug-outbuf")
	dv.OutputBuffer.SetReadOnly(true)
	dv.State.Breaks = nil // get rid of dummy
	dv.OutputBuffer.Settings.LineNumbers = false
	ctv.SetLines(dv.OutputBuffer)

	bv, _ := w.NewTab(DebugTabBreaks)
	tree.AddChild(bv, func(w *core.Table) {
		w.SetSlice(&dv.State.Breaks)
		w.OnDoubleClick(func(e events.Event) {
			idx := w.SelectedIndex
			dv.ShowBreakFile(idx)
		})
		w.OnChange(func(e events.Event) {
			dv.SyncBreaks()
		})
		w.Updater(func() {
			if dv.State.CurBreak > 0 {
				_, idx := cdebug.BreakByID(dv.State.Breaks, dv.State.CurBreak)
				if idx >= 0 {
					w.SelectedIndex = idx
				}
			}
			dv.BackupBreaks()
		})
	})

	sv, _ := w.NewTab(DebugTabStack)
	tree.AddChild(sv, func(w *core.Table) {
		w.SetReadOnly(true)
		w.SetSlice(&dv.State.Stack)
		w.OnDoubleClick(func(e events.Event) {
			dv.SetFrame(w.SelectedIndex)
		})
		w.Updater(func() {
			w.SelectedIndex = dv.State.CurFrame
		})
	})

	if dv.Known == fileinfo.Go { // dv.Dbg.HasTasks() { // todo: not avail here yet
		tv, _ := w.NewTab(DebugTabTasks)
		tree.AddChild(tv, func(w *core.Table) {
			w.SetReadOnly(true)
			w.SetSlice(&dv.State.Tasks)
			w.OnDoubleClick(func(e events.Event) {
				if dv.Dbg != nil && dv.Dbg.HasTasks() {
					dv.SetThreadIndex(w.SelectedIndex)
				}
			})
			w.Updater(func() {
				_, idx := cdebug.TaskByID(dv.State.Tasks, dv.State.CurTask)
				if idx >= 0 {
					w.SelectedIndex = idx
				}
			})
		})
	}

	tv, _ := w.NewTab(DebugTabThreads)
	tree.AddChild(tv, func(w *core.Table) {
		w.SetReadOnly(true)
		w.SetSlice(&dv.State.Threads)
		w.OnDoubleClick(func(e events.Event) {
			if dv.Dbg != nil && !dv.Dbg.HasTasks() {
				dv.SetThreadIndex(w.SelectedIndex)
			}
		})
		w.Updater(func() {
			_, idx := cdebug.ThreadByID(dv.State.Threads, dv.State.CurThread)
			if idx >= 0 {
				w.SelectedIndex = idx
			}
		})
	})

	vv, _ := w.NewTab(DebugTabVars)
	tree.AddChild(vv, func(w *core.Table) {
		w.SetReadOnly(true)
		w.SetSlice(&dv.State.Vars)
		w.OnDoubleClick(func(e events.Event) {
			vr := dv.State.Vars[w.SelectedIndex]
			dv.ShowVar(vr.Name)
		})
	})

	ff, _ := w.NewTab(DebugTabFrames)
	tree.AddChild(ff, func(w *core.Table) {
		w.SetReadOnly(true)
		w.SetSlice(&dv.State.FindFrames)
		w.OnDoubleClick(func(e events.Event) {
			idx := w.SelectedIndex
			if idx >= 0 && idx < len(dv.State.FindFrames) {
				fr := dv.State.FindFrames[idx]
				dv.SetThread(fr.ThreadID)
			}
		})
	})

	gv, _ := w.NewTab(DebugTabGlobals)
	tree.AddChild(gv, func(w *core.Table) {
		w.SetReadOnly(true)
		w.SetSlice(&dv.State.GlobalVars)
		w.OnDoubleClick(func(e events.Event) {
			vr := dv.State.Vars[w.SelectedIndex]
			dv.ShowVar(vr.Name)
		})
	})
}

// panelIsValid checks if panel is valid for use.
// for outside functions calling in, e.g., textbuf.
func (dv *DebugPanel) panelIsValid() bool {
	return !(dv.This == nil || dv.Code == nil || dv.Code.This == nil || dv.Tabs() == nil)
}

// DbgIsActive means debugger is started.
func (dv *DebugPanel) DbgIsActive() bool {
	return dv.Dbg != nil && dv.Dbg.IsActive()
}

// DbgIsAvail means the debugger is started AND process is not currently running --
// it is available for command input.
func (dv *DebugPanel) DbgIsAvail() bool {
	if !dv.DbgIsActive() {
		return false
	}
	if dv.State.State.Running {
		return false
	}
	return true
}

// DbgCanStep means the debugger is started AND process is not currently running,
// AND it is not already waiting for a next step
func (dv *DebugPanel) DbgCanStep() bool {
	if !dv.DbgIsAvail() {
		return false
	}
	if dv.State.State.NextUp {
		return false
	}
	return true
}

func (dv *DebugPanel) Destroy() {
	dv.Detach()
	dv.DeleteAllBreaks()
	dv.DeleteCurPCInBuf()
	dv.Code.ClearDebug()
	dv.Frame.Destroy()
}

// Detach from debugger
func (dv *DebugPanel) Detach() {
	killProc := true
	if dv.State.Mode == cdebug.Attach {
		killProc = false
	}
	if dv.DbgIsAvail() {
		dv.Dbg.Detach(killProc)
	} else if dv.DbgIsActive() {
		dv.Stop()
		dv.Dbg.Detach(killProc)
	}
	dv.Dbg = nil
}

// Start starts the debuger
func (dv *DebugPanel) Start() {
	if dv.Code == nil {
		return
	}
	console := dv.ConsoleText()
	console.Clear()
	rebuild := false
	if dv.Dbg != nil && dv.State.Mode != cdebug.Attach {
		lmod := dv.Code.Files.LatestFileMod(fileinfo.Code)
		rebuild = lmod.After(dv.DbgTime) || dv.Code.LastSaveTStamp.After(dv.DbgTime)
	}
	if dv.Dbg == nil || rebuild {
		dv.SetStatus(cdebug.Building)
		if dv.Dbg != nil {
			dv.Detach()
		}
		rootPath := string(dv.Code.Settings.ProjectRoot)
		pars := &dv.Code.Settings.Debug
		dv.State.Mode = pars.Mode
		pars.StatFunc = func(stat cdebug.Status) {
			dv.AsyncLock()

			if stat == cdebug.Ready && dv.State.Mode == cdebug.Attach {
				dv.UpdateFromState()
			}
			dv.SetStatus(stat)
			if stat == cdebug.Error {
				dv.Dbg = nil
			}
			dv.NeedsLayout()
			dv.AsyncUnlock()
		}
		dbg, err := NewDebugger(dv.Known, dv.ExePath, rootPath, dv.OutputBuffer, pars)
		if err == nil {
			dv.Dbg = dbg
			dv.DbgTime = time.Now()
		} else {
			dv.SetStatus(cdebug.Error)
		}
	} else {
		dv.Dbg.Restart()
		dv.SetStatus(cdebug.Ready)
	}
}

// UpdateView updates current view of state
func (dv *DebugPanel) UpdateView() {
	ds, err := dv.Dbg.GetState()
	if err != nil {
		return
	}
	dv.InitState(ds)
}

// Continue continues running from current point -- this MUST be called
// in a separate goroutine!
func (dv *DebugPanel) Continue() {
	if !dv.DbgIsAvail() {
		return
	}
	dv.SetBreaks()
	dv.State.State.Running = true
	dv.SetStatus(cdebug.Running)
	dsc := dv.Dbg.Continue(&dv.State)
	var ds *cdebug.State
	for ds = range dsc { // get everything
		if dv.This == nil {
			return
		}
	}
	if dv.Code != nil {
		dv.Code.Events().RenderWindow().Raise()
	}
	if ds != nil {
		dv.InitState(ds)
	} else {
		dv.State.State.Running = false
		dv.SetStatus(cdebug.Finished)
	}
}

// StepOver continues to the next source line, not entering function calls.
func (dv *DebugPanel) StepOver() { //types:add
	if !dv.DbgCanStep() {
		return
	}
	dv.SetBreaks()
	ds, err := dv.Dbg.StepOver()
	if err != nil {
		return
	}
	dv.InitState(ds)
}

// StepInto continues to the next source line, entering function calls.
func (dv *DebugPanel) StepInto() { //types:add
	if !dv.DbgCanStep() {
		return
	}
	dv.SetBreaks()
	ds, err := dv.Dbg.StepInto()
	if err != nil {
		return
	}
	dv.InitState(ds)
}

// StepOut continues to the return point of the current function.
func (dv *DebugPanel) StepOut() { //types:add
	if !dv.DbgCanStep() {
		return
	}
	dv.SetBreaks()
	ds, err := dv.Dbg.StepOut()
	if err != nil {
		return
	}
	dv.InitState(ds)
}

// StepSingle steps a single CPU instruction.
func (dv *DebugPanel) SingleStep() { //types:add
	if !dv.DbgCanStep() {
		return
	}
	dv.SetBreaks()
	ds, err := dv.Dbg.StepSingle()
	if err != nil {
		return
	}
	dv.InitState(ds)
}

// Stop stops a running process.
func (dv *DebugPanel) Stop() { //types:add
	// if !dv.DbgIsActive() || dv.DbgIsAvail() {
	// 	return
	// }
	if dv.Dbg == nil {
		return
	}
	_, err := dv.Dbg.Stop()
	if err != nil {
		return
	}
	// note: it will auto update from continue stopping.
}

// SetBreaks sets the current breakpoints from State, call this prior to running
func (dv *DebugPanel) SetBreaks() {
	if !dv.DbgIsAvail() {
		return
	}
	dv.DeleteCurPCInBuf()
	dv.State.CurBreak = 0 // reset
	dv.Dbg.UpdateBreaks(&dv.State.Breaks)
	dv.UpdateAllBreaks()
}

// AddBreak adds a breakpoint at given file path and line number.
// note: all breakpoints are just set in our master list and
// uploaded to the system right before starting running.
func (dv *DebugPanel) AddBreak(fpath string, line int) {
	dv.State.AddBreak(fpath, line)
	dv.BackupBreaks()
	dv.ShowTab(DebugTabBreaks)
	dv.UpdateTab(DebugTabBreaks)
}

// DeleteBreak deletes given breakpoint.  If debugger is not yet
// activated then it just deletes from master list.
// Note that breakpoints can be turned on and off directly using On flag.
func (dv *DebugPanel) DeleteBreak(fpath string, line int) {
	if dv.This == nil {
		return
	}
	dv.DeleteBreakImpl(fpath, line)
	dv.BackupBreaks()
	dv.ShowTab(DebugTabBreaks)
	dv.UpdateTab(DebugTabBreaks)
}

// DeleteBreakImpl deletes given breakpoint with no other updates
func (dv *DebugPanel) DeleteBreakImpl(fpath string, line int) {
	if !dv.DbgIsAvail() {
		dv.State.DeleteBreakByFile(fpath, line) // already doing this!
		return
	}
	bk, _ := dv.State.BreakByFile(fpath, line)
	if bk != nil {
		dv.Dbg.ClearBreak(bk.ID)
		dv.State.DeleteBreakByID(bk.ID)
	}
}

// DeleteBreakIndex deletes break at given index in list of breaks
func (dv *DebugPanel) DeleteBreakIndex(bidx int) {
	if bidx < 0 || bidx >= len(dv.BBreaks) {
		return
	}
	bk := dv.BBreaks[bidx]
	dv.DeleteBreakInBuf(bk.FPath, bk.Line)
	if !dv.DbgIsAvail() {
		dv.State.DeleteBreakByFile(bk.FPath, bk.Line)
		return
	}
	dv.Dbg.ClearBreak(bk.ID)
	dv.State.DeleteBreakByID(bk.ID)
	dv.BBreaks = slices.Delete(dv.BBreaks, bidx, bidx+1)
}

// SyncBreaks synchronizes backup breaks with current breaks, after Breaks Changed
func (dv *DebugPanel) SyncBreaks() {
	if len(dv.State.Breaks) == len(dv.BBreaks) {
		return
	}
	for i, b := range dv.State.Breaks {
		nb := len(dv.BBreaks)
		if i >= nb && nb > 0 {
			dv.DeleteBreakIndex(nb - 1)
		} else if i < nb && dv.BBreaks[i] != b {
			dv.DeleteBreakIndex(i)
		}
	}
}

// DeleteBreakInBuf delete breakpoint in its TextBuf
// line is 1-based line number
func (dv *DebugPanel) DeleteBreakInBuf(fpath string, line int) {
	if !dv.panelIsValid() {
		return
	}
	tb := dv.Code.GetOpenFile(fpath)
	if tb != nil {
		tb.DeleteLineColor(line - 1)
		if ed, _, ok := dv.Code.EditorForLines(tb); ok {
			ed.NeedsRender()
		}
	}
}

// DeleteAllBreaks deletes all breakpoints
func (dv *DebugPanel) DeleteAllBreaks() {
	if !dv.panelIsValid() {
		return
	}
	for _, bk := range dv.State.Breaks {
		dv.DeleteBreakInBuf(bk.FPath, bk.Line)
	}
}

// UpdateBreakInBuf updates break status in its TextBuf
// line is 1-based line number
func (dv *DebugPanel) UpdateBreakInBuf(fpath string, line int, stat DebugBreakStatus) {
	if !dv.panelIsValid() {
		return
	}
	tb := dv.Code.GetOpenFile(fpath)
	if tb != nil {
		tb.SetLineColor(line-1, DebugBreakColors[stat])
		if ed, _, ok := dv.Code.EditorForLines(tb); ok {
			ed.NeedsRender()
		}
	}
}

// UpdateAllBreaks updates all breakpoints
func (dv *DebugPanel) UpdateAllBreaks() {
	if !dv.panelIsValid() {
		return
	}
	for _, bk := range dv.State.Breaks {
		if bk.ID == dv.State.CurBreak {
			dv.UpdateBreakInBuf(bk.FPath, bk.Line, DebugBreakCurrent)
		} else if bk.On {
			dv.UpdateBreakInBuf(bk.FPath, bk.Line, DebugBreakActive)
		} else {
			dv.UpdateBreakInBuf(bk.FPath, bk.Line, DebugBreakInactive)
		}
	}
	dv.UpdateTab(DebugTabBreaks)
	dv.NeedsRender()
}

// BackupBreaks makes a backup copy of current breaks
func (dv *DebugPanel) BackupBreaks() {
	dv.BBreaks = make([]*cdebug.Break, len(dv.State.Breaks))
	for i, b := range dv.State.Breaks {
		dv.BBreaks[i] = b
	}
}

// InitState updates the State and View from given debug state
// Call this when debugger returns from any action update
func (dv *DebugPanel) InitState(ds *cdebug.State) {
	dv.State.State = *ds
	if ds.Running {
		return
	}
	if ds.Exited {
		dv.SetStatus(cdebug.Finished)
	} else {
		dv.SetStatus(cdebug.Stopped)
	}
	err := dv.Dbg.InitAllState(&dv.State)
	if err == cdebug.IsRunningErr {
		dv.SetStatus(cdebug.Running)
		return
	}
	dv.UpdateFromState()
}

// UpdateFromState updates the view from current debugger state
func (dv *DebugPanel) UpdateFromState() {
	if dv == nil || dv.This == nil || dv.Dbg == nil || !dv.HasChildren() {
		return
	}

	cb, err := dv.Dbg.ListBreaks()
	if err == nil {
		dv.State.CurBreaks = cb
		dv.State.MergeBreaks()
	}
	cf := dv.State.StackFrame(dv.State.CurFrame)
	if cf != nil {
		dv.ShowFile(cf.FPath, cf.Line)
		if dv.State.CurBreak > 0 {
			dv.SetStatus(cdebug.Breakpoint)
		}
	}
	dv.UpdateAllBreaks()
	dv.Update()
}

// SetFrame sets the given frame depth level as active
func (dv *DebugPanel) SetFrame(depth int) {
	if !dv.DbgIsAvail() {
		return
	}
	cf := dv.State.StackFrame(depth)
	if cf != nil {
		dv.Dbg.UpdateAllState(&dv.State, dv.State.CurTask, depth) // todo: CurTask is not general!
	}
	dv.UpdateFromState()
}

// SetThread sets the given thread as active -- this must be TaskID if HasTasks
// and ThreadID if not.
func (dv *DebugPanel) SetThread(threadID int) {
	if !dv.DbgIsAvail() {
		return
	}
	dv.Dbg.UpdateAllState(&dv.State, threadID, 0)
	dv.UpdateFromState()
}

// SetThreadIndex sets the given thread by index in threads list as active
// this must be TaskID if HasTasks and ThreadID if not.
func (dv *DebugPanel) SetThreadIndex(thridx int) {
	if !dv.DbgIsAvail() || thridx < 0 {
		return
	}
	thid := 0
	if dv.Dbg.HasTasks() {
		if thridx >= len(dv.State.Tasks) {
			return
		}
		th := dv.State.Tasks[thridx]
		thid = th.ID
	} else {
		if thridx >= len(dv.State.Threads) {
			return
		}
		th := dv.State.Threads[thridx]
		thid = th.ID
	}
	dv.Dbg.UpdateAllState(&dv.State, thid, 0)
	dv.UpdateFromState()
}

// FindFrames finds the frames where given file and line are active
// Selects the one that is closest and shows the others in Find Tab.
// The fpath can be just the path or any string fragment contained
// within the target filename.
func (dv *DebugPanel) FindFrames(fpath string, line int) { //types:add
	if !dv.DbgIsAvail() {
		return
	}
	fr, err := dv.Dbg.FindFrames(&dv.State, fpath, line)
	if fr == nil || err != nil {
		core.MessageSnackbar(dv, fmt.Sprintf("Could not find any stack frames for file name: %v, err: %v", fpath, err))
		return
	}
	dv.State.FindFrames = fr
	dv.ShowTab(DebugTabFrames)
}

// ListGlobalVars lists global vars matching the given optional filter.
func (dv *DebugPanel) ListGlobalVars(filter string) { //types:add
	if !dv.DbgIsAvail() {
		return
	}
	vrs, err := dv.Dbg.ListGlobalVars(filter)
	if err != nil {
		return
	}
	dv.State.GlobalVars = vrs
	dv.ShowTab(DebugTabGlobals)
}

// ShowFile shows the file name in code
func (dv *DebugPanel) ShowFile(fpath string, line int) {
	if fpath == "" || fpath == "?" {
		return
	}

	dv.DeleteCurPCInBuf()
	dv.Code.ShowFile(fpath, line)
	dv.SetCurPCInBuf(fpath, line)
	dv.NeedsRender()
}

// SetCurPCInBuf sets the current PC location in given file
// line is 1-based line number
func (dv *DebugPanel) SetCurPCInBuf(fpath string, line int) {
	tb := dv.Code.GetOpenFile(fpath)
	if tb != nil {
		if _, has := tb.LineColor(line - 1); !has {
			tb.SetLineColor(line-1, DebugBreakColors[DebugPCCurrent])
			if ed, _, ok := dv.Code.EditorForLines(tb); ok {
				ed.NeedsRender()
			}
			dv.CurFileLoc.FPath = fpath
			dv.CurFileLoc.Line = line
		}
	}
}

// DeleteCurPCInBuf deletes the current PC location in given file
// line is 1-based line number
func (dv *DebugPanel) DeleteCurPCInBuf() {
	fpath := dv.CurFileLoc.FPath
	line := dv.CurFileLoc.Line
	if fpath != "" && line > 0 {
		tb := dv.Code.GetOpenFile(fpath)
		if tb != nil {
			tb.DeleteLineColor(line - 1)
			if ed, _, ok := dv.Code.EditorForLines(tb); ok {
				ed.NeedsRender()
			}
		}
	}
	dv.CurFileLoc.FPath = ""
	dv.CurFileLoc.Line = 0
}

// ShowBreakFile shows the file for given break index
func (dv *DebugPanel) ShowBreakFile(bidx int) {
	if bidx < 0 || bidx >= len(dv.State.Breaks) {
		return
	}
	bk := dv.State.Breaks[bidx]
	dv.ShowFile(bk.FPath, bk.Line)
}

// ShowVar shows info on a given variable within the current frame scope in a text view dialog
func (dv *DebugPanel) ShowVar(name string) error {
	if !dv.DbgIsAvail() {
		return nil
	}
	vv, err := dv.Dbg.GetVar(name, dv.State.CurTask, dv.State.CurFrame)
	if err != nil {
		return err
	}
	frinfo := ""
	cf := dv.State.StackFrame(dv.State.CurFrame)
	if cf != nil {
		frinfo = "at: " + cf.FPath + fmt.Sprintf(":%d  Thread: %d  Depth: %d", cf.Line, dv.State.CurTask, dv.State.CurFrame)
	}
	VarViewDialog(vv, frinfo, dv)
	return nil
}

// VarValue returns the value of given variable, first looking in local stack vars
// and then in global vars
func (dv *DebugPanel) VarValue(varNm string) string {
	if !dv.DbgIsAvail() {
		return ""
	}
	if strings.Contains(varNm, ".") {
		vv, err := dv.Dbg.GetVar(varNm, dv.State.CurTask, dv.State.CurFrame)
		if err == nil {
			return vv.Value
		}
	} else {
		vr := dv.State.VarByName(varNm)
		if vr != nil {
			return vr.Value
		}
	}
	return ""
}

// DebugStatusColors contains the status colors for different debugging states.
var DebugStatusColors = map[cdebug.Status]image.Image{
	cdebug.NotInit:    colors.Scheme.SurfaceContainerHighest,
	cdebug.Error:      colors.Scheme.Error.Container,
	cdebug.Building:   colors.Scheme.Warn.Container,
	cdebug.Ready:      colors.Scheme.Success.Container,
	cdebug.Running:    colors.Scheme.Tertiary.Container,
	cdebug.Stopped:    colors.Scheme.Warn.Container,
	cdebug.Breakpoint: colors.Scheme.Warn.Container,
	cdebug.Finished:   colors.Scheme.SurfaceContainerHighest,
}

func (dv *DebugPanel) SetStatus(stat cdebug.Status) {
	if dv == nil || dv.This == nil {
		return
	}

	dv.State.Status = stat
	tb := dv.Toolbar()
	stl := tb.ChildByName("status", 1).(*core.Text)
	text := stat.String()
	if stat == cdebug.Breakpoint {
		text = fmt.Sprintf("Break: %d", dv.State.CurBreak)
	}
	stl.SetText(text)
	tb.Update() // state change
}

// Toolbar returns the debug toolbar
func (dv *DebugPanel) Toolbar() *core.Frame {
	return dv.ChildByName("toolbar", 0).(*core.Frame)
}

// Tabs returns the tabs
func (dv *DebugPanel) Tabs() *core.Tabs {
	tb := dv.ChildByName("tabs", 1)
	if tb == nil {
		return nil
	}
	return tb.(*core.Tabs)
}

// ShowTab shows given tab
func (dv *DebugPanel) ShowTab(tab string) {
	dv.Tabs().SelectTabByName(tab)
}

// UpdateTab updates given tab
func (dv *DebugPanel) UpdateTab(tab string) {
	tf := dv.Tabs().TabByName(tab)
	tf.Update()
}

// ConsoleText returns the console TextEditor
func (dv *DebugPanel) ConsoleText() *textcore.Editor {
	tv := dv.Tabs()
	cv := tv.TabByName(DebugTabConsole).Child(0).(*textcore.Editor)
	return cv
}

func (dv *DebugPanel) MakeToolbar(p *tree.Plan) {
	tree.AddAt(p, "status", func(w *core.Text) {
		w.SetText("Building").Styler(func(s *styles.Style) {
			color := DebugStatusColors[dv.State.Status]
			s.Background = color
			s.Color = colors.Uniform(hct.ContrastColor(colors.ToUniform(color), hct.ContrastAA))
		})
	})

	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.Refresh).
			SetTooltip("(re)start the debugger on exe:" + dv.ExePath + "; automatically rebuilds exe if any source files have changed").
			OnClick(func(e events.Event) {
				dv.Start()
			})
	})

	tree.Add(p, func(w *core.Button) {
		w.SetText("Continue").SetIcon(icons.PlayArrow).SetShortcut("Control+Alt+R")
		w.SetTooltip("continue execution from current point")
		w.FirstStyler(func(s *styles.Style) { s.SetEnabled(dv.DbgIsAvail()) })
		w.OnClick(func(e events.Event) {
			go dv.Continue()
		})
	})

	tree.Add(p, func(w *core.Text) {
		w.SetText("Step: ")
	})

	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(dv.StepOver).SetText("Over").SetShortcut("F6")
		w.FirstStyler(func(s *styles.Style) { s.SetEnabled(dv.DbgIsAvail()) })
	})

	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(dv.StepInto).SetText("Into").SetShortcut("F7")
		w.FirstStyler(func(s *styles.Style) { s.SetEnabled(dv.DbgIsAvail()) })
	})

	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(dv.StepOut).SetText("Out").SetShortcut("F8")
		w.FirstStyler(func(s *styles.Style) { s.SetEnabled(dv.DbgIsAvail()) })
	})

	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(dv.SingleStep).SetText("Single").SetIcon(icons.Step)
		w.FirstStyler(func(s *styles.Style) { s.SetEnabled(dv.DbgIsAvail()) })
	})

	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(dv.Stop)
		w.FirstStyler(func(s *styles.Style) { s.SetEnabled(!dv.DbgIsAvail()) })
	})

	tree.Add(p, func(w *core.Separator) {})

	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(dv.ListGlobalVars).SetText("Global vars").SetIcon(icons.Search)
		w.FirstStyler(func(s *styles.Style) { s.SetEnabled(dv.DbgIsAvail()) })
	})

	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(dv.FindFrames).SetText("Find frames").SetIcon(icons.Search)
		w.FirstStyler(func(s *styles.Style) { s.SetEnabled(dv.DbgIsAvail()) })
	})

	tree.Add(p, func(w *core.Button) {
		w.SetText("Params").SetIcon(icons.Edit).SetTooltip("edit the debugger parameters (e.g., for passing args: use -- (double dash) to separate args passed to program vs. those passed to the debugger itself)")
		w.OnClick(func(e events.Event) {
			DebugSettingsEditor(&dv.Code.Settings.Debug)
		})
	})

}

//////////////////////////////////////////////////////////////////////////////////////
//  VarView

// VarView shows a debug variable in an inspector-like framework,
// with sub-variables in a tree.
type VarView struct {
	core.Frame

	// variable being edited
	Var *cdebug.Variable `set:"-"`

	SelectVar *cdebug.Variable `set:"-"`

	// frame info
	FrameInfo string `set:"-"`

	// parent DebugPanel
	DbgView *DebugPanel `json:"-" xml:"-"`
}

// SetVar sets the source variable and ensures configuration
func (vv *VarView) SetVar(vr *cdebug.Variable, frinfo string) {
	vv.FrameInfo = frinfo
	if vv.Var != vr {
		vv.Var = vr
		vv.SelectVar = vr
		tr := vv.Tree()
		tr.SyncTree(vv.Var)
		fr := vv.Form()
		fr.SetStruct(vv.Var)
	}
	vv.Update()
}

func (vv *VarView) Init() {
	vv.Frame.Init()
	vv.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})

	tree.AddChildAt(vv, "frame-info", func(w *core.Text) {
		w.Updater(func() {
			w.SetText(vv.FrameInfo)
		})
	})
	tree.AddChildAt(vv, "splits", func(w *core.Splits) {
		w.SetSplits(0.3, 0.7)
		tree.AddChild(w, func(w *core.Frame) {
			tree.AddChild(w, func(w *core.Tree) {
				w.OnSelect(func(e events.Event) {
					if len(w.SelectedNodes) > 0 {
						sn := w.SelectedNodes[0].AsCoreTree().SyncNode
						vr, ok := sn.(*cdebug.Variable)
						if ok {
							vv.SelectVar = vr
						}
						fr := vv.Form()
						fr.SetStruct(vv.SelectVar)
						fr.Update()
					}
				})
			})
		})
		tree.AddChild(w, func(w *core.Form) {})
	})
}

// Splits returns the main Splits
func (vv *VarView) Splits() *core.Splits {
	return vv.ChildByName("splits", 1).(*core.Splits)
}

// Tree returns the main Tree
func (vv *VarView) Tree() *core.Tree {
	return vv.Splits().Child(0).AsTree().Child(0).(*core.Tree)
}

// Form returns the main Form
func (vv *VarView) Form() *core.Form {
	return vv.Splits().Child(1).(*core.Form)
}

func (vv *VarView) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.Button) {
		w.SetText("Follow pointer").SetIcon(icons.ArrowForward).
			SetTooltip("FollowPtr loads additional debug state information for pointer variables, so you can continue clicking through the tree to see what it points to.").
			OnClick(func(e events.Event) {
				if vv.SelectVar != nil {
					vv.SelectVar.FollowPtr()
					tv := vv.Tree()
					tv.SyncTree(vv.Var)
				}
			})
	})
}

// VarViewDialog opens an interactive editor of the given variable.
func VarViewDialog(vr *cdebug.Variable, frinfo string, dbgVw *DebugPanel) *VarView {
	if core.RecycleDialog(vr) {
		return nil
	}
	wnm := "var-view"
	wti := "Var View"
	if vr != nil {
		wnm += "-" + vr.Name
		wti += ": " + vr.Name
	}
	b := core.NewBody() // wnm)
	b.Title = wti
	vv := NewVarView(b)
	vv.DbgView = dbgVw
	vv.Update()
	vv.SetVar(vr, frinfo)
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(vv.MakeToolbar)
	})
	b.RunWindow()
	return vv
}
