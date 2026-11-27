// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"slices"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// listed returns whether the given message is in the list of messages.
func listed(a *App, cm *CacheMessage) bool {
	return slices.ContainsFunc(a.listCache, func(m *CacheMessage) bool {
		return m == cm || slices.Contains(m.replies, cm)
	})
}

// TestDeleteRightAway tests that a deleted message leaves the list right
// away, before the server moves it to the trash, that it is back if that
// fails, and that the Delete key function of the key map, as Control+D in
// the Emacs key map, deletes.
func TestDeleteRightAway(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	old := keymap.ActiveMapName // after the settings are applied
	keymap.SetActiveMapName("MacEmacs")
	t.Cleanup(func() { keymap.SetActiveMapName(old) })
	var report *CacheMessage
	inTrash := func() bool {
		a.AsyncLock()
		defer a.AsyncUnlock()
		return slices.Contains(labelNames(report), "[Gmail]/Trash")
	}
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
	}, core.Locked(b, func() {
		report = messageBySubject(a, "Report")
		a.showMailbox(testEmail, "Work")
		require.True(t, listed(a, report))
		a.readMessage, a.readSingle = report, true

		// not connected: an error, and back in the list
		c := a.imapClient[testEmail]
		a.imapClient[testEmail] = nil
		a.Delete()
		a.imapClient[testEmail] = c
		assert.True(t, listed(a, report), "back")
		assert.Empty(t, a.pendingLabels)

		a.Delete()
		assert.False(t, listed(a, report), "gone right away, with the lock still held")
	}), func() {
		require.Eventually(t, func() bool {
			if !inTrash() {
				return false
			}
			a.AsyncLock()
			defer a.AsyncUnlock()
			return len(a.pendingLabels) == 0
		}, 5*time.Second, 10*time.Millisecond, "moved on the server")
		a.AsyncLock()
		assert.Empty(t, a.pendingLabels)
		assert.False(t, listed(a, report))
		a.AsyncUnlock()
	}, core.Locked(b, func() {
		a.showMailbox(testEmail, "INBOX")
		hello := messageBySubject(a, "Hello")
		require.True(t, listed(a, hello))
		a.list.SetFocus()
		a.readMessage, a.readSingle = hello, true
		report = hello
		assert.Equal(t, keymap.Delete, keymap.Of("Control+D"))
		sc, _, ok := shortcutFor("Control+D")
		assert.True(t, ok)
		assert.Equal(t, KeyDelete, sc.kf)
	}), func() {
		r, code, mods, err := key.Chord("Control+D").Decode()
		require.NoError(t, err)
		b.Scene.RenderWindow().SystemWindow.Events().KeyChord(r, code, mods)
		require.Eventually(t, inTrash, 5*time.Second, 10*time.Millisecond, "deleted with Control+D")
	})
}
