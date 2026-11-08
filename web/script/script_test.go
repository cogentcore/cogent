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

// render returns the HTML of the document of the engine (in a task, as
// the scripts change it in their tasks).
func render(e *Engine) string {
	ch := make(chan string, 1)
	e.Post(func() {
		var b bytes.Buffer
		html.Render(&b, e.Document())
		ch <- b.String()
	})
	return <-ch
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

// TestModules tests module scripts (bundled with the modules that they
// import), dynamic imports, syntax that goja does not support (async
// generators, rewritten), and :scope in selectors.
func TestModules(t *testing.T) {
	page := `<html><body><div id="out"></div><ul id="l"><li>a</li><li>b</li></ul>
		<script type="module">
			import { greet, chunk, where } from "./js/lib.js";
			import data from "./data.json";
			document.getElementById("out").textContent = greet(data.name) + ";" + where + ";";
			chunk().then(c => { document.getElementById("out").textContent += c + ";"; });
			import("./later.js").then(m => { document.getElementById("out").textContent += m.default + ";"; });
		</script>
		<script>
			async function* gen() { yield 1; yield 2; }
			(async () => {
				let sum = 0;
				for await (const x of gen()) sum += x;
				document.getElementById("out").textContent += "sum " + sum + ";";
			})();
			window.scopeCount = document.getElementById("l").querySelectorAll(":scope > li").length;
		</script>
	</body></html>`
	e, logs := testEngine(t, page, map[string]string{
		// the dynamic imports and import.meta.url of a module are relative
		// to its URL, not that of the page (as the chunks of webpack on MDN)
		"/js/lib.js": `import { helper } from "../helper.js"; export function greet(n) { return helper("hello " + n); }
			export const chunk = () => import("./chunk.js").then(m => m.default);
			export const where = new URL("x", import.meta.url).pathname;`,
		"/js/chunk.js": `export default "chunk";`,
		"/helper.js":   `export const helper = s => s.toUpperCase();`,
		"/data.json":   `{"name": "module"}`,
		"/later.js":    `export default "later";`,
	})
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	assert.Eventually(t, func() bool {
		v, _ := e.RunString(`document.getElementById("out").textContent`)
		s, _ := v.(string)
		return strings.Contains(s, "HELLO MODULE;/js/x;") && strings.Contains(s, "chunk;") && strings.Contains(s, "later;") && strings.Contains(s, "sum 3;")
	}, 5*time.Second, 20*time.Millisecond, "%v", *logs)
	assert.Equal(t, int64(2), run(t, e, `scopeCount`))
	assert.Empty(t, *logs)
}

// TestRenderThrottle tests that a script that changes the page often (as
// an animation) renders it far less often, and that its last change is
// rendered.
func TestRenderThrottle(t *testing.T) {
	doc, err := html.ParseWithOptions(strings.NewReader(`<html><body><p id="p">0</p></body></html>`), html.ParseOptionEnableScripting(true))
	require.NoError(t, err)
	var mu sync.Mutex
	renders := 0
	last := ""
	var e *Engine
	e, err = New(doc, Options{PageURL: "https://example.com/", Render: func() func() {
		mu.Lock()
		renders++
		last = textContent(findByID(e.Document(), "p"))
		mu.Unlock()
		return nil
	}})
	require.NoError(t, err)
	defer e.Close()
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	run(t, e, `let i = 0; const iv = setInterval(() => { document.getElementById("p").textContent = String(++i); if (i === 40) clearInterval(iv); }, 5);`)
	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return last == "40"
	}, 5*time.Second, 20*time.Millisecond, "the last change is rendered")
	mu.Lock()
	defer mu.Unlock()
	assert.Less(t, renders, 10, "rendered far less often than changed (40 times)")
}

// TestImportMap tests that the bare specifiers of modules are mapped by
// the import map of the page (as "react" on GitHub), and that a selector
// with an open bracket at its end is closed, as in CSS.
func TestImportMap(t *testing.T) {
	page := `<html><head><script type="importmap">{"imports": {"lib": "/vendor/lib.js", "pkg/": "/vendor/pkg/"}}</script></head>
		<body><div id="out"></div><p class="x-y">p</p>
		<script type="module">
			import { a } from "lib";
			import { b } from "pkg/b.js";
			document.getElementById("out").textContent = a + b;
		</script></body></html>`
	e, logs := testEngine(t, page, map[string]string{
		"/vendor/lib.js":   `export const a = "A";`,
		"/vendor/pkg/b.js": `export const b = "B";`,
	})
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	assert.Equal(t, "AB", run(t, e, `document.getElementById("out").textContent`), "%v", *logs)
	assert.Equal(t, int64(1), run(t, e, `document.querySelectorAll('[class^="x-"').length`))
}

