// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"cogentcore.org/core/htmlcore"
	"github.com/stretchr/testify/assert"
)

// TestNavigatorUserAgent tests that the user agent and languages of
// scripts are those of the requests of the page (sites check that they
// match), and that the requests of scripts have the headers of browsers,
// except those that the scripts set.
func TestNavigatorUserAgent(t *testing.T) {
	assert.Equal(t, []any{"fr-CA", "fr", "en"}, navigatorLanguages("fr-CA,fr;q=0.9,en;q=0.8"))
	e, logs := testEngine(t, `<html><body></body></html>`, nil)
	assert.Equal(t, htmlcore.UserAgent, run(t, e, `navigator.userAgent`))
	assert.Equal(t, navigatorLanguages(htmlcore.AcceptLanguage)[0], run(t, e, `navigator.language`))
	assert.Equal(t, navigatorPlatform(), run(t, e, `navigator.platform`))

	var mu sync.Mutex
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		got = r.Header.Clone()
		mu.Unlock()
	}))
	defer srv.Close()
	run(t, e, `window.done = false; fetch("`+srv.URL+`", {method: "POST", body: "x", headers: {"Accept": "application/json", "X-A": "1"}}).then(() => { done = true; })`)
	assert.Eventually(t, func() bool {
		v, _ := e.RunString(`done`)
		return v == true
	}, 3*time.Second, 20*time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, htmlcore.UserAgent, got.Get("User-Agent"))
	assert.Equal(t, htmlcore.AcceptLanguage, got.Get("Accept-Language"))
	assert.Equal(t, []string{"application/json"}, got.Values("Accept"))
	assert.Equal(t, "1", got.Get("X-A"))
	assert.Empty(t, *logs)
}

// TestDocumentsForEditors tests the features of documents that the
// VisualEditor of Wikipedia checks for or uses: documents parsed with
// DOMParser are HTMLDocuments, whose baseURI is that of their base
// element, and svg elements have createSVGRect.
func TestDocumentsForEditors(t *testing.T) {
	e, logs := testEngine(t, `<html><body></body></html>`, nil)
	assert.Equal(t, "true,true,true,https://en.wikipedia.org/wiki/,https://example.com/page.html,https://en.wikipedia.org/wiki/Category:X",
		run(t, e, `const d = new DOMParser().parseFromString('<head><base href="//en.wikipedia.org/wiki/"></head><body><a href="./Category:X">x</a></body>', "text/html");
			[d instanceof HTMLDocument, document instanceof HTMLDocument, "createSVGRect" in document.createElementNS("http://www.w3.org/2000/svg", "svg"),
			d.baseURI, document.baseURI, new URL(d.querySelector("a").getAttribute("href"), d.baseURI).href].join(",")`))
	assert.Empty(t, *logs)
}
