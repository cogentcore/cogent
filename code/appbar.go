// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

func (cv *Code) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.UpdateFiles).SetText("").SetIcon(icons.Refresh).SetShortcut("Command+U")
	})
	tree.Add(p, func(w *core.Switch) {
		w.SetText("Go mod").SetTooltip("Toggles the use of go modules -- saved with project -- if off, uses old school GOPATH mode")
		w.Styler(func(s *styles.Style) {
			w.SetChecked(cv.Settings.GoMod) // todo: update
		})
		w.OnChange(func(e events.Event) {
			cv.Settings.GoMod = w.IsChecked()
			SetGoMod(cv.Settings.GoMod)
		})
	})

	tree.Add(p, func(w *core.Separator) {})

	tree.Add(p, func(w *core.Button) {
		w.SetText("Open recent").SetMenu(func(m *core.Scene) {
			for _, rp := range RecentPaths {
				core.NewButton(m).SetText(rp).OnClick(func(e events.Event) {
					cv.OpenRecent(core.Filename(rp))
				})
			}
			core.NewSeparator(m)
			core.NewButton(m).SetText("Recent recent paths").OnClick(func(e events.Event) {
				RecentPaths = nil
			})
			core.NewButton(m).SetText("Edit recent paths").OnClick(func(e events.Event) {
				cv.EditRecentPaths()
			})
		})
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.OpenPath).
			SetText("Open").SetIcon(icons.Open).SetKey(keymap.Open)
		cv.ConfigActiveFilename(w)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.SaveActiveView).SetText("Save").
			SetIcon(icons.Save).SetKey(keymap.Save)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.SaveAll).SetIcon(icons.Save)
	})

	tree.Add(p, func(w *core.Separator) {})

	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.CursorToHistPrev).SetText("").SetKey(keymap.HistPrev).
			SetIcon(icons.KeyboardArrowLeft)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.CursorToHistNext).SetText("").SetKey(keymap.HistNext).
			SetIcon(icons.KeyboardArrowRight)
	})

	tree.Add(p, func(w *core.Separator) {})

	// todo: this does not work to apply project defaults!
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.Find).SetIcon(icons.FindReplace)
		cv.ConfigFindButton(w)
	})

	tree.Add(p, func(w *core.Separator) {})

	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.Symbols).SetIcon(icons.List)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.Spell).SetIcon(icons.Spellcheck)
	})

	tree.Add(p, func(w *core.Separator) {})

	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.RunBuild).SetText("Build").SetIcon(icons.Build).
			SetShortcut(KeyBuildProject.Chord())
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.Run).SetIcon(icons.PlayArrow).
			SetShortcut(KeyRunProject.Chord())
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.Debug).SetIcon(icons.Debug)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.DebugTest).SetIcon(icons.Debug)
		w.Args[0].SetValue(cv.Settings.Debug.TestName)
	})

	tree.Add(p, func(w *core.Separator) {})

	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.Commit).SetIcon(icons.Star)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.VCSLog).SetText("VCS log").SetIcon(icons.List)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(cv.EditProjectSettings).SetText("Settings").SetIcon(icons.Edit)
	})

	tree.Add(p, func(w *core.Button) {
		w.SetText("Command").SetShortcut(KeyExecCmd.Chord())
		w.SetMenu(func(m *core.Scene) {
			ec := ExecCmds(cv)
			for _, cc := range ec {
				cc := cc
				cat := cc[0]
				icon := CommandIcons[cat]
				core.NewButton(m).SetText(cat).SetIcon(icon).SetMenu(func(mm *core.Scene) {
					nc := len(cc)
					for i := 1; i < nc; i++ {
						cm := cc[i]
						core.NewButton(mm).SetText(cm).SetIcon(icon).OnClick(func(e events.Event) {
							e.SetHandled()
							cv.ExecCmdNameActive(CommandName(cat, cm))
						})
					}
				})
			}
		})
	})

	tree.Add(p, func(w *core.Separator) {})

	tree.Add(p, func(w *core.Button) {
		w.SetText("Splits").SetMenu(func(m *core.Scene) {
			core.NewButton(m).SetText("Set view").
				SetMenu(func(mm *core.Scene) {
					for _, sp := range AvailableSplitNames {
						sn := SplitName(sp)
						mb := core.NewButton(mm).SetText(sp)
						mb.OnClick(func(e events.Event) {
							cv.SplitsSetView(sn)
						})
						if sn == cv.Settings.SplitName {
							mb.SetSelected(true)
						}
					}
				})
			core.NewFuncButton(m).SetFunc(cv.SplitsSaveAs).SetText("Save as").SetIcon(icons.SaveAs)
			core.NewButton(m).SetText("Save").SetIcon(icons.Save).SetMenu(func(mm *core.Scene) {
				for _, sp := range AvailableSplitNames {
					sn := SplitName(sp)
					mb := core.NewButton(mm).SetText(sp)
					mb.OnClick(func(e events.Event) {
						cv.SplitsSave(sn)
					})
					if sn == cv.Settings.SplitName {
						mb.SetSelected(true)
					}
				}
			})
			core.NewFuncButton(m).SetFunc(cv.SplitsEdit).SetText("Edit").SetIcon(icons.Edit)
		})
	})
}

