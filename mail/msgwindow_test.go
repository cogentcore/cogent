// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
	"github.com/cogentcore/go-imap/v2"
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
