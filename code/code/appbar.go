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

func (ge *CodeView) AppBarConfig(parent core.Widget) {
	tb := core.RecycleToolbar(parent)
	core.StandardAppBarBack(tb)
	ac := core.StandardAppBarChooser(tb)
	ge.AddChooserFiles(ac)
	ge.AddChooserSymbols(ac)
	ac.OnFirst(events.KeyChord, func(e events.Event) {
		kf := keymap.Of(e.KeyChord())
		if kf == keymap.Abort {
			ac.ClearText()
			ge.FocusActiveTextEditor()
		}
	})

	core.StandardOverflowMenu(tb)
	core.CurrentWindowAppBar(tb)
	// apps should add their own app-general functions here
}

func (ge *CodeView) ConfigToolbar(tb *core.Toolbar) { //types:add
	views.NewFuncButton(tb, ge.UpdateFiles).SetText("").SetIcon(icons.Refresh).SetShortcut("Command+U")
	sm := core.NewSwitch(tb, "go-mod").SetText("Go mod").SetTooltip("Toggles the use of go modules -- saved with project -- if off, uses old school GOPATH mode")
	sm.Style(func(s *styles.Style) {
		sm.SetChecked(ge.Settings.GoMod)
	})
	sm.OnChange(func(e events.Event) {
		ge.Settings.GoMod = sm.StateIs(states.Checked)
		SetGoMod(ge.Settings.GoMod)
	})

	core.NewSeparator(tb)
	core.NewButton(tb).SetText("Open recent").SetMenu(func(m *core.Scene) {
		for _, rp := range RecentPaths {
			rp := rp
			core.NewButton(m).SetText(rp).OnClick(func(e events.Event) {
				ge.OpenRecent(core.Filename(rp))
			})
		}
		core.NewSeparator(m)
		core.NewButton(m).SetText("Recent recent paths").OnClick(func(e events.Event) {
			RecentPaths = nil
		})
		core.NewButton(m).SetText("Edit recent paths").OnClick(func(e events.Event) {
			ge.EditRecentPaths()
		})
	})

	ge.ConfigActiveFilename(views.NewFuncButton(tb, ge.OpenPath).
		SetText("Open").SetIcon(icons.Open).SetKey(keymap.Open))

	views.NewFuncButton(tb, ge.SaveActiveView).SetText("Save").
		SetIcon(icons.Save).SetKey(keymap.Save)

	views.NewFuncButton(tb, ge.SaveAll).SetIcon(icons.Save)

	core.NewSeparator(tb)

	views.NewFuncButton(tb, ge.CursorToHistPrev).SetText("").SetKey(keymap.HistPrev).
		SetIcon(icons.KeyboardArrowLeft).SetShowReturn(false)
	views.NewFuncButton(tb, ge.CursorToHistNext).SetText("").SetKey(keymap.HistNext).
		SetIcon(icons.KeyboardArrowRight).SetShowReturn(false)

	core.NewSeparator(tb)

	// todo: this does not work to apply project defaults!
	ge.ConfigFindButton(views.NewFuncButton(tb, ge.Find).SetIcon(icons.FindReplace))

	core.NewSeparator(tb)

	views.NewFuncButton(tb, ge.Symbols).SetIcon(icons.List)

	views.NewFuncButton(tb, ge.Spell).SetIcon(icons.Spellcheck)

	core.NewSeparator(tb)

	views.NewFuncButton(tb, ge.Build).SetIcon(icons.Build).
		SetShortcut(key.Chord(ChordForFunction(KeyBuildProject).String()))

	views.NewFuncButton(tb, ge.Run).SetIcon(icons.PlayArrow).
		SetShortcut(key.Chord(ChordForFunction(KeyRunProject).String()))

	views.NewFuncButton(tb, ge.Debug).SetIcon(icons.Debug)

	views.NewFuncButton(tb, ge.DebugTest).SetIcon(icons.Debug)

	core.NewSeparator(tb)

	views.NewFuncButton(tb, ge.Commit).SetIcon(icons.Star)

	core.NewButton(tb).SetText("Command").
		SetShortcut(key.Chord(ChordForFunction(KeyExecCmd).String())).
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
							ge.ExecCmdNameActive(CommandName(cat, cm))
						})
					}
				})
			}
		})

	core.NewSeparator(tb)

	core.NewButton(tb).SetText("Splits").SetMenu(func(m *core.Scene) {
		core.NewButton(m).SetText("Set View").
			SetMenu(func(mm *core.Scene) {
				for _, sp := range AvailableSplitNames {
					sn := SplitName(sp)
					mb := core.NewButton(mm).SetText(sp).OnClick(func(e events.Event) {
						ge.SplitsSetView(sn)
					})
					if sn == ge.Settings.SplitName {
						mb.SetSelected(true)
					}
				}
			})
		views.NewFuncButton(m, ge.SplitsSaveAs).SetText("Save As")
		core.NewButton(m).SetText("Save").
			SetMenu(func(mm *core.Scene) {
				for _, sp := range AvailableSplitNames {
					sn := SplitName(sp)
					mb := core.NewButton(mm).SetText(sp).OnClick(func(e events.Event) {
						ge.SplitsSave(sn)
					})
					if sn == ge.Settings.SplitName {
						mb.SetSelected(true)
					}
				}
			})
		views.NewFuncButton(m, ge.SplitsEdit).SetText("Edit")
	})

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
						ge.FileNodeRunExe(fn)
					},
				})
			default:
				ac.Items = append(ac.Items, core.ChooserItem{
					Text: nmpath,
					Icon: fn.Info.Ic,
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
		if tv == nil || tv.Buffer == nil || !tv.Buffer.Hi.UsingParse() {
			return
		}
		pfs := tv.Buffer.ParseState.Done()
		if len(pfs.ParseState.Scopes) == 0 {
			return
		}
		pkg := pfs.ParseState.Scopes[0] // first scope of parse state is the full set of package symbols
		syms := &SymNode{}
		syms.InitName(syms, "syms")
		syms.OpenSyms(pkg, "", "")
		syms.WalkDown(func(k tree.Node) bool {
			sn := k.(*SymNode)
			ac.Items = append(ac.Items, core.ChooserItem{
				Text: sn.Symbol.Label(),
				Icon: sn.GetIcon(),
				Func: func() {
					SelectSymbol(ge, sn.Symbol)
				},
			})
			return tree.Continue
		})
	})
}
