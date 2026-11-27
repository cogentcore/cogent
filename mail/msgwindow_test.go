// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/base/datasize"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
	"github.com/cogentcore/go-imap/v2"
	gomail "github.com/emersion/go-message/mail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// longMessages adds long messages with the given names to the inbox of the
// test account, which scroll when they are read.
func longMessages(t *testing.T, a *App, names ...string) {
	require.NoError(t, os.MkdirAll(a.messagesDir(testEmail), 0o700))
	a.cache[testEmail] = map[string]*CacheMessage{}
	for i, name := range names {
		msg := "From: a@example.com\r\nSubject: " + name + "\r\nContent-Type: text/plain\r\n\r\n" +
			strings.Repeat("A line of the message "+name+".\n\n", 200)
		require.NoError(t, os.WriteFile(filepath.Join(a.messagesDir(testEmail), name), []byte(msg), 0o600))
		a.cache[testEmail][name] = &CacheMessage{key: name, Filename: name,
			Envelope: imap.Envelope{Subject: name, Date: time.Now().Add(-time.Duration(i) * time.Hour)},
			Labels:   []Label{{Name: "INBOX", UID: imap.UID(i + 1)}}, Flags: []imap.Flag{imap.FlagSeen}}
	}
}

// TestReadPanelKeys tests that the keys of the next panel go on to the
// panel of the message being read, which then scrolls with the arrow keys
// and PageDown, and on from it to the mailboxes.
func TestReadPanelKeys(t *testing.T) {
	b, a := newTestApp(t, nil)
	old := Settings.Keys
	t.Cleanup(func() { Settings.Keys = old })
	Settings.Keys = DefaultKeys() // Control+Tab, as Control+X o in the Emacs keymaps
	longMessages(t, a, "one", "two")
	press := func(chords ...key.Chord) func() {
		return func() {
			for _, c := range chords {
				r, code, mods, err := c.PlatformChord().Decode()
				require.NoError(t, err)
				b.Scene.RenderWindow().SystemWindow.Events().KeyChord(r, code, mods)
			}
		}
	}
	var line float32
	core.RenderImage(b, core.Locked(b, func() {
		a.showMailbox(testEmail, "INBOX")
	}), core.Locked(b, func() {
		a.list.SelectIndexEvent(0, events.SelectOne)
		a.list.SetFocusQuiet()
	}), press("Control+Tab"), core.Locked(b, func() {
		require.True(t, a.readPanel.HasScroll[math32.Y], "long enough to scroll")
		assert.True(t, a.readPanel.IsDimAtContentStart(math32.Y))
	}), press("DownArrow"), core.Locked(b, func() {
		line = a.readPanel.Scrolls[math32.Y].Value
		assert.Greater(t, line, float32(0), "scrolled by a line")
	}), press("PageDown"), core.Locked(b, func() {
		assert.Greater(t, a.readPanel.Scrolls[math32.Y].Value, 2*line, "and a page")
	}), press("Control+Tab"), core.Locked(b, func() {
		assert.True(t, a.sidebar.AsWidget().ContainsFocus(), "on to the mailboxes")
	}))
}

// TestMessageWindow tests that double-clicking a message opens it in a
// window of its own, which is updated with the app, and which is no longer
// updated once closed.
func TestMessageWindow(t *testing.T) {
	b, a := newTestApp(t, nil)
	longMessages(t, a, "one", "two")
	core.RenderImage(b, core.Locked(b, func() {
		a.showMailbox(testEmail, "INBOX")
	}), core.Locked(b, func() {
		a.list.SelectIndexEvent(1, events.SelectOne)
		a.list.Send(events.DoubleClick)
		require.Len(t, a.messageWindows, 1)
	}), func() {
		w := a.messageWindows[0]
		require.Eventually(t, func() bool {
			if !w.AsyncLockIfShown() {
				return false
			}
			defer w.AsyncUnlock()
			var shown *CacheMessage
			w.WalkDown(func(n tree.Node) bool {
				if dmf, ok := n.(*DisplayMessageFrame); ok {
					shown = dmf.Message
				}
				return true
			})
			return shown != nil && shown.Subject == "two"
		}, 5*time.Second, 10*time.Millisecond, "the message double-clicked")
		w.AsyncLock()
		w.Close()
		w.AsyncUnlock()
		require.Eventually(t, func() bool {
			a.AsyncLock()
			defer a.AsyncUnlock()
			return len(a.messageWindows) == 0
		}, 5*time.Second, 10*time.Millisecond, "no longer updated")
	})
}

