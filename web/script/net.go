// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	neturl "net/url"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/dop251/goja"
	"golang.org/x/net/html"
)

// Storage has the localStorage of pages, by origin, shared by the pages
// of an app.
type Storage struct {
	mu   sync.Mutex
	data map[string]map[string]string
}

// NewStorage returns a new empty [Storage].
func NewStorage() *Storage {
	return &Storage{data: map[string]map[string]string{}}
}

// area returns the storage of the given origin, making it if needed.
func (s *Storage) area(origin string) map[string]string {
	a := s.data[origin]
	if a == nil {
		a = map[string]string{}
		s.data[origin] = a
	}
	return a
}

// origin returns the origin of the given URL (scheme://host:port).
func origin(u string) string {
	pu, err := neturl.Parse(u)
	if err != nil || pu.Host == "" {
		return "null"
	}
	return pu.Scheme + "://" + pu.Host
}

// baseURL returns the URL that the URLs of the document are relative to:
// that of its base element, or the URL of the page.
func (e *Engine) baseURL() string {
	base := e.opts.PageURL
	walk(e.doc, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "base" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					if u, err := resolveURL(a.Val, base); err == nil {
						base = u
					}
					return false
				}
			}
		}
		return n.Data != "body"
	})
	return base
}

// resolveURL returns the given URL relative to the given base URL.
func resolveURL(u, base string) (string, error) {
	pu, err := neturl.Parse(strings.TrimSpace(u))
	if err != nil {
		return "", err
	}
	if base == "" {
		return pu.String(), nil
	}
	pb, err := neturl.Parse(base)
	if err != nil {
		return "", err
	}
	return pb.ResolveReference(pu).String(), nil
}

