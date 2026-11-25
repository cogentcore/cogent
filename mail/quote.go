// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/text/textpos"
)

// quotedLines returns the range of lines whose quote level changes in the
// given editor: those of the selection, without its last line if the
// selection ends at its start, or else the line of the cursor.
func quotedLines(ed *textcore.Editor) (start, end int) {
	if !ed.HasSelection() {
		return ed.CursorPos.Line, ed.CursorPos.Line
	}
	rg := ed.SelectRegion
	start, end = rg.Start.Line, rg.End.Line
	if end > start && rg.End.Char == 0 {
		end--
	}
	return start, end
}

// ChangeQuoteLevel increases (delta > 0) or decreases (delta < 0) the
// quote level of the lines of the selection (or the cursor) in the given
// editor, in markdown, as one undoable edit: a line gets "> " in front of
// it (or ">" in front of a quoted one), or loses its first ">", with the
// space after it.
func ChangeQuoteLevel(ed *textcore.Editor, delta int) {
	ls := ed.Lines
	if ls == nil {
		return
	}
	start, end := quotedLines(ed)
	var reps []lines.Replacement
	for ln := start; ln <= end; ln++ {
		line := ls.Line(ln)
		at := textpos.Pos{Line: ln}
		switch {
		case delta > 0:
			text := "> "
			if len(line) > 0 && line[0] == '>' {
				text = ">"
			}
			reps = append(reps, lines.Replacement{Region: textpos.NewRegionPos(at, at), Text: text})
		case len(line) > 0 && line[0] == '>':
			n := 1
			if len(line) > 1 && line[1] == ' ' {
				n = 2
			}
			reps = append(reps, lines.Replacement{Region: textpos.NewRegion(ln, 0, ln, n)})
		}
	}
	if len(reps) == 0 {
		return
	}
	ls.ReplaceTexts(reps, lines.ReplaceNoMatchCase)
	ed.NeedsRender()
}
