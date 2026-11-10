// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cogentcore.org/core/htmlcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
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
// Wikipedia), which goja got wrong before its commit df46281 (some
// cases panicked).
func TestArrowArguments(t *testing.T) {
	e, logs := testEngine(t, `<html><body></body></html>`, nil)
	assert.Equal(t, "1,2;3;5", run(t, e, `function f() { return (() => [...arguments])().join(); }
		function g() { return (() => arguments.length)(); }
		function h() { const m = () => Math.max(...arguments); return m(); }
		f(1, 2) + ";" + g(1, 2, 3) + ";" + h(1, 5, 2)`))
	assert.Empty(t, *logs)
}

// TestWeakRef tests WeakRef and FinalizationRegistry, which the engine does
// not have (the scripts of google.com use WeakRef, without which they
// stopped): a WeakRef derefs to its target.
func TestWeakRef(t *testing.T) {
	e, logs := testEngine(t, `<html><body></body></html>`, nil)
	assert.Equal(t, "true,false", run(t, e, `const o = {}; const r = new WeakRef(o);
		const f = new FinalizationRegistry(() => {}); f.register(o, 1);
		[r.deref() === o, f.unregister(o)].join()`))
	assert.Empty(t, *logs)
}

// TestCanvas2D tests the 2D context of a canvas, which draws nothing, but
// with which scripts that draw (as the QR codes of the postings of
// craigslist.org, whose script stopped with no context) work; other
// contexts are not supported.
func TestCanvas2D(t *testing.T) {
	e, logs := testEngine(t, `<html><body><canvas id="c" width="10" height="10"></canvas></body></html>`, nil)
	assert.Equal(t, "true,true,,40,40", run(t, e, `const c = document.getElementById("c"); const x = c.getContext("2d");
		x.fillStyle = "#fff"; x.fillRect(0, 0, 5, 5); x.beginPath(); x.arc(5, 5, 2, 0, 7); x.fill();
		[x === c.getContext("2d"), x.canvas === c, c.getContext("webgl"), x.getImageData(0, 0, 2, 5).data.length, x.measureText("abcd").width > 0 ? 40 : 0].join()`))
	assert.Empty(t, *logs)
}

// TestThrottleLayout tests that the boxes that scripts get are those of
// the last render (not a new render of the page) when rendering it is
// expensive and it was just rendered (as for the scripts of craigslist.org,
// which re-read its large pages on each of their timers), and not when
// rendering it is cheap or the last render was long enough ago, or when a
// task gets boxes again (a few times).
func TestThrottleLayout(t *testing.T) {
	e := &Engine{pageLayouts: maxPageLayouts}
	e.renderCost, e.lastRender = 2*time.Second, time.Now()
	assert.True(t, e.throttleLayout(), "expensive and just rendered")
	e.lastRender = time.Now().Add(-5 * time.Second)
	assert.False(t, e.throttleLayout(), "rendered long enough ago")
	e.renderCost, e.lastRender = 5*time.Millisecond, time.Now()
	assert.False(t, e.throttleLayout(), "cheap to render")

	// a task that gets boxes again (after it changed the document, as the
	// script of the navigation of edition.cnn.com) gets them rendered, a
	// few times
	e = &Engine{pageLayouts: maxPageLayouts}
	e.renderCost, e.lastRender = 2*time.Second, time.Now()
	assert.True(t, e.throttleLayout(), "the first boxes of a task")
	for i := range maxTaskLayouts {
		assert.False(t, e.throttleLayout(), "again in the task %d", i)
	}
	assert.True(t, e.throttleLayout(), "too many times in the task")

	// the first layouts of a page are not limited
	e = &Engine{}
	e.renderCost, e.lastRender = 2*time.Second, time.Now()
	for i := range maxPageLayouts {
		e.taskBoxes = 0
		assert.False(t, e.throttleLayout(), "a first layout of the page %d", i)
	}
	e.taskBoxes = 0
	assert.True(t, e.throttleLayout(), "after the first layouts")
}

