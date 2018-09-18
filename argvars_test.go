// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"testing"

	"github.com/goki/gi"
	"github.com/goki/gi/giv"
)

func TestBind(t *testing.T) {
	tv := giv.TextView{}
	tv.CursorPos = giv.TextPos{22, 44}
	tv.SelectReg.Start = giv.TextPos{11, 14}
	tv.SelectReg.End = giv.TextPos{55, 0}

	fpath := "/Users/oreilly/go/src/github.com/goki/gide/argvars_test.go"
	projpath := "/Users/oreilly/go/src/github.com"

	pp := ProjPrefs{}
	pp.ProjRoot = gi.FileName(projpath)

	SetArgVarVals(&ArgVarVals, fpath, &pp, &tv)

	bv := BindArgVars("FilePath")
	cv := "FilePath"
	if bv != cv {
		t.Errorf("bind error: should have been: %v  was: %v\n", cv, bv)
	}

	bv = BindArgVars("{FilePath}")
	cv = fpath
	if bv != fpath {
		t.Errorf("bind error: should have been: %v  was: %v\n", cv, bv)
	}

	bv = BindArgVars("\\{FilePath}")
	cv = "{FilePath}"
	if bv != cv {
		t.Errorf("bind error: should have been: %v  was: %v\n", cv, bv)
	}

	bv = BindArgVars("This is the: {FilePath} and so on")
	cv = "This is the: " + fpath + " and so on"
	if bv != cv {
		t.Errorf("bind error: should have been: %v  was: %v\n", cv, bv)
	}

	bv = BindArgVars("{FileDir}/{FileName}")
	cv = "gide/argvars_test.go"
	if bv != cv {
		t.Errorf("bind error: should have been: %v  was: %v\n", cv, bv)
	}

	bv = BindArgVars("{FileDirProjRel}/{FileNameNoExt}")
	cv = "goki/gide/argvars_test"
	if bv != cv {
		t.Errorf("bind error: should have been: %v  was: %v\n", cv, bv)
	}
}
