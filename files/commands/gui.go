// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"fmt"
	"html"
	"image"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
)

// RunFunc runs the given command line, such as in a terminal.
type RunFunc func(cmdline string)

// Changed saves the commands, and updates the command bars and the editor
// of the commands, other than the given editor, in which the change was
// made (or none if it is nil).
func (cs *Commands) Changed(editor *core.TreeTable) {
	errors.Log(cs.Save())
	cs.views.Lock()
	cs.views.bars = slices.DeleteFunc(cs.views.bars, func(tb *core.Toolbar) bool { return tb.This == nil })
	var widgets []core.Widget
	for _, tb := range cs.views.bars {
		widgets = append(widgets, tb)
	}
	if tt := cs.views.editor; tt != nil && tt.This != nil && tt != editor {
		widgets = append(widgets, tt)
	}
	cs.views.Unlock()
	for _, w := range widgets {
		// (not with the lock of this window, which another window may be
		// waiting on)
		go func() {
			wb := w.AsWidget()
			wb.AsyncLock()
			defer wb.AsyncUnlock()
			if wb.This != nil {
				wb.Update()
			}
		}()
	}
}

// Bar adds the command bar to the given parent: a button for each
// top-level entry, which shows the form of its command (see
// [Commands.RunDialog]) to run with the given function, or the menu of
// the entries of a category, and which can be dragged to another place in
// the bar, followed by a button to edit the commands.
func (cs *Commands) Bar(parent core.Widget, run RunFunc) *core.Toolbar {
	tb := core.NewToolbar(parent)
	tb.SetName("commands")
	tb.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
	})
	tb.MoveFunc = func(from, to int) {
		ens := cs.Entries
		if from < 0 || from >= len(ens) || to < 0 || to >= len(ens) {
			return
		}
		en := ens[from]
		ens = slices.Delete(ens, from, from+1)
		cs.Entries = slices.Insert(ens, to, en)
		cs.Changed(nil)
	}
	tb.Maker(func(p *tree.Plan) {
		for _, en := range cs.Entries {
			tree.AddAt(p, fmt.Sprintf("entry-%p", en), func(w *core.Button) {
				w.SetType(core.ButtonAction)
				w.OnClick(func(e events.Event) {
					if !en.IsCategory() {
						cs.RunDialog(w, en.Command, run)
					}
				})
				w.Updater(func() { cs.configButton(w, en, run) })
			})
		}
		tree.AddAt(p, "edit-commands", func(w *core.Button) {
			w.SetType(core.ButtonAction).SetIcon(icons.Edit).SetTooltip("Edit the commands and their categories")
			w.OnClick(func(e events.Event) { cs.Edit(w) })
		})
	})
	cs.views.Lock()
	cs.views.bars = append(cs.views.bars, tb)
	cs.views.Unlock()
	return tb
}

// configButton configures the given button of the command bar for the
// given entry: a category has the menu of its entries.
func (cs *Commands) configButton(w *core.Button, en *Entry, run RunFunc) {
	w.SetText(en.Label())
	if en.IsCategory() {
		w.SetIcon(icons.Folder).SetTooltip(en.Label())
		w.SetMenu(func(m *core.Scene, _ image.Point) { cs.menu(m, en.Children, run) })
		return
	}
	w.SetIcon(icons.Terminal).SetTooltip(cs.tooltip(en.Command))
	w.Menu = nil
}

// tooltip returns the tooltip of the given command: the command and its
// documentation, if it has been parsed.
func (cs *Commands) tooltip(cmd string) string {
	if cm := cs.cached(cmd); cm != nil && cm.Doc != "" {
		return html.EscapeString(cmd + ": " + cm.Doc)
	}
	return cmd
}

// menu adds the buttons of the given entries to the given menu: a category
// has a submenu of its entries.
func (cs *Commands) menu(m *core.Scene, ens []*Entry, run RunFunc) {
	if len(ens) == 0 {
		core.NewText(m).SetText("(no commands)")
		return
	}
	for _, en := range ens {
		b := core.NewButton(m).SetText(en.Label())
		if en.IsCategory() {
			b.SetIcon(icons.Folder)
			b.SetMenu(func(sm *core.Scene, _ image.Point) { cs.menu(sm, en.Children, run) })
			continue
		}
		b.SetIcon(icons.Terminal).SetTooltip(cs.tooltip(en.Command))
		b.OnClick(func(e events.Event) { cs.RunDialog(b, en.Command, run) })
	}
}