func (cv *Code) OverflowMenu(m *core.Scene) {
	core.NewButton(m).SetText("File").SetMenu(func(m *core.Scene) {
		core.NewFuncButton(m).SetFunc(cv.NewProject).SetIcon(icons.NewWindow).SetKey(keymap.New)
		core.NewFuncButton(m).SetFunc(cv.NewFile).SetText("New file").SetIcon(icons.NewWindow)

		core.NewSeparator(m)

		core.NewFuncButton(m).SetFunc(cv.OpenProject).SetIcon(icons.Open)

		core.NewSeparator(m)

		core.NewFuncButton(m).SetFunc(cv.SaveProject).SetIcon(icons.Save)

		cv.ConfigActiveFilename(core.NewFuncButton(m).SetFunc(cv.SaveProjectAs)).SetIcon(icons.SaveAs)

		core.NewSeparator(m)

		core.NewFuncButton(m).SetFunc(cv.RevertActiveView).SetText("Revert file").
			SetIcon(icons.Undo)

		cv.ConfigActiveFilename(core.NewFuncButton(m).SetFunc(cv.SaveActiveViewAs)).
			SetText("Save File As").SetIcon(icons.SaveAs).SetKey(keymap.SaveAs)

	})

	core.NewButton(m).SetText("Edit").SetMenu(func(m *core.Scene) {
		core.NewButton(m).SetText("Paste history").SetIcon(icons.Paste).
			SetKey(keymap.PasteHist)

		core.NewFuncButton(m).SetFunc(cv.RegisterPaste).SetIcon(icons.Paste).
			SetShortcut(KeyRegPaste.Chord())
		core.NewFuncButton(m).SetFunc(cv.RegisterCopy).SetIcon(icons.Copy).
			SetShortcut(KeyRegCopy.Chord())

		core.NewSeparator(m)

		core.NewFuncButton(m).SetFunc(cv.CopyRect).SetIcon(icons.Copy).
			SetShortcut(KeyRectCopy.Chord())
		core.NewFuncButton(m).SetFunc(cv.CutRect).SetIcon(icons.Cut).
			SetShortcut(KeyRectCut.Chord())
		core.NewFuncButton(m).SetFunc(cv.PasteRect).SetIcon(icons.Paste).
			SetShortcut(KeyRectPaste.Chord())

		core.NewSeparator(m)

		core.NewButton(m).SetText("Undo").SetIcon(icons.Undo).SetKey(keymap.Undo)
		core.NewButton(m).SetText("Redo").SetIcon(icons.Redo).SetKey(keymap.Redo)

		core.NewSeparator(m)

		core.NewFuncButton(m).SetFunc(cv.ReplaceInActive).SetText("Replace in file").
			SetIcon(icons.FindReplace)

		core.NewButton(m).SetText("Show completions").SetIcon(icons.CheckCircle).SetKey(keymap.Complete)
		core.NewButton(m).SetText("Lookup symbol").SetIcon(icons.Search).SetKey(keymap.Lookup)
		core.NewButton(m).SetText("Jump to line").SetIcon(icons.GoToLine).SetKey(keymap.Jump)

		core.NewSeparator(m)

		core.NewFuncButton(m).SetFunc(cv.CommentOut).SetText("Comment region").
			SetIcon(icons.Comment).SetShortcut(KeyCommentOut.Chord())
		core.NewFuncButton(m).SetFunc(cv.Indent).SetIcon(icons.FormatIndentIncrease).
			SetShortcut(KeyIndent.Chord())
		core.NewFuncButton(m).SetFunc(cv.ReCase).SetIcon(icons.MatchCase)
		core.NewFuncButton(m).SetFunc(cv.JoinParaLines).SetIcon(icons.Join)
		core.NewFuncButton(m).SetFunc(cv.TabsToSpaces).SetIcon(icons.TabMove)
		core.NewFuncButton(m).SetFunc(cv.SpacesToTabs).SetIcon(icons.TabMove)
	})

	core.NewButton(m).SetText("View").SetMenu(func(m *core.Scene) {
		core.NewFuncButton(m).SetFunc(cv.FocusPrevPanel).SetText("Focus prev").SetIcon(icons.KeyboardArrowLeft).
			SetShortcut(KeyPrevPanel.Chord())
		core.NewFuncButton(m).SetFunc(cv.FocusNextPanel).SetText("Focus next").SetIcon(icons.KeyboardArrowRight).
			SetShortcut(KeyNextPanel.Chord())
		core.NewFuncButton(m).SetFunc(cv.CloneActiveView).SetText("Clone active").SetIcon(icons.Copy).
			SetShortcut(KeyBufClone.Chord())

		core.NewSeparator(m)

		core.NewFuncButton(m).SetFunc(cv.CloseActiveView).SetText("Close file").SetIcon(icons.Close).
			SetShortcut(KeyBufClose.Chord())
		core.NewFuncButton(m).SetFunc(cv.OpenConsoleTab).SetText("Open console").SetIcon(icons.Terminal)
	})

	core.NewButton(m).SetText("Command").SetMenu(func(m *core.Scene) {
		core.NewFuncButton(m).SetFunc(cv.DebugAttach).SetText("Debug attach").SetIcon(icons.Debug)
		core.NewFuncButton(m).SetFunc(cv.VCSUpdateAll).SetText("VCS update all").SetIcon(icons.Update)

		core.NewSeparator(m)

		core.NewFuncButton(m).SetFunc(cv.CountWords).SetText("Count words all").SetIcon(icons.Counter5)
		core.NewFuncButton(m).SetFunc(cv.CountWordsRegion).SetText("Count words region").SetIcon(icons.Counter3)

		core.NewSeparator(m)

		core.NewFuncButton(m).SetFunc(cv.HelpWiki).SetText("Help").SetIcon(icons.Help)
	})
}

