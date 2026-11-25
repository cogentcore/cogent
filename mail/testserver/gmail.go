// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Parts of this file are adapted from imapmemserver of go-imap
// (github.com/cogentcore/go-imap/v2), Copyright (c) 2013 The Go-IMAP Authors,
// under the MIT license.

package testserver

import (
	"bufio"
	"bytes"
	"io"
	"maps"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cogentcore/go-imap/v2"
	"github.com/cogentcore/go-imap/v2/imapserver"
	"github.com/emersion/go-message/textproto"
)

// The Gmail-like backend of the server: like Gmail, each message of an
// account is stored once, with its flags, and has labels, which are the
// mailboxes it is in; each mailbox is a view of the messages with its
// label, with their own UIDs. All Mail has all of the messages, except
// those in the Trash and the Spam. It has the Gmail extension of IMAP
// (X-GM-EXT-1: the X-GM-MSGID, X-GM-THRID, and X-GM-LABELS of messages,
// and changing their labels with STORE X-GM-LABELS) and CONDSTORE (the
// modification sequences of the messages, for the changes since a sync).
//
// It is for testing: there are no notifications of changes made by other
// sessions to a selected mailbox, whose sequence numbers can then differ
// from those of the client, which should use UIDs.

// The mailboxes of the system labels of the Gmail-like backend.
const (
	gmailAllMail   = "[Gmail]/All Mail"
	gmailDrafts    = "[Gmail]/Drafts"
	gmailImportant = "[Gmail]/Important"
	gmailSent      = "[Gmail]/Sent Mail"
	gmailSpam      = "[Gmail]/Spam"
	gmailStarred   = "[Gmail]/Starred"
	gmailTrash     = "[Gmail]/Trash"
)

// gmailSpecialUse are the special-use attributes of the mailboxes of the
// system labels, with their Gmail labels.
var gmailSpecialUse = map[string]struct {
	attr  imap.MailboxAttr
	label string
}{
	gmailAllMail:   {imap.MailboxAttrAll, ""},
	gmailDrafts:    {imap.MailboxAttrDrafts, `\Draft`},
	gmailImportant: {imap.MailboxAttrImportant, `\Important`},
	gmailSent:      {imap.MailboxAttrSent, `\Sent`},
	gmailSpam:      {imap.MailboxAttrJunk, `\Spam`},
	gmailStarred:   {imap.MailboxAttrFlagged, `\Starred`},
	gmailTrash:     {imap.MailboxAttrTrash, `\Trash`},
}

// gmailUser is an account of the Gmail-like backend.
type gmailUser struct {
	mu        sync.Mutex
	name      string
	password  string
	boxes     map[string]*gmailBox
	msgs      map[uint64]*gmailMessage
	nextID    uint64
	modSeq    uint64
	validity  uint32
	byMessage map[string]uint64 // thread IDs by Message-ID
	byID      map[string]*gmailMessage
}

// gmailMessage is a message of a [gmailUser].
type gmailMessage struct {
	id, thread uint64
	buf        []byte
	t          time.Time
	flags      map[imap.Flag]bool
	modSeq     uint64
}

// gmailBox is a mailbox of a [gmailUser]: a label, or All Mail.
type gmailBox struct {
	name       string
	validity   uint32
	uidNext    imap.UID
	entries    []*gmailEntry // in UID order
	subscribed bool

	// added, if not nil, is closed when a message is added to the
	// mailbox, for the sessions idling on it (see [gmailSession.Idle]).
	added chan struct{}
}

// gmailEntry is a message in a [gmailBox], with its UID there.
type gmailEntry struct {
	uid imap.UID
	msg *gmailMessage

	// deleted is the \Deleted flag, which is of the message in this
	// mailbox (expunging it removes the label), unlike the others.
	deleted bool
}

func newGmailUser(name, password string) *gmailUser {
	u := &gmailUser{name: name, password: password, boxes: map[string]*gmailBox{},
		msgs: map[uint64]*gmailMessage{}, nextID: 0x18f0000000000000, byMessage: map[string]uint64{},
		byID: map[string]*gmailMessage{}}
	for _, mb := range []string{"INBOX", gmailAllMail, gmailDrafts, gmailImportant, gmailSent, gmailSpam, gmailStarred, gmailTrash} {
		u.createLocked(mb)
	}
	return u
}

