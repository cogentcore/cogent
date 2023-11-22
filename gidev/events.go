// Copyright (c) 2023, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidev

import (
	"fmt"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/gide/v2/gide"
	"goki.dev/goosi/events"
)

func (ge *GideView) HandleGideViewEvents() {
	// if ge.HasAnyScroll() {
	// 	ge.LayoutScrollEvents()
	// }
	// ge.HandleLayoutEvents()
	ge.HandleGideKeyEvent()
	ge.HandleOSFileEvent()
}

func (ge *GideView) HandleGideKeyEvent() {
	ge.PriorityEvents = []events.Types{events.KeyChord}
	ge.OnKeyChord(func(e events.Event) {
		ge.GideViewKeys(e)
	})
}

func (ge *GideView) GideViewKeys(kt events.Event) {
	gide.SetGoMod(ge.Prefs.GoMod)
	var kf gide.KeyFuns
	kc := kt.KeyChord()
	if gi.KeyEventTrace {
		fmt.Printf("GideView KeyInput: %v\n", ge.Path())
	}
	gkf := keyfun.Of(kc)
	if ge.KeySeq1 != "" {
		kf = gide.KeyFun(ge.KeySeq1, kc)
		seqstr := string(ge.KeySeq1) + " " + string(kc)
		if kf == gide.KeyFunNil || kc == "Escape" {
			if gi.KeyEventTrace {
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
			tv := ge.ActiveTextView()
			if tv != nil {
				tv.CancelComplete()
			}
			ge.KeySeq1 = kt.KeyChord()
			ge.SetStatus(string(ge.KeySeq1))
			if gi.KeyEventTrace {
				fmt.Printf("gide.KeyFun sequence needs 2 after: %v\n", ge.KeySeq1)
			}
			return
		} else if kf != gide.KeyFunNil {
			if gi.KeyEventTrace {
				fmt.Printf("gide.KeyFun got in one: %v = %v\n", ge.KeySeq1, kf)
			}
			gkf = keyfun.Nil // override!
		}
	}

	switch gkf {
	case keyfun.Find:
		kt.SetHandled()
		tv := ge.ActiveTextView()
		if tv != nil && tv.HasSelection() {
			ge.Prefs.Find.Find = string(tv.Selection().ToBytes())
		}
		giv.CallFunc(tv, ge.Find)
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
		giv.CallFunc(ge, ge.ViewFile)
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
		giv.CallFunc(ge, ge.SaveActiveViewAs)
	case gide.KeyFunBufClose:
		kt.SetHandled()
		ge.CloseActiveView()
	case gide.KeyFunExecCmd:
		kt.SetHandled()
		giv.CallFunc(ge, ge.ExecCmd)
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
		giv.CallFunc(ge, ge.RegisterCopy)
	case gide.KeyFunRegPaste:
		kt.SetHandled()
		giv.CallFunc(ge, ge.RegisterPaste)
	case gide.KeyFunCommentOut:
		kt.SetHandled()
		ge.CommentOut()
	case gide.KeyFunIndent:
		kt.SetHandled()
		ge.Indent()
	case gide.KeyFunJump:
		kt.SetHandled()
		tv := ge.ActiveTextView()
		if tv != nil {
			tv.JumpToLineAddText()
		}
		ge.Indent()
	case gide.KeyFunSetSplit:
		kt.SetHandled()
		giv.CallFunc(ge, ge.SplitsSetView)
	case gide.KeyFunBuildProj:
		kt.SetHandled()
		ge.Build()
	case gide.KeyFunRunProj:
		kt.SetHandled()
		ge.Run()
	}
}

func (ge *GideView) HandleOSFileEvent() {
	ge.On(events.OSOpenFiles, func(e events.Event) {
		ofe := e.(*events.OSFiles)
		for _, fn := range ofe.Files {
			ge.OpenFile(fn)
		}
	})
}

/*

func (ge *GideView) MouseEvent() {
	ge.ConnectEvent(oswin.MouseEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
		gee := recv.Embed(KiT_GideView).(*GideView)
		gide.SetGoMod(gee.Prefs.GoMod)
	})
}
*/
