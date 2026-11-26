// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"reflect"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAttachmentCounts tests counting the attachments of the messages when
// they are downloaded, or after for those downloaded before, and showing
// them in the list, empty for none.
func TestAttachmentCounts(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	old := Settings.ListViews
	t.Cleanup(func() { Settings.ListViews = old })
	Settings.ListViews.SetFlag(true, ViewAttachments)
	Settings.ListViews.SetFlag(true, ViewConversations)
	var hello, reply *CacheMessage
	core.RenderImage(b, func() {
		require.NoError(t, a.cacheMessages())
	}, core.Locked(b, func() {
		hello = messageBySubject(a, "Hello")
		reply = messageBySubject(a, "Re: Hello")
		assert.True(t, hello.AttachmentsCounted, "when downloaded")
		assert.Equal(t, 1, hello.Attachments)
		assert.True(t, reply.AttachmentsCounted)
		assert.Equal(t, 0, reply.Attachments)
		a.showMailbox(testEmail, "INBOX")
		assert.True(t, a.listField(fieldOf(t, "Attachments")))
		var conv *messageRow
		for _, mr := range a.listRows {
			if mr.message == hello {
				conv = mr
			}
		}
		require.NotNil(t, conv)
		assert.Equal(t, "1", conv.Attachments, "of the conversation")
		for _, mr := range conv.Children {
			if mr.message == reply {
				assert.Empty(t, mr.Attachments, "none")
			}
		}

		// downloaded before they were counted
		hello.Attachments, hello.AttachmentsCounted = 0, false
		a.indexChecked = map[string]map[string]bool{}
	}), func() {
		require.NoError(t, a.indexMissing(testEmail, 100))
	}, core.Locked(b, func() {
		assert.True(t, hello.AttachmentsCounted, "counted after")
		assert.Equal(t, 1, hello.Attachments)
		stored, err := a.stores[testEmail].load()
		require.NoError(t, err)
		assert.Equal(t, 1, stored[hello.key].Attachments, "saved")

		Settings.ListViews.SetFlag(false, ViewAttachments)
		assert.False(t, a.listField(fieldOf(t, "Attachments")), "not shown")
		list := a.list
		a.Update()
		assert.NotSame(t, list, a.list, "a new list without the column")
		list = a.list
		Settings.ListViews.SetFlag(true, ViewAttachments)
		a.Update()
		assert.NotSame(t, list, a.list, "a new list with the column, when it is switched on")
		assert.Equal(t, icons.AttachFile, a.list.HeaderIcons["Attachments"], "a paperclip in its header")
	}))
}

// fieldOf returns the field of the rows of the list with the given name.
func fieldOf(t *testing.T, name string) reflect.StructField {
	f, ok := reflect.TypeFor[messageRow]().FieldByName(name)
	require.True(t, ok, name)
	return f
}
