// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gidom

// Context contains context information needed for gidom calls.
type Context interface {
	// PageURL returns the URL of the current page, and "" if there
	// is no current page.
	PageURL() string

	// SetStyle adds the given CSS style string to the page's styles.
	SetStyle(style string)

	// GetStyle returns the page's styles as a CSS style string.
	GetStyle() string
}

// NilContext returns a [Context] with placeholder implementations of all functions.
func NilContext() Context {
	return &nilContext{}
}

type nilContext struct{}

func (nc *nilContext) PageURL() string       { return "" }
func (nc *nilContext) SetStyle(style string) {}
func (nc *nilContext) GetStyle() string      { return "" }
