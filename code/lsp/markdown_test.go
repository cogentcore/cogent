// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// hoverField is gopls hover output for a struct field.
const hoverField = "```go\nfield FPath string `edit:\"-\" display:\"-\" table:\"-\"`\n```\n\n---\n\nfull path to file\n\n\n---\n\n[`(cdebug.Break).FPath` on pkg.go.dev](https://pkg.go.dev/cogentcore.org/cogent/code/cdebug#Break.FPath)"

func TestParseMarkdown(t *testing.T) {
	dls := ParseMarkdown(hoverField)
	kinds := make([]DocLineKinds, len(dls))
	for i := range dls {
		kinds[i] = dls[i].Kind
	}
	assert.Equal(t, []DocLineKinds{DocCode, DocRule, DocText, DocRule, DocText}, kinds)
	assert.Equal(t, "go", dls[0].Lang)
	assert.Equal(t, "field FPath string `edit:\"-\" display:\"-\" table:\"-\"`", dls[0].String())
	assert.Equal(t, []DocSpan{{Text: "(cdebug.Break).FPath on pkg.go.dev", URL: "https://pkg.go.dev/cogentcore.org/cogent/code/cdebug#Break.FPath"}}, dls[4].Spans)

	assert.Equal(t, []DocSpan{{Text: "Calls "}, {Text: "f(x)", Code: true}, {Text: " with a * and "}, {Text: "link", URL: "u"}, {Text: "."}},
		parseInline("Calls `f(x)` with a \\* and [link](u)."))

	assert.Equal(t, "field FPath string `edit:\"-\" display:\"-\" table:\"-\"`\n\nfull path to file", MarkdownText(hoverField))
}

func TestSortCodeActions(t *testing.T) {
	acts := []CodeAction{
		{Title: "features", Kind: "gopls.doc.features"},
		{Title: "assembly", Kind: "source.assembly"},
		{Title: "inline", Kind: "refactor.inline.call"},
		{Title: "disabled", Kind: "quickfix", Disabled: &CodeActionDisabled{Reason: "no"}},
		{Title: "fix", Kind: "quickfix"},
		{Title: "preferred", Kind: "quickfix", IsPreferred: true},
		{Title: "other", Kind: "other"},
	}
	SortCodeActions(acts)
	var titles []string
	for _, ca := range acts {
		titles = append(titles, ca.Title)
	}
	assert.Equal(t, []string{"preferred", "fix", "inline", "assembly", "other", "features", "disabled"}, titles)
}