// Edit opens the editor of the commands, a [core.TreeTable] of them in a new
// window, in which they are saved as they are edited.
func (cs *Commands) Edit(ctx core.Widget) {
	if core.RecycleMainWindow(&cs.Entries) {
		return
	}
	b := core.NewBody("Commands")
	b.SetData(&cs.Entries)
	tt := core.NewTreeTable(b)
	tt.SetDragMoves(true)
	tt.SetItems(&cs.Entries)
	tt.OnChange(func(e events.Event) { cs.Changed(tt) })
	cs.views.Lock()
	cs.views.editor = tt
	cs.views.Unlock()
	b.OnClose(func(e events.Event) {
		cs.views.Lock()
		if cs.views.editor == tt {
			cs.views.editor = nil
		}
		cs.views.Unlock()
		go cs.ParseAll() // for new commands
	})
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(func(p *tree.Plan) {
			tree.Add(p, func(w *core.Button) {
				w.SetText("Add category").SetIcon(icons.CreateNewFolder).SetTooltip("Add a category of commands at the end")
				w.OnClick(func(e events.Event) {
					cs.Entries = append(cs.Entries, &Entry{Name: "New category"})
					tt.Update()
					cs.Changed(tt)
				})
			})
			tree.Add(p, func(w *core.Button) {
				w.SetText("Add command").SetIcon(icons.Add).SetTooltip("Add a command at the end, such as git log, to be entered")
				w.OnClick(func(e events.Event) {
					cs.Entries = append(cs.Entries, &Entry{Name: "New command"})
					tt.Update()
					cs.Changed(tt)
				})
			})
			tree.Add(p, func(w *core.Button) {
				w.SetText("Reset").SetIcon(icons.Reset).SetTooltip("Replace the commands with the standard ones")
				w.OnClick(func(e events.Event) {
					d := core.NewBody("Reset commands")
					core.NewText(d).SetType(core.TextSupporting).SetText("Replace all of the commands and categories with the standard ones?")
					d.AddBottomBar(func(bar *core.Frame) {
						d.AddCancel(bar)
						d.AddOK(bar).SetText("Reset").OnClick(func(e events.Event) {
							cs.Entries = Standard()
							tt.Update()
							cs.Changed(tt)
						})
					})
					d.RunDialog(w)
				})
			})
		})
	})
	b.RunWindow()
}

// RunDialog shows the form of the flags and arguments of the given command,
// such as "git commit", once its help is parsed, which runs the resulting
// command line with the given function.
func (cs *Commands) RunDialog(ctx core.Widget, cmd string, run RunFunc) {
	if cm := cs.cached(cmd); cm != nil {
		cs.runDialog(ctx, cm, run)
		return
	}
	core.MessageSnackbar(ctx, "Reading the help of "+cmd+"...")
	go func() {
		cm, err := cs.Cmd(cmd, false)
		ctx.AsWidget().AsyncLock()
		defer ctx.AsWidget().AsyncUnlock()
		if err != nil {
			core.ErrorSnackbar(ctx, err)
			return
		}
		cs.runDialog(ctx, cm, run)
	}()
}

// runDialog shows the form of the given command (see [Commands.RunDialog]).
func (cs *Commands) runDialog(ctx core.Widget, cm *Cmd, run RunFunc) {
	cs.NewDialog(ctx, cm, run).RunWindowDialog(ctx)
}