// createLocked returns the mailbox with the given name, creating it, with
// its parents, as Gmail does for nested labels, if it does not exist.
func (u *gmailUser) createLocked(name string) *gmailBox {
	if b := u.boxes[name]; b != nil {
		return b
	}
	if i := strings.LastIndex(name, "/"); i > 0 && !strings.HasPrefix(name, "[Gmail]/") {
		u.createLocked(name[:i])
	}
	u.validity++
	b := &gmailBox{name: name, validity: u.validity, uidNext: 1, subscribed: true}
	u.boxes[name] = b
	return b
}

// nextModSeq returns a new modification sequence.
func (u *gmailUser) nextModSeq() uint64 {
	u.modSeq++
	return u.modSeq
}

// entry returns the entry of the given message in the given mailbox, or nil.
func (b *gmailBox) entry(msg *gmailMessage) *gmailEntry {
	for _, e := range b.entries {
		if e.msg == msg {
			return e
		}
	}
	return nil
}

// isView returns whether the given mailbox is All Mail, the Trash, or the
// Spam, which are not labels that a message can have with others.
func isView(name string) bool {
	return name == gmailAllMail || name == gmailTrash || name == gmailSpam
}

// addLabel adds the given message to the given mailbox, returning its UID
// there: to the Trash or Spam, out of all of the others; to another one,
// out of the Trash and Spam, and into All Mail.
func (u *gmailUser) addLabel(msg *gmailMessage, b *gmailBox) imap.UID {
	if e := b.entry(msg); e != nil {
		return e.uid
	}
	if b.name == gmailTrash || b.name == gmailSpam {
		for _, o := range u.boxes {
			u.removeEntry(o, msg)
		}
	} else if b.name != gmailAllMail {
		u.removeEntry(u.boxes[gmailTrash], msg)
		u.removeEntry(u.boxes[gmailSpam], msg)
		u.addLabel(msg, u.boxes[gmailAllMail])
	}
	e := &gmailEntry{uid: b.uidNext, msg: msg}
	b.uidNext++
	b.entries = append(b.entries, e)
	msg.modSeq = u.nextModSeq()
	if b.added != nil {
		close(b.added)
		b.added = nil
	}
	return e.uid
}

// removeEntry removes the given message from the given mailbox, if it is
// in it, returning its sequence number there, or 0.
func (u *gmailUser) removeEntry(b *gmailBox, msg *gmailMessage) uint32 {
	for i, e := range b.entries {
		if e.msg == msg {
			b.entries = slices.Delete(b.entries, i, i+1)
			msg.modSeq = u.nextModSeq()
			return uint32(i + 1)
		}
	}
	return 0
}

// removeLabel removes the given message from the given mailbox: from All
// Mail, the Trash, or the Spam, it is deleted, out of all of them.
func (u *gmailUser) removeLabel(msg *gmailMessage, b *gmailBox) {
	if isView(b.name) {
		for _, o := range u.boxes {
			u.removeEntry(o, msg)
		}
		delete(u.msgs, msg.id)
		return
	}
	u.removeEntry(b, msg)
}

// labels returns the Gmail labels of the given message: those of the
// mailboxes it is in, but All Mail.
func (u *gmailUser) labels(msg *gmailMessage) []string {
	var res []string
	for name, b := range u.boxes {
		if name == gmailAllMail || b.entry(msg) == nil {
			continue
		}
		res = append(res, gmailLabelOf(name))
	}
	sort.Strings(res)
	return res
}

// gmailLabelOf returns the Gmail label of the given mailbox.
func gmailLabelOf(name string) string {
	if name == "INBOX" {
		return `\Inbox`
	}
	if su, ok := gmailSpecialUse[name]; ok && su.label != "" {
		return su.label
	}
	return name
}

