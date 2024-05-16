// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databrowser

//go:generate core generate

import (
	"errors"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"cogentcore.org/core/base/dirs"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/shell"
	"cogentcore.org/core/shell/interpreter"
	"cogentcore.org/core/views"
	"github.com/ergochat/readline"
	"github.com/traefik/yaegi/interp"
	"golang.org/x/exp/maps"
)

// Browser is a data browser
type Browser struct {
	core.Frame

	// DataRoot is the path to the root of the data to browse
	DataRoot string

	// ScriptsDir is the directory containing scripts for actions to run
	ScriptsDir string

	// Scripts
	Scripts map[string]string `set:"-"`

	// Interpreter for running scripts
	Interp *interpreter.Interpreter
}

// OnInit initializes with the data and script directories
func (br *Browser) OnInit() {
	br.Frame.OnInit()
	br.Interp = interpreter.NewInterpreter(interp.Options{})

	br.Interp.Interp.Use(interp.Exports{
		"cogentcore.org/cogent/numbers/databrowser/browser": map[string]reflect.Value{
			"Update":      reflect.ValueOf(br.Update),
			"SetDataRoot": reflect.ValueOf(br.SetDataRoot),
			"OpenDataTab": reflect.ValueOf(br.OpenDataTab),
			"DataRoot":    reflect.ValueOf(br.GetDataRoot),
		},
	})
	br.Interp.Interp.ImportUsed()
	br.Interp.RunConfig()
}

func (br *Browser) GetDataRoot() string {
	return br.DataRoot
}

func (br *Browser) RunScript(snm string) {
	sc, ok := br.Scripts[snm]
	if !ok {
		slog.Error("script not found:", "Script:", snm)
		return
	}
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

func (br *Browser) Splits() *core.Splits {
	return br.FindPath("splits").(*core.Splits)
}

func (br *Browser) FileTree() *filetree.Tree {
	sp := br.Splits()
	return sp.Child(0).(*filetree.Tree) // note: gets renamed by dir name
}

func (br *Browser) Tabs() *core.Tabs {
	return br.FindPath("splits/tabs").(*core.Tabs)
}

// UpdateFiles Updates the file view with current files in DataRoot
func (br *Browser) UpdateFiles() { //types:add
	files := br.FileTree()
	files.OpenPath(br.DataRoot)
	os.Chdir(br.DataRoot)
}

func (br *Browser) ConfigAppBar(tb *core.Toolbar) {
	views.NewFuncButton(tb, br.UpdateFiles).SetText("").SetIcon(icons.Refresh).SetShortcut("Command+U")
	scr := maps.Keys(br.Scripts)
	slices.Sort(scr)
	for _, s := range scr {
		core.NewButton(tb).SetText(s).SetIcon(icons.RunCircle).
			SetTooltip("Run script").
			OnClick(func(e events.Event) {
				br.RunScript(s)
			})
	}
}

// Shell runs an interactive shell that allows the user to input cosh.
func (br *Browser) Shell() error {
	in := br.Interp
	rl, err := readline.NewFromConfig(&readline.Config{
		AutoComplete: &shell.ReadlineCompleter{Shell: in.Shell},
		Undo:         true,
	})
	if err != nil {
		return err
	}
	defer rl.Close()
	log.SetOutput(rl.Stderr()) // redraw the prompt correctly after log output

	for {
		rl.SetPrompt(in.Prompt())
		line, err := rl.ReadLine()
		if errors.Is(err, readline.ErrInterrupt) {
			continue
		}
		if errors.Is(err, io.EOF) {
			os.Exit(0)
		}
		if err != nil {
			return err
		}
		in.Eval(line)
	}
}
