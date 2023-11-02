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
	"goki.dev/icons"
)

func (ge *GideView) Toolbar(tb *gi.Toolbar) {
	gi.DefaultTopAppBar(tb)

	giv.NewFuncButton(tb, ge.UpdateFiles).SetIcon(icons.Refresh).SetShortcut("Command+U")
	op := giv.NewFuncButton(tb, ge.OpenPath).SetKey(keyfun.Open)
	_ = op
	// op.Args[0].SetValue(ge.ActiveFilename)
	giv.NewFuncButton(tb, ge.SaveActiveView).SetKey(keyfun.Save)

	gi.NewSeparator(tb)

	sm := gi.NewSwitch(tb, "go-mod").SetText("Go Mod").SetTooltip("Toggles the use of go modules -- saved with project -- if off, uses old school GOPATH mode")
	sm.SetChecked(ge.Prefs.GoMod)
	sm.OnClick(func(e events.Event) {
		ge.Prefs.GoMod = sm.StateIs(states.Checked)
		gide.SetGoMod(ge.Prefs.GoMod)
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
		{"UpdateFiles", ki.Props{
			"shortcut": "Command+U",
			"desc":     "update file browser list of files",
			"icon":     "update",
		}},
		{"NextViewFile", ki.Props{
			"label": "Open...",
			"icon":  "file-open",
			"desc":  "open a file in current active text view",
			"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Button) key.Chord {
				return key.Chord(gide.ChordForFun(gide.KeyFunFileOpen).String())
			}),
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "ActiveFilename",
				}},
			},
		}},
		{"SaveActiveView", ki.Props{
			"label": "Save",
			"desc":  "save active text view file to its current filename",
			"icon":  "file-save",
			"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Button) key.Chord {
				return key.Chord(gide.ChordForFun(gide.KeyFunBufSave).String())
			}),
		}},
		{"SaveActiveViewAs", ki.Props{
			"label": "Save As...",
			"icon":  "file-save",
			"desc":  "save active text view file to a new filename",
			"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Button) key.Chord {
				return key.Chord(gide.ChordForFun(gide.KeyFunBufSaveAs).String())
			}),
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "ActiveFilename",
				}},
			},
		}},
		{"SaveAll", ki.Props{
			"icon": "file-save",
			"desc": "save all open files (if modified) and the current project prefs (if .gide file exists, from prior Save Proj As..)",
		}},
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
		{"CursorToHistPrev", ki.Props{
			"icon":     "wedge-left",
			"shortcut": keyfun.HistPrev,
			"label":    "",
			"desc":     "move cursor to previous location in active text view",
		}},
		{"CursorToHistNext", ki.Props{
			"icon":     "wedge-right",
			"shortcut": keyfun.HistNext,
			"label":    "",
			"desc":     "move cursor to next location in active text view",
		}},
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
		{"Symbols", ki.Props{
			"icon": "structure",
		}},
		{"Spell", ki.Props{
			"label": "Spelling",
			"icon":  "spelling",
		}},
		{"sep-file", ki.BlankProp{}},
		{"Build", ki.Props{
			"icon": "terminal",
			"desc": "build the project -- command(s) specified in Project Prefs",
			"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Button) key.Chord {
				return key.Chord(gide.ChordForFun(gide.KeyFunBuildProj).String())
			}),
		}},
		{"Run", ki.Props{
			"icon": "terminal",
			"desc": "run the project -- command(s) specified in Project Prefs",
			"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Button) key.Chord {
				return key.Chord(gide.ChordForFun(gide.KeyFunRunProj).String())
			}),
		}},
		{"Debug", ki.Props{
			"icon": "terminal",
			"desc": "debug currently selected executable (context menu on executable, select Set Run Exec) -- if none selected, prompts to select one",
		}},
		{"DebugTest", ki.Props{
			"icon": "terminal",
			"desc": "debug test in current active view directory",
		}},
		{"sep-exe", ki.BlankProp{}},
		{"Commit", ki.Props{
			"icon": "star",
		}},
		{"ExecCmdNameActive", ki.Props{
			"icon":            "terminal",
			"label":           "Exec Cmd",
			"desc":            "execute given command on active file / directory / project",
			"subsubmenu-func": giv.SubSubMenuFunc(ExecCmds),
			"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Button) key.Chord {
				return key.Chord(gide.ChordForFun(gide.KeyFunExecCmd).String())
			}),
			"Args": ki.PropSlice{
				{"Cmd Name", ki.Props{}},
			},
		}},
		{"sep-splt", ki.BlankProp{}},
		{"Splits", ki.PropSlice{
			{"SplitsSetView", ki.Props{
				"label":   "Set View",
				"submenu": &gide.AvailSplitNames,
				"Args": ki.PropSlice{
					{"Split Name", ki.Props{
						"default-field": "Prefs.SplitName",
					}},
				},
			}},
			{"SplitsSaveAs", ki.Props{
				"label": "Save As...",
				"desc":  "save current splitter values to a new named split configuration",
				"Args": ki.PropSlice{
					{"Name", ki.Props{
						"width": 60,
					}},
					{"Desc", ki.Props{
						"width": 60,
					}},
				},
			}},
			{"SplitsSave", ki.Props{
				"label":   "Save",
				"submenu": &gide.AvailSplitNames,
				"Args": ki.PropSlice{
					{"Split Name", ki.Props{
						"default-field": "Prefs.SplitName",
					}},
				},
			}},
			{"SplitsEdit", ki.Props{
				"label": "Edit...",
			}},
		}},
	},
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"OpenRecent", ki.Props{
				"submenu": &gide.SavedPaths,
				"Args": ki.PropSlice{
					{"File Name", ki.Props{}},
				},
			}},
			{"OpenProj", ki.Props{
				"shortcut": keyfun.MenuOpen,
				"label":    "Open Project...",
				"desc":     "open a gide project -- can be a .gide file or just a file or directory (projects are just directories with relevant files)",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "ProjFilename",
						"ext":           ".gide",
					}},
				},
			}},
			{"OpenPath", ki.Props{
				"shortcut": keyfun.MenuOpenAlt1,
				"label":    "Open Path...",
				"desc":     "open a gide project for a file or directory (projects are just directories with relevant files)",
				"Args": ki.PropSlice{
					{"Path", ki.Props{}},
				},
			}},
			{"New", ki.PropSlice{
				{"NewProj", ki.Props{
					"shortcut": keyfun.MenuNew,
					"label":    "New Project...",
					"desc":     "Create a new project -- select a path for the parent folder, and a folder name for the new project -- all GideView projects are basically folders with files.  You can also specify the main language and {version control system for the project.  For other options, do <code>Proj Prefs</code> in the File menu of the new project.",
					"Args": ki.PropSlice{
						{"Parent Folder", ki.Props{
							"dirs-only":     true, // todo: support
							"default-field": "ProjRoot",
						}},
						{"Folder", ki.Props{
							"width": 60,
						}},
						{"Main Lang", ki.Props{}},
						{"Version Ctrl", ki.Props{}},
					},
				}},
				{"NewFile", ki.Props{
					"shortcut": keyfun.MenuNewAlt1,
					"label":    "New File...",
					"desc":     "Create a new file in project -- to create in sub-folders, use context menu on folder in file browser",
					"Args": ki.PropSlice{
						{"File Name", ki.Props{
							"width": 60,
						}},
						{"Add To Version Control", ki.Props{}},
					},
				}},
			}},
			{"SaveProj", ki.Props{
				"shortcut": keyfun.MenuSave,
				"label":    "Save Project",
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"SaveProjAs", ki.Props{
				"shortcut": keyfun.MenuSaveAs,
				"label":    "Save Project As...",
				"desc":     "Save project to given file name -- this is the .gide file containing preferences and current settings -- also saves all open files -- once saved, further saving is automatic",
				"updtfunc": GideViewInactiveEmptyFunc,
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "ProjFilename",
						"ext":           ".gide",
					}},
					{"SaveAll", ki.Props{
						"value": false,
					}},
				},
			}},
			{"SaveAll", ki.Props{}},
			{"sep-af", ki.BlankProp{}},
			{"ViewFile", ki.Props{
				"label": "Open File...",
				"shortcut-func": func(gei any, act *gi.Button) key.Chord {
					return key.Chord(gide.ChordForFun(gide.KeyFunFileOpen).String())
				},
				"updtfunc": GideViewInactiveEmptyFunc,
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "ActiveFilename",
					}},
				},
			}},
			{"SaveActiveView", ki.Props{
				"label": "Save File",
				"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Button) key.Chord {
					return key.Chord(gide.ChordForFun(gide.KeyFunBufSave).String())
				}),
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"SaveActiveViewAs", ki.Props{
				"label":    "Save File As...",
				"updtfunc": GideViewInactiveEmptyFunc,
				"desc":     "save active text view file to a new filename",
				"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Button) key.Chord {
					return key.Chord(gide.ChordForFun(gide.KeyFunBufSaveAs).String())
				}),
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "ActiveFilename",
					}},
				},
			}},
			{"RevertActiveView", ki.Props{
				"desc":     "Revert active file to last saved version: this will lose all active changes -- are you sure?",
				"confirm":  true,
				"label":    "Revert File...",
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"CloseActiveView", ki.Props{
				"label":    "Close File",
				"updtfunc": GideViewInactiveEmptyFunc,
				"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Button) key.Chord {
					return key.Chord(gide.ChordForFun(gide.KeyFunBufClose).String())
				}),
			}},
			{"sep-prefs", ki.BlankProp{}},
			{"EditProjPrefs", ki.Props{
				"label":    "Project Prefs...",
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"sep-close", ki.BlankProp{}},
			{"Close Window", ki.BlankProp{}},
		}},
		{"Edit", ki.PropSlice{
			{"Copy", ki.Props{
				"keyfun":   keyfun.Copy,
				"updtfunc": GideViewInactiveTextSelectionFunc,
			}},
			{"Cut", ki.Props{
				"keyfun":   keyfun.Cut,
				"updtfunc": GideViewInactiveTextSelectionFunc,
			}},
			{"Paste", ki.Props{
				"keyfun": keyfun.Paste,
			}},
			{"Paste History...", ki.Props{
				"keyfun": keyfun.PasteHist,
			}},
			{"Registers", ki.PropSlice{
				{"RegisterCopy", ki.Props{
					"label": "Copy...",
					"desc":  "save currently-selected text to a named register, which can be pasted later -- persistent across sessions as well",
					"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Button) key.Chord {
						return key.Chord(gide.ChordForFun(gide.KeyFunRegCopy).String())
					}),
					"updtfunc": GideViewInactiveEmptyFunc,
					"Args": ki.PropSlice{
						{"Register Name", ki.Props{
							"default": "", // override memory of last
						}},
					},
				}},
				{"RegisterPaste", ki.Props{
					"label": "Paste...",
					"desc":  "paste text from named register",
					"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Button) key.Chord {
						return key.Chord(gide.ChordForFun(gide.KeyFunRegPaste).String())
					}),
					"updtfunc": GideViewInactiveEmptyFunc,
					"Args": ki.PropSlice{
						{"Register Name", ki.Props{
							"default-field": "Prefs.Register",
						}},
					},
				}},
			}},
			{"sep-undo", ki.BlankProp{}},
			{"Undo", ki.Props{
				"keyfun": keyfun.Undo,
			}},
			{"Redo", ki.Props{
				"keyfun": keyfun.Redo,
			}},
			{"sep-find", ki.BlankProp{}},
			{"Find", ki.Props{
				"label":    "Find...",
				"shortcut": keyfun.Find,
				"desc":     "Find / replace in all open folders in file browser",
				"updtfunc": GideViewInactiveEmptyFunc,
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
			{"ReplaceInActive", ki.Props{
				"label":    "Replace In Active...",
				"shortcut": keyfun.Replace,
				"desc":     "query-replace in current active text view only (use Find for multi-file)",
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"Spell", ki.Props{
				"label":    "Spelling...",
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"ShowCompletions", ki.Props{
				"keyfun":   keyfun.Complete,
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"LookupSymbol", ki.Props{
				"keyfun":   keyfun.Lookup,
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"sep-adv", ki.BlankProp{}},
			{"CommentOut", ki.Props{
				"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Button) key.Chord {
					return key.Chord(gide.ChordForFun(gide.KeyFunCommentOut).String())
				}),
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"Indent", ki.Props{
				"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Button) key.Chord {
					return key.Chord(gide.ChordForFun(gide.KeyFunIndent).String())
				}),
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"sep-xform", ki.BlankProp{}},
			{"ReCase", ki.Props{
				"desc":     "replace currently-selected text with text of given case",
				"updtfunc": GideViewInactiveEmptyFunc,
				"Args": ki.PropSlice{
					{"To Case", ki.Props{}},
				},
			}},
			{"JoinParaLines", ki.Props{
				"desc":     "merges sequences of lines with hard returns forming paragraphs, separated by blank lines, into a single line per paragraph, for given selected region (full text if no selection)",
				"confirm":  true,
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"TabsToSpaces", ki.Props{
				"desc":     "converts tabs to spaces for given selected region (full text if no selection)",
				"confirm":  true,
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
			{"SpacesToTabs", ki.Props{
				"desc":     "converts spaces to tabs for given selected region (full text if no selection)",
				"confirm":  true,
				"updtfunc": GideViewInactiveEmptyFunc,
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
			{"Splits", ki.PropSlice{
				{"SplitsSetView", ki.Props{
					"label":    "Set View",
					"submenu":  &gide.AvailSplitNames,
					"updtfunc": GideViewInactiveEmptyFunc,
					"Args": ki.PropSlice{
						{"Split Name", ki.Props{}},
					},
				}},
				{"SplitsSaveAs", ki.Props{
					"label":    "Save As...",
					"desc":     "save current splitter values to a new named split configuration",
					"updtfunc": GideViewInactiveEmptyFunc,
					"Args": ki.PropSlice{
						{"Name", ki.Props{
							"width": 60,
						}},
						{"Desc", ki.Props{
							"width": 60,
						}},
					},
				}},
				{"SplitsSave", ki.Props{
					"label":    "Save",
					"submenu":  &gide.AvailSplitNames,
					"updtfunc": GideViewInactiveEmptyFunc,
					"Args": ki.PropSlice{
						{"Split Name", ki.Props{}},
					},
				}},
				{"SplitsEdit", ki.Props{
					"updtfunc": GideViewInactiveEmptyFunc,
					"label":    "Edit...",
				}},
			}},
			{"OpenConsoleTab", ki.Props{
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
		}},
		{"Navigate", ki.PropSlice{
			{"Cursor", ki.PropSlice{
				{"Back", ki.Props{
					"keyfun": keyfun.HistPrev,
				}},
				{"Forward", ki.Props{
					"keyfun": keyfun.HistNext,
				}},
				{"Jump To Line", ki.Props{
					"keyfun": keyfun.Jump,
				}},
			}},
		}},
		{"Command", ki.PropSlice{
			{"Build", ki.Props{
				"updtfunc": GideViewInactiveEmptyFunc,
				"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Button) key.Chord {
					return key.Chord(gide.ChordForFun(gide.KeyFunBuildProj).String())
				}),
			}},
			{"Run", ki.Props{
				"updtfunc": GideViewInactiveEmptyFunc,
				"shortcut-func": giv.ShortcutFunc(func(gei any, act *gi.Button) key.Chord {
					return key.Chord(gide.ChordForFun(gide.KeyFunRunProj).String())
				}),
			}},
			{"Debug", ki.Props{}},
			{"DebugTest", ki.Props{}},
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
			{"Commit", ki.Props{
				"updtfunc": GideViewInactiveEmptyFunc,
			}},
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
		{"Window", "Windows"},
		{"Help", ki.PropSlice{
			{"HelpWiki", ki.Props{}},
		}},
	},
	"CallMethods": ki.PropSlice{
		{"NextViewFile", ki.Props{
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "ActiveFilename",
				}},
			},
		}},
		{"SplitsSetView", ki.Props{
			"Args": ki.PropSlice{
				{"Split Name", ki.Props{}},
			},
		}},
		{"ExecCmd", ki.Props{}},
		{"ChooseRunExec", ki.Props{
			"Args": ki.PropSlice{
				{"Exec File Name", ki.Props{}},
			},
		}},
	},
}
*/
