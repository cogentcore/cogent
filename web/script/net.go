// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	neturl "net/url"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"cogentcore.org/core/htmlcore"
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
	return e.baseURLOf(e.doc)
}

// baseURLOf returns the base URL of the given document (the root of a node,
// as a document parsed by a script with DOMParser): that of its base
// element, relative to the URL of the page, or that of the page.
func (e *Engine) baseURLOf(doc *html.Node) string {
	base := e.opts.PageURL
	walk(doc, func(n *html.Node) bool {
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
	e.setMediaFuncs(set)
	e.setMediaPlayerFuncs(set)

	set("pageURL", func(call goja.FunctionCall) goja.Value { return vm.ToValue(e.opts.PageURL) })
	set("baseURL", func(call goja.FunctionCall) goja.Value {
		// of the document of the given node, if any
		if id := call.Argument(0); !goja.IsUndefined(id) {
			if n := e.nodes.node(int(id.ToInteger())); n != nil {
				for n.Parent != nil {
					n = n.Parent
				}
				return vm.ToValue(e.baseURLOf(n))
			}
		}
		return vm.ToValue(e.baseURL())
	})
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
		normalizeURL(pu)
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
			e.navStack.Store(e.Stack(8))   // (before the goroutine, for it)
			go e.opts.Navigate(u, replace) // after the task, which holds the lock
		}
		return goja.Undefined()
	})
	// createObjectURL returns a new blob: URL for the given data, of the
	// given type, which scripts and fetch can get until it is revoked.
	set("createObjectURL", func(call goja.FunctionCall) goja.Value {
		e.mu.Lock()
		defer e.mu.Unlock()
		if e.blobs == nil {
			e.blobs = map[string]blob{}
		}
		e.timerID++
		u := fmt.Sprintf("blob:%s/%d-%d", origin(e.opts.PageURL), time.Now().UnixNano(), e.timerID)
		e.blobs[u] = blob{data: str(call, 0), typ: str(call, 1)}
		return vm.ToValue(u)
	})
	set("revokeObjectURL", func(call goja.FunctionCall) goja.Value {
		e.mu.Lock()
		delete(e.blobs, str(call, 0))
		e.mu.Unlock()
		return goja.Undefined()
	})
	set("submit", func(call goja.FunctionCall) goja.Value {
		u, err := resolveURL(str(call, 0), e.baseURL())
		if err != nil {
			return goja.Undefined()
		}
		// in the frame of an iframe that the form targets, if any
		if id := call.Argument(3); !goja.IsUndefined(id) && !goja.IsNull(id) && e.opts.SubmitFrame != nil {
			if n := e.nodes.node(int(id.ToInteger())); n != nil {
				post := call.Argument(4).ToBoolean()
				go e.opts.SubmitFrame(n, u, str(call, 1), []byte(str(call, 2)), post) // after the task
				return goja.Undefined()
			}
		}
		if e.opts.Submit != nil {
			go e.opts.Submit(u, str(call, 1), []byte(str(call, 2))) // after the task
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
			if e.lockTask() {
				e.opts.PushState(u, call.Argument(1).ToBoolean())
			}
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
		credentials := call.Argument(5).ToBoolean()
		if err != nil {
			e.callBack(cb, 0, "", nil, nil, "", err.Error())
			return goja.Undefined()
		}
		go func() {
			start := time.Now()
			resp, err := e.request(u, method, headers, body, hasBody, credentials)
			// failed requests are in the console, as in browsers (as those
			// that a captcha makes, which says only that something went
			// wrong)
			m := method
			if m == "" {
				m = "GET"
			}
			if err != nil {
				e.logf(slog.LevelWarn, "%s %s: %v", m, logURL(u), err)
				e.callBack(cb, 0, "", nil, nil, "", err.Error())
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode >= 400 {
				e.logf(slog.LevelWarn, "%s %s: %s", m, logURL(u), resp.Status)
			}
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				e.callBack(cb, 0, "", nil, nil, "", err.Error())
				return
			}
			if e.tracing() {
				typ := ""
				if ct := resp.Header.Get("Content-Type"); ct != "" {
					typ = ", " + ct
				}
				e.tracef("%s %s (%d bytes): %d, %d bytes%s, %v", m, logURL(u), len(body), resp.StatusCode, len(data),
					typ, time.Since(start).Round(time.Millisecond))
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
	// digest returns the digest of the given data with the given algorithm
	// (as SHA-256) for crypto.subtle.digest, or null for an unknown one
	// (Google signs the requests of its APIs with SHA-1 digests).
	set("digest", func(call goja.FunctionCall) goja.Value {
		b := bytesOf(call.Argument(1))
		var sum []byte
		switch strings.ToUpper(str(call, 0)) {
		case "SHA-1":
			s := sha1.Sum(b)
			sum = s[:]
		case "SHA-256":
			s := sha256.Sum256(b)
			sum = s[:]
		case "SHA-384":
			s := sha512.Sum384(b)
			sum = s[:]
		case "SHA-512":
			s := sha512.Sum512(b)
			sum = s[:]
		default:
			return goja.Null()
		}
		return vm.ToValue(vm.NewArrayBuffer(sum))
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

// logURL returns the given URL for the console: without the data of a
// data: URL, and at most 200 characters.
// defaultPorts are the default ports of the schemes of URLs, which are not
// in their hosts and origins, as browsers serialize them.
var defaultPorts = map[string]string{"http": "80", "https": "443", "ws": "80", "wss": "443", "ftp": "21"}

// normalizeURL makes the scheme and host of the given URL lower case and
// drops the default port of its scheme from its host, as browsers parse
// URLs (as https://www.google.com:443, which reCAPTCHA posts messages to).
func normalizeURL(pu *neturl.URL) {
	pu.Scheme = strings.ToLower(pu.Scheme)
	if pu.Host == "" {
		return
	}
	host, port := strings.ToLower(pu.Hostname()), pu.Port()
	if strings.Contains(host, ":") {
		host = "[" + host + "]" // (IPv6)
	}
	if port != "" && port != defaultPorts[pu.Scheme] {
		host += ":" + port
	}
	pu.Host = host
}

// logURL returns the given URL shortened for logs: data URLs without
// their data, and long URLs without their middle.
func logURL(u string) string {
	if strings.HasPrefix(u, "data:") {
		u, _, _ = strings.Cut(u, ",")
		return u + ",…"
	}
	if len(u) > 240 {
		// the ends of long URLs differ (as the module lists of the
		// scripts of Google, m=ZwDk9d,RMhBfe, after 1500 bytes)
		return strings.ToValidUTF8(u[:120], "") + "…" + strings.ToValidUTF8(u[len(u)-120:], "")
	}
	return u
}

// request makes the given request: a GET without a body or headers with
// GetURL, as for the resources of the page, and others with the default
// client. A GET with headers is a request of data (as the JSON of the
// page_data of github.com, with Accept: application/json, which GetURL,
// without them, got as 406 Not Acceptable, and which its cache kept). The
// cookies of other origins are sent if credentials.
func (e *Engine) request(u, method string, headers []string, body []byte, hasBody, credentials bool) (*http.Response, error) {
	if strings.HasPrefix(u, "blob:") {
		return e.blobResponse(u)
	}
	if (method == "" || method == "GET") && !hasBody && len(headers) == 0 && e.opts.GetURL != nil {
		return e.opts.GetURL(u)
	}
	req, err := http.NewRequest(method, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	// the headers of browsers, unless the script sets them
	htmlcore.SetRequestHeaders(req)
	req.Header.Set("Accept", "*/*")
	set := map[string]bool{}
	for i := 0; i+1 < len(headers); i += 2 {
		k := http.CanonicalHeaderKey(headers[i])
		if forbiddenHeaders[k] {
			continue // (which scripts can not set)
		}
		if !set[k] {
			set[k] = true
			req.Header.Del(k)
		}
		req.Header.Add(headers[i], headers[i+1])
	}
	setOriginHeaders(req, e.opts.PageURL, method)
	// The cookies are only sent to the origin of the page, as browsers do
	// by default (not with the requests of a page to other sites), unless
	// the request is with credentials (as the APIs of Google, of other
	// hosts, which were 401 Unauthorized without its cookies).
	client := &http.Client{Timeout: 30 * time.Second}
	if credentials || origin(u) == origin(e.opts.PageURL) {
		client.Jar = e.opts.Cookies
	}
	return client.Do(req)
}

// forbiddenHeaders are the headers of requests that scripts can not set,
// as in browsers.
var forbiddenHeaders = map[string]bool{"Origin": true, "Referer": true, "Cookie": true, "Host": true,
	"Content-Length": true, "Connection": true}

// setOriginHeaders sets the Origin and Referer headers of the given request
// of a script of the page of the given URL, as browsers do: the origin of
// the page for requests to other origins and those that are not GET or
// HEAD, and the referrer of the page (only its origin for other origins).
// The APIs of Google check that the origin in the hash of their
// authorization (SAPISIDHASH) is that of the request, which were 401
// Unauthorized without it.
func setOriginHeaders(req *http.Request, page, method string) {
	po := origin(page)
	if po == "null" {
		return
	}
	cross := origin(req.URL.String()) != po
	if cross || (method != "" && method != "GET" && method != "HEAD") {
		req.Header.Set("Origin", po)
	}
	if cross {
		req.Header.Set("Referer", po+"/")
	} else {
		ref, _, _ := strings.Cut(page, "#")
		req.Header.Set("Referer", ref)
	}
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

// isDeferred returns whether the given script element of the document
// runs after it is parsed: a classic script with a src and the defer
// attribute, or a module, unless it is async.
func isDeferred(n *html.Node) bool {
	if _, async := attr(n, "async"); async {
		return false
	}
	if typ, _ := attr(n, "type"); strings.EqualFold(strings.TrimSpace(typ), "module") {
		return true
	}
	_, src := attr(n, "src")
	_, def := attr(n, "defer")
	return src && def
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
// load event, after which it calls done (if not nil) in the task. The page
// is not rendered again while it loads, but after the load event if the
// scripts have changed it (at the end of its task).
func (e *Engine) Load(done func()) {
	var scripts []*html.Node
	var links []*html.Node
	walk(e.doc, func(n *html.Node) bool {
		if n.Type == html.ElementNode && (n.Data == "template" || n.Data == "noscript") {
			return true // their content is not scripts (it is text with scripting)
		}
		if isScript(n) {
			scripts = append(scripts, n)
		}
		if isStyleLink(n) {
			links = append(links, n)
		}
		return true
	})
	e.Post(func() {
		e.loading = true
		e.readImportMaps()
		if e.ran == nil {
			e.ran = map[*html.Node]bool{}
		}
		for _, s := range scripts {
			e.ran[s] = true // run here, not when inserted
		}
		// the load events of the links of style sheets (and preloads) of
		// the document, once they are gotten
		for _, l := range links {
			e.ran[l] = true
			e.loadStyleLink(l)
		}
	})
	// the deferred scripts (with defer, and modules) run after the others,
	// in order, before DOMContentLoaded, as in browsers (a deferred script
	// in the head uses the variables of inline scripts in the body)
	var deferred []*html.Node
	for _, s := range scripts {
		if isDeferred(s) {
			deferred = append(deferred, s)
			continue
		}
		e.Post(func() { e.runScript(s, true) })
	}
	for _, s := range deferred {
		e.Post(func() { e.runScript(s, true) })
	}
	e.Post(func() {
		e.callPrelude("loaded", "interactive")
	})
	// the load event is after the scripts that were inserted while the
	// page loaded, as in browsers (as frontend.js of pkg.go.dev, which a
	// script of the page inserts, and which sets up its tooltips on load)
	var complete func()
	complete = func() {
		if e.insertedPending > 0 {
			e.Post(complete) // after them
			return
		}
		e.loading = false
		e.callPrelude("loaded", "complete")
		if done != nil {
			done()
		}
	}
	e.Post(complete)
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
	if v == nil {
		// with an error, as when the engine is interrupted as it is closed
		// (as by reloading the page while it is rendered)
		return goja.Undefined()
	}
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
		// the module, and those that it imports, are defined in the
		// registry, and then it is evaluated
		entry, meta := name, ""
		if !hasSrc {
			e.timerID++
			entry = fmt.Sprintf("%s#inline-module-%d", strings.Split(e.baseURL(), "#")[0], e.timerID)
			meta = e.baseURL()
			name = entry
		}
		defs, err := e.moduleGraph(entry, &code, meta)
		if err != nil {
			e.logf(slog.LevelWarn, "module %s: %v", name, err)
			e.callPrelude("fireEvent", e.nodes.id(s), "error")
			return
		}
		e.defineModules(defs)
		code = "__requireModule(" + quoteJS(entry) + ");"
	}
	prev := e.current // (of a script that inserted this one, run in it)
	e.current = s
	e.run(name, code)
	e.current = prev
	if hasSrc {
		e.callPrelude("fireEvent", e.nodes.id(s), "load")
	}
}

// getScript gets the script at the given URL.
func (e *Engine) getScript(u string) (string, error) {
	if strings.HasPrefix(u, "data:") {
		return dataURL(u)
	}
	var resp *http.Response
	var err error
	if strings.HasPrefix(u, "blob:") {
		resp, err = e.blobResponse(u)
	} else if e.opts.GetURL == nil {
		return "", fmt.Errorf("no GetURL")
	} else {
		resp, err = e.opts.GetURL(u)
	}
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
	e.tracef("script %s: %d bytes", logURL(u), len(b))
	return string(b), nil
}

// runInserted runs the scripts that the given node (that was inserted in
// the document) is or has, which have not run yet: the external ones once
// they are gotten (in another goroutine), and the others in a new task, or
// now if sync (inline classic scripts inserted by a script, as browsers
// run them in appendChild: Google inserts the code of its bot check and
// removes it right after, and then uses what it defines).
func (e *Engine) runInserted(n *html.Node, sync bool) {
	if !connected(n, e.doc) {
		return
	}
	walk(n, func(c *html.Node) bool {
		if c.Type == html.ElementNode && c.Data == "iframe" && e.tracing() && !e.ran[c] {
			// (once, as its parents are inserted again)
			if e.ran == nil {
				e.ran = map[*html.Node]bool{}
			}
			e.ran[c] = true
			src, _ := attr(c, "src")
			e.tracef("iframe %q inserted: %s", nameAttr(c), logURL(src))
		}
		if e.ran[c] {
			return true
		}
		if isStyleLink(c) {
			if e.ran == nil {
				e.ran = map[*html.Node]bool{}
			}
			e.ran[c] = true
			e.loadStyleLink(c)
			return true
		}
		if !isScript(c) {
			return true
		}
		if e.ran == nil {
			e.ran = map[*html.Node]bool{}
		}
		e.ran[c] = true
		_, hasSrc := attr(c, "src")
		if !hasSrc && strings.TrimSpace(textContent(c)) == "" {
			return true
		}
		if typ, _ := attr(c, "type"); sync && !hasSrc && !strings.EqualFold(strings.TrimSpace(typ), "module") {
			e.runScript(c, false)
			return true
		}
		pending := e.loading // which the load event is after (see Load)
		if pending {
			e.insertedPending++
		}
		e.Post(func() {
			e.runScript(c, false)
			if pending {
				e.insertedPending--
			}
		})
		return true
	})
}

// isStyleLink returns whether the given node is a link to a style sheet.
func isStyleLink(n *html.Node) bool {
	if n.Type != html.ElementNode || n.Data != "link" || n.Namespace != "" {
		return false
	}
	rel, _ := attr(n, "rel")
	for _, r := range strings.Fields(strings.ToLower(rel)) {
		// (and the links that preload resources, whose load events pages
		// use to make them style sheets, as the preloaded style sheets of
		// trekbikes.com, with onload="this.rel='stylesheet'", which were
		// not applied)
		if r == "stylesheet" || r == "preload" || r == "modulepreload" {
			return true
		}
	}
	return false
}

// loadStyleLink gets the style sheet of the given link that was inserted
// in the document (in another goroutine), and then fires its load (or
// error) event in a task, as scripts that load style sheets wait for it
// (as craigslist does to show its pages). The style sheet is applied when
// the page is read.
func (e *Engine) loadStyleLink(n *html.Node) {
	href, ok := attr(n, "href")
	if !ok {
		return
	}
	u, err := resolveURL(href, e.baseURL()) // in the task: it reads the document
	go func() {
		event := "load"
		if err == nil {
			_, err = e.getScript(u)
		}
		if err != nil {
			e.logf(slog.LevelWarn, "style sheet %s: %v", href, err)
			event = "error"
		}
		e.Post(func() { e.callPrelude("fireEvent", e.nodes.id(n), event) })
	}()
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
		e.runInserted(k, false)
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

// Activate does the default action of the given element (in a task), as
// after a click on it that was not prevented: submitting its form for a
// submit button, or for a text field (as for Enter).
func (e *Engine) Activate(n *html.Node) {
	e.callPrelude("activate", e.nodes.id(n))
}

// dataURL returns the content of the given data URL.
func dataURL(u string) (string, error) {
	meta, data, ok := strings.Cut(strings.TrimPrefix(u, "data:"), ",")
	if !ok {
		return "", fmt.Errorf("invalid data URL")
	}
	if strings.HasSuffix(meta, ";base64") {
		b, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			b, err = base64.RawStdEncoding.DecodeString(data)
		}
		return string(b), err
	}
	return neturl.PathUnescape(data)
}

// ControlState returns the state of the given form control that is not
// in its attributes (which the page is read from): its value and whether
// it is checked, and whether the user or the scripts have changed each.
// It must be called when no script runs (as while rendering the page).
func (e *Engine) ControlState(n *html.Node) (value string, hasValue, checked, hasChecked bool) {
	v := e.callPrelude("state", e.nodes.id(n))
	o, ok := v.(*goja.Object)
	if !ok {
		return
	}
	if x := o.Get("0"); x != nil && !goja.IsNull(x) && !goja.IsUndefined(x) {
		value, hasValue = x.String(), true
	}
	if x := o.Get("1"); x != nil && !goja.IsNull(x) && !goja.IsUndefined(x) {
		checked, hasChecked = x.ToBoolean(), true
	}
	return
}

// blob is the data of a blob: URL, and its content type.
type blob struct {
	data, typ string
}

// blobResponse returns a response with the data of the given blob: URL,
// or an error if there is none (as when it was revoked).
func (e *Engine) blobResponse(u string) (*http.Response, error) {
	e.mu.Lock()
	b, ok := e.blobs[strings.SplitN(u, "#", 2)[0]]
	e.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("no blob at %s", u)
	}
	pu, _ := neturl.Parse(u)
	h := http.Header{}
	if b.typ != "" {
		h.Set("Content-Type", b.typ)
	}
	return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: h, Request: &http.Request{URL: pu},
		Body: io.NopCloser(strings.NewReader(b.data))}, nil
}
