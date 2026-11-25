// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/tree"
	"github.com/cogentcore/go-imap/v2"
	"github.com/cogentcore/go-imap/v2/imapclient"
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
	p := tree.Plan{}
	a.messageButtons(&p, false)
	p.Update(m)
}

// messageButtons adds the buttons of the actions on the messages being
// read to the given plan, with their shortcuts: for the toolbar, or else
// for the context menu, with separators between the groups.
func (a *App) messageButtons(p *tree.Plan, toolbar bool) {
	button := func(fun any, name string, icon icons.Icon) {
		tree.AddAt(p, name, func(w *core.FuncButton) {
			w.SetFunc(fun).SetIcon(icon).SetShortcut(shortcutOf(name))
		})
	}
	separator := func(name string) {
		tree.AddAt(p, name, func(w *core.Separator) {})
	}
	button(a.Reply, "Reply", icons.Reply)
	button(a.ReplyAll, "ReplyAll", icons.ReplyAll)
	button(a.Forward, "Forward", icons.Forward)
	separator("sep-file")
	button(a.File, "File", icons.DriveFileMove)
	button(a.Tag, "Tag", icons.Label)
	button(a.Delete, "Delete", icons.Delete)
	separator("sep-read")
	if !toolbar {
		button(a.MarkAsRead, "MarkAsRead", icons.MarkEmailRead)
	}
	button(a.MarkAsUnread, "MarkAsUnread", icons.MarkAsUnread)
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
			err = a.saveAllMessages(email)
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
		if err := c.Rename(old, name, nil).Wait(); err != nil {
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
		st := a.stores[email]
		for _, l := range a.labels[email] {
			if r := renamed(l); r != l {
				if ms, err := st.mailbox(l); err == nil && ms.UIDValidity != 0 {
					st.setMailbox(r, ms)
					st.removeMailbox(l)
				}
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
		for _, name := range names {
			if err := a.removeMailboxUIDs(email, name, nil); err != nil {
				return err
			}
			a.stores[email].removeMailbox(name)
		}
		a.AsyncLock()
		defer a.AsyncUnlock()
		for _, cm := range a.cache[email] { // labels without UIDs, as on Gmail
			cm.Labels = slices.DeleteFunc(cm.Labels, func(l Label) bool { return slices.Contains(names, l.Name) })
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
		}, nil).Close()
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
// (folder), out of the mailbox being shown. The mailboxes of the
// [SettingsData.FilingAccount] are first, if set, as for filing all of the
// messages into the folders of one account, and then those of the other
// accounts, which the messages are copied to.
func (a *App) File() { //types:add
	from := a.showLabel
	a.chooseMailbox("File", "Move to", a.filingAccounts(), mailboxID{a.currentEmail, from}, func(target mailboxID) {
		if target.email == a.currentEmail {
			a.fileTo(from, target.label)
		} else {
			a.fileToAccount(from, target)
		}
	})
}

// Tag adds another label (mailbox) to the current message or conversation,
// which stays in the mailbox being shown.
func (a *App) Tag() { //types:add
	a.chooseMailbox("Tag", "Add the label", []string{a.currentEmail}, mailboxID{}, func(target mailboxID) {
		a.tagWith(target.label)
	})
}

// filingAccounts returns the accounts whose mailboxes are offered for
// filing, in order: the [SettingsData.FilingAccount] if set, else the
// current account, and then the others.
func (a *App) filingAccounts() []string {
	first := a.currentEmail
	if fa := string(Settings.FilingAccount); fa != "" && slices.Contains(Settings.Accounts, fa) {
		first = fa
	}
	res := []string{first}
	for _, acct := range Settings.Accounts {
		if acct != first {
			res = append(res, acct)
		}
	}
	return res
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

// fileToAccount moves the messages being read from the given mailbox to
// the target mailbox of another account: it appends a copy of each of
// them to it, with its flags and date, from the cached message, and once
// they are all there, removes them from the given mailbox of the current
// account, keeping their other labels (as Gmail keeps them in All Mail).
func (a *App) fileToAccount(from string, target mailboxID) {
	email := a.currentEmail
	msgs := a.readMessages()
	next := a.selectNext()
	go func() {
		mu, c := a.imapMu[target.email], a.imapClient[target.email]
		if mu == nil || c == nil {
			core.ErrorSnackbar(a, fmt.Errorf("%s is not connected yet", target.email))
			return
		}
		mu.Lock()
		err := a.appendMessages(c, email, msgs, target.label)
		mu.Unlock()
		if err != nil {
			core.ErrorSnackbar(a, err, "Error filing in "+target.email)
			return
		}
		a.setLabelsOn(email, msgs, func(cm *CacheMessage) []string {
			var res []string
			for _, l := range cm.Labels {
				if l.Name != from {
					res = append(res, l.Name)
				}
			}
			return res
		}, next)
		a.SyncNow() // to get them in the other account
	}()
}

// appendMessages appends the given cached messages of the given account
// to the given mailbox with the given client, of another account.
func (a *App) appendMessages(c *imapclient.Client, email string, msgs []*CacheMessage, mailbox string) error {
	for _, cm := range msgs {
		data, err := os.ReadFile(filepath.Join(a.messagesDir(email), cm.Filename))
		if err != nil {
			return err
		}
		flags := slices.DeleteFunc(slices.Clone(cm.Flags), func(f imap.Flag) bool {
			return f == "\\Recent" // not settable
		})
		cmd := c.Append(mailbox, int64(len(data)), &imap.AppendOptions{Flags: flags, Time: cm.Date})
		if _, err := cmd.Write(data); err != nil {
			cmd.Close()
			return err
		}
		if err := cmd.Close(); err != nil {
			return err
		}
		if _, err := cmd.Wait(); err != nil {
			return fmt.Errorf("appending %q to %s: %w", cm.Subject, mailbox, err)
		}
	}
	return nil
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
// one of the mailboxes of the given accounts, other than the given one,
// calling the given function with it. Those of the first account have
// their names, and those of the others have the account first.
func (a *App) chooseMailbox(title, prompt string, accounts []string, except mailboxID, fun func(target mailboxID)) {
	if a.readMessage == nil {
		return
	}
	d := core.NewBody(title)
	core.NewText(d).SetText(prompt)
	ch := core.NewChooser(d).SetEditable(true).SetPlaceholder("Mailbox")
	for i, email := range accounts {
		for _, label := range sortedLabels(a.labels[email]) {
			mb := mailboxID{email, label}
			if skipLabels[label] || mb == except || labelRank(label) == 1 { // not Drafts
				continue
			}
			text := friendlyLabelName(label)
			if i > 0 {
				text = email + ": " + text
			}
			ch.Items = append(ch.Items, core.ChooserItem{Value: mb, Text: text})
		}
	}
	ch.OnShow(func(e events.Event) { ch.SetFocus() })
	ch.OnChange(func(e events.Event) {
		mb, ok := ch.CurrentItem.Value.(mailboxID)
		if !ok || mb.label == "" {
			return
		}
		d.Close()
		fun(mb)
	})
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
	})
	d.RunDialog(a)
}
