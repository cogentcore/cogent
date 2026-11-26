// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"cogentcore.org/cogent/mail/testserver"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/textcore"
	gomail "github.com/emersion/go-message/mail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// syncAll syncs all of the messages of all of the accounts.
func syncAll(t *testing.T, a *App) {
	t.Helper()
	for {
		a.syncMore = false
		require.NoError(t, a.cacheMessages())
		if !a.syncMore {
			return
		}
	}
}

// offlineServer returns the server configuration of the accounts of the
// test server that cannot be reached, as when offline.
func offlineServer() Server {
	return Server{IMAP: "127.0.0.1:1", SMTP: "127.0.0.1:1", Insecure: true, Password: testserver.Password}
}

// TestOffline tests starting the app offline: the downloaded mail is
// shown, with the mailboxes of the last sync, the status says that it is
// offline, and the actions on messages on the server are errors.
func TestOffline(t *testing.T) {
	b, a := newTestingApp(t)
	you := Settings.Accounts[0]
	dir := a.dataDirectory
	core.RenderImage(b, func() { syncAll(t, a) })
	n, labels := len(a.cache[you]), a.labels[you]
	require.NotZero(t, n)
	a.closeAccounts()

	// the app starts again, offline
	b2 := core.NewBody()
	a2 := NewApp(b2)
	a2.dataDirectory = dir
	for _, account := range Settings.Accounts {
		a2.servers[account] = offlineServer()
	}
	a2.currentEmail = you
	t.Cleanup(a2.closeStores)
	core.RenderImage(b2, func() {
		a2.openAllLocal()
		err := a2.cacheMessages()
		require.Error(t, err)
		assert.True(t, isOffline(err), err)
		a2.syncDone(err)
	}, core.Locked(b2, func() {
		assert.Len(t, a2.cache[you], n, "the downloaded mail")
		assert.Equal(t, labels, a2.labels[you], "its mailboxes")
		assert.True(t, a2.gmail[you])
		assert.NotEmpty(t, a2.trash[you])
		assert.Nil(t, a2.imapClient[you])
		var offline bool
		for w := range a2.statusBar.AllWidgets() {
			if bt, ok := w.(*core.Button); ok && bt.Text == "Offline" {
				offline = true
			}
		}
		assert.True(t, offline, "the status")
		a2.readMessage = findSubject(a2, you, a2.listCache[0].Subject, "INBOX")
		a2.readSingle = true
		a2.MarkAsUnread() // an error, not a panic
	}))
}

// TestDraftsOutbox tests saving a message as a draft, sending it offline,
// which puts it in the outbox until it can be sent, the drafts being kept
// between runs, editing it again, and discarding a draft.
func TestDraftsOutbox(t *testing.T) {
	b, a := newTestingApp(t)
	you, alice := Settings.Accounts[0], Settings.Accounts[1]
	online := a.servers[you]
	var sm *SendMessage
	var drafts string
	core.RenderImage(b, func() { syncAll(t, a) }, core.Locked(b, func() {
		sm = &SendMessage{From: []*gomail.Address{{Address: you}}, To: []*gomail.Address{{Address: alice}},
			Subject: "Plans", body: "Let's *go*"}
		require.NoError(t, a.saveDraft(sm))
		drafts = a.draftsLabel(you)
		assert.Equal(t, "[Gmail]/Drafts", drafts)
		cm := findSubject(a, you, "Plans", drafts)
		require.NotNil(t, cm, "in the drafts")
		assert.True(t, cm.isLocal())
		assert.FileExists(t, filepath.Join(a.messagesDir(you), cm.Filename))
		a.showMailbox(you, drafts)
	}), core.Locked(b, func() {
		assert.True(t, slices.ContainsFunc(a.listCache, func(cm *CacheMessage) bool { return cm.Subject == "Plans" }), "listed")

		// offline: it waits in the outbox
		a.servers[you] = offlineServer()
		_, err := a.queue(sm)
		require.NoError(t, err)
		assert.Nil(t, findSubject(a, you, "Plans", drafts))
		assert.NotNil(t, findSubject(a, you, "Plans", outboxLabel))
		assert.Contains(t, a.mailboxLabels(you), outboxLabel)
	}), func() {
		err := a.sendOutbox(you)
		assert.True(t, isOffline(err), err)
		n, refused := a.outboxCount()
		assert.Equal(t, 1, n)
		assert.Zero(t, refused)
		assert.NotNil(t, findSubject(a, you, "Plans", outboxLabel), "still there")

		// kept for the next run of the app
		a2 := NewApp(core.NewBody())
		theApp = a // not the new one
		a2.dataDirectory = a.dataDirectory
		a2.loadDrafts(you)
		n, _ = a2.outboxCount()
		assert.Equal(t, 1, n)
		assert.NotNil(t, findSubject(a2, you, "Plans", outboxLabel))
	}, core.Locked(b, func() {
		// edited again, it is back in the drafts
		a.readMessage = findSubject(a, you, "Plans", outboxLabel)
		a.readSingle = true
		a.EditDraft()
		assert.NotNil(t, findSubject(a, you, "Plans", drafts))
		n, _ := a.outboxCount()
		assert.Zero(t, n)
		assert.Same(t, sm.draft, a.composeMessage.draft)
		_, err := a.queue(a.composeMessage)
		require.NoError(t, err)
	}), func() {
		// online again: sent in the next sync
		a.servers[you] = online
		syncAll(t, a)
		n, _ := a.outboxCount()
		assert.Zero(t, n)
		assert.Nil(t, findSubject(a, you, "Plans", outboxLabel))
		sent := findSubject(a, you, "Plans", "[Gmail]/Sent Mail")
		require.NotNil(t, sent, "on the server")
		assert.False(t, sent.isLocal())
		assert.NotNil(t, findSubject(a, alice, "Plans", "INBOX"))
		files, _ := os.ReadDir(a.localDir(you))
		assert.Empty(t, files, "its files")
	}, core.Locked(b, func() {
		other := &SendMessage{From: []*gomail.Address{{Address: you}}, Subject: "Never mind"}
		require.NoError(t, a.saveDraft(other))
		a.readMessage = findSubject(a, you, "Never mind", drafts)
		a.readSingle = true
		a.Delete()
		assert.Nil(t, findSubject(a, you, "Never mind", drafts), "discarded")
		assert.Nil(t, a.readMessage)
	}))
}

