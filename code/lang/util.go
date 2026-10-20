// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lang

import (
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/textpos"
)

// WordAt returns the word at the given position in the buffer,
// for backends that need the word without referencing a language.
func WordAt(ln *lines.Lines, pos textpos.Pos) string {
	if ln == nil || !ln.IsValidLine(pos.Line) {
		return ""
	}
	reg := ln.WordAt(pos)
	line := ln.Line(reg.Start.Line)
	if reg.Start.Line != reg.End.Line || reg.End.Char > len(line) || reg.Start.Char > reg.End.Char {
		return ""
	}
	return string(line[reg.Start.Char:reg.End.Char])
}

// RuneToUTF16 converts a rune offset within the given line of text
// into a UTF-16 code unit offset, as used by the [LSP] protocol.
// It clamps to the length of the line.
//
// [LSP]: https://microsoft.github.io/language-server-protocol/specification
func RuneToUTF16(line string, runePos int) int {
	cnt := 0
	i := 0
	for _, r := range line {
		if i >= runePos {
			break
		}
		cnt += runeUTF16Len(r)
		i++
	}
	return cnt
}

// UTF16ToRune converts a UTF-16 code unit offset within the given line
// of text into a rune offset, as used by the editor. It clamps to the
// length of the line.
func UTF16ToRune(line string, utf16Pos int) int {
	cnt := 0
	runes := 0
	for _, r := range line {
		if cnt >= utf16Pos {
			return runes
		}
		w := runeUTF16Len(r)
		if cnt+w > utf16Pos {
			// utf16 position is inside a surrogate pair; round to this rune
			return runes + 1
		}
		cnt += w
		runes++
	}
	return runes // clamp to end
}

func runeUTF16Len(r rune) int {
	if r > 0xFFFF {
		return 2
	}
	return 1
}
