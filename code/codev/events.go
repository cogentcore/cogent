// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codev

import (
	"fmt"
	"log/slog"

	"cogentcore.org/cogent/code/code"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/keyfun"
)

func (ge *CodeView) HandleEvents() {
	ge.OnFirst(events.KeyChord, func(e events.Event) {
		ge.CodeViewKeys(e)
	})
	ge.On(events.OSOpenFiles, func(e events.Event) {
		ofe := e.(*events.OSFiles)
		for _, fn := range ofe.Files {
			ge.OpenFile(fn)
		}
	})
}

func (ge *CodeView) CodeViewKeys(kt events.Event) {
	code.SetGoMod(ge.Settings.GoMod)
	var kf code.KeyFuns
	kc := kt.KeyChord()
	gkf := keyfun.Of(kc)
	if core.DebugSettings.KeyEventTrace {
		slog.Info("CodeView KeyInput", "widget", ge, "keyfun", gkf)
	}
	if ge.KeySeq1 != "" {
		kf = code.KeyFun(ge.KeySeq1, kc)
		seqstr := string(ge.KeySeq1) + " " + string(kc)
		if kf == code.KeyFunNil || kc == "Escape" {
			if core.DebugSettings.KeyEventTrace {
				fmt.Printf("code.KeyFun sequence: %v aborted\n", seqstr)
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
		kf = code.KeyFun(kc, "")
		if kf == code.KeyFunNeeds2 {
			kt.SetHandled()
			tv := ge.ActiveTextEditor()
			if tv != nil {
				tv.CancelComplete()
			}
			ge.KeySeq1 = kt.KeyChord()
			ge.SetStatus(string(ge.KeySeq1))
			if core.DebugSettings.KeyEventTrace {
				fmt.Printf("code.KeyFun sequence needs 2 after: %v\n", ge.KeySeq1)
			}
			return
		} else if kf != code.KeyFunNil {
			if core.DebugSettings.KeyEventTrace {
				fmt.Printf("code.KeyFun got in one: %v = %v\n", ge.KeySeq1, kf)
			}
			gkf = keyfun.Nil // override!
		}
	}

	atv := ge.ActiveTextEditor()
	switch gkf {
	case keyfun.Find:
		kt.SetHandled()
		if atv != nil && atv.HasSelection() {
			ge.Settings.Find.Find = string(atv.Selection().ToBytes())
		}
		ge.CallFind(atv)
	}
	if kt.IsHandled() {
		return
	}
	switch kf {
	case code.KeyFunNextPanel:
		kt.SetHandled()
		ge.FocusNextPanel()
	case code.KeyFunPrevPanel:
		kt.SetHandled()
		ge.FocusPrevPanel()
	case code.KeyFunFileOpen:
		kt.SetHandled()
		ge.CallViewFile(atv)
	case code.KeyFunBufSelect:
		kt.SetHandled()
		ge.SelectOpenNode()
	case code.KeyFunBufClone:
		kt.SetHandled()
		ge.CloneActiveView()
	case code.KeyFunBufSave:
		kt.SetHandled()
		ge.SaveActiveView()
	case code.KeyFunBufSaveAs:
		kt.SetHandled()
		ge.CallSaveActiveViewAs(atv)
	case code.KeyFunBufClose:
		kt.SetHandled()
		ge.CloseActiveView()
	case code.KeyFunExecCmd:
		kt.SetHandled()
		giv.CallFunc(atv, ge.ExecCmd)
	case code.KeyFunRectCut:
		kt.SetHandled()
		ge.CutRect()
	case code.KeyFunRectCopy:
		kt.SetHandled()
		ge.CopyRect()
	case code.KeyFunRectPaste:
		kt.SetHandled()
		ge.PasteRect()
	case code.KeyFunRegCopy:
		kt.SetHandled()
		giv.CallFunc(atv, ge.RegisterCopy)
	case code.KeyFunRegPaste:
		kt.SetHandled()
		giv.CallFunc(atv, ge.RegisterPaste)
	case code.KeyFunCommentOut:
		kt.SetHandled()
		ge.CommentOut()
	case code.KeyFunIndent:
		kt.SetHandled()
		ge.Indent()
	case code.KeyFunJump:
		kt.SetHandled()
		tv := ge.ActiveTextEditor()
		if tv != nil {
			tv.JumpToLinePrompt()
		}
	case code.KeyFunSetSplit:
		kt.SetHandled()
		ge.CallSplitsSetView(atv)
	case code.KeyFunBuildProj:
		kt.SetHandled()
		ge.Build()
	case code.KeyFunRunProj:
		kt.SetHandled()
		ge.Run()
	}
}