// defineNet defines the Go functions of __dom for URLs, the network,
// storage, cookies, and navigation.
func (e *Engine) defineNet() {
	vm := e.vm
	d := e.jsDOM
	set := func(name string, f func(call goja.FunctionCall) goja.Value) {
		d.Set(name, f)
	}
	str := func(call goja.FunctionCall, i int) string { return call.Argument(i).String() }

	set("pageURL", func(call goja.FunctionCall) goja.Value { return vm.ToValue(e.opts.PageURL) })
	set("baseURL", func(call goja.FunctionCall) goja.Value { return vm.ToValue(e.baseURL()) })
	// parseURL returns the parts of the given URL relative to the given
	// base (if not undefined), as for the URL class, or null if invalid.
	set("parseURL", func(call goja.FunctionCall) goja.Value {
		raw := strings.TrimSpace(str(call, 0))
		var pu *neturl.URL
		var err error
		if b := call.Argument(1); !goja.IsUndefined(b) && !goja.IsNull(b) {
			var base *neturl.URL
			base, err = neturl.Parse(b.String())
			if err != nil || base.Scheme == "" {
				return goja.Null()
			}
			var ref *neturl.URL
			ref, err = neturl.Parse(raw)
			if err == nil {
				pu = base.ResolveReference(ref)
			}
		} else {
			pu, err = neturl.Parse(raw)
		}
		if err != nil || pu.Scheme == "" {
			return goja.Null()
		}
		if (pu.Scheme == "http" || pu.Scheme == "https") && pu.Path == "" {
			pu.Path = "/"
		}
		user, pass := "", ""
		if pu.User != nil {
			user = pu.User.Username()
			pass, _ = pu.User.Password()
		}
		search := ""
		if pu.RawQuery != "" {
			search = "?" + pu.RawQuery
		}
		hash := ""
		if pu.Fragment != "" {
			hash = "#" + pu.EscapedFragment()
		}
		org := "null"
		if pu.Host != "" {
			org = pu.Scheme + "://" + pu.Host
		}
		path := pu.EscapedPath()
		if pu.Opaque != "" {
			path = pu.Opaque
		}
		return vm.NewArray(pu.String(), pu.Scheme+":", user, pass, pu.Host, pu.Hostname(), pu.Port(), path, search, hash, org)
	})
	set("navigate", func(call goja.FunctionCall) goja.Value {
		u, err := resolveURL(str(call, 0), e.baseURL())
		if err == nil && e.opts.Navigate != nil {
			replace := call.Argument(1).ToBoolean()
			go e.opts.Navigate(u, replace) // after the task, which holds the lock
		}
		return goja.Undefined()
	})
	set("pushState", func(call goja.FunctionCall) goja.Value {
		u, err := resolveURL(str(call, 0), e.opts.PageURL)
		if err != nil || origin(u) != origin(e.opts.PageURL) {
			panic(vm.NewGoError(&domException{name: "SecurityError", msg: "the URL is not of the origin of the page"}))
		}
		e.opts.PageURL = u
		if e.opts.PushState != nil {
			e.opts.PushState(u, call.Argument(1).ToBoolean())
		}
		return vm.ToValue(u)
	})
	set("storage", func(call goja.FunctionCall) goja.Value {
		op, kind := str(call, 0), str(call, 1)
		st := e.opts.Storage
		if kind == "session" || st == nil {
			if e.session == nil {
				e.session = NewStorage()
			}
			st = e.session
		}
		st.mu.Lock()
		defer st.mu.Unlock()
		a := st.area(origin(e.opts.PageURL))
		switch op {
		case "get":
			if v, ok := a[str(call, 2)]; ok {
				return vm.ToValue(v)
			}
			return goja.Null()
		case "set":
			a[str(call, 2)] = str(call, 3)
		case "remove":
			delete(a, str(call, 2))
		case "clear":
			clear(a)
		case "keys":
			var ks []any
			for k := range a {
				ks = append(ks, k)
			}
			return vm.NewArray(ks...)
		}
		return goja.Undefined()
	})
	set("cookie", func(call goja.FunctionCall) goja.Value {
		u, err := neturl.Parse(e.opts.PageURL)
		if err != nil || e.opts.Cookies == nil {
			return vm.ToValue("")
		}
		var parts []string
		for _, c := range e.opts.Cookies.Cookies(u) {
			parts = append(parts, c.Name+"="+c.Value)
		}
		return vm.ToValue(strings.Join(parts, "; "))
	})
	set("setCookie", func(call goja.FunctionCall) goja.Value {
		u, err := neturl.Parse(e.opts.PageURL)
		if err != nil || e.opts.Cookies == nil {
			return goja.Undefined()
		}
		if c, err := http.ParseSetCookie(str(call, 0)); err == nil {
			c.HttpOnly = false
			e.opts.Cookies.SetCookies(u, []*http.Cookie{c})
		}
		return goja.Undefined()
	})
	// fetch gets the given URL with the given method, headers (an array of
	// names and values) and body (a string or undefined) in another
	// goroutine, and calls the callback of the given id with the status,
	// status text, headers, body (ArrayBuffer), final URL, and error.
	set("fetch", func(call goja.FunctionCall) goja.Value {
		cb := int(call.Argument(0).ToInteger())
		u, err := resolveURL(str(call, 1), e.baseURL())
		method := strings.ToUpper(str(call, 2))
		var headers []string
		if h, ok := call.Argument(3).Export().([]any); ok {
			for _, v := range h {
				headers = append(headers, fmt.Sprint(v))
			}
		}
		var body []byte
		hasBody := false
		if b := call.Argument(4); !goja.IsUndefined(b) && !goja.IsNull(b) {
			hasBody = true
			if ab, ok := b.Export().(goja.ArrayBuffer); ok {
				body = ab.Bytes()
			} else {
				body = []byte(b.String())
			}
		}
		if err != nil {
			e.callBack(cb, 0, "", nil, nil, "", err.Error())
			return goja.Undefined()
		}
		go func() {
			resp, err := e.request(u, method, headers, body, hasBody)
			if err != nil {
				e.callBack(cb, 0, "", nil, nil, "", err.Error())
				return
			}
			defer resp.Body.Close()
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				e.callBack(cb, 0, "", nil, nil, "", err.Error())
				return
			}
			var hs []any
			for k, vs := range resp.Header {
				hs = append(hs, strings.ToLower(k), strings.Join(vs, ", "))
			}
			final := u
			if resp.Request != nil && resp.Request.URL != nil {
				final = resp.Request.URL.String()
			}
			text := http.StatusText(resp.StatusCode)
			e.Post(func() {
				_, err := e.callback(goja.Undefined(), vm.ToValue(cb), vm.ToValue(resp.StatusCode), vm.ToValue(text),
					vm.NewArray(hs...), vm.ToValue(vm.NewArrayBuffer(data)), vm.ToValue(final), goja.Null())
				e.reportError(err)
			})
		}()
		return goja.Undefined()
	})
	set("decodeUTF8", func(call goja.FunctionCall) goja.Value {
		b := bytesOf(call.Argument(0))
		if !utf8.Valid(b) {
			b = bytes.ToValidUTF8(b, []byte("�"))
		}
		return vm.ToValue(string(b))
	})
	set("encodeUTF8", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(vm.NewArrayBuffer([]byte(str(call, 0))))
	})
}

