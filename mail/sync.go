// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"cmp"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cogentcore/go-imap/v2"
	"github.com/cogentcore/go-imap/v2/imapclient"
)

// The syncing of the messages of an account with its server keeps a local
// copy of all of them: their metadata in a [messageStore], their contents
// in files (see [App.messagesDir]), and their text in a [searchIndex].
//
// On Gmail (with the X-GM-EXT-1 extension), each message is synced once,
// from [Gmail]/All Mail, or from the Trash or Spam, which All Mail does
// not have, with its Gmail labels (X-GM-LABELS), which are the mailboxes
// it is in. On other servers, each mailbox is synced, and the messages
// in more than one are the same message, by their Message-IDs.
//
// Each sync of a mailbox gets only what changed since the last one: the
// new messages, after the UIDNEXT of the last sync, the changes of the
// flags (and labels) since its HIGHESTMODSEQ, with CONDSTORE, and the
// messages that are gone, if its number of messages differs. The contents
// of the messages are downloaded after, the newest first, a number of them
// in each sync.

// gmailKey returns the key in the cache of the message with the given
// Gmail message ID.
func gmailKey(id uint64) string {
	return "gm-" + strconv.FormatUint(id, 16)
}

// bodiesPerSync is the number of contents of messages downloaded in each
// sync of an account, the newest first.
var bodiesPerSync = 200

// openAccount opens the store and search index of the given account,
// loading its messages into the cache, and connects to its server, if it
// has not already.
func (a *App) openAccount(email string) error {
	if a.stores[email] == nil {
		if err := a.openLocal(email); err != nil {
			return err
		}
	}
	if a.imapClient[email] != nil {
		return nil
	}
	c, err := a.dialIMAP(email)
	if err != nil {
		return fmt.Errorf("connecting: %w", err)
	}
	a.setClient(email, c)
	return nil
}

// openLocal opens the store and search index of the given account,
// loading its messages into the cache.
func (a *App) openLocal(email string) error {
	if err := os.MkdirAll(a.messagesDir(email), 0o700); err != nil {
		return err
	}
	st, err := openStore(a.storeFile(email))
	if err != nil {
		return err
	}
	si, err := openSearch(a.searchDir(email))
	if err != nil {
		st.Close()
		return err
	}
	cached, err := st.load()
	if err != nil {
		st.Close()
		si.Close()
		return err
	}
	for k, cm := range cached {
		cm.key = k
	}
	info, err := st.info()
	if err != nil {
		st.Close()
		si.Close()
		return err
	}
	a.AsyncLock()
	defer a.AsyncUnlock()
	if a.totalMessages[email] == nil {
		a.totalMessages[email] = map[string]int{}
	}
	a.stores[email] = st
	a.searches[email] = si
	a.cache[email] = cached
	a.setAccountInfo(email, info)
	a.loadDrafts(email)
	return nil
}

// setClient sets the given IMAP client, which is logged in, as that of
// the given account, with the extensions of its server.
func (a *App) setClient(email string, c *imapclient.Client) {
	caps := c.Caps()
	a.AsyncLock()
	defer a.AsyncUnlock()
	a.gmail[email] = caps.Has(imap.CapGmailExt1)
	a.condStore[email] = caps.Has(imap.CapCondStore)
	a.imapClient[email] = c
	a.imapMu[email] = &sync.Mutex{}
}

// closeAccounts closes the connections, stores, and search indexes of
// all of the accounts, as when the app quits.
func (a *App) closeAccounts() {
	for email, c := range a.imapClient {
		if mu := a.imapMu[email]; mu != nil {
			mu.Lock()
		}
		c.Logout().Wait()
		c.Close()
	}
	a.closeStores()
}

// closeStores closes the stores and search indexes of all of the accounts.
func (a *App) closeStores() {
	for email, st := range a.stores {
		st.Close()
		delete(a.stores, email)
	}
	for email, si := range a.searches {
		si.Close()
		delete(a.searches, email)
	}
}

