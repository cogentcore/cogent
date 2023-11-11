// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidom

import (
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/girl/paint"
	"goki.dev/girl/styles"
	"goki.dev/goosi"
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
// in [ElementHandlers] and returns the result. It uses the given page URL for context
// when resolving URLs, but it can be omitted if not available.
func HandleElement(par gi.Widget, n *html.Node, pageURL string) gi.Widget {
	tag := n.DataAtom.String()
	h, ok := ElementHandlers[tag]
	if ok {
		return h(par, n)
	}
	if slices.Contains(TextTags, tag) {
		HandleLabelTag(par, n, pageURL)
		return nil
	}
	switch tag {
	case "head", "script", "style":
		// we don't render anything in heads, scripts, and styles
	case "button":
		HandleLabel(par, n, pageURL)
	case "h1":
		HandleLabel(par, n, pageURL).SetType(gi.LabelHeadlineLarge)
	case "h2":
		HandleLabel(par, n, pageURL).SetType(gi.LabelHeadlineSmall)
	case "h3":
		HandleLabel(par, n, pageURL).SetType(gi.LabelTitleLarge)
	case "h4":
		HandleLabel(par, n, pageURL).SetType(gi.LabelTitleMedium)
	case "h5":
		HandleLabel(par, n, pageURL).SetType(gi.LabelTitleSmall)
	case "h6":
		HandleLabel(par, n, pageURL).SetType(gi.LabelLabelSmall)
	case "p":
		HandleLabel(par, n, pageURL)
	case "pre":
		HandleLabel(par, n, pageURL).Style(func(s *styles.Style) {
			s.Text.WhiteSpace = styles.WhiteSpacePre
		})
	case "ol", "ul":
		// if we are already in a treeview, we just return in the last item in it
		// (which is the list item we are contained in), which fixes the associativity
		// of nested list items and prevents the created of duplicated tree view items.
		if ptv, ok := par.(*giv.TreeView); ok {
			w := ki.LastChild(ptv).(gi.Widget)
			// we also set its class so that the orderedness of nested items works properly
			w.AsWidget().SetClass(tag)
			return w
		}
		tv := giv.NewTreeView(par).SetText("").SetIcon(icons.None).SetClass(tag)
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

		etxt := ExtractText(par, n, pageURL)
		ntv.SetName(etxt)
		ntv.SetText(ftxt + etxt)
	case "img":
		src := GetAttr(n, "src")
		u := grr.Log(ParseRelativeURL(src, pageURL))
		resp, err := http.Get(u.String())
		if grr.Log0(err) != nil {
			return par
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			slog.Error("got error status code", "status", resp.Status)
		}
		if strings.Contains(resp.Header.Get("Content-Type"), "svg") {
			// TODO(kai/gidom): support svg
		} else {
			img := gi.NewImage(par)
			im, _, err := images.Read(resp.Body)
			if err != nil {
				slog.Error("error loading image", "url", u.String(), "err", err)
				return par
			}
			img.Filename = gi.FileName(src)
			img.SetImage(im, 0, 0)
		}
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
		buf.SetText([]byte(ExtractText(par, n, pageURL)))
		texteditor.NewEditor(par).SetBuf(buf)
	default:
		return par
	}
	return nil
}

// OpenURLFunc is the function called to open URLs. Glide sets it
// to a function that opens URLs in glide.
var OpenURLFunc = func(url string) {
	goosi.TheApp.OpenURL(url)
}

// HandleLabel creates a new label from the given information, setting the text and
// the label click function so that URLs are opened according to [OpenURLFunc].
func HandleLabel(par gi.Widget, n *html.Node, pageURL string) *gi.Label {
	lb := gi.NewLabel(par).SetText(ExtractText(par, n, pageURL))
	lb.HandleLabelClick(func(tl *paint.TextLink) {
		url := grr.Log(ParseRelativeURL(tl.URL, pageURL))
		OpenURLFunc(url.String())
	})
	return lb
}

// HandleLabelTag creates a new label from the given information, setting the text and
// the label click function so that URLs are opened according to [OpenURLFunc]. Also,
// it wraps the label text with the [NodeString] of the given node, meaning that it
// should be used for standalone elements that are meant to only exist in labels
// (eg: a, span, b, code, etc).
func HandleLabelTag(par gi.Widget, n *html.Node, pageURL string) *gi.Label {
	start, end := NodeString(n)
	str := start + ExtractText(par, n, pageURL) + end
	lb := gi.NewLabel(par).SetText(str)
	lb.HandleLabelClick(func(tl *paint.TextLink) {
		url := grr.Log(ParseRelativeURL(tl.URL, pageURL))
		OpenURLFunc(url.String())
	})
	return lb
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