// TestShowMailboxReads tests that showing another mailbox selects its first
// message and shows it in the panel of the message being read, without
// moving the keyboard focus into the list, and shows nothing for an empty
// mailbox.
func TestShowMailboxReads(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	shown := func() []*CacheMessage { // newest first, for a conversation
		var res []*CacheMessage
		a.readPanel.WalkDown(func(n tree.Node) bool {
			if dmf, ok := n.(*DisplayMessageFrame); ok {
				res = append(res, dmf.Message)
			}
			return true
		})
		return res
	}
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
	}, core.Locked(b, func() {
		a.showMailbox(testEmail, "INBOX")
		a.mailboxes.SetFocusQuiet()
		a.showMailbox(testEmail, "Work")
		require.NotNil(t, a.rowAt(0))
		assert.Equal(t, a.rowAt(0).message, a.readMessage, "the first message")
		assert.Equal(t, 0, a.list.SelectedIndex)
		assert.Len(t, a.list.SelectedIndexes, 1)
		assert.Contains(t, shown(), a.readMessage, "in the panel")
		assert.False(t, a.list.ContainsFocus(), "the focus stays")
		work := a.readMessage
		a.showMailbox(testEmail, "INBOX")
		require.NotNil(t, a.rowAt(0))
		assert.Equal(t, a.rowAt(0).message, a.readMessage, "that of the other mailbox")
		assert.Contains(t, shown(), a.readMessage)
		a.showMailbox(testEmail, "Work")
		assert.Equal(t, work, a.readMessage, "the first again")
		a.showMailbox(testEmail, "[Gmail]/Trash")
		assert.Nil(t, a.readMessage, "none in an empty mailbox")
		assert.Empty(t, shown())
		assert.Empty(t, a.list.SelectedIndexes)
	}))
}

// TestMessageWindowReply tests the Reply, Reply all, and Forward buttons of
// a window of a message, which apply to its message, not to the message
// being read in the main window.
func TestMessageWindowReply(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	var report *CacheMessage
	var w *core.Body
	buttons := map[string]*core.Button{}
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
	}, core.Locked(b, func() {
		report = messageBySubject(a, "Report")
		a.showMailbox(testEmail, "Work")
		for i := 0; a.rowAt(i) != nil; i++ {
			if a.rowAt(i).message == report {
				a.openMessageWindow(a.rowAt(i))
			} else {
				a.list.SelectIndexEvent(i, events.SelectOne) // another one read
			}
		}
		require.Len(t, a.messageWindows, 1)
		require.NotEqual(t, report, a.readMessage)
		w = a.messageWindows[0]
	}), func() {
		require.Eventually(t, func() bool {
			if !w.AsyncLockIfShown() {
				return false
			}
			defer w.AsyncUnlock()
			for n := range w.Scene.AllWidgets() {
				if bt, ok := n.(*core.Button); ok {
					buttons[bt.Text] = bt
				}
			}
			return buttons["Forward"] != nil
		}, 5*time.Second, 10*time.Millisecond, "the buttons")
		click := func(text string) *SendMessage {
			a.AsyncLock()
			a.composeMessage = nil
			a.AsyncUnlock()
			w.AsyncLock()
			buttons[text].Send(events.Click)
			w.AsyncUnlock()
			var sm *SendMessage
			require.Eventually(t, func() bool {
				a.AsyncLock()
				defer a.AsyncUnlock()
				sm = a.composeMessage
				return sm != nil
			}, 5*time.Second, 10*time.Millisecond, text)
			return sm
		}
		sm := click("Reply")
		assert.Equal(t, "Re: Report", sm.Subject)
		require.NotEmpty(t, sm.To)
		assert.Equal(t, "carol@example.com", sm.To[0].Address)
		assert.Equal(t, report, sm.original, "the message of the window")
		assert.Equal(t, testEmail, string(sm.Sender.From))
		assert.Contains(t, sm.body, "Carol")
		sm = click("Reply all")
		assert.Equal(t, "Re: Report", sm.Subject)
		sm = click("Forward")
		assert.Equal(t, "Fwd: Report", sm.Subject)
		assert.Equal(t, report, sm.original)
	})
}

