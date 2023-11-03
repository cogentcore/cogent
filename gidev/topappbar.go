// Copyright (c) 2023, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidev

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/gide/v2/gide"
	"goki.dev/girl/states"
	"goki.dev/goosi/events"
	"goki.dev/goosi/events/key"
	"goki.dev/icons"
)

// DefaultTopAppBar is the default top app bar for gide
func DefaultTopAppBar(tb *gi.TopAppBar) {
	gi.DefaultTopAppBarStd(tb)
	tb.AddOverflowMenu(func(m *gi.Scene) {
		gi.NewButton(m).SetText("Gide Prefs").SetIcon(icons.Settings).
			OnClick(func(e events.Event) {
				gide.PrefsView(&gide.Prefs)
			})
	})
}

func (ge *GideView) TopAppBar(tb *gi.TopAppBar) { //gti:add
	gi.DefaultTopAppBar(tb)

	giv.NewFuncButton(tb, ge.UpdateFiles).SetText("").SetIcon(icons.Refresh).SetShortcut("Command+U")
	sm := gi.NewSwitch(tb, "go-mod").SetText("Go Mod").SetTooltip("Toggles the use of go modules -- saved with project -- if off, uses old school GOPATH mode")
	sm.SetChecked(ge.Prefs.GoMod)
	sm.OnClick(func(e events.Event) {
		ge.Prefs.GoMod = sm.StateIs(states.Checked)
		gide.SetGoMod(ge.Prefs.GoMod)
	})

	gi.NewSeparator(tb)
	gi.NewButton(tb).SetText("Open Recent").SetMenu(func(m *gi.Scene) {
		for _, sp := range gide.SavedPaths {
			sp := sp
			gi.NewButton(m).SetText(sp).OnClick(func(e events.Event) {
				ge.OpenRecent(gi.FileName(sp))
			})
		}
	})

	op := giv.NewFuncButton(tb, ge.NextViewFile).SetText("Open").SetIcon(icons.Open).SetKey(keyfun.Open)
	op.Args[0].SetValue(ge.ActiveFilename)

	giv.NewFuncButton(tb, ge.SaveActiveView).SetText("Save").SetIcon(icons.Save).SetKey(keyfun.Save)

	giv.NewFuncButton(tb, ge.SaveAll).SetIcon(icons.Save)

	gi.NewSeparator(tb)

	giv.NewFuncButton(tb, ge.CursorToHistPrev).SetText("").SetKey(keyfun.HistPrev).
		SetIcon(icons.KeyboardArrowLeft).SetShowReturn(false)
	giv.NewFuncButton(tb, ge.CursorToHistNext).SetText("").SetKey(keyfun.HistNext).
		SetIcon(icons.KeyboardArrowRight).SetShowReturn(false)

	gi.NewSeparator(tb)

	giv.NewFuncButton(tb, ge.Find).SetIcon(icons.FindReplace)

	gi.NewSeparator(tb)

	giv.NewFuncButton(tb, ge.Symbols).SetIcon(icons.List)

	giv.NewFuncButton(tb, ge.Spell).SetIcon(icons.Spellcheck)

	gi.NewSeparator(tb)

	giv.NewFuncButton(tb, ge.Build).SetIcon(icons.Build).
		SetShortcut(key.Chord(gide.ChordForFun(gide.KeyFunBuildProj).String()))

	giv.NewFuncButton(tb, ge.Run).SetIcon(icons.PlayArrow).
		SetShortcut(key.Chord(gide.ChordForFun(gide.KeyFunRunProj).String()))

	giv.NewFuncButton(tb, ge.Debug).SetIcon(icons.Debug)

	giv.NewFuncButton(tb, ge.DebugTest).SetIcon(icons.Debug)

	gi.NewSeparator(tb)

	giv.NewFuncButton(tb, ge.Commit).SetIcon(icons.Star)

	gi.NewButton(tb).SetText("Command").
		SetShortcut(key.Chord(gide.ChordForFun(gide.KeyFunExecCmd).String())).
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
							ge.ExecCmdNameActive(gide.CommandName(cat, cm))
						})
					}
				})
			}
		})

	gi.NewSeparator(tb)

	gi.NewButton(tb).SetText("Splits").SetMenu(func(m *gi.Scene) {
		gi.NewButton(m).SetText("Set View").
			SetMenu(func(mm *gi.Scene) {
				for _, sp := range gide.AvailSplitNames {
					sn := gide.SplitName(sp)
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
				for _, sp := range gide.AvailSplitNames {
					sn := gide.SplitName(sp)
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

			giv.NewFuncButton(mm, ge.OpenPath).SetText("Open Path").
				SetIcon(icons.Open)

			gi.NewSeparator(mm)

			giv.NewFuncButton(mm, ge.EditProjPrefs).SetText("Project Prefs").
				SetIcon(icons.Edit)

			giv.NewFuncButton(mm, ge.SaveProj).SetText("Save Project").
				SetIcon(icons.Save)

			giv.NewFuncButton(mm, ge.SaveProjAs).SetText("Save Project As").
				SetIcon(icons.SaveAs)

			gi.NewSeparator(mm)

			giv.NewFuncButton(mm, ge.RevertActiveView).SetText("Revert File").
				SetIcon(icons.Undo)

			sa := giv.NewFuncButton(mm, ge.SaveActiveViewAs).SetText("Save File As").
				SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs)
			sa.Args[0].SetValue(ge.ActiveFilename)

		})

		gi.NewButton(m).SetText("Edit").SetMenu(func(mm *gi.Scene) {
			gi.NewButton(mm).SetText("Paste history").SetIcon(icons.Paste).
				SetKey(keyfun.PasteHist)

			giv.NewFuncButton(mm, ge.RegisterPaste).SetIcon(icons.Paste).
				SetShortcut(key.Chord(gide.ChordForFun(gide.KeyFunRegCopy).String()))

			giv.NewFuncButton(mm, ge.RegisterCopy).SetIcon(icons.Copy).
				SetShortcut(key.Chord(gide.ChordForFun(gide.KeyFunRegPaste).String()))

			gi.NewSeparator(mm)

			giv.NewFuncButton(mm, ge.CopyRect).SetIcon(icons.Copy).
				SetShortcut(key.Chord(gide.ChordForFun(gide.KeyFunRectCopy).String()))

			giv.NewFuncButton(mm, ge.CutRect).SetIcon(icons.Cut).
				SetShortcut(key.Chord(gide.ChordForFun(gide.KeyFunRectCut).String()))

			giv.NewFuncButton(mm, ge.PasteRect).SetIcon(icons.Paste).
				SetShortcut(key.Chord(gide.ChordForFun(gide.KeyFunRectPaste).String()))

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
				SetIcon(icons.Comment).SetShortcut(key.Chord(gide.ChordForFun(gide.KeyFunCommentOut).String()))

			giv.NewFuncButton(mm, ge.Indent).SetIcon(icons.FormatIndentIncrease).
				SetShortcut(key.Chord(gide.ChordForFun(gide.KeyFunIndent).String()))

			giv.NewFuncButton(mm, ge.ReCase).SetIcon(icons.MatchCase)

			giv.NewFuncButton(mm, ge.JoinParaLines).SetIcon(icons.Join)

			giv.NewFuncButton(mm, ge.TabsToSpaces).SetIcon(icons.TabMove)

			giv.NewFuncButton(mm, ge.SpacesToTabs).SetIcon(icons.TabMove)
		})

		gi.NewSeparator(m)
	})

}

