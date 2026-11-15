// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"io"
	"log/slog"
	"net/http"
	neturl "net/url"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// fakePage is the page of an engine, as the web package has, with a lock
// of its widgets, renders, restyles and rereads that it counts, boxes for
// the elements with ids, and the other options of a page.
type fakePage struct {
	mu sync.Mutex

	// gone is whether the page is gone, which its lock reports.
	gone atomic.Bool

	// locks, renders, restyles, and rereads count the calls.
	locks, renders, restyles, rereads atomic.Int32

	// restyleOK and rereadOK are what the commits of restyles and
	// rereads return.
	restyleOK, rereadOK atomic.Bool

	// renderSleep is how long a render takes.
	renderSleep time.Duration

	// navigated has the URLs that were navigated to.
	navMu     sync.Mutex
	navigated []string
	pushed    []string

	logMu sync.Mutex
	logs  []string
}

// testPageEngine returns an engine for the given HTML page with a fake
// page (see [fakePage]), whose options can be changed by the given
// functions (as [traced]).
func testPageEngine(t *testing.T, page string, resources map[string]string, opts ...func(o *Options)) (*Engine, *fakePage) {
	t.Helper()
	doc, err := html.ParseWithOptions(strings.NewReader(page), html.ParseOptionEnableScripting(true))
	require.NoError(t, err)
	fp := &fakePage{}
	fp.restyleOK.Store(true)
	fp.rereadOK.Store(true)
	o := Options{
		PageURL: "https://example.com/page.html",
		GetURL: func(u string) (*http.Response, error) {
			pu, _ := neturl.Parse(u)
			body, ok := resources[pu.Path]
			if !ok {
				return &http.Response{StatusCode: 404, Status: "404 Not Found", Body: io.NopCloser(strings.NewReader("")), Request: &http.Request{URL: pu}}, nil
			}
			ct := "text/plain"
			switch {
			case strings.HasSuffix(pu.Path, ".css"):
				ct = "text/css"
			case strings.HasSuffix(pu.Path, ".js"), strings.HasSuffix(pu.Path, ".mjs"):
				ct = "text/javascript"
			}
			return &http.Response{StatusCode: 200, Status: "200 OK", Body: io.NopCloser(strings.NewReader(body)),
				Header: http.Header{"Content-Type": {ct}}, Request: &http.Request{URL: pu}}, nil
		},
		Storage: NewStorage(),
		Lock: func() bool {
			fp.locks.Add(1)
			fp.mu.Lock()
			if fp.gone.Load() {
				fp.mu.Unlock()
				return false
			}
			return true
		},
		Unlock: func() { fp.mu.Unlock() },
		Render: func() func() {
			fp.renders.Add(1)
			time.Sleep(fp.renderSleep)
			return func() {}
		},
		Restyle: func(nodes []*html.Node) func() bool {
			fp.restyles.Add(1)
			return func() bool { return fp.restyleOK.Load() }
		},
		Reread: func(self, children, styled []*html.Node) func() bool {
			fp.rereads.Add(1)
			return func() bool { return fp.rereadOK.Load() }
		},
		Layout: func(n *html.Node) (x, y, w, h float32, ok bool) {
			if id, _ := attr(n, "id"); id != "" {
				return 10, 20, 30, 40, true
			}
			return 0, 0, 0, 0, false
		},
		ElementAt: func(x, y float32) *html.Node {
			if x < 0 {
				return nil
			}
			return findByID(doc, "a")
		},
		ComputedStyle: func(n *html.Node, prop string) string {
			if prop == "color" {
				return "rgb(1, 2, 3)"
			}
			return ""
		},
		Viewport:     func() (w, h float32) { return 800, 600 },
		MediaMatches: func(query string) bool { return strings.Contains(query, "min-width") },
		Navigate: func(url string, replace bool) {
			fp.navMu.Lock()
			fp.navigated = append(fp.navigated, url)
			fp.navMu.Unlock()
		},
		PushState: func(url string, replace bool) {
			fp.navMu.Lock()
			fp.pushed = append(fp.pushed, url)
			fp.navMu.Unlock()
		},
		Log: func(level slog.Level, msg string) {
			fp.logMu.Lock()
			fp.logs = append(fp.logs, level.String()+": "+msg)
			fp.logMu.Unlock()
		},
	}
	for _, f := range opts {
		f(&o)
	}
	e, err := New(doc, o)
	require.NoError(t, err)
	t.Cleanup(e.Close)
	return e, fp
}

