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

	// inboxes is whether it is the node of the inboxes of all of the
	// accounts, at the top (see [App.initMailboxes]).
	inboxes bool

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
	if mt.inboxes {
		return "Inboxes"
	}
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
			open = mt.account || mt.inboxes
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

// initMailboxes initializes the tree of the accounts and their mailboxes:
// the inboxes of all of the accounts first, and then the accounts with
// their other mailboxes. The mailbox being shown is always selected.
// Selecting an account shows its INBOX.
func (a *App) initMailboxes(w *core.Tree) {
	w.SetText("Accounts")
	w.ContextMenus = nil // only those of the mailboxes
	w.AddContextMenu(func(m *core.Scene, pos image.Point) { a.addAccountButton(m) })
	a.mailboxes = w
	w.Updater(a.countUnread)
	w.Maker(func(p *tree.Plan) {
		tree.AddAt(p, "Inboxes", a.initInboxes)
		for _, email := range Settings.Emails() {
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

// initInboxes initializes the node of the inboxes of all of the accounts,
// with the unread messages in all of them, which is open unless closed,
// and shows them all together when selected (see [App.ShowAllInboxes]).
func (a *App) initInboxes(w *MailboxTree) {
	w.inboxes = true
	w.SetIcon(icons.Inbox)
	w.ContextMenus = nil
	w.OnSelect(func(e events.Event) {
		if !w.StateIs(states.Selected) {
			return
		}
		if a.allInboxes {
			a.mailboxes.Update() // to keep the selection on it
			return
		}
		a.ShowAllInboxes()
	})
	w.Updater(func() {
		if !w.openApplied && len(Settings.Accounts) > 0 {
			w.setOpenFromSettings()
		}
		n := 0
		for _, email := range Settings.Emails() {
			n += a.unread[email]["INBOX"]
		}
		w.SetText(withCount("Inboxes", n))
		if a.allInboxes {
			w.selectOnly()
		} else if w.StateIs(states.Selected) {
			w.Unselect()
		}
	})
	w.Maker(func(p *tree.Plan) {
		for _, email := range Settings.Emails() {
			tree.AddAt(p, email, func(w *MailboxTree) {
				w.mailbox = mailboxID{email, "INBOX"}
				w.SetIcon(icons.Inbox)
				w.ContextMenus = nil
				w.AddContextMenu(a.mailboxMenu(email, "INBOX"))
				w.OnSelect(func(e events.Event) { a.mailboxSelected(w) })
				w.Updater(func() {
					w.SetText(withCount(email, a.unread[email]["INBOX"]))
					if !a.allInboxes && email == a.currentEmail && a.showLabel == "INBOX" {
						w.selectOnly()
					} else if w.StateIs(states.Selected) {
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
	if !a.allInboxes && mb.email == a.currentEmail && mb.label == a.showLabel {
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
		starred := a.starredLabel(email)
		for _, cm := range mp {
			if slices.Contains(cm.Flags, imap.FlagSeen) {
				continue
			}
			for _, l := range cm.Labels {
				if l.Name != starred {
					counts[l.Name]++
				}
			}
			if cm.isStarred() { // by the flag, as the Starred mailbox shows
				counts[starred]++
			}
			if a.isUntagged(email, cm) {
				counts[untaggedLabel]++
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
	case "Starred": // after the Inbox, as on Gmail
		return 1
	case "Drafts", outboxLabel:
		return 2
	case "Sent", untaggedLabel: // (Sent first, by name)
		return 3
	case "Spam", "Junk":
		return 4
	case "Trash":
		return 5
	}
	return 10
}

// isSpecial returns whether the given mailbox of the given account is one
// of its special mailboxes, as the server says (with SPECIAL-USE): its
// Inbox, Drafts, Sent, Spam, Trash, and Starred, or the outbox, which are
// shown with their friendly names and icons, first. Others with names of
// special mailboxes are ordinary labels, as those that other apps made on
// Gmail ("Drafts", "Sent Messages", and "Deleted Messages" of Apple Mail,
// besides [Gmail]/Drafts, Sent Mail, and Trash), which were shown twice
// with the same names. Without the special mailboxes of the server, all are
// special by their names.
func (a *App) isSpecial(email, label string) bool {
	if label == "INBOX" || label == outboxLabel || label == untaggedLabel || label == a.starredLabel(email) || label == a.draftsLabel(email) {
		return true
	}
	su := a.specialUse[email]
	if len(su) == 0 {
		return true
	}
	for _, mb := range su {
		if mb == label {
			return true
		}
	}
	return false
}

// mailboxName returns the name of the given mailbox of the given account as
// shown: the friendly name of a special one (see [App.isSpecial]), as
// Sent for [Gmail]/Sent Mail, or else its own name.
func (a *App) mailboxName(email, label string) string {
	if a.isSpecial(email, label) {
		return friendlyLabelName(label)
	}
	return label
}

// mailboxRank is [labelRank] for the given mailbox of the given account,
// which is that of the labels for one that is not special.
func (a *App) mailboxRank(email, label string) int {
	if !a.isSpecial(email, label) {
		return 10
	}
	return labelRank(label)
}

// sortedMailboxes returns the given labels of the given account in the
// order of its mailboxes: the special ones first, then the others by name.
func (a *App) sortedMailboxes(email string, labels []string) []string {
	labels = slices.Clone(labels)
	slices.SortStableFunc(labels, func(x, y string) int {
		return cmp.Or(cmp.Compare(a.mailboxRank(email, x), a.mailboxRank(email, y)),
			cmp.Compare(strings.ToLower(a.mailboxName(email, x)), strings.ToLower(a.mailboxName(email, y))))
	})
	return labels
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
		friendlyParentLabel := a.mailboxName(email, parentLabel)
		labels := a.sortedMailboxes(email, a.mailboxLabels(email))
		for _, label := range labels {
			if skipLabels[label] || label == "INBOX" { // in the inboxes at the top
				continue
			}
			friendlyLabel := a.mailboxName(email, label)
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
				return !skipLabels[l] && strings.HasPrefix(a.mailboxName(email, l), friendlyLabel+"/")
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
					name := strings.TrimPrefix(a.mailboxName(email, label), a.mailboxName(email, parentLabel)+"/")
					if ic, ok := labelIcons[name]; ok && parentLabel == "" && a.isSpecial(email, label) {
						w.SetIcon(ic)
					} else if hasChildren {
						w.SetIcon(icons.Folder)
					} else {
						w.SetIcon(icons.Label)
					}
					n := a.unread[email][label]
					if label == a.draftsLabel(email) { // the drafts, not unread
						n = a.totalMessages[email][label]
					}
					w.SetText(withCount(name, n))
					// The mailbox being shown is always selected.
					current := !a.allInboxes && email == a.currentEmail && label == a.showLabel
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