// syncAccount syncs the messages of the given account with the server.
func (a *App) syncAccount(email string) error {
	if err := a.openAccount(email); err != nil {
		return err
	}
	c := a.imapClient[email]
	mu := a.imapMu[email]
	mu.Lock()
	mailboxes, err := a.listMailboxes(c, email)
	mu.Unlock()
	if err != nil {
		return err
	}
	a.AsyncLock()
	err = a.saveAccountInfo(email)
	a.AsyncUnlock()
	if err != nil {
		return err
	}
	var sync []string
	if a.gmail[email] {
		for _, attr := range []imap.MailboxAttr{imap.MailboxAttrAll, imap.MailboxAttrTrash, imap.MailboxAttrJunk} {
			if name := a.specialUse[email][attr]; name != "" {
				sync = append(sync, name)
			}
		}
	} else {
		for _, mb := range mailboxes {
			if !slices.Contains(mb.Attrs, imap.MailboxAttrNoSelect) {
				sync = append(sync, mb.Mailbox)
			}
		}
	}
	for _, mb := range sync {
		if err := a.syncMailbox(email, mb, false); err != nil {
			return fmt.Errorf("syncing mailbox %q: %w", mb, err)
		}
	}
	a.countMessages(email)
	if err := a.downloadBodies(email, bodiesPerSync); err != nil {
		return err
	}
	return a.indexMissing(email, indexChecksPerSync)
}

// indexChecksPerSync is the number of downloaded messages checked by
// [App.indexMissing] in each sync of an account.
var indexChecksPerSync = 2000

// indexMissing adds the downloaded messages of the given account that are
// not in its search index to it, as those that could not be read when they
// were downloaded (such as those in windows-1252 before it was supported),
// and counts the attachments of those downloaded before they were counted,
// checking up to the given number of messages that have not been checked
// yet in this run of the app.
func (a *App) indexMissing(email string, n int) error {
	si := a.searches[email]
	if si == nil {
		return nil
	}
	a.AsyncLock()
	if a.indexChecked[email] == nil {
		a.indexChecked[email] = map[string]bool{}
	}
	checked := a.indexChecked[email]
	var check []*CacheMessage
	for _, cm := range a.cache[email] {
		if cm.Filename != "" && !checked[cm.key] && !cm.isLocal() {
			if len(check) == n {
				a.syncMore = true
				break
			}
			check = append(check, cm)
		}
	}
	a.AsyncUnlock()
	if len(check) > 0 {
		a.setStatus(email+": checking the downloaded messages", -1)
	}
	dir := a.messagesDir(email)
	docs := map[string]*searchDoc{}
	counted := map[string]*CacheMessage{}
	for _, cm := range check {
		checked[cm.key] = true // only used by the sync, after the above
		a.AsyncLock()
		count := !cm.AttachmentsCounted
		a.AsyncUnlock()
		index := !si.has(cm.key)
		if !index && !count {
			continue
		}
		doc, err := readSearchDoc(filepath.Join(dir, cm.Filename))
		if err != nil {
			slog.Error("indexing message", "file", cm.Filename, "err", err)
			continue
		}
		if index {
			docs[cm.key] = doc
		}
		if count { // downloaded before they were counted
			a.AsyncLock()
			cm.Attachments, cm.AttachmentsCounted = doc.attachments, true
			a.AsyncUnlock()
			counted[cm.key] = cm
		}
	}
	if len(counted) > 0 {
		if err := a.stores[email].put(counted); err != nil {
			return err
		}
		a.syncUpdate()
	}
	return si.add(docs)
}

// countMessages sets the numbers of messages in the mailboxes of the given
// account from the cache, for the mailboxes that are not synced on their
// own, as the labels on Gmail.
func (a *App) countMessages(email string) {
	if !a.gmail[email] {
		return
	}
	a.AsyncLock()
	defer a.AsyncUnlock()
	counts := map[string]int{}
	for _, cm := range a.cache[email] {
		for _, l := range cm.Labels {
			counts[l.Name]++
		}
		if a.isUntagged(email, cm) {
			counts[untaggedLabel]++
		}
	}
	a.totalMessages[email] = counts
}

