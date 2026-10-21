// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"strings"
)

// DocLineKinds are the kinds of [DocLine]s in documentation.
type DocLineKinds int32

const (
	// DocText is a line of text, with inline code and links.
	DocText DocLineKinds = iota

	// DocCode is a line of a code block, in the [DocLine.Lang] language.
	DocCode

	// DocHeading is a heading line.
	DocHeading

	// DocRule is a horizontal rule separating sections.
	DocRule
)

// DocLine is one line of documentation, parsed from the markdown
// that servers provide for hover docs, completion item documentation,
// and signature help, by [ParseMarkdown].
type DocLine struct {

	// Kind is the kind of line.
	Kind DocLineKinds

	// Lang is the language of a code block line (e.g., "go"),
	// as given after the opening fence.
	Lang string

	// Spans are the spans of the line. A code block line has a single
	// span with its raw text.
	Spans []DocSpan
}

// DocSpan is one span of a [DocLine].
type DocSpan struct {

	// Text is the text of the span.
	Text string

	// Code indicates inline code.
	Code bool

	// URL is the link target, if this is a link.
	URL string
}

// String returns the plain text of the line.
func (dl *DocLine) String() string {
	var sb strings.Builder
	for _, sp := range dl.Spans {
		sb.WriteString(sp.Text)
	}
	return sb.String()
}

// ParseMarkdown parses the subset of markdown that language servers
// use for documentation: code blocks, headings, rules, inline code,
// links, and backslash escapes. Runs of blank lines are collapsed, and
// blank lines around rules and at the start and end are removed.
func ParseMarkdown(md string) []DocLine {
	var out []DocLine
	blank := false // pending blank line
	lastKind := func() DocLineKinds {
		if len(out) == 0 {
			return DocRule // nothing before: same as after a rule
		}
		return out[len(out)-1].Kind
	}
	add := func(dl DocLine) {
		switch {
		case dl.Kind == DocText && len(dl.Spans) == 0:
			blank = lastKind() != DocRule
			return
		case dl.Kind == DocRule:
			blank = false
			if lastKind() == DocRule {
				return
			}
		}
		if blank {
			out = append(out, DocLine{})
			blank = false
		}
		out = append(out, dl)
	}
	inCode, lang := false, ""
	for _, ln := range strings.Split(md, "\n") {
		ln = strings.TrimRight(ln, " \t\r")
		tln := strings.TrimSpace(ln)
		switch {
		case strings.HasPrefix(tln, "```"):
			inCode = !inCode
			lang = strings.TrimSpace(strings.TrimPrefix(tln, "```"))
		case inCode:
			if blank {
				out = append(out, DocLine{})
				blank = false
			}
			out = append(out, DocLine{Kind: DocCode, Lang: lang, Spans: []DocSpan{{Text: ln}}})
		case tln == "---" || tln == "***" || tln == "___":
			add(DocLine{Kind: DocRule})
		case strings.HasPrefix(tln, "#"):
			add(DocLine{Kind: DocHeading, Spans: parseInline(strings.TrimLeft(tln, "# "))})
		default:
			add(DocLine{Kind: DocText, Spans: parseInline(ln)})
		}
	}
	if len(out) > 0 && out[len(out)-1].Kind == DocRule {
		out = out[:len(out)-1]
	}
	return out
}

// parseInline parses the inline code, links, and escapes of a line.
func parseInline(s string) []DocSpan {
	var spans []DocSpan
	var sb strings.Builder
	flush := func() {
		if sb.Len() > 0 {
			spans = append(spans, DocSpan{Text: sb.String()})
			sb.Reset()
		}
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\' && i+1 < len(s) && strings.IndexByte("\\`*_{}[]()#+-.!<>|", s[i+1]) >= 0:
			i++
			sb.WriteByte(s[i])
		case c == '`':
			if j := strings.IndexByte(s[i+1:], '`'); j >= 0 {
				flush()
				spans = append(spans, DocSpan{Text: s[i+1 : i+1+j], Code: true})
				i += j + 1
				continue
			}
			sb.WriteByte(c)
		case c == '[':
			txt, url, n := parseLink(s[i:])
			if n == 0 {
				sb.WriteByte(c)
				continue
			}
			flush()
			spans = append(spans, DocSpan{Text: txt, URL: url})
			i += n - 1
		default:
			sb.WriteByte(c)
		}
	}
	flush()
	return spans
}

// parseLink parses a [text](url) link at the start of s, returning the
// text (without inline code backticks), the url, and the length of the
// link in s, which is 0 if there is no link.
func parseLink(s string) (text, url string, n int) {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth > 0 {
				continue
			}
			if i+1 >= len(s) || s[i+1] != '(' {
				return "", "", 0
			}
			j := strings.IndexByte(s[i+2:], ')')
			if j < 0 {
				return "", "", 0
			}
			return strings.ReplaceAll(s[1:i], "`", ""), s[i+2 : i+2+j], i + 3 + j
		}
	}
	return "", "", 0
}

// MarkdownText returns the plain text of the given documentation
// markdown, for display in tooltips etc: without code fences,
// backticks, or link targets, with rules as blank lines, and without
// lines that are only a link, which cannot be followed in plain text.
func MarkdownText(md string) string {
	var lns []string
	for _, dl := range ParseMarkdown(md) {
		if len(dl.Spans) == 1 && dl.Spans[0].URL != "" {
			continue
		}
		if ln := dl.String(); ln != "" || (len(lns) > 0 && lns[len(lns)-1] != "") {
			lns = append(lns, ln)
		}
	}
	return strings.TrimSpace(strings.Join(lns, "\n"))
}
