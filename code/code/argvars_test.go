// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"path/filepath"
	"testing"

	"cogentcore.org/core/gi"
	"cogentcore.org/core/pi/lex"
	"cogentcore.org/core/texteditor"
	"github.com/stretchr/testify/assert"
)

func TestBind(t *testing.T) {
	tv := texteditor.Editor{}
	tv.CursorPos = lex.Pos{22, 44}
	tv.SelectRegion.Start = lex.Pos{11, 14}
	tv.SelectRegion.End = lex.Pos{55, 0}

	fpath := "/Users/oreilly/go/src/cogentcore.org/cogent/code/argvars_test.go"
	projpath := "/Users/oreilly/go/src/cogentcore.org"

	afpath, err := filepath.Abs(fpath)
	assert.NoError(t, err)

	pp := ProjSettings{}
	pp.ProjRoot = gi.Filename(projpath)

	var avp ArgVarVals
	avp.Set(fpath, &pp, &tv)

	bv := avp.Bind("FilePath")
	cv := "FilePath"
	if bv != cv {
		t.Errorf("bind error: should have been: %v  was: %v\n", cv, bv)
	}

	bv = avp.Bind("{FilePath}")
	cv = afpath
	if bv != cv {
		t.Errorf("bind error: should have been: %v  was: %v\n", cv, bv)
	}

	bv = avp.Bind("\\{FilePath}")
	cv = "{FilePath}"
	if bv != cv {
		t.Errorf("bind error: should have been: %v  was: %v\n", cv, bv)
	}

	bv = avp.Bind("This is the: {FilePath} and so on")
	cv = "This is the: " + afpath + " and so on"
	if bv != cv {
		t.Errorf("bind error: should have been: %v  was: %v\n", cv, bv)
	}

	bv = avp.Bind("{FileDir}/{Filename}")
	cv = filepath.Join("code", "argvars_test.go")
	if bv != cv {
		t.Errorf("bind error: should have been: %v  was: %v\n", cv, bv)
	}

	bv = avp.Bind("{FileDirProjRel}/{FilenameNoExt}")
	cv = filepath.Join("cogent", "code", "argvars_test")
	if bv != cv {
		t.Errorf("bind error: should have been: %v  was: %v\n", cv, bv)
	}
}
