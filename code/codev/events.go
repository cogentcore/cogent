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
	code.SetGoMod(ge.Settings.GoMod)
	var kf code.KeyFunctions
	kc := kt.KeyChord()
	gkf := keymap.Of(kc)
	if core.DebugSettings.KeyEventTrace {
		slog.Info("CodeView KeyInput", "widget", ge, "keyfun", gkf)
	}
	if ge.KeySeq1 != "" {
		kf = code.KeyFunction(ge.KeySeq1, kc)
		seqstr := string(ge.KeySeq1) + " " + string(kc)
		if kf == code.KeyNone || kc == "Escape" {
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
		gkf = keymap.None // override!
	} else {
		kf = code.KeyFunction(kc, "")
		if kf == code.KeyNeeds2 {
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
		} else if kf != code.KeyNone {
			if core.DebugSettings.KeyEventTrace {
				fmt.Printf("code.KeyFun got in one: %v = %v\n", ge.KeySeq1, kf)
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
	case code.KeyNextPanel:
		kt.SetHandled()
		ge.FocusNextPanel()
	case code.KeyPrevPanel:
		kt.SetHandled()
		ge.FocusPrevPanel()
	case code.KeyFileOpen:
		kt.SetHandled()
		ge.CallViewFile(atv)
	case code.KeyBufSelect:
		kt.SetHandled()
		ge.SelectOpenNode()
	case code.KeyBufClone:
		kt.SetHandled()
		ge.CloneActiveView()
	case code.KeyBufSave:
		kt.SetHandled()
		ge.SaveActiveView()
	case code.KeyBufSaveAs:
		kt.SetHandled()
		ge.CallSaveActiveViewAs(atv)
	case code.KeyBufClose:
		kt.SetHandled()
		ge.CloseActiveView()
	case code.KeyExecCmd:
		kt.SetHandled()
		views.CallFunc(atv, ge.ExecCmd)
	case code.KeyRectCut:
		kt.SetHandled()
		ge.CutRect()
	case code.KeyRectCopy:
		kt.SetHandled()
		ge.CopyRect()
	case code.KeyRectPaste:
		kt.SetHandled()
		ge.PasteRect()
	case code.KeyRegCopy:
		kt.SetHandled()
		views.CallFunc(atv, ge.RegisterCopy)
	case code.KeyRegPaste:
		kt.SetHandled()
		views.CallFunc(atv, ge.RegisterPaste)
	case code.KeyCommentOut:
		kt.SetHandled()
		ge.CommentOut()
	case code.KeyIndent:
		kt.SetHandled()
		ge.Indent()
	case code.KeyJump:
		kt.SetHandled()
		tv := ge.ActiveTextEditor()
		if tv != nil {
			tv.JumpToLinePrompt()
		}
	case code.KeySetSplit:
		kt.SetHandled()
		ge.CallSplitsSetView(atv)
	case code.KeyBuildProject:
		kt.SetHandled()
		ge.Build()
	case code.KeyRunProject:
		kt.SetHandled()
		ge.Run()
	}
}
