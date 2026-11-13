// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"image"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCommit returns the parsed git commit subcommand.
func testCommit() *SubCmd {
	sc := &SubCmd{Name: "commit", Cmd: "git commit", Source: "git commit -h", Doc: "Record changes to the repository"}
	sc.SetFromHelp(gitCommitHelp)
	return sc
}

// TestMacroValues tests the form of a macro: its values and prompts, and
// the command line of a macro with prompted values.
func TestMacroValues(t *testing.T) {
	sc := testCommit()
	values := sc.FormStruct()
	v := reflect.ValueOf(values).Elem()
	v.FieldByName("Verbose").SetBool(true)
	v.FieldByName("Message").SetString("default message")
	mc := &Macro{SubCmd: "commit", Values: sc.ArgsOf(values)}

	mv := sc.MacroStruct()
	sc.SetMacroValues(mc, mv)
	m := reflect.ValueOf(mv).Elem()
	assert.Equal(t, "default message", m.FieldByName("Message").FieldByName("Value").String(), "carried over")
	m.FieldByName("Name").SetString(" wip ")
	m.FieldByName("Message").FieldByName("Prompt").SetBool(true)
	m.FieldByName("Arguments").FieldByName("Value").SetString("-- .")
	field, _ := reflect.TypeOf(mv).Elem().FieldByName("Message")
	assert.Equal(t, "-m, --message &lt;message&gt;", field.Tag.Get("label"))
	pf, _ := field.Type.FieldByName("Prompt")
	assert.Equal(t, "if checked, this will be prompted for when the macro runs", pf.Tag.Get("doc"))

	mc = sc.MacroOf(mv)
	assert.Equal(t, "wip", mc.Name)
	assert.Equal(t, []string{"--message"}, mc.Prompts)
	assert.Equal(t, "Git commit wip", mc.Label("git"))
	assert.Equal(t, "git commit --verbose --message 'default message' -- .", sc.ArgsCommandLine(mc.withPrompted(sc, nil, nil)))

	// the prompt has only the prompted arguments, with the defaults
	keys := mc.promptKeys()
	pv := sc.formStruct(keys)
	assert.Equal(t, 1, reflect.TypeOf(pv).Elem().NumField())
	mc.Values.SetValues(sc, pv)
	assert.Equal(t, "default message", reflect.ValueOf(pv).Elem().Field(0).String())
	reflect.ValueOf(pv).Elem().Field(0).SetString("the fix")
	assert.Equal(t, "git commit --verbose --message 'the fix' -- .", sc.ArgsCommandLine(mc.withPrompted(sc, sc.ArgsOf(pv), keys)))
	reflect.ValueOf(pv).Elem().Field(0).SetString("")
	assert.Equal(t, "git commit --verbose -- .", sc.ArgsCommandLine(mc.withPrompted(sc, sc.ArgsOf(pv), keys)))
}

// TestMacros tests saving and deleting macros, which are kept when their
// command is parsed again, and in a file of another version, and are at
// the top of the menu of the command.
func TestMacros(t *testing.T) {
	file := filepath.Join(t.TempDir(), "commands.json")
	cs := testCommands(t, file)
	wip := &Macro{Name: "wip", SubCmd: "commit", Values: &SubCmdArgs{Flags: map[string]string{"--verbose": "true"}}, Prompts: []string{"--message"}}
	require.NoError(t, cs.SaveMacro("git", wip, nil))
	require.NoError(t, cs.SaveMacro("ls", &Macro{Name: "long", SubCmd: MainName, Values: &SubCmdArgs{Flags: map[string]string{"-l": "true"}}}, nil))
	renamed := *wip
	renamed.Name = "work"
	require.NoError(t, cs.SaveMacro("git", &renamed, wip))
	assert.Len(t, cs.cached("git").Macros, 1)
	assert.Equal(t, "work", cs.cached("git").Macros[0].Name)

	texts := func(m *core.Scene) []string {
		var ts []string
		for _, kid := range m.Children {
			if b, ok := kid.(*core.Button); ok {
				ts = append(ts, b.Text)
			}
		}
		return ts
	}
	m := core.NewScene()
	cs.subCmdsMenu(m, "git", nil)
	assert.Equal(t, "Git commit work", texts(m)[0])
	m = core.NewScene()
	cs.subCmdsMenu(m, "ls", nil)
	assert.Equal(t, []string{"Ls long", "Ls"}, texts(m), "ls has a menu with its macros")

	// another process, with a file of another version
	b, err := os.ReadFile(file)
	require.NoError(t, err)
	old := filepath.Join(t.TempDir(), "commands.json")
	require.Contains(t, string(b), `"Version": 8`)
	require.NoError(t, os.WriteFile(old, []byte(strings.Replace(string(b), `"Version": 8`, `"Version": 2`, 1)), 0o644))
	cs2 := &Commands{file: sharedFile{path: old}, Parsed: map[string]*Cmd{}}
	cs2.reload()
	require.NotNil(t, cs2.cached("git"))
	assert.False(t, cs2.cached("git").IsParsed(), "parsed again")
	assert.Equal(t, "work", cs2.cached("git").Macros[0].Name, "kept")

	require.NoError(t, cs.DeleteMacro("git", &renamed))
	assert.Empty(t, cs.cached("git").Macros)
}

