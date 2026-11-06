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
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles"
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
		v, err := pg.scripts.engine.RunString(`const r = document.getElementById("box").getBoundingClientRect(); [Math.round(r.width), Math.round(r.height), innerWidth > 0].join(",")`)
		assert.NoError(t, err)
		assert.Equal(t, "300,50,true", v)
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
