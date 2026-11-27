// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"cogentcore.org/core/events"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"github.com/cogentcore/go-imap/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStripAttachments tests removing the attachments of a message: each
// is replaced with a note of its name and size, and the header and the
// other parts are kept, in nested parts too.
func TestStripAttachments(t *testing.T) {
	hello := testMessages[0].text
	res, removed, err := stripAttachments([]byte(hello))
	require.NoError(t, err)
	assert.Equal(t, []string{"notes.txt"}, removed)
	s := string(res)
	header, _ := splitHeader([]byte(hello))
	assert.True(t, strings.HasPrefix(s, string(header)), "the same header")
	assert.Contains(t, s, "<p>Hi <b>there</b></p>")
	assert.Contains(t, s, "Hi there\r\nhow are you")
	assert.NotContains(t, s, "some notes")
	assert.Contains(t, s, "X-Attachment-Removed: notes.txt")
	assert.Contains(t, s, "[Attachment removed: notes.txt (10 bytes)]")

	plain := testMessages[1].text
	res, removed, err = stripAttachments([]byte(plain))
	require.NoError(t, err)
	assert.Empty(t, removed)
	assert.Equal(t, plain, string(res), "no attachments: as it is")

	nested := "Subject: x\r\nContent-Type: multipart/mixed; boundary=out\r\n\r\n" +
		"--out\r\nContent-Type: multipart/related; boundary=in\r\n\r\n" +
		"--in\r\nContent-Type: text/html\r\n\r\n<p>x</p>\r\n" +
		"--in\r\nContent-Type: image/png; name=\"=?utf-8?q?pic=C3=A9.png?=\"\r\nContent-Transfer-Encoding: base64\r\n\r\naGVsbG8gd29ybGQ=\r\n--in--\r\n" +
		"--out\r\nContent-Type: application/pdf\r\nContent-Disposition: attachment; filename=\"a b.pdf\"\r\n\r\n%PDF\r\n--out--\r\n"
	res, removed, err = stripAttachments([]byte(nested))
	require.NoError(t, err)
	assert.Equal(t, []string{"picé.png", "a b.pdf"}, removed)
	assert.Contains(t, string(res), "[Attachment removed: picé.png (11 bytes)]")
	assert.Contains(t, string(res), "<p>x</p>")
	assert.NotContains(t, string(res), "aGVsbG8")
}

// TestRemoveAttachments tests removing the attachments of a message on a
// server other than Gmail: it is replaced in each of its mailboxes, and
// shown without them, with the same key after syncing again.
func TestRemoveAttachments(t *testing.T) {
	c := newIMAP(t)
	b, a := newTestApp(t, c)
	var hello *CacheMessage
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
		hello = messageBySubject(a, "Hello")
		require.NotEmpty(t, hello.Filename)
		assert.Equal(t, 1, hello.Attachments)
		old := hello.Filename
		oldLabels := slices.Clone(hello.Labels)
		require.Len(t, oldLabels, 2, "INBOX and Work")

		require.NoError(t, a.removeAttachments(c, testEmail, hello))
		assert.Equal(t, 0, hello.Attachments)
		assert.NotEqual(t, old, hello.Filename)
		assert.NoFileExists(t, filepath.Join(a.messagesDir(testEmail), old))
		data, err := os.ReadFile(filepath.Join(a.messagesDir(testEmail), hello.Filename))
		require.NoError(t, err)
		assert.NotContains(t, string(data), "some notes")
		for i, l := range hello.Labels {
			assert.Equal(t, oldLabels[i].Name, l.Name)
			assert.NotEqual(t, oldLabels[i].UID, l.UID, "a new UID")
		}

		// on the server
		for _, mb := range []string{"INBOX", "Work"} {
			_, err := c.Select(mb, nil).Wait()
			require.NoError(t, err)
			msgs, err := c.Fetch(imap.SeqSetNum(1, 2), &imap.FetchOptions{BodySection: []*imap.FetchItemBodySection{{Peek: true}}}).Collect()
			require.NoError(t, err)
			n := 0
			for _, m := range msgs {
				if s := string(m.BodySection[0].Bytes); strings.Contains(s, "Subject: Hello") {
					n++
					assert.NotContains(t, s, "some notes", mb)
					assert.Contains(t, s, "[Attachment removed: notes.txt", mb)
				}
			}
			assert.Equal(t, 1, n, mb)
		}
		require.NoError(t, c.Unselect().Wait())
		a.setSelected(testEmail, "")

		// syncing again: the same message
		require.NoError(t, a.cacheMessages())
		assert.Same(t, hello, messageBySubject(a, "Hello"))
		assert.Len(t, hello.Labels, 2)
	}, core.Locked(b, func() {
		a.readMessage = hello
		a.readSingle = true
		a.updateReading()
	}), core.Locked(b, func() {
		assert.Equal(t, []string{"notes.txt"}, hello.parsed.removed, "shown as removed")
		assert.Empty(t, hello.parsed.attachments)
	}))
}