/*
// GideViewInactiveEmptyFunc is an ActionUpdateFunc that inactivates action if project is empty
var GideViewInactiveEmptyFunc = giv.ActionUpdateFunc(func(gei any, act *gi.Button) {
	ge := gei.(ki.Ki).Embed(KiT_GideView).(*GideView)
	if !ge.IsConfiged() {
		return
	}
	act.SetInactiveState(ge.IsEmpty())
})

// GideViewInactiveTextViewFunc is an ActionUpdateFunc that inactivates action there is no active text view
var GideViewInactiveTextViewFunc = giv.ActionUpdateFunc(func(gei any, act *gi.Button) {
	ge := gei.(ki.Ki).Embed(KiT_GideView).(*GideView)
	if !ge.IsConfiged() {
		return
	}
	act.SetInactiveState(ge.ActiveTextView().Buf == nil)
})

// GideViewInactiveTextSelectionFunc is an ActionUpdateFunc that inactivates action there is no active text view
var GideViewInactiveTextSelectionFunc = giv.ActionUpdateFunc(func(gei any, act *gi.Button) {
	ge := gei.(ki.Ki).Embed(KiT_GideView).(*GideView)
	if !ge.IsConfiged() {
		return
	}
	if ge.ActiveTextView() != nil && ge.ActiveTextView().Buf != nil {
		act.SetActiveState(ge.ActiveTextView().HasSelection())
	} else {
		act.SetActiveState(false)
	}
})
*/