// traced makes the engine trace what the scripts do (which profiles them,
// see [Engine.watchActivity]).
func traced(o *Options) { o.Trace = func() bool { return true } }

// logged returns whether the page logged a message containing s.
func (fp *fakePage) logged(s string) bool {
	fp.logMu.Lock()
	defer fp.logMu.Unlock()
	return slices.ContainsFunc(fp.logs, func(l string) bool { return strings.Contains(l, s) })
}

// allLogs returns the messages that the page logged.
func (fp *fakePage) allLogs() string {
	fp.logMu.Lock()
	defer fp.logMu.Unlock()
	return strings.Join(fp.logs, "\n")
}

// TestPageLayout tests the functions of the scripts that use the layout
// of the page: the boxes of elements, the element at a point, computed
// styles, the viewport, media queries, and focus.
func TestPageLayout(t *testing.T) {
	e, fp := testPageEngine(t, `<html><body><div id="a">A</div><p>no id</p></body></html>`, nil)
	assert.Equal(t, []any{int64(10), int64(20), int64(30), int64(40)}, run(t, e, `
		const r = document.getElementById("a").getBoundingClientRect();
		[r.x, r.y, r.width, r.height]`))
	assert.Equal(t, int64(0), run(t, e, `document.querySelector("p").getBoundingClientRect().width`), "no box")
	assert.Equal(t, "a", run(t, e, `document.elementFromPoint(1, 1).id`))
	assert.Nil(t, run(t, e, `document.elementFromPoint(-1, 1)`))
	assert.Equal(t, "rgb(1, 2, 3)", run(t, e, `getComputedStyle(document.getElementById("a")).color`))
	assert.Equal(t, int64(800), run(t, e, `innerWidth`))
	assert.Equal(t, int64(600), run(t, e, `innerHeight`))
	assert.Equal(t, true, run(t, e, `matchMedia("(min-width: 10px)").matches`))
	assert.Equal(t, false, run(t, e, `matchMedia("(max-width: 10px)").matches`))

	// a change of the document is rendered for a box (in the task)
	r0 := fp.renders.Load()
	assert.Equal(t, int64(30), run(t, e, `
		const d = document.createElement("div");
		d.id = "b";
		document.body.appendChild(d);
		d.getBoundingClientRect().width`))
	e.Wait()
	assert.Greater(t, fp.renders.Load()+fp.rereads.Load(), r0)

	run(t, e, `document.getElementById("a").focus()`)
	e.Wait()
	ch := make(chan *html.Node, 1)
	e.Post(func() { ch <- e.TakeFocus() })
	n := <-ch
	if n != nil { // (unless a render took it)
		assert.Equal(t, "a", htmlAttr(n, "id"))
	}
	assert.Greater(t, fp.locks.Load(), int32(0))
}

// TestPageRenders tests the kinds of renders of the page after tasks:
// restyles for changes of styles, rereads for changes of elements, and
// whole renders, when the others fail, and for events of the app.
func TestPageRenders(t *testing.T) {
	e, fp := testPageEngine(t, `<html><body><div id="a" class="x">A</div><ul id="l"></ul></body></html>`, nil)
	waitFor := func(what string, cond func() bool) {
		t.Helper()
		require.Eventually(t, cond, 5*time.Second, 5*time.Millisecond, what)
	}
	r := fp.restyles.Load()
	run(t, e, `document.getElementById("a").style.color = "red"`)
	waitFor("restyled", func() bool { return fp.restyles.Load() > r })

	fp.restyleOK.Store(false) // as for a change of display: reread
	rr := fp.rereads.Load()
	run(t, e, `document.getElementById("a").className = "y"`)
	waitFor("reread", func() bool { return fp.rereads.Load() > rr })

	fp.rereadOK.Store(false) // rendered whole
	rd := fp.renders.Load()
	run(t, e, `document.getElementById("l").appendChild(document.createElement("li"))`)
	waitFor("rendered", func() bool { return fp.renders.Load() > rd })

	// many changes are rendered whole
	rd = fp.renders.Load()
	run(t, e, `for (let i = 0; i < 2000; i++) document.getElementById("l").appendChild(document.createElement("li"))`)
	waitFor("rendered whole", func() bool { return fp.renders.Load() > rd })

	rd = fp.renders.Load()
	e.Rerender()
	waitFor("rerendered", func() bool { return fp.renders.Load() > rd })

	rd = fp.renders.Load()
	e.Post(e.Changed)
	waitFor("changed", func() bool { return fp.renders.Load() > rd })

	fp.restyleOK.Store(true)
	r = fp.restyles.Load()
	e.Post(func() { e.StateChanged(findByID(e.Document(), "a")) })
	waitFor("state changed", func() bool { return fp.restyles.Load() > r })

	run(t, e, `window.resizes = 0; window.scrolls = 0;
		addEventListener("resize", () => resizes++);
		addEventListener("scroll", () => scrolls++)`)
	e.Resized()
	e.Scrolled()
	e.Scrolled() // merged with the one before, if it has not run
	e.Wait()
	assert.Equal(t, int64(1), run(t, e, `resizes`))
	assert.GreaterOrEqual(t, run(t, e, `scrolls`), int64(1))
}

