// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/lang/en"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/emersion/go-message/mail"
	"golang.org/x/net/html"
)

// searchIndex is the full-text search index of the messages of an
// account: a Bleve index of their [searchDoc]s, by their keys in the cache.
type searchIndex struct {
	idx bleve.Index
}

// searchDoc is the text of a message that is searched.
type searchDoc struct {
	From, To, Cc, Subject string

	// Body is the text of the message: the plain text, or else the
	// text of the HTML.
	Body string

	// Attachments are the names of the files of the attachments.
	Attachments string

	// Date is the date of the message.
	Date time.Time
}

// searchMapping returns the mapping of the [searchDoc]s: English text,
// with stemming, for the subject and the body, and the standard
// analysis (words) for the addresses and the names of the attachments.
func searchMapping() mapping.IndexMapping {
	text := bleve.NewTextFieldMapping()
	text.Analyzer = en.AnalyzerName
	words := bleve.NewTextFieldMapping()
	words.Analyzer = "standard"
	doc := bleve.NewDocumentMapping()
	for _, f := range []string{"From", "To", "Cc", "Attachments"} {
		doc.AddFieldMappingsAt(f, words)
	}
	doc.AddFieldMappingsAt("Subject", text)
	doc.AddFieldMappingsAt("Body", text)
	doc.AddFieldMappingsAt("Date", bleve.NewDateTimeFieldMapping())
	m := bleve.NewIndexMapping()
	m.DefaultMapping = doc
	m.DefaultAnalyzer = en.AnalyzerName
	return m
}

// openSearch opens the search index in the given directory,
// creating it if needed.
func openSearch(dir string) (*searchIndex, error) {
	idx, err := bleve.Open(dir)
	if errors.Is(err, bleve.ErrorIndexPathDoesNotExist) {
		if err := os.MkdirAll(filepath.Dir(dir), 0o700); err != nil {
			return nil, err
		}
		idx, err = bleve.New(dir, searchMapping())
	}
	if err != nil {
		return nil, fmt.Errorf("opening the search index: %w", err)
	}
	return &searchIndex{idx: idx}, nil
}

// Close closes the index.
func (si *searchIndex) Close() error {
	return si.idx.Close()
}

// add indexes the given documents, by their keys, in one batch.
func (si *searchIndex) add(docs map[string]*searchDoc) error {
	if len(docs) == 0 {
		return nil
	}
	b := si.idx.NewBatch()
	for k, d := range docs {
		if err := b.Index(k, d); err != nil {
			return err
		}
	}
	return si.idx.Batch(b)
}

// remove removes the documents with the given keys from the index.
func (si *searchIndex) remove(keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	b := si.idx.NewBatch()
	for _, k := range keys {
		b.Delete(k)
	}
	return si.idx.Batch(b)
}

// has returns whether the document with the given key is indexed.
func (si *searchIndex) has(key string) bool {
	d, err := si.idx.Document(key)
	return err == nil && d != nil
}

// search returns the keys of the messages that match the given query (see
// [searchIndex.query]), the best matches first, up to the given maximum
// number.
func (si *searchIndex) search(query string, max int) ([]string, error) {
	q := si.query(query)
	if q == nil {
		return nil, nil
	}
	req := bleve.NewSearchRequestOptions(q, max, 0, false)
	res, err := si.idx.Search(req)
	if err != nil {
		return nil, err
	}
	keys := make([]string, len(res.Hits))
	for i, h := range res.Hits {
		keys[i] = h.ID
	}
	return keys, nil
}

// searchFields are the fields that can be searched on their own, as
// field:words, by their names in lower case.
var searchFields = map[string]string{"from": "From", "to": "To", "cc": "Cc",
	"subject": "Subject", "body": "Body", "attachment": "Attachments", "attachments": "Attachments"}

