// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testserver

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-message/mail"
)

// person is a sender or recipient of the sample messages.
type person struct {
	name, addr string
}

func (p person) address() *mail.Address { return &mail.Address{Name: p.name, Address: p.addr} }

var (
	alice  = person{"Alice Archer", "alice@mail.test"}
	bob    = person{"Bob Baker", "bob@mail.test"}
	carol  = person{"Carol Chen", "carol@example.org"}
	dan    = person{"Dan Diaz", "dan@example.org"}
	erin   = person{"Erin Ellis", "erin@lab.example.edu"}
	frank  = person{"Frank Fu", "frank@lab.example.edu"}
	mom    = person{"Mom", "mom@family.example"}
	shop   = person{"Shop Orders", "orders@shop.example"}
	fly    = person{"Fly Air", "travel@fly.example"}
	weekly = person{"Go Weekly", "news@goweekly.example"}
	nuts   = person{"golang-nuts", "golang-nuts@lists.example"}
)

// topic is a kind of sample conversation, filed in a folder.
type topic struct {
	folder   string
	people   []person
	subjects []string
	// count is the number of conversations of the topic.
	count int
	// maxReplies is the maximum number of replies in a conversation.
	maxReplies int
	// newsletter conversations are HTML only, and never replied to.
	newsletter bool
	// attach is the probability of an attachment.
	attach float64
}

var topics = []topic{
	{"Projects/Alpha", []person{alice, erin, carol}, []string{
		"Alpha design review", "Alpha milestone %d", "Data for the alpha model", "Alpha meeting notes",
		"Question about the alpha results", "Alpha figures, draft %d", "Alpha grant timeline"}, 22, 5, false, 0.3},
	{"Projects/Beta", []person{bob, frank}, []string{
		"Beta build is failing", "Beta release %d", "Beta: API changes", "Beta benchmarks",
		"Beta code review", "Beta roadmap"}, 18, 4, false, 0.2},
	{"Admin/Receipts", []person{shop}, []string{
		"Your order #%d has shipped", "Receipt for order #%d", "Order #%d confirmation"}, 15, 0, false, 0.5},
	{"Admin/Travel", []person{fly, dan}, []string{
		"Your flight confirmation %d", "Itinerary update", "Check in for your flight", "Conference travel plans"}, 10, 2, false, 0.3},
	{"Lists/go-nuts", []person{nuts}, []string{
		"[go-nuts] generics question %d", "[go-nuts] iterators and range funcs", "[go-nuts] best way to test %d",
		"[go-nuts] proposal: %d", "[go-nuts] gopls slow on large repo"}, 25, 4, false, 0},
	{"Family", []person{mom, dan}, []string{
		"Photos from the weekend", "Dinner on Sunday?", "Happy birthday!", "Holiday plans %d"}, 12, 3, false, 0.4},
	{"Newsletters", []person{weekly}, []string{"Go Weekly issue %d"}, 14, 0, true, 0},
}

var paragraphs = []string{
	"Thanks for sending this over. I had a look and I think we are mostly on the same page, but there are a couple of details we should go over before we finalize anything.",
	"Can you take a look at the attached and let me know what you think? No rush, but it would be good to have it settled by the end of the week.",
	"I ran everything again with the new settings and the numbers look much more reasonable now. The variance is still higher than I would like.",
	"Sounds good to me. Let's plan to talk about it at the next meeting.",
	"Quick update: the first part is done, the second part is still in progress, and the third part is blocked on the review.",
	"I'm not sure I follow the second point. Could you explain what you mean by that, maybe with an example?",
	"Here are my notes from today. Please add anything I missed.",
	"Great news, it works! Thanks for all of your help with this.",
}

