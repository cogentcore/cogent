// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidom

import (
	"slices"

	"goki.dev/gi/v2/gi"
	"golang.org/x/net/html"
)

// ExtractText recursively extracts all of the text from the children
// of the given [*html.Node], adding any appropriate inline markup for
// formatted text. It adds any non-text elements to the given [gi.Widget]
// using [ReadHTMLNode]. It should not be called on text nodes themselves;
// for that, you can directly access the [html.Node.Data] field. It uses
// the given page URL for context when resolving URLs, but it can be
// omitted if not available.
func ExtractText(par gi.Widget, n *html.Node, pageURL string) string {
	if n.FirstChild == nil {
		return ""
	}
	return extractTextImpl(par, n.FirstChild, pageURL)
}

func extractTextImpl(par gi.Widget, n *html.Node, pageURL string) string {
	str := ""
	if n.Type == html.TextNode {
		str += n.Data
	}
	it := IsText(n)
	if !it {
		ReadHTMLNode(par, n, pageURL)
	}
	if it && n.FirstChild != nil {
		start, end := NodeString(n)
		str = start + extractTextImpl(par, n.FirstChild, pageURL) + end
	}
	if n.NextSibling != nil {
		str += extractTextImpl(par, n.NextSibling, pageURL)
	}
	return str
}

// NodeString returns the given node as starting and ending strings in the format:
//
//	<tag attr0="value0" attr1="value1">
//
// and
//
//	</tag>
//
// It returns "", "" if the given node is not an [html.ElementNode]
func NodeString(n *html.Node) (start, end string) {
	if n.Type != html.ElementNode {
		return
	}
	tag := n.DataAtom.String()
	start = "<" + tag
	for _, a := range n.Attr {
		start += " " + a.Key + "=" + `"` + a.Val + `"`
	}
	start += ">"
	end = "</" + tag + ">"
	return
}

// TextTags are all of the node tags that result in a true return value for [IsText].
var TextTags = []string{
	"a", "abbr", "b", "bdi", "bdo", "br", "cite", "code", "data", "dfn",
	"em", "i", "kbd", "mark", "q", "rp", "rt", "ruby", "s", "samp", "small",
	"span", "strong", "sub", "sup", "time", "u", "var", "wbr",
}

// IsText returns true if the given node is a [html.TextNode] or
// an [html.ElementNode] designed for holding text (a, span, b, code, etc),
// and false otherwise.
func IsText(n *html.Node) bool {
	if n.Type == html.TextNode {
		return true
	}
	if n.Type == html.ElementNode {
		tag := n.DataAtom.String()
		return slices.Contains(TextTags, tag)
	}
	return false
}
