// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"log"
	"os"
	"sync"

	"cogentcore.org/core/gi"
	"cogentcore.org/core/texteditor"
)

// Console redirects our os.Stdout and os.Stderr to a buffer for display within app
type Console struct {

	// std out writer -- set to os.Stdout
	StdoutWrite *os.File `json:"-" xml:"-"`

	// std out reader -- used to read os.Stdout
	StdoutRead *os.File `json:"-" xml:"-"`

	// std err writer -- set to os.Stderr
	StderrWrite *os.File `json:"-" xml:"-"`

	// std err reader -- used to read os.Stderr
	StderrRead *os.File `json:"-" xml:"-"`

	// text buffer holding all output
	Buf *texteditor.Buffer `json:"-" xml:"-"`

	// set to true to cancel monitoring
	Cancel bool `json:"-" xml:"-"`

	// mutex protecting updating of buffer between out / err
	Mu sync.Mutex `json:"-" xml:"-"`

	// original os.Stdout writer
	OrgoutWrite *os.File `json:"-" xml:"-"`

	// original os.Stderr writer
	OrgerrWrite *os.File `json:"-" xml:"-"`

	// log file writer
	LogWrite *os.File `json:"-" xml:"-"`
}

var TheConsole Console

// Init initializes the console -- sets up the capture, Buf, and
// starts the routine that monitors output.
// if logFile is non-empty, writes output to that file as well.
func (cn *Console) Init(logFile string) {
	cn.StdoutRead, cn.StdoutWrite, _ = os.Pipe() // seriously, does this ever fail?
	cn.StderrRead, cn.StderrWrite, _ = os.Pipe() // seriously, does this ever fail?
	cn.OrgoutWrite = os.Stdout
	cn.OrgerrWrite = os.Stderr
	os.Stdout = cn.StdoutWrite
	os.Stderr = cn.StderrWrite
	log.SetOutput(cn.StderrWrite)
	cn.Buf = texteditor.NewBuffer()
	cn.Buf.Opts.LineNos = false
	cn.Buf.Filename = gi.Filename("console-buf")
	if logFile != "" {
		cn.LogWrite, _ = os.Create(logFile)
	}
	go cn.MonitorOut()
	go cn.MonitorErr()
}

// Close closes all the files -- call on exit
func (cn *Console) Close() {
	if cn.LogWrite != nil {
		cn.LogWrite.Close()
		cn.LogWrite = nil
	}
	os.Stdout = cn.OrgoutWrite
	os.Stderr = cn.OrgerrWrite
}

// MonitorOut monitors std output and appends it to the buffer
// should be in a separate routine
func (cn *Console) MonitorOut() {
	obuf := texteditor.OutputBuffer{}
	obuf.Init(cn.StdoutRead, cn.Buf, 0, MarkupStdout)
	obuf.MonitorOutput()
}

// MonitorErr monitors std error and appends it to the buffer
// should be in a separate routine
func (cn *Console) MonitorErr() {
	obuf := texteditor.OutputBuffer{}
	obuf.Init(cn.StderrRead, cn.Buf, 0, MarkupStderr)
	obuf.MonitorOutput()
}

func MarkupStdout(out []byte) []byte {
	fmt.Fprintln(TheConsole.OrgoutWrite, string(out))
	if TheConsole.LogWrite != nil {
		fmt.Fprintln(TheConsole.LogWrite, string(out))
	}
	return MarkupCmdOutput(out)
}

func MarkupStderr(out []byte) []byte {
	sst := []byte(`<span style="color:red">`)
	est := []byte(`</span>`)
	esz := len(sst) + len(est)

	fmt.Fprintln(TheConsole.OrgerrWrite, string(out))
	if TheConsole.LogWrite != nil {
		fmt.Fprintln(TheConsole.LogWrite, string(out))
	}
	mb := MarkupCmdOutput(out)
	mbb := make([]byte, 0, len(mb)+esz)
	mbb = append(mbb, sst...)
	mbb = append(mbb, mb...)
	mbb = append(mbb, est...)
	return mbb
}
