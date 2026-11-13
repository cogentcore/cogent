// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"cmp"
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
	cs.updateViews(editor)
}

// updateViews updates the command bars and the editor of the commands,
// other than the given editor (or none if it is nil).
func (cs *Commands) updateViews(editor *core.TreeTable) {
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
// the entries of a category, or of the subcommands of a command, and which
// can be dragged to another place in the bar, followed by a button to edit
// the commands.
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
	// the changes of other apps are shown when the bar is shown, or the
	// mouse enters it (the menus are made from the file as it is)
	tb.OnShow(func(e events.Event) { go cs.reload() })
	tb.On(events.MouseEnter, func(e events.Event) { go cs.reload() })
	tb.Maker(func(p *tree.Plan) {
		for _, en := range cs.Entries {
			tree.AddAt(p, fmt.Sprintf("entry-%p", en), func(w *core.Button) {
				w.SetType(core.ButtonAction)
				w.OnClick(func(e events.Event) {
					if w.Menu == nil && !en.IsCategory() {
						cs.RunDialog(w, en.Command, run)
					}
				})
				w.Updater(func() { cs.configEntry(w, en, run) })
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

// configEntry configures the given button, of the command bar or of a
// menu, for the given entry: a category has the menu of its entries, and
// a command that of its subcommands (see [Commands.configCommand]).
func (cs *Commands) configEntry(w *core.Button, en *Entry, run RunFunc) {
	w.SetText(en.Label())
	if en.IsCategory() {
		w.SetIcon(icons.Folder).SetTooltip(en.Label())
		w.SetMenu(func(m *core.Scene, _ image.Point) { cs.menu(m, en.Children, run) })
		return
	}
	cs.configCommand(w, en.Command, run)
}

// configCommand configures the given button for the given command or
// subcommand: a command with subcommands, such as git, has the menu of
// them, as does a subcommand with subcommands, such as git stash, and
// otherwise clicking it shows the form of the command (see
// [Commands.RunDialog], which the caller does on a click without a menu).
// A command is parsed when it is first clicked, if it has not been parsed
// yet, and its menu shown if it has subcommands.
func (cs *Commands) configCommand(w *core.Button, cmd string, run RunFunc) {
	w.SetIcon(icons.Terminal)
	w.Menu = nil
	root, name := splitCmd(cmd)
	tip := cmd
	if cm := cs.cached(root); cm != nil {
		if name == "" && cm.HasSubCmds() {
			w.SetTooltip(tooltip(root, cm.Doc))
			w.SetMenu(func(m *core.Scene, _ image.Point) { cs.subCmdsMenu(m, root, run) })
			return
		}
		if sc := cm.SubCmd(cmp.Or(name, MainName)); sc != nil {
			tip = tooltip(cmd, cmp.Or(sc.Doc, cm.Doc))
			if len(sc.Cmds) > 0 {
				w.SetTooltip(tip)
				w.SetMenu(func(m *core.Scene, _ image.Point) { cs.nestedMenu(m, sc, run) })
				return
			}
		}
	}
	w.SetTooltip(tip)
}

// tooltip returns the tooltip of the given command with the given
// documentation, as markup.
func tooltip(cmd, doc string) string {
	if doc == "" {
		return html.EscapeString(cmd)
	}
	return html.EscapeString(cmd + ": " + doc)
}

// commandButton adds a button for the given command or subcommand to the
// given menu, with the given text (see [Commands.configCommand]).
func (cs *Commands) commandButton(m *core.Scene, text, cmd string, run RunFunc) {
	b := core.NewButton(m).SetText(text)
	cs.configCommand(b, cmd, run)
	b.OnClick(func(e events.Event) {
		if b.Menu == nil {
			cs.RunDialog(b, cmd, run)
		}
	})
}

// menu adds the buttons of the given entries to the given menu: a category
// has a submenu of its entries, and a command with subcommands a submenu
// of them.
func (cs *Commands) menu(m *core.Scene, ens []*Entry, run RunFunc) {
	if len(ens) == 0 {
		core.NewText(m).SetText("(no commands)")
		return
	}
	for _, en := range ens {
		if en.IsCategory() {
			b := core.NewButton(m)
			cs.configEntry(b, en, run)
			continue
		}
		cs.commandButton(m, en.Label(), en.Command, run)
	}
}

// subCmdsMenu adds the buttons of the subcommands of the given command,
// such as git, to the given menu: those of the command, not the nested
// ones (see [Commands.nestedMenu]). The subcommands of the first group
// (see [SubCmd.Group]) are in the menu, and those of the others in
// submenus after them.
func (cs *Commands) subCmdsMenu(m *core.Scene, root string, run RunFunc) {
	cm := cs.cached(root)
	if cm == nil {
		return
	}
	var groups []string
	byGroup := map[string][]*SubCmd{}
	for _, sc := range cm.SubCmds {
		if strings.Contains(sc.Name, " ") {
			continue // nested
		}
		if _, ok := byGroup[sc.Group]; !ok {
			groups = append(groups, sc.Group)
		}
		byGroup[sc.Group] = append(byGroup[sc.Group], sc)
	}
	add := func(m *core.Scene, scs []*SubCmd) {
		for _, sc := range scs {
			cs.commandButton(m, sc.Label(), sc.Cmd, run)
		}
	}
	for i, g := range groups {
		if i == 0 {
			add(m, byGroup[g])
			continue
		}
		if i == 1 {
			core.NewSeparator(m)
		}
		b := core.NewButton(m).SetText(cmp.Or(g, "Other")).SetIcon(icons.Folder)
		b.SetMenu(func(sm *core.Scene, _ image.Point) { add(sm, byGroup[g]) })
	}
}

// nestedMenu adds the buttons of the given subcommand, such as git stash,
// and of its subcommands, such as git stash list, to the given menu.
func (cs *Commands) nestedMenu(m *core.Scene, sc *SubCmd, run RunFunc) {
	b := core.NewButton(m).SetText(sc.Label()).SetIcon(icons.Terminal).SetTooltip(tooltip(sc.Cmd, sc.Doc))
	b.OnClick(func(e events.Event) { cs.RunDialog(b, sc.Cmd, run) })
	core.NewSeparator(m)
	for _, sub := range sc.Cmds {
		cs.commandButton(m, label(sub.Cmd), sub.Cmd, run)
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

// RunDialog shows the form of the flags and arguments of the given
// subcommand, such as "git commit", or command without subcommands, such
// as "ls", once its help is parsed, which runs the resulting command line
// with the given function. For a command with subcommands, such as git, it
// shows the menu of its subcommands, as the menu of the given widget.
func (cs *Commands) RunDialog(ctx core.Widget, cmd string, run RunFunc) {
	if sc := cs.cachedSub(cmd); sc != nil {
		cs.runDialog(ctx, sc, run)
		return
	}
	core.MessageSnackbar(ctx, "Reading the help of "+cmd+"...")
	go func() {
		root, name := splitCmd(cmd)
		cm, err := cs.Cmd(root, false)
		var sc *SubCmd
		if err == nil && !(name == "" && cm.HasSubCmds()) {
			sc, err = cs.SubCmd(cmd, false)
		}
		cs.updateViews(nil)
		ctx.AsWidget().AsyncLock()
		defer ctx.AsWidget().AsyncUnlock()
		switch {
		case err != nil:
			core.ErrorSnackbar(ctx, err)
		case sc == nil: // the menu of the subcommands
			core.NewMenu(func(m *core.Scene, _ image.Point) { cs.subCmdsMenu(m, root, run) }, ctx, ctx.AsWidget().ContextMenuPos(nil)).Run()
		default:
			cs.runDialog(ctx, sc, run)
		}
	}()
}

// runDialog shows the form of the given subcommand (see [Commands.RunDialog]).
func (cs *Commands) runDialog(ctx core.Widget, sc *SubCmd, run RunFunc) {
	cs.NewDialog(ctx, sc, run).RunWindowDialog(ctx)
}

// NewDialog returns the body of the dialog of the form of the given
// subcommand (see [Commands.RunDialog]), in the context of the given
// widget, which runs the command line with the given function. The form
// starts with the arguments last used for the subcommand, which are saved
// when it is run (see [Args]).
func (cs *Commands) NewDialog(ctx core.Widget, sc *SubCmd, run RunFunc) *core.Body {
	d := core.NewBody(sc.Cmd)
	if sc.Doc != "" {
		core.NewText(d).SetType(core.TextSupporting).SetText(html.EscapeString(sc.Doc))
	}
	if sc.Usage != "" {
		core.NewText(d).SetType(core.TextSupporting).SetText(html.EscapeString(sc.Usage)).Styler(func(s *styles.Style) {
			s.Font.Family = rich.Monospace
		})
	}
	values := sc.FormStruct()
	if cs.Args != nil {
		cs.Args.SetValues(sc, values)
	}
	line := core.NewText(d)
	line.Styler(func(s *styles.Style) {
		s.Font.Family = rich.Monospace
		s.Grow.Set(1, 0)
	})
	line.Updater(func() { line.SetText(html.EscapeString(sc.CommandLine(values))) })
	fm := core.NewForm(d).SetStruct(values)
	fm.OnChange(func(e events.Event) { line.Update() })
	fm.OnInput(func(e events.Event) { line.Update() })
	d.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(func(p *tree.Plan) {
			if len(sc.Cmds) > 0 {
				tree.Add(p, func(w *core.Button) {
					w.SetText("Subcommands").SetIcon(icons.Menu)
					w.SetMenu(func(m *core.Scene, _ image.Point) {
						for _, sub := range sc.Cmds {
							core.NewButton(m).SetText(sub.Cmd).SetTooltip(html.EscapeString(sub.Doc)).OnClick(func(e events.Event) {
								d.Close()
								cs.RunDialog(ctx, sub.Cmd, run)
							})
						}
					})
				})
			}
			tree.Add(p, func(w *core.Button) {
				w.SetText("Clear").SetIcon(icons.ClearAll).SetTooltip("Clear the flags and arguments")
				w.OnClick(func(e events.Event) {
					reflect.ValueOf(values).Elem().SetZero()
					fm.Update()
					line.Update()
				})
			})
			tree.Add(p, func(w *core.Button) {
				w.SetText("Reread help").SetIcon(icons.Refresh).SetTooltip(html.EscapeString("Read the help of the command again, as for a new version (from " + sc.Source + ")"))
				w.OnClick(func(e events.Event) {
					go func() {
						nsc, err := cs.SubCmd(sc.Cmd, true)
						ctx.AsWidget().AsyncLock()
						defer ctx.AsWidget().AsyncUnlock()
						if err != nil {
							core.ErrorSnackbar(ctx, err)
							return
						}
						d.Close()
						cs.runDialog(ctx, nsc, run)
					}()
				})
			})
		})
	})
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Run").SetIcon(icons.PlayArrow).OnClick(func(e events.Event) {
			if cs.Args != nil {
				errors.Log(cs.Args.Set(sc, values))
			}
			run(sc.CommandLine(values))
		})
	})
	return d
}

// argsField is the name of the field of the arguments in the
// [SubCmd.FormStruct].
const argsField = "Arguments"

// FormStruct returns a pointer to a new struct for a form of the flags of
// the command: a field for the other arguments, and a bool or string field
// for each flag, labeled with its names, with its documentation.
func (cm *SubCmd) FormStruct() any {
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
// the command made by [SubCmd.FormStruct]: the command, the flags that are on
// or have values, and the other arguments.
func (cm *SubCmd) CommandLine(values any) string {
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
