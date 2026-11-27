// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events/key"
	"github.com/cogentcore/go-imap/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStar tests starring a message, which is shown right away, in the
// Starred mailbox and with a star in the list, and sets its \Flagged flag
// on the server, and unstarring it with Command+Shift+L.
func TestStar(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	var report *CacheMessage
	serverStarred := func() bool {
		uid := report.Labels[0].UID
		return slices.Contains(serverFlags(t, a, report.Labels[0].Name, uid), imap.FlagFlagged)
	}
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
	}, core.Locked(b, func() {
		report = messageBySubject(a, "Report")
		assert.Equal(t, "Starred", a.starredLabel(testEmail), "a local one, without one on the server")
		assert.Contains(t, a.mailboxLabels(testEmail), "Starred")
		a.showMailbox(testEmail, "Work")
		a.readMessage, a.readSingle = report, true
		a.Star()
		assert.True(t, report.isStarred(), "right away")
		var row *messageRow
		for _, mr := range a.listRows {
			if mr.message == report {
				row = mr
			}
		}
		require.NotNil(t, row)
		assert.True(t, bool(row.Starred), "with a star in the list")
		fields := reflect.VisibleFields(reflect.TypeFor[messageRow]())
		var shown []string
		for _, f := range fields {
			if f.IsExported() && f.Tag.Get("display") != "-" {
				shown = append(shown, f.Name)
			}
		}
		assert.Equal(t, "Starred", shown[len(shown)-1], "the last column")
		assert.Equal(t, "From", shown[1], "the first column after the compact summary")
		a.showMailbox(testEmail, "Starred")
		assert.True(t, listed(a, report), "in the Starred mailbox")
		a.countUnread()
		assert.Equal(t, 1, a.unread[testEmail]["Starred"], "unread")
	}), func() {
		require.Eventually(t, serverStarred, 5*time.Second, 10*time.Millisecond, "flagged on the server")
	}, core.Locked(b, func() {
		a.readMessage, a.readSingle = report, true
		a.list.SetFocus()
	}), func() {
		r, code, mods, err := key.Chord("Command+Shift+L").PlatformChord().Decode()
		require.NoError(t, err)
		b.Scene.RenderWindow().SystemWindow.Events().KeyChord(r, code, mods)
		require.Eventually(t, func() bool { return !serverStarred() }, 5*time.Second, 10*time.Millisecond, "unstarred")
	}, core.Locked(b, func() {
		assert.False(t, report.isStarred())
		a.Update()
		assert.False(t, listed(a, report), "out of the Starred mailbox")
	}))
}
