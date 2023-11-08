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
func ReadHTML(par ki.Ki, r io.Reader) error {
	n, err := html.Parse(r)
	if err != nil {
		return fmt.Errorf("error parsing HTML: %w", err)
	}
	return ReadHTMLNode(par, n)
}

// ReadHTMLString reads HTML from the given string and adds corresponding GoGi
// widgets to the given [ki.Ki].
func ReadHTMLString(par ki.Ki, s string) error {
	b := bytes.NewBufferString(s)
	return ReadHTML(par, b)
}

// ReadHTMLNode reads HTML from the given [*html.Node] and adds corresponding GoGi
// widgets to the given [ki.Ki].
func ReadHTMLNode(par ki.Ki, n *html.Node) error {
	var newPar gi.Widget
	switch n.Type {
	case html.TextNode:
		gi.NewLabel(par).SetText(n.Data)
	case html.ElementNode:
		typ := n.DataAtom.String()
		switch typ {
		case "button":
			gi.NewButton(par).SetText(ExtractText(n))
		case "h1":
			gi.NewLabel(par).SetType(gi.LabelHeadlineLarge).SetText(ExtractText(n))
		case "p":
			gi.NewLabel(par).SetType(gi.LabelBodyLarge).SetText(ExtractText(n))
		}
	}

	if newPar != nil && n.FirstChild != nil {
		ReadHTMLNode(newPar, n.FirstChild)
	}
	if n.NextSibling != nil {
		ReadHTMLNode(par, n.NextSibling)
	}
	return nil
}
