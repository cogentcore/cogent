// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testserver

import (
	"slices"
	"testing"
	"time"

	"github.com/cogentcore/go-imap/v2"
	"github.com/cogentcore/go-imap/v2/imapclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fetchAll returns the messages of the given mailbox, with their UIDs,
// flags, modseqs, and Gmail items.
func fetchAll(t *testing.T, c *imapclient.Client, mailbox string) []*imapclient.FetchMessageBuffer {
	t.Helper()
	_, err := c.Select(mailbox, &imap.SelectOptions{CondStore: true}).Wait()
	require.NoError(t, err)
	msgs, err := c.Fetch(imap.UIDSet{{Start: 1, Stop: 0}}, &imap.FetchOptions{UID: true, Flags: true, ModSeq: true,
		Envelope: true, GmailMsgID: true, GmailThreadID: true, GmailLabels: true}).Collect()
	require.NoError(t, err)
	return msgs
}

func TestGmail(t *testing.T) {
	s := newServer(t, Options{})
	you := DefaultAccounts[0]
	c := login(t, s, you)
	assert.True(t, c.Caps().Has(imap.CapGmailExt1))
	assert.True(t, c.Caps().Has(imap.CapCondStore))
	msg := "From: a@x.test\r\nSubject: one\r\nMessage-ID: <one@x.test>\r\n\r\nHello\r\n"
	reply := "From: b@x.test\r\nSubject: Re: one\r\nMessage-ID: <two@x.test>\r\nIn-Reply-To: <one@x.test>\r\n\r\nHi\r\n"
	require.NoError(t, s.Append(you, []string{"INBOX", "Work"}, []byte(msg), nil, time.Time{}))
	require.NoError(t, s.Append(you, []string{"INBOX"}, []byte(reply), nil, time.Time{}))

	all := fetchAll(t, c, "[Gmail]/All Mail")
	require.Len(t, all, 2, "once each")
	one, two := all[0], all[1]
	assert.ElementsMatch(t, []string{`\Inbox`, "Work"}, one.GmailLabels)
	assert.NotZero(t, one.GmailMsgID)
	assert.NotEqual(t, one.GmailMsgID, two.GmailMsgID)
	assert.Equal(t, one.GmailThreadID, two.GmailThreadID, "the reply is in its thread")
	assert.Equal(t, 1, count(t, c, "Work"))

	// the labels change with STORE X-GM-LABELS, in one message
	_, err := c.Select("[Gmail]/All Mail", nil).Wait()
	require.NoError(t, err)
	require.NoError(t, c.StoreGmailLabels(imap.UIDSetNum(one.UID), &imap.StoreGmailLabels{
		Op: imap.StoreFlagsAdd, Silent: true, Labels: []string{"Projects/Alpha"}}).Close())
	require.NoError(t, c.StoreGmailLabels(imap.UIDSetNum(one.UID), &imap.StoreGmailLabels{
		Op: imap.StoreFlagsDel, Silent: true, Labels: []string{`\Inbox`, "Work"}}).Close())
	assert.Equal(t, 0, count(t, c, "Work"))
	assert.Equal(t, 1, count(t, c, "INBOX"), "archived, the reply stays")
	assert.Equal(t, 1, count(t, c, "Projects/Alpha"))

	// the flags are of the message, in all of its mailboxes
	_, err = c.Select("Projects/Alpha", nil).Wait()
	require.NoError(t, err)
	require.NoError(t, c.Store(imap.UIDSetNum(1), &imap.StoreFlags{Op: imap.StoreFlagsAdd, Silent: true, Flags: []imap.Flag{imap.FlagSeen}}, nil).Close())

	// the changes since the modseq of before
	sd, err := c.Select("[Gmail]/All Mail", &imap.SelectOptions{CondStore: true}).Wait()
	require.NoError(t, err)
	assert.Greater(t, sd.HighestModSeq, two.ModSeq)
	changed, err := c.Fetch(imap.UIDSet{{Start: 1, Stop: 0}}, &imap.FetchOptions{UID: true, Flags: true, GmailLabels: true,
		ChangedSince: two.ModSeq}).Collect()
	require.NoError(t, err)
	require.Len(t, changed, 1, "only the one that changed")
	assert.Equal(t, one.UID, changed[0].UID)
	assert.Equal(t, []imap.Flag{imap.FlagSeen}, changed[0].Flags)
	assert.Equal(t, []string{"Projects/Alpha"}, changed[0].GmailLabels)

	// to the Trash: out of All Mail and the labels, and back
	md, err := c.Move(imap.UIDSetNum(one.UID), "[Gmail]/Trash").Wait()
	require.NoError(t, err)
	require.NotNil(t, md)
	assert.Len(t, fetchAll(t, c, "[Gmail]/All Mail"), 1)
	assert.Equal(t, 0, count(t, c, "Projects/Alpha"))
	trash := fetchAll(t, c, "[Gmail]/Trash")
	require.Len(t, trash, 1)
	assert.Equal(t, one.GmailMsgID, trash[0].GmailMsgID, "the same message")
	_, err = c.Move(imap.UIDSetNum(trash[0].UID), "INBOX").Wait()
	require.NoError(t, err)
	assert.Len(t, fetchAll(t, c, "[Gmail]/All Mail"), 2, "in All Mail again")
	assert.Equal(t, 0, count(t, c, "[Gmail]/Trash"))

	// the special-use mailboxes
	list, err := c.List("", "*", nil).Collect()
	require.NoError(t, err)
	attrs := map[string][]imap.MailboxAttr{}
	for _, l := range list {
		attrs[l.Mailbox] = l.Attrs
	}
	assert.Contains(t, attrs["[Gmail]/All Mail"], imap.MailboxAttrAll)
	assert.Contains(t, attrs["[Gmail]/Sent Mail"], imap.MailboxAttrSent)
	assert.Contains(t, attrs["[Gmail]"], imap.MailboxAttrNoSelect)
	assert.True(t, slices.ContainsFunc(list, func(l *imap.ListData) bool { return l.Mailbox == "Projects" }), "the parent label")
}
