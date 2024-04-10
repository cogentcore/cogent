// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codev

import (
	"strings"

	"cogentcore.org/cogent/code/code"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

func (ge *CodeView) AppBarConfig(parent core.Widget) {
	tb := core.RecycleToolbar(parent)
	core.StandardAppBarBack(tb)
	ac := core.StandardAppBarChooser(tb)
	ge.AddChooserFiles(ac)
	ge.AddChooserSymbols(ac)
	ac.OnFirst(events.KeyChord, func(e events.Event) {
		kf := keyfun.Of(e.KeyChord())
		if kf == keyfun.Abort {
			ge.FocusActiveTextEditor()
		}
	})

	core.StandardOverflowMenu(tb)
	core.CurrentWindowAppBar(tb)
	// apps should add their own app-general functions here
}

func (ge *CodeView) ConfigToolbar(tb *core.Toolbar) { //gti:add
	giv.NewFuncButton(tb, ge.UpdateFiles).SetText("").SetIcon(icons.Refresh).SetShortcut("Command+U")
	sm := core.NewSwitch(tb, "go-mod").SetText("Go mod").SetTooltip("Toggles the use of go modules -- saved with project -- if off, uses old school GOPATH mode")
	sm.Style(func(s *styles.Style) {
		sm.SetChecked(ge.Settings.GoMod)
	})
	sm.OnChange(func(e events.Event) {
		ge.Settings.GoMod = sm.StateIs(states.Checked)
		code.SetGoMod(ge.Settings.GoMod)
	})

	core.NewSeparator(tb)
	core.NewButton(tb).SetText("Open recent").SetMenu(func(m *core.Scene) {
		for _, rp := range code.RecentPaths {
			rp := rp
			core.NewButton(m).SetText(rp).OnClick(func(e events.Event) {
				ge.OpenRecent(core.Filename(rp))
			})
		}
		core.NewSeparator(m)
		core.NewButton(m).SetText("Recent recent paths").OnClick(func(e events.Event) {
			code.RecentPaths = nil
		})
		core.NewButton(m).SetText("Edit recent paths").OnClick(func(e events.Event) {
			ge.EditRecentPaths()
		})
	})

	ge.ConfigActiveFilename(giv.NewFuncButton(tb, ge.OpenPath).
		SetText("Open").SetIcon(icons.Open).SetKey(keyfun.Open))

	giv.NewFuncButton(tb, ge.SaveActiveView).SetText("Save").
		SetIcon(icons.Save).SetKey(keyfun.Save)

	giv.NewFuncButton(tb, ge.SaveAll).SetIcon(icons.Save)

	core.NewSeparator(tb)

	giv.NewFuncButton(tb, ge.CursorToHistPrev).SetText("").SetKey(keyfun.HistPrev).
		SetIcon(icons.KeyboardArrowLeft).SetShowReturn(false)
	giv.NewFuncButton(tb, ge.CursorToHistNext).SetText("").SetKey(keyfun.HistNext).
		SetIcon(icons.KeyboardArrowRight).SetShowReturn(false)

	core.NewSeparator(tb)

	ge.ConfigFindButton(giv.NewFuncButton(tb, ge.Find).SetIcon(icons.FindReplace))

	core.NewSeparator(tb)

	giv.NewFuncButton(tb, ge.Symbols).SetIcon(icons.List)

	giv.NewFuncButton(tb, ge.Spell).SetIcon(icons.Spellcheck)

	core.NewSeparator(tb)

	giv.NewFuncButton(tb, ge.Build).SetIcon(icons.Build).
		SetShortcut(key.Chord(code.ChordForFun(code.KeyFunBuildProj).String()))

	giv.NewFuncButton(tb, ge.Run).SetIcon(icons.PlayArrow).
		SetShortcut(key.Chord(code.ChordForFun(code.KeyFunRunProj).String()))

	giv.NewFuncButton(tb, ge.Debug).SetIcon(icons.Debug)

	giv.NewFuncButton(tb, ge.DebugTest).SetIcon(icons.Debug)

	core.NewSeparator(tb)

	giv.NewFuncButton(tb, ge.Commit).SetIcon(icons.Star)

	core.NewButton(tb).SetText("Command").
		SetShortcut(key.Chord(code.ChordForFun(code.KeyFunExecCmd).String())).
		SetMenu(func(m *core.Scene) {
			ec := ExecCmds(ge)
			for _, cc := range ec {
				cc := cc
				cat := cc[0]
				ic := icons.Icon(strings.ToLower(cat))
				core.NewButton(m).SetText(cat).SetIcon(ic).SetMenu(func(mm *core.Scene) {
					nc := len(cc)
					for i := 1; i < nc; i++ {
						cm := cc[i]
						core.NewButton(mm).SetText(cm).SetIcon(ic).OnClick(func(e events.Event) {
							e.SetHandled()
							ge.ExecCmdNameActive(code.CommandName(cat, cm))
						})
					}
				})
			}
		})

	core.NewSeparator(tb)

	core.NewButton(tb).SetText("Splits").SetMenu(func(m *core.Scene) {
		core.NewButton(m).SetText("Set View").
			SetMenu(func(mm *core.Scene) {
				for _, sp := range code.AvailableSplitNames {
					sn := code.SplitName(sp)
					mb := core.NewButton(mm).SetText(sp).OnClick(func(e events.Event) {
						ge.SplitsSetView(sn)
					})
					if sn == ge.Settings.SplitName {
						mb.SetSelected(true)
					}
				}
			})
		giv.NewFuncButton(m, ge.SplitsSaveAs).SetText("Save As")
		core.NewButton(m).SetText("Save").
			SetMenu(func(mm *core.Scene) {
				for _, sp := range code.AvailableSplitNames {
					sn := code.SplitName(sp)
					mb := core.NewButton(mm).SetText(sp).OnClick(func(e events.Event) {
						ge.SplitsSave(sn)
					})
					if sn == ge.Settings.SplitName {
						mb.SetSelected(true)
					}
				}
			})
		giv.NewFuncButton(m, ge.SplitsEdit).SetText("Edit")
	})

	tb.AddOverflowMenu(func(m *core.Scene) {
		core.NewButton(m).SetText("File").SetMenu(func(mm *core.Scene) {
			giv.NewFuncButton(mm, ge.NewProj).SetText("New Project").
				SetIcon(icons.NewWindow).SetKey(keyfun.New)

			giv.NewFuncButton(mm, ge.NewFile).SetText("New File").
				SetIcon(icons.NewWindow)

			core.NewSeparator(mm)

			giv.NewFuncButton(mm, ge.OpenProj).SetText("Open Project").
				SetIcon(icons.Open)

			core.NewSeparator(mm)

			giv.NewFuncButton(mm, ge.EditProjSettings).SetText("Project Settings").
				SetIcon(icons.Edit)

			giv.NewFuncButton(mm, ge.SaveProj).SetText("Save Project").
				SetIcon(icons.Save)

			ge.ConfigActiveFilename(giv.NewFuncButton(mm, ge.SaveProjAs).
				SetText("Save Project As").SetIcon(icons.SaveAs))

			core.NewSeparator(mm)

			giv.NewFuncButton(mm, ge.RevertActiveView).SetText("Revert File").
				SetIcon(icons.Undo)

			ge.ConfigActiveFilename(giv.NewFuncButton(mm, ge.SaveActiveViewAs).
				SetText("Save File As").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs))

		})

		core.NewButton(m).SetText("Edit").SetMenu(func(mm *core.Scene) {
			core.NewButton(mm).SetText("Paste history").SetIcon(icons.Paste).
				SetKey(keyfun.PasteHist)

			giv.NewFuncButton(mm, ge.RegisterPaste).SetIcon(icons.Paste).
				SetShortcut(key.Chord(code.ChordForFun(code.KeyFunRegCopy).String()))

			giv.NewFuncButton(mm, ge.RegisterCopy).SetIcon(icons.Copy).
				SetShortcut(key.Chord(code.ChordForFun(code.KeyFunRegPaste).String()))

			core.NewSeparator(mm)

			giv.NewFuncButton(mm, ge.CopyRect).SetIcon(icons.Copy).
				SetShortcut(key.Chord(code.ChordForFun(code.KeyFunRectCopy).String()))

			giv.NewFuncButton(mm, ge.CutRect).SetIcon(icons.Cut).
				SetShortcut(key.Chord(code.ChordForFun(code.KeyFunRectCut).String()))

			giv.NewFuncButton(mm, ge.PasteRect).SetIcon(icons.Paste).
				SetShortcut(key.Chord(code.ChordForFun(code.KeyFunRectPaste).String()))

			core.NewSeparator(mm)

			core.NewButton(mm).SetText("Undo").SetIcon(icons.Undo).SetKey(keyfun.Undo)

			core.NewButton(mm).SetText("Redo").SetIcon(icons.Redo).SetKey(keyfun.Redo)

			core.NewSeparator(mm)

			giv.NewFuncButton(mm, ge.ReplaceInActive).SetText("Replace in File").
				SetIcon(icons.FindReplace)

			core.NewButton(mm).SetText("Show completions").SetIcon(icons.CheckCircle).SetKey(keyfun.Complete)

			core.NewButton(mm).SetText("Lookup symbol").SetIcon(icons.Search).SetKey(keyfun.Lookup)

			core.NewButton(mm).SetText("Jump to line").SetIcon(icons.GoToLine).SetKey(keyfun.Jump)

			core.NewSeparator(mm)

			giv.NewFuncButton(mm, ge.CommentOut).SetText("Comment region").
				SetIcon(icons.Comment).SetShortcut(key.Chord(code.ChordForFun(code.KeyFunCommentOut).String()))

			giv.NewFuncButton(mm, ge.Indent).SetIcon(icons.FormatIndentIncrease).
				SetShortcut(key.Chord(code.ChordForFun(code.KeyFunIndent).String()))

			giv.NewFuncButton(mm, ge.ReCase).SetIcon(icons.MatchCase)

			giv.NewFuncButton(mm, ge.JoinParaLines).SetIcon(icons.Join)

			giv.NewFuncButton(mm, ge.TabsToSpaces).SetIcon(icons.TabMove)

			giv.NewFuncButton(mm, ge.SpacesToTabs).SetIcon(icons.TabMove)
		})

		core.NewButton(m).SetText("View").SetMenu(func(mm *core.Scene) {
			giv.NewFuncButton(mm, ge.FocusPrevPanel).SetText("Focus prev").SetIcon(icons.KeyboardArrowLeft).
				SetShortcut(key.Chord(code.ChordForFun(code.KeyFunPrevPanel).String()))
			giv.NewFuncButton(mm, ge.FocusNextPanel).SetText("Focus next").SetIcon(icons.KeyboardArrowRight).
				SetShortcut(key.Chord(code.ChordForFun(code.KeyFunNextPanel).String()))
			giv.NewFuncButton(mm, ge.CloneActiveView).SetText("Clone active").SetIcon(icons.Copy).
				SetShortcut(key.Chord(code.ChordForFun(code.KeyFunBufClone).String()))
			core.NewSeparator(m)
			giv.NewFuncButton(mm, ge.CloseActiveView).SetText("Close file").SetIcon(icons.Close).
				SetShortcut(key.Chord(code.ChordForFun(code.KeyFunBufClose).String()))
			giv.NewFuncButton(mm, ge.OpenConsoleTab).SetText("Open console").SetIcon(icons.Terminal)
		})

		core.NewButton(m).SetText("Command").SetMenu(func(mm *core.Scene) {
			giv.NewFuncButton(mm, ge.DebugAttach).SetText("Debug attach").SetIcon(icons.Debug)
			giv.NewFuncButton(mm, ge.VCSLog).SetText("VCS Log").SetIcon(icons.List)
			giv.NewFuncButton(mm, ge.VCSUpdateAll).SetText("VCS update all").SetIcon(icons.Update)
			core.NewSeparator(m)
			giv.NewFuncButton(mm, ge.CountWords).SetText("Count words all").SetIcon(icons.Counter5)
			giv.NewFuncButton(mm, ge.CountWordsRegion).SetText("Count words region").SetIcon(icons.Counter3)
			core.NewSeparator(m)
			giv.NewFuncButton(mm, ge.HelpWiki).SetText("Help").SetIcon(icons.Help)
		})

		core.NewSeparator(m)
	})

}

