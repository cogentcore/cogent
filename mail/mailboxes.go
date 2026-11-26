// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"cmp"
	"fmt"
	"image"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
	"github.com/cogentcore/go-imap/v2"
)

// mailboxID identifies a mailbox of an account, as the
// "mailbox" property of the nodes of the tree of mailboxes.
type mailboxID struct {
	email, label string
}

// MailboxTree is a node of the tree of mailboxes, for an account (with
// the INBOX as its mailbox) or a mailbox. It records whether it is open
// in the [SettingsData.OpenMailboxes].
type MailboxTree struct {
	core.Tree

	// mailbox is the account and mailbox of the node.
	mailbox mailboxID

	// account is whether it is the node of the account.
	account bool

	// openApplied is whether the saved open state has been applied.
	openApplied bool

	// applying is whether the saved open state is being applied,
	// so that it is not saved again.
	applying bool
}

// mailboxKey returns the key of the given mailbox of the given account in
// [SettingsData.OpenMailboxes], or of the account for an empty label.
func mailboxKey(email, label string) string {
	if label == "" {
		return email
	}
	return email + "|" + label
}

// key returns the key of the node in [SettingsData.OpenMailboxes].
func (mt *MailboxTree) key() string {
	if mt.account {
		return mailboxKey(mt.mailbox.email, "")
	}
	return mailboxKey(mt.mailbox.email, mt.mailbox.label)
}

// setOpenFromSettings opens or closes the node as saved in the settings,
// or else open for an account, and closed for a mailbox, after the next
// render, once its children are made. If they are not made yet, as a node
// without children is closed and cannot be opened, it is done in the next
// update (see [MailboxTree.openApplied]).
func (mt *MailboxTree) setOpenFromSettings() {
	mt.openApplied = true
	slog.Debug("mail: opening the mailbox node", "key", mt.key(), "children", len(mt.Children), "closed", mt.Closed)
	mt.Defer(func() {
		if !mt.HasChildren() {
			slog.Debug("mail: mailbox node without children yet", "key", mt.key())
			mt.openApplied = false // again in the next update
			return
		}
		open, ok := Settings.OpenMailboxes[mt.key()]
		if !ok {
			open = mt.account
		}
		slog.Debug("mail: opened the mailbox node", "key", mt.key(), "open", open, "saved", ok, "children", len(mt.Children))
		mt.applying = true
		if open {
			mt.Open()
		} else {
			mt.Close()
		}
		mt.applying = false
	})
}

// selectOnly makes the node the only selected node, if it is not already.
func (mt *MailboxTree) selectOnly() {
	sel := mt.GetSelectedNodes()
	if len(sel) == 1 && sel[0] == mt.This && mt.StateIs(states.Selected) {
		return
	}
	mt.Root.AsCoreTree().UnselectAll()
	mt.Select()
}

func (mt *MailboxTree) OnOpen()  { mt.saveOpen(true) }
func (mt *MailboxTree) OnClose() { mt.saveOpen(false) }

// saveOpen saves whether the node is open in the settings.
func (mt *MailboxTree) saveOpen(open bool) {
	if mt.applying {
		return
	}
	if Settings.OpenMailboxes == nil {
		Settings.OpenMailboxes = map[string]bool{}
	}
	if old, ok := Settings.OpenMailboxes[mt.key()]; ok && old == open {
		return
	}
	Settings.OpenMailboxes[mt.key()] = open
	errors.Log(os.MkdirAll(filepath.Dir(Settings.Filename()), 0o700))
	errors.Log(core.SaveSettings(Settings))
}

// initMailboxes initializes the tree of the accounts and their mailboxes.
// The mailbox being shown is always selected. Selecting an account shows
// its INBOX.
func (a *App) initMailboxes(w *core.Tree) {
	w.SetText("Accounts")
	w.ContextMenus = nil // only those of the mailboxes
	w.AddContextMenu(func(m *core.Scene, pos image.Point) { a.addAccountButton(m) })
	a.mailboxes = w
	w.Updater(a.countUnread)
	w.Maker(func(p *tree.Plan) {
		for _, email := range Settings.Accounts {
			tree.AddAt(p, email, func(w *MailboxTree) {
				w.mailbox = mailboxID{email, "INBOX"}
				w.account = true
				w.ContextMenus = nil
				w.AddContextMenu(a.mailboxMenu(email, ""))
				w.OnSelect(func(e events.Event) { a.mailboxSelected(w) })
				a.makeLabelTree(w, email, "")
				w.Updater(func() {
					// its mailboxes are made in this update once they are known
					if !w.openApplied && len(a.mailboxLabels(email)) > 0 {
						w.setOpenFromSettings()
					}
					w.SetText(withCount(email, a.unread[email]["INBOX"]))
					if w.StateIs(states.Selected) { // the INBOX is
						w.Unselect()
					}
				})
			})
		}
	})
}

