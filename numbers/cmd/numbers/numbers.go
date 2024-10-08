// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command numbers is an interactive cli for
// Numbers data management, analysis and math system.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/cogent/numbers/databrowser"
	"cogentcore.org/cogent/numbers/random"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/cli"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/shell"
	"cogentcore.org/core/shell/interpreter"
	"github.com/cogentcore/yaegi/interp"
)

//go:generate core generate -add-types -add-funcs

// Config is the configuration information for the numbers cli.
type Config struct {

	// The input file to run/compile.
	// If this is provided as the first argument,
	// then the program will exit after running,
	// unless the Interactive mode is flagged.
	Input string `posarg:"0" required:"-"`

	// the Go file to output the transpiled Input file to,
	// as an optional second argument in build mode.
	// It defaults to the input file with .numbers changed to .go.
	Output string `cmd:"build" posarg:"1" required:"-"`

	// an optional expression to evaluate, which can be used
	// in addition to the Input file to run, to execute commands
	// defined within that file for example, or as a command to run
	// prior to starting interactive mode if no Input is specified.
	Expr string `flag:"e,expr"`

	// runs the interactive command line after processing an Input file.
	// Interactive mode is the default for all cases except when
	// an Input file is specified, and is not available
	// if an Output file is specified for transpiling.
	Interactive bool `flag:"i,interactive"`
}

func main() { //types:skip
	opts := cli.DefaultOptions("numbers", "Cogent Numbers, an interactive tool for data management, analysis and math.")
	cli.Run(opts, &Config{}, Run, Build)
}

// Run runs the specified numbers file. If no file is specified,
// it runs an interactive shell that allows the user to input numbers.
func Run(c *Config) error { //cli:cmd -root
	if c.Input == "" {
		return Interactive(c)
	}
	code := ""
	if errors.Log1(fsx.FileExists(c.Input)) {
		b, err := os.ReadFile(c.Input)
		if err != nil && c.Expr == "" {
			return err
		}
		code = string(b)
	}
	if c.Expr != "" {
		if code != "" {
			code += "\n"
		}
		code += c.Expr + "\n"
	}
	in := interpreter.NewInterpreter(interp.Options{})
	in.Config()
	_, _, err := in.Eval(code)
	if err == nil {
		err = in.Shell.DepthError()
	}
	if c.Interactive {
		return Interactive(c)
	}
	return err
}

// Interactive runs an interactive shell that allows the user to input numbers.
func Interactive(c *Config) error {
	b := core.NewBody("Cogent Numbers")
	core.NewText(b).SetText("Cogent Numbers").SetType(core.TextHeadlineSmall)

	ts := core.NewTabs(b)

	db, _ := ts.NewTab("Data browser")
	core.NewText(db).SetType(core.TextTitleLarge).SetText("Use your terminal to enter data browser commands.")

	rt, _ := ts.NewTab("Random")
	tb := core.NewToolbar(rt)
	rd := random.NewRandom(rt)
	tb.Maker(rd.MakeToolbar)

	in := interpreter.NewInterpreter(interp.Options{})
	in.HistFile = "~/.numbers-history"
	in.Interp.Use(databrowser.Symbols)
	in.Config()

	b.OnShow(func(e events.Event) {
		go func() {
			if c.Expr != "" {
				in.Eval(c.Expr)
			}
			in.Interactive()
		}()
	})
	b.RunMainWindow()
	return nil
}

// Build builds the specified input numbers file to the specified output Go file.
func Build(c *Config) error {
	if c.Input == "" {
		return fmt.Errorf("need input file")
	}
	if c.Output == "" {
		c.Output = strings.TrimSuffix(c.Input, filepath.Ext(c.Input)) + ".go"
	}
	return shell.NewShell().TranspileFile(c.Input, c.Output)
}
