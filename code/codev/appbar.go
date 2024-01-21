// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codev

import (
	"strings"

	"cogentcore.org/cogent/code/code"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/fi/uri"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
)

func (ge *CodeView) AppBarConfig(pw gi.Widget) {
	tb := gi.RecycleToolbar(pw)
	// StdAppBarStart(tb)
	gi.StdAppBarBack(tb)
	ac := gi.StdAppBarChooser(tb)
	ac.Resources.Add(ge.ResourceCommands)
	ac.Resources.Add(ge.ResourceFiles)
	ac.Resources.Add(ge.ResourceSymbols)

	gi.StdOverflowMenu(tb)
	gi.CurrentWindowAppBar(tb)
	// apps should add their own app-general functions here
}

func (ge *CodeView) ConfigToolbar(tb *gi.Toolbar) { //gti:add
	giv.NewFuncButton(tb, ge.UpdateFiles).SetText("").SetIcon(icons.Refresh).SetShortcut("Command+U")
	sm := gi.NewSwitch(tb, "go-mod").SetText("Go Mod").SetTooltip("Toggles the use of go modules -- saved with project -- if off, uses old school GOPATH mode")
	sm.Style(func(s *styles.Style) {
		sm.SetChecked(ge.Prefs.GoMod)
	})
	sm.OnChange(func(e events.Event) {
		ge.Prefs.GoMod = sm.StateIs(states.Checked)
		code.SetGoMod(ge.Prefs.GoMod)
	})

	gi.NewSeparator(tb)
	gi.NewButton(tb).SetText("Open Recent").SetMenu(func(m *gi.Scene) {
		for _, sp := range code.SavedPaths {
			sp := sp
			if sp == code.SavedPathsExtras[0] {
				gi.NewSeparator(m)
			}
			gi.NewButton(m).SetText(sp).OnClick(func(e events.Event) {
				ge.OpenRecent(gi.Filename(sp))
			})
		}
	})

	ge.ConfigActiveFilename(giv.NewFuncButton(tb, ge.OpenPath).
		SetText("Open").SetIcon(icons.Open).SetKey(keyfun.Open))

	giv.NewFuncButton(tb, ge.SaveActiveView).SetText("Save").
		SetIcon(icons.Save).SetKey(keyfun.Save)

	giv.NewFuncButton(tb, ge.SaveAll).SetIcon(icons.Save)

	gi.NewSeparator(tb)

	giv.NewFuncButton(tb, ge.CursorToHistPrev).SetText("").SetKey(keyfun.HistPrev).
		SetIcon(icons.KeyboardArrowLeft).SetShowReturn(false)
	giv.NewFuncButton(tb, ge.CursorToHistNext).SetText("").SetKey(keyfun.HistNext).
		SetIcon(icons.KeyboardArrowRight).SetShowReturn(false)

	gi.NewSeparator(tb)

	ge.ConfigFindButton(giv.NewFuncButton(tb, ge.Find).SetIcon(icons.FindReplace))

	gi.NewSeparator(tb)

	giv.NewFuncButton(tb, ge.Symbols).SetIcon(icons.List)

	giv.NewFuncButton(tb, ge.Spell).SetIcon(icons.Spellcheck)

	gi.NewSeparator(tb)

	giv.NewFuncButton(tb, ge.Build).SetIcon(icons.Build).
		SetShortcut(key.Chord(code.ChordForFun(code.KeyFunBuildProj).String()))

	giv.NewFuncButton(tb, ge.Run).SetIcon(icons.PlayArrow).
		SetShortcut(key.Chord(code.ChordForFun(code.KeyFunRunProj).String()))

	giv.NewFuncButton(tb, ge.Debug).SetIcon(icons.Debug)

	giv.NewFuncButton(tb, ge.DebugTest).SetIcon(icons.Debug)

	gi.NewSeparator(tb)

	giv.NewFuncButton(tb, ge.Commit).SetIcon(icons.Star)

	gi.NewButton(tb).SetText("Command").
		SetShortcut(key.Chord(code.ChordForFun(code.KeyFunExecCmd).String())).
		SetMenu(func(m *gi.Scene) {
			ec := ExecCmds(ge)
			for _, cc := range ec {
				cc := cc
				cat := cc[0]
				gi.NewButton(m).SetText(cat).SetMenu(func(mm *gi.Scene) {
					nc := len(cc)
					for i := 1; i < nc; i++ {
						cm := cc[i]
						gi.NewButton(mm).SetText(cm).OnClick(func(e events.Event) {
							e.SetHandled()
							ge.ExecCmdNameActive(code.CommandName(cat, cm))
						})
					}
				})
			}
		})

	gi.NewSeparator(tb)

	gi.NewButton(tb).SetText("Splits").SetMenu(func(m *gi.Scene) {
		gi.NewButton(m).SetText("Set View").
			SetMenu(func(mm *gi.Scene) {
				for _, sp := range code.AvailSplitNames {
					sn := code.SplitName(sp)
					mb := gi.NewButton(mm).SetText(sp).OnClick(func(e events.Event) {
						ge.SplitsSetView(sn)
					})
					if sn == ge.Prefs.SplitName {
						mb.SetSelected(true)
					}
				}
			})
		giv.NewFuncButton(m, ge.SplitsSaveAs).SetText("Save As")
		gi.NewButton(m).SetText("Save").
			SetMenu(func(mm *gi.Scene) {
				for _, sp := range code.AvailSplitNames {
					sn := code.SplitName(sp)
					mb := gi.NewButton(mm).SetText(sp).OnClick(func(e events.Event) {
						ge.SplitsSave(sn)
					})
					if sn == ge.Prefs.SplitName {
						mb.SetSelected(true)
					}
				}
			})
		giv.NewFuncButton(m, ge.SplitsEdit).SetText("Edit")
	})

	tb.AddOverflowMenu(func(m *gi.Scene) {
		gi.NewButton(m).SetText("File").SetMenu(func(mm *gi.Scene) {
			giv.NewFuncButton(mm, ge.NewProj).SetText("New Project").
				SetIcon(icons.NewWindow).SetKey(keyfun.New)

			giv.NewFuncButton(mm, ge.NewFile).SetText("New File").
				SetIcon(icons.NewWindow)

			gi.NewSeparator(mm)

			giv.NewFuncButton(mm, ge.OpenProj).SetText("Open Project").
				SetIcon(icons.Open)

			gi.NewSeparator(mm)

			giv.NewFuncButton(mm, ge.EditProjPrefs).SetText("Project Prefs").
				SetIcon(icons.Edit)

			giv.NewFuncButton(mm, ge.SaveProj).SetText("Save Project").
				SetIcon(icons.Save)

			ge.ConfigActiveFilename(giv.NewFuncButton(mm, ge.SaveProjAs).
				SetText("Save Project As").SetIcon(icons.SaveAs))

			gi.NewSeparator(mm)

			giv.NewFuncButton(mm, ge.RevertActiveView).SetText("Revert File").
				SetIcon(icons.Undo)

			ge.ConfigActiveFilename(giv.NewFuncButton(mm, ge.SaveActiveViewAs).
				SetText("Save File As").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs))

		})

		gi.NewButton(m).SetText("Edit").SetMenu(func(mm *gi.Scene) {
			gi.NewButton(mm).SetText("Paste history").SetIcon(icons.Paste).
				SetKey(keyfun.PasteHist)

			giv.NewFuncButton(mm, ge.RegisterPaste).SetIcon(icons.Paste).
				SetShortcut(key.Chord(code.ChordForFun(code.KeyFunRegCopy).String()))

			giv.NewFuncButton(mm, ge.RegisterCopy).SetIcon(icons.Copy).
				SetShortcut(key.Chord(code.ChordForFun(code.KeyFunRegPaste).String()))

			gi.NewSeparator(mm)

			giv.NewFuncButton(mm, ge.CopyRect).SetIcon(icons.Copy).
				SetShortcut(key.Chord(code.ChordForFun(code.KeyFunRectCopy).String()))

			giv.NewFuncButton(mm, ge.CutRect).SetIcon(icons.Cut).
				SetShortcut(key.Chord(code.ChordForFun(code.KeyFunRectCut).String()))

			giv.NewFuncButton(mm, ge.PasteRect).SetIcon(icons.Paste).
				SetShortcut(key.Chord(code.ChordForFun(code.KeyFunRectPaste).String()))

			gi.NewSeparator(mm)

			gi.NewButton(mm).SetText("Undo").SetIcon(icons.Undo).SetKey(keyfun.Undo)

			gi.NewButton(mm).SetText("Redo").SetIcon(icons.Redo).SetKey(keyfun.Redo)

			gi.NewSeparator(mm)

			giv.NewFuncButton(mm, ge.ReplaceInActive).SetText("Replace in File").
				SetIcon(icons.FindReplace)

			gi.NewButton(mm).SetText("Show completions").SetIcon(icons.CheckCircle).SetKey(keyfun.Complete)

			gi.NewButton(mm).SetText("Lookup symbol").SetIcon(icons.Search).SetKey(keyfun.Lookup)

			gi.NewButton(mm).SetText("Jump to line").SetIcon(icons.GoToLine).SetKey(keyfun.Jump)

			gi.NewSeparator(mm)

			giv.NewFuncButton(mm, ge.CommentOut).SetText("Comment region").
				SetIcon(icons.Comment).SetShortcut(key.Chord(code.ChordForFun(code.KeyFunCommentOut).String()))

			giv.NewFuncButton(mm, ge.Indent).SetIcon(icons.FormatIndentIncrease).
				SetShortcut(key.Chord(code.ChordForFun(code.KeyFunIndent).String()))

			giv.NewFuncButton(mm, ge.ReCase).SetIcon(icons.MatchCase)

			giv.NewFuncButton(mm, ge.JoinParaLines).SetIcon(icons.Join)

			giv.NewFuncButton(mm, ge.TabsToSpaces).SetIcon(icons.TabMove)

			giv.NewFuncButton(mm, ge.SpacesToTabs).SetIcon(icons.TabMove)
		})

		gi.NewButton(m).SetText("View").SetMenu(func(mm *gi.Scene) {
			giv.NewFuncButton(mm, ge.FocusPrevPanel).SetText("Focus prev").SetIcon(icons.KeyboardArrowLeft).
				SetShortcut(key.Chord(code.ChordForFun(code.KeyFunPrevPanel).String()))
			giv.NewFuncButton(mm, ge.FocusNextPanel).SetText("Focus next").SetIcon(icons.KeyboardArrowRight).
				SetShortcut(key.Chord(code.ChordForFun(code.KeyFunNextPanel).String()))
			giv.NewFuncButton(mm, ge.CloneActiveView).SetText("Clone active").SetIcon(icons.Copy).
				SetShortcut(key.Chord(code.ChordForFun(code.KeyFunBufClone).String()))
			gi.NewSeparator(m)
			giv.NewFuncButton(mm, ge.CloseActiveView).SetText("Close file").SetIcon(icons.Close).
				SetShortcut(key.Chord(code.ChordForFun(code.KeyFunBufClose).String()))
			giv.NewFuncButton(mm, ge.OpenConsoleTab).SetText("Open console").SetIcon(icons.Terminal)
		})

		gi.NewButton(m).SetText("Command").SetMenu(func(mm *gi.Scene) {
			giv.NewFuncButton(mm, ge.DebugAttach).SetText("Debug attach").SetIcon(icons.Debug)
			giv.NewFuncButton(mm, ge.VCSLog).SetText("VCS Log").SetIcon(icons.List)
			giv.NewFuncButton(mm, ge.VCSUpdateAll).SetText("VCS update all").SetIcon(icons.Update)
			gi.NewSeparator(m)
			giv.NewFuncButton(mm, ge.CountWords).SetText("Count words all").SetIcon(icons.Counter5)
			giv.NewFuncButton(mm, ge.CountWordsRegion).SetText("Count words region").SetIcon(icons.Counter3)
			gi.NewSeparator(m)
			giv.NewFuncButton(mm, ge.HelpWiki).SetText("Help").SetIcon(icons.Help)
		})

		gi.NewSeparator(m)
	})

}

