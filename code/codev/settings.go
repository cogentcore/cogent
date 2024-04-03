// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codev

import (
	"path/filepath"

	"cogentcore.org/cogent/code/cdebug"
	"cogentcore.org/cogent/code/code"
	"cogentcore.org/core/events"
	"cogentcore.org/core/fi"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
)

// Defaults sets new project defaults based on overall settings
func (ge *CodeView) Defaults() {
	ge.Settings.Files = code.Settings.Files
	ge.Settings.Editor = gi.SystemSettings.Editor
	ge.Settings.Splits = []float32{.1, .325, .325, .25}
	ge.Settings.Debug = cdebug.DefaultParams
}

// GrabPrefs grabs the current project preference settings from various
// places, e.g., prior to saving or editing.
func (ge *CodeView) GrabPrefs() {
	sv := ge.Splits()
	ge.Settings.Splits = sv.Splits
	ge.Settings.Dirs = ge.Files.Dirs
}

// ApplyPrefs applies current project preference settings into places where
// they are used -- only for those done prior to loading
func (ge *CodeView) ApplyPrefs() {
	ge.ProjFilename = ge.Settings.ProjFilename
	ge.ProjRoot = ge.Settings.ProjRoot
	if ge.Files != nil {
		ge.Files.Dirs = ge.Settings.Dirs
		ge.Files.DirsOnTop = ge.Settings.Files.DirsOnTop
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
		ge.Splits().SetSplits(ge.Settings.Splits...)
	}
	gi.UpdateAll() // drives full rebuild
}

// ApplyPrefsAction applies current settings to the project, and updates the project
func (ge *CodeView) ApplyPrefsAction() {
	ge.ApplyPrefs()
	ge.SplitsSetView(ge.Settings.SplitName)
	ge.SetStatus("Applied prefs")
}

// EditProjSettings allows editing of project settings (settings specific to this project)
func (ge *CodeView) EditProjSettings() { //gti:add
	sv := code.ProjSettingsView(&ge.Settings)
	if sv != nil {
		sv.OnChange(func(e events.Event) {
			ge.ApplyPrefsAction()
		})
	}
}

func (ge *CodeView) CallSplitsSetView(ctx gi.Widget) {
	fb := giv.NewSoloFuncButton(ctx, ge.SplitsSetView)
	fb.Args[0].SetValue(ge.Settings.SplitName)
	fb.CallFunc()
}

// SplitsSetView sets split view splitters to given named setting
func (ge *CodeView) SplitsSetView(split code.SplitName) { //gti:add
	sv := ge.Splits()
	sp, _, ok := code.AvailableSplits.SplitByName(split)
	if ok {
		sv.SetSplits(sp.Splits...).NeedsLayout()
		ge.Settings.SplitName = split
		if !ge.PanelIsOpen(ge.ActiveTextEditorIdx + TextEditor1Idx) {
			ge.SetActiveTextEditorIdx((ge.ActiveTextEditorIdx + 1) % 2)
		}
	}
}

// SplitsSave saves current splitter settings to named splitter settings under
// existing name, and saves to prefs file
func (ge *CodeView) SplitsSave(split code.SplitName) { //gti:add
	sv := ge.Splits()
	sp, _, ok := code.AvailableSplits.SplitByName(split)
	if ok {
		sp.SaveSplits(sv.Splits)
		code.AvailableSplits.SavePrefs()
	}
}

// SplitsSaveAs saves current splitter settings to new named splitter settings, and
// saves to prefs file
func (ge *CodeView) SplitsSaveAs(name, desc string) { //gti:add
	sv := ge.Splits()
	code.AvailableSplits.Add(name, desc, sv.Splits)
	code.AvailableSplits.SavePrefs()
}

// SplitsEdit opens the SplitsView editor to customize saved splitter settings
func (ge *CodeView) SplitsEdit() { //gti:add
	code.SplitsView(&code.AvailableSplits)
}

// LangDefaults applies default language settings based on MainLang
func (ge *CodeView) LangDefaults() {
	ge.Settings.RunCmds = code.CmdNames{"Build: Run Proj"}
	ge.Settings.BuildDir = ge.Settings.ProjRoot
	ge.Settings.BuildTarg = ge.Settings.ProjRoot
	ge.Settings.RunExec = gi.Filename(filepath.Join(string(ge.Settings.ProjRoot), ge.Nm))
	if len(ge.Settings.BuildCmds) == 0 {
		switch ge.Settings.MainLang {
		case fi.Go:
			ge.Settings.BuildCmds = code.CmdNames{"Go: Build Proj"}
		case fi.TeX:
			ge.Settings.BuildCmds = code.CmdNames{"LaTeX: LaTeX PDF"}
			ge.Settings.RunCmds = code.CmdNames{"File: Open Target"}
		default:
			ge.Settings.BuildCmds = code.CmdNames{"Build: Make"}
		}
	}
	if ge.Settings.VersionControl == "" {
		repo, _ := ge.Files.FirstVCS()
		if repo != nil {
			ge.Settings.VersionControl = filetree.VersionControlName(repo.Vcs())
		}
	}
}

// GuessMainLang guesses the main language in the project -- returns true if successful
func (ge *CodeView) GuessMainLang() bool {
	ecsc := ge.Files.FileExtCounts(fi.Code)
	ecsd := ge.Files.FileExtCounts(fi.Doc)
	ecs := append(ecsc, ecsd...)
	filetree.NodeNameCountSort(ecs)
	for _, ec := range ecs {
		ls := fi.ExtKnown(ec.Name)
		if ls != fi.Unknown {
			ge.Settings.MainLang = ls
			return true
		}
	}
	return false
}
