// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidom

import (
	"time"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/girl/styles"
	"goki.dev/grr"
	"golang.org/x/net/html"
)

// Handler is a function that can be used to describe the behavior
// of gidom parsing for a specific type of element. It takes the
// parent [gi.Widget] to add widgets to and the [*html.Node] to read from.
// If it returns nil, that indicates that the children of this node have
// already been handled (or will not be handled), and thus should not
// be handled further. If it returns a non-nil widget, then the children
// will be handled, with the returned widget as their parent.
type Handler func(par gi.Widget, n *html.Node) gi.Widget

// ElementHandlers is a map of [Handler] functions for each HTML element
// type (eg: "button", "input", "p"). It is initialized to contain appropriate
// handlers for all of the standard HTML elements, but can be extended or
// modified for anyone in need of different behavior.
var ElementHandlers = map[string]Handler{}

// HandleELement calls the [Handler] associated with the given element [*html.Node]
// in [ElementHandlers] and returns the result.
func HandleElement(par gi.Widget, n *html.Node) gi.Widget {
	typ := n.DataAtom.String()
	h, ok := ElementHandlers[typ]
	if ok {
		return h(par, n)
	}
	switch typ {
	case "head":
		// we don't render anything in the head
	case "button":
		gi.NewButton(par).SetText(ExtractText(par, n))
	case "h1":
		gi.NewLabel(par).SetType(gi.LabelHeadlineLarge).SetText(ExtractText(par, n))
	case "h2":
		gi.NewLabel(par).SetType(gi.LabelHeadlineSmall).SetText(ExtractText(par, n))
	case "h3":
		gi.NewLabel(par).SetType(gi.LabelTitleLarge).SetText(ExtractText(par, n))
	case "h4":
		gi.NewLabel(par).SetType(gi.LabelTitleMedium).SetText(ExtractText(par, n))
	case "h5":
		gi.NewLabel(par).SetType(gi.LabelTitleSmall).SetText(ExtractText(par, n))
	case "h6":
		gi.NewLabel(par).SetType(gi.LabelLabelSmall).SetText(ExtractText(par, n))
	case "p":
		gi.NewLabel(par).SetText(ExtractText(par, n))
	case "pre":
		gi.NewLabel(par).SetText(ExtractText(par, n)).Style(func(s *styles.Style) {
			s.Text.WhiteSpace = styles.WhiteSpacePre
		})
	case "ol", "ul":
		tv := giv.NewTreeView(par, "list")
		tv.RootView = tv
		return tv
	case "li":
		ntv := giv.NewTreeView(par, ExtractText(par, n))
		ptv := par.(*giv.TreeView)
		ntv.RootView = ptv.RootView
	case "img":
		src := gi.FileName(GetAttr(n, "src"))
		grr.Log0(gi.NewImage(par).OpenImage(src, 0, 0))
	case "input":
		ityp := GetAttr(n, "type")
		switch ityp {
		case "number":
			gi.NewSpinner(par)
		case "color":
			giv.NewValue(par, colors.Black)
		case "datetime":
			giv.NewValue(par, time.Now())
		default:
			gi.NewTextField(par)
		}
	case "textarea":
		buf := texteditor.NewBuf()
		buf.SetText([]byte(ExtractText(par, n)))
		texteditor.NewEditor(par).SetBuf(buf)
	default:
		return par
	}
	return nil
}

// GetAttr gets the given attribute from the given node, returning ""
// if the attribute is not found.
func GetAttr(n *html.Node, attr string) string {
	res := ""
	for _, a := range n.Attr {
		if a.Key == attr {
			res = a.Val
		}
	}
	return res
}
