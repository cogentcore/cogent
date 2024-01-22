// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"testing"

	"cogentcore.org/core/gi"
	"cogentcore.org/core/pi/lex"
	"cogentcore.org/core/texteditor"
)

func TestBind(t *testing.T) {
	tv := texteditor.Editor{}
	tv.CursorPos = lex.Pos{22, 44}
	tv.SelectReg.Start = lex.Pos{11, 14}
	tv.SelectReg.End = lex.Pos{55, 0}

	fpath := "/Users/oreilly/go/src/cogentcore.org/cogent/code/argvars_test.go"
	projpath := "/Users/oreilly/go/src/github.com"

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
	cv = fpath
	if bv != fpath {
		t.Errorf("bind error: should have been: %v  was: %v\n", cv, bv)
	}

	bv = avp.Bind("\\{FilePath}")
	cv = "{FilePath}"
	if bv != cv {
		t.Errorf("bind error: should have been: %v  was: %v\n", cv, bv)
	}

	bv = avp.Bind("This is the: {FilePath} and so on")
	cv = "This is the: " + fpath + " and so on"
	if bv != cv {
		t.Errorf("bind error: should have been: %v  was: %v\n", cv, bv)
	}

	bv = avp.Bind("{FileDir}/{Filename}")
	cv = "code/argvars_test.go"
	if bv != cv {
		t.Errorf("bind error: should have been: %v  was: %v\n", cv, bv)
	}

	bv = avp.Bind("{FileDirProjRel}/{FilenameNoExt}")
	cv = "goki/code/argvars_test"
	if bv != cv {
		t.Errorf("bind error: should have been: %v  was: %v\n", cv, bv)
	}
}
