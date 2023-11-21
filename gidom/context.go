// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidom

import (
	"fmt"
	"strings"

	"github.com/aymerick/douceur/css"
	"github.com/aymerick/douceur/parser"
	selcss "github.com/ericchiang/css"
	"goki.dev/gi/v2/gi"
	"goki.dev/goosi"
	"goki.dev/grr"
	"golang.org/x/net/html"
)

// Context contains context information needed for gidom calls.
type Context interface {
	Node() *html.Node
	SetNode(node *html.Node)

	// ParentFor returns the current parent widget that a widget
	// associated with the given node should be added to.
	Parent() gi.Widget

	// Parent returns the current parent widget that non-inline elements
	// should be added to.
	BlockParent() gi.Widget

	// InlineParent returns the current parent widget that inline
	// elements should be added to.
	InlineParent() gi.Widget

	// SetParent sets the current parent widget that non-inline elements
	// should be added to.
	SetParent(pw gi.Widget)

	// PageURL returns the URL of the current page, and "" if there
	// is no current page.
	PageURL() string

	// OpenURL opens the given URL.
	OpenURL(url string) error

	// SetWidgetForNode associates the given widget with the given node.
	SetWidgetForNode(w gi.Widget, n *html.Node)

	// WidgetForNode returns the widget associated with the given node.
	WidgetForNode(n *html.Node) gi.Widget

	// AddStyle adds the given CSS style string to the page's styles.
	AddStyle(style string)

	Style() []*css.Rule
}

// BaseContext returns a [Context] with basic implementations of all functions.
func BaseContext() Context {
	return &ContextBase{}
}

// ContextBase contains basic implementations of all [Context] functions.
type ContextBase struct {
	Nd *html.Node

	Rules map[*html.Node][]*css.Rule

	WidgetsForNodes map[*html.Node]gi.Widget
	BlockPw         gi.Widget
	InlinePw        gi.Widget
}

func (cb *ContextBase) Node() *html.Node {
	return cb.Nd
}

func (cb *ContextBase) SetNode(node *html.Node) {
	cb.Nd = node
}

func (cb *ContextBase) Parent() gi.Widget {
	return cb.BlockParent()
}

func (cb *ContextBase) BlockParent() gi.Widget {
	return cb.BlockPw
}

func (cb *ContextBase) InlineParent() gi.Widget {
	if cb.InlinePw != nil {
		return cb.InlinePw
	}
	cb.InlinePw = gi.NewLayout(cb.BlockPw, fmt.Sprintf("inline-container-%d", cb.BlockPw.NumLifetimeChildren()))
	return cb.InlinePw
}

func (cb *ContextBase) SetParent(pw gi.Widget) {
	cb.BlockPw = pw
	cb.InlinePw = nil // gets reset
}

// PageURL returns the URL of the current page, and "" if there
// is no current page.
func (cb *ContextBase) PageURL() string { return "" }

// OpenURL opens the given URL.
func (cb *ContextBase) OpenURL(url string) error {
	goosi.TheApp.OpenURL(url)
	return nil
}

// SetWidgetForNode associates the given widget with the given node.
func (cb *ContextBase) SetWidgetForNode(w gi.Widget, n *html.Node) {
	if cb.WidgetsForNodes == nil {
		cb.WidgetsForNodes = make(map[*html.Node]gi.Widget)
	}
	cb.WidgetsForNodes[n] = w
}

// WidgetForNode returns the widget associated with the given node.
func (cb *ContextBase) WidgetForNode(n *html.Node) gi.Widget {
	if cb.WidgetsForNodes == nil {
		cb.WidgetsForNodes = make(map[*html.Node]gi.Widget)
	}
	return cb.WidgetsForNodes[n]
}

// AddStyle adds the given CSS style string to the page's styles.
func (cb *ContextBase) AddStyle(style string) {
	ss, err := parser.Parse(style)
	if grr.Log0(err) != nil {
		return
	}

	if cb.Rules == nil {
		cb.Rules = map[*html.Node][]*css.Rule{}
	}

	for _, rule := range ss.Rules {
		var sel *selcss.Selector
		if len(rule.Selectors) > 0 {
			s, err := selcss.Parse(strings.Join(rule.Selectors, ","))
			if grr.Log0(err) != nil {
				s = &selcss.Selector{}
			}
			sel = s
		} else {
			sel = &selcss.Selector{}
		}

		matches := sel.Select(cb.Node())
		for _, match := range matches {
			cb.Rules[match] = append(cb.Rules[match], rule)
		}
	}
}

// Style returns the styles for the current node.
func (cb *ContextBase) Style() []*css.Rule {
	if cb.Rules == nil {
		return nil
	}
	return cb.Rules[cb.Node()]
}

// // ParseStyle returns the page's styles as a CSS style sheet and a slice
// // of selectors with the indices corresponding to those of the rules in
// // the stylesheet.
// func (cb *ContextBase) ParseStyle() {
// 	ss, err := parser.Parse(cb.CurStyle)
// 	if grr.Log0(err) != nil {
// 		return
// 	}
// 	cb.Ss = ss

// 	sels := make([]*selcss.Selector, len(ss.Rules))
// 	for i, rule := range ss.Rules {
// 		var sel *selcss.Selector
// 		if len(rule.Selectors) > 0 {
// 			s, err := selcss.Parse(strings.Join(rule.Selectors, ","))
// 			if grr.Log0(err) != nil {
// 				s = &selcss.Selector{}
// 			}
// 			sel = s
// 		} else {
// 			sel = &selcss.Selector{}
// 		}
// 		sels[i] = sel
// 	}
// 	cb.Sels = sels
// 	cb.StyleValid = true
// 	return
// }
