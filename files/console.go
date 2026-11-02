// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"os"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textcore"
)

// ConsoleTabName is the name of the console tab.
const ConsoleTabName = "Console"

// Console redirects os.Stdout and os.Stderr to its Lines, to show them in
// the console tab, as in Cogent Code, also writing them to the originals.
type Console struct {

	// Lines has all of the output.
	Lines *lines.Lines

	// stdout and stderr are the original os.Stdout and os.Stderr.
	stdout, stderr *os.File
}

// TheConsole is the console of the app.
var TheConsole Console

// Init redirects os.Stdout, os.Stderr, and the log to the console.
func (cn *Console) Init() {
	outRead, outWrite := errors.Log2(os.Pipe())
	errRead, errWrite := errors.Log2(os.Pipe())
	cn.stdout, cn.stderr = os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outWrite, errWrite
	log.SetOutput(errWrite)
	cn.Lines = lines.NewLines()
	cn.Lines.Settings.LineNumbers = false
	cn.Lines.SetFilename("console-buf")
	cn.Lines.SetReadOnly(true)
	go cn.monitor(outRead, cn.stdout, false)
	go cn.monitor(errRead, cn.stderr, true)
}

// monitor appends the output read from r to the Lines, writing it to orig.
func (cn *Console) monitor(r, orig *os.File, isErr bool) {
	obuf := textcore.OutputBuffer{}
	obuf.SetOutput(r).SetLines(cn.Lines).SetMarkupFunc(func(buf *lines.Lines, out []rune) rich.Text {
		fmt.Fprintln(orig, string(out))
		return rich.NewText(buf.FontStyle(), out)
	})
	if isErr {
		obuf.Lines.FontStyle().SetFillColor(colors.ToUniform(colors.Scheme.Error.Base))
	}
	obuf.MonitorOutput()
}

// OpenConsoleTab shows the console tab, making it if needed.
func (fl *Files) OpenConsoleTab() { //types:add
	ts := fl.Tabs()
	if ts == nil {
		return
	}
	fr := ts.RecycleTab(ConsoleTabName)
	if fr.HasChildren() {
		return
	}
	fr.Styler(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowHidden) // the editor scrolls itself
	})
	if TheConsole.Lines == nil { // not initialized, as in tests
		TheConsole.Lines = lines.NewLines()
	}
	ed := textcore.NewEditor(fr)
	ed.SetLines(TheConsole.Lines)
	ed.SetReadOnly(true)
	ed.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	ts.Update()
}