func (cv *Code) MenuSearch(items *[]core.ChooserItem) {
	cv.addSearchFiles(items)
	cv.addSearchSymbols(items)
}

func (cv *Code) addSearchFiles(items *[]core.ChooserItem) {
	if cv.Files == nil {
		return
	}
	cv.Files.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		fn := filetree.AsNode(cw)
		if fn == nil || fn.IsIrregular() {
			return tree.Continue
		}
		rpath := fn.RelativePath()
		nmpath := fn.Name + ":" + rpath
		switch {
		case fn.IsDir():
			*items = append(*items, core.ChooserItem{
				Text: nmpath,
				Icon: icons.Folder,
				Func: func() {
					fn.Open()
					fn.ScrollToThis()
				},
			})
		case fn.IsExec():
			*items = append(*items, core.ChooserItem{
				Text: nmpath,
				Icon: icons.FileExe,
				Func: func() {
					cv.FileNodeRunExe(fn)
				},
			})
		default:
			*items = append(*items, core.ChooserItem{
				Text: nmpath,
				Icon: fn.Info.Ic,
				Func: func() {
					cv.NextViewFileNode(fn)
				},
			})
		}
		return tree.Continue
	})
}

func (cv *Code) addSearchSymbols(items *[]core.ChooserItem) {
	tv := cv.ActiveTextEditor()
	if tv == nil || tv.Lines == nil || !tv.Lines.Highlighter.UsingParse() {
		return
	}
	_, ps := tv.Lines.ParseState()
	if ps == nil {
		return
	}
	pfs := ps.Done()
	if len(pfs.ParseState.Scopes) == 0 {
		return
	}
	pkg := pfs.ParseState.Scopes[0] // first scope of parse state is the full set of package symbols
	syms := NewSymNode()
	syms.SetName("syms")
	syms.OpenSyms(pkg, "", "")
	syms.WalkDown(func(k tree.Node) bool {
		sn := k.(*SymNode)
		*items = append(*items, core.ChooserItem{
			Text: sn.Symbol.Label(),
			Icon: sn.GetIcon(),
			Func: func() {
				SelectSymbol(cv, sn.Symbol)
			},
		})
		return tree.Continue
	})
}
