// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"cogentcore.org/core/core"
	"github.com/emersion/go-message/mail"
)

// mailto is the data of a mailto: URL (RFC 6068).
type mailto struct {
	to, cc, bcc   []*mail.Address
	subject, body string
}

// isMailto returns whether the given URL is a mailto: URL.
func isMailto(u string) bool {
	return len(u) >= 7 && strings.EqualFold(u[:7], "mailto:")
}

// parseMailto parses the given mailto: URL: its addresses, and those of its
// to, cc, and bcc fields, and its subject and body.
func parseMailto(s string) (*mailto, error) {
	rest := s[len("mailto:"):]
	addrs, query, _ := strings.Cut(rest, "?")
	// a + is itself in a mailto: URL, not a space as in a form
	q, err := url.ParseQuery(strings.ReplaceAll(query, "+", "%2B"))
	if err != nil {
		return nil, fmt.Errorf("the link %q: %w", s, err)
	}
	m := &mailto{subject: q.Get("subject"), body: q.Get("body")}
	parse := func(lists ...string) ([]*mail.Address, error) {
		var res []*mail.Address
		for _, list := range lists {
			list, err := url.PathUnescape(list)
			if err != nil {
				return nil, err
			}
			if strings.TrimSpace(list) == "" {
				continue
			}
			as, err := mail.ParseAddressList(list)
			if err != nil {
				return nil, err
			}
			res = append(res, as...)
		}
		return res, nil
	}
	for _, f := range []struct {
		dst   *[]*mail.Address
		lists []string
	}{{&m.to, append([]string{addrs}, q["to"]...)}, {&m.cc, q["cc"]}, {&m.bcc, q["bcc"]}} {
		as, err := parse(f.lists...)
		if err != nil {
			return nil, fmt.Errorf("the addresses of the link %q: %w", s, err)
		}
		*f.dst = as
	}
	return m, nil
}

// linkAddress returns the email address of the given text pasted into an
// address field: the first address of a mailto: link, without the rest
// of the link, or else the text itself, without spaces around it.
func linkAddress(text string) string {
	text = strings.TrimSpace(text)
	if !isMailto(text) {
		return text
	}
	if m, err := parseMailto(text); err == nil && len(m.to) > 0 {
		return m.to[0].Address
	}
	addr, _, _ := strings.Cut(text[len("mailto:"):], "?")
	if u, err := url.PathUnescape(addr); err == nil {
		addr = u
	}
	return addr
}

// replyMailto opens a new message to the addresses of the given mailto:
// URL, clicked in the given message, as a reply to it, which can be
// turned into a reply to all of it in the compose window (see
// [SendMessage.replyAll]), returning the body of the window.
func (a *App) replyMailto(u string, cm *CacheMessage) *core.Body {
	m, err := parseMailto(u)
	if err != nil {
		core.ErrorSnackbar(a, err, "Error opening the link")
		return nil
	}
	a.composeMessage = &SendMessage{To: m.to, Cc: m.cc, Bcc: m.bcc}
	if len(a.composeMessage.To) == 0 {
		a.composeMessage.To = []*mail.Address{{}}
	}
	a.replyTo(cm, "Reply", false)
	if m.subject != "" {
		a.composeMessage.Subject = m.subject
	}
	if m.body != "" {
		a.composeMessage.body = m.body + a.composeMessage.body
	}
	a.composeMessage.replyAll = &replyAll{
		to: slices.Concat(IMAPToMailAddresses(cm.From), IMAPToMailAddresses(cm.To)),
		cc: IMAPToMailAddresses(cm.Cc),
	}
	return a.compose("Reply")
}

// replyAll are the recipients of a reply to all of a message, added to a
// message being written with the Reply all button of its compose window.
type replyAll struct {
	to, cc []*mail.Address
}

// addReplyAll adds the recipients of [SendMessage.replyAll] to the message,
// those that are not already among its recipients, nor its sender.
func (sm *SendMessage) addReplyAll() {
	if sm.replyAll == nil {
		return
	}
	has := func(addr string) bool {
		for _, as := range [][]*mail.Address{sm.From, sm.To, sm.Cc, sm.Bcc} {
			if slices.ContainsFunc(as, func(ad *mail.Address) bool {
				return ad != nil && strings.EqualFold(strings.TrimSpace(ad.Address), addr)
			}) {
				return true
			}
		}
		return false
	}
	add := func(dst *[]*mail.Address, as []*mail.Address) {
		*dst = nonEmpty(*dst)
		for _, ad := range as {
			if addr := strings.TrimSpace(ad.Address); addr != "" && !has(addr) {
				c := *ad
				*dst = append(*dst, &c)
			}
		}
	}
	add(&sm.To, sm.replyAll.to)
	add(&sm.Cc, sm.replyAll.cc)
	if len(sm.To) == 0 {
		sm.To = []*mail.Address{{}}
	}
	sm.replyAll = nil
}
