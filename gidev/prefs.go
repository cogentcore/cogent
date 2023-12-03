// Copyright (c) 2023, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidev

import (
	"path/filepath"

	"goki.dev/fi"
	"goki.dev/gi/v2/filetree"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gide/v2/gide"
	"goki.dev/gide/v2/gidebug"
	"goki.dev/goosi/events"
)

// Defaults sets new project defaults based on overall preferences
func (ge *GideView) Defaults() {
	ge.Prefs.Files = gide.Prefs.Files
	ge.Prefs.Editor = gi.Prefs.Editor
	ge.Prefs.Splits = []float32{.1, .325, .325, .25}
	ge.Prefs.Debug = gidebug.DefaultParams
}

// GrabPrefs grabs the current project preference settings from various
// places, e.g., prior to saving or editing.
func (ge *GideView) GrabPrefs() {
	sv := ge.Splits()
	ge.Prefs.Splits = sv.Splits
	ge.Prefs.Dirs = ge.Files.Dirs
}

// ApplyPrefs applies current project preference settings into places where
// they are used -- only for those done prior to loading
func (ge *GideView) ApplyPrefs() {
	ge.ProjFilename = ge.Prefs.ProjFilename
	ge.ProjRoot = ge.Prefs.ProjRoot
	if ge.Files != nil {
		ge.Files.Dirs = ge.Prefs.Dirs
		ge.Files.DirsOnTop = ge.Prefs.Files.DirsOnTop
	}
	if len(ge.Kids) > 0 {
		for i := 0; i < NTextEditors; i++ {
			tv := ge.TextEditorByIndex(i)
			if tv.Buf != nil {
				ge.ConfigTextBuf(tv.Buf)
			}
		}
		for _, ond := range ge.OpenNodes {
			if ond.Buf != nil {
				ge.ConfigTextBuf(ond.Buf)
			}
		}
		split := ge.Splits()
		split.SetSplits(ge.Prefs.Splits...)
	}
	gi.Prefs.UpdateAll() // drives full rebuild
}

// ApplyPrefsAction applies current preferences to the project, and updates the project
func (ge *GideView) ApplyPrefsAction() {
	ge.ApplyPrefs()
	ge.SplitsSetView(ge.Prefs.SplitName)
	ge.SetStatus("Applied prefs")
}

// EditProjPrefs allows editing of project preferences (settings specific to this project)
func (ge *GideView) EditProjPrefs() { //gti:add
	sv := gide.ProjPrefsView(&ge.Prefs)
	if sv != nil {
		sv.OnChange(func(e events.Event) {
			ge.ApplyPrefsAction()
		})
	}
}

func (ge *GideView) CallSplitsSetView(ctx gi.Widget) {
	fb := giv.NewSoloFuncButton(ctx, ge.SplitsSetView)
	fb.Args[0].SetValue(ge.Prefs.SplitName)
	fb.CallFunc()
}

// SplitsSetView sets split view splitters to given named setting
func (ge *GideView) SplitsSetView(split gide.SplitName) { //gti:add
	sv := ge.Splits()
	sp, _, ok := gide.AvailSplits.SplitByName(split)
	if ok {
		sv.SetSplitsAction(sp.Splits...)
		ge.Prefs.SplitName = split
		if !ge.PanelIsOpen(ge.ActiveTextEditorIdx + TextEditor1Idx) {
			ge.SetActiveTextEditorIdx((ge.ActiveTextEditorIdx + 1) % 2)
		}
	}
}

// SplitsSave saves current splitter settings to named splitter settings under
// existing name, and saves to prefs file
func (ge *GideView) SplitsSave(split gide.SplitName) { //gti:add
	sv := ge.Splits()
	sp, _, ok := gide.AvailSplits.SplitByName(split)
	if ok {
		sp.SaveSplits(sv.Splits)
		gide.AvailSplits.SavePrefs()
	}
}

// SplitsSaveAs saves current splitter settings to new named splitter settings, and
// saves to prefs file
func (ge *GideView) SplitsSaveAs(name, desc string) { //gti:add
	sv := ge.Splits()
	gide.AvailSplits.Add(name, desc, sv.Splits)
	gide.AvailSplits.SavePrefs()
}

// SplitsEdit opens the SplitsView editor to customize saved splitter settings
func (ge *GideView) SplitsEdit() { //gti:add
	gide.SplitsView(&gide.AvailSplits)
}

// LangDefaults applies default language settings based on MainLang
func (ge *GideView) LangDefaults() {
	ge.Prefs.RunCmds = gide.CmdNames{"Build: Run Proj"}
	ge.Prefs.BuildDir = ge.Prefs.ProjRoot
	ge.Prefs.BuildTarg = ge.Prefs.ProjRoot
	ge.Prefs.RunExec = gi.FileName(filepath.Join(string(ge.Prefs.ProjRoot), ge.Nm))
	if len(ge.Prefs.BuildCmds) == 0 {
		switch ge.Prefs.MainLang {
		case fi.Go:
			ge.Prefs.BuildCmds = gide.CmdNames{"Go: Build Proj"}
		case fi.TeX:
			ge.Prefs.BuildCmds = gide.CmdNames{"LaTeX: LaTeX PDF"}
			ge.Prefs.RunCmds = gide.CmdNames{"File: Open Target"}
		default:
			ge.Prefs.BuildCmds = gide.CmdNames{"Build: Make"}
		}
	}
	if ge.Prefs.VersCtrl == "" {
		repo, _ := ge.Files.FirstVCS()
		if repo != nil {
			ge.Prefs.VersCtrl = filetree.VersCtrlName(repo.Vcs())
		}
	}
}

// GuessMainLang guesses the main language in the project -- returns true if successful
func (ge *GideView) GuessMainLang() bool {
	ecsc := ge.Files.FileExtCounts(fi.Code)
	ecsd := ge.Files.FileExtCounts(fi.Doc)
	ecs := append(ecsc, ecsd...)
	filetree.NodeNameCountSort(ecs)
	for _, ec := range ecs {
		ls := fi.ExtKnown(ec.Name)
		if ls != fi.Unknown {
			ge.Prefs.MainLang = ls
			return true
		}
	}
	return false
}
