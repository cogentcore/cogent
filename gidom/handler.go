// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidom

import (
	"fmt"
	"io"
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
	"goki.dev/grows/images"
	"goki.dev/grr"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"golang.org/x/net/html"
)

// Handler is a function that can be used to describe the behavior
// of gidom parsing for a specific type of element. It takes the
// parent [gi.Widget] to add widgets to and the [*html.Node] to read from.
// It returns the widget, if any, that has been constructed for this node.
// If it returns false, that indicates that the children of this node have
// already been handled (or will not be handled), and thus should not
// be handled further. If it returns a true, then the children
// will be handled, with the returned widget as their parent.
type Handler func(par gi.Widget, n *html.Node) (w gi.Widget, handleChildren bool)

// ElementHandlers is a map of [Handler] functions for each HTML element
// type (eg: "button", "input", "p"). It is empty by default, but can be
// used by anyone in need of behavior different than the default behavior
// defined in [HandleElement].
var ElementHandlers = map[string]Handler{}

// HandleELement calls the [Handler] associated with the given element [*html.Node]
// in [ElementHandlers] and returns the result, using the given context. If there
// is no handler associated with it, it uses default hardcoded configuration code.
func HandleElement(ctx Context, par gi.Widget, n *html.Node) (w gi.Widget, handleChildren bool) {
	tag := n.DataAtom.String()
	h, ok := ElementHandlers[tag]
	if ok {
		return h(par, n)
	}

	if slices.Contains(TextTags, tag) {
		return HandleLabelTag(ctx, par, n), false
	}

	switch tag {
	case "script", "title", "meta":
		// we don't render anything
	case "link":
		rel := GetAttr(n, "rel")
		// TODO(kai/gidom): maybe handle preload
		if rel == "preload" {
			return
		}
		// TODO(kai/gidom): support links other than stylesheets
		if rel != "stylesheet" {
			return
		}
		resp, err := Get(ctx, GetAttr(n, "href"))
		if grr.Log0(err) != nil {
			return
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if grr.Log0(err) != nil {
			return
		}
		ctx.SetStyle(string(b))
	case "style":
		ctx.SetStyle(ExtractText(ctx, par, n))
	case "div", "section", "nav", "footer", "header":
		w = gi.NewFrame(par)
		handleChildren = true
	case "body", "main":
		w = gi.NewFrame(par)
		handleChildren = true
		w.Style(func(s *styles.Style) {
			s.Direction = styles.Column
		})
	case "button":
		w = HandleLabel(ctx, par, n)
	case "h1":
		w = HandleLabel(ctx, par, n).SetType(gi.LabelHeadlineLarge)
	case "h2":
		w = HandleLabel(ctx, par, n).SetType(gi.LabelHeadlineSmall)
	case "h3":
		w = HandleLabel(ctx, par, n).SetType(gi.LabelTitleLarge)
	case "h4":
		w = HandleLabel(ctx, par, n).SetType(gi.LabelTitleMedium)
	case "h5":
		w = HandleLabel(ctx, par, n).SetType(gi.LabelTitleSmall)
	case "h6":
		w = HandleLabel(ctx, par, n).SetType(gi.LabelLabelSmall)
	case "p":
		w = HandleLabel(ctx, par, n)
	case "pre":
		w = HandleLabel(ctx, par, n).Style(func(s *styles.Style) {
			s.Text.WhiteSpace = styles.WhiteSpacePre
		})
	case "ol", "ul":
		// if we are already in a treeview, we just return in the last item in it
		// (which is the list item we are contained in), which fixes the associativity
		// of nested list items and prevents the created of duplicated tree view items.
		if ptv, ok := par.(*giv.TreeView); ok {
			w := ki.LastChild(ptv).(gi.Widget)
			return w, true
		}
		tv := giv.NewTreeView(par).SetText("").SetIcon(icons.None)
		tv.RootView = tv
		return tv, true
	case "li":
		ntv := giv.NewTreeView(par)
		ftxt := ""
		ptv, ok := par.(*giv.TreeView)
		if ok {
			ntv.RootView = ptv.RootView
			if ptv.Prop("tag") == "ol" {
				ip, _ := ntv.IndexInParent()
				ftxt = strconv.Itoa(ip+1) + ". " // start at 1
			} else {
				// TODO(kai/gidom): have different bullets for different depths
				ftxt = "• "
			}
		} else {
			ntv.RootView = ntv
		}

		etxt := ExtractText(ctx, par, n)
		ntv.SetName(etxt)
		ntv.SetText(ftxt + etxt)
		ntv.OnWidgetAdded(func(w gi.Widget) {
			switch w := w.(type) {
			case *gi.Label:
				w.HandleLabelClick(func(tl *paint.TextLink) {
					url := grr.Log(ParseRelativeURL(tl.URL, ctx.PageURL()))
					grr.Log0(ctx.OpenURL(url.String()))
				})
			}
		})
		w = ntv
	case "img":
		src := GetAttr(n, "src")
		resp, err := Get(ctx, src)
		if grr.Log0(err) != nil {
			return par, true
		}
		defer resp.Body.Close()
		if strings.Contains(resp.Header.Get("Content-Type"), "svg") {
			// TODO(kai/gidom): support svg
		} else {
			img := gi.NewImage(par)
			im, _, err := images.Read(resp.Body)
			if err != nil {
				slog.Error("error loading image", "url", src, "err", err)
				return par, true
			}
			img.Filename = gi.FileName(src)
			img.SetImage(im, 0, 0)
			w = img
		}
	case "input":
		ityp := GetAttr(n, "type")
		switch ityp {
		case "number":
			w = gi.NewSpinner(par)
		case "color":
			w = giv.NewValue(par, colors.Black).AsWidget()
		case "datetime":
			w = giv.NewValue(par, time.Now()).AsWidget()
		default:
			w = gi.NewTextField(par)
		}
	case "textarea":
		buf := texteditor.NewBuf()
		buf.SetText([]byte(ExtractText(ctx, par, n)))
		w = texteditor.NewEditor(par).SetBuf(buf)
	default:
		return par, true
	}
	return
}

// ConfigWidget sets the properties of the given widget based on the properties
// of the given node. It should be called on all widgets in [HandleElement] and
// [Handler] functions.
func ConfigWidget[T gi.Widget](ctx Context, w T, n *html.Node) T {
	wb := w.AsWidget()
	// if we already have the tag prop, we have already been configured
	if _, err := wb.PropTry("tag"); err == nil {
		return w
	}
	for _, attr := range n.Attr {
		switch attr.Key {
		case "id":
			wb.SetName(attr.Val)
		case "class":
			wb.SetClass(attr.Val)
		default:
			wb.SetProp(attr.Key, attr.Val)
		}
	}
	wb.SetProp("tag", n.DataAtom.String())
	ctx.SetWidgetForNode(w, n)
	return w
}

// HandleLabel creates a new label from the given information, setting the text and
// the label click function so that URLs are opened according to [OpenURLFunc].
func HandleLabel(ctx Context, par gi.Widget, n *html.Node) *gi.Label {
	lb := gi.NewLabel(par).SetText(ExtractText(ctx, par, n))
	lb.HandleLabelClick(func(tl *paint.TextLink) {
		url := grr.Log(ParseRelativeURL(tl.URL, ctx.PageURL()))
		grr.Log0(ctx.OpenURL(url.String()))
	})
	return lb
}

// HandleLabelTag creates a new label from the given information, setting the text and
// the label click function so that URLs are opened according to [OpenURLFunc]. Also,
// it wraps the label text with the [NodeString] of the given node, meaning that it
// should be used for standalone elements that are meant to only exist in labels
// (eg: a, span, b, code, etc).
func HandleLabelTag(ctx Context, par gi.Widget, n *html.Node) *gi.Label {
	start, end := NodeString(n)
	str := start + ExtractText(ctx, par, n) + end
	lb := gi.NewLabel(par).SetText(str)
	lb.HandleLabelClick(func(tl *paint.TextLink) {
		url := grr.Log(ParseRelativeURL(tl.URL, ctx.PageURL()))
		grr.Log0(ctx.OpenURL(url.String()))
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

// HasAttr returns whether the given node has the given attribute defined.
func HasAttr(n *html.Node, attr string) bool {
	return slices.ContainsFunc(n.Attr, func(a html.Attribute) bool {
		return a.Key == attr
	})
}

// Get is a helper function that calls [http.Get] with the given URL, parsed
// relative to the page URL of the given context. It also checks the status
// code of the response and closes the response body and returns an error if
// it is not [http.StatusOK]. If the error is nil, then the response body is
// not closed and must be closed by the caller.
func Get(ctx Context, url string) (*http.Response, error) {
	u, err := ParseRelativeURL(url, ctx.PageURL())
	if err != nil {
		return nil, err
	}
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return resp, fmt.Errorf("got error status %q (code %d)", resp.Status, resp.StatusCode)
	}
	return resp, nil
}
