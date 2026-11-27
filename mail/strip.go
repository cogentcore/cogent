// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"github.com/cogentcore/go-imap/v2"
	"github.com/cogentcore/go-imap/v2/imapclient"
)

// removedHeader is the header of the part of a message that replaces an
// attachment removed from it, with the name of the attachment, which is
// shown as removed when reading it (see [App.RemoveAttachments]).
const removedHeader = "X-Attachment-Removed"

// RemoveAttachments removes the attachments of the messages being read,
// to save space, replacing each with a note of its name and size, and
// replaces the messages on the server with these versions. The
// attachments cannot be recovered after, so it asks first.
func (a *App) RemoveAttachments() { //types:add
	var msgs []*CacheMessage
	n, notDownloaded := 0, 0
	for _, cm := range a.readMessages() {
		if cm.isLocal() || cm.Attachments == 0 && cm.AttachmentsCounted {
			continue
		}
		if cm.Filename == "" {
			notDownloaded++
			continue
		}
		msgs = append(msgs, cm)
		n += cm.Attachments
	}
	if len(msgs) == 0 {
		msg := "There are no attachments to remove"
		if notDownloaded > 0 {
			msg = "The message is not downloaded yet: try again once it is"
		}
		core.MessageSnackbar(a, msg)
		return
	}
	d := core.NewBody("Remove attachments?")
	what := fmt.Sprintf("the attachments of %d messages", len(msgs))
	if len(msgs) == 1 {
		what = "the attachments of the message"
		if n == 1 {
			what = "the attachment of the message"
		}
	}
	core.NewText(d).SetType(core.TextSupporting).SetText("Remove " + what +
		", here and on the server, to save space? Each is replaced with a note of its name and size. They cannot be recovered.")
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Remove").OnClick(func(e events.Event) {
			email := a.currentEmail
			a.actionOn(email, msgs, func(c *imapclient.Client, cm *CacheMessage) error {
				return a.removeAttachments(c, email, cm)
			}, a.updateReading)
		})
	})
	d.RunDialog(a)
}

// removeAttachments removes the attachments of the given message of the
// given account (see [App.RemoveAttachments]), with its client, replacing
// it on the server, and its file and search document here.
func (a *App) removeAttachments(c *imapclient.Client, email string, cm *CacheMessage) error {
	a.AsyncLock()
	filename := cm.Filename
	a.AsyncUnlock()
	if filename == "" {
		return nil
	}
	dir := a.messagesDir(email)
	orig, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		return err
	}
	stripped, removed, err := stripAttachments(orig)
	if err != nil {
		return fmt.Errorf("removing the attachments of %q: %w", cm.Subject, err)
	}
	if len(removed) == 0 {
		return nil
	}
	if a.gmail[email] {
		err = a.replaceGmailMessage(c, email, cm, orig, stripped)
	} else {
		err = a.replaceMessage(c, email, cm, stripped)
	}
	if err != nil {
		return fmt.Errorf("removing the attachments of %q: %w", cm.Subject, err)
	}

	// its new file, read again when it is shown, and search document
	newName := shardName(messageFilename())
	fn := filepath.Join(dir, newName)
	if err := os.MkdirAll(filepath.Dir(fn), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(fn, stripped, 0o600); err != nil {
		return err
	}
	os.Remove(filepath.Join(dir, filename))
	a.AsyncLock()
	cm.Filename = newName
	cm.Attachments, cm.AttachmentsCounted = 0, true
	cm.parsed.attachments = nil
	key := cm.key
	a.AsyncUnlock()
	if doc, err := readSearchDoc(fn); err == nil {
		return a.searches[email].add(map[string]*searchDoc{key: doc})
	}
	return nil
}

// appendFlags returns the flags of the given message for appending it.
func appendFlags(cm *CacheMessage) []imap.Flag {
	return slices.DeleteFunc(slices.Clone(cm.Flags), func(f imap.Flag) bool {
		return f == "\\Recent" // not settable
	})
}

// appendTo appends the given message to the given mailbox with the given
// options, returning its UID there, or 0 if the server does not say.
func appendTo(c *imapclient.Client, mailbox string, msg []byte, opts *imap.AppendOptions) (imap.UID, error) {
	cmd := c.Append(mailbox, int64(len(msg)), opts)
	if _, err := cmd.Write(msg); err != nil {
		cmd.Close()
		return 0, err
	}
	if err := cmd.Close(); err != nil {
		return 0, err
	}
	data, err := cmd.Wait()
	if err != nil {
		return 0, err
	}
	return data.UID, nil
}

// expunge permanently deletes the message with the given UID in the given
// mailbox, which must be selected.
func expunge(c *imapclient.Client, uid imap.UID) error {
	var uids imap.UIDSet
	uids.AddNum(uid)
	err := c.Store(uids, &imap.StoreFlags{Op: imap.StoreFlagsAdd, Silent: true, Flags: []imap.Flag{imap.FlagDeleted}}, nil).Close()
	if err != nil {
		return err
	}
	return c.UIDExpunge(uids).Close()
}

