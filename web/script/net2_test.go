// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	neturl "net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// eventuallyJS waits until the given JavaScript expression is the given
// value in the engine.
func eventuallyJS(t *testing.T, e *Engine, expr string, want any) {
	t.Helper()
	var got any
	var err error
	ok := assert.Eventually(t, func() bool {
		got, err = e.RunString(expr)
		return fmt.Sprint(got) == fmt.Sprint(want)
	}, 5*time.Second, 10*time.Millisecond)
	if !ok {
		t.Fatalf("%s: %v (%v), not %v", expr, got, err, want)
	}
}

// TestFormSubmit tests submitting forms: to the page, and to the frame of
// an iframe that the form targets, with POST and GET, and the default
// action of a submit button (Activate).
func TestFormSubmit(t *testing.T) {
	var mu sync.Mutex
	var subs []string
	e, _ := testPageEngine(t, `<html><body>
		<form id="f" method="post" action="/post"><input name="q" value="a b"><button id="s">Go</button></form>
		<form id="g" target="fr" action="/frame"><input name="x" value="1"></form>
		<iframe name="fr" id="fr"></iframe></body></html>`, nil, func(o *Options) {
		o.Submit = func(url, ct string, body []byte) {
			mu.Lock()
			subs = append(subs, "page "+url+" "+string(body))
			mu.Unlock()
		}
		o.SubmitFrame = func(n *html.Node, url, ct string, body []byte, post bool) {
			mu.Lock()
			subs = append(subs, fmt.Sprintf("frame %s %s %v", htmlAttr(n, "id"), url, post))
			mu.Unlock()
		}
	})
	run(t, e, `document.getElementById("f").submit(); document.getElementById("g").submit()`)
	e.Post(func() { e.Activate(findByID(e.Document(), "s")) })
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(subs) == 3
	}, 5*time.Second, 10*time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	assert.Contains(t, subs, "page https://example.com/post q=a+b")
	assert.Contains(t, subs, "frame fr https://example.com/frame?x=1 false")
}

// TestStorageAndCookies tests sessionStorage (of the engine) and
// localStorage, and document.cookie with the cookies of the pages.
func TestStorageAndCookies(t *testing.T) {
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	e, _ := testPageEngine(t, `<html><body></body></html>`, nil, func(o *Options) { o.Cookies = jar })
	assert.Equal(t, "1||0|b", run(t, e, `
		sessionStorage.setItem("a", "1");
		const a = sessionStorage.getItem("a");
		sessionStorage.removeItem("a");
		localStorage.setItem("b", "2");
		localStorage.setItem("c", "3");
		localStorage.removeItem("c");
		const k = localStorage.key(0);
		sessionStorage.setItem("d", "4");
		sessionStorage.clear();
		[a, sessionStorage.getItem("a"), sessionStorage.length, k].join("|")`))
	run(t, e, `document.cookie = "x=1; Path=/"; document.cookie = "y=2"; document.cookie = "bad"`)
	assert.Equal(t, "x=1; y=2", run(t, e, `document.cookie`))
	u, _ := neturl.Parse("https://example.com/")
	assert.Len(t, jar.Cookies(u), 2)

	e2, _ := testPageEngine(t, `<html><body></body></html>`, nil)
	assert.Equal(t, "", run(t, e2, `document.cookie = "z=1"; document.cookie`), "no jar")
}

