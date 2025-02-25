// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"testing"

	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

func TestFindUrl(t *testing.T) {
	spath := "code/test.go"
	ur := findURL(spath, 2, 1, 10, 13, 24)
	fpath, reg, resultIndex, matchIndex, err := parseFindURL(ur)
	assert.NoError(t, err)
	assert.Equal(t, spath, fpath)
	assert.Equal(t, resultIndex, 2)
	assert.Equal(t, matchIndex, 1)
	assert.Equal(t, textpos.Pos{10, 13}, reg.Start)
	assert.Equal(t, textpos.Pos{10, 24}, reg.End)
}