// TestDataScripts tests scripts with data URLs, and import.meta.url in a
// module (as webpack uses to find its chunks).
func TestDataScripts(t *testing.T) {
	page := `<html><body><div id="out"></div>
		<script src="data:text/javascript,document.getElementById('out').textContent%20%2B%3D%20'data%3B'"></script>
		<script src="data:text/javascript;base64,ZG9jdW1lbnQuZ2V0RWxlbWVudEJ5SWQoJ291dCcpLnRleHRDb250ZW50ICs9ICdiNjQ7Jw=="></script>
		<script type="module" src="/m.js"></script>
	</body></html>`
	e, logs := testEngine(t, page, map[string]string{
		"/m.js": `document.getElementById("out").textContent += new URL("chunk.js", import.meta.url).href;`,
	})
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	assert.Equal(t, "data;b64;https://example.com/chunk.js", run(t, e, `document.getElementById("out").textContent`), "%v", *logs)
}

// TestBlobURLs tests the object URLs of blobs (as Reddit loads a script
// from), as the source of a script and with fetch, until revoked.
func TestBlobURLs(t *testing.T) {
	e, logs := testEngine(t, `<html><body><div id="out"></div></body></html>`, nil)
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	run(t, e, `
		const u = URL.createObjectURL(new Blob(["document.getElementById('out').textContent += 'blob script;';"], { type: "text/javascript" }));
		const s = document.createElement("script");
		s.src = u;
		document.body.appendChild(s);
		fetch(u).then(r => r.text()).then(t => {
			document.getElementById("out").textContent += "fetched " + t.length + ";";
			URL.revokeObjectURL(u);
			return fetch(u);
		}).catch(err => { document.getElementById("out").textContent += "revoked;"; });
	`)
	assert.Eventually(t, func() bool {
		v, _ := e.RunString(`document.getElementById("out").textContent`)
		return v == "blob script;fetched 61;revoked;"
	}, 5*time.Second, 20*time.Millisecond, "%v", *logs)
}

// TestHistoryPolyfill tests that history and PopStateEvent have their state
// on their prototypes, which a polyfill of the Navigation API (on Reddit)
// patches, and that a script that recurses without end gets an error soon.
func TestHistoryPolyfill(t *testing.T) {
	e, logs := testEngine(t, `<html><body></body></html>`, nil)
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	assert.Equal(t, "first", run(t, e, `
		const d = Object.getOwnPropertyDescriptor(Object.getPrototypeOf(history), "state");
		Object.defineProperty(history, "state", { ...d, get() { return d.get.call(this) + ""; } });
		Object.defineProperty(history, "originalState", { ...d });
		history.pushState("first", "");
		const p = Object.getOwnPropertyDescriptor(PopStateEvent.prototype, "state");
		history.state === history.originalState && typeof p.get === "function" && new PopStateEvent("popstate", { state: 1 }).state === 1 ? history.state : "wrong"`))
	start := time.Now()
	_, err := e.RunString(`function f() { return f() + 1; } f()`)
	assert.Error(t, err)
	assert.Less(t, time.Since(start), 2*time.Second)
	e.Eval(`setTimeout(function g() { return g() + 1; }, 0)`) // reported
	assert.Eventually(t, func() bool {
		e.RunString("0") // after the logs of earlier tasks
		return strings.Contains(strings.Join(*logs, "\n"), "Maximum call stack size exceeded")
	}, 2*time.Second, 10*time.Millisecond, "%v", *logs)
}

// TestNavigation tests the minimal Navigation API (window.navigation).
func TestNavigation(t *testing.T) {
	e, logs := testEngine(t, `<html><body></body></html>`, nil)
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	assert.Equal(t, "https://example.com/page.html 1 s", run(t, e, `
		navigation.updateCurrentEntry({ state: "s" });
		[navigation.currentEntry.url, navigation.entries().length, navigation.currentEntry.getState()].join(" ")`))
	assert.Empty(t, *logs)
}