var markdownBodies = []string{
	"Here is the plan:\n\n1. Finish the **draft**\n2. Send it around for comments\n3. Submit by *Friday*\n\nLet me know if that works.",
	"The fix is a one-liner:\n\n```go\nif err != nil {\n\treturn err\n}\n```\n\nSee the [issue](https://example.org/issue/1) for details.",
	"## Summary\n\n- results are in the `data` folder\n- the plots are attached\n- the code is on the `main` branch\n\n> Note: the old results are no longer valid.",
}

// Seed adds a deterministic set of sample messages, for the given random
// seed, to the accounts of the server, with dates relative to now: a tree
// of folders for the first account, with conversations filed in them and
// newer ones still in the INBOX, some with more than one label; and
// conversations with the first account in the others.
func (s *Server) Seed(seed uint64) error {
	g := &seeder{s: s, rnd: rand.New(rand.NewPCG(seed, seed+1)), now: time.Now()}
	return g.seed()
}

type seeder struct {
	s   *Server
	rnd *rand.Rand
	now time.Time
	n   int
}

// sample is a sample message.
type sample struct {
	from      person
	to        []person
	subject   string
	date      time.Time
	id        string
	refs      []string
	plain     string
	html      string
	attach    []Attachment
	mailboxes []string // for the first account
	seen      bool
	flagged   bool
}

// Attachment is an attachment of a message made by [Message].
type Attachment struct {
	Name, ContentType string
	Data              []byte
}

func (g *seeder) seed() error {
	me := person{"You", g.s.Accounts[0]}
	for _, tp := range topics {
		for range tp.count {
			if err := g.conversation(me, tp); err != nil {
				return err
			}
		}
	}
	// a few unfiled ones in the INBOX
	for i := range 6 {
		p := []person{carol, dan, erin, frank}[i%4]
		m := g.message(p, []person{me}, fmt.Sprintf("Hello from %s", p.name), g.now.Add(-time.Duration(i*7+1)*time.Hour), nil)
		m.mailboxes = []string{"INBOX"}
		if err := g.append(me, m); err != nil {
			return err
		}
	}
	return nil
}

// conversation adds a sample conversation for the given topic.
func (g *seeder) conversation(me person, tp topic) error {
	age := time.Duration(g.rnd.Float64()*float64(90*24*time.Hour)) + time.Hour
	start := g.now.Add(-age)
	other := tp.people[g.rnd.IntN(len(tp.people))]
	subject := tp.subjects[g.rnd.IntN(len(tp.subjects))]
	if strings.Contains(subject, "%d") {
		subject = fmt.Sprintf(subject, 100+g.rnd.IntN(900))
	}
	// newer conversations are still in the INBOX, older ones are filed,
	// and some newer ones are labeled while still in the INBOX
	days := age.Hours() / 24
	var mailboxes []string
	switch {
	case days < 10 && g.rnd.Float64() < 0.4:
		mailboxes = []string{"INBOX", tp.folder}
	case days < 10:
		mailboxes = []string{"INBOX"}
	default:
		mailboxes = []string{tp.folder}
	}
	nreplies := 0
	if tp.maxReplies > 0 {
		nreplies = g.rnd.IntN(tp.maxReplies + 1)
	}
	var refs []string
	date := start
	from, to := other, me
	for i := 0; i <= nreplies && date.Before(g.now); i++ {
		subj := subject
		if i > 0 {
			subj = "Re: " + subject
		}
		m := g.message(from, []person{to}, subj, date, refs)
		if tp.newsletter {
			m.plain = ""
			m.html = g.newsletterHTML(subject)
		}
		if g.rnd.Float64() < tp.attach {
			m.attach = append(m.attach, g.attachment())
		}
		m.mailboxes = mailboxes
		if from == me {
			m.mailboxes = append(slices.Clone(mailboxes), g.s.SentMailbox())
		}
		if err := g.append(me, m); err != nil {
			return err
		}
		refs = append(slices.Clone(refs), m.id)
		date = date.Add(time.Duration(g.rnd.Float64()*float64(36*time.Hour)) + 10*time.Minute)
		if tp.people[0] != nuts {
			from, to = to, from
		} else { // a mailing list: someone else replies
			from = []person{carol, dan, erin, frank}[g.rnd.IntN(4)]
			to = nuts
		}
	}
	return nil
}

