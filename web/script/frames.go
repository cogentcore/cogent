// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/dop251/goja"
	"golang.org/x/net/html"
)

// The windows of frames: the page of each iframe has its own engine, and
// the scripts of a page see the windows of the others (window.parent and
// top, and the contentWindow of an iframe) as window proxies, which can
// only post messages to them (and get their parent and top), as for the
// windows of other origins in browsers, as the captchas of bot checks
// (which tell the page that they are in that they are solved with
// postMessage). The windows of the same origin are the same, as engines
// cannot share their objects; the messages are copied as JSON.

// windowID returns the number of the window proxy of the given engine
// for the scripts of this engine: -1 for none, 0 for this one (the global
// object), and others in the order that they are first seen, so that the
// proxy of a window is the same object whenever scripts get it (as
// event.source and the contentWindow of its iframe). It is called in
// tasks.
func (e *Engine) windowID(other *Engine) int {
	switch {
	case other == nil:
		return -1
	case other == e:
		return 0
	}
	if e.windowIDs == nil {
		e.windowIDs = map[*Engine]int{}
	}
	if id, ok := e.windowIDs[other]; ok {
		return id
	}
	e.windows = append(e.windows, other)
	id := len(e.windows) // from 1
	e.windowIDs[other] = id
	return id
}

// windowOf returns the engine of the given window proxy number (see
// [Engine.windowID]), or nil.
func (e *Engine) windowOf(id int) *Engine {
	switch {
	case id == 0:
		return e
	case id < 1 || id > len(e.windows):
		return nil
	}
	return e.windows[id-1]
}

// parent returns the engine of the window that this one is in a frame of,
// or nil for a top window.
func (e *Engine) parent() *Engine {
	if e.opts.Parent == nil {
		return nil
	}
	return e.opts.Parent()
}

// top returns the engine of the top window that this one is in (itself
// if it is not in a frame).
func (e *Engine) top() *Engine {
	t := e
	for p := t.parent(); p != nil; p = t.parent() {
		t = p
	}
	return t
}

// Origin returns the origin of the page of the engine (as
// https://example.com), or "null" for a URL that has none.
func (e *Engine) Origin() string {
	return urlOrigin(e.opts.PageURL)
}

// urlOrigin returns the origin of the given URL, or "null".
func urlOrigin(u string) string {
	pu, err := url.Parse(u)
	if err != nil || pu.Host == "" {
		return "null"
	}
	// the default port of the scheme is not in an origin, as browsers
	// serialize it (as the target origin https://www.google.com:443 of the
	// messages of the frame of reCAPTCHA to its page, which were not
	// delivered, so that it said that something went wrong)
	normalizeURL(pu)
	if pu.Scheme != "http" && pu.Scheme != "https" {
		return "null"
	}
	return pu.Scheme + "://" + pu.Host
}

// postMessage posts the given message (as JSON) from this engine to the
// given one, if its origin is the given target origin ("*" for any, and
// "/" for that of this one), as for the postMessage of a window. It is
// dispatched as a message event to the window of the other engine, in a
// task of it.
func (e *Engine) postMessage(to *Engine, data, targetOrigin string, ports []int) {
	if to == nil {
		return
	}
	origin := e.Origin()
	switch targetOrigin {
	case "*":
	case "/":
		if to.Origin() != origin {
			e.tracef("postMessage to %s not delivered: its origin is not that of the sender (/)", to.Origin())
			return
		}
	default:
		if to.Origin() != urlOrigin(targetOrigin) {
			e.tracef("postMessage to %s not delivered: target origin %s", to.Origin(), targetOrigin)
			return // as browsers, which do not send it (or report an error)
		}
	}
	e.tracef("postMessage to %s: %s", to.Origin(), traceData(data))
	pairs := e.transferPorts(to, ports)
	to.Post(func() {
		to.callPrelude("message", data, origin, to.windowID(e), pairs)
	})
}

// nameAttr returns the name attribute of the given element.
func nameAttr(n *html.Node) string {
	v, _ := attr(n, "name")
	return v
}

// forFrames calls the given function for the iframe elements of the
// document of the engine, in order, until it returns false. It is called
// in the tasks of engines of the same page (or its frames), with the lock
// of the page held, as the documents are only changed in their tasks.
//
// The elements are found again only after the document has changed (see
// [Engine.domChanges]), as this is called for each property of the window
// that it does not have (as for the features that scripts check for).
func (e *Engine) forFrames(f func(n *html.Node) bool) {
	if e.framesAt != e.domChanges+1 {
		e.frameNodes = e.frameNodes[:0]
		var walk func(n *html.Node)
		walk = func(n *html.Node) {
			for k := n.FirstChild; k != nil; k = k.NextSibling {
				if k.Type != html.ElementNode || k.Data == "template" {
					continue
				}
				if k.Data == "iframe" || k.Data == "frame" {
					e.frameNodes = append(e.frameNodes, k)
					continue
				}
				walk(k)
			}
		}
		if e.doc != nil {
			walk(e.doc)
		}
		e.framesAt = e.domChanges + 1
	}
	for _, n := range e.frameNodes {
		if !f(n) {
			return
		}
	}
}

