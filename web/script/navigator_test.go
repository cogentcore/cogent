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

// TestArrowArguments tests the arguments of a function used only in an
// arrow function in it (as OO.ui.deferMsg of the VisualEditor of
// Wikipedia), which goja gets wrong without the fix of
// https://github.com/dop251/goja/pull/859 (some cases panic), which is
// used locally through a replace in the go.work.
func TestArrowArguments(t *testing.T) {
	e, logs := testEngine(t, `<html><body></body></html>`, nil)
	if run(t, e, `function probe() { return (() => [...arguments])().join(); } probe(1, 2)`) != "1,2" {
		t.Skip("goja without https://github.com/dop251/goja/pull/859")
	}
	assert.Equal(t, "1,2;3;5", run(t, e, `function f() { return (() => [...arguments])().join(); }
		function g() { return (() => arguments.length)(); }
		function h() { const m = () => Math.max(...arguments); return m(); }
		f(1, 2) + ";" + g(1, 2, 3) + ";" + h(1, 5, 2)`))
	assert.Empty(t, *logs)
}
