// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
	"github.com/emersion/go-imap/v2"
)

// mailboxID identifies a mailbox of an account, as the
// "mailbox" property of the nodes of the tree of mailboxes.
type mailboxID struct {
	email, label string
}

// initMailboxes initializes the tree of the accounts and their mailboxes.
// The mailbox being shown is always selected. Selecting an account shows
// its INBOX.
func (a *App) initMailboxes(w *core.Tree) {
	w.SetText("Accounts")
	w.ContextMenus = nil // only those of the mailboxes
	a.mailboxes = w
	w.Updater(a.countUnread)
	w.Maker(func(p *tree.Plan) {
		for _, email := range Settings.Accounts {
			tree.AddAt(p, email, func(w *core.Tree) {
				w.SetProperty("mailbox", mailboxID{email, "INBOX"})
				w.ContextMenus = nil
				w.AddContextMenu(a.mailboxMenu(email, ""))
				a.makeLabelTree(w, email, "")
				w.Updater(func() {
					w.SetText(withCount(email, a.unread[email]["INBOX"]))
					if w.StateIs(states.Selected) { // the INBOX is
						w.Unselect()
					}
				})
			})
		}
	})
	// The select events of all of the nodes are sent to the root, including
	// those from moving with the keyboard, which are not sent to the newly
	// selected node.
	w.OnSelect(func(e events.Event) {
		sel := w.GetSelectedNodes()
		if len(sel) == 0 {
			return
		}
		mb, ok := sel[len(sel)-1].AsCoreTree().Property("mailbox").(mailboxID)
		if !ok {
			return
		}
		if mb.email == a.currentEmail && mb.label == a.showLabel {
			w.Update() // to keep the selection on the mailbox being shown
			return
		}
		a.showMailbox(mb.email, mb.label)
	})
}

// withCount returns the given text with the given count of unread
// messages, if any.
func withCount(text string, n int) string {
	if n == 0 {
		return text
	}
	return fmt.Sprintf("%s (%d)", text, n)
}

// countUnread counts the unread messages in each mailbox of each account.
func (a *App) countUnread() {
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
	case "Drafts":
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
func (a *App) makeLabelTree(w *core.Tree, email, parentLabel string) {
	w.Maker(func(p *tree.Plan) {
		friendlyParentLabel := friendlyLabelName(parentLabel)
		labels := sortedLabels(a.labels[email])
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
			tree.AddAt(p, label, func(w *core.Tree) {
				w.SetProperty("mailbox", mailboxID{email, label})
				w.ContextMenus = nil
				w.AddContextMenu(a.mailboxMenu(email, label))
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
					if current != w.StateIs(states.Selected) {
						if current {
							w.Select()
						} else {
							w.Unselect()
						}
					}
				})
			})
		}
	})
}