// TestPageForcedRender tests a render that a script forces for the box of
// an element in its task, which releases the lock while the page is read,
// and the lock that is held for the uses that follow each other, released
// when idle.
func TestPageForcedRender(t *testing.T) {
	e, fp := testPageEngine(t, `<html><body><div id="a">A</div></body></html>`, nil, traced, func(o *Options) {
		o.Restyle, o.Reread = nil, nil
	})
	fp.renderSleep = 60 * time.Millisecond // slow: traced
	rd := fp.renders.Load()
	assert.Equal(t, int64(30), run(t, e, `
		const a = document.getElementById("a");
		a.getBoundingClientRect(); // takes the lock
		a.textContent = "B";
		a.getBoundingClientRect().width // renders, in the task`))
	assert.Greater(t, fp.renders.Load(), rd)
	// uses of the lock that follow each other, for longer than lockYield
	run(t, e, `const end = Date.now() + 120; while (Date.now() < end) document.getElementById("a").getBoundingClientRect()`)
	// a lock that is idle is released, while the task goes on
	run(t, e, `document.getElementById("a").getBoundingClientRect(); const end2 = Date.now() + 60; while (Date.now() < end2) {}`)
	e.Wait()
	assert.True(t, fp.logged("render of the page took"), fp.allLogs())
}

// TestPageGone tests that the engine closes when its page is gone (when
// its lock fails).
func TestPageGone(t *testing.T) {
	e, fp := testPageEngine(t, `<html><body><div id="a">A</div></body></html>`, nil)
	fp.gone.Store(true)
	_, _ = e.RunString(`document.getElementById("a").getBoundingClientRect()`)
	require.Eventually(t, e.Closed, 5*time.Second, 5*time.Millisecond)
}

// TestSetLock tests setting the lock of the page after the engine is made,
// and the URL of the page, and whether a task runs.
func TestSetLock(t *testing.T) {
	e, fp := testPageEngine(t, `<html><body><div id="a">A</div></body></html>`, nil, func(o *Options) {
		o.Lock, o.Unlock = nil, nil
	})
	var mu sync.Mutex
	var locks atomic.Int32
	e.SetLock(func() bool { locks.Add(1); mu.Lock(); return true }, mu.Unlock)
	run(t, e, `document.getElementById("a").getBoundingClientRect()`)
	assert.Greater(t, locks.Load(), int32(0))
	ch := make(chan []any, 1)
	e.Post(func() { ch <- []any{e.PageURL(), e.InTask(), e.Stack(5)} })
	got := <-ch
	assert.Equal(t, "https://example.com/page.html", got[0])
	assert.Equal(t, true, got[1])
	assert.Equal(t, "", got[2], "no script runs")
	assert.False(t, (func() bool { return e.InTask() })(), "not outside of tasks")
	_ = fp
}

// TestNavigateStack tests the stack of the script that navigates the page,
// for the console.
func TestNavigateStack(t *testing.T) {
	e, fp := testPageEngine(t, `<html><body></body></html>`, nil)
	run(t, e, `function go() { location.href = "/next.html" } go()`)
	require.Eventually(t, func() bool { // (in a goroutine)
		fp.navMu.Lock()
		defer fp.navMu.Unlock()
		return slices.Equal([]string{"https://example.com/next.html"}, fp.navigated)
	}, 5*time.Second, 5*time.Millisecond)
	assert.Contains(t, e.NavigateStack(), "go")
	run(t, e, `history.pushState({}, "", "/pushed.html")`)
	require.Eventually(t, func() bool {
		fp.navMu.Lock()
		defer fp.navMu.Unlock()
		return slices.Equal([]string{"https://example.com/pushed.html"}, fp.pushed)
	}, 5*time.Second, 5*time.Millisecond)
}

