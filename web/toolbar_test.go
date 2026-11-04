// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// TestToolbar tests the toolbar of a page in a top bar, as in the web app.
func TestToolbar(t *testing.T) {
	b := core.NewBody()
	pg := NewPage(b)
	pg.Context.GetURL = func(u string) (*http.Response, error) {
		pu, _ := url.Parse(u)
		r := strings.NewReader(`<p>A page</p>`)
		return &http.Response{StatusCode: 200, Status: "200 OK", Request: &http.Request{URL: pu}, Body: io.NopCloser(r)}, nil
	}
	b.AddTopBar(func(bar *core.Frame) {
		tb := core.NewToolbar(bar)
		tb.Maker(pg.MakeToolbar)
		pg.OnOpen = func(url string) { tb.Update() }
	})
	pg.OpenURL("https://example.com/", nil)
	var back *core.Button
	var addr *core.TextField
	b.RenderImage(func() {
		b.Scene.WalkDown(func(n tree.Node) bool {
			switch w := n.(type) {
			case *core.Button:
				if w.Name == "back" {
					back = w
				}
			case *core.TextField:
				if w.Name == "url" {
					addr = w
				}
			}
			return tree.Continue
		})
	})
	if assert.NotNil(t, back) && assert.NotNil(t, addr) {
		assert.Equal(t, "https://example.com/", addr.Text())
		assert.True(t, back.StateIs(states.Disabled), "no page to go back to")
	}
}