// message returns a new sample message with a plain text body.
func (g *seeder) message(from person, to []person, subject string, date time.Time, refs []string) *sample {
	g.n++
	m := &sample{from: from, to: to, subject: subject, date: date, refs: refs,
		id: fmt.Sprintf("seed-%d@mail.test", g.n)}
	if g.rnd.Float64() < 0.25 {
		m.plain = markdownBodies[g.rnd.IntN(len(markdownBodies))]
	} else {
		var ps []string
		for range 1 + g.rnd.IntN(3) {
			ps = append(ps, paragraphs[g.rnd.IntN(len(paragraphs))])
		}
		m.plain = "Hi,\n\n" + strings.Join(ps, "\n\n") + "\n\n" + strings.Fields(from.name)[0]
		if g.rnd.Float64() < 0.4 { // also as HTML
			m.html = "<p>Hi,</p><p>" + strings.Join(ps, "</p><p>") + "</p><p><i>" + strings.Fields(from.name)[0] + "</i></p>"
		}
	}
	old := g.now.Sub(date) > 3*24*time.Hour
	m.seen = old || g.rnd.Float64() < 0.4
	m.flagged = g.rnd.Float64() < 0.05
	return m
}

func (g *seeder) newsletterHTML(subject string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<h1>%s</h1><p>The latest news from the Go community.</p><table border=\"1\">", subject)
	fmt.Fprintf(&b, "<tr><th>Article</th><th>Author</th></tr>")
	for i := range 4 {
		fmt.Fprintf(&b, "<tr><td><a href=\"https://example.org/%d\">Article %d</a></td><td>%s</td></tr>", i, i+1, []string{"Carol", "Dan", "Erin", "Frank"}[i])
	}
	b.WriteString("</table><p><small>You are receiving this because you subscribed. <a href=\"https://example.org/unsubscribe\">Unsubscribe</a></small></p>")
	return b.String()
}

// attachment returns a random sample attachment.
func (g *seeder) attachment() Attachment {
	switch g.rnd.IntN(3) {
	case 0:
		return Attachment{Name: "notes.txt", ContentType: "text/plain", Data: []byte("Some notes.\nLine two of the notes.\n")}
	case 1:
		return Attachment{Name: "data.csv", ContentType: "text/csv", Data: []byte("name,value\nalpha,1.5\nbeta,2.25\n")}
	default:
		img := image.NewRGBA(image.Rect(0, 0, 32, 32))
		c := color.RGBA{uint8(g.rnd.IntN(256)), uint8(g.rnd.IntN(256)), 200, 255}
		for y := range 32 {
			for x := range 32 {
				if (x/8+y/8)%2 == 0 {
					img.Set(x, y, c)
				}
			}
		}
		var b bytes.Buffer
		png.Encode(&b, img)
		return Attachment{Name: "image.png", ContentType: "image/png", Data: b.Bytes()}
	}
}

// append adds the given sample message to the mailboxes of the first
// account, and to the other accounts that sent or received it.
func (g *seeder) append(me person, m *sample) error {
	data, err := m.bytes()
	if err != nil {
		return err
	}
	var flags []imap.Flag
	if m.seen || m.from == me {
		flags = append(flags, imap.FlagSeen)
	}
	if m.flagged {
		flags = append(flags, imap.FlagFlagged)
	}
	if err := g.s.Append(me.addr, m.mailboxes, data, flags, m.date); err != nil {
		return err
	}
	// the other accounts of the server have it too
	others := append([]person{m.from}, m.to...)
	for _, p := range others {
		if p == me || !g.s.HasAccount(p.addr) {
			continue
		}
		mb := "INBOX"
		var flags []imap.Flag
		if p == m.from {
			mb = g.s.SentMailbox()
			flags = []imap.Flag{imap.FlagSeen}
		} else if g.now.Sub(m.date) > 5*24*time.Hour {
			flags = []imap.Flag{imap.FlagSeen}
		}
		if err := g.s.Append(p.addr, []string{mb}, data, flags, m.date); err != nil {
			return err
		}
	}
	return nil
}