// TestDoubleClickDraft tests that double-clicking a draft opens it for
// writing it more, not in a window for reading it.
func TestDoubleClickDraft(t *testing.T) {
	b, a := newTestingApp(t)
	you := Settings.Accounts[0].Email
	core.RenderImage(b, func() { syncAll(t, a) }, core.Locked(b, func() {
		require.NoError(t, a.saveDraft(&SendMessage{Sender: MessageSender{From: SenderAccount(you)},
			To: []*gomail.Address{{Address: "zed@example.com"}}, Subject: "Unfinished", body: "Almost done"}))
		a.showMailbox(you, a.draftsLabel(you))
		var row int = -1
		for i := 0; a.rowAt(i) != nil; i++ {
			if a.rowAt(i).message.Subject == "Unfinished" {
				row = i
			}
		}
		require.GreaterOrEqual(t, row, 0, "in the drafts")
		a.list.SelectIndexEvent(row, events.SelectOne)
		a.composeMessage = nil
		a.list.Send(events.DoubleClick)
		require.NotNil(t, a.composeMessage, "written more")
		assert.Equal(t, "Unfinished", a.composeMessage.Subject)
		assert.Contains(t, a.composeMessage.body, "Almost done")
		assert.Empty(t, a.messageWindows, "not a window for reading it")
	}))
}

// TestHeaderSize tests that the header of a message shows its size, on
// the line of its date.
func TestHeaderSize(t *testing.T) {
	b, a := newTestApp(t, nil)
	longMessages(t, a, "one")
	core.RenderImage(b, core.Locked(b, func() {
		a.showMailbox(testEmail, "INBOX")
		a.list.SelectIndexEvent(0, events.SelectOne)
		require.NotNil(t, a.readMessage)
	}), core.Locked(b, func() {
		var date string
		a.readPanel.WalkDown(func(n tree.Node) bool {
			if n.AsTree().Name == "Date" {
				date = n.(*core.Text).Text
			}
			return true
		})
		size := messageSize(a.readMessage)
		require.Greater(t, size, datasize.Size(1000))
		assert.Contains(t, date, "<b>Size</b>")
		assert.Contains(t, date, size.String(), "in user-friendly units")
		assert.Contains(t, size.String(), "KB")
	}))
}

