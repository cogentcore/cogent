// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package numshell

import (
	"errors"
	"io"
	"log"
	"os"

	"cogentcore.org/cogent/numbers/imports"
	"cogentcore.org/core/shell"
	"cogentcore.org/core/shell/interpreter"
	"github.com/ergochat/readline"
	"github.com/traefik/yaegi/interp"
)

// NumbersInterpreter is the interpreter for Numbers app
type Interpreter struct {
	interpreter.Interpreter
}

func NewInterpreter(options interp.Options) *Interpreter {
	in := &Interpreter{}
	in.Interpreter = *interpreter.NewInterpreter(options)
	in.InitInterp()
	return in
}

// InitInterp initializes the interpreter with symbols
func (in *Interpreter) InitInterp() {
	in.Interp.Use(imports.Symbols)
}

// RunScript runs given script code on the interpreter
func (in *Interpreter) RunScript(script string) {
	in.Interp.Eval(script)
}

// Interactive runs an interactive shell that allows the user to input.
// Does not return until the interactive session ends.
func (in *Interpreter) Interactive() error {
	in.Interp.ImportUsed()
	in.RunConfig()
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
