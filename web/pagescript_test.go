// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// newScriptPage returns a body with a page that serves the given pages
// and resources by path (of https://example.com/), opening "/".
func newScriptPage(t *testing.T, pages map[string]string) (*core.Body, *Page) {
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.X.Dp(800); s.Min.Y.Dp(600) })
	pg := NewPage(b)
	pg.Context.GetURL = func(u string) (*http.Response, error) {
		pu, _ := url.Parse(u)
		body, ok := pages[pu.Path]
		if !ok {
			return &http.Response{StatusCode: 404, Status: "404 Not Found", Request: &http.Request{URL: pu}, Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		return &http.Response{StatusCode: 200, Status: "200 OK", Request: &http.Request{URL: pu}, Body: io.NopCloser(strings.NewReader(body))}, nil
	}
	pg.OpenURL("https://example.com/", nil)
	return b, pg
}

// pageTexts returns the texts of the widgets of the page.
func pageTexts(b *core.Body, pg *Page) string {
	b.AsyncLock()
	defer b.AsyncUnlock()
	var ts []string
	pg.WalkDown(func(n tree.Node) bool {
		if tx, ok := n.(*core.Text); ok {
			ts = append(ts, tx.Text)
		}
		return tree.Continue
	})
	return strings.Join(ts, "|")
}

// settle waits for the page to open and its scripts to be idle.
func settle(pg *Page) {
	pg.wait()
	if ps := pg.scripts; ps != nil {
		ps.engine.Wait()
	}
}

// eventually returns whether the given condition becomes true soon.
func eventually(cond func() bool) bool {
	for range 200 {
		if cond() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// TestScriptsBuildPage tests a page whose content is made by its script,
// and that it is updated by a timer and by a list that it fetches.
func TestScriptsBuildPage(t *testing.T) {
	b, pg := newScriptPage(t, map[string]string{
		"/": `<html><body><div id="app"></div><p id="later">waiting</p><ul id="list"></ul>
			<noscript><p>Please enable JavaScript</p></noscript>
			<script>
				const h = document.createElement("h1");
				h.textContent = "Made by a script";
				document.getElementById("app").appendChild(h);
				setTimeout(() => { document.getElementById("later").textContent = "timer fired"; }, 30);
				fetch("/items.json").then(r => r.json()).then(items => {
					document.getElementById("list").innerHTML = items.map(i => "<li>" + i + "</li>").join("");
				});
			</script></body></html>`,
		"/items.json": `["first item", "second item"]`,
	})
	var texts string
	b.RenderImage(func() { settle(pg) }, func() {
		assert.True(t, eventually(func() bool {
			texts = pageTexts(b, pg)
			return strings.Contains(texts, "timer fired") && strings.Contains(texts, "second item")
		}), "the page is updated: %s", texts)
	})
	assert.Contains(t, texts, "Made by a script")
	assert.Contains(t, texts, "first item")
	assert.NotContains(t, texts, "Please enable JavaScript", "noscript is not shown")
}

// TestScriptsClick tests that clicking an element with a click handler
// runs it, and that the page shows what it changes.
func TestScriptsClick(t *testing.T) {
	b, pg := newScriptPage(t, map[string]string{
		"/": `<html><body>
			<button id="b" onclick="this.textContent = 'Clicked ' + (++window.n) + ' times'">Click me</button>
			<div id="d">Div</div>
			<script>
				window.n = 0;
				document.getElementById("d").addEventListener("click", e => { e.target.textContent = "Div clicked"; });
			</script></body></html>`,
	})
	// clickNode sends a click to the widget of the element of the given id
	clickNode := func(id string) {
		b.AsyncLock()
		var target core.Widget
		pg.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			if cwb.Property("id") == id && target == nil {
				target = cw
			}
			return tree.Continue
		})
		if assert.NotNil(t, target, id) {
			target.AsWidget().Send(events.Click) // as the event loop does, with the lock held
		}
		b.AsyncUnlock()
	}
	b.RenderImage(func() { settle(pg) }, func() {
		clickNode("b")
		clickNode("b")
		clickNode("d")
	}, func() {
		assert.True(t, eventually(func() bool {
			s := pageTexts(b, pg)
			return strings.Contains(s, "Clicked 2 times") && strings.Contains(s, "Div clicked")
		}), pageTexts(b, pg))
	})
}

// TestScriptsLinks tests that a link whose click a script prevents does
// not open its page, and that one that it does not prevent does.
func TestScriptsLinks(t *testing.T) {
	b, pg := newScriptPage(t, map[string]string{
		"/": `<html><body><p><a id="stay" href="/other">Stay</a> <a id="go" href="/other">Go</a></p>
			<script>
				document.getElementById("stay").addEventListener("click", e => { e.preventDefault(); e.target.textContent = "Stayed"; });
			</script></body></html>`,
		"/other": `<html><body><p>The other page</p></body></html>`,
	})
	b.RenderImage(func() { settle(pg) }, func() {
		b.AsyncLock()
		pg.scripts.openURL("https://example.com/other", nil) // the first link
		b.AsyncUnlock()
	}, func() {
		settle(pg)
		assert.True(t, eventually(func() bool { return strings.Contains(pageTexts(b, pg), "Stayed") }))
		assert.Equal(t, 1, len(pg.History), "the page is not opened")
		// a script navigates to the other page
		pg.scripts.engine.Eval(`location.href = "/other"`)
	}, func() {
		assert.True(t, eventually(func() bool { return strings.Contains(pageTexts(b, pg), "The other page") }), pageTexts(b, pg))
	})
	assert.Equal(t, "https://example.com/other", pg.currentURL())
}

// TestScriptsLayout tests that scripts get the sizes of the boxes of the
// elements of the page as it is laid out.
func TestScriptsLayout(t *testing.T) {
	b, pg := newScriptPage(t, map[string]string{
		"/": `<html><body style="margin: 0"><div id="box" style="width: 300px; height: 50px">box</div></body></html>`,
	})
	b.RenderImage(func() { settle(pg) }, func() {
		v, err := pg.scripts.engine.RunString(`const r = document.getElementById("box").getBoundingClientRect(); [Math.round(r.width), Math.round(r.height), innerWidth > 0, document.documentElement.clientWidth === innerWidth].join(",")`)
		assert.NoError(t, err)
		assert.Equal(t, "300,50,true,true", v, "the client width of the root element is that of the viewport")
	})
}

// TestScriptsOff tests that the scripts of pages do not run when the
// scripts are turned off, and that noscript content is then shown.
func TestScriptsOff(t *testing.T) {
	b := core.NewBody()
	pg := NewPage(b)
	pg.Scripts = false
	pg.Context.GetURL = func(u string) (*http.Response, error) {
		pu, _ := url.Parse(u)
		return &http.Response{StatusCode: 200, Status: "200 OK", Request: &http.Request{URL: pu}, Body: io.NopCloser(strings.NewReader(
			`<html><body><p id="p">static</p><noscript><p>No scripts here</p></noscript><script>document.getElementById("p").textContent = "changed";</script></body></html>`))}, nil
	}
	pg.OpenURL("https://example.com/", nil)
	b.RenderImage(func() { settle(pg) }, func() {
		s := pageTexts(b, pg)
		assert.Contains(t, s, "static")
		assert.NotContains(t, s, "changed")
		assert.Contains(t, s, "No scripts here")
		assert.Nil(t, pg.scripts)
	})
}

// TestScriptsForms tests that the scripts get the submit event of a form
// with a submit listener, which can prevent it (as to send it themselves),
// and that a form without listeners is submitted as in HTML.
func TestScriptsForms(t *testing.T) {
	b, pg := newScriptPage(t, map[string]string{
		"/": `<html><body>
			<form id="js" action="/search"><input name="q" value="first"><button id="go">Go</button></form>
			<p id="out">none</p>
			<form id="plain" action="/search"><input name="q" value="second"><button id="plaingo">Search</button></form>
			<script>
				const f = document.getElementById("js");
				f.addEventListener("submit", e => {
					e.preventDefault();
					document.getElementById("out").textContent = "submitted " + new FormData(f).get("q");
				});
			</script></body></html>`,
		"/search": `<html><body><p>Results</p></body></html>`,
	})
	clickID := func(id string) {
		b.AsyncLock()
		defer b.AsyncUnlock()
		var target core.Widget
		pg.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			if cwb.Property("id") == id && target == nil {
				target = cw
			}
			return tree.Continue
		})
		if assert.NotNil(t, target, id) {
			target.AsWidget().Send(events.Click)
		}
	}
	b.RenderImage(func() { settle(pg) }, func() {
		clickID("go")
	}, func() {
		assert.True(t, eventually(func() bool { return strings.Contains(pageTexts(b, pg), "submitted first") }), pageTexts(b, pg))
		assert.Equal(t, 1, len(pg.History), "the form with a listener is not submitted")
		clickID("plaingo")
	}, func() {
		assert.True(t, eventually(func() bool { return strings.Contains(pageTexts(b, pg), "Results") }), pageTexts(b, pg))
	})
	assert.Equal(t, "https://example.com/search?q=second", pg.currentURL())
}

