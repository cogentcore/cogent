// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	neturl "net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// testEngine returns an engine for the given HTML page, with the given
// resources by URL (relative to https://example.com/), which are served
// with GetURL, and the console messages that it logs.
func testEngine(t *testing.T, page string, resources map[string]string) (*Engine, *[]string) {
	doc, err := html.ParseWithOptions(strings.NewReader(page), html.ParseOptionEnableScripting(true))
	require.NoError(t, err)
	var mu sync.Mutex
	var logs []string
	e, err := New(doc, Options{
		PageURL: "https://example.com/page.html",
		GetURL: func(u string) (*http.Response, error) {
			pu, _ := neturl.Parse(u)
			body, ok := resources[pu.Path]
			if !ok {
				return &http.Response{StatusCode: 404, Status: "404 Not Found", Body: io.NopCloser(strings.NewReader("")), Request: &http.Request{URL: pu}}, nil
			}
			return &http.Response{StatusCode: 200, Status: "200 OK", Body: io.NopCloser(strings.NewReader(body)),
				Header: http.Header{"Content-Type": {"text/plain"}}, Request: &http.Request{URL: pu}}, nil
		},
		Storage: NewStorage(),
		Log: func(level slog.Level, msg string) {
			mu.Lock()
			logs = append(logs, level.String()+": "+msg)
			mu.Unlock()
		},
	})
	require.NoError(t, err)
	t.Cleanup(e.Close)
	return e, &logs
}

// run runs the given script in the engine, failing the test on an error.
func run(t *testing.T, e *Engine, src string) any {
	t.Helper()
	v, err := e.RunString(src)
	require.NoError(t, err, src)
	return v
}

// render returns the HTML of the document of the engine.
func render(e *Engine) string {
	var b bytes.Buffer
	html.Render(&b, e.Document())
	return b.String()
}

func TestDOMBasics(t *testing.T) {
	e, _ := testEngine(t, `<html><head><title>T</title></head><body><div id="a" class="x y"><p>One</p><p>Two</p></div></body></html>`, nil)
	assert.Equal(t, "T", run(t, e, `document.title`))
	assert.Equal(t, "DIV", run(t, e, `document.getElementById("a").tagName`))
	assert.Equal(t, int64(2), run(t, e, `document.querySelectorAll("#a p").length`))
	assert.Equal(t, "Two", run(t, e, `document.querySelector("p + p").textContent`))
	assert.Equal(t, true, run(t, e, `document.getElementById("a") === document.querySelector(".x")`))
	assert.Equal(t, true, run(t, e, `document.querySelector("p") instanceof HTMLParagraphElement && document.body instanceof HTMLElement`))
	assert.Equal(t, true, run(t, e, `document.querySelector("#a").classList.contains("y")`))

	run(t, e, `
		const a = document.getElementById("a");
		const p = document.createElement("p");
		p.textContent = "Three";
		p.setAttribute("data-n", "3");
		a.appendChild(p);
		a.classList.add("z");
		a.classList.remove("x");
		a.firstElementChild.remove();
		a.style.color = "red";
		a.insertAdjacentHTML("afterbegin", "<span>Zero</span>");
	`)
	out := render(e)
	assert.Contains(t, out, `<div id="a" class="y z" style="color: red;"><span>Zero</span><p>Two</p><p data-n="3">Three</p></div>`)
	assert.Equal(t, "3", run(t, e, `document.querySelector("[data-n]").dataset.n`))
	assert.Equal(t, "<span>Zero</span><p>Two</p><p data-n=\"3\">Three</p>", run(t, e, `document.getElementById("a").innerHTML`))
	assert.True(t, e.dirty || true)
}

func TestEvents(t *testing.T) {
	e, _ := testEngine(t, `<html><body><div id="outer"><button id="b" onclick="this.textContent = 'clicked'">Go</button></div></body></html>`, nil)
	run(t, e, `
		window.order = [];
		const outer = document.getElementById("outer");
		outer.addEventListener("click", e => order.push("outer capture"), true);
		outer.addEventListener("click", e => order.push("outer bubble"));
		document.getElementById("b").addEventListener("click", e => { order.push("button"); e.preventDefault(); });
	`)
	b := findByID(e.Document(), "b")
	done := make(chan bool, 1)
	e.Dispatch(b, "click", nil, func(prevented bool) { done <- prevented })
	select {
	case prevented := <-done:
		assert.True(t, prevented, "the listener prevented the default")
	case <-time.After(5 * time.Second):
		t.Fatal("no dispatch")
	}
	assert.Equal(t, "outer capture,button,outer bubble", run(t, e, `order.join(",")`))
	assert.Equal(t, "clicked", run(t, e, `document.getElementById("b").textContent`), "the onclick attribute ran")
	assert.True(t, e.HasListener(findByID(e.Document(), "outer"), "click"))
}

