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
// widgets to the given [gi.Widget].
func ReadHTML(par gi.Widget, r io.Reader) error {
	n, err := html.Parse(r)
	if err != nil {
		return fmt.Errorf("error parsing HTML: %w", err)
	}
	return ReadHTMLNode(par, n)
}

// ReadHTMLString reads HTML from the given string and adds corresponding GoGi
// widgets to the given [gi.Widget].
func ReadHTMLString(par gi.Widget, s string) error {
	b := bytes.NewBufferString(s)
	return ReadHTML(par, b)
}

// ReadHTMLNode reads HTML from the given [*html.Node] and adds corresponding GoGi
// widgets to the given [gi.Widget].
func ReadHTMLNode(par gi.Widget, n *html.Node) error {
	newPar := par
	switch n.Type {
	case html.TextNode:
		str := strings.TrimSpace(n.Data)
		if str != "" {
			gi.NewLabel(par).SetText(str)
		}
		newPar = nil
	case html.ElementNode:
		newPar = HandleElement(par, n)
	}

	if newPar != nil && n.FirstChild != nil {
		ReadHTMLNode(newPar, n.FirstChild)
	}
	if n.NextSibling != nil {
		ReadHTMLNode(par, n.NextSibling)
	}
	return nil
}