// TestRemoveAttachmentsGmail tests removing the attachments of a message on
// the Gmail-like test server: the old one is deleted, and the new one is
// in All Mail with the same labels and flags, with a new Gmail ID (key).
func TestRemoveAttachmentsGmail(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	old := *Settings
	t.Cleanup(func() { *Settings = old })
	t.Cleanup(core.TrackWindows())
	b := core.NewBody()
	a := NewApp(b)
	require.NoError(t, a.StartTesting(TestingOptions{Seed: 1}))
	t.Cleanup(a.testServer.Close)
	t.Cleanup(a.closeStores)
	you := Settings.Accounts[0].Email
	syncAll := func() {
		for {
			a.syncMore = false
			require.NoError(t, a.syncAccount(you))
			if !a.syncMore {
				return
			}
		}
	}
	core.RenderImage(b, func() {
		syncAll()
		var cm *CacheMessage
		a.AsyncLock()
		for _, m := range a.cache[you] {
			if m.Attachments > 0 && m.Filename != "" && len(m.Labels) > 1 && (cm == nil || m.MessageID < cm.MessageID) {
				cm = m
			}
		}
		a.AsyncUnlock()
		require.NotNil(t, cm, "a message with attachments and labels")
		oldKey, n := cm.key, cm.Attachments
		var names []string
		for _, l := range cm.Labels {
			names = append(names, l.Name)
		}
		flags := slices.Clone(cm.Flags)

		a.imapMu[you].Lock()
		err := a.removeAttachments(a.imapClient[you], you, cm)
		a.imapMu[you].Unlock()
		require.NoError(t, err)
		assert.NotEqual(t, oldKey, cm.key, "a new Gmail ID")
		assert.Nil(t, a.cache[you][oldKey])
		assert.Same(t, cm, a.cache[you][cm.key])
		data, err := os.ReadFile(filepath.Join(a.messagesDir(you), cm.Filename))
		require.NoError(t, err)
		assert.Equal(t, n, strings.Count(string(data), removedHeader+":"))
		require.NoError(t, a.saveMessages(you, cm))

		syncAll()
		var same []*CacheMessage
		a.AsyncLock()
		for _, m := range a.cache[you] {
			if m.MessageID == cm.MessageID {
				same = append(same, m)
			}
		}
		a.AsyncUnlock()
		require.Len(t, same, 1, "only the new one")
		assert.Same(t, cm, same[0])
		var after []string
		for _, l := range cm.Labels {
			after = append(after, l.Name)
		}
		assert.ElementsMatch(t, names, after, "the same labels")
		assert.ElementsMatch(t, flags, cm.Flags, "the same flags")
		assert.Equal(t, 0, cm.Attachments)
	})
}

// TestRemoveAttachmentsSelected tests removing the attachments of several
// selected messages at once, of two accounts in all of the inboxes, each
// in its own account.
func TestRemoveAttachmentsSelected(t *testing.T) {
	b, a := newTestingApp(t)
	you, alice := Settings.Accounts[0].Email, Settings.Accounts[1].Email
	withAttachment := func(to, subject string) []byte {
		return []byte("From: zed@example.com\r\nTo: " + to + "\r\nSubject: " + subject +
			"\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=XX\r\n\r\n" +
			"--XX\r\nContent-Type: text/plain\r\n\r\nSee the notes.\r\n" +
			"--XX\r\nContent-Type: text/plain\r\nContent-Disposition: attachment; filename=notes.txt\r\n\r\nsome secret notes\r\n--XX--\r\n")
	}
	var msgs []*CacheMessage
	views := Settings.ListViews
	t.Cleanup(func() { Settings.ListViews = views })
	core.RenderImage(b, func() {
		require.NoError(t, a.testServer.Append(you, []string{"INBOX"}, withAttachment(you, "Notes for you"), nil, time.Now()))
		require.NoError(t, a.testServer.Append(alice, []string{"INBOX"}, withAttachment(alice, "Notes for Alice"), nil, time.Now()))
		syncAll(t, a)
	}, core.Locked(b, func() {
		Settings.ListViews.SetFlag(false, ViewConversations) // each message on its own
		a.ShowAllInboxes()
		mode := events.SelectOne
		for i := 0; a.rowAt(i) != nil; i++ {
			if s := a.rowAt(i).message.Subject; s == "Notes for you" || s == "Notes for Alice" {
				a.list.SelectIndexEvent(i, mode)
				mode = events.ExtendOne
				msgs = append(msgs, a.rowAt(i).message)
			}
		}
		require.Len(t, msgs, 2)
		require.Len(t, a.selectedRows, 2)
		for _, cm := range msgs {
			require.NotEmpty(t, cm.Filename, "downloaded")
			require.Equal(t, 1, cm.Attachments)
		}
		a.RemoveAttachments()
	}), core.Locked(b, func() {
		d := core.TopScene(b)
		require.NotEqual(t, b.Scene, d, "asking first")
		bt := core.ButtonByText(d, "Remove")
		require.NotNil(t, bt)
		bt.Send(events.Click)
	}), func() {
		require.Eventually(t, func() bool {
			a.AsyncLock()
			defer a.AsyncUnlock()
			return msgs[0].Attachments == 0 && msgs[1].Attachments == 0
		}, 10*time.Second, 10*time.Millisecond, "both, in their accounts")
		for _, cm := range msgs {
			data, err := os.ReadFile(filepath.Join(a.messagesDir(cm.account), cm.Filename))
			require.NoError(t, err)
			assert.NotContains(t, string(data), "some secret notes", cm.Subject)
			assert.Contains(t, string(data), "Attachment removed: notes.txt", cm.Subject)
		}
	})
}