// TestComposeDraft tests that the message being written in a compose
// window is saved as a draft when the window is closed, once it is
// changed, and not if it is not.
func TestComposeDraft(t *testing.T) {
	b, a := newTestingApp(t)
	you := Settings.Accounts[0]
	locals := func() []*CacheMessage {
		var res []*CacheMessage
		for _, cm := range a.cache[you] {
			if cm.isLocal() {
				res = append(res, cm)
			}
		}
		return res
	}
	var cb *core.Body
	core.RenderImage(b, func() { syncAll(t, a) }, core.Locked(b, func() {
		a.composeMessage = &SendMessage{To: []*gomail.Address{{}}}
		cb = a.compose("Compose")
	}), core.Locked(b, func() {
		core.Locked(cb, func() { cb.Scene.Close() })()
	}), core.Locked(b, func() {
		assert.Empty(t, locals(), "not saved without changes")
		a.composeMessage = &SendMessage{To: []*gomail.Address{{}}}
		cb = a.compose("Compose")
	}), func() {
		core.Locked(cb, func() {
			var ed *textcore.Editor
			for w := range cb.AllWidgets() {
				if e, ok := w.(*textcore.Editor); ok {
					ed = e
				}
			}
			require.NotNil(t, ed)
			ed.Lines.SetString("Back *soon*")
			ed.Send(events.Input)
			cb.Scene.Close()
		})()
	}, func() {
		// saved in the background, with the lock of the app
		require.Eventually(t, func() bool {
			a.AsyncLock()
			defer a.AsyncUnlock()
			return len(locals()) > 0
		}, 5*time.Second, 10*time.Millisecond)
	}, core.Locked(b, func() {
		got := locals()
		require.Len(t, got, 1, "saved")
		d := a.draftOf(got[0])
		require.NotNil(t, d)
		assert.Equal(t, "Back *soon*", strings.TrimSpace(d.Body))
		assert.False(t, d.Queued)
	}))
}

// TestStartupOpen tests that the accounts are open in the tree of
// mailboxes when the app starts with the mail of an earlier run,
// showing their mailboxes.
func TestStartupOpen(t *testing.T) {
	b, a := newTestingApp(t)
	you := Settings.Accounts[0]
	dir := a.dataDirectory
	core.RenderImage(b, func() { syncAll(t, a) })
	a.closeAccounts()

	b2 := core.NewBody()
	a2 := NewApp(b2)
	a2.dataDirectory = dir
	for _, account := range Settings.Accounts {
		a2.servers[account] = offlineServer()
	}
	a2.currentEmail = you
	t.Cleanup(a2.closeStores)
	opened := make(chan struct{})
	b2.OnShow(func(e events.Event) { // as GetMail does
		go func() {
			a2.openAllLocal()
			close(opened)
		}()
	})
	core.RenderImage(b2, func() {
		<-opened
	}, core.Locked(b2, func() {
		require.Len(t, a2.mailboxes.Children, len(Settings.Accounts))
		for _, c := range a2.mailboxes.Children {
			mt := c.(*MailboxTree)
			assert.False(t, mt.Closed, mt.mailbox.email)
			require.NotEmpty(t, mt.Children, mt.mailbox.email)
			assert.False(t, mt.Children[0].(*MailboxTree).StateIs(states.Invisible), mt.mailbox.email)
		}
	}))
}