// AddChooserFiles adds the files to the app chooser.
func (ge *CodeView) AddChooserFiles(ac *core.Chooser) {
	ac.AddItemsFunc(func() {
		if ge.Files == nil {
			return
		}
		ge.Files.WidgetWalkPre(func(wi core.Widget, wb *core.WidgetBase) bool {
			fn := filetree.AsNode(wi)
			if fn == nil || fn.IsIrregular() {
				return tree.Continue
			}
			rpath := fn.MyRelPath()
			nmpath := fn.Nm + ":" + rpath
			switch {
			case fn.IsDir():
				ac.Items = append(ac.Items, core.ChooserItem{
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
				ac.Items = append(ac.Items, core.ChooserItem{
					Label: nmpath,
					Icon:  icons.FileExe,
					Func: func() {
						ge.FileNodeRunExe(fn)
					},
				})
			default:
				ac.Items = append(ac.Items, core.ChooserItem{
					Label: nmpath,
					Icon:  fn.Info.Ic,
					Func: func() {
						ge.NextViewFileNode(fn)
					},
				})
			}
			return tree.Continue
		})
	})
}

// AddChooserSymbols adds the symbols to the app chooser.
func (ge *CodeView) AddChooserSymbols(ac *core.Chooser) {
	ac.AddItemsFunc(func() {
		tv := ge.ActiveTextEditor()
		if tv == nil || tv.Buffer == nil || !tv.Buffer.Hi.UsingPi() {
			return
		}
		pfs := tv.Buffer.PiState.Done()
		if len(pfs.ParseState.Scopes) == 0 {
			return
		}
		pkg := pfs.ParseState.Scopes[0] // first scope of parse state is the full set of package symbols
		syms := &code.SymNode{}
		syms.InitName(syms, "syms")
		syms.OpenSyms(pkg, "", "")
		syms.WalkDown(func(k tree.Node) bool {
			sn := k.(*code.SymNode)
			ac.Items = append(ac.Items, core.ChooserItem{
				Label: sn.Symbol.Label(),
				Icon:  sn.GetIcon(),
				Func: func() {
					code.SelectSymbol(ge, sn.Symbol)
				},
			})
			return tree.Continue
		})
	})
}