// query returns the query for the given search: all of its terms must
// match, where a term is a word, a "phrase", or either of them after a
// field name and a colon, as from:alice, and a term with a - before it
// must not match. Words that are not searched, such as "the", are
// ignored. It returns nil if there is nothing to search for.
func (si *searchIndex) query(search string) query.Query {
	var must, mustNot []query.Query
	for _, term := range searchTerms(search) {
		neg := strings.HasPrefix(term, "-")
		term = strings.TrimPrefix(term, "-")
		field := ""
		if name, rest, ok := strings.Cut(term, ":"); ok {
			if f, ok := searchFields[strings.ToLower(name)]; ok {
				field, term = f, rest
			}
		}
		phrase := strings.HasPrefix(term, `"`)
		term = strings.Trim(term, `"`)
		if term == "" || (field == "" && !si.analyzes(term)) {
			continue
		}
		var q query.FieldableQuery
		if phrase {
			q = bleve.NewMatchPhraseQuery(term)
		} else {
			mq := bleve.NewMatchQuery(term)
			mq.SetOperator(query.MatchQueryOperatorAnd)
			q = mq
		}
		if field != "" {
			q.SetField(field)
		}
		if neg {
			mustNot = append(mustNot, q)
		} else {
			must = append(must, q)
		}
	}
	if len(must) == 0 {
		return nil
	}
	bq := bleve.NewBooleanQuery()
	bq.AddMust(must...)
	bq.AddMustNot(mustNot...)
	return bq
}

// analyzes returns whether the given text has any words that are searched
// in the text of messages, unlike "the".
func (si *searchIndex) analyzes(text string) bool {
	an := si.idx.Mapping().AnalyzerNamed(en.AnalyzerName)
	return an == nil || len(an.Analyze([]byte(text))) > 0
}

// searchTerms returns the terms of the given search, separated by spaces
// that are not in quotes.
func searchTerms(search string) []string {
	var terms []string
	var b strings.Builder
	inQuote := false
	for _, r := range search {
		if r == '"' {
			inQuote = !inQuote
		}
		if r == ' ' && !inQuote {
			if b.Len() > 0 {
				terms = append(terms, b.String())
				b.Reset()
			}
			continue
		}
		b.WriteRune(r)
	}
	if b.Len() > 0 {
		terms = append(terms, b.String())
	}
	return terms
}

// readSearchDoc returns the [searchDoc] of the message in the given file.
func readSearchDoc(filename string) (*searchDoc, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	mr, err := mail.CreateReader(f)
	if err != nil {
		return nil, err
	}
	doc := &searchDoc{}
	addrs := func(key string) string {
		as, _ := mr.Header.AddressList(key)
		var res []string
		for _, a := range as {
			res = append(res, strings.TrimSpace(a.Name+" "+a.Address))
		}
		return strings.Join(res, ", ")
	}
	doc.From, doc.To, doc.Cc = addrs("From"), addrs("To"), addrs("Cc")
	doc.Subject, _ = mr.Header.Subject()
	doc.Date, _ = mr.Header.Date()
	var plain, htmlText string
	var atts []string
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			return doc, nil // what was read
		}
		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			ct, _, _ := h.ContentType()
			b, err := io.ReadAll(io.LimitReader(p.Body, 1<<20))
			if err != nil {
				continue
			}
			switch ct {
			case "text/plain":
				plain += string(b) + "\n"
			case "text/html":
				htmlText += htmlToText(string(b)) + "\n"
			}
		case *mail.AttachmentHeader:
			if name, err := h.Filename(); err == nil && name != "" {
				atts = append(atts, name)
			}
		}
	}
	doc.Body = plain
	if strings.TrimSpace(doc.Body) == "" {
		doc.Body = htmlText
	}
	doc.Attachments = strings.Join(atts, " ")
	return doc, nil
}

// htmlToText returns the text of the given HTML, without its tags, scripts,
// and styles, for searching.
func htmlToText(s string) string {
	z := html.NewTokenizer(strings.NewReader(s))
	var b strings.Builder
	skip := 0
	for {
		switch z.Next() {
		case html.ErrorToken:
			return b.String()
		case html.StartTagToken:
			if name, _ := z.TagName(); string(name) == "script" || string(name) == "style" {
				skip++
			}
		case html.EndTagToken:
			if name, _ := z.TagName(); (string(name) == "script" || string(name) == "style") && skip > 0 {
				skip--
			}
			b.WriteByte(' ')
		case html.TextToken:
			if skip == 0 {
				b.Write(z.Text())
			}
		}
	}
}
