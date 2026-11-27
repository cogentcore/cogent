// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"path/filepath"
	"testing"

	"github.com/cogentcore/go-imap/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageStore(t *testing.T) {
	fn := filepath.Join(t.TempDir(), "store", "messages.db")
	ms, err := openStore(fn)
	require.NoError(t, err)
	cm := &CacheMessage{Envelope: imap.Envelope{Subject: "Hello", MessageID: "1@x"}, Filename: "f1",
		Flags: []imap.Flag{imap.FlagSeen}, Labels: []Label{{"INBOX", 3}}, GmailMsgID: 42}
	require.NoError(t, ms.put(map[string]*CacheMessage{"1@x": cm, "2@x": {Filename: "f2"}}))
	require.NoError(t, ms.setMailbox("INBOX", mailboxState{UIDValidity: 7, UIDNext: 4, HighestModSeq: 99}))
	require.NoError(t, ms.Close())

	ms, err = openStore(fn)
	require.NoError(t, err)
	defer ms.Close()
	all, err := ms.load()
	require.NoError(t, err)
	require.Len(t, all, 2)
	assert.Equal(t, "Hello", all["1@x"].Subject)
	assert.Equal(t, []Label{{"INBOX", 3}}, all["1@x"].Labels)
	assert.Equal(t, uint64(42), all["1@x"].GmailMsgID)
	st, err := ms.mailbox("INBOX")
	require.NoError(t, err)
	assert.Equal(t, mailboxState{UIDValidity: 7, UIDNext: 4, HighestModSeq: 99}, st)
	st, err = ms.mailbox("Other")
	require.NoError(t, err)
	assert.Zero(t, st)

	require.NoError(t, ms.remove("2@x"))
	all, err = ms.load()
	require.NoError(t, err)
	assert.Len(t, all, 1)
}