// replaceMessage replaces the given message of the given account on a
// server other than Gmail with the given version of it, in each of its
// mailboxes: the new one is appended, and then the old one deleted.
func (a *App) replaceMessage(c *imapclient.Client, email string, cm *CacheMessage, msg []byte) error {
	a.AsyncLock()
	labels := slices.Clone(cm.Labels)
	flags := appendFlags(cm)
	a.AsyncUnlock()
	for i, l := range labels {
		if l.UID == 0 {
			continue
		}
		uid, err := appendTo(c, l.Name, msg, &imap.AppendOptions{Flags: flags, Time: cm.Date})
		if err != nil {
			return err
		}
		if err := a.selectMailbox(c, email, l.Name); err != nil {
			return err
		}
		if err := expunge(c, l.UID); err != nil {
			return err
		}
		// a UID of 0 (unknown) is set in the next sync, by its Message-ID
		labels[i].UID = uid
	}
	a.AsyncLock()
	cm.Labels = labels
	a.AsyncUnlock()
	a.SyncNow()
	return nil
}

// replaceGmailMessage replaces the given message of the given account on
// Gmail, whose original contents are given, with the given version of it.
// Gmail keeps only one message with each Message-ID, so the old one is
// deleted first (moved to the Trash and deleted there), and the new one is
// appended to its mailbox (All Mail, or the Trash or the Spam), with its
// labels; if that fails, the old one is put back.
func (a *App) replaceGmailMessage(c *imapclient.Client, email string, cm *CacheMessage, orig, msg []byte) error {
	a.AsyncLock()
	view, ok := viewLabel(cm)
	var gmailLabels []string
	for _, l := range cm.Labels {
		if l.UID == 0 {
			gmailLabels = append(gmailLabels, a.gmailLabel(email, l.Name))
		}
	}
	flags := appendFlags(cm)
	oldKey := cm.key
	a.AsyncUnlock()
	if !ok {
		return fmt.Errorf("the message is not synced yet")
	}
	trash := a.specialUse[email][imap.MailboxAttrTrash]
	if trash == "" {
		return fmt.Errorf("no trash mailbox for %s", email)
	}

	// deleting the old one
	if err := a.selectMailbox(c, email, view.Name); err != nil {
		return err
	}
	uid := view.UID
	if view.Name != trash {
		md, err := c.Move(view.UIDSet(), trash).Wait()
		if err != nil {
			return err
		}
		uid = 0
		if md != nil {
			if dst, ok := md.DestUIDs.(imap.UIDSet); ok {
				if uids, ok := dst.Nums(); ok && len(uids) == 1 {
					uid = uids[0]
				}
			}
		}
		if uid == 0 {
			return fmt.Errorf("moved to %s, but its UID there is not known", trash)
		}
		if err := a.selectMailbox(c, email, trash); err != nil {
			return err
		}
	}
	if err := expunge(c, uid); err != nil {
		return err
	}

	// appending the new one, or else the old one back
	opts := &imap.AppendOptions{Flags: flags, Time: cm.Date}
	newUID, err := appendTo(c, view.Name, msg, opts)
	if err != nil {
		if _, err2 := appendTo(c, view.Name, orig, opts); err2 != nil {
			return fmt.Errorf("%w; and putting the original back in %s: %w", err, view.Name, err2)
		}
		a.SyncNow()
		return err
	}
	if err := a.selectMailbox(c, email, view.Name); err != nil {
		return err
	}
	if newUID == 0 {
		sd, err := c.UIDSearch(&imap.SearchCriteria{Header: []imap.SearchCriteriaHeaderField{{Key: "Message-ID", Value: cm.MessageID}}}, nil).Wait()
		if err != nil {
			return err
		}
		uids := sd.AllUIDs()
		if len(uids) == 0 {
			a.SyncNow()
			return nil
		}
		newUID = uids[len(uids)-1]
	}
	var uids imap.UIDSet
	uids.AddNum(newUID)
	if len(gmailLabels) > 0 {
		err := c.StoreGmailLabels(uids, &imap.StoreGmailLabels{Op: imap.StoreFlagsAdd, Silent: true, Labels: gmailLabels}).Close()
		if err != nil {
			return err
		}
	}
	mds, err := c.Fetch(uids, &imap.FetchOptions{UID: true, GmailMsgID: true, GmailThreadID: true}).Collect()
	if err != nil {
		return err
	}
	if len(mds) == 0 || mds[0].GmailMsgID == 0 {
		a.SyncNow()
		return nil
	}

	// it has a new Gmail message ID, its key
	newKey := gmailKey(mds[0].GmailMsgID)
	a.AsyncLock()
	delete(a.cache[email], oldKey)
	cm.key = newKey
	cm.GmailMsgID, cm.GmailThreadID = mds[0].GmailMsgID, mds[0].GmailThreadID
	labels := []Label{{view.Name, newUID}}
	for _, l := range cm.Labels {
		if l.UID == 0 {
			labels = append(labels, l)
		}
	}
	cm.Labels = labels
	a.cache[email][newKey] = cm
	a.AsyncUnlock()
	if err := a.stores[email].remove(oldKey); err != nil {
		return err
	}
	return a.searches[email].remove(oldKey)
}