/*
var GideViewProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
	"#title": ki.Props{
		"max-width":        -1,
		"horizontal-align": styles.AlignCenter,
		"vertical-align":   styles.AlignTop,
	},
	"MethodViewNoUpdateAfter": true, // no update after is default for everything
	"Toolbar": ki.PropSlice{
		{"ViewOpenNodeName", ki.Props{
			"icon":         "file-text",
			"label":        "Edit",
			"desc":         "select an open file to view in active text view",
			"submenu-func": giv.SubMenuFunc(GideViewOpenNodes),
			"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Button) key.Chord {
				return key.Chord(gide.ChordForFun(gide.KeyFunBufSelect).String())
			}),
			"Args": ki.PropSlice{
				{"Node Name", ki.Props{}},
			},
		}},
		{"sep-find", ki.BlankProp{}},
		{"Find", ki.Props{
			"label":    "Find...",
			"icon":     "search",
			"desc":     "Find / replace in all open folders in file browser",
			"shortcut": keyfun.Find,
			"Args": ki.PropSlice{
				{"Search For", ki.Props{
					"default-field": "Prefs.Find.Find",
					"history-field": "Prefs.Find.FindHist",
					"width":         80,
				}},
				{"Replace With", ki.Props{
					"desc":          "Optional replace string -- replace will be controlled interactively in Find panel, including replace all",
					"default-field": "Prefs.Find.Replace",
					"history-field": "Prefs.Find.ReplHist",
					"width":         80,
				}},
				{"Ignore Case", ki.Props{
					"default-field": "Prefs.Find.IgnoreCase",
				}},
				{"Regexp", ki.Props{
					"default-field": "Prefs.Find.Regexp",
				}},
				{"Location", ki.Props{
					"desc":          "location to find in",
					"default-field": "Prefs.Find.Loc",
				}},
				{"Languages", ki.Props{
					"desc":          "restrict find to files associated with these languages -- leave empty for all files",
					"default-field": "Prefs.Find.Langs",
				}},
			},
		}},
	},
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"CloseActiveView", ki.Props{
				"label":    "Close File",
				"updtfunc": GideViewInactiveEmptyFunc,
				"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Button) key.Chord {
					return key.Chord(gide.ChordForFun(gide.KeyFunBufClose).String())
				}),
			}},
		}},
		{"View", ki.PropSlice{
			{"Panels", ki.PropSlice{
				{"FocusNextPanel", ki.Props{
					"label": "Focus Next",
					"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Button) key.Chord {
						return key.Chord(gide.ChordForFun(gide.KeyFunNextPanel).String())
					}),
					"updtfunc": GideViewInactiveEmptyFunc,
				}},
				{"FocusPrevPanel", ki.Props{
					"label": "Focus Prev",
					"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Button) key.Chord {
						return key.Chord(gide.ChordForFun(gide.KeyFunPrevPanel).String())
					}),
					"updtfunc": GideViewInactiveEmptyFunc,
				}},
				{"CloneActiveView", ki.Props{
					"label": "Clone Active",
					"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Button) key.Chord {
						return key.Chord(gide.ChordForFun(gide.KeyFunBufClone).String())
					}),
					"updtfunc": GideViewInactiveEmptyFunc,
				}},
			}},
			{"OpenConsoleTab", ki.Props{
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
		}},
		{"Command", ki.PropSlice{
			{"DebugAttach", ki.Props{
				"desc": "attach to an already running process: enter the process PID",
				"Args": ki.PropSlice{
					{"Process PID", ki.Props{}},
				},
			}},
			{"ChooseRunExec", ki.Props{
				"desc": "choose the executable to run for this project using the Run button",
				"Args": ki.PropSlice{
					{"RunExec", ki.Props{
						"default-field": "Prefs.RunExec",
					}},
				},
			}},
			{"sep-run", ki.BlankProp{}},
			{"VCSLog", ki.Props{
				"label":    "VCS Log View",
				"desc":     "shows the VCS log of commits to repository associated with active file, optionally with a since date qualifier: If since is non-empty, it should be a date-like expression that the VCS will understand, such as 1/1/2020, yesterday, last year, etc (SVN only supports a max number of entries).",
				"updtfunc": GideViewInactiveEmptyFunc,
				"Args": ki.PropSlice{
					{"Since Date", ki.Props{}},
				},
			}},
			{"VCSUpdateAll", ki.Props{
				"label":    "VCS Update All",
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"sep-cmd", ki.BlankProp{}},
			{"ExecCmdNameActive", ki.Props{
				"label":           "Exec Cmd",
				"subsubmenu-func": giv.SubSubMenuFunc(ExecCmds),
				"updtfunc":        GideViewInactiveEmptyFunc,
				"Args": ki.PropSlice{
					{"Cmd Name", ki.Props{}},
				},
			}},
			{"DiffFiles", ki.Props{
				"updtfunc": GideViewInactiveEmptyFunc,
				"Args": ki.PropSlice{
					{"File Name 1", ki.Props{}},
					{"File Name 2", ki.Props{}},
				},
			}},
			{"sep-cmd", ki.BlankProp{}},
			{"CountWords", ki.Props{
				"updtfunc":    GideViewInactiveEmptyFunc,
				"show-return": true,
			}},
			{"CountWordsRegion", ki.Props{
				"updtfunc":    GideViewInactiveEmptyFunc,
				"show-return": true,
			}},
		}},
		{"Help", ki.PropSlice{
			{"HelpWiki", ki.Props{}},
		}},
	},
}
*/