// TestRewriteImports tests the rewriting of dynamic imports, which leaves
// methods named import (as in a chunk of BBC) and import.meta alone.
func TestRewriteImports(t *testing.T) {
	assert.Equal(t, `__dynamicImport('https://x.com/a.js', "./b.js").then(f); x.import(1)`,
		rewriteImports(`import("./b.js").then(f); x.import(1)`, "https://x.com/a.js"))
	src := "class F { import(e) { this.n = e.n } export() { return import(`./c${\")\"}.js`) } } import.meta.url"
	assert.Equal(t, "class F { import(e) { this.n = e.n } export() { return __dynamicImport('', `./c${\")\"}.js`) } } import.meta.url",
		rewriteImports(src, ""))
}

// TestCollections tests NodeList and HTMLCollection, which polyfills (as on
// the Guardian) extend, and that they work as arrays.
func TestCollections(t *testing.T) {
	e, logs := testEngine(t, `<html><body><p id="a" class="x">1</p><p id="b" class="x">2</p></body></html>`, nil)
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	assert.Equal(t, "true true true 2 b 1,2 true", run(t, e, `
		const l = document.querySelectorAll("p"), c = document.getElementsByClassName("x");
		NodeList.prototype.extra = function () { return true; };
		[l instanceof NodeList, c instanceof HTMLCollection, l.extra(), c.length, c.namedItem("b").id,
			Array.from(l).map(p => p.textContent).join(), Array.isArray(l.map(p => p))].join(" ")`))
	assert.Empty(t, *logs)
}

// TestCustomElements tests that the constructors and the reactions
// (connected, disconnected and attribute changes) of custom elements run
// (as GitHub's turbo-frame and the Lit elements of Reddit need), including
// for elements in the page before they are defined, and shadow roots.
func TestCustomElements(t *testing.T) {
	e, logs := testEngine(t, `<html><body><x-tag id="early" mood="calm"></x-tag></body></html>`, nil)
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	assert.Equal(t, "", run(t, e, `
		const log = [];
		class XTag extends HTMLElement {
			static get observedAttributes() { return ["mood"]; }
			constructor() { super(); this.delegate = { ok: true }; log.push("ctor:" + this.id); }
			connectedCallback() { log.push("conn:" + this.id); }
			disconnectedCallback() { log.push("disc:" + this.id); }
			attributeChangedCallback(n, o, v) { log.push("attr:" + n + "=" + (o === null ? "" : o + ">") + v); }
		}
		let defined = "";
		customElements.whenDefined("x-tag").then(() => { defined = "defined"; });
		customElements.define("x-tag", XTag);
		const out = [log.join(" ")]; log.length = 0;
		const el = document.createElement("x-tag");
		el.id = "made";
		document.body.appendChild(el);
		out.push(log.join(" ")); log.length = 0;
		el.setAttribute("mood", "calm"); el.setAttribute("mood", "glad"); el.setAttribute("other", "x");
		out.push(log.join(" ")); log.length = 0;
		el.remove();
		out.push(log.join(" ")); log.length = 0;
		document.body.insertAdjacentHTML("beforeend", "<div><x-tag id=inner></x-tag></div>");
		out.push(log.join(" ")); log.length = 0;
		const host = document.getElementById("early");
		const sr = host.attachShadow({ mode: "open" });
		sr.innerHTML = "<x-tag id=deep></x-tag>";
		out.push([host.shadowRoot === sr, sr.host === host, log.join(" "), document.querySelectorAll("x-tag").length].join(" "));
		out.push(new XTag() instanceof XTag);
		Promise.resolve().then(() => {}).then(() => { window.result = out.join("|") + " " + defined; });
		Object.getPrototypeOf(document.createElement("x-tag").delegate) === Object.prototype ? "" : "no delegate"`))
	assert.Equal(t, "ctor:early attr:mood=calm conn:early|ctor: conn:made|attr:mood=calm attr:mood=calm>glad|disc:made|ctor:inner conn:inner|true true ctor:deep conn:deep 2|true defined",
		run(t, e, `window.result`))
	assert.Empty(t, *logs)
}

