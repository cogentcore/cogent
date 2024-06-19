// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package stubdbg

import (
	"time"

	"cogentcore.org/cogent/code/cdebug"
	"cogentcore.org/core/texteditor"
)

// GiDebug is the interface for all supported debuggers.
// It is based directly on the Delve Client interface.
type Stub struct {
}

func (st *Stub) HasTasks() bool {
	return true
}

func (st *Stub) Start(path, rootPath string, outbuf *texteditor.Buffer, pars *cdebug.Params) error {
	return nil
}

func (st *Stub) SetParams(params *cdebug.Params) {

}

func (st *Stub) IsActive() bool {
	return false
}

func (st *Stub) ProcessPid() int {
	return 0
}

func (st *Stub) LastModified() time.Time {
	return time.Now()
}

func (st *Stub) Detach(killProcess bool) error {
	return nil
}

func (st *Stub) Disconnect(cont bool) error {
	return nil
}

func (st *Stub) Restart() error {
	return nil
}

func (st *Stub) GetState() (*cdebug.State, error) {
	return &cdebug.State{}, nil
}

func (st *Stub) Continue(all *cdebug.AllState) <-chan *cdebug.State {
	sc := make(chan *cdebug.State)
	return sc
}

func (st *Stub) StepOver() (*cdebug.State, error) {
	return &cdebug.State{}, nil
}

func (st *Stub) StepInto() (*cdebug.State, error) {
	return &cdebug.State{}, nil
}

func (st *Stub) StepOut() (*cdebug.State, error) {
	return &cdebug.State{}, nil
}

func (st *Stub) StepSingle() (*cdebug.State, error) {
	return &cdebug.State{}, nil
}

func (st *Stub) SwitchThread(threadID int) (*cdebug.State, error) {
	return &cdebug.State{}, nil
}

func (st *Stub) SwitchTask(threadID int) (*cdebug.State, error) {
	return &cdebug.State{}, nil
}

func (st *Stub) Stop() (*cdebug.State, error) {
	return &cdebug.State{}, nil
}

func (st *Stub) GetBreak(id int) (*cdebug.Break, error) {
	return nil, nil
}

func (st *Stub) SetBreak(fname string, line int) (*cdebug.Break, error) {
	return nil, nil
}

func (st *Stub) ListBreaks() ([]*cdebug.Break, error) {
	return nil, nil
}

func (st *Stub) ClearBreak(id int) error {
	return nil
}

func (st *Stub) AmendBreak(id int, fname string, line int, cond string, trace bool) error {
	return nil
}

func (st *Stub) UpdateBreaks(brk *[]*cdebug.Break) error {
	return nil
}

func (st *Stub) CancelNext() error {
	return nil
}

func (st *Stub) InitAllState(all *cdebug.AllState) error {
	return nil
}

func (st *Stub) UpdateAllState(all *cdebug.AllState, threadID int, frame int) error {
	return nil
}

func (st *Stub) FindFrames(all *cdebug.AllState, fname string, line int) ([]*cdebug.Frame, error) {
	return nil, nil
}

func (st *Stub) CurThreadID(all *cdebug.AllState) int {
	return 0
}

func (st *Stub) ListThreads() ([]*cdebug.Thread, error) {
	return nil, nil
}

func (st *Stub) GetThread(id int) (*cdebug.Thread, error) {
	return nil, nil
}

func (st *Stub) ListTasks() ([]*cdebug.Task, error) {
	return nil, nil
}

func (st *Stub) Stack(threadID int, depth int) ([]*cdebug.Frame, error) {
	return nil, nil
}

func (st *Stub) ListGlobalVars(filter string) ([]*cdebug.Variable, error) {
	return nil, nil
}

func (st *Stub) ListVars(threadID int, frame int) ([]*cdebug.Variable, error) {
	return nil, nil
}

func (st *Stub) GetVar(expr string, threadID int, frame int) (*cdebug.Variable, error) {
	return nil, nil
}

func (st *Stub) FollowPtr(vr *cdebug.Variable) error {
	return nil
}

func (st *Stub) SetVar(name, value string, threadID int, frame int) error {
	return nil
}

func (st *Stub) ListSources(filter string) ([]string, error) {
	return nil, nil
}

func (st *Stub) ListFuncs(filter string) ([]string, error) {
	return nil, nil
}

func (st *Stub) ListTypes(filter string) ([]string, error) {
	return nil, nil
}

func (st *Stub) WriteToConsole(msg string) {
}
