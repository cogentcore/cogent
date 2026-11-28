// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/lang/en"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
)

// searchIndex is the full-text search index of the photos of a library:
// a Bleve index of their [searchDoc]s, by their names, as in Cogent Mail.
type searchIndex struct {
	idx bleve.Index
}

// searchDoc is the text of a photo that is searched.
type searchDoc struct {

	// Tags are the tags of the photo, with their words split (as
	// AlpineLEGO is also Alpine LEGO), and without their years.
	Tags string

	// Desc is the description of the photo.
	Desc string

	// When is the year, month, and weekday the photo was taken, in words.
	When string

	// Camera is the make and model of the camera.
	Camera string

	// Kind is video for videos, and photo for photos.
	Kind string

	// Name is the name of the file of the photo, and its name before
	// it was renamed.
	Name string

	// Date is when the photo was taken.
	Date time.Time
}

// newSearchDoc returns the [searchDoc] of the given photo.
func newSearchDoc(ph *Photo) *searchDoc {
	doc := &searchDoc{Desc: ph.Desc, Date: ph.DateTaken}
	var tags []string
	for _, t := range ph.Tags { // the groups and the name
		for _, p := range strings.Split(t, tagSep) {
			tags = append(tags, p, splitWords(p))
		}
	}
	doc.Tags = strings.Join(tags, " ")
	if !ph.DateTaken.IsZero() {
		doc.When = ph.DateTaken.Format("2006 January Jan Monday")
	}
	doc.Camera = strings.TrimSpace(ph.Make + " " + ph.Model)
	doc.Kind = "photo"
	if ph.IsVideo() {
		doc.Kind = "video"
	}
	doc.Name = ph.Name
	if ph.OrigName != "" && ph.OrigName != ph.Name {
		doc.Name += " " + ph.OrigName
	}
	return doc
}

// splitWords returns the given name with its words split by spaces, at
// changes from lower to upper case, letters to digits, and _ and -, as
// AlpineVRBOposs is Alpine VRBOposs.
func splitWords(name string) string {
	var b strings.Builder
	var prev rune
	for i, r := range name {
		if r == '_' || r == '-' {
			b.WriteByte(' ')
			prev = ' '
			continue
		}
		if i > 0 && ((unicode.IsUpper(r) && unicode.IsLower(prev)) || (unicode.IsDigit(r) != unicode.IsDigit(prev) && prev != ' ')) {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
		prev = r
	}
	return b.String()
}

// searchMapping returns the mapping of the [searchDoc]s.
func searchMapping() mapping.IndexMapping {
	text := bleve.NewTextFieldMapping()
	text.Analyzer = en.AnalyzerName
	words := bleve.NewTextFieldMapping()
	words.Analyzer = "standard"
	doc := bleve.NewDocumentMapping()
	for _, f := range []string{"Tags", "When", "Camera", "Kind", "Name"} {
		doc.AddFieldMappingsAt(f, words)
	}
	doc.AddFieldMappingsAt("Desc", text)
	doc.AddFieldMappingsAt("Date", bleve.NewDateTimeFieldMapping())
	m := bleve.NewIndexMapping()
	m.DefaultMapping = doc
	m.DefaultAnalyzer = "standard"
	return m
}

// openSearch opens the search index in the given directory,
// creating it if needed.
func openSearch(dir string) (*searchIndex, error) {
	idx, err := bleve.Open(dir)
	if errors.Is(err, bleve.ErrorIndexPathDoesNotExist) {
		if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
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

// add indexes the given photos, in batches.
func (si *searchIndex) add(phs ...*Photo) error {
	const batchSize = 1000
	for len(phs) > 0 {
		n := min(len(phs), batchSize)
		b := si.idx.NewBatch()
		for _, ph := range phs[:n] {
			if err := b.Index(ph.Name, newSearchDoc(ph)); err != nil {
				return err
			}
		}
		if err := si.idx.Batch(b); err != nil {
			return err
		}
		phs = phs[n:]
	}
	return nil
}

// remove removes the photos with the given names from the index.
func (si *searchIndex) remove(names ...string) error {
	if len(names) == 0 {
		return nil
	}
	b := si.idx.NewBatch()
	for _, n := range names {
		b.Delete(n)
	}
	return si.idx.Batch(b)
}

// count returns the number of photos in the index.
func (si *searchIndex) count() int {
	n, _ := si.idx.DocCount()
	return int(n)
}

// search returns the names of the photos that match the given search
// (see [searchIndex.query]).
func (si *searchIndex) search(search string) (map[string]bool, error) {
	q := si.query(search)
	if q == nil {
		return nil, nil
	}
	n := si.count()
	req := bleve.NewSearchRequestOptions(q, max(n, 1), 0, false)
	res, err := si.idx.Search(req)
	if err != nil {
		return nil, err
	}
	names := make(map[string]bool, len(res.Hits))
	for _, h := range res.Hits {
		names[h.ID] = true
	}
	return names, nil
}

// searchFields are the fields that can be searched on their own, as
// field:words, by their names in lower case.
var searchFields = map[string]string{"tag": "Tags", "tags": "Tags", "desc": "Desc",
	"description": "Desc", "when": "When", "year": "When", "month": "When",
	"camera": "Camera", "kind": "Kind", "name": "Name", "file": "Name"}

// query returns the query for the given search: all of its terms must
// match, where a term is a word, a "phrase", or either of them after a
// field name and a colon, as tag:kyoto, and a term with a - before it
// must not match. A word matches the start of words, so kyo matches
// Kyoto. It returns nil if there is nothing to search for.
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
		if term == "" {
			continue
		}
		var q query.Query
		if phrase {
			pq := bleve.NewMatchPhraseQuery(term)
			if field != "" {
				pq.SetField(field)
			}
			q = pq
		} else {
			var qs []query.Query
			for _, w := range strings.Fields(splitWords(term)) {
				pq := bleve.NewPrefixQuery(strings.ToLower(w))
				if field != "" {
					pq.SetField(field)
				}
				mq := bleve.NewMatchQuery(w) // stemmed, for descriptions
				if field != "" {
					mq.SetField(field)
				}
				qs = append(qs, bleve.NewDisjunctionQuery(pq, mq))
			}
			q = bleve.NewConjunctionQuery(qs...)
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