// syncMailbox syncs the messages of the given mailbox of the given account:
// the new ones, the changes of their flags (and labels), and those that are
// gone, as described in the doc of the package. With Gmail, the mailbox is
// All Mail, the Trash, or the Spam. A quiet sync, as the quick ones of
// [App.quickSyncLoop], does not show what it does in the status.
func (a *App) syncMailbox(email, mailbox string, quiet bool) error {
	a.AsyncLock() // as a quick sync runs along with the regular one
	c := a.imapClient[email]
	mu := a.imapMu[email]
	st := a.stores[email]
	gmail, condStore := a.gmail[email], a.condStore[email]
	a.AsyncUnlock()
	if c == nil || st == nil {
		return errNotConnected
	}

	if !quiet {
		a.setStatus(fmt.Sprintf("%s: syncing %s", email, friendlyLabelName(mailbox)), -1)
	}
	mu.Lock()
	defer mu.Unlock()
	// always selected again, for its current state
	a.setSelected(email, "")
	sd, err := c.Select(mailbox, &imap.SelectOptions{CondStore: condStore}).Wait()
	if err != nil {
		return fmt.Errorf("selecting: %w", err)
	}
	a.setSelected(email, mailbox)
	if !gmail {
		a.AsyncLock()
		a.totalMessages[email][mailbox] = int(sd.NumMessages)
		a.AsyncUnlock()
	}
	state, err := st.mailbox(mailbox)
	if err != nil {
		return err
	}
	if state.UIDValidity != sd.UIDValidity { // all new
		if state.UIDValidity != 0 {
			slog.Info("mail: new UIDVALIDITY, syncing again", "account", email, "mailbox", mailbox)
		}
		if err := a.removeMailboxUIDs(email, mailbox, nil); err != nil {
			return err
		}
		state = mailboxState{UIDValidity: sd.UIDValidity}
	}

	fetchOptions := &imap.FetchOptions{UID: true, Flags: true, ModSeq: condStore,
		GmailMsgID: gmail, GmailThreadID: gmail, GmailLabels: gmail}

	// the new messages
	if sd.UIDNext == 0 || sd.UIDNext > state.UIDNext {
		start := max(state.UIDNext, 1)
		opts := *fetchOptions
		opts.Envelope = true
		opts.RFC822Size = true
		var uids imap.UIDSet
		uids.AddRange(start, 0)                                    // start:*
		total := int(sd.NumMessages) - a.countUIDs(email, mailbox) // about
		if quiet {
			total = -1
		}
		if err := a.fetchMessages(c, email, mailbox, uids, &opts, start, total); err != nil {
			return fmt.Errorf("fetching new messages: %w", err)
		}
	}

	// the changes of the flags (and labels) of the other messages
	if state.UIDNext > 1 {
		if condStore && state.HighestModSeq > 0 {
			if sd.HighestModSeq > state.HighestModSeq {
				opts := *fetchOptions
				opts.ChangedSince = state.HighestModSeq
				var uids imap.UIDSet
				uids.AddRange(1, state.UIDNext-1)
				if err := a.fetchChanges(c, email, mailbox, uids, &opts); err != nil {
					return fmt.Errorf("fetching changes: %w", err)
				}
			}
		} else {
			var uids imap.UIDSet
			uids.AddRange(1, state.UIDNext-1)
			if err := a.fetchChanges(c, email, mailbox, uids, fetchOptions); err != nil {
				return fmt.Errorf("fetching flags: %w", err)
			}
		}
	}

	// the messages that are gone
	if a.countUIDs(email, mailbox) != int(sd.NumMessages) {
		sdata, err := c.UIDSearch(&imap.SearchCriteria{}, nil).Wait()
		if err != nil {
			return fmt.Errorf("searching for the messages: %w", err)
		}
		if err := a.removeMailboxUIDs(email, mailbox, sdata.AllUIDs()); err != nil {
			return err
		}
	}

	state.UIDNext = sd.UIDNext
	state.HighestModSeq = sd.HighestModSeq
	state.Synced = time.Now()
	return st.setMailbox(mailbox, state)
}

// countUIDs returns the number of messages of the given account that
// are in the given mailbox, with UIDs in it.
func (a *App) countUIDs(email, mailbox string) int {
	a.AsyncLock()
	defer a.AsyncUnlock()
	n := 0
	for _, cm := range a.cache[email] {
		for _, l := range cm.Labels {
			if l.Name == mailbox && l.UID > 0 {
				n++
			}
		}
	}
	return n
}

