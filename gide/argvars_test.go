// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"testing"

	"goki.dev/gi"
	"goki.dev/pi/lex"
	"goki.dev/texteditor"
)

func TestBind(t *testing.T) {
	tv := texteditor.Editor{}
	tv.CursorPos = lex.Pos{22, 44}
	tv.SelectReg.Start = lex.Pos{11, 14}
	tv.SelectReg.End = lex.Pos{55, 0}

	fpath := "/Users/oreilly/go/src/github.com/goki/gide/v2/argvars_test.go"
	projpath := "/Users/oreilly/go/src/github.com"

	pp := ProjPrefs{}
	pp.ProjRoot = gi.FileName(projpath)

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

	bv = avp.Bind("{FileDir}/{FileName}")
	cv = "gide/argvars_test.go"
	if bv != cv {
		t.Errorf("bind error: should have been: %v  was: %v\n", cv, bv)
	}

	bv = avp.Bind("{FileDirProjRel}/{FileNameNoExt}")
	cv = "goki/gide/argvars_test"
	if bv != cv {
		t.Errorf("bind error: should have been: %v  was: %v\n", cv, bv)
	}
}
