// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"

	"cogentcore.org/cogent/code/lang"
)

// inlayLayer is the virtual text layer of the inlay hints from the
// language servers, which are shown in all views of a buffer.
const inlayLayer = "inlay"

// InlayRefresh requests the inlay hints for the given buffer from the
// language server asynchronously, and shows them in the editors as
// virtual text, which is display-only, when the InlayHints setting is
// on, and otherwise removes any. The results are discarded if a newer
// request (or a change/close) has superseded this one.
// The hints of a Goal file are those of its Go code (see [goalDoc]).
func (ls *LangService) InlayRefresh(ln *lines.Lines) {
	if ln == nil || ln.Filename() == "" {
		return
	}
	doc := ls.goalDocFor(ln)
	if !ls.cv.Settings.InlayHints || (doc == nil && !ls.LSPActive(ln)) {
		ls.bumpInlay(ln.Filename())
		if len(ln.VirtualText(inlayLayer, -1)) > 0 {
			ln.ClearVirtualText(inlayLayer, -1)
		}
		return
	}
	tln := ln
	if doc != nil {
		tln = doc.goLines
	}
	if !ls.lspB.Ready(tln) {
		return // server still loading: the next change re-triggers
	}
	fname := ln.Filename()
	ls.mu.Lock()
	ls.inlayGen[fname]++
	gen := ls.inlayGen[fname]
	ls.mu.Unlock()
	go ls.inlayPass(ln, doc, fname, gen)
}

// bumpInlay marks any in-flight inlay hints request for this file stale.
func (ls *LangService) bumpInlay(fname string) {
	ls.mu.Lock()
	ls.inlayGen[fname]++
	ls.mu.Unlock()
}

// inlayPass runs one inlay hints request and shows the hints, for the
// given buffer, or the given Goal file, if it is non-nil.
func (ls *LangService) inlayPass(ln *lines.Lines, doc *goalDoc, fname string, gen int) {
	stale := func() bool {
		ls.mu.Lock()
		defer ls.mu.Unlock()
		return ls.inlayGen[fname] != gen || ls.cv.Scene == nil
	}
	tln := ln
	if doc != nil {
		tln = doc.goLines
	}
	hints, err := ls.lspB.InlayHintTexts(tln, textpos.Region{End: textpos.Pos{Line: tln.NumLines()}})
	if err != nil || stale() {
		return
	}
	if doc != nil {
		hints = ls.goalHints(doc, hints)
	}
	ls.cv.AsyncLock()
	defer ls.cv.AsyncUnlock()
	if stale() {
		return
	}
	ls.setInlayHints(ln, hints)
}

// setInlayHints shows the given inlay hints in the views of the given
// buffer, as virtual text in the hint style of its editor.
func (ls *LangService) setInlayHints(ln *lines.Lines, hints []lang.InlayHint) {
	var sty *rich.Style
	if tv, _, has := ls.cv.EditorForLines(ln); has && tv != nil {
		sty = tv.HintStyle()
	} else {
		sty = rich.NewStyle()
	}
	vts := make([]lines.VirtualText, len(hints))
	for i, h := range hints {
		vts[i] = lines.VirtualText{Pos: h.Pos, Text: rich.NewText(sty, []rune(h.Text))}
	}
	ln.SetVirtualText(inlayLayer, -1, vts...)
}

// applyInlayHints configures the language servers with the given
// [ProjectSettings.InlayHints] setting, if it changed, and then refreshes
// the inlay hints of the open buffers, in the background. Must be called
// from the event loop.
func (ls *LangService) applyInlayHints(on bool) {
	if on == ls.inlayHints {
		return
	}
	ls.inlayHints = on
	go func() {
		ls.lspB.SetOptions(serverOptions(on))
		ls.lspB.Reconfigure()
		if ls.cv.Scene == nil {
			return
		}
		ls.cv.AsyncLock()
		defer ls.cv.AsyncUnlock()
		for _, ln := range ls.cv.OpenFiles.Values {
			ls.InlayRefresh(ln)
		}
	}()
}