// bytesOf returns the bytes of the given ArrayBuffer or typed array value.
func bytesOf(v goja.Value) []byte {
	switch x := v.Export().(type) {
	case goja.ArrayBuffer:
		return x.Bytes()
	case []byte:
		return x
	}
	if o, ok := v.(*goja.Object); ok {
		if ab, ok := o.Get("buffer").Export().(goja.ArrayBuffer); ok {
			b := ab.Bytes()
			off := int(o.Get("byteOffset").ToInteger())
			n := int(o.Get("byteLength").ToInteger())
			if off >= 0 && off+n <= len(b) {
				return b[off : off+n]
			}
		}
	}
	return []byte(v.String())
}

// request makes the given request: a GET without a body with GetURL, as
// for the resources of the page, and others with the default client.
func (e *Engine) request(u, method string, headers []string, body []byte, hasBody bool) (*http.Response, error) {
	if (method == "" || method == "GET") && !hasBody && e.opts.GetURL != nil {
		return e.opts.GetURL(u)
	}
	req, err := http.NewRequest(method, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Add(headers[i], headers[i+1])
	}
	client := &http.Client{Jar: e.opts.Cookies}
	return client.Do(req)
}

// isScript returns whether the given node is a script element of
// JavaScript (classic or module).
func isScript(n *html.Node) bool {
	if n.Type != html.ElementNode || n.Data != "script" || n.Namespace != "" {
		return false
	}
	typ := ""
	for _, a := range n.Attr {
		if a.Key == "type" {
			typ = strings.ToLower(strings.TrimSpace(a.Val))
		}
	}
	switch typ {
	case "", "text/javascript", "application/javascript", "module", "text/ecmascript", "application/ecmascript":
		return true
	}
	return false
}