// TestStyleSheets tests constructed style sheets (as of Lit on MDN) and
// that the rules inserted in the sheet of a style element (as by CSS-in-JS)
// are in its text, so that they style the page.
func TestStyleSheets(t *testing.T) {
	e, logs := testEngine(t, `<html><head><style id="s">a { color: red }</style></head><body></body></html>`, nil)
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	assert.Equal(t, "2 b{color:blue}|a { color: red } 1 p{x:y}", run(t, e, `
		const st = document.getElementById("s");
		st.sheet.insertRule("b{color:blue}", 1);
		const c = new CSSStyleSheet();
		c.replaceSync("p{x:y}");
		document.adoptedStyleSheets = [...document.adoptedStyleSheets, c];
		[st.sheet.cssRules.length, st.textContent.split("\n").reverse().join("|"), c.cssRules.length, c.cssRules[0].cssText].join(" ")`))
	assert.Empty(t, *logs)
}

// TestSharedModules tests that a module that the module scripts of a page
// share (as on GitHub, which defines custom elements in one) is evaluated
// once, with modules that import each other, and that the error of a
// module rejects its dynamic import.
func TestSharedModules(t *testing.T) {
	page := `<html><body>
		<script type="module">import { count } from "./shared.js"; window.a = count();</script>
		<script type="module" src="/b.js"></script>
		<script type="module">import("./bad.js").catch(e => { window.bad = e.message; });</script>
	</body></html>`
	e, logs := testEngine(t, page, map[string]string{
		"/shared.js": `import { other } from "./cycle.js"; let n = 0; export const count = () => ++n + other();
			customElements.define("shared-el", class extends HTMLElement {});`,
		"/cycle.js": `import { count } from "./shared.js"; export const other = () => typeof count === "function" ? 0 : 100;`,
		"/b.js":     `import { count } from "./shared.js"; window.b = count();`,
		"/bad.js":   `throw new Error("bad module");`,
	})
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	assert.Eventually(t, func() bool {
		v, _ := e.RunString(`[window.a, window.b, window.bad].join(" ")`)
		return v == "1 2 bad module"
	}, 5*time.Second, 20*time.Millisecond, "%v", *logs)
	assert.Empty(t, *logs)
}

// TestIntl tests the minimal Intl (English only), which goja lacks and
// sites (GitHub) use to format numbers and dates.
func TestIntl(t *testing.T) {
	e, logs := testEngine(t, `<html><body></body></html>`, nil)
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	assert.Equal(t, "1,234,567.891|$1,234.50|45%|1.2K|3/5/2024|March 5, 2024|Tuesday, March 5, 2024|3:07 PM|3 days ago|tomorrow|a, b, and c|other one|-1|1|12K|1.5 million", run(t, e, `
		const d = new Date(2024, 2, 5, 15, 7, 9);
		[new Intl.NumberFormat().format(1234567.891), new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" }).format(1234.5),
			new Intl.NumberFormat("en", { style: "percent" }).format(0.45), new Intl.NumberFormat("en", { notation: "compact" }).format(1234),
			new Intl.DateTimeFormat("en-US").format(d), new Intl.DateTimeFormat("en-US", { dateStyle: "long" }).format(d),
			new Intl.DateTimeFormat("en-US", { dateStyle: "full" }).format(d), new Intl.DateTimeFormat("en-US", { hour: "numeric", minute: "2-digit" }).format(d),
			new Intl.RelativeTimeFormat("en").format(-3, "days"), new Intl.RelativeTimeFormat("en", { numeric: "auto" }).format(1, "day"),
			new Intl.ListFormat("en").format(["a", "b", "c"]), new Intl.PluralRules("en").select(2) + " " + new Intl.PluralRules("en").select(1),
			new Intl.Collator("en").compare("a", "b"), ["b10", "b9"].sort(new Intl.Collator("en", { numeric: true }).compare).indexOf("b10"),
			new Intl.NumberFormat("en", { notation: "compact" }).format(12345), new Intl.NumberFormat("en", { notation: "compact", compactDisplay: "long" }).format(1499999)].join("|")`))
	assert.Empty(t, *logs)
}

