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
// widgets to the given [gi.Widget], using the given context.
func ReadHTML(ctx Context, par gi.Widget, r io.Reader) error {
	n, err := html.Parse(r)
	if err != nil {
		return fmt.Errorf("error parsing HTML: %w", err)
	}
	return ReadHTMLNode(ctx, par, n)
}

// ReadHTMLString reads HTML from the given string and adds corresponding GoGi
// widgets to the given [gi.Widget], using the given context.
func ReadHTMLString(ctx Context, par gi.Widget, s string) error {
	b := bytes.NewBufferString(s)
	return ReadHTML(ctx, par, b)
}

// ReadHTMLNode reads HTML from the given [*html.Node] and adds corresponding GoGi
// widgets to the given [gi.Widget], using the given context.
func ReadHTMLNode(ctx Context, par gi.Widget, n *html.Node) error {
	newPar := par
	handleChildren := true
	switch n.Type {
	case html.TextNode:
		str := strings.TrimSpace(n.Data)
		if str != "" {
			newPar = ConfigWidget(ctx, gi.NewLabel(par).SetText(str), n)
		}
		handleChildren = false
	case html.ElementNode:
		ctx.SetNode(n)
		ctx.SetParent(par)
		newPar, handleChildren = HandleElement(ctx)
		if newPar != nil {
			ConfigWidget(ctx, newPar, n)
		}
	}

	if handleChildren && newPar != nil && n.FirstChild != nil {
		ReadHTMLNode(ctx, newPar, n.FirstChild)
	}
	if n.NextSibling != nil {
		ReadHTMLNode(ctx, par, n.NextSibling)
	}

	// nil parent means we are root, so we apply style here
	if n.Parent == nil {
		return ApplyStyle(ctx, par, n)
	}
	return nil
}
