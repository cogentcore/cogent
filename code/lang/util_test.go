// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lang

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestUTF16 tests converting between the rune offsets of the editor and
// the UTF-16 offsets of the language servers, with characters outside
// of the basic plane, which are two UTF-16 units.
func TestUTF16(t *testing.T) {
	line := "a\U0001F363b"
	assert.Equal(t, 0, RuneToUTF16(line, 0))
	assert.Equal(t, 1, RuneToUTF16(line, 1))
	assert.Equal(t, 3, RuneToUTF16(line, 2), "after the emoji")
	assert.Equal(t, 4, RuneToUTF16(line, 10), "clamped")
	assert.Equal(t, 0, UTF16ToRune(line, 0))
	assert.Equal(t, 1, UTF16ToRune(line, 1))
	assert.Equal(t, 2, UTF16ToRune(line, 2), "within the emoji")
	assert.Equal(t, 2, UTF16ToRune(line, 3))
	assert.Equal(t, 3, UTF16ToRune(line, 10), "clamped")
}

// TestDiagSeverity tests the strings of the severities of diagnostics.
func TestDiagSeverity(t *testing.T) {
	var s DiagSeverity
	assert.NoError(t, s.SetString(DiagErr.String()))
	assert.Equal(t, DiagErr, s)
	assert.NotEmpty(t, DiagSeverityValues())
	b, err := DiagErr.MarshalText()
	assert.NoError(t, err)
	assert.NoError(t, s.UnmarshalText(b))
	s.SetInt64(DiagErr.Int64())
	assert.Equal(t, DiagErr, s)
	assert.Len(t, s.Values(), len(DiagSeverityValues()))
	_ = s.Desc()
}