// fetchMessages fetches the messages with the given UIDs (from the given
// first one) in the given mailbox, adding them to the cache and the store
// in batches, or adding the mailbox to their labels if they are cached,
// showing the progress of the given (approximate) total in the status,
// unless it is negative.
func (a *App) fetchMessages(c *imapclient.Client, email, mailbox string, uids imap.UIDSet, options *imap.FetchOptions, first imap.UID, total int) error {
	cmd := c.Fetch(uids, options)
	n := 0
	batch := map[string]*CacheMessage{}
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		err := a.stores[email].put(batch)
		batch = map[string]*CacheMessage{}
		a.syncUpdate()
		return err
	}
	for {
		msg := cmd.Next()
		if msg == nil {
			break
		}
		md, err := msg.Collect()
		if err != nil {
			cmd.Close()
			return err
		}
		if md.UID < first || md.Envelope == nil { // n:* has the last message if there are no new ones
			continue
		}
		key := cacheKey(md.Envelope)
		if a.gmail[email] {
			key = gmailKey(md.GmailMsgID)
		}
		a.AsyncLock()
		cm := a.cache[email][key]
		if cm == nil {
			cm = &CacheMessage{Envelope: *md.Envelope, GmailMsgID: md.GmailMsgID, GmailThreadID: md.GmailThreadID, key: key}
			a.cache[email][key] = cm
		}
		cm.Flags = md.Flags
		a.setMessageLabels(email, cm, mailbox, md.UID, md.GmailLabels)
		a.AsyncUnlock()
		batch[key] = cm
		n++
		if total >= 0 {
			a.progressStatus(email, "listing messages in "+friendlyLabelName(mailbox), n, max(total, n), 200)
		}
		if len(batch) >= 200 {
			if err := flush(); err != nil {
				cmd.Close()
				return err
			}
		}
	}
	if err := cmd.Close(); err != nil {
		return err
	}
	return flush()
}

// setMessageLabels sets the labels of the given message, which is in the given
// mailbox with the given UID: on Gmail, those of its Gmail labels, and the
// mailbox, which is All Mail, the Trash, or the Spam, with its UID; on other
// servers, the mailbox with its UID is added to them. It must be called
// with the lock of the window.
func (a *App) setMessageLabels(email string, cm *CacheMessage, mailbox string, uid imap.UID, gmailLabels []string) {
	if !a.gmail[email] {
		i := slices.IndexFunc(cm.Labels, func(l Label) bool { return l.Name == mailbox })
		if i >= 0 {
			cm.Labels[i].UID = uid
		} else {
			cm.Labels = append(cm.Labels, Label{mailbox, uid})
		}
		return
	}
	labels := []Label{{mailbox, uid}}
	for _, gl := range gmailLabels {
		if name := a.gmailMailbox(email, gl); name != "" && name != mailbox {
			labels = append(labels, Label{Name: name})
		}
	}
	cm.Labels = labels
}

// fetchChanges fetches the flags (and Gmail labels) of the messages with
// the given UIDs in the given mailbox, with the given options, as of those
// changed since the last sync, updating the cached messages.
func (a *App) fetchChanges(c *imapclient.Client, email, mailbox string, uids imap.UIDSet, options *imap.FetchOptions) error {
	a.AsyncLock()
	byUID := map[imap.UID]string{}
	for k, cm := range a.cache[email] {
		for _, l := range cm.Labels {
			if l.Name == mailbox && l.UID > 0 {
				byUID[l.UID] = k
			}
		}
	}
	a.AsyncUnlock()
	cmd := c.Fetch(uids, options)
	changed := map[string]*CacheMessage{}
	for {
		msg := cmd.Next()
		if msg == nil {
			break
		}
		md, err := msg.Collect()
		if err != nil {
			cmd.Close()
			return err
		}
		k, ok := byUID[md.UID]
		if !ok {
			continue
		}
		a.AsyncLock()
		cm := a.cache[email][k]
		if cm != nil && (!slices.Equal(cm.Flags, md.Flags) || (a.gmail[email] && options.GmailLabels)) {
			cm.Flags = md.Flags
			if a.gmail[email] {
				a.setMessageLabels(email, cm, mailbox, md.UID, md.GmailLabels)
			}
			changed[k] = cm
		}
		a.AsyncUnlock()
	}
	if err := cmd.Close(); err != nil {
		return err
	}
	if len(changed) == 0 {
		return nil
	}
	a.syncUpdate()
	return a.stores[email].put(changed)
}