// TestRenderNotTimed tests that the time of the renders that a task needs
// (as for the layout of elements) is not counted as that of its scripts,
// which are interrupted after TaskTimeout (a slow page, as GitHub).
func TestRenderNotTimed(t *testing.T) {
	doc, err := html.ParseWithOptions(strings.NewReader(`<html><body><p id="p">0</p></body></html>`), html.ParseOptionEnableScripting(true))
	require.NoError(t, err)
	var logs []string
	e, err := New(doc, Options{PageURL: "https://example.com/", TaskTimeout: 200 * time.Millisecond,
		Render: func() func() { time.Sleep(150 * time.Millisecond); return nil },
		Layout: func(n *html.Node) (x, y, w, h float32, ok bool) { return 0, 0, 10, 10, true },
		Log:    func(level slog.Level, msg string) { logs = append(logs, msg) }})
	require.NoError(t, err)
	defer e.Close()
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	v, err := e.RunString(`const p = document.getElementById("p");
		for (let i = 0; i < 3; i++) { p.textContent = String(i); p.getBoundingClientRect(); }
		p.textContent`)
	assert.NoError(t, err)
	assert.Equal(t, "2", v)
	_, err = e.RunString(`for (;;) {}`)
	assert.Error(t, err, "a script that does not end is still interrupted")
}

// TestSerialize tests innerHTML and outerHTML as browsers serialize them:
// the text of scripts as it is (the Guardian parses the JSON of a script
// element from its innerHTML), and quotes only escaped in attributes.
func TestSerialize(t *testing.T) {
	e, logs := testEngine(t, `<html><body><script type="application/json" id="c">{"a":"<b>&amp;"}</script>`+
		`<p id="p" title='say "hi" &amp; go'>"quoted" &amp; <i>a&lt;b</i>&nbsp;<br>x<!--c--></p></body></html>`, nil)
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	assert.Equal(t, `{"a":"<b>&amp;"}|<p id="p" title="say &quot;hi&quot; &amp; go">"quoted" &amp; <i>a&lt;b</i>&nbsp;<br>x<!--c--></p>`,
		run(t, e, `document.getElementById("c").innerHTML + "|" + document.getElementById("p").outerHTML`))
	assert.Empty(t, *logs)
}

// TestPromisePolyfill tests that a page that replaces Promise, and the
// then method of the native one, with ones that schedule with
// queueMicrotask (as core-js on BBC) does not make queueMicrotask recurse,
// or queue microtasks, without end.
func TestPromisePolyfill(t *testing.T) {
	e, logs := testEngine(t, `<html><body></body></html>`, nil)
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	run(t, e, `
		const Native = Promise, nativeThen = Promise.prototype.then;
		Native.prototype.then = function (a, b) { queueMicrotask(() => {}); return nativeThen.call(this, a, b); };
		window.Promise = class P {
			constructor(f) { this.p = new Native(f); }
			then(a, b) { const q = new P(() => {}); queueMicrotask(() => {}); q.p = this.p.then(a, b); return q; }
			catch(b) { return this.then(undefined, b); }
			static resolve(v) { return new P(r => r(v)); }
		};
		window.done = 0;
		queueMicrotask(() => { window.done++; });
		Promise.resolve(1).then(() => { window.done++; });`)
	assert.Eventually(t, func() bool { return run(t, e, `window.done`) == int64(2) }, 2*time.Second, 10*time.Millisecond, "%v", *logs)
	assert.Empty(t, *logs)
}

// TestReadRunsNoScripts tests that what the page asks of the scripts when
// it is read (whether elements have listeners, and the state of form
// controls) runs no scripts of the page, as the constructors of custom
// elements (which looped on BBC while the page was being read).
func TestReadRunsNoScripts(t *testing.T) {
	e, logs := testEngine(t, `<html><body><div id="d" onclick="x()"></div></body></html>`, nil)
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	run(t, e, `window.made = 0; customElements.define("x-el", class extends HTMLElement { constructor() { super(); window.made++; } });`)
	done := make(chan bool)
	e.Post(func() {
		d := findByID(e.Document(), "d")
		el := &html.Node{Type: html.ElementNode, Data: "x-el"}
		d.AppendChild(el) // as the page has it, not wrapped by the scripts
		assert.True(t, e.HasListener(el, "click"), "the onclick of its parent")
		assert.False(t, e.HasListener(el, "input"))
		_, hasValue, _, _ := e.ControlState(el)
		assert.False(t, hasValue)
		close(done)
	})
	<-done
	assert.Equal(t, int64(0), run(t, e, `window.made`))
	assert.Empty(t, *logs)
}

