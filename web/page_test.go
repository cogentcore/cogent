// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"cogentcore.org/core/styles"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
)

// TestPageFile shows the page in the file of the WEB_PAGE environment
// variable, served as the URL in WEB_URL (with its other resources from
// the web), printing its first texts, and saving its image to WEB_OUT.
func TestPageFile(t *testing.T) {
	file := os.Getenv("WEB_PAGE")
	if file == "" {
		t.Skip("set WEB_PAGE to a saved HTML page")
	}
	b := core.NewBody()
	pg := NewPage(b)
	pg.Context.GetURL = func(u string) (*http.Response, error) {
		pu, _ := url.Parse(u)
		if pu.Path == "/" || pu.Path == "" || pu.Path == os.Getenv("WEB_PATH") {
			f, err := os.Open(file)
			if err != nil {
				return nil, err
			}
			return &http.Response{StatusCode: 200, Status: "200 OK", Body: f, Request: &http.Request{URL: pu}}, nil
		}
		return http.Get(u)
	}
	pg.OpenURL(os.Getenv("WEB_URL"), nil)
	img := b.RenderImage(pg.wait, func() {
		n := 0
		pg.WalkDown(func(nd tree.Node) bool {
			if tx, ok := nd.(*core.Text); ok && n < 5 {
				fmt.Println(tx.Text, tx.Geom.Pos.Total, tx.Geom.TotalBBox, tx.Styles.Color, tx.IsVisible())
				n++
			}
			return true
		})
	})
	if out := os.Getenv("WEB_OUT"); out != "" {
		imagex.Save(img, out)
	}
}

func TestLocationURL(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.html")
	os.WriteFile(file, []byte("<p>x</p>"), 0666)
	for loc, want := range map[string]string{
		"https://example.com/": "https://example.com/",
		"example.com":          "https://example.com",
		"cogent core":          "https://html.duckduckgo.com/html/?q=cogent+core",
		file:                   "file://" + filepath.ToSlash(file),
	} {
		assert.Equal(t, want, LocationURL(loc), loc)
	}
}

// TestOpenFile tests opening a local file and a relative link in it.
func TestOpenFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.html"), []byte(`<p><a href="b.html">to b</a></p>`), 0666)
	os.WriteFile(filepath.Join(dir, "b.html"), []byte(`<p>page b</p>`), 0666)
	b := core.NewBody()
	pg := NewPage(b)
	pg.OpenURL(LocationURL(filepath.Join(dir, "a.html")), nil)
	b.RenderImage(pg.wait, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		if assert.Len(t, pg.History, 1) {
			assert.True(t, strings.HasPrefix(pg.History[0], "file://"))
		}
		pg.OpenURL("b.html", nil) // as for a click on the link
	}, pg.wait, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		if assert.Len(t, pg.History, 2) {
			assert.Equal(t, "file://"+filepath.ToSlash(filepath.Join(dir, "b.html")), pg.History[1])
		}
		found := false
		pg.WalkDown(func(n tree.Node) bool {
			if tx, ok := n.(*core.Text); ok && strings.Contains(tx.Text, "page b") {
				found = true
			}
			return tree.Continue
		})
		assert.True(t, found)
	})
}

// TestPageErrorStatusShown tests that a page of HTML with an error status
// is shown, as browsers show it (as the page of google.com about unusual
// traffic, of status 429), and that one without HTML is an error.
func TestPageErrorStatusShown(t *testing.T) {
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.X.Dp(800); s.Min.Y.Dp(600) })
	pg := NewPage(b)
	pg.Scripts = false
	pg.Context.GetURL = func(u string) (*http.Response, error) {
		pu, _ := url.Parse(u)
		h := http.Header{}
		body := "not a page"
		if pu.Path == "/sorry" {
			h.Set("Content-Type", "text/html; charset=UTF-8")
			body = `<html><body><p>Our systems have detected unusual traffic</p></body></html>`
		}
		return &http.Response{StatusCode: 429, Status: "429 Too Many Requests", Header: h, Request: &http.Request{URL: pu}, Body: io.NopCloser(strings.NewReader(body))}, nil
	}
	pg.OpenURL("https://example.com/sorry", nil)
	b.RenderImage(func() { pg.wait() }, func() {
		assert.Contains(t, pageTexts(b, pg), "unusual traffic")
	})
}
