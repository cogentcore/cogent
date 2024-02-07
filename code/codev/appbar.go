// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codev

import (
	"strings"

	"cogentcore.org/cogent/code/code"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
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
	gi.StdAppBarBack(tb)
	ac := gi.StdAppBarChooser(tb)
	ge.AddChooserFiles(ac)
	ge.AddChooserSymbols(ac)

	gi.StdOverflowMenu(tb)
	gi.CurrentWindowAppBar(tb)
	// apps should add their own app-general functions here
}

func (ge *CodeView) ConfigToolbar(tb *gi.Toolbar) { //gti:add
	giv.NewFuncButton(tb, ge.UpdateFiles).SetText("").SetIcon(icons.Refresh).SetShortcut("Command+U")
	sm := gi.NewSwitch(tb, "go-mod").SetText("Go mod").SetTooltip("Toggles the use of go modules -- saved with project -- if off, uses old school GOPATH mode")
	sm.Style(func(s *styles.Style) {
		sm.SetChecked(ge.Settings.GoMod)
	})
	sm.OnChange(func(e events.Event) {
		ge.Settings.GoMod = sm.StateIs(states.Checked)
		code.SetGoMod(ge.Settings.GoMod)
	})

	gi.NewSeparator(tb)
	gi.NewButton(tb).SetText("Open recent").SetMenu(func(m *gi.Scene) {
		for _, rp := range code.RecentPaths {
			rp := rp
			gi.NewButton(m).SetText(rp).OnClick(func(e events.Event) {
				ge.OpenRecent(gi.Filename(rp))
			})
		}
		gi.NewSeparator(m)
		gi.NewButton(m).SetText("Recent recent paths").OnClick(func(e events.Event) {
			code.RecentPaths = nil
		})
		gi.NewButton(m).SetText("Edit recent paths").OnClick(func(e events.Event) {
			ge.EditRecentPaths()
		})
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
				ic := icons.Icon(strings.ToLower(cat))
				gi.NewButton(m).SetText(cat).SetIcon(ic).SetMenu(func(mm *gi.Scene) {
					nc := len(cc)
					for i := 1; i < nc; i++ {
						cm := cc[i]
						gi.NewButton(mm).SetText(cm).SetIcon(ic).OnClick(func(e events.Event) {
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
					if sn == ge.Settings.SplitName {
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
					if sn == ge.Settings.SplitName {
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

			giv.NewFuncButton(mm, ge.EditProjSettings).SetText("Project Settings").
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

// AddChooserFiles adds the files to the app chooser.
func (ge *CodeView) AddChooserFiles(ac *gi.Chooser) {
	ac.AddItemsFunc(func() {
		if ge.Files == nil {
			return
		}
		ge.Files.WidgetWalkPre(func(wi gi.Widget, wb *gi.WidgetBase) bool {
			fn := filetree.AsNode(wi)
			if fn == nil || fn.IsIrregular() {
				return ki.Continue
			}
			rpath := fn.MyRelPath()
			nmpath := fn.Nm + ":" + rpath
			switch {
			case fn.IsDir():
				ac.Items = append(ac.Items, gi.ChooserItem{
					Label: nmpath,
					Icon:  icons.Folder,
					Func: func() {
						if !fn.HasChildren() {
							fn.OpenEmptyDir()
						}
						fn.Open()
						fn.ScrollToMe()
					},
				})
			case fn.IsExec():
				ac.Items = append(ac.Items, gi.ChooserItem{
					Label: nmpath,
					Icon:  icons.FileExe,
					Func: func() {
						ge.FileNodeRunExe(fn)
					},
				})
			default:
				ac.Items = append(ac.Items, gi.ChooserItem{
					Label: nmpath,
					Icon:  fn.Info.Ic,
					Func: func() {
						ge.NextViewFileNode(fn)
					},
				})
			}
			return ki.Continue
		})
	})
}

// AddChooserSymbols adds the symbols to the app chooser.
func (ge *CodeView) AddChooserSymbols(ac *gi.Chooser) {
	ac.AddItemsFunc(func() {
		tv := ge.ActiveTextEditor()
		if tv == nil || tv.Buf == nil || !tv.Buf.Hi.UsingPi() {
			return
		}
		pfs := tv.Buf.PiState.Done()
		if len(pfs.ParseState.Scopes) == 0 {
			return
		}
		pkg := pfs.ParseState.Scopes[0] // first scope of parse state is the full set of package symbols
		syms := &code.SymNode{}
		syms.InitName(syms, "syms")
		syms.OpenSyms(pkg, "", "")
		syms.WalkPre(func(k ki.Ki) bool {
			sn := k.(*code.SymNode)
			ac.Items = append(ac.Items, gi.ChooserItem{
				Label: sn.Symbol.Label(),
				Icon:  sn.GetIcon(),
				Func: func() {
					code.SelectSymbol(ge, sn.Symbol)
				},
			})
			return ki.Continue
		})
	})
}