// bytes returns the message in the RFC 5322 format.
func (m *sample) bytes() ([]byte, error) {
	var h mail.Header
	h.SetDate(m.date)
	h.SetAddressList("From", []*mail.Address{m.from.address()})
	var to []*mail.Address
	for _, p := range m.to {
		to = append(to, p.address())
	}
	h.SetAddressList("To", to)
	h.SetSubject(m.subject)
	h.SetMessageID(m.id)
	if len(m.refs) > 0 {
		h.SetMsgIDList("In-Reply-To", m.refs[len(m.refs)-1:])
		h.SetMsgIDList("References", m.refs)
	}
	return Message(h, m.plain, m.html, m.attach)
}

// Message returns a message with the given header, plain text and HTML
// bodies (either may be empty) and attachments.
func Message(h mail.Header, plain, html string, attach []Attachment) ([]byte, error) {
	var b bytes.Buffer
	if len(attach) == 0 && (plain == "" || html == "") {
		// a single part
		ct, body := "text/plain", plain
		if plain == "" {
			ct, body = "text/html", html
		}
		h.SetContentType(ct, map[string]string{"charset": "utf-8"})
		w, err := mail.CreateSingleInlineWriter(&b, h)
		if err != nil {
			return nil, err
		}
		w.Write([]byte(crlf(body)))
		if err := w.Close(); err != nil {
			return nil, err
		}
		return b.Bytes(), nil
	}
	mw, err := mail.CreateWriter(&b, h)
	if err != nil {
		return nil, err
	}
	iw, err := mw.CreateInline()
	if err != nil {
		return nil, err
	}
	for _, part := range []struct{ ct, body string }{{"text/plain", plain}, {"text/html", html}} {
		if part.body == "" {
			continue
		}
		var ph mail.InlineHeader
		ph.SetContentType(part.ct, map[string]string{"charset": "utf-8"})
		pw, err := iw.CreatePart(ph)
		if err != nil {
			return nil, err
		}
		pw.Write([]byte(crlf(part.body)))
		if err := pw.Close(); err != nil {
			return nil, err
		}
	}
	if err := iw.Close(); err != nil {
		return nil, err
	}
	for _, at := range attach {
		var ah mail.AttachmentHeader
		ah.SetFilename(at.Name)
		ah.SetContentType(at.ContentType, nil)
		aw, err := mw.CreateAttachment(ah)
		if err != nil {
			return nil, err
		}
		aw.Write(at.Data)
		if err := aw.Close(); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func crlf(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\n", "\r\n")
}

// RandomMessage delivers a new random message to the INBOX of the given
// account, from one of a few senders, as if it were just received.
func (s *Server) RandomMessage(account string) error {
	g := &seeder{s: s, rnd: rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 7)), now: time.Now()}
	p := []person{carol, dan, erin, frank, mom}[g.rnd.IntN(5)]
	subjects := []string{"Quick question", "Are you free tomorrow?", "Follow up", "FYI", "Draft for review"}
	m := g.message(p, []person{{"", account}}, subjects[g.rnd.IntN(len(subjects))], g.now, nil)
	m.id = strings.Trim(s.newMessageID(), "<>")
	m.seen = false
	data, err := m.bytes()
	if err != nil {
		return err
	}
	if err := s.Append(account, []string{"INBOX"}, data, nil, time.Time{}); err != nil {
		return err
	}
	if s.OnDeliver != nil {
		s.OnDeliver([]string{account})
	}
	return nil
}
