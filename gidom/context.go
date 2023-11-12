// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidom

import (
	"github.com/aymerick/douceur/css"
	selcss "github.com/ericchiang/css"
	"goki.dev/goosi"
)

// Context contains context information needed for gidom calls.
type Context interface {
	// PageURL returns the URL of the current page, and "" if there
	// is no current page.
	PageURL() string

	// OpenURL opens the given URL.
	OpenURL(url string) error

	// SetStyle adds the given CSS style string to the page's styles.
	SetStyle(style string)

	// GetStyle returns the page's styles as a CSS style sheet and a slice
	// of selectors with the indices corresponding to those of the rules in
	// the stylesheet.
	GetStyle() (*css.Stylesheet, []*selcss.Selector)
}

// NilContext returns a [Context] with placeholder implementations of all functions.
func NilContext() Context {
	return &nilContext{}
}

type nilContext struct{}

func (nc *nilContext) PageURL() string { return "" }
func (nc *nilContext) OpenURL(url string) error {
	goosi.TheApp.OpenURL(url)
	return nil
}
func (nc *nilContext) SetStyle(style string) {}
func (nc *nilContext) GetStyle() (*css.Stylesheet, []*selcss.Selector) {
	return css.NewStylesheet(), []*selcss.Selector{}
}
