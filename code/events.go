// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"log/slog"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/views"
)

func (cv *CodeView) codeViewKeys(e events.Event) {
	SetGoMod(cv.Settings.GoMod)
	kc := e.KeyChord()
	kf := keymap.Of(kc)
	if core.DebugSettings.KeyEventTrace {
		slog.Info("CodeView KeyInput", "widget", cv, "keymap", kf, kc)
	}
	if cv.KeySeq1 != "" {
		kc2 := string(cv.KeySeq1) + " " + string(kc)
		kf2 := keymap.Of(key.Chord(kc2))
		if kf2 == keymap.None || kf == keymap.CancelSelect || kc == "Escape" {
			if core.DebugSettings.KeyEventTrace {
				fmt.Printf("KeyMap sequence: %v aborted\n", cv.KeySeq1)
			}
			cv.SetStatus(string(cv.KeySeq1) + " -- aborted")
			e.SetHandled() // abort key sequence, don't send esc to anyone else
			cv.KeySeq1 = ""
			return
		}
		cv.SetStatus(kc2)
		cv.KeySeq1 = ""
		kf = kf2
	} else {
		if kf == keymap.MultiA || kf == keymap.MultiB {
			e.SetHandled()
			tv := cv.ActiveTextEditor()
			if tv != nil {
				tv.CancelComplete()
			}
			cv.KeySeq1 = e.KeyChord()
			cv.SetStatus(string(cv.KeySeq1))
			if core.DebugSettings.KeyEventTrace {
				fmt.Printf("KeyMap sequence needs 2 after: %v\n", cv.KeySeq1)
			}
			return
		} else if kf != keymap.None {
			if core.DebugSettings.KeyEventTrace {
				fmt.Printf("KeyMap got in one: %v = %v\n", cv.KeySeq1, kf)
			}
		}
	}

	atv := cv.ActiveTextEditor()
	switch kf {
	case keymap.Find:
		e.SetHandled()
		if atv != nil && atv.HasSelection() {
			cv.Settings.Find.Find = string(atv.Selection().ToBytes())
		}
		cv.CallFind(atv)
	}
	if e.IsHandled() {
		return
	}
	switch kf {
	case KeyNextPanel:
		e.SetHandled()
		cv.FocusNextPanel()
	case KeyPrevPanel:
		e.SetHandled()
		cv.FocusPrevPanel()
	case KeyFileOpen:
		e.SetHandled()
		cv.CallViewFile(atv)
	case KeyBufSelect:
		e.SetHandled()
		cv.SelectOpenNode()
	case KeyBufClone:
		e.SetHandled()
		cv.CloneActiveView()
	case KeyBufSave:
		e.SetHandled()
		cv.SaveActiveView()
	case KeyBufSaveAs:
		e.SetHandled()
		cv.CallSaveActiveViewAs(atv)
	case KeyBufClose:
		e.SetHandled()
		cv.CloseActiveView()
	case KeyExecCmd:
		e.SetHandled()
		views.CallFunc(atv, cv.ExecCmd)
	case KeyRectCut:
		e.SetHandled()
		cv.CutRect()
	case KeyRectCopy:
		e.SetHandled()
		cv.CopyRect()
	case KeyRectPaste:
		e.SetHandled()
		cv.PasteRect()
	case KeyRegCopy:
		e.SetHandled()
		views.CallFunc(atv, cv.RegisterCopy)
	case KeyRegPaste:
		e.SetHandled()
		views.CallFunc(atv, cv.RegisterPaste)
	case KeyCommentOut:
		e.SetHandled()
		cv.CommentOut()
	case KeyIndent:
		e.SetHandled()
		cv.Indent()
	case KeyJump:
		e.SetHandled()
		tv := cv.ActiveTextEditor()
		if tv != nil {
			tv.JumpToLinePrompt()
		}
	case KeySetSplit:
		e.SetHandled()
		cv.CallSplitsSetView(atv)
	case KeyBuildProject:
		e.SetHandled()
		cv.RunBuild()
	case KeyRunProject:
		e.SetHandled()
		cv.Run()
	}
}
