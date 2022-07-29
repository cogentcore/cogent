// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import "github.com/goki/pi/filecat"

// Use these for more obvious command options
const (
	CmdWait      = true
	CmdNoWait    = false
	CmdFocus     = true
	CmdNoFocus   = false
	CmdConfirm   = true
	CmdNoConfirm = false
)

// StdCmds is the original compiled-in set of standard commands.
var StdCmds = Commands{
	{Name: "Run Proj",
		Desc: "run RunExec executable set in project",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "{RunExecPath}"}},
		Dir:  "{RunExecDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},
	{Name: "Run Prompt",
		Desc: "run any command you enter at the prompt",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "{PromptString1}"}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	// Make
	{Name: "Make",
		Desc: "run make with no args",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "make"}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Make Prompt",
		Desc: "run make with prompted make target",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "make",
			Args: []string{"{PromptString1}"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	// Go
	{Name: "Imports Go File",
		Desc: "run goimports on file",
		Lang: filecat.Go,
		Cmds: []CmdAndArgs{{Cmd: "goimports",
			Args: []string{"-w", "{FilePath}"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Fmt Go File",
		Desc: "run go fmt on file",
		Lang: filecat.Go,
		Cmds: []CmdAndArgs{{Cmd: "gofmt",
			Args: []string{"-w", "{FilePath}"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Build Go Dir",
		Desc: "run go build to build in current dir",
		Lang: filecat.Go,
		Cmds: []CmdAndArgs{{Cmd: "go",
			Args: []string{"build", "-v"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Build Go Proj",
		Desc: "run go build for project BuildDir",
		Lang: filecat.Go,
		Cmds: []CmdAndArgs{{Cmd: "go",
			Args: []string{"build", "-v"}}},
		Dir:  "{BuildDir}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Install Go Proj",
		Desc: "run go install for project BuildDir",
		Lang: filecat.Go,
		Cmds: []CmdAndArgs{{Cmd: "go",
			Args: []string{"install", "-v"}}},
		Dir:  "{BuildDir}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Generate Go",
		Desc: "run go generate in current dir",
		Lang: filecat.Go,
		Cmds: []CmdAndArgs{{Cmd: "go",
			Args: []string{"generate"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Test Go",
		Desc: "run go test in current dir",
		Lang: filecat.Go,
		Cmds: []CmdAndArgs{{Cmd: "go",
			Args: []string{"test", "-v"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Vet Go",
		Desc: "run go vet in current dir",
		Lang: filecat.Go,
		Cmds: []CmdAndArgs{{Cmd: "go",
			Args: []string{"vet"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Mod Tidy Go",
		Desc: "run go mod tidy in current dir",
		Lang: filecat.Go,
		Cmds: []CmdAndArgs{{Cmd: "go",
			Args: []string{"mod", "tidy"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Mod Init Go",
		Desc: "run go mod init in current dir with module path from prompt",
		Lang: filecat.Go,
		Cmds: []CmdAndArgs{{Cmd: "go",
			Args: []string{"mod", "init", "{PromptString1}"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Get Go",
		Desc: "run go get on package you enter at prompt",
		Lang: filecat.Go,
		Cmds: []CmdAndArgs{{Cmd: "go",
			Args: []string{"get", "{PromptString1}"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Get Go Updt",
		Desc: "run go get -u (updt) on package you enter at prompt",
		Lang: filecat.Go,
		Cmds: []CmdAndArgs{{Cmd: "go",
			Args: []string{"get", "{PromptString1}"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	// Git
	{Name: "Add Git",
		Desc: "git add file",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "git",
			Args: []string{"add", "{FilePath}"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Checkout Git",
		Desc: "git checkout: file, directory, branch; -b <branch> creates a new branch",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "git",
			Args: []string{"checkout", "{PromptString1}"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Status Git",
		Desc: "git status",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "git",
			Args: []string{"status"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Diff Git",
		Desc: "git diff -- see changes since last checkin",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "git",
			Args: []string{"diff"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Log Git",
		Desc: "git log",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "git",
			Args: []string{"log"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Commit Git",
		Desc: "git commit",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "git",
			Args: []string{"commit", "-am", "{PromptString1}"}, PromptIsString: true}},
		Dir:  "{FileDirPath}",
		Wait: CmdWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm}, // promptstring1 provided during normal commit process, MUST be wait!

	{Name: "Pull Git ",
		Desc: "git pull",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "git",
			Args:    []string{"pull", "{PromptString1}"},
			Default: "origin"}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Push Git ",
		Desc: "git push",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "git",
			Args:    []string{"push", "{PromptString1}"},
			Default: "origin"}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Branch Git",
		Desc: "git branch: -a shows all; <branchname> makes a new one, optionally given sha",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "git",
			Args:    []string{"branch", "{PromptString1}"},
			Default: "-a"}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	// SVN
	{Name: "Add SVN",
		Desc: "svn add file",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "svn",
			Args: []string{"add", "{FilePath}"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Status SVN",
		Desc: "svn status",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "svn",
			Args: []string{"status"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Info SVN",
		Desc: "svn info",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "svn",
			Args: []string{"info"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Log SVN",
		Desc: "svn log",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "svn",
			Args: []string{"log", "-v"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Commit SVN Proj",
		Desc: "svn commit for entire project directory",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "svn",
			Args: []string{"commit", "-m", "{PromptString1}"}, PromptIsString: true}},
		Dir:  "{ProjPath}",
		Wait: CmdWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm}, // promptstring1 provided during normal commit process

	{Name: "Commit SVN Dir",
		Desc: "svn commit in directory of current file",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "svn",
			Args: []string{"commit", "-m", "{PromptString1}"}, PromptIsString: true}},
		Dir:  "{FileDirPath}",
		Wait: CmdWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm}, // promptstring1 provided during normal commit process

	{Name: "Update SVN",
		Desc: "svn update",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "svn",
			Args: []string{"update"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	// LaTeX
	{Name: "LaTeX PDF",
		Desc: "run PDFLaTeX on file",
		Lang: filecat.TeX,
		Cmds: []CmdAndArgs{{Cmd: "pdflatex",
			Args: []string{"-file-line-error", "-interaction=nonstopmode", "{FilePath}"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "BibTeX",
		Desc: "run BibTeX on file",
		Lang: filecat.TeX,
		Cmds: []CmdAndArgs{{Cmd: "bibtex",
			Args: []string{"{FileNameNoExt}"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Biber",
		Desc: "run Biber on file",
		Lang: filecat.TeX,
		Cmds: []CmdAndArgs{{Cmd: "biber",
			Args: []string{"{FileNameNoExt}"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "CleanTeX",
		Desc: "remove aux LaTeX files",
		Lang: filecat.TeX,
		Cmds: []CmdAndArgs{{Cmd: "rm",
			Args: []string{"*.aux", "*.log", "*.blg", "*.bbl", "*.fff", "*.lof", "*.ttt", "*.toc", "*.spl"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	// Generic files / images / etc
	{Name: "Open File",
		Desc: "open file using OS 'open' command",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "open",
			Args: []string{"{FilePath}"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Open Target File",
		Desc: "open project target file using OS 'open' command",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "open",
			Args: []string{"{RunExecPath}"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	// Misc
	{Name: "List Dir",
		Desc: "list current dir",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "ls",
			Args: []string{"-la"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},

	{Name: "Grep",
		Desc: "recursive grep of all files for prompted value",
		Lang: filecat.Any,
		Cmds: []CmdAndArgs{{Cmd: "grep",
			Args: []string{"-R", "-e", "{PromptString1}", "{FileDirPath}"}}},
		Dir:  "{FileDirPath}",
		Wait: CmdNoWait, Focus: CmdNoFocus, Confirm: CmdNoConfirm},
}
