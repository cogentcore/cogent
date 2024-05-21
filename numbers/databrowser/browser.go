// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databrowser

//go:generate core generate
//go:generate yaegi extract cogentcore.org/cogent/numbers/databrowser

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"cogentcore.org/cogent/numbers/numshell"
	"cogentcore.org/core/base/dirs"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/views"
	"github.com/traefik/yaegi/interp"
	"golang.org/x/exp/maps"
)

// TheBrowser is the current browser,
// which is valid immediately after NewBrowserWindow
// where it is used to get a local variable for subsequent use.
var TheBrowser *Browser

// Symbols variable stores the map of stdlib symbols per package.
var Symbols = map[string]map[string]reflect.Value{}

// MapTypes variable contains a map of functions which have an interface{} as parameter but
// do something special if the parameter implements a given interface.
var MapTypes = map[reflect.Value][]reflect.Type{}

// Browser is a data browser, for browsing data typically organized into
// separate directories, with .cosh Scripts as toolbar actions to perform
// regular tasks on the data.
// Scripts are ordered alphabetically and any leading #- prefix is automatically
// removed from the label, so you can use numbers to specify a custom order.
type Browser struct {
	core.Frame

	// DataRoot is the path to the root of the data to browse
	DataRoot string

	// ScriptsDir is the directory containing scripts for toolbar actions.
	// It defaults to DataDir/dbscripts
	ScriptsDir string

	// Scripts
	Scripts map[string]string `set:"-"`

	// ScriptInterp is the interpreter to use for running Browser scripts
	ScriptInterp *numshell.Interpreter `set:"-"`
}

// OnInit initializes with the data and script directories
func (br *Browser) OnInit() {
	br.Frame.OnInit()
	br.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	br.InitInterp()
}

// NewBrowserWindow opens a new data Browser for given data directory.
// By default the scripts for this data directory are located in
// dbscripts relative to the data directory.
func NewBrowserWindow(dataDir string) *Browser {
	b := core.NewBody("Cogent Data Browser")
	br := NewBrowser(b)
	ddr := errors.Log1(filepath.Abs(dataDir))
	fmt.Println(ddr)
	br.SetDataRoot(ddr)
	br.SetScriptsDir(filepath.Join(ddr, "dbscripts"))
	br.GetScripts()
	b.AddAppBar(br.ConfigAppBar)
	b.RunWindow()
	TheBrowser = br
	br.ScriptInterp.Eval("br := databrowser.TheBrowser") // grab it
	return br
}

func (br *Browser) InitInterp() {
	br.ScriptInterp = numshell.NewInterpreter(interp.Options{})
	br.ScriptInterp.Interp.Use(Symbols)
	// br.ScriptInterp.Interp.Use(interp.Exports{
	// 	"cogentcore.org/cogent/numbers/databrowser/databrowser": map[string]reflect.Value{
	// 		"br": reflect.ValueOf(br).Elem(), // note this does not work
	// 	},
	// })
	br.ScriptInterp.Config()
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
	fmt.Println("\n################\nrunning script:\n", sc, "\n")
	br.ScriptInterp.Eval(sc)
}

func (br *Browser) GetScripts() {
	scr := dirs.ExtFilenames(br.ScriptsDir, ".cosh")
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

func (br *Browser) Config(c *core.Plan) {
	br.GetScripts()

	core.Configure(c, "splits", func(w *core.Splits) {
		w.SetSplits(.2, .8)
	})
	core.Configure(c, "splits/files", func(w *filetree.Tree) {
	}, func(w *filetree.Tree) {
		if br.DataRoot != "" {
			errors.Log(os.Chdir(br.DataRoot))
			wd := errors.Log1(os.Getwd())
			w.OpenPath(wd)
		}
	})
	core.Configure(c, "splits/tabs", func(w *core.Tabs) {
		w.Type = core.FunctionalTabs
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
	br.GetScripts()
	br.Update()
}

func (br *Browser) ConfigAppBar(c *core.Plan) {
	core.Configure(c, "", func(w *views.FuncButton) {
		w.SetFunc(br.UpdateFiles).SetText("").SetIcon(icons.Refresh).SetShortcut("Command+U")
	})
	scr := maps.Keys(br.Scripts)
	slices.Sort(scr)
	for _, s := range scr {
		core.Configure(c, s, func(w *core.Button) {
			w.SetText(s).SetIcon(icons.RunCircle).
				SetTooltip("Run script").
				OnClick(func(e events.Event) {
					br.RunScript(s)
				})
		})
	}
}