// attr returns the value of the given attribute of the given node.
func attr(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

// Load runs the scripts of the document, in tasks: those in it, in order
// (getting the external ones), then the DOMContentLoaded event, then the
// load event, after which it calls done (if not nil) in the task.
func (e *Engine) Load(done func()) {
	var scripts []*html.Node
	walk(e.doc, func(n *html.Node) bool {
		if n.Type == html.ElementNode && (n.Data == "template" || n.Data == "noscript") {
			return true // their content is not scripts (it is text with scripting)
		}
		if isScript(n) {
			scripts = append(scripts, n)
		}
		return true
	})
	e.Post(func() {
		e.loading = true
		if e.ran == nil {
			e.ran = map[*html.Node]bool{}
		}
		for _, s := range scripts {
			e.ran[s] = true // run here, not when inserted
		}
	})
	for _, s := range scripts {
		e.Post(func() { e.runScript(s, true) })
	}
	e.Post(func() {
		e.callPrelude("loaded", "interactive")
	})
	e.Post(func() {
		e.loading = false
		e.callPrelude("loaded", "complete")
		if done != nil {
			done()
		}
		// done reads the document as the scripts have made it (with
		// the lock of the page, which this task does not hold)
		e.dirty = false
	})
}

// callPrelude calls the function of the given name of __dom.
func (e *Engine) callPrelude(name string, args ...any) goja.Value {
	f, ok := goja.AssertFunction(e.jsDOM.Get(name))
	if !ok {
		e.logf(slog.LevelError, "prelude: no __dom.%s", name)
		return goja.Undefined()
	}
	vals := make([]goja.Value, len(args))
	for i, a := range args {
		vals[i] = e.vm.ToValue(a)
	}
	v, err := f(goja.Undefined(), vals...)
	e.reportError(err)
	return v
}

// runScript runs the given script element: its external script (gotten
// in this goroutine, as scripts run in order while the page loads), or
// its text, and fires its load (or error) event.
func (e *Engine) runScript(s *html.Node, sync bool) {
	src, hasSrc := attr(s, "src")
	name := "inline script"
	code := textContent(s)
	if hasSrc {
		u, err := resolveURL(src, e.baseURL())
		if err == nil {
			name = u
			code, err = e.getScript(u)
		}
		if err != nil {
			e.logf(slog.LevelWarn, "script %s: %v", src, err)
			e.callPrelude("fireEvent", e.nodes.id(s), "error")
			return
		}
	}
	if typ, _ := attr(s, "type"); strings.EqualFold(strings.TrimSpace(typ), "module") {
		var err error
		code, err = e.bundleModule(code, name)
		if err != nil {
			e.logf(slog.LevelWarn, "module %s: %v", name, err)
			e.callPrelude("fireEvent", e.nodes.id(s), "error")
			return
		}
	}
	e.current = s
	e.run(name, code)
	e.current = nil
	if hasSrc {
		e.callPrelude("fireEvent", e.nodes.id(s), "load")
	}
}

// getScript gets the script at the given URL.
func (e *Engine) getScript(u string) (string, error) {
	if e.opts.GetURL == nil {
		return "", fmt.Errorf("no GetURL")
	}
	resp, err := e.opts.GetURL(u)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %s", resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// runInserted runs the scripts that the given node (that was inserted in
// the document) is or has, which have not run yet: the external ones once
// they are gotten (in another goroutine), and the others in a new task.
func (e *Engine) runInserted(n *html.Node) {
	if !connected(n, e.doc) {
		return
	}
	walk(n, func(c *html.Node) bool {
		if !isScript(c) || e.ran[c] {
			return true
		}
		if e.ran == nil {
			e.ran = map[*html.Node]bool{}
		}
		e.ran[c] = true
		if _, ok := attr(c, "src"); ok {
			e.Post(func() { e.runScript(c, false) })
			return true
		}
		if strings.TrimSpace(textContent(c)) != "" {
			e.Post(func() { e.runScript(c, false) })
		}
		return true
	})
}

// write writes the given HTML to the document, as document.write: after
// the script that is running while the page loads, or at the end of the
// body otherwise (as a document is not opened again).
func (e *Engine) write(s string) {
	parent := e.current
	var before *html.Node
	if parent != nil {
		before = parent.NextSibling
		parent = parent.Parent
	}
	if parent == nil {
		walk(e.doc, func(n *html.Node) bool {
			if n.Type == html.ElementNode && n.Data == "body" {
				parent = n
				return false
			}
			return true
		})
	}
	if parent == nil {
		return
	}
	for _, k := range e.parseFragment(s, parent) {
		parent.InsertBefore(k, before)
		e.runInserted(k)
	}
	e.changed(parent)
}

// Dispatch dispatches a trusted event of the given type (as "click") to
// the given node in a task, with the given properties of the event (as
// for a MouseEvent), and then calls after (if not nil) in the task with
// whether the default action of the event was prevented (as by
// preventDefault), for the app to do it otherwise (as following a link).
func (e *Engine) Dispatch(n *html.Node, typ string, props map[string]any, after func(prevented bool)) {
	e.Post(func() {
		v, err := e.dispatch(goja.Undefined(), e.vm.ToValue(e.nodes.id(n)), e.vm.ToValue(typ), e.vm.ToValue(props))
		e.reportError(err)
		prevented := err == nil && v.ToBoolean()
		if after != nil {
			after(prevented)
		}
	})
}

// HasListener returns whether the given node or one that it is in has a
// listener for events of the given type (or an on<type> handler), as for
// whether its widget needs to handle them. It must be called in a task,
// or while the engine has no task running (as while rendering in a task).
func (e *Engine) HasListener(n *html.Node, typ string) bool {
	v := e.callPrelude("hasListener", e.nodes.id(n), typ)
	return v.ToBoolean()
}