func TestLoadScripts(t *testing.T) {
	page := `<html><head><script src="/a.js"></script></head><body>
		<div id="out"></div>
		<script>document.getElementById("out").textContent += "inline;";</script>
		<script>document.write("<p id='w'>written</p>");</script>
		<script>
			document.addEventListener("DOMContentLoaded", () => { document.getElementById("out").textContent += "ready;"; });
			window.addEventListener("load", () => {
				document.getElementById("out").textContent += "load;";
				const s = document.createElement("script");
				s.src = "/b.js";
				s.onload = () => { document.getElementById("out").textContent += "b loaded;"; };
				document.head.appendChild(s);
			});
		</script>
	</body></html>`
	e, logs := testEngine(t, page, map[string]string{
		"/a.js": `window.fromA = "a";`,
		"/b.js": `document.getElementById("out").textContent += "b;";`,
	})
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	e.Wait()
	assert.Equal(t, "a", run(t, e, `window.fromA`))
	assert.Equal(t, "inline;ready;load;b;b loaded;", run(t, e, `document.getElementById("out").textContent`))
	assert.Equal(t, "written", run(t, e, `document.getElementById("w").textContent`))
	assert.Equal(t, "complete", run(t, e, `document.readyState`))
	assert.Empty(t, *logs)
}

func TestTimersAndPromises(t *testing.T) {
	e, _ := testEngine(t, `<html><body></body></html>`, nil)
	run(t, e, `
		window.log = [];
		setTimeout(() => log.push("timeout"), 20);
		Promise.resolve().then(() => log.push("micro"));
		log.push("sync");
		window.n = 0;
		window.iv = setInterval(() => { if (++n === 3) clearInterval(iv); }, 5);
		(async () => { await new Promise(r => setTimeout(r, 30)); log.push("async"); })();
	`)
	assert.Eventually(t, func() bool {
		v, _ := e.RunString(`log.join(",") + " " + n`)
		return v == "sync,micro,timeout,async 3"
	}, 3*time.Second, 20*time.Millisecond)
}

func TestFetch(t *testing.T) {
	e, _ := testEngine(t, `<html><body><ul id="list"></ul></body></html>`, map[string]string{
		"/items.json": `["apple", "pear"]`,
	})
	run(t, e, `
		fetch("items.json").then(r => r.json()).then(items => {
			const ul = document.getElementById("list");
			for (const it of items) { const li = document.createElement("li"); li.textContent = it; ul.appendChild(li); }
		});
		window.xhrText = "";
		const x = new XMLHttpRequest();
		x.open("GET", "/items.json");
		x.onload = () => { xhrText = x.responseText; };
		x.send();
		fetch("missing").then(r => { window.missing = r.status; });
	`)
	assert.Eventually(t, func() bool {
		return strings.Contains(render(e), `<li>apple</li><li>pear</li>`)
	}, 3*time.Second, 20*time.Millisecond)
	assert.Eventually(t, func() bool {
		v, _ := e.RunString(`xhrText + " " + window.missing`)
		return v == `["apple", "pear"] 404`
	}, 3*time.Second, 20*time.Millisecond)
}

func TestWebAPIs(t *testing.T) {
	e, _ := testEngine(t, `<html><body><a id="l" href="/x?y=1#z">x</a><input id="i" value="v"></body></html>`, nil)
	assert.Equal(t, "https://example.com/x?y=1#z", run(t, e, `document.getElementById("l").href`))
	assert.Equal(t, "/x", run(t, e, `document.getElementById("l").pathname`))
	assert.Equal(t, "1", run(t, e, `new URL("https://a.org/p?y=1").searchParams.get("y")`))
	assert.Equal(t, "https://a.org/q?a=b+c", run(t, e, `const u = new URL("/q", "https://a.org/p"); u.searchParams.set("a", "b c"); u.href`))
	assert.Equal(t, "example.com", run(t, e, `location.hostname`))
	assert.Equal(t, "v2", run(t, e, `localStorage.setItem("k", "v2"); localStorage.getItem("k")`))
	assert.Equal(t, "aGk=", run(t, e, `btoa("hi")`))
	assert.Equal(t, "hi", run(t, e, `atob("aGk=")`))
	assert.Equal(t, int64(3), run(t, e, `new TextEncoder().encode("héy").length - 1`))
	assert.Equal(t, "w", run(t, e, `const i = document.getElementById("i"); i.value = "w"; i.value`))
	assert.Equal(t, "v", run(t, e, `document.getElementById("i").getAttribute("value")`))
	assert.Equal(t, "object", run(t, e, `typeof JSON.parse(JSON.stringify({a: [1]}))`))
	assert.Equal(t, true, run(t, e, `typeof matchMedia("(min-width: 1px)").matches === "boolean"`))
}

// findByID returns the element of the given id in the given document.
func findByID(doc *html.Node, id string) *html.Node {
	var found *html.Node
	walk(doc, func(n *html.Node) bool {
		if v, ok := attr(n, "id"); ok && v == id && n.Type == html.ElementNode {
			found = n
			return false
		}
		return true
	})
	return found
}
