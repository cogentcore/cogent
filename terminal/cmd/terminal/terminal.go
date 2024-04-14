// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

//go:generate core generate

import (
	"cogentcore.org/cogent/terminal"
	"cogentcore.org/core/cli"
	"cogentcore.org/core/core"
)

type config struct { //types:add
	// Command is the command to run terminal on
	Command string `posarg:"0" required:"-" default:"ls"`
}

func main() {
	opts := cli.DefaultOptions("Terminal", "Terminal provides a terminal with support for the generation of GUIs and interactive CLIs for any existing command line tools.")
	cli.Run(opts, &config{}, &cli.Cmd[*config]{
		Func: run,
		Root: true,
	})
}

func run(c *config) error {
	b := core.NewBody("Cogent Terminal")
	cmd := terminal.NewCmd(c.Command)
	err := cmd.Parse()
	if err != nil {
		return err
	}
	app := terminal.NewApp(b).SetCmd(cmd)
	b.AddAppBar(app.AppBar)
	b.RunMainWindow()
	return nil
}
