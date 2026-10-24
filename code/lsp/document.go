// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// IsLocalURL reports whether the given URL is a web page served on the
// local machine, such as the pages gopls serves for its "Browse" code
// actions (documentation, assembly, free symbols), as opposed to an
// external web site.
func IsLocalURL(uri string) bool {
	u, err := url.Parse(uri)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	switch u.Hostname() {
	case "127.0.0.1", "localhost", "::1":
		return true
	}
	return false
}

// DocumentText fetches the web page at the given URL, such as one that
// a language server asks to show (see [IsLocalURL]), returning its title
// and its contents converted to plain text, for display in a text editor.
func DocumentText(uri string) (title, text string, err error) {
	client := http.Client{Timeout: defaultCallTimeout}
	resp, err := client.Get(uri)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("lsp: %s: %s", uri, resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "html") {
		return "", string(b), nil
	}
	title, text = htmlText(string(b))
	return title, text, nil
}

// htmlBlocks are the HTML elements that start on a new line in text.
var htmlBlocks = map[string]bool{
	"address": true, "article": true, "aside": true, "blockquote": true,
	"br": true, "dd": true, "details": true, "div": true, "dl": true,
	"dt": true, "fieldset": true, "figure": true, "footer": true,
	"form": true, "h1": true, "h2": true, "h3": true, "h4": true,
	"h5": true, "h6": true, "header": true, "hr": true, "li": true,
	"main": true, "nav": true, "ol": true, "p": true, "pre": true,
	"section": true, "summary": true, "table": true, "tr": true, "ul": true,
}

// htmlSkips are the HTML elements whose contents are not text.
var htmlSkips = map[string]bool{"script": true, "style": true, "head": true, "noscript": true}

// blankLines matches runs of blank lines, to collapse them.
var blankLines = regexp.MustCompile(`\n[ \t]*(\n[ \t]*)+\n`)

// trailingSpaces matches the spaces at the ends of lines.
var trailingSpaces = regexp.MustCompile(`(?m) +$`)

// spaces matches runs of whitespace, to collapse them.
var spaces = regexp.MustCompile(`\s+`)

// htmlText returns the title and the plain text contents of the given
// HTML source, keeping the whitespace of preformatted text.
func htmlText(src string) (title, text string) {
	var sb strings.Builder
	last := byte('\n') // last byte written, treating the start as a new line
	write := func(t string) {
		if t != "" {
			sb.WriteString(t)
			last = t[len(t)-1]
		}
	}
	newline := func() {
		if last != '\n' {
			write("\n")
		}
	}
	z := html.NewTokenizer(strings.NewReader(src))
	skip, pre, inTitle := 0, 0, false
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			text = trailingSpaces.ReplaceAllString(sb.String(), "")
			text = blankLines.ReplaceAllString(text, "\n\n")
			return strings.TrimSpace(title), strings.TrimSpace(text) + "\n"
		case html.StartTagToken, html.EndTagToken, html.SelfClosingTagToken:
			nm, _ := z.TagName()
			tag := string(nm)
			start := tt != html.EndTagToken
			switch {
			case tag == "title":
				inTitle = start
			case htmlSkips[tag]:
				if tt == html.StartTagToken {
					skip++
				} else if tt == html.EndTagToken && skip > 0 {
					skip--
				}
			case tag == "pre":
				if tt == html.StartTagToken {
					pre++
				} else if tt == html.EndTagToken && pre > 0 {
					pre--
				}
				newline()
			case tag == "td" || tag == "th":
				if start && last != '\n' {
					write("\t")
				}
			case htmlBlocks[tag]:
				newline()
			}
		case html.TextToken:
			t := html.UnescapeString(string(z.Text()))
			switch {
			case inTitle:
				title += t
			case skip > 0:
			case pre > 0:
				write(t)
			default:
				t = spaces.ReplaceAllString(t, " ")
				if last == '\n' || last == '\t' || last == ' ' {
					t = strings.TrimLeft(t, " ")
				}
				write(t)
			}
		}
	}
}
