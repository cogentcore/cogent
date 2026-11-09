// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"image"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestScriptsComputedStyle tests that the computed styles of scripts have
// the values of the style sheets of the page (as the text-align that
// github.com copies to the lines of its heading), as browsers report them.
func TestScriptsComputedStyle(t *testing.T) {
	b, pg := newScriptPage(t, map[string]string{
		"/": `<html><head><style>:root{--accent: #0f0} h1{text-align:center;color:#0000ff;font-size:32px} .faded{color:rgba(255,0,0,0.5)}</style></head>` +
			`<body><h1 id="h">Heading <span id="s">x</span></h1><p id="p" class="faded">p</p></body></html>`,
	})
	b.RenderImage(func() { settle(pg) }, func() {
		v, err := pg.scripts.engine.RunString(`const h = getComputedStyle(document.getElementById("h")), s = getComputedStyle(document.getElementById("s")), p = getComputedStyle(document.getElementById("p"));
			[h.textAlign, h.color, s.textAlign, s.color, h.fontSize, s.getPropertyValue("--accent"), p.color, p.textAlign, p.backgroundColor].join(";")`)
		assert.NoError(t, err)
		assert.Equal(t, "center;rgb(0, 0, 255);center;rgb(0, 0, 255);32px;#0f0;rgba(255, 0, 0, 0.5);;", v)
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

// TestScriptsClickInLink tests that a click on a box in a link (as the
// results of craigslist), which gets the click as the page has a click
// listener, opens the link unless a listener prevents its default, and
// that it opens it after the scripts are stopped.
func TestScriptsClickInLink(t *testing.T) {
	for _, mode := range []string{"open", "prevented", "stopped"} {
		t.Run(mode, func(t *testing.T) {
			prevent := ""
			if mode == "prevented" {
				prevent = "e.preventDefault();"
			}
			b, pg := newScriptPage(t, map[string]string{
				"/": `<html><body><a href="/item"><div id="card" style="height: 40px">card</div></a>
					<script>document.addEventListener("click", e => { ` + prevent + ` });</script></body></html>`,
				"/item": `<html><body><p>the item</p></body></html>`,
			})
			b.RenderImage(func() { settle(pg) }, func() {
				if mode == "stopped" {
					b.AsyncLock()
					ps := pg.scripts
					b.AsyncUnlock()
					ps.engine.Close()
				}
				b.AsyncLock()
				var card core.Widget
				pg.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
					if cwb.Property("id") == "card" {
						card = cw
					}
					return tree.Continue
				})
				if assert.NotNil(t, card) {
					card.AsWidget().Send(events.Click)
				}
				b.AsyncUnlock()
			}, func() {
				if mode == "prevented" {
					settle(pg)
					assert.Equal(t, "https://example.com/", pg.currentURL())
					return
				}
				assert.True(t, eventually(func() bool { return strings.Contains(pageTexts(b, pg), "the item") }), pageTexts(b, pg))
			})
		})
	}
}

// TestScriptsCheckedMenu tests a menu that CSS shows when a hidden
// checkbox is checked (as the menus of Wikipedia): checking it shows the
// menu (the page is read again for the :checked selector), and a click on
// text outside of it goes to a listener of the document, which unchecks it.
func TestScriptsCheckedMenu(t *testing.T) {
	b, pg := newScriptPage(t, map[string]string{
		"/": `<html><head><style>.menu { display: none } .cb:checked ~ .menu { display: block }</style></head><body>
			<div><input type="checkbox" class="cb" id="cb"><div class="menu"><p>Menu item</p></div></div>
			<p>Outside text</p>
			<script>
				document.addEventListener("click", e => {
					if (e.target.id !== "cb") document.getElementById("cb").checked = false;
				});
			</script></body></html>`,
	})
	// send sends a click to the first widget that the given function selects
	send := func(sel func(w core.Widget) bool) {
		b.AsyncLock()
		defer b.AsyncUnlock()
		var target core.Widget
		pg.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			if target == nil && sel(cw) {
				target = cw
			}
			return tree.Continue
		})
		if assert.NotNil(t, target) {
			target.AsWidget().Send(events.Click)
		}
	}
	b.RenderImage(func() { settle(pg) }, func() {
		assert.NotContains(t, pageTexts(b, pg), "Menu item")
		send(func(w core.Widget) bool { _, ok := w.(*core.Switch); return ok })
	}, func() {
		assert.True(t, eventually(func() bool { return strings.Contains(pageTexts(b, pg), "Menu item") }), pageTexts(b, pg))
		send(func(w core.Widget) bool { tx, ok := w.(*core.Text); return ok && strings.Contains(tx.Text, "Outside") })
	}, func() {
		assert.True(t, eventually(func() bool { return !strings.Contains(pageTexts(b, pg), "Menu item") }), pageTexts(b, pg))
	})
}

