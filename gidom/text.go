// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidom

import (
	"strings"

	"golang.org/x/net/html"
)

// ExtractText recursively extracts all of the text from the given [*html.Node],
// adding any appropriate inline markup for formatted text.
func ExtractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return strings.TrimSpace(n.Data)
	}
	if n.FirstChild != nil {
		return ExtractText(n.FirstChild)
	}
	return ""
}
