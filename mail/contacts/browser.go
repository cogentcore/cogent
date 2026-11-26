// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package contacts

import (
	"fmt"
	"image"
	"slices"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// Browser is a browser of contacts, as the Contacts app of the Mac: a
// search field and the list of the contacts that match it on the left,
// and an editor of the selected contact on the right, which saves each
// change to it.
type Browser struct {
	core.Frame

	// Store is the store of the contacts.
	Store *Store `set:"-"`

	// current is the contact being shown, or nil.
	current *Contact

	// shown are the contacts in the list, which match the search, and
	// names are their names, shown in the list.
	shown []*Contact
	names []string

	// search is the text of the search field.
	search string

	// list is the list of the contacts.
	list *core.List

	// editor is the panel of the editor of the current contact.
	editor *core.Frame

	// toolbar is the toolbar made by [Browser.MakeToolbar], if any,
	// which is updated when the current contact changes.
	toolbar *core.Toolbar
}

// SetStore sets the store of the contacts that the browser shows.
// It is updated when the contacts change in a sync.
func (br *Browser) SetStore(s *Store) *Browser {
	br.Store = s
	br.updateShown()
	s.OnChange(func() {
		if !br.AsyncLockIfShown() {
			return // closed, or not shown yet
		}
		defer br.AsyncUnlock()
		br.updateShown()
		br.Update()
	})
	return br
}

// Current returns the contact being shown, or nil.
func (br *Browser) Current() *Contact { return br.current }

func (br *Browser) Init() {
	br.Frame.Init()
	br.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	tree.AddChild(br, func(w *core.Splits) {
		w.SetSplits(0.35, 0.65)
		tree.AddChild(w, func(w *core.Frame) { // the search and the list
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
			})
			tree.AddChild(w, func(w *core.TextField) {
				w.SetPlaceholder("Search").SetLeadingIcon(icons.Search)
				w.Styler(func(s *styles.Style) {
					s.Grow.Set(1, 0)
					s.Max.X.Zero()
				})
				w.SetTrailingIcon(icons.Close, func(e events.Event) {
					w.SetText("")
					br.SearchContacts("")
				})
				w.OnInput(func(e events.Event) { br.SearchContacts(w.Text()) }) // as it is typed
				w.OnChange(func(e events.Event) { br.SearchContacts(w.Text()) })
			})
			tree.AddChild(w, func(w *core.List) {
				br.list = w
				w.SetReadOnly(true)
				w.SetSlice(&br.names)
				w.Styler(func(s *styles.Style) {
					s.Grow.Set(1, 1)
				})
				w.OnSelect(func(e events.Event) {
					if i := w.SelectedIndex; i >= 0 && i < len(br.shown) {
						br.Select(br.shown[i])
					}
				})
				// first in the menu, with the right-clicked contact selected
				w.AddContextMenu(func(m *core.Scene, pos image.Point) {
					core.NewButton(m).SetText("New contact").SetIcon(icons.PersonAdd).OnClick(func(e events.Event) {
						br.NewContact()
					})
					core.NewButton(m).SetText("Delete contact").SetIcon(icons.Delete).SetKey(keymap.Delete).
						SetEnabled(br.current != nil).OnClick(func(e events.Event) {
						br.confirmDelete()
					})
				})
				// the Delete keys of the key map (as Control+D in Emacs)
				w.OnFirst(events.KeyChord, func(e events.Event) {
					if kf := keymap.Of(e.KeyChord()); kf == keymap.Delete || kf == keymap.Backspace {
						e.SetHandled()
						br.confirmDelete()
					}
				})
			})
		})
		tree.AddChild(w, func(w *core.Frame) { // the editor
			br.editor = w
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
				s.Overflow.Set(styles.OverflowAuto)
			})
			w.Maker(func(p *tree.Plan) {
				c := br.current
				if c == nil {
					tree.Add(p, func(w *core.Text) {
						w.SetType(core.TextSupporting).SetText("No contact selected")
					})
					return
				}
				tree.AddAt(p, "name-"+c.uid, func(w *core.Text) {
					w.SetType(core.TextHeadlineSmall)
					w.Updater(func() { w.SetText(c.Name) })
				})
				tree.AddAt(p, "form-"+c.uid, func(w *core.Form) {
					w.SetStruct(c)
					w.OnChange(func(e events.Event) {
						core.ErrorSnackbar(br, br.Store.Save(c), "Error saving contact")
						br.updateShown()
						br.Update()
					})
				})
			})
		})
	})
}

