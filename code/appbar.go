// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/views"
)

func (cv *CodeView) AppBarConfig(parent core.Widget) {
	tb := core.RecycleToolbar(parent)
	tb.Maker(func(p *core.Plan) {
		core.StandardAppBarBack(p)
		core.AddAt(p, "app-chooser", func(w *core.Chooser) {
			core.ConfigAppChooser(w)
			cv.AddChooserFiles(w)
			cv.AddChooserSymbols(w)
			w.OnFirst(events.KeyChord, func(e events.Event) {
				kf := keymap.Of(e.KeyChord())
				if kf == keymap.Abort {
					w.ClearError()
					cv.FocusActiveTextEditor()
				}
			})
		})
		cv.MakeToolbar(p)
	})
	core.StandardOverflowMenu(tb)
	// core.CurrentWindowAppBar(tb)
	// apps should add their own app-general functions here
}

func (cv *CodeView) MakeToolbar(p *core.Plan) { //types:add
	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(cv.UpdateFiles).SetText("").SetIcon(icons.Refresh).SetShortcut("Command+U")
	})
	core.Add(p, func(w *core.Switch) {
		w.SetText("Go mod").SetTooltip("Toggles the use of go modules -- saved with project -- if off, uses old school GOPATH mode")
		w.Style(func(s *styles.Style) {
			w.SetChecked(cv.Settings.GoMod) // todo: update
		})
		w.OnChange(func(e events.Event) {
			cv.Settings.GoMod = w.StateIs(states.Checked)
			SetGoMod(cv.Settings.GoMod)
		})
	})

	core.Add[*core.Separator](p)
	core.Add(p, func(w *core.Button) {
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

	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(cv.OpenPath).
			SetText("Open").SetIcon(icons.Open).SetKey(keymap.Open)
		cv.ConfigActiveFilename(w)
	})

	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(cv.SaveActiveView).SetText("Save").
			SetIcon(icons.Save).SetKey(keymap.Save)
	})

	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(cv.SaveAll).SetIcon(icons.Save)
	})

	core.Add[*core.Separator](p)

	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(cv.CursorToHistPrev).SetText("").SetKey(keymap.HistPrev).
			SetIcon(icons.KeyboardArrowLeft).SetShowReturn(false)
	})
	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(cv.CursorToHistNext).SetText("").SetKey(keymap.HistNext).
			SetIcon(icons.KeyboardArrowRight).SetShowReturn(false)
	})

	core.Add[*core.Separator](p)

	// todo: this does not work to apply project defaults!
	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(cv.Find).SetIcon(icons.FindReplace)
		cv.ConfigFindButton(w)
	})

	core.Add[*core.Separator](p)

	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(cv.Symbols).SetIcon(icons.List)
	})

	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(cv.Spell).SetIcon(icons.Spellcheck)
	})

	core.Add[*core.Separator](p)

	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(cv.RunBuild).SetText("Build").SetIcon(icons.Build).
			SetShortcut(key.Chord(ChordForFunction(KeyBuildProject).String()))
	})

	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(cv.Run).SetIcon(icons.PlayArrow).
			SetShortcut(key.Chord(ChordForFunction(KeyRunProject).String()))
	})

	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(cv.Debug).SetIcon(icons.Debug)
	})

	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(cv.DebugTest).SetIcon(icons.Debug)
	})

	core.Add[*core.Separator](p)

	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(cv.Commit).SetIcon(icons.Star)
	})

	core.Add(p, func(w *core.Button) {
		w.SetText("Command").
			SetShortcut(key.Chord(ChordForFunction(KeyExecCmd).String())).
			SetMenu(func(m *core.Scene) {
				ec := ExecCmds(cv)
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
								cv.ExecCmdNameActive(CommandName(cat, cm))
							})
						}
					})
				}
			})
	})

	core.Add[*core.Separator](p)

	core.Add(p, func(w *core.Button) {
		w.SetText("Splits").SetMenu(func(m *core.Scene) {
			core.NewButton(m).SetText("Set View").
				SetMenu(func(mm *core.Scene) {
					for _, sp := range AvailableSplitNames {
						sn := SplitName(sp)
						mb := core.NewButton(mm).SetText(sp).OnClick(func(e events.Event) {
							cv.SplitsSetView(sn)
						})
						if sn == cv.Settings.SplitName {
							mb.SetSelected(true)
						}
					}
				})
			views.NewFuncButton(m, cv.SplitsSaveAs).SetText("Save As")
			core.NewButton(m).SetText("Save").
				SetMenu(func(mm *core.Scene) {
					for _, sp := range AvailableSplitNames {
						sn := SplitName(sp)
						mb := core.NewButton(mm).SetText(sp).OnClick(func(e events.Event) {
							cv.SplitsSave(sn)
						})
						if sn == cv.Settings.SplitName {
							mb.SetSelected(true)
						}
					}
				})
			views.NewFuncButton(m, cv.SplitsEdit).SetText("Edit")
		})
	})

	/*
		todo:
		tb.AddOverflowMenu(func(m *core.Scene) {
			core.NewButton(m).SetText("File").SetMenu(func(mm *core.Scene) {
				views.NewFuncButton(mm, ge.NewProject).SetIcon(icons.NewWindow).SetKey(keymap.New)
				views.NewFuncButton(mm, ge.NewFile).SetText("New File").SetIcon(icons.NewWindow)

				core.NewSeparator(mm)

				views.NewFuncButton(mm, ge.OpenProject).SetIcon(icons.Open)

				core.NewSeparator(mm)

				views.NewFuncButton(mm, ge.EditProjectSettings).SetText("Project Settings").SetIcon(icons.Edit)

				views.NewFuncButton(mm, ge.SaveProject).SetIcon(icons.Save)

				ge.ConfigActiveFilename(views.NewFuncButton(mm, ge.SaveProjectAs).SetIcon(icons.SaveAs))

				core.NewSeparator(mm)

				views.NewFuncButton(mm, ge.RevertActiveView).SetText("Revert File").
					SetIcon(icons.Undo)

				ge.ConfigActiveFilename(views.NewFuncButton(mm, ge.SaveActiveViewAs).
					SetText("Save File As").SetIcon(icons.SaveAs).SetKey(keymap.SaveAs))

			})

			core.NewButton(m).SetText("Edit").SetMenu(func(mm *core.Scene) {
				core.NewButton(mm).SetText("Paste history").SetIcon(icons.Paste).
					SetKey(keymap.PasteHist)

				views.NewFuncButton(mm, ge.RegisterPaste).SetIcon(icons.Paste).
					SetShortcut(key.Chord(ChordForFunction(KeyRegCopy).String()))

				views.NewFuncButton(mm, ge.RegisterCopy).SetIcon(icons.Copy).
					SetShortcut(key.Chord(ChordForFunction(KeyRegPaste).String()))

				core.NewSeparator(mm)

				views.NewFuncButton(mm, ge.CopyRect).SetIcon(icons.Copy).
					SetShortcut(key.Chord(ChordForFunction(KeyRectCopy).String()))

				views.NewFuncButton(mm, ge.CutRect).SetIcon(icons.Cut).
					SetShortcut(key.Chord(ChordForFunction(KeyRectCut).String()))

				views.NewFuncButton(mm, ge.PasteRect).SetIcon(icons.Paste).
					SetShortcut(key.Chord(ChordForFunction(KeyRectPaste).String()))

				core.NewSeparator(mm)

				core.NewButton(mm).SetText("Undo").SetIcon(icons.Undo).SetKey(keymap.Undo)

				core.NewButton(mm).SetText("Redo").SetIcon(icons.Redo).SetKey(keymap.Redo)

				core.NewSeparator(mm)

				views.NewFuncButton(mm, ge.ReplaceInActive).SetText("Replace in File").
					SetIcon(icons.FindReplace)

				core.NewButton(mm).SetText("Show completions").SetIcon(icons.CheckCircle).SetKey(keymap.Complete)

				core.NewButton(mm).SetText("Lookup symbol").SetIcon(icons.Search).SetKey(keymap.Lookup)

				core.NewButton(mm).SetText("Jump to line").SetIcon(icons.GoToLine).SetKey(keymap.Jump)

				core.NewSeparator(mm)

				views.NewFuncButton(mm, ge.CommentOut).SetText("Comment region").
					SetIcon(icons.Comment).SetShortcut(key.Chord(ChordForFunction(KeyCommentOut).String()))

				views.NewFuncButton(mm, ge.Indent).SetIcon(icons.FormatIndentIncrease).
					SetShortcut(key.Chord(ChordForFunction(KeyIndent).String()))

				views.NewFuncButton(mm, ge.ReCase).SetIcon(icons.MatchCase)

				views.NewFuncButton(mm, ge.JoinParaLines).SetIcon(icons.Join)

				views.NewFuncButton(mm, ge.TabsToSpaces).SetIcon(icons.TabMove)

				views.NewFuncButton(mm, ge.SpacesToTabs).SetIcon(icons.TabMove)
			})

			core.NewButton(m).SetText("View").SetMenu(func(mm *core.Scene) {
				views.NewFuncButton(mm, ge.FocusPrevPanel).SetText("Focus prev").SetIcon(icons.KeyboardArrowLeft).
					SetShortcut(key.Chord(ChordForFunction(KeyPrevPanel).String()))
				views.NewFuncButton(mm, ge.FocusNextPanel).SetText("Focus next").SetIcon(icons.KeyboardArrowRight).
					SetShortcut(key.Chord(ChordForFunction(KeyNextPanel).String()))
				views.NewFuncButton(mm, ge.CloneActiveView).SetText("Clone active").SetIcon(icons.Copy).
					SetShortcut(key.Chord(ChordForFunction(KeyBufClone).String()))
				core.NewSeparator(m)
				views.NewFuncButton(mm, ge.CloseActiveView).SetText("Close file").SetIcon(icons.Close).
					SetShortcut(key.Chord(ChordForFunction(KeyBufClose).String()))
				views.NewFuncButton(mm, ge.OpenConsoleTab).SetText("Open console").SetIcon(icons.Terminal)
			})

			core.NewButton(m).SetText("Command").SetMenu(func(mm *core.Scene) {
				views.NewFuncButton(mm, ge.DebugAttach).SetText("Debug attach").SetIcon(icons.Debug)
				views.NewFuncButton(mm, ge.VCSLog).SetText("VCS Log").SetIcon(icons.List)
				views.NewFuncButton(mm, ge.VCSUpdateAll).SetText("VCS update all").SetIcon(icons.Update)
				core.NewSeparator(m)
				views.NewFuncButton(mm, ge.CountWords).SetText("Count words all").SetIcon(icons.Counter5)
				views.NewFuncButton(mm, ge.CountWordsRegion).SetText("Count words region").SetIcon(icons.Counter3)
				core.NewSeparator(m)
				views.NewFuncButton(mm, ge.HelpWiki).SetText("Help").SetIcon(icons.Help)
			})

			core.NewSeparator(m)
		})
	*/
}