// ResourceFiles adds the files
func (ge *CodeView) ResourceFiles() uri.URIs {
	if ge.Files == nil {
		return nil
	}
	var ul uri.URIs
	ge.Files.WidgetWalkPre(func(wi gi.Widget, wb *gi.WidgetBase) bool {
		fn := filetree.AsNode(wi)
		if fn == nil || fn.IsIrregular() {
			return ki.Continue
		}
		rpath := fn.MyRelPath()
		nmpath := fn.Nm + ":" + rpath
		switch {
		case fn.IsDir():
			ur := uri.URI{Label: nmpath, Icon: icons.Folder}
			ur.SetURL("dir", "", rpath)
			ur.Func = func() {
				if !fn.HasChildren() {
					fn.OpenEmptyDir()
				}
				fn.Open()
				fn.ScrollToMe()
			}
			ul = append(ul, ur)
		case fn.IsExec():
			ur := uri.URI{Label: nmpath, Icon: icons.FileExe}
			ur.SetURL("exe", "", rpath)
			ur.Func = func() {
				ge.FileNodeRunExe(fn)
			}
			ul = append(ul, ur)
		default:
			ur := uri.URI{Label: nmpath, Icon: fn.Info.Ic}
			ur.SetURL("file", "", rpath)
			ur.Func = func() {
				ge.NextViewFileNode(fn)
			}
			ul = append(ul, ur)
		}
		return ki.Continue
	})
	return ul
}

