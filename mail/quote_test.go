// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

func TestChangeQuoteLevel(t *testing.T) {
	b := core.NewBody()
	ed := textcore.NewEditor(b)
	ed.Lines.SetString("one\n> two\nthree\nfour\n")
	// the selection ends at the start of the line of four, which is not in it
	ed.SelectRegion = textpos.NewRegion(0, 1, 3, 0)
	ChangeQuoteLevel(ed, 1)
	assert.Equal(t, "> one\n>> two\n> three\nfour\n", ed.Lines.String())
	ChangeQuoteLevel(ed, -1)
	ChangeQuoteLevel(ed, -1)
	assert.Equal(t, "one\ntwo\nthree\nfour\n", ed.Lines.String(), "no lower than none")

	ed.SelectRegion = textpos.Region{}
	ed.CursorPos = textpos.Pos{Line: 3, Char: 2}
	ChangeQuoteLevel(ed, 1)
	assert.Equal(t, "one\ntwo\nthree\n> four\n", ed.Lines.String(), "the line of the cursor")
	ed.Lines.Undo()
	assert.Equal(t, "one\ntwo\nthree\nfour\n", ed.Lines.String(), "one undo")
}
