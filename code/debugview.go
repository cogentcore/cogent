// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"slices"
	"strings"
	"time"

	"cogentcore.org/cogent/code/cdebug"
	"cogentcore.org/cogent/code/cdebug/cdelve"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/cam/hct"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/views"
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
		DebugBreakInactive: colors.C(colors.Scheme.Warn.Base),
		DebugBreakActive:   colors.C(colors.Scheme.Error.Base),
		DebugBreakCurrent:  colors.C(colors.Scheme.Success.Base),
		DebugPCCurrent:     colors.C(colors.Scheme.Primary.Base),
	}

	// Debuggers is the list of supported debuggers
	Debuggers = map[fileinfo.Known]func(path, rootPath string, outbuf *texteditor.Buffer, pars *cdebug.Params) (cdebug.GiDebug, error){
		fileinfo.Go: func(path, rootPath string, outbuf *texteditor.Buffer, pars *cdebug.Params) (cdebug.GiDebug, error) {
			return cdelve.NewGiDelve(path, rootPath, outbuf, pars)
		},
	}
)

// NewDebugger returns a new debugger for given supported file type
func NewDebugger(sup fileinfo.Known, path, rootPath string, outbuf *texteditor.Buffer, pars *cdebug.Params) (cdebug.GiDebug, error) {
	df, ok := Debuggers[sup]
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

// DebugView is the debugger
type DebugView struct {
	core.Frame

	// supported file type to determine debugger
	Sup fileinfo.Known

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
	OutputBuffer *texteditor.Buffer `set:"-" json:"-" xml:"-"`

	// parent code project
	Code *CodeView `set:"-" json:"-" xml:"-"`
}

// Config sets parameters that must be set for a new view
func (dv *DebugView) Config(cv *CodeView, sup fileinfo.Known, exePath string) {
	dv.Code = cv
	dv.Sup = sup
	dv.ExePath = exePath
}

func (dv *DebugView) Init() {
	dv.Frame.Init()
	dv.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	core.AddChildAt(dv, "toolbar", func(w *core.Frame) {
		core.ToolbarStyles(w)
		w.Maker(dv.MakeToolbar)
	})
	core.AddChildAt(dv, "tabs", func(w *core.Tabs) {
		dv.Updater(dv.InitTabs) // note: this is necessary to allow config to happen first
	})
}

func (dv *DebugView) InitTabs() {
	w := dv.Tabs()
	if w.NumTabs() > 0 {
		return
	}
	ctv := texteditor.NewEditor(w.NewTab(DebugTabConsole))
	ctv.SetName("dbg-console")
	ConfigOutputTextEditor(ctv)
	dv.State.BlankState()
	dv.OutputBuffer = texteditor.NewBuffer()
	dv.OutputBuffer.Filename = core.Filename("debug-outbuf")
	dv.State.Breaks = nil // get rid of dummy
	dv.OutputBuffer.Options.LineNumbers = false
	ctv.SetBuffer(dv.OutputBuffer)

	bv := w.NewTab(DebugTabBreaks)
	core.AddChild(bv, func(w *views.TableView) {
		w.SetFlag(false, views.SliceViewShowIndex)
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

	sv := w.NewTab(DebugTabStack)
	core.AddChild(sv, func(w *views.TableView) {
		w.SetFlag(false, views.SliceViewShowIndex)
		w.SetReadOnly(true)
		w.SetSlice(&dv.State.Stack)
		w.OnDoubleClick(func(e events.Event) {
			dv.SetFrame(w.SelectedIndex)
		})
		w.Updater(func() {
			w.SelectedIndex = dv.State.CurFrame
		})
	})

	if dv.Sup == fileinfo.Go { // dv.Dbg.HasTasks() { // todo: not avail here yet
		tv := w.NewTab(DebugTabTasks)
		core.AddChild(tv, func(w *views.TableView) {
			w.SetFlag(false, views.SliceViewShowIndex)
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

	tv := w.NewTab(DebugTabThreads)
	core.AddChild(tv, func(w *views.TableView) {
		w.SetFlag(false, views.SliceViewShowIndex)
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

	vv := w.NewTab(DebugTabVars)
	core.AddChild(vv, func(w *views.TableView) {
		w.SetFlag(false, views.SliceViewShowIndex)
		w.SetReadOnly(true)
		w.SetSlice(&dv.State.Vars)
		w.OnDoubleClick(func(e events.Event) {
			vr := dv.State.Vars[w.SelectedIndex]
			dv.ShowVar(vr.Name)
		})
	})

	ff := w.NewTab(DebugTabFrames)
	core.AddChild(ff, func(w *views.TableView) {
		w.SetFlag(false, views.SliceViewShowIndex)
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

	gv := w.NewTab(DebugTabGlobals)
	core.AddChild(gv, func(w *views.TableView) {
		w.SetFlag(false, views.SliceViewShowIndex)
		w.SetReadOnly(true)
		w.SetSlice(&dv.State.GlobalVars)
		w.OnDoubleClick(func(e events.Event) {
			vr := dv.State.Vars[w.SelectedIndex]
			dv.ShowVar(vr.Name)
		})
	})
}

// DbgIsActive means debugger is started.
func (dv *DebugView) DbgIsActive() bool {
	return dv.Dbg != nil && dv.Dbg.IsActive()
}

// DbgIsAvail means the debugger is started AND process is not currently running --
// it is available for command input.
func (dv *DebugView) DbgIsAvail() bool {
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
func (dv *DebugView) DbgCanStep() bool {
	if !dv.DbgIsAvail() {
		return false
	}
	if dv.State.State.NextUp {
		return false
	}
	return true
}

func (dv *DebugView) Destroy() {
	dv.Detach()
	dv.DeleteAllBreaks()
	dv.DeleteCurPCInBuf()
	dv.Code.ClearDebug()
	dv.Frame.Destroy()
}

// Detach from debugger
func (dv *DebugView) Detach() {
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
func (dv *DebugView) Start() {
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
		dbg, err := NewDebugger(dv.Sup, dv.ExePath, rootPath, dv.OutputBuffer, pars)
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
func (dv *DebugView) UpdateView() {
	ds, err := dv.Dbg.GetState()
	if err != nil {
		return
	}
	dv.InitState(ds)
}

// Continue continues running from current point -- this MUST be called
// in a separate goroutine!
func (dv *DebugView) Continue() {
	if !dv.DbgIsAvail() {
		return
	}
	dv.SetBreaks()
	dv.State.State.Running = true
	dv.SetStatus(cdebug.Running)
	dsc := dv.Dbg.Continue(&dv.State)
	var ds *cdebug.State
	for ds = range dsc { // get everything
		if dv.This() == nil {
			return
		}
	}
	if dv.Code != nil {
		sc := dv.Code.AsWidget().Scene
		if sc != nil && sc.Stage.Mains != nil {
			sc.Stage.Mains.RenderWindow.Raise()
		}
	}
	if ds != nil {
		dv.InitState(ds)
	} else {
		dv.State.State.Running = false
		dv.SetStatus(cdebug.Finished)
	}
}

// StepOver continues to the next source line, not entering function calls.
func (dv *DebugView) StepOver() {
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
func (dv *DebugView) StepInto() {
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

// StepOut continues to the return point of the current function
func (dv *DebugView) StepOut() {
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

// StepSingle steps a single cpu instruction.
func (dv *DebugView) SingleStep() {
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

// Stop stops a running process
func (dv *DebugView) Stop() {
	// if !dv.DbgIsActive() || dv.DbgIsAvail() {
	// 	return
	// }
	_, err := dv.Dbg.Stop()
	if err != nil {
		return
	}
	// note: it will auto update from continue stopping.
}

// SetBreaks sets the current breakpoints from State, call this prior to running
func (dv *DebugView) SetBreaks() {
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
func (dv *DebugView) AddBreak(fpath string, line int) {
	dv.State.AddBreak(fpath, line)
	dv.BackupBreaks()
	dv.ShowTab(DebugTabBreaks)
	dv.UpdateTab(DebugTabBreaks)
}

// DeleteBreak deletes given breakpoint.  If debugger is not yet
// activated then it just deletes from master list.
// Note that breakpoints can be turned on and off directly using On flag.
func (dv *DebugView) DeleteBreak(fpath string, line int) {
	if dv.This() == nil {
		return
	}
	dv.DeleteBreakImpl(fpath, line)
	dv.BackupBreaks()
	dv.ShowTab(DebugTabBreaks)
	dv.UpdateTab(DebugTabBreaks)
}

// DeleteBreakImpl deletes given breakpoint with no other updates
func (dv *DebugView) DeleteBreakImpl(fpath string, line int) {
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
func (dv *DebugView) DeleteBreakIndex(bidx int) {
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
func (dv *DebugView) SyncBreaks() {
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
func (dv *DebugView) DeleteBreakInBuf(fpath string, line int) {
	if dv.Code == nil || dv.Code.This() == nil {
		return
	}
	tb := dv.Code.TextBufForFile(fpath, false)
	if tb != nil {
		tb.DeleteLineColor(line - 1)
		tb.Update()
	}
}

// DeleteAllBreaks deletes all breakpoints
func (dv *DebugView) DeleteAllBreaks() {
	if dv.Code == nil || dv.Code.This() == nil {
		return
	}
	for _, bk := range dv.State.Breaks {
		dv.DeleteBreakInBuf(bk.FPath, bk.Line)
	}
}

// UpdateBreakInBuf updates break status in its TextBuf
// line is 1-based line number
func (dv *DebugView) UpdateBreakInBuf(fpath string, line int, stat DebugBreakStatus) {
	if dv.Code == nil || dv.Code.This() == nil {
		return
	}
	tb := dv.Code.TextBufForFile(fpath, false)
	if tb != nil {
		tb.SetLineColor(line-1, DebugBreakColors[stat])
		tb.Update()
	}
}

// UpdateAllBreaks updates all breakpoints
func (dv *DebugView) UpdateAllBreaks() {
	if dv.Code == nil || dv.Code.This() == nil {
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
func (dv *DebugView) BackupBreaks() {
	dv.BBreaks = make([]*cdebug.Break, len(dv.State.Breaks))
	for i, b := range dv.State.Breaks {
		dv.BBreaks[i] = b
	}
}

// InitState updates the State and View from given debug state
// Call this when debugger returns from any action update
func (dv *DebugView) InitState(ds *cdebug.State) {
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
func (dv *DebugView) UpdateFromState() {
	if dv == nil || dv.This() == nil || dv.Dbg == nil {
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
func (dv *DebugView) SetFrame(depth int) {
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
func (dv *DebugView) SetThread(threadID int) {
	if !dv.DbgIsAvail() {
		return
	}
	dv.Dbg.UpdateAllState(&dv.State, threadID, 0)
	dv.UpdateFromState()
}

// SetThreadIndex sets the given thread by index in threads list as active
// this must be TaskID if HasTasks and ThreadID if not.
func (dv *DebugView) SetThreadIndex(thridx int) {
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
// Selects the one that is closest and shows the others in Find Tab
func (dv *DebugView) FindFrames(fpath string, line int) {
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

// ListGlobalVars lists global vars matching given optional filter in Global Vars tab
func (dv *DebugView) ListGlobalVars(filter string) {
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
func (dv *DebugView) ShowFile(fpath string, line int) {
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
func (dv *DebugView) SetCurPCInBuf(fpath string, line int) {
	tb := dv.Code.TextBufForFile(fpath, false)
	if tb != nil {
		if !tb.HasLineColor(line - 1) {
			tb.SetLineColor(line-1, DebugBreakColors[DebugPCCurrent])
			tb.Update()
			dv.CurFileLoc.FPath = fpath
			dv.CurFileLoc.Line = line
		}
	}
}

// DeleteCurPCInBuf deletes the current PC location in given file
// line is 1-based line number
func (dv *DebugView) DeleteCurPCInBuf() {
	fpath := dv.CurFileLoc.FPath
	line := dv.CurFileLoc.Line
	if fpath != "" && line > 0 {
		tb := dv.Code.TextBufForFile(fpath, false)
		if tb != nil {
			tb.DeleteLineColor(line - 1)
			tb.Update()
		}
	}
	dv.CurFileLoc.FPath = ""
	dv.CurFileLoc.Line = 0
}

// ShowBreakFile shows the file for given break index
func (dv *DebugView) ShowBreakFile(bidx int) {
	if bidx < 0 || bidx >= len(dv.State.Breaks) {
		return
	}
	bk := dv.State.Breaks[bidx]
	dv.ShowFile(bk.FPath, bk.Line)
}

// ShowVar shows info on a given variable within the current frame scope in a text view dialog
func (dv *DebugView) ShowVar(name string) error {
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
func (dv *DebugView) VarValue(varNm string) string {
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
var DebugStatusColors = map[cdebug.Status]color.RGBA{
	cdebug.NotInit:    colors.Scheme.SurfaceContainerHighest,
	cdebug.Error:      colors.Scheme.Error.Container,
	cdebug.Building:   colors.Scheme.Warn.Container,
	cdebug.Ready:      colors.Scheme.Success.Container,
	cdebug.Running:    colors.Scheme.Tertiary.Container,
	cdebug.Stopped:    colors.Scheme.Warn.Container,
	cdebug.Breakpoint: colors.Scheme.Warn.Container,
	cdebug.Finished:   colors.Scheme.SurfaceContainerHighest,
}

func (dv *DebugView) SetStatus(stat cdebug.Status) {
	if dv == nil || dv.This() == nil {
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
func (dv *DebugView) Toolbar() *core.Frame {
	return dv.ChildByName("toolbar", 0).(*core.Frame)
}

// Tabs returns the tabs
func (dv *DebugView) Tabs() *core.Tabs {
	return dv.ChildByName("tabs", 1).(*core.Tabs)
}

// ShowTab shows given tab
func (dv *DebugView) ShowTab(tab string) {
	dv.Tabs().SelectTabByName(tab)
}

// UpdateTab updates given tab
func (dv *DebugView) UpdateTab(tab string) {
	tf := dv.Tabs().TabByName(tab)
	tf.Update()
}

// ConsoleText returns the console TextEditor
func (dv *DebugView) ConsoleText() *texteditor.Editor {
	tv := dv.Tabs()
	cv := tv.TabByName(DebugTabConsole).Child(0).(*texteditor.Editor)
	return cv
}

func (dv *DebugView) MakeToolbar(p *core.Plan) {
	core.AddAt(p, "status", func(w *core.Text) {
		w.SetText("Building").Styler(func(s *styles.Style) {
			color := DebugStatusColors[dv.State.Status]
			s.Background = colors.C(color)
			s.Color = colors.C(hct.ContrastColor(color, hct.ContrastAA))
		})
	})

	core.Add(p, func(w *core.Button) {
		w.SetIcon(icons.Refresh).
			SetTooltip("(re)start the debugger on exe:" + dv.ExePath + "; automatically rebuilds exe if any source files have changed").
			OnClick(func(e events.Event) {
				dv.Start()
			})
	})

	core.Add(p, func(w *core.Button) {
		w.SetText("Cont").SetIcon(icons.PlayArrow).SetShortcut("Control+Alt+R")
		w.SetTooltip("continue execution from current point").
			FirstStyler(func(s *styles.Style) { s.SetEnabled(dv.DbgIsAvail()) }).
			OnClick(func(e events.Event) {
				go dv.Continue()
			})
	})

	core.Add(p, func(w *core.Text) {
		w.SetText("Step: ")
	})

	core.Add(p, func(w *core.Button) {
		w.SetText("Over").SetIcon(icons.StepOver).SetShortcut("F6")
		w.SetTooltip("continues to the next source line, not entering function calls").
			FirstStyler(func(s *styles.Style) { s.SetEnabled(dv.DbgIsAvail()) }).
			OnClick(func(e events.Event) {
				dv.StepOver()
			})
	})

	core.Add(p, func(w *core.Button) {
		w.SetText("Into").SetIcon(icons.StepInto).SetShortcut("F7")
		w.SetTooltip("continues to the next source line, entering into function calls").
			FirstStyler(func(s *styles.Style) { s.SetEnabled(dv.DbgIsAvail()) }).
			OnClick(func(e events.Event) {
				dv.StepInto()
			})
	})

	core.Add(p, func(w *core.Button) {
		w.SetText("Out").SetIcon(icons.StepOut).SetShortcut("F8")
		w.SetTooltip("continues to the return point of the current function").
			FirstStyler(func(s *styles.Style) { s.SetEnabled(dv.DbgIsAvail()) }).
			OnClick(func(e events.Event) {
				dv.StepOut()
			})
	})

	core.Add(p, func(w *core.Button) {
		w.SetText("Single").SetIcon(icons.Step).
			SetTooltip("steps a single CPU instruction").
			FirstStyler(func(s *styles.Style) { s.SetEnabled(dv.DbgIsAvail()) }).
			OnClick(func(e events.Event) {
				dv.StepOut()
			})
	})

	core.Add(p, func(w *core.Button) {
		w.SetText("Stop").SetIcon(icons.Stop).
			SetTooltip("stop execution").
			FirstStyler(func(s *styles.Style) { s.SetEnabled(!dv.DbgIsAvail()) }).
			OnClick(func(e events.Event) {
				dv.Stop()
			})
	})

	core.Add(p, func(w *core.Separator) {})

	core.Add(p, func(w *core.Button) {
		w.SetText("Global Vars").SetIcon(icons.Search).
			SetTooltip("list variables at global scope, subject to filter (name contains)").
			FirstStyler(func(s *styles.Style) { s.SetEnabled(dv.DbgIsAvail()) }).
			OnClick(func(e events.Event) {
				views.CallFunc(dv, dv.ListGlobalVars)
			})
	})

	core.Add(p, func(w *core.Button) {
		w.SetText("Params").SetIcon(icons.Edit).
			SetTooltip("edit the debugger parameters (e.g., for passing args: use -- (double dash) to separate args passed to program vs. those passed to the debugger itself)").
			FirstStyler(func(s *styles.Style) { s.SetEnabled(dv.DbgIsAvail()) }).
			OnClick(func(e events.Event) {
				DebugSettingsView(&dv.Code.Settings.Debug)
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

	// parent DebugView
	DbgView *DebugView `json:"-" xml:"-"`
}

// SetVar sets the source variable and ensures configuration
func (vv *VarView) SetVar(vr *cdebug.Variable, frinfo string) {
	vv.FrameInfo = frinfo
	if vv.Var != vr {
		vv.Var = vr
		vv.SelectVar = vr
	}
	vv.Update()
}

func (vv *VarView) Init() {
	vv.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})

	core.AddChildAt(vv, "frame-info", func(w *core.Text) {
		w.SetText(vv.FrameInfo)
	})
	core.AddChildAt(vv, "splits", func(w *core.Splits) {
		w.SetSplits(0.3, 0.7)
		core.AddChild(w, func(w *core.Frame) {
			core.AddChild(w, func(w *views.TreeView) {
				w.SyncTree(vv.Var)
				w.OnSelect(func(e events.Event) {
					if len(w.SelectedNodes) > 0 {
						sn := w.SelectedNodes[0].AsTreeView().SyncNode
						vr, ok := sn.(*cdebug.Variable)
						if ok {
							vv.SelectVar = vr
						}
						vv := vv.StructView()
						vv.SetStruct(sn)
					}
				})
			})
		})
		core.AddChild(w, func(w *views.StructView) {
			w.SetStruct(vv.Var)
		})
	})
}

// Splits returns the main Splits
func (vv *VarView) Splits() *core.Splits {
	return vv.ChildByName("splits", 1).(*core.Splits)
}

// TreeView returns the main TreeView
func (vv *VarView) TreeView() *views.TreeView {
	return vv.Splits().Child(0).AsTree().Child(0).(*views.TreeView)
}

// StructView returns the main StructView
func (vv *VarView) StructView() *views.StructView {
	return vv.Splits().Child(1).(*views.StructView)
}

func (vv *VarView) MakeToolbar(p *core.Plan) {
	core.Add(p, func(w *core.Button) {
		w.SetText("Follow pointer").SetIcon(icons.ArrowForward).
			SetTooltip("FollowPtr loads additional debug state information for pointer variables, so you can continue clicking through the tree to see what it points to.").
			OnClick(func(e events.Event) {
				if vv.SelectVar != nil {
					vv.SelectVar.FollowPtr()
					tv := vv.TreeView()
					tv.SyncTree(vv.Var)
				}
			})
	})
}

// VarViewDialog opens an interactive editor of the given variable.
func VarViewDialog(vr *cdebug.Variable, frinfo string, dbgVw *DebugView) *VarView {
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
	vv.SetVar(vr, frinfo)
	b.AddAppBar(vv.MakeToolbar)
	b.RunWindow()
	return vv
}