// TestMultiSelectMouse tests selecting several messages with the mouse
// (Shift and a click), which shows the first of them, and filing all of
// them at once, and that the File dialog chooses the mailbox filed into
// last at first, for filing more there with its File button.
func TestMultiSelectMouse(t *testing.T) {
	b, a := newTestingApp(t)
	you := Settings.Accounts[0].Email
	old := Settings.LastFiled
	t.Cleanup(func() { Settings.LastFiled = old })
	Settings.LastFiled = ""
	click := func(row int, mods key.Modifiers) func() {
		return func() {
			var at image.Point
			core.Locked(b, func() {
				for w := range a.list.AllWidgets() {
					if tf, ok := w.(*core.TextField); ok && tf.Text() == a.rowAt(row).From { // (read-only cells), in the tree column
						bb := tf.Geom.TotalBBox // on its text, as clicked
						at = image.Pt(bb.Min.X+bb.Dx()/3, (bb.Min.Y+bb.Max.Y)/2)
						break
					}
				}
			})()
			require.NotEqual(t, image.Point{}, at, "the row %d", row)
			src := b.Scene.RenderWindow().SystemWindow.Events()
			src.MouseButton(events.MouseDown, events.Left, at, mods)
			src.MouseButton(events.MouseUp, events.Left, at, mods)
		}
	}
	var shift key.Modifiers
	shift.SetFlag(true, key.Shift)
	var first, second *CacheMessage
	var chooser *core.Chooser
	dialog := func() *core.Body {
		d := core.TopScene(b)
		require.NotEqual(t, b.Scene, d, "the dialog")
		chooser = nil
		for w := range d.AllWidgets() {
			if ch, ok := w.(*core.Chooser); ok {
				chooser = ch
			}
		}
		require.NotNil(t, chooser)
		return d.Body
	}
	views := Settings.ListViews
	t.Cleanup(func() { Settings.ListViews = views })
	core.RenderImage(b, func() { syncAll(t, a) }, core.Locked(b, func() {
		// a flat list and all of the inboxes, then each conversation
		Settings.ListViews.SetFlag(false, ViewConversations)
		a.ShowAllInboxes()
		a.Update()
	}), click(0, 0), click(2, shift), core.Locked(b, func() {
		t.Log("inboxes", a.list.SelectedIndexes, len(a.selectedRows))
		assert.Len(t, a.selectedRows, 3, "three in a flat list of the inboxes")
		Settings.ListViews.SetFlag(true, ViewConversations)
		a.showMailbox(you, "INBOX")
		require.NotNil(t, a.rowAt(2), "at least 3 rows")
	}), click(0, 0), core.Locked(b, func() {
		a.updateMail() // as after marking it read, or a sync
	}), click(1, shift), core.Locked(b, func() {
		require.Len(t, a.selectedRows, 2, "two selected")
		// both shown as selected, with the background of a selected row
		// (in a read-only list, only with ReadOnlyMultiSelect)
		for row := range 2 {
			var cell core.Widget
			for _, k := range a.list.ListGrid.Children {
				if r, ok := k.AsTree().Property(core.ListRowProperty).(int); ok && r == row {
					cell = k.(core.Widget)
					break
				}
			}
			require.NotNil(t, cell)
			assert.Equal(t, colors.Scheme.Select.Container, a.list.ListGrid.ChildBackground(cell), "row %d shown selected", row)
		}
		a.updateMail() // as from the syncs, while selected
	}), core.Locked(b, func() {
		t.Log("after update", a.list.SelectedIndexes, a.list.SelectedIndex, len(a.selectedRows), len(a.rowsSelected()))
		require.Len(t, a.rowsSelected(), 2, "still two selected")
		assert.Equal(t, a.rowAt(0).message, a.readMessage, "the first one shown")
		first, second = a.rowAt(0).message, a.rowAt(1).message
		var note string
		a.readPanel.WalkDown(func(n tree.Node) bool {
			if n.AsTree().Name == "selected" {
				note = n.(*core.Text).Text
			}
			return true
		})
		assert.Contains(t, note, "2 selected")
		a.File()
	}), core.Locked(b, func() {
		dialog()
		chooser.SetCurrentValue(mailboxID{you, "Family"})
		chooser.SendChange()
	}), func() {
		require.Eventually(t, func() bool {
			a.AsyncLock()
			defer a.AsyncUnlock()
			return !inMailbox(first, "INBOX") && !inMailbox(second, "INBOX")
		}, 5*time.Second, 10*time.Millisecond, "both filed")
		assert.Equal(t, mailboxKey(you, "Family"), Settings.LastFiled, "remembered")
	}, click(0, 0), core.Locked(b, func() {
		require.NotNil(t, a.readMessage)
		first = a.readMessage
		a.File()
	}), core.Locked(b, func() {
		d := dialog()
		assert.Equal(t, mailboxID{you, "Family"}, chooser.CurrentItem.Value, "the last one at first")
		_ = d
	}), func() { // Return, in the chooser, which has the focus
		src := b.Scene.RenderWindow().SystemWindow.Events()
		src.KeyChord(0, key.CodeReturnEnter, 0)
	}, func() {
		require.Eventually(t, func() bool {
			a.AsyncLock()
			defer a.AsyncUnlock()
			return inMailbox(first, "Family")
		}, 5*time.Second, 10*time.Millisecond, "filed there again")
	})
}

