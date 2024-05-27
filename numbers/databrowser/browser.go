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
	"strconv"
	"strings"
	"unicode"

	"cogentcore.org/cogent/numbers/numshell"
	"cogentcore.org/core/base/dirs"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/logx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
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

	// StartDir is the starting directory, where the numbers app
	// was originally started.
	StartDir string

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

	br.Maker(func(p *core.Plan) {
		sp := core.AddAt(p, "splits", func(w *core.Splits) {
			w.SetSplits(.15, .85)
		})
		core.AddAt(sp, "files", func(w *filetree.Tree) {
			w.Builder(func() {
				if br.DataRoot != "" {
					errors.Log(os.Chdir(br.DataRoot))
					wd := errors.Log1(os.Getwd())
					w.OpenPath(wd)
				}
			})
		})
		core.AddAt(sp, "tabs", func(w *core.Tabs) {
			w.Type = core.FunctionalTabs
		})
	})
}

// NewBrowserWindow opens a new data Browser for given data directory.
// By default the scripts for this data directory are located in
// dbscripts relative to the data directory.
func NewBrowserWindow(dataDir string) *Browser {
	b := core.NewBody("Cogent Data Browser")
	br := NewBrowser(b)
	br.StartDir, _ = os.Getwd()
	br.StartDir = errors.Log1(filepath.Abs(br.StartDir))
	ddr := errors.Log1(filepath.Abs(dataDir))
	fmt.Println(ddr)
	b.AddAppBar(br.MakeToolbar)

	br.SetDataRoot(ddr)
	br.SetScriptsDir(filepath.Join(ddr, "dbscripts"))
	TheBrowser = br
	br.ScriptInterp.Eval("br := databrowser.TheBrowser") // grab it
	br.UpdateScripts()
	b.RunWindow()
	return br
}

// ParentBrowser returns the Browser parent of given node
func ParentBrowser(tn tree.Node) (*Browser, bool) {
	var res *Browser
	tn.WalkUp(func(n tree.Node) bool {
		if c, ok := n.This().(*Browser); ok {
			res = c
			return false
		}
		return true
	})
	return res, res != nil
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
	// logx.UserLevel = slog.LevelDebug // for debugging of init loading
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
	logx.PrintlnDebug("\n################\nrunning script:\n", sc, "\n")
	_, _, err := br.ScriptInterp.Eval(sc)
	if err == nil {
		err = br.ScriptInterp.Shell.DepthError()
	}
	br.ScriptInterp.Shell.ResetDepth()
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

// UpdateFiles Updates the file view with current files in DataRoot,
func (br *Browser) UpdateFiles() { //types:add
	files := br.FileTree()
	files.OpenPath(br.DataRoot)
	os.Chdir(br.DataRoot)
	br.Update()
}

// UpdateScripts updates the Scripts and updates the toolbar.
func (br *Browser) UpdateScripts() { //types:add
	redo := (br.Scripts != nil)
	scr := dirs.ExtFilenames(br.ScriptsDir, ".cosh")
	br.Scripts = make(map[string]string)
	for _, s := range scr {
		snm := strings.TrimSuffix(s, ".cosh")
		sc, err := os.ReadFile(filepath.Join(br.ScriptsDir, s))
		if err == nil {
			if unicode.IsLower(rune(snm[0])) {
				if !redo {
					fmt.Println("run init script:", snm)
					br.ScriptInterp.Eval(string(sc))
				}
			} else {
				br.Scripts[snm] = string(sc)
			}
		} else {
			slog.Error(err.Error())
		}
	}
	tb := br.Scene.GetTopAppBar()
	if tb != nil {
		tb.Update()
	}
}

func (br *Browser) MakeToolbar(p *core.Plan) {
	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(br.UpdateFiles).SetText("").SetIcon(icons.Refresh).SetShortcut("Command+U")
	})
	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(br.UpdateScripts).SetText("").SetIcon(icons.Code)
	})
	scr := maps.Keys(br.Scripts)
	slices.Sort(scr)
	for _, s := range scr {
		lbl := TrimOrderPrefix(s)
		core.AddAt(p, lbl, func(w *core.Button) {
			w.SetText(lbl).SetIcon(icons.RunCircle).
				SetTooltip("Run script").
				OnClick(func(e events.Event) {
					br.RunScript(s)
				})
		})
	}
}

// TrimOrderPrefix trims any optional #- prefix from given string,
// used for ordering items by name.
func TrimOrderPrefix(s string) string {
	i := strings.Index(s, "-")
	if i < 0 {
		return s
	}
	ds := s[:i]
	if _, err := strconv.Atoi(ds); err != nil {
		return s
	}
	return s[i+1:]
}