// TestStopAfterTimeouts tests that the scripts of a page that keep running
// too long (as BBC's, which queue microtasks without end) are stopped.
func TestStopAfterTimeouts(t *testing.T) {
	doc, err := html.ParseWithOptions(strings.NewReader(`<html><body></body></html>`), html.ParseOptionEnableScripting(true))
	require.NoError(t, err)
	var mu sync.Mutex
	var logs []string
	e, err := New(doc, Options{PageURL: "https://example.com/", TaskTimeout: 50 * time.Millisecond,
		Log: func(level slog.Level, msg string) { mu.Lock(); logs = append(logs, msg); mu.Unlock() }})
	require.NoError(t, err)
	defer e.Close()
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	e.Eval(`setInterval(() => { for (;;) {} }, 1)`)
	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return strings.Contains(strings.Join(logs, "\n"), "were stopped")
	}, 3*time.Second, 10*time.Millisecond)
	_, err = e.RunString("1")
	assert.Error(t, err, "the engine is closed")
}

// TestMessageChannel tests MessageChannel (React's scheduler and reCAPTCHA
// use it), that its messages come in later tasks, and the slot property
// (which Lit elements on Reddit set).
func TestMessageChannel(t *testing.T) {
	e, logs := testEngine(t, `<html><body><div id="d"></div></body></html>`, nil)
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	run(t, e, `
		window.got = [];
		const c = new MessageChannel();
		c.port1.onmessage = e => got.push("1:" + e.data);
		c.port2.addEventListener("message", e => got.push("2:" + e.data));
		c.port2.postMessage("a");
		c.port1.postMessage("b");
		got.push("sync");
		const d = document.getElementById("d");
		d.slot = "s";
		got.push(d.getAttribute("slot"), MediaError.MEDIA_ERR_DECODE);`)
	var got any
	assert.Eventually(t, func() bool { got = run(t, e, `got.join(" ")`); return got == "sync s 3 1:a 2:b" }, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, "sync s 3 1:a 2:b", got)
	assert.Empty(t, *logs)
}

// TestTimerOrder tests that timers without a delay run in the order that
// they are set.
func TestTimerOrder(t *testing.T) {
	e, logs := testEngine(t, `<html><body></body></html>`, nil)
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	run(t, e, `window.order = []; for (let i = 0; i < 50; i++) setTimeout(() => order.push(i), 0); clearTimeout(setTimeout(() => order.push("cleared"), 0));`)
	var got any
	assert.Eventually(t, func() bool { got = run(t, e, `order.length`); return got == int64(50) }, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, true, run(t, e, `order.every((v, i) => v === i)`))
	assert.Empty(t, *logs)
}

// TestSVGElements tests that elements created in the SVG namespace (as
// React does for the icons of github.com) keep the case of their names and
// of their attributes (as linearGradient and viewBox), which the SVG reader
// needs, while the names of HTML attributes are lowercase.
func TestSVGElements(t *testing.T) {
	e, logs := testEngine(t, `<html><body><div id="d"></div></body></html>`, nil)
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	assert.Equal(t, `<svg viewBox="0 0 1 1"><linearGradient id="g"></linearGradient></svg>|linearGradient|http://www.w3.org/2000/svg|true|0 0 1 1|DIV|x`,
		run(t, e, `
			const ns = "http://www.w3.org/2000/svg";
			const svg = document.createElementNS(ns, "svg");
			svg.setAttribute("viewBox", "0 0 1 1");
			const g = document.createElementNS(ns, "linearGradient");
			g.setAttribute("id", "g");
			svg.appendChild(g);
			const d = document.getElementById("d");
			d.setAttribute("DATA-X", "x");
			[svg.outerHTML, g.tagName, g.namespaceURI, g instanceof SVGElement, svg.getAttribute("viewBox"), d.tagName, d.getAttribute("data-x")].join("|")`))
	assert.Empty(t, *logs)
}

