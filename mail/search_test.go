// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchIndex(t *testing.T) {
	dir := t.TempDir()
	write := func(name, text string) string {
		fn := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(fn, []byte(text), 0o600))
		return fn
	}
	// the first message of the tests, with HTML and an attachment
	d1, err := readSearchDoc(write("1", testMessages[0].text))
	require.NoError(t, err)
	assert.Equal(t, "Alice alice@example.com", d1.From)
	assert.Equal(t, "Hello", d1.Subject)
	assert.Contains(t, d1.Body, "how are you")
	assert.Equal(t, "notes.txt", d1.Attachments)
	d2, err := readSearchDoc(write("2", "From: Bob <bob@example.com>\r\nSubject: Budget\r\n"+
		"Content-Type: text/html\r\n\r\n<html><style>p {}</style><p>The <b>quarterly</b> numbers are running late</p></html>\r\n"))
	require.NoError(t, err)
	assert.Contains(t, d2.Body, "quarterly")
	assert.NotContains(t, d2.Body, "p {}", "not the style")

	idx, err := openSearch(filepath.Join(dir, "search", "index.bleve"))
	require.NoError(t, err)
	require.NoError(t, idx.add(map[string]*searchDoc{"1": d1, "2": d2}))
	search := func(q string) []string {
		t.Helper()
		keys, err := idx.search(q, 10)
		require.NoError(t, err)
		return keys
	}
	assert.Equal(t, []string{"1"}, search("hi how are you"))
	assert.Equal(t, []string{"2"}, search("number run"), "stemmed")
	assert.Equal(t, []string{"2"}, search("from:bob"))
	assert.Equal(t, []string{"2"}, search("FROM:Bob budget"))
	assert.Empty(t, search("from:bob -budget"), "not")
	assert.Equal(t, []string{"2"}, search("subject:budget"))
	assert.Empty(t, search("subject:numbers"), "only in the body")
	assert.Empty(t, search("the"), "nothing to search for")
	assert.Equal(t, []string{"1"}, search("notes.txt"))
	assert.Empty(t, search("hello quarterly"), "all of the words")
	assert.Equal(t, []string{"2"}, search(`"running late"`))
	assert.Empty(t, search(`"late running"`), "a phrase")
	assert.True(t, idx.has("1"))
	require.NoError(t, idx.remove("1"))
	assert.False(t, idx.has("1"))
	assert.Empty(t, search("hi how are you"))
	require.NoError(t, idx.Close())

	idx, err = openSearch(filepath.Join(dir, "search", "index.bleve"))
	require.NoError(t, err)
	defer idx.Close()
	assert.Equal(t, []string{"2"}, search("budget"), "reopened")
}

func TestSearchTerms(t *testing.T) {
	assert.Equal(t, []string{"a", `"b c"`, "-d", `from:"e f"`}, searchTerms(` a  "b c" -d from:"e f"`))
}

// TestCharsets tests a message in windows-1252, which is not one of the
// character sets of the standard library, in its header and its body.
func TestCharsets(t *testing.T) {
	fn := filepath.Join(t.TempDir(), "msg.eml")
	msg := "From: =?windows-1252?Q?Ren=E9e?= <renee@example.com>\r\n" +
		"Subject: =?windows-1252?Q?Caf=E9_=93menu=94?=\r\n" +
		"Content-Type: text/plain; charset=windows-1252\r\n" +
		"Content-Transfer-Encoding: 8bit\r\n\r\n" +
		"The \x93caf\xe9\x94 is open \x96 come by.\r\n"
	require.NoError(t, os.WriteFile(fn, []byte(msg), 0o600))
	doc, err := readSearchDoc(fn)
	require.NoError(t, err)
	assert.Equal(t, "Café “menu”", doc.Subject)
	assert.Equal(t, "Renée renee@example.com", doc.From)
	assert.Contains(t, doc.Body, "The “café” is open – come by.")

	addr, err := imapOptions.WordDecoder.DecodeHeader("=?windows-1252?Q?Ren=E9e?=")
	require.NoError(t, err)
	assert.Equal(t, "Renée", addr, "in the envelopes from the server")

	// an unknown 8-bit charset: UTF-8, and other bytes as Windows-1252
	msg = "From: a@example.com\r\nSubject: Unknown\r\n" +
		"Content-Type: text/plain; charset=unknown-8bit\r\n" +
		"Content-Transfer-Encoding: 8bit\r\n\r\n" +
		"Na\xefve caf\xc3\xa9 \x93ok\x94\r\n"
	require.NoError(t, os.WriteFile(fn, []byte(msg), 0o600))
	doc, err = readSearchDoc(fn)
	require.NoError(t, err)
	assert.Contains(t, doc.Body, "Naïve café “ok”")
}