// SearchContacts shows the contacts that match the given search (see [Store.Search]).
func (br *Browser) SearchContacts(search string) {
	br.search = search
	br.updateShown()
	br.Update()
}

// updateShown updates the contacts shown in the list, keeping the current
// one selected if it is among them.
func (br *Browser) updateShown() {
	if br.Store == nil {
		return
	}
	if br.current != nil && !slices.Contains(br.Store.Contacts(), br.current) {
		br.current = br.Store.byUID(br.current.uid) // replaced in a sync, or deleted
	}
	br.shown = br.Store.Search(br.search)
	br.names = br.names[:0]
	for _, c := range br.shown {
		name := c.Name
		if name == "" {
			name = "(no name)"
		}
		br.names = append(br.names, name)
	}
	if br.list != nil {
		br.list.SelectedIndex = slices.Index(br.shown, br.current)
	}
}

// Select shows the given contact in the editor, and selects it in the list.
func (br *Browser) Select(c *Contact) {
	br.current = c
	if br.list != nil {
		br.list.SelectedIndex = slices.Index(br.shown, c)
	}
	if br.editor != nil {
		br.editor.Update()
	}
	if br.toolbar != nil {
		br.toolbar.Update() // as Delete contact
	}
}

// NewContact adds a new contact, and shows it to be edited.
func (br *Browser) NewContact() { //types:add
	c := br.Store.New()
	c.Name = "New Contact"
	c.Emails = []Labeled{{}}
	if err := br.Store.Save(c); err != nil {
		core.ErrorSnackbar(br, err, "Error adding contact")
		return
	}
	br.search = ""
	br.updateShown()
	br.Select(c)
	br.Update()
}

// DeleteContact deletes the contact being shown.
func (br *Browser) DeleteContact() { //types:add
	c := br.current
	if c == nil {
		return
	}
	i := slices.Index(br.shown, c)
	if err := br.Store.Delete(c); err != nil {
		core.ErrorSnackbar(br, err, "Error deleting contact")
		return
	}
	br.current = nil
	br.updateShown()
	if len(br.shown) > 0 && i >= 0 { // the next one
		br.Select(br.shown[min(i, len(br.shown)-1)])
	}
	br.Update()
}

// confirmDelete asks whether to delete the contact being shown, and
// deletes it if so.
func (br *Browser) confirmDelete() {
	c := br.current
	if c == nil {
		return
	}
	d := core.NewBody("Delete contact")
	core.NewText(d).SetType(core.TextSupporting).SetText(fmt.Sprintf("Delete %s?", c.Name))
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Delete").OnClick(func(e events.Event) {
			if br.current == c {
				br.DeleteContact()
			}
		})
	})
	d.RunDialog(br)
}

// ImportVCards imports the contacts in the given vCard (.vcf) file, as
// exported by Contacts on the Mac (File > Export > Export vCard) or by
// Google Contacts, replacing those that were imported before.
func (br *Browser) ImportVCards(file core.Filename) error { //types:add
	n, err := br.Store.ImportFile(string(file))
	br.updateShown()
	br.Update()
	if err == nil {
		core.MessageSnackbar(br, fmt.Sprintf("Imported %d contacts", n))
	}
	return err
}

// MakeToolbar makes the toolbar of the browser.
func (br *Browser) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.FuncButton) {
		br.toolbar, _ = w.Parent.(*core.Toolbar)
		w.SetFunc(br.NewContact).SetIcon(icons.PersonAdd)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(br.DeleteContact).SetConfirm(true).SetIcon(icons.Delete)
		w.FirstStyler(func(s *styles.Style) {
			s.SetEnabled(br.current != nil)
		})
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(br.ImportVCards).SetText("Import vCards").SetIcon(icons.Upload)
	})
}