// inMailbox returns whether the given message has the given label.
func inMailbox(cm *CacheMessage, label string) bool {
	return slices.ContainsFunc(cm.Labels, func(l Label) bool { return l.Name == label })
}

// TestDuplicateMailboxNames tests that labels with the names of special
// mailboxes on Gmail (as "Drafts", "Sent Messages", and "Deleted Messages"
// made by Apple Mail) are shown with their own names, as labels, not as
// second Drafts, Sent, and Trash mailboxes, and that the Drafts mailbox
// shows the number of drafts.
func TestDuplicateMailboxNames(t *testing.T) {
	b, a := newTestingApp(t)
	you := Settings.Accounts[0].Email
	for _, mb := range []string{"Drafts", "Sent Messages", "Deleted Messages"} {
		require.NoError(t, a.testServer.CreateMailbox(you, mb))
	}
	core.RenderImage(b, func() { syncAll(t, a) }, core.Locked(b, func() {
		require.NoError(t, a.saveDraft(&SendMessage{Sender: MessageSender{From: SenderAccount(you)}, Subject: "Unfinished"}))
	}), func() {
		a.countMessages(you) // with the lock of the window
	}, core.Locked(b, func() {
		a.Update()
	}), core.Locked(b, func() {
		var account *MailboxTree
		for _, n := range a.mailboxes.Children {
			if mt, ok := n.(*MailboxTree); ok && mt.account && mt.mailbox.email == you {
				account = mt
			}
		}
		require.NotNil(t, account)
		names := map[string]int{}
		icon := map[string]icons.Icon{}
		for _, n := range account.Children {
			mt := n.(*MailboxTree)
			names[mt.Text]++
			icon[mt.mailbox.label] = mt.Icon
		}
		for name, n := range names {
			assert.Equal(t, 1, n, "one %q: %v", name, names)
		}
		assert.Contains(t, names, "Sent")
		assert.Contains(t, names, "Trash")
		assert.Contains(t, names, "Sent Messages", "a label, by its own name")
		assert.Contains(t, names, "Deleted Messages")
		assert.Equal(t, icons.Label, icon["Sent Messages"])
		assert.Equal(t, icons.Label, icon["Drafts"], "a label")
		assert.Equal(t, icons.Draft, icon[a.draftsLabel(you)], "the Drafts mailbox")
		assert.Contains(t, names, withCount("Drafts", 1), "with the number of drafts: %v", names)
	}))
}

// TestFileDialogTyping tests that the mailbox filed into last starts
// selected in the File dialog, so that typing another replaces it.
func TestFileDialogTyping(t *testing.T) {
	b, a := newTestingApp(t)
	you := Settings.Accounts[0].Email
	old := Settings.LastFiled
	t.Cleanup(func() { Settings.LastFiled = old })
	Settings.LastFiled = mailboxKey(you, "Family")
	var tf *core.TextField
	core.RenderImage(b, func() { syncAll(t, a) }, core.Locked(b, func() {
		a.showMailbox(you, "INBOX")
		a.list.SelectIndexEvent(0, events.SelectOne)
		a.File()
	}), core.Locked(b, func() {
		d := core.TopScene(b)
		require.NotEqual(t, b.Scene, d, "the dialog")
		for w := range d.AllWidgets() {
			if ch, ok := w.(*core.Chooser); ok {
				tf = ch.TextField()
			}
		}
		require.NotNil(t, tf)
		assert.Equal(t, "Family", tf.Text())
	}), func() {
		b.Scene.RenderWindow().SystemWindow.Events().KeyChord('W', key.CodeW, 0)
	}, core.Locked(b, func() {
		assert.Equal(t, "W", tf.Text(), "typing replaces it")
	}))
}

