// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

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

// Init redirects os.Stdout, os.Stderr, and the log to the console,
// except on js, which has no pipes: the output stays in the browser console.
func (cn *Console) Init() {
	cn.Lines = lines.NewLines()
	cn.Lines.Settings.LineNumbers = false
	cn.Lines.SetFilename("console-buf")
	cn.Lines.SetReadOnly(true)
	if runtime.GOOS == "js" {
		return
	}
	outRead, outWrite := errors.Log2(os.Pipe())
	errRead, errWrite := errors.Log2(os.Pipe())
	cn.stdout, cn.stderr = os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outWrite, errWrite
	log.SetOutput(errWrite)
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
		// SetLines sets the line numbers from the editor settings
		if ed.Lines != nil {
			ed.Lines.Settings.LineNumbers = false
		}
	})
	// (its Clear, in its context menu and with the Kill key, Command+K on
	// macOS, is that of read-only editors)
	ts.Update()
}

// Clear clears all of the output in the console.
func (cn *Console) Clear() {
	if cn.Lines != nil {
		cn.Lines.SetText(nil)
	}
}
