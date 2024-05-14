// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databrowser

//go:generate core generate

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/base/dirs"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/shell/interpreter"
	"cogentcore.org/core/views"
	"github.com/traefik/yaegi/interp"
	"golang.org/x/exp/maps"
)

// Browser is a data browser
type Browser struct {
	core.Layout

	// DataRoot is the path to the root of the data to browse
	DataRoot string

	// ScriptsDir is the directory containing scripts for actions to run
	ScriptsDir string

	// Scripts
	Scripts map[string]string

	// Interpreter for running scripts
	Interp *interpreter.Interpreter
}

// OnInit initializes with the data and script directories
func (br *Browser) OnInit() {
	br.Layout.OnInit()
	br.Interp = interpreter.NewInterpreter(interp.Options{})
}

func (br *Browser) RunScript(snm string) {
	sc := br.Scripts[snm]
	br.Interp.Eval(sc)
}

func (br *Browser) GetScripts() {
	scr := dirs.ExtFilenames(br.ScriptsDir, []string{".cosh"})
	br.Scripts = make(map[string]string)
	for _, s := range scr {
		snm := strings.TrimSuffix(s, ".cosh")
		sc, err := os.ReadFile(filepath.Join(br.ScriptsDir, s))
		if err == nil {
			br.Scripts[snm] = string(sc)
		} else {
			slog.Error(err.Error())
		}
	}
	// todo: tb needs to use new config!
	// tb := br.Scene.GetTopAppBar()
	// if tb != nil {
	// 	tb.Update()
	// }
}

func (br *Browser) Config(c *core.Config) {
	br.GetScripts()

	core.AddConfig(c, "splits", func() *core.Splits {
		w := core.NewSplits()
		w.SetSplits(.2, .8)
		return w
	})
	core.AddConfig(c, "splits/files", func() *filetree.Tree {
		w := filetree.NewTree()
		return w
	}, func(w *filetree.Tree) {
		if br.DataRoot != "" {
			os.Chdir(br.DataRoot)
			w.OpenPath(br.DataRoot)
		}
	})
	core.AddConfig(c, "splits/tabs", func() *core.Tabs {
		w := core.NewTabs()
		return w
	}, func(w *core.Tabs) {

	})

}

// UpdateFiles Updates the file view with current files in DataRoot
func (br *Browser) UpdateFiles() { //types:add
	files := br.FindPath("splits/files").(*filetree.Tree)
	files.OpenPath(br.DataRoot)
	os.Chdir(br.DataRoot)
}

func (br *Browser) ConfigAppBar(tb *core.Toolbar) {
	views.NewFuncButton(tb, br.UpdateFiles).SetText("").SetIcon(icons.Refresh).SetShortcut("Command+U")
	scr := maps.Keys(br.Scripts)
	slices.Sort(scr)
	for _, s := range scr {
		fmt.Println(scr)
		core.NewButton(tb).SetText(s).SetIcon(icons.RunCircle).
			SetTooltip("Run script").
			OnClick(func(e events.Event) {
				br.RunScript(s)
			})
	}
}
