// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"fmt"
	"image"
	"slices"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// isUserMailbox returns whether the given mailbox is one of the user,
// which can be renamed and deleted, rather than a standard one.
func isUserMailbox(label string) bool {
	return label != "INBOX" && !strings.HasPrefix(label, "[Gmail]") && labelRank(label) == 10
}

// mailboxMenu returns the context menu of the node of the given mailbox
// of the given account in the tree of mailboxes, or of the account
// itself for an empty label.
func (a *App) mailboxMenu(email, label string) func(m *core.Scene, pos image.Point) {
	return func(m *core.Scene, pos image.Point) {
		if label == "" || isUserMailbox(label) {
			core.NewButton(m).SetText("New folder").SetIcon(icons.CreateNewFolder).OnClick(func(e events.Event) {
				a.promptName("New folder", "Name of the new folder", "", "Create", func(name string) {
					if label != "" {
						name = label + "/" + name
					}
					a.CreateMailbox(email, name)
				})
			})
		}
		if isUserMailbox(label) {
			core.NewButton(m).SetText("Rename folder").SetIcon(icons.Edit).OnClick(func(e events.Event) {
				parent, base := "", label
				if i := strings.LastIndex(label, "/"); i >= 0 {
					parent, base = label[:i+1], label[i+1:]
				}
				a.promptName("Rename folder", "New name of "+friendlyLabelName(label), base, "Rename", func(name string) {
					a.RenameMailbox(email, label, parent+name)
				})
			})
			core.NewButton(m).SetText("Delete folder").SetIcon(icons.Delete).OnClick(func(e events.Event) {
				a.confirmDeleteMailbox(email, label)
			})
		}
		if label != "" {
			core.NewSeparator(m)
			core.NewButton(m).SetText("Mark all as read").SetIcon(icons.MarkEmailRead).OnClick(func(e events.Event) {
				a.MarkAllAsRead(email, label)
			})
		} else {
			core.NewButton(m).SetText("Sync now").SetIcon(icons.Sync).OnClick(func(e events.Event) {
				a.SyncNow()
			})
		}
	}
}

// messageMenu is the context menu of the list of messages.
func (a *App) messageMenu(m *core.Scene, pos image.Point) {
	if a.readMessage == nil {
		return
	}
	core.NewFuncButton(m).SetFunc(a.Reply).SetIcon(icons.Reply)
	core.NewFuncButton(m).SetFunc(a.ReplyAll).SetIcon(icons.ReplyAll)
	core.NewFuncButton(m).SetFunc(a.Forward).SetIcon(icons.Forward)
	core.NewSeparator(m)
	core.NewFuncButton(m).SetFunc(a.File).SetIcon(icons.DriveFileMove)
	core.NewFuncButton(m).SetFunc(a.Tag).SetIcon(icons.Label)
	core.NewFuncButton(m).SetFunc(a.Delete).SetIcon(icons.Delete)
	core.NewSeparator(m)
	core.NewFuncButton(m).SetFunc(a.MarkAsRead).SetIcon(icons.MarkEmailRead)
	core.NewFuncButton(m).SetFunc(a.MarkAsUnread).SetIcon(icons.MarkAsUnread)
}

// promptName runs a dialog with the given title for the user to enter a
// name, with the given prompt, initial name, and text of the OK button,
// which calls the given function with the name, if it is not empty.
func (a *App) promptName(title, prompt, initial, ok string, fun func(name string)) {
	d := core.NewBody(title)
	core.NewText(d).SetText(prompt)
	tf := core.NewTextField(d).SetText(initial)
	tf.OnShow(func(e events.Event) { tf.SetFocus() })
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText(ok).OnClick(func(e events.Event) {
			name := strings.Trim(strings.TrimSpace(tf.Text()), "/")
			if name != "" {
				fun(name)
			}
		})
	})
	d.RunDialog(a)
}

// mailboxAction runs the given function with the IMAP client of the given
// account in a goroutine, with the lock of the account, and then gets its
// mailboxes again and updates the app, showing any error.
func (a *App) mailboxAction(email string, f func(c *imapclient.Client) error) {
	go func() {
		mu := a.imapMu[email]
		c := a.imapClient[email]
		if mu == nil || c == nil {
			core.ErrorSnackbar(a, fmt.Errorf("%s is not connected yet", email))
			return
		}
		mu.Lock()
		err := f(c)
		if err == nil {
			_, err = a.listMailboxes(c, email)
		}
		if err == nil {
			err = a.saveCacheFile(a.cache[email], email)
		}
		mu.Unlock()
		core.ErrorSnackbar(a, err, "Error")
		a.AsyncLock()
		a.Update()
		a.AsyncUnlock()
		a.SyncNow()
	}()
}

// CreateMailbox creates the given mailbox (folder) for the given account.
func (a *App) CreateMailbox(email, name string) {
	a.mailboxAction(email, func(c *imapclient.Client) error {
		return c.Create(name, nil).Wait()
	})
}

// RenameMailbox renames the given mailbox (folder) of the given account,
// with its subfolders.
func (a *App) RenameMailbox(email, old, name string) {
	if old == name {
		return
	}
	a.mailboxAction(email, func(c *imapclient.Client) error {
		if a.selectedMailbox[email] == old || strings.HasPrefix(a.selectedMailbox[email], old+"/") {
			if err := c.Unselect().Wait(); err != nil {
				return err
			}
			a.selectedMailbox[email] = ""
		}
		if err := c.Rename(old, name).Wait(); err != nil {
			return err
		}
		renamed := func(l string) string {
			if l == old || strings.HasPrefix(l, old+"/") {
				return name + strings.TrimPrefix(l, old)
			}
			return l
		}
		a.AsyncLock()
		defer a.AsyncUnlock()
		for _, cm := range a.cache[email] {
			for i := range cm.Labels {
				cm.Labels[i].Name = renamed(cm.Labels[i].Name)
			}
		}
		if email == a.currentEmail {
			a.showLabel = renamed(a.showLabel)
		}
		return nil
	})
}