// TestScriptsTyping tests that typing in a field whose input listener
// changes the page (as a search as you type) keeps what is typed in the
// field, and its focus, when the page is rendered again.
func TestScriptsTyping(t *testing.T) {
	b, pg := newScriptPage(t, map[string]string{
		"/": `<html><body><input id="q" value=""><p id="out">nothing</p>
			<script>
				document.getElementById("q").addEventListener("input", e => {
					document.getElementById("out").textContent = "typed " + e.target.value;
				});
			</script></body></html>`,
	})
	field := func() *core.TextField {
		var tf *core.TextField
		pg.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			if x, ok := cw.(*core.TextField); ok && tf == nil {
				tf = x
			}
			return tree.Continue
		})
		return tf
	}
	var first *core.TextField
	b.RenderImage(func() { settle(pg) }, func() {
		b.AsyncLock()
		first = field()
		if assert.NotNil(t, first) {
			first.SetFocus()
			first.SetText("abc")
			first.Send(events.Input)
		}
		b.AsyncUnlock()
	}, func() {
		assert.True(t, eventually(func() bool { return strings.Contains(pageTexts(b, pg), "typed abc") }), pageTexts(b, pg))
		b.AsyncLock()
		defer b.AsyncUnlock()
		tf := field()
		if assert.NotNil(t, tf) {
			assert.NotSame(t, first, tf, "the page was rendered again")
			assert.Equal(t, "abc", tf.Text(), "the field keeps what was typed")
			assert.True(t, tf.StateIs(states.Focused), "the field keeps the focus")
		}
	})
}