// ResourceCommands adds the commands
func (ge *CodeView) ResourceCommands() uri.URIs {
	lang := ge.Prefs.MainLang
	vcnm := ge.VersCtrl()
	fn := ge.ActiveFileNode()
	if fn != nil {
		lang = fn.Info.Known
		if repo, _ := fn.Repo(); repo != nil {
			vcnm = filetree.VersCtrlName(repo.Vcs())
		}
	}
	var ul uri.URIs
	cmds := code.AvailCmds.FilterCmdNames(lang, vcnm)
	for _, cc := range cmds {
		cc := cc
		n := len(cc)
		if n < 2 {
			continue
		}
		cmdCat := cc[0]
		for ii := 1; ii < n; ii++ {
			ii := ii
			it := cc[ii]
			cmdNm := code.CommandName(cmdCat, it)
			ur := uri.URI{Label: cmdNm, Icon: icons.Icon(strings.ToLower(cmdCat))}
			ur.SetURL("cmd", "", cmdNm)
			ur.Func = func() {
				cmd := code.CmdName(cmdNm)
				ge.CmdHist().Add(cmd)          // only save commands executed via chooser
				ge.SaveAllCheck(true, func() { // true = cancel option
					ge.ExecCmdNameFileNode(fn, cmd, true, true) // sel, clear
				})
			}
			ul = append(ul, ur)
		}
	}
	return ul
}

// ResourceSymbols adds the symbols
func (ge *CodeView) ResourceSymbols() uri.URIs {
	tv := ge.ActiveTextEditor()
	if tv == nil || tv.Buf == nil || !tv.Buf.Hi.UsingPi() {
		return nil
	}
	pfs := tv.Buf.PiState.Done()
	if len(pfs.ParseState.Scopes) == 0 {
		return nil
	}
	pkg := pfs.ParseState.Scopes[0] // first scope of parse state is the full set of package symbols
	syms := &code.SymNode{}
	syms.InitName(syms, "syms")
	syms.OpenSyms(pkg, "", "")
	var ul uri.URIs
	syms.WalkPre(func(k ki.Ki) bool {
		sn := k.(*code.SymNode)
		ur := uri.URI{Label: sn.Symbol.Label(), Icon: sn.GetIcon()}
		ur.SetURL("sym", "", sn.PathFrom(syms))
		ur.Func = func() {
			code.SelectSymbol(ge, sn.Symbol)
		}
		ul = append(ul, ur)
		return ki.Continue
	})
	return ul
}