// NewDialog returns the body of the dialog of the form of the given command
// (see [Commands.RunDialog]), in the context of the given widget, which
// runs the command line with the given function.
func (cs *Commands) NewDialog(ctx core.Widget, cm *Cmd, run RunFunc) *core.Body {
	d := core.NewBody(cm.Cmd)
	if cm.Doc != "" {
		core.NewText(d).SetType(core.TextSupporting).SetText(html.EscapeString(cm.Doc))
	}
	if cm.Usage != "" {
		core.NewText(d).SetType(core.TextSupporting).SetText(html.EscapeString(cm.Usage)).Styler(func(s *styles.Style) {
			s.Font.Family = rich.Monospace
		})
	}
	values := cm.FormStruct()
	line := core.NewText(d)
	line.Styler(func(s *styles.Style) {
		s.Font.Family = rich.Monospace
		s.Grow.Set(1, 0)
	})
	line.Updater(func() { line.SetText(html.EscapeString(cm.CommandLine(values))) })
	fm := core.NewForm(d).SetStruct(values)
	fm.OnChange(func(e events.Event) { line.Update() })
	fm.OnInput(func(e events.Event) { line.Update() })
	d.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(func(p *tree.Plan) {
			if len(cm.Cmds) > 0 {
				tree.Add(p, func(w *core.Button) {
					w.SetText("Subcommands").SetIcon(icons.Menu)
					w.SetMenu(func(m *core.Scene, _ image.Point) {
						for _, sub := range cm.Cmds {
							core.NewButton(m).SetText(sub.Cmd).SetTooltip(html.EscapeString(sub.Doc)).OnClick(func(e events.Event) {
								d.Close()
								cs.RunDialog(ctx, sub.Cmd, run)
							})
						}
					})
				})
			}
			tree.Add(p, func(w *core.Button) {
				w.SetText("Reread help").SetIcon(icons.Refresh).SetTooltip("Read the help of the command again, as for a new version (from " + cm.Source + ")")
				w.OnClick(func(e events.Event) {
					go func() {
						ncm, err := cs.Cmd(cm.Cmd, true)
						ctx.AsWidget().AsyncLock()
						defer ctx.AsWidget().AsyncUnlock()
						if err != nil {
							core.ErrorSnackbar(ctx, err)
							return
						}
						d.Close()
						cs.runDialog(ctx, ncm, run)
					}()
				})
			})
		})
	})
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Run").SetIcon(icons.PlayArrow).OnClick(func(e events.Event) {
			run(cm.CommandLine(values))
		})
	})
	return d
}

// argsField is the name of the field of the arguments in the
// [Cmd.FormStruct].
const argsField = "Arguments"

// FormStruct returns a pointer to a new struct for a form of the flags of
// the command: a field for the other arguments, and a bool or string field
// for each flag, labeled with its names, with its documentation.
func (cm *Cmd) FormStruct() any {
	fields := []reflect.StructField{{
		Name: argsField, Type: reflect.TypeFor[string](),
		Tag: `immediate:"+" doc:"The other arguments of the command, such as files and paths, as typed in a shell"`,
	}}
	used := map[string]bool{argsField: true}
	for _, fl := range cm.Flags {
		name := fieldName(fl.Name)
		for i := 2; used[name]; i++ {
			name = fieldName(fl.Name) + strconv.Itoa(i)
		}
		used[name] = true
		typ := reflect.TypeFor[string]()
		if fl.IsBool() {
			typ = reflect.TypeFor[bool]()
		}
		// (the label and doc are shown as markup)
		tag := `immediate:"+" label:` + strconv.Quote(html.EscapeString(fl.Label())) + ` doc:` + strconv.Quote(html.EscapeString(fl.Doc))
		fields = append(fields, reflect.StructField{Name: name, Type: typ, Tag: reflect.StructTag(tag)})
	}
	return reflect.New(reflect.StructOf(fields)).Interface()
}

// fieldName returns an exported Go field name for the given flag name.
func fieldName(flag string) string {
	name := strcase.ToCamel(strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '-'
	}, strings.TrimLeft(flag, "-")))
	if name == "" || !unicode.IsUpper(rune(name[0])) {
		name = "F" + name
	}
	return name
}

// CommandLine returns the command line for the given values of the form of
// the command made by [Cmd.FormStruct]: the command, the flags that are on
// or have values, and the other arguments.
func (cm *Cmd) CommandLine(values any) string {
	v := reflect.ValueOf(values).Elem()
	words := []string{cm.Cmd}
	for i, fl := range cm.Flags {
		f := v.Field(i + 1)
		switch f.Kind() {
		case reflect.Bool:
			if f.Bool() {
				words = append(words, fl.Arg(""))
			}
		case reflect.String:
			if s := f.String(); s != "" {
				words = append(words, fl.Arg(s))
			}
		}
	}
	if args := strings.TrimSpace(v.Field(0).String()); args != "" {
		words = append(words, args)
	}
	return strings.Join(words, " ")
}
