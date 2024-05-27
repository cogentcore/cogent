// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"log/slog"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/views"
)

func (ge *CodeView) codeViewKeys(e events.Event) {
	SetGoMod(ge.Settings.GoMod)
	var kf KeyFunctions
	kc := e.KeyChord()
	gkf := keymap.Of(kc)
	if core.DebugSettings.KeyEventTrace {
		slog.Info("CodeView KeyInput", "widget", ge, "keyfun", gkf)
	}
	if ge.KeySeq1 != "" {
		kf = KeyFunction(ge.KeySeq1, kc)
		seqstr := string(ge.KeySeq1) + " " + string(kc)
		if kf == KeyNone || kc == "Escape" {
			if core.DebugSettings.KeyEventTrace {
				fmt.Printf("KeyFun sequence: %v aborted\n", seqstr)
			}
			ge.SetStatus(seqstr + " -- aborted")
			e.SetHandled() // abort key sequence, don't send esc to anyone else
			ge.KeySeq1 = ""
			return
		}
		ge.SetStatus(seqstr)
		ge.KeySeq1 = ""
		gkf = keymap.None // override!
	} else {
		kf = KeyFunction(kc, "")
		if kf == KeyNeeds2 {
			e.SetHandled()
			tv := ge.ActiveTextEditor()
			if tv != nil {
				tv.CancelComplete()
			}
			ge.KeySeq1 = e.KeyChord()
			ge.SetStatus(string(ge.KeySeq1))
			if core.DebugSettings.KeyEventTrace {
				fmt.Printf("KeyFun sequence needs 2 after: %v\n", ge.KeySeq1)
			}
			return
		} else if kf != KeyNone {
			if core.DebugSettings.KeyEventTrace {
				fmt.Printf("KeyFun got in one: %v = %v\n", ge.KeySeq1, kf)
			}
			gkf = keymap.None // override!
		}
	}

	atv := ge.ActiveTextEditor()
	switch gkf {
	case keymap.Find:
		e.SetHandled()
		if atv != nil && atv.HasSelection() {
			ge.Settings.Find.Find = string(atv.Selection().ToBytes())
		}
		ge.CallFind(atv)
	}
	if e.IsHandled() {
		return
	}
	switch kf {
	case KeyNextPanel:
		e.SetHandled()
		ge.FocusNextPanel()
	case KeyPrevPanel:
		e.SetHandled()
		ge.FocusPrevPanel()
	case KeyFileOpen:
		e.SetHandled()
		ge.CallViewFile(atv)
	case KeyBufSelect:
		e.SetHandled()
		ge.SelectOpenNode()
	case KeyBufClone:
		e.SetHandled()
		ge.CloneActiveView()
	case KeyBufSave:
		e.SetHandled()
		ge.SaveActiveView()
	case KeyBufSaveAs:
		e.SetHandled()
		ge.CallSaveActiveViewAs(atv)
	case KeyBufClose:
		e.SetHandled()
		ge.CloseActiveView()
	case KeyExecCmd:
		e.SetHandled()
		views.CallFunc(atv, ge.ExecCmd)
	case KeyRectCut:
		e.SetHandled()
		ge.CutRect()
	case KeyRectCopy:
		e.SetHandled()
		ge.CopyRect()
	case KeyRectPaste:
		e.SetHandled()
		ge.PasteRect()
	case KeyRegCopy:
		e.SetHandled()
		views.CallFunc(atv, ge.RegisterCopy)
	case KeyRegPaste:
		e.SetHandled()
		views.CallFunc(atv, ge.RegisterPaste)
	case KeyCommentOut:
		e.SetHandled()
		ge.CommentOut()
	case KeyIndent:
		e.SetHandled()
		ge.Indent()
	case KeyJump:
		e.SetHandled()
		tv := ge.ActiveTextEditor()
		if tv != nil {
			tv.JumpToLinePrompt()
		}
	case KeySetSplit:
		e.SetHandled()
		ge.CallSplitsSetView(atv)
	case KeyBuildProject:
		e.SetHandled()
		ge.Build()
	case KeyRunProject:
		e.SetHandled()
		ge.Run()
	}
}