// removeMailboxUIDs removes the given mailbox from the labels of the
// messages of the given account whose UIDs in it are not in the given
// ones (all of them for nil), and removes the messages that are then in
// no mailboxes, with their files and search documents. On Gmail, a
// message that is not in All Mail, the Trash, or the Spam is gone.
func (a *App) removeMailboxUIDs(email, mailbox string, keep []imap.UID) error {
	keepSet := map[imap.UID]bool{}
	for _, uid := range keep {
		keepSet[uid] = true
	}
	changed := map[string]*CacheMessage{}
	var gone []string
	var files []string
	a.AsyncLock()
	for k, cm := range a.cache[email] {
		i := slices.IndexFunc(cm.Labels, func(l Label) bool { return l.Name == mailbox && l.UID > 0 })
		if i < 0 || keepSet[cm.Labels[i].UID] {
			continue
		}
		if a.gmail[email] {
			cm.Labels = nil // only in the Trash or Spam, if it is
		} else {
			cm.Labels = slices.Delete(cm.Labels, i, i+1)
		}
		if !slices.ContainsFunc(cm.Labels, func(l Label) bool { return l.UID > 0 }) {
			gone = append(gone, k)
			if cm.Filename != "" {
				files = append(files, filepath.Join(a.messagesDir(email), cm.Filename))
			}
			delete(a.cache[email], k)
			continue
		}
		changed[k] = cm
	}
	a.AsyncUnlock()
	if len(gone) > 0 || len(changed) > 0 {
		a.syncUpdate()
	}
	for _, f := range files {
		os.Remove(f)
	}
	if err := a.stores[email].remove(gone...); err != nil {
		return err
	}
	if err := a.searches[email].remove(gone...); err != nil {
		return err
	}
	return a.stores[email].put(changed)
}

