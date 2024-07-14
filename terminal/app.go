// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package terminal

//go:generate core generate -add-types

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/shell/interpreter"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/tree"
	"github.com/cogentcore/yaegi/interp"
	"github.com/robert-nix/ansihtml"
)

// App is a GUI view of a terminal command.
type App struct {
	core.Frame

	// Cmd is the root command associated with this app.
	Cmd *Cmd

	// CurCmd is the current root command being typed in.
	CurCmd string

	// Dir is the current directory of the app.
	Dir string
}

var _ tree.Node = (*App)(nil)

func (a *App) Init() {
	a.Frame.Init()
	a.Dir = errors.Log1(os.Getwd())

	tree.AddChild(a, func(w *core.Form) {
		st := StructForFlags(a.Cmd.Flags)
		w.SetStruct(st)
	})

	tree.AddChildAt(a, "splits", func(w *core.Splits) {
		w.SetSplits(0.8, 0.2)
		w.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
		})

		tree.AddChildAt(w, "commands", func(w *core.Frame) {
			w.Styler(func(s *styles.Style) {
				s.Wrap = true
				s.Align.Content = styles.End
			})
		})

		tree.AddChildAt(w, "editor-frame", func(w *core.Frame) {
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
			})
			tree.AddChildAt(w, "dir", func(w *core.Text) {
				w.SetText(a.Dir)
			})
			tree.AddChild(w, func(w *texteditor.Editor) {
				w.Buffer.SetLanguage("go")
				w.Buffer.Options.LineNumbers = false

				w.OnKeyChord(func(e events.Event) {
					kf := keymap.Of(e.KeyChord())
					if kf == keymap.Enter && e.Modifiers() == 0 {
						e.SetHandled()
						txt := w.Buffer.String()
						w.Buffer.SetString("")
						cmds := a.FindPath("splits/commands").(*core.Frame)
						dir := a.FindPath("splits/editor-frame/dir").(*core.Text)
						errors.Log(a.RunCmd(txt, cmds, dir))
						return
					}
				})
			})
		})
	})
}

func (a *App) MakeToolbar(p *tree.Plan) {
	for _, cmd := range a.Cmd.Cmds {
		cmd := cmd
		fields := strings.Fields(cmd.Cmd)
		text := strcase.ToSentence(strings.Join(fields[1:], " "))
		tree.AddAt(p, text, func(w *core.Button) {
			w.SetText(text).SetTooltip(cmd.Doc)
			w.OnClick(func(e events.Event) {
				d := core.NewBody().AddTitle(text).AddText(cmd.Doc)
				st := StructForFlags(cmd.Flags)
				core.NewForm(d).SetStruct(st)
				d.AddBottomBar(func(parent core.Widget) {
					d.AddCancel(parent)
					d.AddOK(parent).SetText(text).OnClick(func(e events.Event) {
						errors.Log(exec.Verbose().Run(fields[0], fields[1:]...))
					})
				})
				d.RunFullDialog(w)
			})
		})
	}
}

// RunCmd runs the given command in the context of the given commands frame
// and current directory text.
func (a *App) RunCmd(cmd string, cmds *core.Frame, dir *core.Text) error {
	// ctx, cancel := context.WithCancel(context.Background())

	cfr := core.NewFrame(cmds)
	cfr.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
		s.Direction = styles.Column
		s.Border.Radius = styles.BorderRadiusLarge
		s.Background = colors.Scheme.SurfaceContainer
	})
	tr := core.NewFrame(cfr)
	tr.Styler(func(s *styles.Style) {
		s.Align.Items = styles.Center
		s.Padding.Set(units.Dp(8)).SetBottom(units.Zero())
	})
	core.NewText(tr).SetType(core.TextTitleLarge).SetText(cmd).Styler(func(s *styles.Style) {
		s.SetMono(true)
		s.Grow.Set(1, 0)
	})
	core.NewButton(tr).SetType(core.ButtonAction).SetIcon(icons.Close).OnClick(func(e events.Event) {
		// cancel()
		fmt.Println("canceled")
	})

	// output and input readers and writers
	or, ow := io.Pipe()
	ir, iw := io.Pipe()
	var ib []byte

	buf := texteditor.NewBuffer()
	buf.NewBuffer(0)
	buf.Options.LineNumbers = false

	ed := texteditor.NewEditor(cfr).SetBuffer(buf)
	ed.Styler(func(s *styles.Style) {
		s.Min.Set(units.Em(30), units.Em(10))
		s.Background = cfr.Styles.Background
	})
	ed.OnKeyChord(func(e events.Event) {
		kc := e.KeyChord()
		kf := keymap.Of(kc)

		fmt.Println(kc, kf)

		switch kf {
		case keymap.Enter:
			iw.Write(ib)
			iw.Write([]byte{'\n'})
			ib = nil
		case keymap.Backspace:
			if len(ib) > 0 {
				ib = slices.Delete(ib, len(ib)-1, len(ib))
			}
		default:
			ib = append(ib, kc...)
		}

	})

	ob := &texteditor.OutputBuffer{}
	ob.SetOutput(or).SetBuffer(buf).SetMarkupFunc(func(line []byte) []byte {
		return ansihtml.ConvertToHTML(line)
	})
	go func() {
		ob.MonitorOutput()
	}()

	cmds.Update()

	in := interpreter.NewInterpreter(interp.Options{Stdin: ir, Stdout: ow, Stderr: ow})
	go in.Eval(cmd)
	return nil
}

// StructForFlags returns a new struct object for the given flags.
func StructForFlags(flags []*Flag) any {
	sfs := make([]reflect.StructField, len(flags))

	used := map[string]bool{}
	for i, flag := range flags {
		sf := reflect.StructField{}
		sf.Name = strings.Trim(flag.Name, "-[]")
		sf.Name = strcase.ToCamel(sf.Name)

		// TODO(kai/terminal): better type determination
		if flag.Type == "bool" {
			sf.Type = reflect.TypeOf(false)
		} else if flag.Type == "int" {
			sf.Type = reflect.TypeOf(0)
		} else if flag.Type == "float" || flag.Type == "float32" || flag.Type == "float64" || flag.Type == "number" {
			sf.Type = reflect.TypeOf(0.0)
		} else {
			sf.Type = reflect.TypeOf("")
		}

		sf.Tag = reflect.StructTag(`desc:"` + flag.Doc + `"`)

		if used[sf.Name] {
			// TODO(kai/terminal): consider better approach to unique names
			nm := sf.Name + "1"
			for i := 2; used[nm]; i++ {
				nm = sf.Name + strconv.Itoa(i)
			}
			sf.Name = nm
		}
		used[sf.Name] = true
		sfs[i] = sf
	}
	stt := reflect.StructOf(sfs)
	st := reflect.New(stt)
	return st.Interface()
}