// boxOfLabel returns the mailbox of the given Gmail label, creating it
// for a new label, as Gmail does.
func (u *gmailUser) boxOfLabel(label string) *gmailBox {
	if strings.EqualFold(label, `\Inbox`) {
		return u.boxes["INBOX"]
	}
	if strings.HasPrefix(label, `\`) {
		for name, su := range gmailSpecialUse {
			if strings.EqualFold(su.label, label) {
				return u.boxes[name]
			}
		}
		return nil
	}
	return u.createLocked(label)
}

// appendMessage adds a new message with the given contents, flags, and
// date, with the given labels (mailboxes), returning it.
// A message with the Message-ID of one that the user has is that message,
// which gets the labels, as on Gmail.
func (u *gmailUser) appendMessage(buf []byte, flags []imap.Flag, t time.Time, mailboxes ...string) *gmailMessage {
	if t.IsZero() {
		t = time.Now()
	}
	if hdr, err := textproto.ReadHeader(bufio.NewReader(bytes.NewReader(buf))); err == nil {
		if msg := u.byID[strings.Trim(hdr.Get("Message-Id"), "<> ")]; msg != nil && u.msgs[msg.id] == msg {
			for _, mb := range mailboxes {
				u.addLabel(msg, u.createLocked(mb))
			}
			return msg
		}
	}
	u.nextID++
	msg := &gmailMessage{id: u.nextID, buf: buf, t: t, flags: map[imap.Flag]bool{}}
	for _, f := range flags {
		msg.flags[canonicalFlag(f)] = true
	}
	msg.thread = msg.id
	if hdr, err := textproto.ReadHeader(bufio.NewReader(bytes.NewReader(buf))); err == nil {
		refs := strings.Fields(hdr.Get("References") + " " + hdr.Get("In-Reply-To"))
		for _, r := range refs {
			if th, ok := u.byMessage[strings.Trim(r, "<>")]; ok {
				msg.thread = th
				break
			}
		}
		if id := strings.Trim(hdr.Get("Message-Id"), "<> "); id != "" {
			if _, ok := u.byMessage[id]; !ok {
				u.byMessage[id] = msg.thread
			}
			u.byID[id] = msg
		}
	}
	u.msgs[msg.id] = msg
	if len(mailboxes) == 0 {
		mailboxes = []string{gmailAllMail}
	}
	for _, mb := range mailboxes {
		u.addLabel(msg, u.createLocked(mb))
	}
	return msg
}

func canonicalFlag(f imap.Flag) imap.Flag {
	return imap.Flag(strings.ToLower(string(f)))
}

// Append, Create, and Status are for the [Server], as for an
// imapmemserver.User.

func (u *gmailUser) Append(mailbox string, r imap.LiteralReader, options *imap.AppendOptions) (*imap.AppendData, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	b := u.boxes[mailbox]
	if b == nil {
		return nil, &imap.Error{Type: imap.StatusResponseTypeNo, Code: imap.ResponseCodeTryCreate, Text: "No such mailbox"}
	}
	msg := u.appendMessage(buf, options.Flags, options.Time, mailbox)
	return &imap.AppendData{UIDValidity: b.validity, UID: b.entry(msg).uid}, nil
}

func (u *gmailUser) Create(name string, options *imap.CreateOptions) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	name = strings.TrimRight(name, "/")
	if u.boxes[name] != nil {
		return &imap.Error{Type: imap.StatusResponseTypeNo, Code: imap.ResponseCodeAlreadyExists, Text: "Mailbox already exists"}
	}
	u.createLocked(name)
	return nil
}

func (u *gmailUser) Status(name string, options *imap.StatusOptions) (*imap.StatusData, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	b := u.boxes[name]
	if b == nil {
		return nil, &imap.Error{Type: imap.StatusResponseTypeNo, Code: imap.ResponseCodeNonExistent, Text: "No such mailbox"}
	}
	return u.statusLocked(b, options), nil
}

func (u *gmailUser) statusLocked(b *gmailBox, options *imap.StatusOptions) *imap.StatusData {
	data := &imap.StatusData{Mailbox: b.name}
	if options.NumMessages {
		n := uint32(len(b.entries))
		data.NumMessages = &n
	}
	if options.UIDNext {
		data.UIDNext = b.uidNext
	}
	if options.UIDValidity {
		data.UIDValidity = b.validity
	}
	if options.NumUnseen {
		n := uint32(0)
		for _, e := range b.entries {
			if !e.msg.flags[canonicalFlag(imap.FlagSeen)] {
				n++
			}
		}
		data.NumUnseen = &n
	}
	if options.HighestModSeq {
		data.HighestModSeq = u.modSeq
	}
	return data
}

// gmailSession is a session of the Gmail-like backend.
type gmailSession struct {
	s    *Server
	u    *gmailUser
	box  *gmailBox
	read bool // read-only (EXAMINE)
}

var (
	_ imapserver.Session      = (*gmailSession)(nil)
	_ imapserver.SessionMove  = (*gmailSession)(nil)
	_ imapserver.SessionGmail = (*gmailSession)(nil)

	_ imapserver.SessionIMAP4rev2 = (*gmailSession)(nil)
)

func (sess *gmailSession) Namespace() (*imap.NamespaceData, error) {
	return &imap.NamespaceData{Personal: []imap.NamespaceDescriptor{{Delim: '/'}}}, nil
}

func (sess *gmailSession) Close() error { return nil }

func (sess *gmailSession) Login(username, password string) error {
	u := sess.s.gmailUsers[strings.ToLower(username)]
	if u == nil || password != u.password {
		return imapserver.ErrAuthFailed
	}
	sess.u = u
	return nil
}

func (sess *gmailSession) Select(name string, options *imap.SelectOptions) (*imap.SelectData, error) {
	u := sess.u
	u.mu.Lock()
	defer u.mu.Unlock()
	b := u.boxes[name]
	if b == nil || name == "[Gmail]" {
		return nil, &imap.Error{Type: imap.StatusResponseTypeNo, Code: imap.ResponseCodeNonExistent, Text: "No such mailbox"}
	}
	sess.box = b
	sess.read = options != nil && options.ReadOnly
	flags := []imap.Flag{imap.FlagSeen, imap.FlagAnswered, imap.FlagFlagged, imap.FlagDeleted, imap.FlagDraft}
	return &imap.SelectData{
		Flags:          flags,
		PermanentFlags: append(slices.Clone(flags), imap.FlagWildcard),
		NumMessages:    uint32(len(b.entries)),
		UIDNext:        b.uidNext,
		UIDValidity:    b.validity,
		HighestModSeq:  u.modSeq,
	}, nil
}

func (sess *gmailSession) Unselect() error {
	sess.box = nil
	return nil
}

func (sess *gmailSession) Create(name string, options *imap.CreateOptions) error {
	return sess.u.Create(name, options)
}

func (sess *gmailSession) Delete(name string) error {
	u := sess.u
	u.mu.Lock()
	defer u.mu.Unlock()
	b := u.boxes[name]
	if b == nil || isView(name) || name == "INBOX" || gmailSpecialUse[name].attr != "" {
		return &imap.Error{Type: imap.StatusResponseTypeNo, Text: "Cannot delete this mailbox"}
	}
	for _, e := range b.entries {
		e.msg.modSeq = u.nextModSeq() // its labels changed
	}
	delete(u.boxes, name)
	return nil
}

func (sess *gmailSession) Rename(name, newName string, options *imap.RenameOptions) error {
	u := sess.u
	u.mu.Lock()
	defer u.mu.Unlock()
	b := u.boxes[name]
	if b == nil || isView(name) || name == "INBOX" || gmailSpecialUse[name].attr != "" {
		return &imap.Error{Type: imap.StatusResponseTypeNo, Text: "Cannot rename this mailbox"}
	}
	if u.boxes[newName] != nil {
		return &imap.Error{Type: imap.StatusResponseTypeNo, Code: imap.ResponseCodeAlreadyExists, Text: "Mailbox already exists"}
	}
	for old, ob := range u.boxes {
		if old == name || strings.HasPrefix(old, name+"/") {
			delete(u.boxes, old)
			ob.name = newName + strings.TrimPrefix(old, name)
			u.boxes[ob.name] = ob
			for _, e := range ob.entries {
				e.msg.modSeq = u.nextModSeq()
			}
		}
	}
	return nil
}

func (sess *gmailSession) Subscribe(name string) error   { return sess.setSubscribed(name, true) }
func (sess *gmailSession) Unsubscribe(name string) error { return sess.setSubscribed(name, false) }

func (sess *gmailSession) setSubscribed(name string, on bool) error {
	sess.u.mu.Lock()
	defer sess.u.mu.Unlock()
	if b := sess.u.boxes[name]; b != nil {
		b.subscribed = on
		return nil
	}
	return &imap.Error{Type: imap.StatusResponseTypeNo, Code: imap.ResponseCodeNonExistent, Text: "No such mailbox"}
}

func (sess *gmailSession) List(w *imapserver.ListWriter, ref string, patterns []string, options *imap.ListOptions) error {
	u := sess.u
	u.mu.Lock()
	defer u.mu.Unlock()
	if len(patterns) == 0 {
		return w.WriteList(&imap.ListData{Attrs: []imap.MailboxAttr{imap.MailboxAttrNoSelect}, Delim: '/'})
	}
	names := []string{"[Gmail]"}
	for name := range u.boxes {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		match := false
		for _, pattern := range patterns {
			if imapserver.MatchList(name, '/', ref, pattern) {
				match = true
				break
			}
		}
		if !match {
			continue
		}
		data := &imap.ListData{Mailbox: name, Delim: '/'}
		if name == "[Gmail]" {
			data.Attrs = []imap.MailboxAttr{imap.MailboxAttrNoSelect, imap.MailboxAttrHasChildren}
		} else {
			b := u.boxes[name]
			if options.SelectSubscribed && !b.subscribed {
				continue
			}
			if b.subscribed {
				data.Attrs = append(data.Attrs, imap.MailboxAttrSubscribed)
			}
			// Gmail has the special-use attributes in all of the lists
			if su, ok := gmailSpecialUse[name]; ok {
				data.Attrs = append(data.Attrs, su.attr)
			}
			if options.ReturnStatus != nil {
				data.Status = u.statusLocked(b, options.ReturnStatus)
			}
		}
		if err := w.WriteList(data); err != nil {
			return err
		}
	}
	return nil
}

func (sess *gmailSession) Status(name string, options *imap.StatusOptions) (*imap.StatusData, error) {
	return sess.u.Status(name, options)
}

func (sess *gmailSession) Append(mailbox string, r imap.LiteralReader, options *imap.AppendOptions) (*imap.AppendData, error) {
	return sess.u.Append(mailbox, r, options)
}

func (sess *gmailSession) Poll(w *imapserver.UpdateWriter, allowExpunge bool) error {
	return nil
}

// Idle sends the number of messages of the selected mailbox when messages
// are added to it, as for new mail, until stopped.
func (sess *gmailSession) Idle(w *imapserver.UpdateWriter, stop <-chan struct{}) error {
	u := sess.u
	for {
		u.mu.Lock()
		b := sess.box
		if b == nil {
			u.mu.Unlock()
			<-stop
			return nil
		}
		if b.added == nil {
			b.added = make(chan struct{})
		}
		added := b.added
		u.mu.Unlock()
		select {
		case <-stop:
			return nil
		case <-added:
		}
		u.mu.Lock()
		n := len(b.entries)
		u.mu.Unlock()
		if err := w.WriteNumMessages(uint32(n)); err != nil {
			return err
		}
	}
}

// forEach calls the given function for each entry of the selected mailbox
// in the given set of numbers, with its sequence number. It must be called
// with the lock of the user.
func (sess *gmailSession) forEach(numSet imap.NumSet, f func(seqNum uint32, e *gmailEntry)) {
	b := sess.box
	entries := slices.Clone(b.entries)
	for i, e := range entries {
		seqNum := uint32(i + 1)
		var contains bool
		switch ns := numSet.(type) {
		case imap.SeqSet:
			contains = seqSetContains(ns, seqNum, uint32(len(entries)))
		case imap.UIDSet:
			contains = uidSetContains(ns, e.uid, b.uidNext-1)
		}
		if contains {
			f(seqNum, e)
		}
	}
}

// seqSetContains returns whether the given set has the given number, with
// the given number for * (0 in a range).
func seqSetContains(set imap.SeqSet, n, last uint32) bool {
	for _, r := range set {
		start, stop := r.Start, r.Stop
		if start == 0 {
			start = last
		}
		if stop == 0 {
			stop = last
		}
		if start > stop {
			start, stop = stop, start
		}
		if n >= start && n <= stop {
			return true
		}
	}
	return false
}

// uidSetContains is [seqSetContains] for UIDs.
func uidSetContains(set imap.UIDSet, uid, last imap.UID) bool {
	for _, r := range set {
		start, stop := r.Start, r.Stop
		if start == 0 {
			start = last
		}
		if stop == 0 {
			stop = last
		}
		if start > stop {
			start, stop = stop, start
		}
		if uid >= start && uid <= stop {
			return true
		}
	}
	return false
}

// flagList returns the flags of the given entry, which has the \Deleted
// flag of its mailbox.
func flagList(e *gmailEntry) []imap.Flag {
	var res []imap.Flag
	for f := range e.msg.flags {
		res = append(res, systemFlag(f))
	}
	if e.deleted {
		res = append(res, imap.FlagDeleted)
	}
	slices.Sort(res)
	return res
}

// systemFlag returns the given flag, in lower case, with the case of the
// system flags, as \Seen.
func systemFlag(f imap.Flag) imap.Flag {
	for _, sf := range []imap.Flag{imap.FlagSeen, imap.FlagAnswered, imap.FlagFlagged, imap.FlagDeleted, imap.FlagDraft} {
		if strings.EqualFold(string(sf), string(f)) {
			return sf
		}
	}
	return f
}

func (sess *gmailSession) Fetch(w *imapserver.FetchWriter, numSet imap.NumSet, options *imap.FetchOptions) error {
	u := sess.u
	u.mu.Lock()
	defer u.mu.Unlock()
	markSeen := false
	for _, bs := range options.BodySection {
		if !bs.Peek && !sess.read {
			markSeen = true
		}
	}
	var err error
	sess.forEach(numSet, func(seqNum uint32, e *gmailEntry) {
		if err != nil {
			return
		}
		if options.ChangedSince > 0 && e.msg.modSeq <= options.ChangedSince {
			return
		}
		if markSeen && !e.msg.flags[canonicalFlag(imap.FlagSeen)] {
			e.msg.flags[canonicalFlag(imap.FlagSeen)] = true
			e.msg.modSeq = u.nextModSeq()
		}
		err = sess.fetchEntry(w.CreateMessage(seqNum), e, options)
	})
	return err
}

// fetchEntry writes the fetch response of the given entry.
func (sess *gmailSession) fetchEntry(w *imapserver.FetchResponseWriter, e *gmailEntry, options *imap.FetchOptions) error {
	msg := e.msg
	w.WriteUID(e.uid)
	if options.Flags {
		w.WriteFlags(flagList(e))
	}
	if options.InternalDate {
		w.WriteInternalDate(msg.t)
	}
	if options.RFC822Size {
		w.WriteRFC822Size(int64(len(msg.buf)))
	}
	if options.Envelope {
		if hdr, err := textproto.ReadHeader(bufio.NewReader(bytes.NewReader(msg.buf))); err == nil {
			w.WriteEnvelope(imapserver.ExtractEnvelope(hdr))
		}
	}
	if options.ModSeq || options.ChangedSince > 0 {
		w.WriteModSeq(msg.modSeq)
	}
	if options.GmailMsgID {
		w.WriteGmailMsgID(msg.id)
	}
	if options.GmailThreadID {
		w.WriteGmailThreadID(msg.thread)
	}
	if options.GmailLabels {
		w.WriteGmailLabels(sess.u.labels(msg))
	}
	if options.BodyStructure != nil {
		w.WriteBodyStructure(imapserver.ExtractBodyStructure(bytes.NewReader(msg.buf)))
	}
	for _, bs := range options.BodySection {
		buf := imapserver.ExtractBodySection(bytes.NewReader(msg.buf), bs)
		wc := w.WriteBodySection(bs, int64(len(buf)))
		_, werr := wc.Write(buf)
		cerr := wc.Close()
		if werr != nil {
			return werr
		}
		if cerr != nil {
			return cerr
		}
	}
	return w.Close()
}

func (sess *gmailSession) Store(w *imapserver.FetchWriter, numSet imap.NumSet, flags *imap.StoreFlags, options *imap.StoreOptions) error {
	u := sess.u
	u.mu.Lock()
	defer u.mu.Unlock()
	var err error
	sess.forEach(numSet, func(seqNum uint32, e *gmailEntry) {
		if err != nil {
			return
		}
		if options != nil && options.UnchangedSince > 0 && e.msg.modSeq > options.UnchangedSince {
			return
		}
		set := func(f imap.Flag, on bool) {
			if strings.EqualFold(string(f), string(imap.FlagDeleted)) {
				e.deleted = on
				return
			}
			if on {
				e.msg.flags[canonicalFlag(f)] = true
			} else {
				delete(e.msg.flags, canonicalFlag(f))
			}
		}
		if flags.Op == imap.StoreFlagsSet {
			for f := range e.msg.flags {
				delete(e.msg.flags, f)
			}
			e.deleted = false
		}
		for _, f := range flags.Flags {
			set(f, flags.Op != imap.StoreFlagsDel)
		}
		e.msg.modSeq = u.nextModSeq()
		if !flags.Silent {
			err = sess.fetchEntry(w.CreateMessage(seqNum), e, &imap.FetchOptions{Flags: true, ModSeq: true})
		}
	})
	return err
}

func (sess *gmailSession) StoreGmailLabels(w *imapserver.FetchWriter, numSet imap.NumSet, store *imap.StoreGmailLabels) error {
	u := sess.u
	u.mu.Lock()
	defer u.mu.Unlock()
	var boxes []*gmailBox
	for _, l := range store.Labels {
		if b := u.boxOfLabel(l); b != nil {
			boxes = append(boxes, b)
		}
	}
	var err error
	sess.forEach(numSet, func(seqNum uint32, e *gmailEntry) {
		if err != nil {
			return
		}
		msg := e.msg
		switch store.Op {
		case imap.StoreFlagsSet:
			for _, b := range u.boxes {
				if !isView(b.name) && !slices.Contains(boxes, b) {
					u.removeEntry(b, msg)
				}
			}
			fallthrough
		case imap.StoreFlagsAdd:
			for _, b := range boxes {
				u.addLabel(msg, b)
			}
		case imap.StoreFlagsDel:
			for _, b := range boxes {
				if !isView(b.name) {
					u.removeEntry(b, msg)
				}
			}
		}
		if !store.Silent && sess.box.entry(msg) != nil {
			err = sess.fetchEntry(w.CreateMessage(seqNum), e, &imap.FetchOptions{GmailLabels: true})
		}
	})
	return err
}

func (sess *gmailSession) Expunge(w *imapserver.ExpungeWriter, uids *imap.UIDSet) error {
	u := sess.u
	u.mu.Lock()
	defer u.mu.Unlock()
	for i := len(sess.box.entries) - 1; i >= 0; i-- {
		e := sess.box.entries[i]
		if !e.deleted || (uids != nil && !uids.Contains(e.uid)) {
			continue
		}
		u.removeLabel(e.msg, sess.box)
		if err := w.WriteExpunge(uint32(i + 1)); err != nil {
			return err
		}
	}
	return nil
}

func (sess *gmailSession) Copy(numSet imap.NumSet, dest string) (*imap.CopyData, error) {
	u := sess.u
	u.mu.Lock()
	defer u.mu.Unlock()
	db := u.boxes[dest]
	if db == nil {
		return nil, &imap.Error{Type: imap.StatusResponseTypeNo, Code: imap.ResponseCodeTryCreate, Text: "No such mailbox"}
	}
	var src, dst imap.UIDSet
	sess.forEach(numSet, func(seqNum uint32, e *gmailEntry) {
		src.AddNum(e.uid)
		dst.AddNum(u.addLabel(e.msg, db))
	})
	return &imap.CopyData{UIDValidity: db.validity, SourceUIDs: src, DestUIDs: dst}, nil
}

func (sess *gmailSession) Move(w *imapserver.MoveWriter, numSet imap.NumSet, dest string) error {
	u := sess.u
	u.mu.Lock()
	defer u.mu.Unlock()
	db := u.boxes[dest]
	if db == nil {
		return &imap.Error{Type: imap.StatusResponseTypeNo, Code: imap.ResponseCodeTryCreate, Text: "No such mailbox"}
	}
	var src, dst imap.UIDSet
	var moved []*gmailMessage
	sess.forEach(numSet, func(seqNum uint32, e *gmailEntry) {
		src.AddNum(e.uid)
		moved = append(moved, e.msg)
	})
	// the sequence numbers of the messages before they are moved
	var seqNums []uint32
	for i, e := range sess.box.entries {
		if slices.Contains(moved, e.msg) {
			seqNums = append(seqNums, uint32(i+1))
		}
	}
	for _, msg := range moved {
		dst.AddNum(u.addLabel(msg, db))
		if !isView(sess.box.name) || db.name == gmailAllMail {
			u.removeEntry(sess.box, msg) // else out of the view by the label
		}
	}
	if err := w.WriteCopyData(&imap.CopyData{UIDValidity: db.validity, SourceUIDs: src, DestUIDs: dst}); err != nil {
		return err
	}
	for i := len(seqNums) - 1; i >= 0; i-- {
		if err := w.WriteExpunge(seqNums[i]); err != nil {
			return err
		}
	}
	return nil
}

func (sess *gmailSession) Search(kind imapserver.NumKind, criteria *imap.SearchCriteria, options *imap.SearchOptions) (*imap.SearchData, error) {
	u := sess.u
	u.mu.Lock()
	defer u.mu.Unlock()
	data := &imap.SearchData{}
	var seqs imap.SeqSet
	var uids imap.UIDSet
	last := uint32(len(sess.box.entries))
	for i, e := range sess.box.entries {
		seqNum := uint32(i + 1)
		if !gmailMatch(e, seqNum, last, sess.box.uidNext-1, criteria) {
			continue
		}
		seqs.AddNum(seqNum)
		uids.AddNum(e.uid)
		data.Count++
	}
	if kind == imapserver.NumKindUID {
		data.All = uids
	} else {
		data.All = seqs
	}
	data.ModSeq = u.modSeq
	return data, nil
}

// gmailMatch returns whether the given entry matches the given criteria.
func gmailMatch(e *gmailEntry, seqNum, last uint32, lastUID imap.UID, c *imap.SearchCriteria) bool {
	for _, set := range c.SeqNum {
		if !seqSetContains(set, seqNum, last) {
			return false
		}
	}
	for _, set := range c.UID {
		if !uidSetContains(set, e.uid, lastUID) {
			return false
		}
	}
	has := func(f imap.Flag) bool {
		if strings.EqualFold(string(f), string(imap.FlagDeleted)) {
			return e.deleted
		}
		return e.msg.flags[canonicalFlag(f)]
	}
	for _, f := range c.Flag {
		if !has(f) {
			return false
		}
	}
	for _, f := range c.NotFlag {
		if has(f) {
			return false
		}
	}
	if c.ModSeq != nil && e.msg.modSeq < c.ModSeq.ModSeq {
		return false
	}
	text := strings.ToLower(string(e.msg.buf))
	for _, t := range append(slices.Clone(c.Text), c.Body...) {
		if !strings.Contains(text, strings.ToLower(t)) {
			return false
		}
	}
	if c.GmailRaw != "" {
		for _, w := range strings.Fields(strings.ToLower(c.GmailRaw)) {
			if !strings.Contains(text, w) {
				return false
			}
		}
	}
	if c.GmailThreadID != 0 && e.msg.thread != c.GmailThreadID {
		return false
	}
	for _, not := range c.Not {
		if gmailMatch(e, seqNum, last, lastUID, &not) {
			return false
		}
	}
	for _, or := range c.Or {
		if !gmailMatch(e, seqNum, last, lastUID, &or[0]) && !gmailMatch(e, seqNum, last, lastUID, &or[1]) {
			return false
		}
	}
	return true
}

// stored returns the messages of the user for saving, with their labels,
// and the mailboxes of the user, without messages.
func (u *gmailUser) stored() []storedMailbox {
	u.mu.Lock()
	defer u.mu.Unlock()
	var mbs []storedMailbox
	for name := range u.boxes {
		mbs = append(mbs, storedMailbox{Name: name})
	}
	all := storedMailbox{}
	ids := slices.Sorted(maps.Keys(u.msgs))
	for _, id := range ids {
		msg := u.msgs[id]
		var labels []string
		var flags []imap.Flag
		for name, b := range u.boxes {
			if name != gmailAllMail && b.entry(msg) != nil {
				labels = append(labels, name)
			}
		}
		for f := range msg.flags {
			flags = append(flags, systemFlag(f))
		}
		all.Messages = append(all.Messages, storedMessage{Flags: flags, Time: msg.t, Data: msg.buf, Labels: labels})
	}
	return append(mbs, all)
}

// load adds the saved mailboxes and messages of [gmailUser.stored].
func (u *gmailUser) load(mbs []storedMailbox) {
	u.mu.Lock()
	defer u.mu.Unlock()
	for _, mb := range mbs {
		if mb.Name != "" {
			u.createLocked(mb.Name)
		}
		for _, m := range mb.Messages {
			u.appendMessage(m.Data, m.Flags, m.Time, m.Labels...)
		}
	}
}