// TestDeleteLabelWithSpecialName tests that labels with the names of
// special mailboxes that are not them (as "Sent Messages" of Apple Mail on
// Gmail) can be deleted, unlike the special mailboxes themselves, and that
// deleting one removes it from the server and the tree.
func TestDeleteLabelWithSpecialName(t *testing.T) {
	b, a := newTestingApp(t)
	you := Settings.Accounts[0].Email
	for _, mb := range []string{"Drafts", "Sent Messages", "Deleted Messages"} {
		require.NoError(t, a.testServer.CreateMailbox(you, mb))
	}
	core.RenderImage(b, func() { syncAll(t, a) }, core.Locked(b, func() {
		for _, mb := range []string{"Drafts", "Sent Messages", "Deleted Messages", "Family"} {
			assert.True(t, a.isUserMailbox(you, mb), "%s can be deleted", mb)
		}
		for _, mb := range []string{"INBOX", "[Gmail]/Sent Mail", "[Gmail]/Trash", "[Gmail]/Drafts", a.starredLabel(you), outboxLabel} {
			assert.False(t, a.isUserMailbox(you, mb), "%s cannot", mb)
		}
		a.DeleteMailbox(you, "Sent Messages")
	}), func() {
		require.Eventually(t, func() bool {
			a.AsyncLock()
			defer a.AsyncUnlock()
			return !slices.Contains(a.labels[you], "Sent Messages")
		}, 5*time.Second, 10*time.Millisecond, "gone")
		syncAll(t, a)
		a.AsyncLock()
		assert.NotContains(t, a.labels[you], "Sent Messages", "not on the server")
		assert.Contains(t, a.labels[you], "Deleted Messages")
		a.AsyncUnlock()
	})
}

// TestEditServerDraft tests editing a draft on the server, as one written
// in Gmail: double-clicking it opens it for writing, with its recipients,
// subject, and text, and once saved here, the draft on the server is moved
// to the trash, replaced by the draft of the app.
func TestEditServerDraft(t *testing.T) {
	b, a := newTestingApp(t)
	you := Settings.Accounts[0].Email
	var drafts string
	var server *CacheMessage
	core.RenderImage(b, func() {
		syncAll(t, a)
		a.AsyncLock()
		drafts = a.draftsLabel(you)
		a.AsyncUnlock()
		msg := "From: " + you + "\r\nTo: zed@example.com\r\nCc: amy@example.com\r\nSubject: Written in Gmail\r\nContent-Type: text/plain\r\n\r\nAlmost finished, from the web.\r\n"
		require.NoError(t, a.testServer.Append(you, []string{drafts}, []byte(msg), []imap.Flag{imap.FlagDraft, imap.FlagSeen}, time.Now()))
		syncAll(t, a)
	}, core.Locked(b, func() {
		a.showMailbox(you, drafts)
		row := -1
		for i := 0; a.rowAt(i) != nil; i++ {
			if a.rowAt(i).message.Subject == "Written in Gmail" {
				row = i
			}
		}
		require.GreaterOrEqual(t, row, 0, "in the drafts")
		server = a.rowAt(row).message
		assert.False(t, server.isLocal(), "on the server")
		assert.True(t, a.isServerDraft(server))
		a.list.SelectIndexEvent(row, events.SelectOne)
		a.composeMessage = nil
		a.list.Send(events.DoubleClick)
		sm := a.composeMessage
		require.NotNil(t, sm, "written more")
		assert.Empty(t, a.messageWindows, "not a window for reading it")
		assert.Equal(t, "Written in Gmail", sm.Subject)
		assert.Equal(t, []string{"zed@example.com"}, addresses(sm.To))
		assert.Equal(t, []string{"amy@example.com"}, addresses(sm.Cc))
		assert.Contains(t, sm.body, "Almost finished, from the web.")
		assert.Equal(t, SenderAccount(you), sm.Sender.From)
		// saved here (as when it is changed), which replaces it
		require.NoError(t, a.saveDraft(sm))
		a.removeServerDraft(sm)
	}), func() {
		trash := ""
		require.Eventually(t, func() bool {
			a.AsyncLock()
			defer a.AsyncUnlock()
			trash = a.trash[you]
			return !inMailbox(server, drafts) && inMailbox(server, trash)
		}, 5*time.Second, 10*time.Millisecond, "the server draft in the trash")
	})
}
