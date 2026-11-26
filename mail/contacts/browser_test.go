// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package contacts

import (
	"image"
	"os"
	"slices"
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBrowser tests searching the contacts, showing one in the editor,
// saving its changes, and adding and deleting contacts.
func TestBrowser(t *testing.T) {
	t.Cleanup(core.TrackWindows())
	s, err := Open(t.TempDir())
	require.NoError(t, err)
	_, err = s.Import(strings.NewReader(appleExport))
	require.NoError(t, err)
	b := core.NewBody()
	br := NewBrowser(b).SetStore(s)
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(br.MakeToolbar)
	})
	core.RenderImage(b, core.Locked(b, func() {
		assert.Equal(t, []string{"Acme Support", "Yuko Munakata"}, br.names)
		br.SearchContacts("yu")
		assert.Equal(t, []string{"Yuko Munakata"}, br.names)
		br.list.SelectIndexEvent(0, 0)
	}), core.Locked(b, func() {
		require.NotNil(t, br.Current())
		assert.Equal(t, "Yuko Munakata", br.Current().Name)
		del := core.ButtonByText(br.toolbar, "Delete contact")
		require.NotNil(t, del)
		assert.False(t, del.IsDisabled(), "with a contact to delete")
		assert.NotNil(t, core.ButtonByText(br.toolbar, "Import vCards"))
		var form *core.Form
		for w := range br.editor.AllWidgets() {
			if f, ok := w.(*core.Form); ok {
				form = f
			}
		}
		require.NotNil(t, form, "the editor")
		br.Current().Notes = "Met at CNS"
		form.SendChange()
		s2, err := Open(s.Dir)
		require.NoError(t, err)
		assert.Equal(t, "Met at CNS", s2.Contacts()[1].Notes, "saved")
	}), core.Locked(b, func() {
		br.SearchContacts("")
		br.NewContact()
		require.NotNil(t, br.Current())
		assert.Equal(t, "New Contact", br.Current().Name)
		assert.Len(t, s.Contacts(), 3)
		br.DeleteContact()
		assert.Len(t, s.Contacts(), 2)
		assert.NotNil(t, br.Current(), "the next one")
		m := core.NewScene()
		for _, f := range slices.Backward(br.list.ContextMenus) { // as the menu is made
			f(m, image.Point{})
		}
		del := core.ButtonByText(m, "Delete contact")
		require.NotNil(t, del, "in the context menu of the list")
		assert.Equal(t, keymap.Delete.Chord(), del.Shortcut, "with the shortcut shown")
		assert.NotNil(t, core.ButtonByText(m, "New contact"))
		assert.NotNil(t, core.ButtonByText(m, "Copy"), "and those of the list")
		br.list.SetFocus()
	}), func() {
		// the Delete key of the key map, which asks first
		r, code, mods, err := keymap.Delete.Chord().Decode()
		require.NoError(t, err)
		b.Scene.RenderWindow().SystemWindow.Events().KeyChord(r, code, mods)
	}, core.Locked(b, func() {
		d := core.TopScene(b)
		require.NotEqual(t, b.Scene, d, "the dialog")
		del := core.ButtonByText(d, "Delete")
		require.NotNil(t, del)
		del.Send(events.Click)
		assert.Len(t, s.Contacts(), 1, "deleted")
	}))
}

// TestLabelChooser tests the editable choosers of the labels of the values
// of a contact, which offer the standard labels and those of the contacts.
func TestLabelChooser(t *testing.T) {
	t.Cleanup(core.TrackWindows())
	s, err := Open(t.TempDir())
	require.NoError(t, err)
	_, err = s.Import(strings.NewReader(appleExport))
	require.NoError(t, err)
	labels := Labels()
	assert.Contains(t, labels, "work", "standard")
	assert.Contains(t, labels, "cell", "of the contacts")
	b := core.NewBody()
	br := NewBrowser(b).SetStore(s)
	core.RenderImage(b, core.Locked(b, func() {
		br.Select(s.Contacts()[1])
	}), core.Locked(b, func() {
		var chs []*core.Chooser
		for w := range br.editor.AllWidgets() {
			if ch, ok := w.(*core.Chooser); ok {
				chs = append(chs, ch)
			}
		}
		require.Len(t, chs, 3, "of the two emails and the phone (an address opens in its own form)")
		var addr *core.FormButton
		tips := map[string]string{}
		for w := range br.editor.AllWidgets() {
			if fb, ok := w.(*core.FormButton); ok {
				addr = fb
			}
			for _, f := range []string{"Emails", "Phones", "Addresses"} {
				if strings.HasPrefix(w.AsTree().Name, "value-"+f+"-") {
					tips[f] = w.AsWidget().Tooltip
				}
			}
		}
		require.NotNil(t, addr)
		assert.Equal(t, "1 Main St", addr.Text, "the first line of the address")
		for _, f := range []string{"Emails", "Phones", "Addresses"} {
			assert.Contains(t, tips[f], "Open in dialog", f)
		}
		ch := chs[0]
		assert.True(t, ch.Editable)
		var texts []string
		for _, it := range ch.Items {
			texts = append(texts, it.GetText())
		}
		assert.Subset(t, texts, labels, "all of the labels")
		ch.SetCurrentValue(Label("work")).SendChange()
		assert.Equal(t, Label("work"), br.Current().Emails[0].Label, "chosen")
	}))
	s2, err := Open(s.Dir)
	require.NoError(t, err)
	assert.Equal(t, Label("work"), s2.Contacts()[1].Emails[0].Label, "saved")
}

// TestBrowserSplits tests the proportions of the browser: 0.2 for the list
// by default, and those set by the user, which are saved in the settings.
func TestBrowserSplits(t *testing.T) {
	t.Cleanup(func() { Settings.Splits = nil })
	s, err := Open(t.TempDir())
	require.NoError(t, err)
	b := core.NewBody()
	br := NewBrowser(b).SetStore(s)
	var sp *core.Splits
	core.RenderImage(b, core.Locked(b, func() {
		sp = br.Child(0).(*core.Splits)
		assert.Equal(t, []float32{0.2, 0.8}, sp.Splits(), "by default")
		sp.SetSplits(0.3, 0.7)
		sp.SendChange()
	}))
	assert.Equal(t, []float32{0.3, 0.7}, Settings.Splits)
	data, err := os.ReadFile(Settings.Filename())
	require.NoError(t, err)
	assert.Contains(t, string(data), "Splits", "saved")
	b2 := core.NewBody()
	br2 := NewBrowser(b2).SetStore(s)
	core.RenderImage(b2, core.Locked(b2, func() {
		assert.Equal(t, []float32{0.3, 0.7}, br2.Child(0).(*core.Splits).Splits(), "as the user set them")
	}))
}