// TestScriptsPost tests a form with the POST method that a script fills in
// and submits (as the challenge page of Reddit does), and that the cookie
// that the response sets is sent with the request of the page that it
// redirects to.
func TestScriptsPost(t *testing.T) {
	var posted string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			io.WriteString(w, `<html><body><p>checking</p>
				<form method="post" action="/check"><input type="hidden" name="solution"></form>
				<script>
					document.addEventListener("DOMContentLoaded", () => {
						const f = document.forms[0];
						f.elements.namedItem("solution").value = "a b&c";
						f.requestSubmit();
					});
				</script></body></html>`)
		case "/check":
			r.ParseForm()
			posted = r.Method + " " + r.Header.Get("Content-Type") + " " + r.PostForm.Get("solution")
			http.SetCookie(w, &http.Cookie{Name: "pass", Value: "granted", Path: "/"})
			http.Redirect(w, r, "/content", http.StatusSeeOther)
		case "/content":
			c, err := r.Cookie("pass")
			if err != nil {
				io.WriteString(w, `<html><body><p>no cookie</p></body></html>`)
				return
			}
			io.WriteString(w, `<html><body><p>content with `+c.Value+`</p></body></html>`)
		}
	}))
	defer srv.Close()
	b := core.NewBody()
	pg := NewPage(b)
	pg.OpenURL(srv.URL+"/", nil)
	b.RenderImage(func() { settle(pg) }, func() {
		assert.True(t, eventually(func() bool { return strings.Contains(pageTexts(b, pg), "content with granted") }), pageTexts(b, pg))
	})
	assert.Equal(t, "POST application/x-www-form-urlencoded a b&c", posted)
	assert.Equal(t, srv.URL+"/content", pg.currentURL())
}

