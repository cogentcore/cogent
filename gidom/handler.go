// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidom

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/girl/styles"
	"goki.dev/grows/images"
	"goki.dev/grr"
	"goki.dev/icons"
	"goki.dev/ki/v2"
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
	case "head", "script", "style":
		// we don't render anything in heads, scripts, and styles
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
		// if we are already in a treeview, we just return in the last item in it
		// (which is the list item we are contained in), which fixes the associativity
		// of nested list items and prevents the created of duplicated tree view items.
		if ptv, ok := par.(*giv.TreeView); ok {
			w := ki.LastChild(ptv).(gi.Widget)
			// we also set its class so that the orderedness of nested items works properly
			w.AsWidget().SetClass(typ)
			return w
		}
		tv := giv.NewTreeView(par).SetText("").SetIcon(icons.None).SetClass(typ)
		tv.RootView = tv
		return tv
	case "li":
		ntv := giv.NewTreeView(par)
		ftxt := ""
		ptv, ok := par.(*giv.TreeView)
		if ok {
			ntv.RootView = ptv.RootView
			if ptv.HasClass("ol") {
				ip, _ := ntv.IndexInParent()
				ftxt = strconv.Itoa(ip+1) + ". " // start at 1
			} else {
				// TODO(kai/gidom): have different bullets for different depths
				ftxt = "• "
			}
		} else {
			ntv.RootView = ntv
		}

		etxt := ExtractText(par, n)
		ntv.SetName(etxt)
		ntv.SetText(ftxt + etxt)
	case "img":
		img := gi.NewImage(par)
		src := GetAttr(n, "src")
		u := grr.Log(ParseRelativeURL(src, ""))
		resp, err := http.Get(u.String())
		if grr.Log0(err) != nil {
			return par
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			slog.Error("got error status code", "status", resp.Status)
		}
		im, _, err := images.Read(resp.Body)
		if grr.Log0(err) != nil {
			return par
		}
		img.Filename = gi.FileName(src)
		img.SetImage(im, 0, 0)
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