// traceData returns the given data of a message (JSON) for the console,
// shortened.
func traceData(data string) string {
	if len(data) > 300 {
		return data[:300] + "…"
	}
	return data
}

// FrameLoaded dispatches the load event to the given iframe element, after
// the page in its frame has loaded (or failed to), as in browsers.
func (e *Engine) FrameLoaded(n *html.Node) {
	e.Post(func() {
		e.callPrelude("frameLoaded", e.nodes.id(n))
	})
}

// defineFrames defines the functions of the windows of frames for the
// prelude (see [Engine.windowID]).
func (e *Engine) defineFrames() {
	vm := e.vm
	d := e.jsDOM
	d.Set("windowRel", func(call goja.FunctionCall) goja.Value {
		w := e.windowOf(int(call.Argument(0).ToInteger()))
		if w == nil {
			return vm.ToValue(-1)
		}
		switch call.Argument(1).String() {
		case "parent":
			if p := w.parent(); p != nil {
				return vm.ToValue(e.windowID(p))
			}
			return vm.ToValue(e.windowID(w)) // a top window is its own parent
		case "top":
			return vm.ToValue(e.windowID(w.top()))
		}
		return vm.ToValue(-1)
	})
	d.Set("frameWindow", func(call goja.FunctionCall) goja.Value {
		n := e.nodes.node(int(call.Argument(0).ToInteger()))
		if n == nil || e.opts.FrameOf == nil {
			return vm.ToValue(-1)
		}
		return vm.ToValue(e.windowID(e.opts.FrameOf(n)))
	})
	// windowFrame returns the number of the window of the frame of the
	// given window of the given name (of the name attribute of its
	// iframe) or index, as window[name] and window.frames[name] (as
	// reCAPTCHA gets the window of its checkbox from that of its
	// challenge, by parent.frames[name]), or -1 for none.
	d.Set("windowFrame", func(call goja.FunctionCall) goja.Value {
		w := e.windowOf(int(call.Argument(0).ToInteger()))
		if w == nil || w.opts.FrameOf == nil {
			return vm.ToValue(-1)
		}
		key := call.Argument(1).String()
		index, err := strconv.Atoi(key)
		if err != nil || index < 0 || strconv.Itoa(index) != key {
			index = -1
		}
		i := 0
		var found *html.Node
		w.forFrames(func(n *html.Node) bool {
			if i == index || index < 0 && key != "" && nameAttr(n) == key {
				found = n
				return false
			}
			i++
			return true
		})
		if found == nil {
			return vm.ToValue(-1)
		}
		fe := w.opts.FrameOf(found)
		if fe == nil {
			e.tracef("frame %q of %s has no window (its page has no scripts yet)", key, w.Origin())
		}
		return vm.ToValue(e.windowID(fe))
	})
	// frameName returns the name of the iframe of the frame of this
	// window, which is its window.name (as reCAPTCHA gets the name of the
	// frame of its checkbox from that of its challenge), or "".
	d.Set("frameName", func(call goja.FunctionCall) goja.Value {
		p := e.parent()
		if p == nil || p.opts.FrameOf == nil {
			return vm.ToValue("")
		}
		name := ""
		p.forFrames(func(n *html.Node) bool {
			if p.opts.FrameOf(n) == e {
				name = nameAttr(n)
				return false
			}
			return true
		})
		return vm.ToValue(name)
	})
	// windowFrameNames returns the names of the frames of the given
	// window, for the console.
	d.Set("windowFrameNames", func(call goja.FunctionCall) goja.Value {
		w := e.windowOf(int(call.Argument(0).ToInteger()))
		var names []string
		if w != nil {
			w.forFrames(func(n *html.Node) bool { names = append(names, strconv.Quote(nameAttr(n))); return true })
		}
		url := ""
		if w != nil {
			url = w.opts.PageURL
		}
		return vm.ToValue(strings.Join(names, ", ") + " in " + url)
	})
	// windowLength returns the number of frames of the given window, as
	// window.length.
	d.Set("windowLength", func(call goja.FunctionCall) goja.Value {
		w := e.windowOf(int(call.Argument(0).ToInteger()))
		n := 0
		if w != nil {
			w.forFrames(func(*html.Node) bool { n++; return true })
		}
		return vm.ToValue(n)
	})
	d.Set("windowClosed", func(call goja.FunctionCall) goja.Value {
		w := e.windowOf(int(call.Argument(0).ToInteger()))
		return vm.ToValue(w == nil || w.Closed())
	})
	d.Set("postTo", func(call goja.FunctionCall) goja.Value {
		w := e.windowOf(int(call.Argument(0).ToInteger()))
		e.postMessage(w, call.Argument(1).String(), call.Argument(2).String(), portIDs(call.Argument(3)))
		return goja.Undefined()
	})
}
