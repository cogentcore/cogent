// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gidom converts HTML and MD into GoGi DOM widget trees.
package gidom

import (
	"bytes"
	"fmt"
	"io"

	"goki.dev/gi/v2/gi"
	"goki.dev/ki/v2"
	"golang.org/x/net/html"
)

// ReadHTML reads HTML from the given [io.Reader] and adds corresponding GoGi
// widgets to the given [ki.Ki].
func ReadHTML(k ki.Ki, r io.Reader) error {
	n, err := html.Parse(r)
	if err != nil {
		return fmt.Errorf("error parsing HTML: %w", err)
	}
	return ReadHTMLNode(k, n)
}

// ReadHTMLString reads HTML from the given string and adds corresponding GoGi
// widgets to the given [ki.Ki].
func ReadHTMLString(k ki.Ki, s string) error {
	b := bytes.NewBufferString(s)
	return ReadHTML(k, b)
}

// ReadHTMLNode reads HTML from the given [*html.Node] and adds corresponding GoGi
// widgets to the given [ki.Ki].
func ReadHTMLNode(k ki.Ki, n *html.Node) error {
	par := k
	switch n.Type {
	case html.TextNode:
		par = gi.NewLabel(k).SetText(n.Data)
	case html.ElementNode:
		typ := n.DataAtom.String()
		switch typ {
		case "button":
			bt := gi.NewButton(k)
			if n.FirstChild != nil {
				bt.SetText(n.FirstChild.Data)
				n.FirstChild = nil
			}
		}
	}

	if n.FirstChild != nil {
		ReadHTMLNode(par, n.FirstChild)
	}
	if n.NextSibling != nil {
		ReadHTMLNode(par, n.NextSibling)
	}
	return nil
}
