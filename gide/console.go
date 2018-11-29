// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/goki/gi/giv"
	"github.com/goki/ki/kit"
)

// Console redirects our os.Stdout and os.Stderr to a buffer for display within app
type Console struct {
	StdoutWrite *os.File     `json:"-" xml:"-" desc:"std out writer -- set to os.Stdout"`
	StdoutRead  *os.File     `json:"-" xml:"-" desc:"std out reader -- used to read os.Stdout"`
	StderrWrite *os.File     `json:"-" xml:"-" desc:"std err writer -- set to os.Stderr"`
	StderrRead  *os.File     `json:"-" xml:"-" desc:"std err reader -- used to read os.Stderr"`
	Buf         *giv.TextBuf `json:"-" xml:"-" desc:"text buffer holding all output"`
	Cancel      bool         `json:"-" xml:"-" desc:"set to true to cancel monitoring"`
	Mu          sync.Mutex   `json:"-" xml:"-" desc:"mutex protecting updating of buffer between out / err"`
	OrgoutWrite *os.File     `json:"-" xml:"-" desc:"original os.Stdout writer"`
	OrgerrWrite *os.File     `json:"-" xml:"-" desc:"original os.Stderr writer"`
}

var KiT_Console = kit.Types.AddType(&Console{}, nil)

var TheConsole Console

// Init initializes the console -- sets up the capture, Buf, and
// starts the routine that monitors output
func (cn *Console) Init() {
	cn.StdoutRead, cn.StdoutWrite, _ = os.Pipe() // seriously, does this ever fail?
	cn.StderrRead, cn.StderrWrite, _ = os.Pipe() // seriously, does this ever fail?
	cn.OrgoutWrite = os.Stdout
	cn.OrgerrWrite = os.Stderr
	os.Stdout = cn.StdoutWrite
	os.Stderr = cn.StderrWrite
	log.SetOutput(cn.StderrWrite)
	cn.Buf = &giv.TextBuf{}
	cn.Buf.InitName(cn.Buf, "console-buf")
	go cn.MonitorOut()
	go cn.MonitorErr()
}

// MonitorOut monitors std output and appends it to the buffer
// should be in a separate routine
func (cn *Console) MonitorOut() {
	obuf := giv.OutBuf{}
	obuf.Init(cn.StdoutRead, cn.Buf, 0, MarkupStdout)
	obuf.MonOut()
}

// MonitorErr monitors std error and appends it to the buffer
// should be in a separate routine
func (cn *Console) MonitorErr() {
	obuf := giv.OutBuf{}
	obuf.Init(cn.StderrRead, cn.Buf, 0, MarkupStderr)
	obuf.MonOut()
}

func MarkupStdout(out []byte) []byte {
	fmt.Fprintln(TheConsole.OrgoutWrite, string(out))
	return MarkupCmdOutput(out)
}

func MarkupStderr(out []byte) []byte {
	sst := []byte(`<span style="color:red">`)
	est := []byte(`</span>`)
	esz := len(sst) + len(est)

	fmt.Fprintln(TheConsole.OrgerrWrite, string(out))
	mb := MarkupCmdOutput(out)
	mbb := make([]byte, 0, len(mb)+esz)
	mbb = append(mbb, sst...)
	mbb = append(mbb, mb...)
	mbb = append(mbb, est...)
	return mbb
}
