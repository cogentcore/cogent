// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gidom converts HTML and MD into GoGi DOM widget trees.
package gidom

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"goki.dev/gi/v2/gi"
	"golang.org/x/net/html"
)

// ReadHTML reads HTML from the given [io.Reader] and adds corresponding GoGi
// widgets to the given [gi.Widget]. It uses the given page URL for context
// when resolving URLs, but it can be omitted if not available.
func ReadHTML(par gi.Widget, r io.Reader, pageURL string) error {
	n, err := html.Parse(r)
	if err != nil {
		return fmt.Errorf("error parsing HTML: %w", err)
	}
	return ReadHTMLNode(par, n, pageURL)
}

// ReadHTMLString reads HTML from the given string and adds corresponding GoGi
// widgets to the given [gi.Widget]. It uses the given page URL for context
// when resolving URLs, but it can be omitted if not available.
func ReadHTMLString(par gi.Widget, s string, pageURL string) error {
	b := bytes.NewBufferString(s)
	return ReadHTML(par, b, pageURL)
}

// ReadHTMLNode reads HTML from the given [*html.Node] and adds corresponding GoGi
// widgets to the given [gi.Widget]. It uses the given page URL for context
// when resolving URLs, but it can be omitted if not available.
func ReadHTMLNode(par gi.Widget, n *html.Node, pageURL string) error {
	newPar := par
	handleChildren := true
	switch n.Type {
	case html.TextNode:
		str := strings.TrimSpace(n.Data)
		if str != "" {
			newPar = ConfigWidget(gi.NewLabel(par).SetText(str), n)
		}
		handleChildren = false
	case html.ElementNode:
		newPar, handleChildren = HandleElement(par, n, pageURL)
		if newPar != nil {
			ConfigWidget(newPar, n)
		}
	}

	if handleChildren && newPar != nil && n.FirstChild != nil {
		ReadHTMLNode(newPar, n.FirstChild, pageURL)
	}
	if n.NextSibling != nil {
		ReadHTMLNode(par, n.NextSibling, pageURL)
	}
	return nil
}
