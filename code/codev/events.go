// Copyright (c) 2023, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidev

import (
	"fmt"

	"cogentcore.org/cogent/code/code/gide"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/keyfun"
)

func (ge *GideView) HandleEvents() {
	ge.PriorityEvents = []events.Types{events.KeyChord}
	ge.OnKeyChord(func(e events.Event) {
		ge.GideViewKeys(e)
	})
	ge.On(events.OSOpenFiles, func(e events.Event) {
		ofe := e.(*events.OSFiles)
		for _, fn := range ofe.Files {
			ge.OpenFile(fn)
		}
	})
}

func (ge *GideView) GideViewKeys(kt events.Event) {
	gide.SetGoMod(ge.Prefs.GoMod)
	var kf gide.KeyFuns
	kc := kt.KeyChord()
	if gi.DebugSettings.KeyEventTrace {
		fmt.Printf("GideView KeyInput: %v\n", ge.Path())
	}
	gkf := keyfun.Of(kc)
	if ge.KeySeq1 != "" {
		kf = gide.KeyFun(ge.KeySeq1, kc)
		seqstr := string(ge.KeySeq1) + " " + string(kc)
		if kf == gide.KeyFunNil || kc == "Escape" {
			if gi.DebugSettings.KeyEventTrace {
				fmt.Printf("gide.KeyFun sequence: %v aborted\n", seqstr)
			}
			ge.SetStatus(seqstr + " -- aborted")
			kt.SetHandled() // abort key sequence, don't send esc to anyone else
			ge.KeySeq1 = ""
			return
		}
		ge.SetStatus(seqstr)
		ge.KeySeq1 = ""
		gkf = keyfun.Nil // override!
	} else {
		kf = gide.KeyFun(kc, "")
		if kf == gide.KeyFunNeeds2 {
			kt.SetHandled()
			tv := ge.ActiveTextEditor()
			if tv != nil {
				tv.CancelComplete()
			}
			ge.KeySeq1 = kt.KeyChord()
			ge.SetStatus(string(ge.KeySeq1))
			if gi.DebugSettings.KeyEventTrace {
				fmt.Printf("gide.KeyFun sequence needs 2 after: %v\n", ge.KeySeq1)
			}
			return
		} else if kf != gide.KeyFunNil {
			if gi.DebugSettings.KeyEventTrace {
				fmt.Printf("gide.KeyFun got in one: %v = %v\n", ge.KeySeq1, kf)
			}
			gkf = keyfun.Nil // override!
		}
	}

	atv := ge.ActiveTextEditor()
	switch gkf {
	case keyfun.Find:
		kt.SetHandled()
		if atv != nil && atv.HasSelection() {
			ge.Prefs.Find.Find = string(atv.Selection().ToBytes())
		}
		ge.CallFind(atv)
	}
	if kt.IsHandled() {
		return
	}
	switch kf {
	case gide.KeyFunNextPanel:
		kt.SetHandled()
		ge.FocusNextPanel()
	case gide.KeyFunPrevPanel:
		kt.SetHandled()
		ge.FocusPrevPanel()
	case gide.KeyFunFileOpen:
		kt.SetHandled()
		ge.CallViewFile(atv)
	case gide.KeyFunBufSelect:
		kt.SetHandled()
		ge.SelectOpenNode()
	case gide.KeyFunBufClone:
		kt.SetHandled()
		ge.CloneActiveView()
	case gide.KeyFunBufSave:
		kt.SetHandled()
		ge.SaveActiveView()
	case gide.KeyFunBufSaveAs:
		kt.SetHandled()
		ge.CallSaveActiveViewAs(atv)
	case gide.KeyFunBufClose:
		kt.SetHandled()
		ge.CloseActiveView()
	case gide.KeyFunExecCmd:
		kt.SetHandled()
		giv.CallFunc(atv, ge.ExecCmd)
	case gide.KeyFunRectCut:
		kt.SetHandled()
		ge.CutRect()
	case gide.KeyFunRectCopy:
		kt.SetHandled()
		ge.CopyRect()
	case gide.KeyFunRectPaste:
		kt.SetHandled()
		ge.PasteRect()
	case gide.KeyFunRegCopy:
		kt.SetHandled()
		giv.CallFunc(atv, ge.RegisterCopy)
	case gide.KeyFunRegPaste:
		kt.SetHandled()
		giv.CallFunc(atv, ge.RegisterPaste)
	case gide.KeyFunCommentOut:
		kt.SetHandled()
		ge.CommentOut()
	case gide.KeyFunIndent:
		kt.SetHandled()
		ge.Indent()
	case gide.KeyFunJump:
		kt.SetHandled()
		tv := ge.ActiveTextEditor()
		if tv != nil {
			tv.JumpToLineAddText()
		}
		ge.Indent()
	case gide.KeyFunSetSplit:
		kt.SetHandled()
		ge.CallSplitsSetView(atv)
	case gide.KeyFunBuildProj:
		kt.SetHandled()
		ge.Build()
	case gide.KeyFunRunProj:
		kt.SetHandled()
		ge.Run()
	}
}