// TestFormPost tests that a form with the POST method and no listeners
// (as the cookie consent of google.com) is submitted when its button is
// clicked, with scripts and without.
func TestFormPost(t *testing.T) {
	for _, scripts := range []bool{true, false} {
		var posted string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/":
				io.WriteString(w, `<html><body><form method="post" action="/save">
					<input type="hidden" name="set_eom" value="true"><button id="reject">Reject all</button></form></body></html>`)
			case "/save":
				r.ParseForm()
				posted = r.Method + " " + r.PostForm.Get("set_eom")
				http.Redirect(w, r, "/search", http.StatusSeeOther)
			case "/search":
				io.WriteString(w, `<html><body><p>Results</p></body></html>`)
			}
		}))
		b := core.NewBody()
		pg := NewPage(b)
		pg.Scripts = scripts
		pg.OpenURL(srv.URL+"/", nil)
		b.RenderImage(func() { settle(pg) }, func() {
			b.AsyncLock()
			defer b.AsyncUnlock()
			var target core.Widget
			pg.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
				if cwb.Property("id") == "reject" && target == nil {
					target = cw
				}
				return tree.Continue
			})
			if assert.NotNil(t, target) {
				target.AsWidget().Send(events.Click)
			}
		}, func() {
			assert.True(t, eventually(func() bool { return strings.Contains(pageTexts(b, pg), "Results") }), pageTexts(b, pg))
		})
		assert.Equal(t, "POST true", posted, "scripts %v", scripts)
		assert.Equal(t, srv.URL+"/search", pg.currentURL())
		srv.Close()
	}
}

// TestScriptsStyleLinkLoad tests that a style sheet link that a script
// inserts fires its load event (or error event, if it can not be gotten),
// as craigslist waits for it before it shows its pages, and that the
// style sheet is applied.
func TestScriptsStyleLinkLoad(t *testing.T) {
	b, pg := newScriptPage(t, map[string]string{
		"/": `<html><body><p id="out" class="hidden">waiting</p>
			<script>
				const p = document.getElementById("out");
				const ok = document.createElement("link");
				ok.rel = "stylesheet";
				ok.href = "/style.css";
				ok.onload = () => { p.textContent = "loaded"; p.className = ""; };
				document.head.appendChild(ok);
				const bad = document.createElement("link");
				bad.rel = "stylesheet";
				bad.href = "/missing.css";
				bad.addEventListener("error", () => { p.textContent += " and failed"; });
				document.head.appendChild(bad);
			</script></body></html>`,
		"/style.css": `.hidden { display: none }`,
	})
	b.RenderImage(func() { settle(pg) }, func() {
		assert.True(t, eventually(func() bool { return strings.Contains(pageTexts(b, pg), "loaded and failed") }), pageTexts(b, pg))
	})
}

// TestScriptsResize tests that the scripts get a resize event when the
// viewport has another width than they got, as craigslist switches
// between its mobile and desktop layouts by them.
func TestScriptsResize(t *testing.T) {
	b, pg := newScriptPage(t, map[string]string{
		"/": `<html><body><p id="out">none</p><script>
			const start = innerWidth;
			addEventListener("resize", () => { document.getElementById("out").textContent = start + " to " + innerWidth; });
		</script></body></html>`,
	})
	b.RenderImage(func() { settle(pg) }, func() {
		b.AsyncLock()
		ps := pg.scripts
		b.AsyncUnlock()
		ps.engine.Post(func() {
			ps.width = 500 // as for a narrower window (see Page.readMedia)
			ps.engine.Changed()
		})
		assert.True(t, eventually(func() bool { return strings.Contains(pageTexts(b, pg), "790 to 500") }), pageTexts(b, pg))
	})
}