// gmailMailbox returns the name of the mailbox of the given Gmail label of
// the given account: those of the system labels, as \Inbox and \Sent, are
// from the special-use attributes of the mailboxes, and the others are the
// names of their mailboxes. It returns "" for one that is not a mailbox.
func (a *App) gmailMailbox(email, label string) string {
	if !strings.HasPrefix(label, `\`) {
		return label
	}
	if strings.EqualFold(label, `\Inbox`) {
		return "INBOX"
	}
	attr := gmailLabelAttrs[strings.ToLower(label)]
	if attr == "" {
		return ""
	}
	return a.specialUse[email][attr]
}

// gmailLabel returns the Gmail label of the given mailbox of the given
// account, the inverse of [App.gmailMailbox].
func (a *App) gmailLabel(email, mailbox string) string {
	if mailbox == "INBOX" {
		return `\Inbox`
	}
	for label, attr := range gmailLabelAttrs {
		if a.specialUse[email][attr] == mailbox {
			return gmailSystemLabels[label]
		}
	}
	return mailbox
}

// gmailLabelAttrs are the special-use attributes of the mailboxes of the
// Gmail system labels, by the labels in lower case.
var gmailLabelAttrs = map[string]imap.MailboxAttr{
	`\sent`: imap.MailboxAttrSent, `\draft`: imap.MailboxAttrDrafts, `\drafts`: imap.MailboxAttrDrafts,
	`\important`: imap.MailboxAttrImportant, `\starred`: imap.MailboxAttrFlagged,
	`\trash`: imap.MailboxAttrTrash, `\spam`: imap.MailboxAttrJunk,
}

// gmailSystemLabels are the Gmail system labels, by them in lower case.
var gmailSystemLabels = map[string]string{
	`\sent`: `\Sent`, `\draft`: `\Draft`, `\drafts`: `\Draft`, `\important`: `\Important`,
	`\starred`: `\Starred`, `\trash`: `\Trash`, `\spam`: `\Spam`,
}

// downloadBodies downloads the contents of up to the given number of the
// messages of the given account that do not have them, the newest first,
// saving them in files and indexing their text for search.
func (a *App) downloadBodies(email string, n int) error {
	a.AsyncLock()
	var need []*CacheMessage
	for _, cm := range a.cache[email] {
		if cm.Filename == "" {
			need = append(need, cm)
		}
	}
	total := len(a.cache[email])
	a.AsyncUnlock()
	done := total - len(need)
	if len(need) == 0 {
		return nil
	}
	slices.SortFunc(need, func(x, y *CacheMessage) int { return cmp.Compare(y.Date.UnixNano(), x.Date.UnixNano()) })
	if len(need) > n {
		need = need[:n]
		a.syncMore = true
	}
	return a.downloadMessages(email, need, func() {
		done++
		a.progressStatus(email, "downloading messages", done, total, 20)
	})
}

// downloadMessages downloads the contents of the given messages of the
// given account, by mailbox, saving them in files and indexing them,
// calling the given function, if any, after each one.
func (a *App) downloadMessages(email string, msgs []*CacheMessage, progress func()) error {
	byMailbox := map[string]map[imap.UID]*CacheMessage{}
	for _, cm := range msgs {
		for _, l := range cm.Labels {
			if l.UID > 0 {
				if byMailbox[l.Name] == nil {
					byMailbox[l.Name] = map[imap.UID]*CacheMessage{}
				}
				byMailbox[l.Name][l.UID] = cm
				break
			}
		}
	}
	c, mu := a.imapClient[email], a.imapMu[email]
	dir := a.messagesDir(email)
	reading := false // whether the message being read is downloaded
	defer func() {
		if reading { // shown in place of "Downloading the message...", now
			a.AsyncLock()
			a.updateReading()
			a.AsyncUnlock()
		}
	}()
	for mailbox, byUID := range byMailbox {
		var uids imap.UIDSet
		for uid := range byUID {
			uids.AddNum(uid)
		}
		mu.Lock()
		err := a.selectMailbox(c, email, mailbox)
		if err != nil {
			mu.Unlock()
			return err
		}
		cmd := c.Fetch(uids, &imap.FetchOptions{UID: true, BodySection: []*imap.FetchItemBodySection{{Peek: true}}})
		saved := map[string]*CacheMessage{}
		docs := map[string]*searchDoc{}
		for {
			msg := cmd.Next()
			if msg == nil {
				break
			}
			md, err := msg.Collect()
			if err != nil {
				cmd.Close()
				mu.Unlock()
				return err
			}
			cm := byUID[md.UID]
			if cm == nil || len(md.BodySection) == 0 {
				continue
			}
			filename := shardName(messageFilename())
			fn := filepath.Join(dir, filename)
			if err := os.MkdirAll(filepath.Dir(fn), 0o700); err != nil {
				cmd.Close()
				mu.Unlock()
				return err
			}
			if err := os.WriteFile(fn, md.BodySection[0].Bytes, 0o600); err != nil {
				cmd.Close()
				mu.Unlock()
				return err
			}
			doc, docErr := readSearchDoc(fn)
			a.AsyncLock()
			if cm.Filename != "" { // downloaded at the same time, as when read
				a.AsyncUnlock()
				os.Remove(fn)
				continue
			}
			cm.Filename = filename
			if docErr == nil {
				cm.Attachments, cm.AttachmentsCounted = doc.attachments, true
			}
			if cm == a.readMessage {
				reading = true
			}
			a.AsyncUnlock()
			k := cm.key
			saved[k] = cm
			if progress != nil {
				progress()
			}
			if docErr == nil {
				docs[k] = doc
			} else {
				slog.Error("indexing message", "file", filename, "err", docErr)
			}
		}
		err = cmd.Close()
		mu.Unlock()
		if err != nil {
			return err
		}
		if err := a.stores[email].put(saved); err != nil {
			return err
		}
		if err := a.searches[email].add(docs); err != nil {
			return err
		}
	}
	a.syncUpdate()
	return nil
}

// downloadNow downloads the contents of the given message of the given
// account now, in the background, if it does not have them, as when it
// is read before they are downloaded, updating the app after.
func (a *App) downloadNow(email string, cm *CacheMessage) {
	if cm.Filename != "" || a.downloading[cm] || a.imapClient[email] == nil {
		return // downloaded in the next sync once connected
	}
	a.downloading[cm] = true
	go func() {
		err := a.downloadMessages(email, []*CacheMessage{cm}, nil)
		a.AsyncLock()
		delete(a.downloading, cm)
		a.AsyncUnlock()
		if err != nil {
			slog.Error("mail: downloading a message", "err", err)
		}
	}()
}

// saveMessages saves the given messages of the given account in its store.
func (a *App) saveMessages(email string, msgs ...*CacheMessage) error {
	st := a.stores[email]
	if st == nil || len(msgs) == 0 {
		return nil
	}
	m := map[string]*CacheMessage{}
	for _, cm := range msgs {
		if cm.key != "" && !cm.isLocal() {
			m[cm.key] = cm
		}
	}
	return st.put(m)
}

// saveAllMessages saves all of the messages of the given account in its
// store, as after changes to many of them.
func (a *App) saveAllMessages(email string) error {
	a.AsyncLock()
	msgs := make([]*CacheMessage, 0, len(a.cache[email]))
	for _, cm := range a.cache[email] {
		msgs = append(msgs, cm)
	}
	a.AsyncUnlock()
	return a.saveMessages(email, msgs...)
}
