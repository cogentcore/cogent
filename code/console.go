// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"log"
	"os"
	"sync"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textcore"
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

	// text lines holding all output
	Lines *lines.Lines `json:"-" xml:"-"`

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
	cn.StdoutRead, cn.StdoutWrite = errors.Log2(os.Pipe())
	cn.StderrRead, cn.StderrWrite = errors.Log2(os.Pipe())
	cn.OrgoutWrite = os.Stdout
	cn.OrgerrWrite = os.Stderr
	os.Stdout = cn.StdoutWrite
	os.Stderr = cn.StderrWrite
	log.SetOutput(cn.StderrWrite)
	cn.Lines = lines.NewLines()
	cn.Lines.Settings.LineNumbers = false
	cn.Lines.SetFilename("console-buf")
	cn.Lines.SetReadOnly(true)
	if logFile != "" {
		cn.LogWrite = errors.Log1(os.Create(logFile))
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
	obuf := textcore.OutputBuffer{}
	obuf.SetOutput(cn.StdoutRead).SetLines(cn.Lines).SetMarkupFunc(MarkupStdout)
	obuf.MonitorOutput()
}

// MonitorErr monitors std error and appends it to the buffer
// should be in a separate routine
func (cn *Console) MonitorErr() {
	obuf := textcore.OutputBuffer{}
	fs := obuf.Lines.FontStyle()
	fs.SetFillColor(colors.ToUniform(colors.Scheme.Error.Base))
	obuf.SetOutput(cn.StderrRead).SetLines(cn.Lines).SetMarkupFunc(MarkupStderr)
	obuf.MonitorOutput()
}

func MarkupStdout(buf *lines.Lines, out []rune) rich.Text {
	sout := string(out)
	fmt.Fprintln(TheConsole.OrgoutWrite, sout)
	if TheConsole.LogWrite != nil {
		fmt.Fprintln(TheConsole.LogWrite, sout)
	}
	return MarkupCmdOutput(buf, out, "")
}

func MarkupStderr(buf *lines.Lines, out []rune) rich.Text {
	sout := string(out)
	fmt.Fprintln(TheConsole.OrgerrWrite, sout)
	if TheConsole.LogWrite != nil {
		fmt.Fprintln(TheConsole.LogWrite, sout)
	}
	return MarkupCmdOutput(buf, out, "")
}