// TestScriptsTransparentControlHover tests that the label after a
// transparent checkbox shows its hover in a page whose document has a
// click listener (as the menu buttons of Wikipedia): the state layer of
// the label, which is clickable for the listener, is kept when hovered.
func TestScriptsTransparentControlHover(t *testing.T) {
	b, pg := newScriptPage(t, map[string]string{
		"/": `<html><body><div style="position: relative"><input type="checkbox" id="cb" style="opacity: 0; position: absolute">` +
			`<label for="cb" id="lb" style="display: inline-block">Menu</label></div>` +
			`<script>document.addEventListener("click", e => {});</script></body></html>`,
	})
	b.RenderImage(func() { settle(pg) }, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		var sw *core.Switch
		var label *core.WidgetBase
		pg.WalkDown(func(n tree.Node) bool {
			if s, ok := n.(*core.Switch); ok {
				sw = s
			}
			if wb := core.AsWidget(n); wb != nil && wb.Property("id") == "lb" && label == nil {
				label = wb
			}
			return tree.Continue
		})
		if !assert.NotNil(t, sw) || !assert.NotNil(t, label) {
			return
		}
		sw.Send(events.MouseEnter)
		label.Style()
		assert.Greater(t, label.Styles.StateLayer, float32(0), "the hover of the label")
	})
}

// TestScriptsSummaryTextClick tests that a click on the text of a summary
// (in a span, which gets the click, on a page with a click listener, as
// the sidebar of MDN) opens its details, in the document.
func TestScriptsSummaryTextClick(t *testing.T) {
	b, pg := newScriptPage(t, map[string]string{
		"/": `<html><body><details id="d"><summary><span id="s">Modules</span></summary><p>Anchor positioning</p></details>` +
			`<script>document.addEventListener("click", e => {});</script></body></html>`,
	})
	var text *core.Text
	b.RenderImage(func() { settle(pg) }, func() {
		b.AsyncLock()
		pg.WalkDown(func(n tree.Node) bool {
			if tx, ok := n.(*core.Text); ok && strings.Contains(tx.Text, "Modules") {
				text = tx
			}
			return tree.Continue
		})
		b.AsyncUnlock()
		if !assert.NotNil(t, text) {
			return
		}
		bb := text.Geom.TotalBBox
		c := bb.Min.Add(bb.Size().Div(2))
		ev := b.Scene.RenderWindow().SystemWindow.Events()
		ev.MouseButton(events.MouseDown, events.Left, c, 0)
		ev.MouseButton(events.MouseUp, events.Left, c, 0)
	}, func() {
		settle(pg)
		assert.Eventually(t, func() bool {
			v, _ := pg.scripts.engine.RunString(`document.getElementById("d").open`)
			return v == true
		}, 3*time.Second, 20*time.Millisecond, "opened")
	})
}

