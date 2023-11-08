// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidom

import (
	"strings"

	"golang.org/x/net/html"
)

// ExtractText recursively extracts all of the text from the children
// of the given [*html.Node], adding any appropriate inline markup for
// formatted text. It should not be called on text nodes themselves;
// for that, you can directly access the [html.Node.Data] field.
func ExtractText(n *html.Node) string {
	if n.FirstChild == nil {
		return ""
	}
	return extractTextImpl(n.FirstChild)
}

func extractTextImpl(n *html.Node) string {
	str := ""
	if n.Type == html.TextNode {
		str += strings.TrimSpace(n.Data)
	}
	if n.FirstChild != nil {
		str += extractTextImpl(n.FirstChild)
	}
	if n.NextSibling != nil {
		str += " " + extractTextImpl(n.NextSibling)
	}
	return str
}