// TestHasListenerInterrupted tests HasListener while the engine is
// interrupted, as when it is closed while the page is rendered (as by
// reloading pkg.go.dev, which crashed with a nil value): it is false.
func TestHasListenerInterrupted(t *testing.T) {
	e, _ := testEngine(t, `<html><body><a href="#">x</a></body></html>`, nil)
	body := e.doc.LastChild.LastChild
	e.vm.Interrupt(errClosed)
	assert.NotPanics(t, func() { assert.False(t, e.HasListener(body, "click")) })
}

// TestDialogModal tests showModal, which marks the dialog for the :modal
// selector of the style sheets (htmlcore.ModalAttr, as the jump dialog of
// pkg.go.dev, centered over the page), and close, which unmarks it.
func TestDialogModal(t *testing.T) {
	e, logs := testEngine(t, `<html><body><dialog id="d"><p>x</p></dialog></body></html>`, nil)
	assert.Equal(t, "true,true,false,false,close", run(t, e, `const d = document.getElementById("d"); let ev = "";
		d.addEventListener("close", e => ev = e.type); d.showModal();
		const a = [d.open, d.hasAttribute("data-htmlcore-modal")]; d.close();
		a.concat([d.open, d.hasAttribute("data-htmlcore-modal"), ev]).join()`))
	assert.Empty(t, *logs)
}

// TestResizeObserverChanges tests that a ResizeObserver reports the size
// of its target when it is observed and again when it changes (as the
// header of edition.cnn.com, which sizes itself by its ad slot as that is
// laid out), and not when it does not.
func TestResizeObserverChanges(t *testing.T) {
	doc, err := html.ParseWithOptions(strings.NewReader(`<html><body><div id="d"></div></body></html>`), html.ParseOptionEnableScripting(true))
	require.NoError(t, err)
	var h atomic.Int64
	h.Store(10)
	e, err := New(doc, Options{PageURL: "https://example.com/",
		Layout: func(n *html.Node) (x, y, w, hh float32, ok bool) { return 0, 0, 100, float32(h.Load()), true }})
	require.NoError(t, err)
	defer e.Close()
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	_, err = e.RunString(`window.sizes = []; new ResizeObserver(es => { for (const x of es) sizes.push(x.contentRect.height) }).observe(document.getElementById("d"))`)
	require.NoError(t, err)
	time.Sleep(600 * time.Millisecond)
	h.Store(40)
	time.Sleep(600 * time.Millisecond)
	v, err := e.RunString(`sizes.join()`)
	require.NoError(t, err)
	assert.Equal(t, "10,40", v)
}

// TestFetchGetHeaders tests that a GET of a script with headers is sent
// with them (not with GetURL, which has none: the page_data of github.com,
// with Accept: application/json, was 406 Not Acceptable).
func TestFetchGetHeaders(t *testing.T) {
	e, logs := testEngine(t, `<html><body></body></html>`, nil)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/json" {
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}
		w.Write([]byte(`{"ok": 1}`))
	}))
	defer srv.Close()
	run(t, e, `window.got = ""; fetch("`+srv.URL+`/page_data", {headers: {"Accept": "application/json"}}).then(r => r.status === 200 ? r.json() : {ok: r.status}).then(j => { got = String(j.ok); })`)
	assert.Eventually(t, func() bool {
		v, _ := e.RunString(`got`)
		return v == "1"
	}, 3*time.Second, 20*time.Millisecond, "%v", *logs)
}

// TestNavigatorMediaDevicesCredentials tests the media devices (none, as
// Google account pages list them) and the credentials (which sites fall
// back from) of navigator.
func TestNavigatorMediaDevicesCredentials(t *testing.T) {
	e, logs := testEngine(t, `<html><body></body></html>`, nil)
	run(t, e, `window.res = "";
		navigator.mediaDevices.enumerateDevices().then(ds => { res += "devices " + ds.length + ";"; });
		navigator.mediaDevices.getUserMedia({audio: true}).catch(err => { res += err.name + ";"; });
		navigator.credentials.get({publicKey: {}}).catch(err => { res += "credentials " + err.name + ";"; }); 0`)
	assert.Eventually(t, func() bool {
		v, _ := e.RunString(`res`)
		s, _ := v.(string)
		return strings.Contains(s, "devices 0;") && strings.Contains(s, "NotAllowedError;") && strings.Contains(s, "credentials NotAllowedError;")
	}, 3*time.Second, 20*time.Millisecond, "%v", *logs)
}