// TestMacroDialogs renders the dialog of a new macro and the prompt of a
// macro.
func TestMacroDialogs(t *testing.T) {
	imagex.UpdateTestImagesFor(t)
	cs := &Commands{Parsed: map[string]*Cmd{}, Args: OpenArgs("")}
	sc := testCommit()
	mc := &Macro{Name: "wip", SubCmd: "commit", Values: &SubCmdArgs{Args: "-- .", Flags: map[string]string{"--verbose": "true", "--message": "wip"}}, Prompts: []string{"--message"}}
	d := cs.NewMacroDialog(nil, "git", sc, mc, nil, func(string) {})
	d.AssertRender(t, "macro")
	ran := ""
	d = cs.NewPromptDialog(nil, "git", sc, mc, func(cmdline string) { ran = cmdline })
	d.AssertRender(t, "macro-prompt", func() {
		fm := tree.ChildByType[*core.Form](d)
		require.NotNil(t, fm)
		reflect.ValueOf(fm.Struct).Elem().Field(0).SetString("the fix")
		fm.Update()
	})
	assert.Equal(t, "", ran)
}

// TestContext tests the context of the dialogs of the commands: the scene
// of the window of an item of a menu, which closes when it is clicked, and
// a command bar for an item of a closed menu (which crashed making a macro
// from the dialog of a subcommand in a menu).
func TestContext(t *testing.T) {
	imagex.UpdateTestImagesFor(t) // the image is not compared
	cs := &Commands{Parsed: map[string]*Cmd{}, Entries: Standard()}
	b := core.NewBody()
	tb := cs.Bar(b, nil)
	b.AssertRender(t, "context", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		st := core.NewMenu(func(m *core.Scene, pos image.Point) { core.NewButton(m).SetText("Item") }, tb, image.Point{})
		st.Run()
		item := st.Scene.Child(0).(core.Widget)
		assert.Equal(t, core.Widget(b.Scene), cs.context(item), "the scene of the window")
		assert.Equal(t, core.Widget(tb), cs.context(tb))
		st.ClosePopup()
		assert.Equal(t, core.Widget(tb), cs.context(item), "a command bar")
	})
}

// TestDialogFocus tests that the first field of the forms of the dialogs
// is in focus when they are shown: the first prompt of a macro, the name
// of a macro, and the arguments of a subcommand.
func TestDialogFocus(t *testing.T) {
	imagex.UpdateTestImagesFor(t) // the images are not compared
	cs := &Commands{Parsed: map[string]*Cmd{}, Args: OpenArgs("")}
	sc := testCommit()
	mc := &Macro{Name: "wip", SubCmd: "commit", Values: &SubCmdArgs{Flags: map[string]string{"--message": "wip"}}, Prompts: []string{"--file", "--message"}}
	for _, c := range []struct {
		name, field string
		d           *core.Body
	}{
		{"prompt", "File", cs.NewPromptDialog(nil, "git", sc, mc, func(string) {})},
		{"macro", "Name", cs.NewMacroDialog(nil, "git", sc, mc, nil, func(string) {})},
		{"form", "Arguments", cs.NewDialog(nil, sc, func(string) {})},
	} {
		c.d.AssertRender(t, "focus-"+c.name, func() {
			focused := ""
			for range 50 {
				c.d.AsyncLock()
				c.d.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
					if _, ok := cw.(*core.TextField); ok && cwb.StateIs(states.Focused) {
						focused = cwb.Name
					}
					return tree.Continue
				})
				c.d.AsyncUnlock()
				if focused != "" {
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
			assert.Equal(t, "value-"+c.field+"-string", focused, c.name)
		})
	}
}

// TestFieldWidths tests that the text fields of the forms of the dialogs,
// and of the inline forms of the arguments of macros, are wide, with no
// maximum width.
func TestFieldWidths(t *testing.T) {
	imagex.UpdateTestImagesFor(t)
	cs := &Commands{Parsed: map[string]*Cmd{}, Args: OpenArgs("")}
	sc := testCommit()
	mc := &Macro{Name: "wip", SubCmd: "commit", Prompts: []string{"--message"}}
	for name, d := range map[string]*core.Body{
		"form":  cs.NewDialog(nil, sc, func(string) {}),
		"macro": cs.NewMacroDialog(nil, "git", sc, mc, nil, func(string) {}),
	} {
		d.AssertRender(t, "field-widths-"+name, func() {
			d.AsyncLock() // (not while it is styled)
			defer d.AsyncUnlock()
			n := 0
			d.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
				if tf, ok := cw.(*core.TextField); ok {
					n++
					assert.Equal(t, FieldMinWidth, tf.Styles.Min.X.Value, name)
					assert.Equal(t, units.UnitCh, tf.Styles.Min.X.Unit, name)
					assert.Zero(t, tf.Styles.Max.X.Value, name)
				}
				return tree.Continue
			})
			assert.Greater(t, n, 3, name)
		})
	}
}
