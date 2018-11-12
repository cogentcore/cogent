// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

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
	outscan := bufio.NewScanner(cn.StdoutRead)
	outlns := make([][]byte, 0, 100)
	outmus := make([][]byte, 0, 100)
	lfb := []byte("\n")
	ts := time.Now()
	for outscan.Scan() {
		if cn.Cancel {
			break
		}
		b := outscan.Bytes()
		ob := make([]byte, len(b)) // note: scanner bytes are temp -- must copy!
		copy(ob, b)
		outlns = append(outlns, ob)
		outmus = append(outmus, MarkupCmdOutput(ob))
		fmt.Fprintln(cn.OrgoutWrite, string(ob))
		now := time.Now()
		lag := int(now.Sub(ts) / time.Millisecond)
		if lag > 200 {
			ts = now
			tlns := bytes.Join(outlns, lfb)
			mlns := bytes.Join(outmus, lfb)
			tlns = append(tlns, lfb...)
			mlns = append(mlns, lfb...)
			cn.Mu.Lock()
			cn.Buf.AppendTextMarkup(tlns, mlns, false, true) // no undo, yes signal
			cn.Buf.AutoScrollViews()
			cn.Mu.Unlock()
			outlns = make([][]byte, 0, 100)
			outmus = make([][]byte, 0, 100)
		}
	}
}

// MonitorErr monitors std error and appends it to the buffer
// should be in a separate routine
func (cn *Console) MonitorErr() {
	outscan := bufio.NewScanner(cn.StderrRead)
	outlns := make([][]byte, 0, 100)
	outmus := make([][]byte, 0, 100)
	lfb := []byte("\n")
	sst := []byte(`<span style="color:red">`)
	est := []byte(`</span>`)
	esz := len(sst) + len(est)
	ts := time.Now()
	for outscan.Scan() {
		if cn.Cancel {
			break
		}
		b := outscan.Bytes()
		ob := make([]byte, len(b)) // note: scanner bytes are temp -- must copy!
		copy(ob, b)
		outlns = append(outlns, ob)
		mb := MarkupCmdOutput(ob)
		fmt.Fprintln(cn.OrgerrWrite, string(ob))
		mbb := make([]byte, 0, len(mb)+esz)
		mbb = append(mbb, sst...)
		mbb = append(mbb, mb...)
		mbb = append(mbb, est...)
		outmus = append(outmus, mbb)
		now := time.Now()
		lag := int(now.Sub(ts) / time.Millisecond)
		if lag > 200 {
			ts = now
			tlns := bytes.Join(outlns, lfb)
			mlns := bytes.Join(outmus, lfb)
			tlns = append(tlns, lfb...)
			mlns = append(mlns, lfb...)
			cn.Mu.Lock()
			cn.Buf.AppendTextMarkup(tlns, mlns, false, true) // no undo, yes signal
			cn.Buf.AutoScrollViews()
			cn.Mu.Unlock()
			outlns = make([][]byte, 0, 100)
			outmus = make([][]byte, 0, 100)
		}
	}
}