// AddChooserFiles adds the files to the app chooser.
func (cv *CodeView) AddChooserFiles(ac *core.Chooser) {
	ac.AddItemsFunc(func() {
		if cv.Files == nil {
			return
		}
		cv.Files.WidgetWalkDown(func(wi core.Widget, wb *core.WidgetBase) bool {
			fn := filetree.AsNode(wi)
			if fn == nil || fn.IsIrregular() {
				return tree.Continue
			}
			rpath := fn.MyRelPath()
			nmpath := fn.Nm + ":" + rpath
			switch {
			case fn.IsDir():
				ac.Items = append(ac.Items, core.ChooserItem{
					Text: nmpath,
					Icon: icons.Folder,
					Func: func() {
						if !fn.HasChildren() {
							fn.OpenEmptyDir()
						}
						fn.Open()
						fn.ScrollToMe()
						ac.CallItemsFuncs() // refresh avail files
					},
				})
			case fn.IsExec():
				ac.Items = append(ac.Items, core.ChooserItem{
					Text: nmpath,
					Icon: icons.FileExe,
					Func: func() {
						cv.FileNodeRunExe(fn)
					},
				})
			default:
				ac.Items = append(ac.Items, core.ChooserItem{
					Text: nmpath,
					Icon: fn.Info.Ic,
					Func: func() {
						cv.NextViewFileNode(fn)
						ac.CallItemsFuncs() // refresh avail files
					},
				})
			}
			return tree.Continue
		})
	})
}

// AddChooserSymbols adds the symbols to the app chooser.
func (cv *CodeView) AddChooserSymbols(ac *core.Chooser) {
	ac.AddItemsFunc(func() {
		tv := cv.ActiveTextEditor()
		if tv == nil || tv.Buffer == nil || !tv.Buffer.Hi.UsingParse() {
			return
		}
		pfs := tv.Buffer.ParseState.Done()
		if len(pfs.ParseState.Scopes) == 0 {
			return
		}
		pkg := pfs.ParseState.Scopes[0] // first scope of parse state is the full set of package symbols
		syms := NewSymNode()
		syms.SetName("syms")
		syms.OpenSyms(pkg, "", "")
		syms.WalkDown(func(k tree.Node) bool {
			sn := k.(*SymNode)
			ac.Items = append(ac.Items, core.ChooserItem{
				Text: sn.Symbol.Label(),
				Icon: sn.GetIcon(),
				Func: func() {
					SelectSymbol(cv, sn.Symbol)
				},
			})
			return tree.Continue
		})
	})
}