// TestScriptsSummaryClickStopped tests that a click on the text of a
// summary opens its details in place when the scripts were stopped (as
// after timeouts), as the document is not read again then.
func TestScriptsSummaryClickStopped(t *testing.T) {
	b, pg := newScriptPage(t, map[string]string{
		"/": `<html><body><details id="d"><summary><span id="s">Modules</span></summary><p>Anchor positioning</p></details>` +
			`<script>document.addEventListener("click", e => {});</script></body></html>`,
	})
	find := func(s string) *core.Text {
		var tx *core.Text
		pg.WalkDown(func(n tree.Node) bool {
			if t, ok := n.(*core.Text); ok && strings.Contains(t.Text, s) {
				tx = t
			}
			return tree.Continue
		})
		return tx
	}
	shown := func(tx *core.Text) bool {
		for w := tx.AsWidget(); w != nil; w = core.AsWidget(w.Parent) {
			if w.Styles.Display == styles.DisplayNone {
				return false
			}
		}
		return true
	}
	b.RenderImage(func() { settle(pg) }, func() {
		pg.scripts.engine.Close()
		b.AsyncLock()
		text := find("Modules")
		b.AsyncUnlock()
		if !assert.NotNil(t, text) {
			return
		}
		bb := text.Geom.TotalBBox
		c := bb.Min.Add(bb.Size().Div(2))
		ev := b.Scene.RenderWindow().SystemWindow.Events()
		ev.MouseButton(events.MouseDown, events.Left, c, 0)
		ev.MouseButton(events.MouseUp, events.Left, c, 0)
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		if p := find("Anchor positioning"); assert.NotNil(t, p) {
			assert.True(t, shown(p), "opened in place")
		}
	})
}

// TestScriptsScrollKept tests that an element that scrolls keeps its
// scroll position when the page is read again after a change by the
// scripts (as the sidebar of MDN, which scrolled to its top when an item
// in it opened).
func TestScriptsScrollKept(t *testing.T) {
	items := strings.Repeat(`<p style="height: 40px">item</p>`, 30)
	b, pg := newScriptPage(t, map[string]string{
		"/": `<html><body><div id="side" style="height: 200px; overflow: auto">` + items +
			`<details id="d"><summary><span>Modules</span></summary><p>Anchor positioning</p></details>` + items + `</div>` +
			`<script>document.addEventListener("click", e => {});</script></body></html>`,
	})
	side := func() *core.Frame {
		var fr *core.Frame
		pg.WalkDown(func(n tree.Node) bool {
			if f := core.AsFrame(n); f != nil && f.Property("id") == "side" {
				fr = f
			}
			return tree.Continue
		})
		return fr
	}
	b.RenderImage(func() { settle(pg) }, func() {
		b.AsyncLock()
		fr := side()
		if assert.NotNil(t, fr) && assert.True(t, fr.HasScroll[math32.Y]) {
			fr.Geom.Scroll.Y = -600
			fr.ScrollUpdateFromGeom(math32.Y)
		}
		b.AsyncUnlock()
		pg.scripts.engine.RunString(`document.getElementById("d").open = true`)
	}, func() {
		settle(pg)
		time.Sleep(300 * time.Millisecond)
	}, func() {
		v, _ := pg.scripts.engine.RunString(`document.getElementById("d").open`)
		assert.Equal(t, true, v)
		b.AsyncLock()
		defer b.AsyncUnlock()
		if fr := side(); assert.NotNil(t, fr) {
			assert.InDelta(t, -600, fr.Geom.Scroll.Y, 2, "kept its scroll")
		}
	})
}

// TestScriptsHoverEvents tests that the scripts get the events of the
// mouse moving over the elements (as the onPointerEnter of React, with
// which the repository grid of github.com reveals its panels).
func TestScriptsHoverEvents(t *testing.T) {
	b, pg := newScriptPage(t, map[string]string{
		"/": `<html><body><div id="a" style="height: 50px">a</div><div id="b" style="height: 50px">b</div>` +
			`<script>window.got = []; for (const id of ["a", "b"]) { const el = document.getElementById(id);` +
			` el.addEventListener("pointerenter", () => got.push("enter " + id)); el.addEventListener("pointerleave", () => got.push("leave " + id)); }</script></body></html>`,
	})
	center := func(id string) image.Point {
		var p image.Point
		pg.WalkDown(func(n tree.Node) bool {
			if wb := core.AsWidget(n); wb != nil && wb.Property("id") == id {
				bb := wb.Geom.TotalBBox
				p = bb.Min.Add(bb.Size().Div(2))
			}
			return tree.Continue
		})
		return p
	}
	b.RenderImage(func() { settle(pg) }, func() {
		b.AsyncLock()
		a, bp := center("a"), center("b")
		b.AsyncUnlock()
		ev := b.Scene.RenderWindow().SystemWindow.Events()
		ev.MouseMove(a)
		time.Sleep(50 * time.Millisecond)
		ev.MouseMove(bp)
	}, func() {
		assert.Eventually(t, func() bool {
			v, _ := pg.scripts.engine.RunString(`got.join(", ")`)
			return v == "enter a, leave a, enter b"
		}, 3*time.Second, 20*time.Millisecond)
	})
}

