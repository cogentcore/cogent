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
	SetGoMod(ge.Settings.GoMod)
	var kf KeyFunctions
	kc := kt.KeyChord()
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
			kt.SetHandled() // abort key sequence, don't send esc to anyone else
			ge.KeySeq1 = ""
			return
		}
		ge.SetStatus(seqstr)
		ge.KeySeq1 = ""
		gkf = keymap.None // override!
	} else {
		kf = KeyFunction(kc, "")
		if kf == KeyNeeds2 {
			kt.SetHandled()
			tv := ge.ActiveTextEditor()
			if tv != nil {
				tv.CancelComplete()
			}
			ge.KeySeq1 = kt.KeyChord()
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
	case KeyNextPanel:
		kt.SetHandled()
		ge.FocusNextPanel()
	case KeyPrevPanel:
		kt.SetHandled()
		ge.FocusPrevPanel()
	case KeyFileOpen:
		kt.SetHandled()
		ge.CallViewFile(atv)
	case KeyBufSelect:
		kt.SetHandled()
		ge.SelectOpenNode()
	case KeyBufClone:
		kt.SetHandled()
		ge.CloneActiveView()
	case KeyBufSave:
		kt.SetHandled()
		ge.SaveActiveView()
	case KeyBufSaveAs:
		kt.SetHandled()
		ge.CallSaveActiveViewAs(atv)
	case KeyBufClose:
		kt.SetHandled()
		ge.CloseActiveView()
	case KeyExecCmd:
		kt.SetHandled()
		views.CallFunc(atv, ge.ExecCmd)
	case KeyRectCut:
		kt.SetHandled()
		ge.CutRect()
	case KeyRectCopy:
		kt.SetHandled()
		ge.CopyRect()
	case KeyRectPaste:
		kt.SetHandled()
		ge.PasteRect()
	case KeyRegCopy:
		kt.SetHandled()
		views.CallFunc(atv, ge.RegisterCopy)
	case KeyRegPaste:
		kt.SetHandled()
		views.CallFunc(atv, ge.RegisterPaste)
	case KeyCommentOut:
		kt.SetHandled()
		ge.CommentOut()
	case KeyIndent:
		kt.SetHandled()
		ge.Indent()
	case KeyJump:
		kt.SetHandled()
		tv := ge.ActiveTextEditor()
		if tv != nil {
			tv.JumpToLinePrompt()
		}
	case KeySetSplit:
		kt.SetHandled()
		ge.CallSplitsSetView(atv)
	case KeyBuildProject:
		kt.SetHandled()
		ge.Build()
	case KeyRunProject:
		kt.SetHandled()
		ge.Run()
	}
}