// confirmDeleteMailbox asks the user to confirm deleting the given mailbox.
func (a *App) confirmDeleteMailbox(email, label string) {
	subs := 0
	for _, l := range a.labels[email] {
		if strings.HasPrefix(l, label+"/") {
			subs++
		}
	}
	msg := fmt.Sprintf("Delete the folder %s", friendlyLabelName(label))
	if subs > 0 {
		msg += fmt.Sprintf(" and its %d subfolders", subs)
	}
	msg += "? The messages that are only in it are deleted"
	if a.trash[email] == "[Gmail]/Trash" {
		msg = strings.TrimSuffix(msg, " are deleted") + " stay in All Mail"
	}
	d := core.NewBody("Delete folder")
	core.NewText(d).SetType(core.TextSupporting).SetText(msg + ".")
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Delete").OnClick(func(e events.Event) {
			a.DeleteMailbox(email, label)
		})
	})
	d.RunDialog(a)
}

// DeleteMailbox deletes the given mailbox (folder) of the given account,
// with its subfolders.
func (a *App) DeleteMailbox(email, label string) {
	var names []string
	for _, l := range a.labels[email] {
		if l == label || strings.HasPrefix(l, label+"/") {
			names = append(names, l)
		}
	}
	// the subfolders first
	slices.SortFunc(names, func(x, y string) int { return strings.Count(y, "/") - strings.Count(x, "/") })
	a.mailboxAction(email, func(c *imapclient.Client) error {
		if slices.Contains(names, a.selectedMailbox[email]) {
			if err := c.Unselect().Wait(); err != nil {
				return err
			}
			a.selectedMailbox[email] = ""
		}
		for _, name := range names {
			if err := c.Delete(name).Wait(); err != nil {
				return err
			}
		}
		a.AsyncLock()
		defer a.AsyncUnlock()
		for id, cm := range a.cache[email] {
			cm.Labels = slices.DeleteFunc(cm.Labels, func(l Label) bool { return slices.Contains(names, l.Name) })
			if len(cm.Labels) == 0 {
				delete(a.cache[email], id)
			}
		}
		if email == a.currentEmail && slices.Contains(names, a.showLabel) {
			a.showLabel = "INBOX"
			a.readMessage = nil
		}
		return nil
	})
}

// MarkAllAsRead marks all of the messages in the given mailbox of the
// given account as read.
func (a *App) MarkAllAsRead(email, label string) {
	a.mailboxAction(email, func(c *imapclient.Client) error {
		if err := a.selectMailbox(c, email, label); err != nil {
			return err
		}
		err := c.Store(imap.UIDSet{imap.UIDRange{Start: 1, Stop: 0}}, &imap.StoreFlags{
			Op: imap.StoreFlagsAdd, Silent: true, Flags: []imap.Flag{imap.FlagSeen},
		}, nil).Wait()
		if err != nil {
			return err
		}
		a.AsyncLock()
		defer a.AsyncUnlock()
		for _, cm := range a.cache[email] {
			if slices.ContainsFunc(cm.Labels, func(l Label) bool { return l.Name == label }) &&
				!slices.Contains(cm.Flags, imap.FlagSeen) {
				cm.Flags = append(cm.Flags, imap.FlagSeen)
			}
		}
		return nil
	})
}

// File moves the current message or conversation to another mailbox
// (folder), out of the mailbox being shown.
func (a *App) File() { //types:add
	from := a.showLabel
	a.chooseMailbox("File", "Move to", from, func(target string) {
		a.fileTo(from, target)
	})
}

// Tag adds another label (mailbox) to the current message or conversation,
// which stays in the mailbox being shown.
func (a *App) Tag() { //types:add
	a.chooseMailbox("Tag", "Add the label", "", a.tagWith)
}

// fileTo moves the messages being read from the given mailbox to the
// target mailbox, keeping their other labels.
func (a *App) fileTo(from, target string) {
	a.setLabels(func(cm *CacheMessage) []string {
		var res []string
		for _, l := range cm.Labels {
			if l.Name != from && l.Name != target {
				res = append(res, l.Name)
			}
		}
		return append(res, target)
	}, a.selectNext())
}

// tagWith adds the given label to the messages being read.
func (a *App) tagWith(target string) {
	a.setLabels(func(cm *CacheMessage) []string {
		var res []string
		for _, l := range cm.Labels {
			if l.Name != target {
				res = append(res, l.Name)
			}
		}
		return append(res, target)
	})
}

// chooseMailbox runs a dialog with the given title for the user to choose
// one of the mailboxes of the current account, other than the given one,
// calling the given function with it.
func (a *App) chooseMailbox(title, prompt, except string, fun func(label string)) {
	if a.readMessage == nil {
		return
	}
	d := core.NewBody(title)
	core.NewText(d).SetText(prompt)
	ch := core.NewChooser(d).SetEditable(true).SetPlaceholder("Mailbox")
	for _, label := range sortedLabels(a.labels[a.currentEmail]) {
		if skipLabels[label] || label == except || label == "[Gmail]/Drafts" {
			continue
		}
		ch.Items = append(ch.Items, core.ChooserItem{Value: label, Text: friendlyLabelName(label)})
	}
	ch.OnShow(func(e events.Event) { ch.SetFocus() })
	ch.OnChange(func(e events.Event) {
		label, ok := ch.CurrentItem.Value.(string)
		if !ok || label == "" {
			return
		}
		d.Close()
		fun(label)
	})
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
	})
	d.RunDialog(a)
}