// TestComposedTree tests that the page is rendered as its composed tree:
// the content of a shadow root in place of the children of its host, with
// the children of the host in the slots that they are assigned to, and the
// document as it was after the render; and the scopes and composed parents
// of the nodes of the last render.
func TestComposedTree(t *testing.T) {
	var mu sync.Mutex
	var rendered string
	var e *Engine
	e, _ = testPageEngine(t, `<html><body><x-a id="a"><b slot="s" id="b">B</b><i id="i">I</i><!-- c --></x-a></body></html>`, nil, func(o *Options) {
		o.Restyle, o.Reread = nil, nil
		o.Render = func() func() {
			var sb strings.Builder
			html.Render(&sb, findByID(e.Document(), "a"))
			mu.Lock()
			rendered = sb.String()
			mu.Unlock()
			return nil
		}
	})
	run(t, e, `
		const r = document.getElementById("a").attachShadow({ mode: "open" });
		r.innerHTML = '<p id="p">[<slot name="s">fallback</slot>]</p><slot></slot><slot name="none">none</slot>';`)
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return rendered != ""
	}, 5*time.Second, 5*time.Millisecond)
	mu.Lock()
	assert.Equal(t, `<x-a id="a"><p id="p">[<slot name="s"><b id="b" slot="s">B</b></slot>]</p><slot><i id="i">I</i></slot><slot name="none">none</slot></x-a>`, rendered)
	mu.Unlock()
	assert.Equal(t, `<x-a id="a"><b id="b" slot="s">B</b><i id="i">I</i><!-- c --></x-a>`, run(t, e, `document.getElementById("a").outerHTML`), "the document after")

	ch := make(chan []*html.Node, 1)
	e.Post(func() {
		a, b, p := findByID(e.Document(), "a"), findByID(e.Document(), "b"), e.shadowRoots[findByID(e.Document(), "a")].FirstChild
		ch <- []*html.Node{e.ShadowScope(p), e.ShadowScope(b), e.ComposedParent(p), e.ComposedParent(b), e.ComposedParent(a), a}
	})
	got := <-ch
	a := got[5]
	assert.Equal(t, a, got[0], "the shadow tree of a")
	assert.Nil(t, got[1], "in the document")
	assert.Equal(t, a, got[2], "the host")
	assert.Equal(t, "slot", got[3].Data, "the slot")
	assert.Equal(t, a.Parent, got[4], "the parent in the document")
}

// TestClassicScriptModules tests that a classic script with the syntax of
// modules (imports) is run as a bundled module, with the modules, JSON,
// and CSS that it imports, and import.meta.url; that one with syntax that
// goja does not support is lowered; and that the errors of bundling are
// reported.
func TestClassicScriptModules(t *testing.T) {
	e, fp := testPageEngine(t, `<html><body>
		<script>
			import { x } from "./m.js";
			import data from "./d.json";
			import "./s.css";
			window.got = [x, data.n, import.meta.url].join("|");
		</script>
		<script>
			async function* gen() { yield 1; yield 2 }
			(async () => { let s = 0; for await (const v of gen()) s += v; window.sum = s })();
		</script>
		<script>import { y } from "./missing.js"; window.bad = 1;</script>
	</body></html>`, map[string]string{
		"/m.js":   `export const x = "mod " + new URL(import.meta.url).pathname;`,
		"/d.json": `{"n": 7}`,
		"/s.css":  `p { color: red }`,
	})
	e.Load(nil)
	e.Wait()
	require.Eventually(t, func() bool {
		v, _ := e.RunString(`window.got + " " + window.sum`)
		return v == "mod /m.js|7|https://example.com/page.html 3"
	}, 5*time.Second, 10*time.Millisecond, "%v", func() any { v, _ := e.RunString(`window.got + " " + window.sum`); return v }())
	assert.Equal(t, "undefined", run(t, e, `typeof window.bad`))
	assert.True(t, fp.logged("SyntaxError") || fp.logged("missing"), fp.allLogs())

	_, err := lower("let = = 1", "bad.js")
	assert.Error(t, err)
	_, err = e.bundleModule(`import { a } from "./none.js"; import "./none2.js"; import "./none3.js"; import "./none4.js";`, "x.js")
	assert.Error(t, err)
}