// stripAttachments returns the given message (in MIME) without its
// attachments, each replaced with a short text part noting its name and
// size, with the [removedHeader], and the names of those removed. The
// header of the message and its other parts are kept as they are.
// A message without attachments is returned as it is.
func stripAttachments(msg []byte) ([]byte, []string, error) {
	header, body := splitHeader(msg)
	h, err := textproto.NewReader(bufio.NewReader(bytes.NewReader(append(slices.Clip(header), "\r\n"...)))).ReadMIMEHeader()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, nil, err
	}
	mt, params, _ := mime.ParseMediaType(h.Get("Content-Type"))
	if !strings.HasPrefix(mt, "multipart/") || params["boundary"] == "" {
		return msg, nil, nil // a single part, the text
	}
	var removed []string
	nb, err := stripMultipart(body, params["boundary"], &removed)
	if err != nil {
		return nil, nil, err
	}
	if len(removed) == 0 {
		return msg, nil, nil
	}
	return append(slices.Clip(header), nb...), removed, nil
}

// splitHeader returns the header of the given message, with the blank
// line after it, and its body.
func splitHeader(msg []byte) (header, body []byte) {
	i := bytes.Index(msg, []byte("\r\n\r\n"))
	n := 4
	if j := bytes.Index(msg, []byte("\n\n")); j >= 0 && (i < 0 || j < i) {
		i, n = j, 2
	}
	if i < 0 {
		return msg, nil
	}
	return msg[:i+n], msg[i+n:]
}

// stripMultipart returns the given body of a multipart entity with the
// given boundary without its attachments, adding their names to removed.
func stripMultipart(body []byte, boundary string, removed *[]string) ([]byte, error) {
	mr := multipart.NewReader(bytes.NewReader(body), boundary)
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	if err := mw.SetBoundary(boundary); err != nil {
		return nil, err
	}
	for {
		p, err := mr.NextRawPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(p)
		if err != nil {
			return nil, err
		}
		h := p.Header
		mt, params, _ := mime.ParseMediaType(h.Get("Content-Type"))
		switch {
		case strings.HasPrefix(mt, "multipart/"):
			if data, err = stripMultipart(data, params["boundary"], removed); err != nil {
				return nil, err
			}
		case isAttachment(h, mt):
			name := attachmentName(h, params)
			*removed = append(*removed, name)
			note := fmt.Sprintf("[Attachment removed: %s (%s)]", name, sizeText(decodedSize(h, data)))
			h = textproto.MIMEHeader{}
			h.Set("Content-Type", "text/plain; charset=utf-8")
			h.Set("Content-Disposition", "inline")
			h.Set("Content-Transfer-Encoding", "8bit")
			h.Set(removedHeader, mime.QEncoding.Encode("utf-8", name))
			data = []byte(note)
		}
		pw, err := mw.CreatePart(h)
		if err != nil {
			return nil, err
		}
		if _, err := pw.Write(data); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// isAttachment returns whether the part with the given header and media
// type is an attachment, as it is shown when reading it (as in the mail
// reader of go-message): one marked as an attachment, or one that is not
// text and not marked as inline.
func isAttachment(h textproto.MIMEHeader, mt string) bool {
	disp, _, _ := mime.ParseMediaType(h.Get("Content-Disposition"))
	disp = strings.ToLower(disp)
	return disp == "attachment" || (disp != "inline" && !strings.HasPrefix(mt, "text/"))
}

// attachmentName returns the file name of the attachment with the given
// header and parameters of its type.
func attachmentName(h textproto.MIMEHeader, params map[string]string) string {
	_, dp, _ := mime.ParseMediaType(h.Get("Content-Disposition"))
	name := dp["filename"]
	if name == "" {
		name = params["name"]
	}
	if dec, err := new(mime.WordDecoder).DecodeHeader(name); err == nil {
		name = dec
	}
	if name == "" {
		name = "attachment"
	}
	return name
}

// decodedSize returns the size of the given data of a part with the given
// header, once decoded.
func decodedSize(h textproto.MIMEHeader, data []byte) int {
	if strings.EqualFold(strings.TrimSpace(h.Get("Content-Transfer-Encoding")), "base64") {
		clean := bytes.Map(func(r rune) rune {
			if r == '\r' || r == '\n' || r == ' ' || r == '\t' {
				return -1
			}
			return r
		}, data)
		return base64.StdEncoding.DecodedLen(len(clean)) - bytes.Count(clean, []byte("="))
	}
	return len(data)
}

// sizeText returns the given number of bytes as text, as 1.2 MB.
func sizeText(n int) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d bytes", n)
}
