// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glide

import "goki.dev/glide/gidom"

// Context implements [gidom.Context]
type Context struct {
	gidom.ContextBase
	// Page is the page associated with the context
	Page *Page
}

// Context returns the [Context] for the page.
func (pg *Page) Context() *Context {
	return &Context{
		ContextBase: gidom.ContextBase{},
		Page:        pg,
	}
}

func (c *Context) PageURL() string {
	return c.Page.PageURL
}

func (c *Context) OpenURL(url string) error {
	return c.Page.OpenURL(url)
}