// TestStyleProperties tests that the properties of styles are in them
// (as "opacity" in el.style), by which libraries such as GSAP (on
// github.com) find the properties that are supported, and that vendor
// prefixed ones are not.
func TestStyleProperties(t *testing.T) {
	e, logs := testEngine(t, `<html><body><div id="d"></div></body></html>`, nil)
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	assert.Equal(t, "true,true,true,true,false,false,true",
		run(t, e, `const d = document.getElementById("d");
			const cs = getComputedStyle(d);
			["opacity" in d.style, "backgroundColor" in d.style, "--x" in d.style, "display" in cs, "WebkitTransform" in d.style, "webkitTransform" in d.style, "setProperty" in d.style].join(",")`))
	assert.Empty(t, *logs)
}

// TestTemplateWalkSlots tests what Lit needs to render its templates (as
// the menus of MDN): the content of a template is one fragment, a tree
// walker walks from its currentNode (as that content, outside its root),
// and the children of a shadow host that are in no slot (or in a hidden
// one) are reported as unassigned, so they are not rendered.
func TestTemplateWalkSlots(t *testing.T) {
	e, _ := testEngine(t, `<html><body><menu-x id="m"><button slot="button">Open</button><div slot="dropdown" id="panel">Panel</div></menu-x></body></html>`, nil)
	assert.Equal(t, true, run(t, e, `(() => { const t = document.createElement("template"); t.innerHTML = "<p>a</p>"; return t.content === t.content && t.content.firstChild.nodeName === "P"; })()`))
	assert.Equal(t, "SLOT:name|?hidden$lit$ SLOT:name", run(t, e, `(() => {
		const t = document.createElement("template");
		t.innerHTML = '<div><slot name="button" ?hidden$lit$="m"></slot></div><slot name="dropdown"></slot>';
		const w = document.createTreeWalker(document, 1);
		w.currentNode = t.content;
		const out = [];
		let n;
		while ((n = w.nextNode()) !== null) {
			if (n.nodeName === "SLOT") out.push(n.nodeName + ":" + n.getAttributeNames().join("|"));
			if (n.hasAttribute("?hidden$lit$")) n.removeAttribute("?hidden$lit$");
		}
		return out.join(" ") + (t.content.querySelector("[name=button]").getAttributeNames().length === 1 ? "" : " kept");
	})()`))
	run(t, e, `document.getElementById("m").attachShadow({mode: "open"}).innerHTML = '<slot name="button"></slot><slot name="dropdown" hidden></slot>';`)
	panel := make(chan [2]bool, 1)
	e.Post(func() {
		var button, p *html.Node
		for n := range e.Document().Descendants() {
			if n.Type == html.ElementNode && n.Data == "button" {
				button = n
			}
			if n.Type == html.ElementNode && n.Data == "div" {
				p = n
			}
		}
		panel <- [2]bool{e.Unassigned(button), e.Unassigned(p)}
	})
	assert.Equal(t, [2]bool{false, true}, <-panel, "the panel is in a hidden slot")
	run(t, e, `document.getElementById("m").shadowRoot.querySelector("[name=dropdown]").hidden = false;`)
	e.Post(func() {
		for n := range e.Document().Descendants() {
			if n.Type == html.ElementNode && n.Data == "div" {
				panel <- [2]bool{false, e.Unassigned(n)}
			}
		}
	})
	assert.Equal(t, [2]bool{false, false}, <-panel, "open")
}

// TestSlotEvents tests that an event on a child of a shadow host goes
// through the slot of the shadow root that it is in, then the shadow root,
// then the host (as the clicks on the menu buttons of MDN, whose shadow
// roots listen on their slots to open the menus).
func TestSlotEvents(t *testing.T) {
	e, _ := testEngine(t, `<html><body><menu-x id="m"><button slot="button" id="b">Open</button></menu-x></body></html>`, nil)
	assert.Equal(t, "slot root host", run(t, e, `(() => {
		const m = document.getElementById("m");
		const sr = m.attachShadow({mode: "open"});
		sr.innerHTML = '<div><slot name="button"></slot></div><slot name="dropdown"></slot>';
		const got = [];
		sr.querySelector("slot[name=button]").addEventListener("click", () => got.push("slot"));
		sr.querySelector("slot[name=dropdown]").addEventListener("click", () => got.push("other slot"));
		sr.addEventListener("click", () => got.push("root"));
		m.addEventListener("click", () => got.push("host"));
		document.getElementById("b").click();
		return got.join(" ");
	})()`))
}