// TestFetchErrors tests the errors of requests: of getting them, and of
// their URLs, and the digests and texts of their responses, traced.
func TestFetchErrors(t *testing.T) {
	e, fp := testPageEngine(t, `<html><body></body></html>`, map[string]string{"/bin": "\xff\xfeok"}, traced, func(o *Options) {
		get := o.GetURL
		o.GetURL = func(u string) (*http.Response, error) {
			if strings.Contains(u, "fail") {
				return nil, errors.New("no network")
			}
			return get(u)
		}
	})
	run(t, e, `window.r = [];
		fetch("/fail").catch(err => r.push("fail " + err.name));
		fetch("http://[bad").catch(err => r.push("bad " + err.name));
		fetch("/bin").then(x => x.text()).then(s => r.push("text " + s.length));
		const data = new Uint8Array([0, 1, 2, 3, 4]).subarray(1, 3);
		for (const alg of ["SHA-384", "SHA-512"]) crypto.subtle.digest(alg, data).then(d => r.push(alg + " " + d.byteLength));`)
	eventuallyJS(t, e, `r.slice().sort().join()`, "SHA-384 48,SHA-512 64,bad TypeError,fail TypeError,text 3") // (a run of invalid bytes is one U+FFFD)
	assert.True(t, fp.logged("GET"), fp.allLogs())
	assert.Equal(t, "https://example.com/b.js", LogURL("https://example.com/b.js"))
	assert.Equal(t, "data:text/plain,…", LogURL("data:text/plain,abc"))
	long := "https://example.com/" + strings.Repeat("x", 300)
	assert.Equal(t, long[:120]+"…"+long[len(long)-120:], LogURL(long))
}

// TestStyleLinks tests the load and error events of the links of style
// sheets, in the document and inserted, and of preloads, and the scripts
// that fail to load.
func TestStyleLinks(t *testing.T) {
	e, _ := testPageEngine(t, `<html><head>
		<link rel="stylesheet" href="/a.css" onload="r.push('a load')">
		<link rel="stylesheet" href="/none.css" onerror="r.push('none error')">
		<link rel="icon" href="/i.png">
		<script>window.r = []</script>
		<script src="/none.js" onerror="r.push('script error')"></script>
		<script type="module" src="/none.mjs" onerror="r.push('module error')"></script>
		<template><script>r.push("not run")</script></template>
		</head><body></body></html>`, map[string]string{"/a.css": "p {}", "/b.css": "p {}"})
	e.Load(nil)
	eventuallyJS(t, e, `r.slice().sort().join()`, "a load,module error,none error,script error")
	run(t, e, `const l = document.createElement("link"); l.rel = "preload"; l.href = "/b.css";
		l.onload = () => r.push("b load"); document.head.appendChild(l);
		const f = document.createElement("iframe"); f.src = "/f.html"; f.name = "f"; document.body.appendChild(f);`)
	eventuallyJS(t, e, `r.includes("b load")`, true)
	// the scripts in templates are inert, as the template is inserted
	run(t, e, `document.body.insertAdjacentHTML("beforeend", "<template><script>r.push('inserted run')<\/script></template>")`)
	e.Wait()
	assert.Equal(t, false, run(t, e, `r.includes("not run") || r.includes("inserted run")`))
}

// TestWrite tests document.write after the page has loaded, which writes
// at the end of the body, and the state of form controls changed by the
// scripts.
func TestWrite(t *testing.T) {
	e, _ := testPageEngine(t, `<html><body><input id="i" value="a"><input id="c" type="checkbox"><p id="p"></p></body></html>`, nil)
	run(t, e, `document.write("<b id='w'>written</b>")`)
	assert.Equal(t, "BODY", run(t, e, `document.getElementById("w").parentNode.tagName`))
	run(t, e, `document.getElementById("i").value = "changed"; document.getElementById("c").checked = true`)
	ch := make(chan []any, 1)
	e.Post(func() {
		v, hv, _, _ := e.ControlState(findByID(e.Document(), "i"))
		_, _, c, hc := e.ControlState(findByID(e.Document(), "c"))
		_, hp, _, hcp := e.ControlState(findByID(e.Document(), "p"))
		ch <- []any{v, hv, c, hc, hp, hcp}
	})
	assert.Equal(t, []any{"changed", true, true, true, false, false}, <-ch)
	_, err := dataURL("data:no-comma")
	assert.Error(t, err)
	s, err := dataURL("data:;base64,YWI") // without padding
	require.NoError(t, err)
	assert.Equal(t, "ab", s)
}