// mailboxSelected handles a select event on the node of a mailbox (by a
// click or moving with the keyboard), showing its mailbox if it is selected.
func (a *App) mailboxSelected(mt *MailboxTree) {
	if !mt.StateIs(states.Selected) {
		return
	}
	mb := mt.mailbox
	if mb.email == a.currentEmail && mb.label == a.showLabel {
		a.mailboxes.Update() // to keep the selection on the mailbox being shown
		return
	}
	a.showMailbox(mb.email, mb.label)
}

// withCount returns the given text with the given count of unread
// messages, if any.
func withCount(text string, n int) string {
	if n == 0 {
		return text
	}
	return fmt.Sprintf("%s\u2002(%d)", text, n) // a wider space
}

// countUnread counts the unread messages in each mailbox of each account.
func (a *App) countUnread() {
	start := time.Now()
	defer func() {
		if d := time.Since(start); d > 50*time.Millisecond {
			slog.Debug("mail: counting the unread messages took a while", "took", d)
		}
	}()
	a.unread = map[string]map[string]int{}
	for email, mp := range a.cache {
		counts := map[string]int{}
		for _, cm := range mp {
			if slices.Contains(cm.Flags, imap.FlagSeen) {
				continue
			}
			for _, l := range cm.Labels {
				counts[l.Name]++
			}
		}
		a.unread[email] = counts
	}
}

// labelRank is the rank of the given label for the order of the mailboxes:
// the standard mailboxes first, in the usual order, then the others.
func labelRank(label string) int {
	switch friendlyLabelName(label) {
	case "Inbox":
		return 0
	case "Drafts", outboxLabel:
		return 1
	case "Sent":
		return 2
	case "Spam", "Junk":
		return 3
	case "Trash":
		return 4
	}
	return 10
}

// sortedLabels returns the given labels in the order of the mailboxes.
func sortedLabels(labels []string) []string {
	labels = slices.Clone(labels)
	slices.SortStableFunc(labels, func(x, y string) int {
		return cmp.Or(cmp.Compare(labelRank(x), labelRank(y)),
			cmp.Compare(strings.ToLower(friendlyLabelName(x)), strings.ToLower(friendlyLabelName(y))))
	})
	return labels
}

// makeLabelTree recursively adds a Maker to the given tree to form a nested tree of labels.
func (a *App) makeLabelTree(w *MailboxTree, email, parentLabel string) {
	w.Maker(func(p *tree.Plan) {
		friendlyParentLabel := friendlyLabelName(parentLabel)
		labels := sortedLabels(a.mailboxLabels(email))
		for _, label := range labels {
			if skipLabels[label] {
				continue
			}
			friendlyLabel := friendlyLabelName(label)
			// Skip labels that are not directly nested under the parent label.
			if parentLabel == "" && strings.Contains(friendlyLabel, "/") {
				continue
			} else if parentLabel != "" {
				if !strings.HasPrefix(friendlyLabel, friendlyParentLabel+"/") ||
					strings.Count(friendlyLabel, "/") > strings.Count(friendlyParentLabel, "/")+1 {
					continue
				}
			}
			hasChildren := slices.ContainsFunc(labels, func(l string) bool {
				return !skipLabels[l] && strings.HasPrefix(friendlyLabelName(l), friendlyLabel+"/")
			})
			tree.AddAt(p, label, func(w *MailboxTree) {
				w.mailbox = mailboxID{email, label}
				if hasChildren {
					w.setOpenFromSettings()
				}
				w.ContextMenus = nil
				w.AddContextMenu(a.mailboxMenu(email, label))
				w.OnSelect(func(e events.Event) { a.mailboxSelected(w) })
				a.makeLabelTree(w, email, label)
				w.Updater(func() {
					// Recompute the friendly labels in case they have changed.
					name := strings.TrimPrefix(friendlyLabelName(label), friendlyLabelName(parentLabel)+"/")
					if ic, ok := labelIcons[name]; ok && parentLabel == "" {
						w.SetIcon(ic)
					} else if hasChildren {
						w.SetIcon(icons.Folder)
					} else {
						w.SetIcon(icons.Label)
					}
					w.SetText(withCount(name, a.unread[email][label]))
					// The mailbox being shown is always selected.
					current := email == a.currentEmail && label == a.showLabel
					if current {
						w.selectOnly()
					} else if w.StateIs(states.Selected) {
						w.Unselect()
					}
				})
			})
		}
	})
}