// TestScriptsHoverMenu tests a submenu that shows on the :hover of its menu
// item (as the header menus of pkg.go.dev): the page is read again when the
// mouse moves over the item, and again when it moves out.
func TestScriptsHoverMenu(t *testing.T) {
	b, pg := newScriptPage(t, map[string]string{
		"/": `<html><head><style>.sub { display: none } .item:hover > .label ~ .sub { display: block }</style></head><body>` +
			`<div class="item" id="item" style="height: 50px"><span class="label">Why Go</span><div class="sub"><p>Case studies</p></div></div>` +
			`<div id="other" style="height: 200px">other</div><script>window.x = 1</script></body></html>`,
	})
	find := func(id, text string) (image.Point, bool) {
		var p image.Point
		shown := false
		pg.WalkDown(func(n tree.Node) bool {
			if wb := core.AsWidget(n); wb != nil && id != "" && wb.Property("id") == id {
				bb := wb.Geom.TotalBBox
				p = bb.Min.Add(image.Pt(5, 5))
			}
			if tx, ok := n.(*core.Text); ok && tx.Text == text {
				shown = true
			}
			return tree.Continue
		})
		return p, shown
	}
	shownNow := func() bool {
		b.AsyncLock()
		defer b.AsyncUnlock()
		_, shown := find("", "Case studies")
		return shown
	}
	b.RenderImage(func() { settle(pg) }, func() {
		b.AsyncLock()
		item, shown := find("item", "Case studies")
		b.AsyncUnlock()
		assert.False(t, shown, "hidden before the hover")
		ev := b.Scene.RenderWindow().SystemWindow.Events()
		ev.MouseMove(item)
		assert.Eventually(t, shownNow, 3*time.Second, 20*time.Millisecond, "shown on the hover")
		b.AsyncLock()
		other, _ := find("other", "") // below the submenu that is shown
		b.AsyncUnlock()
		ev.MouseMove(other)
		assert.Eventually(t, func() bool { return !shownNow() }, 3*time.Second, 20*time.Millisecond, "hidden after")
	}, func() {})
}

// TestScriptsFocusAndKeys tests a text input that a script focuses (as the
// input of the jump dialog of pkg.go.dev): its text field gets the focus,
// and the keys typed in it are dispatched to the scripts, keydown before
// it gets them and keyup after (with its value), on which pages filter.
func TestScriptsFocusAndKeys(t *testing.T) {
	b, pg := newScriptPage(t, map[string]string{
		"/": `<html><body><input id="in" type="text"><script>window.keys = [];
			const i = document.getElementById("in");
			i.addEventListener("keydown", e => keys.push("down " + e.key + " " + e.keyCode));
			i.addEventListener("keyup", e => keys.push("up " + i.value));
			setTimeout(() => i.focus(), 0);</script></body></html>`,
	})
	var tf *core.TextField
	b.RenderImage(func() { settle(pg) }, func() {
		b.AsyncLock()
		pg.WalkDown(func(n tree.Node) bool {
			if f, ok := n.(*core.TextField); ok {
				tf = f
			}
			return tree.Continue
		})
		focused := tf != nil && tf.StateIs(states.Focused)
		b.AsyncUnlock()
		require.True(t, focused, "the text field has the focus")
		ev := b.Scene.RenderWindow().SystemWindow.Events()
		ev.KeyChord('h', 0, 0)
		time.Sleep(200 * time.Millisecond) // as typed by a person
		ev.KeyChord('i', 0, 0)
	}, func() {
		var got any
		assert.Eventually(t, func() bool {
			got, _ = pg.scripts.engine.RunString(`keys.join(", ")`)
			return got == "down h 72, up h, down i 73, up hi"
		}, 3*time.Second, 20*time.Millisecond, "the keys")
		t.Logf("got %v", got)
	})
}
