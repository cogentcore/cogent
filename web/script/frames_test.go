// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// frameEngine returns an engine for the given page at the given URL, in a
// frame of the given parent engine (if any), with the given function for
// the engines of its iframes.
func frameEngine(t *testing.T, page, url string, parent *Engine, frameOf func(n *html.Node) *Engine) *Engine {
	doc, err := html.Parse(strings.NewReader(page))
	require.NoError(t, err)
	opts := Options{PageURL: url, FrameOf: frameOf}
	if parent != nil {
		opts.Parent = func() *Engine { return parent }
	}
	e, err := New(doc, opts)
	require.NoError(t, err)
	t.Cleanup(e.Close)
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	return e
}

// TestFrameMessages tests the windows of frames: window.parent and top of
// a page in a frame, the contentWindow of its iframe, and the messages
// that they post to each other (as a captcha in a frame does when it is
// solved), with their origins and sources, and that a message to another
// target origin is not sent.
func TestFrameMessages(t *testing.T) {
	var child *Engine
	parent := frameEngine(t, `<html><body><iframe id="f" src="https://captcha.example.net/c"></iframe></body></html>`,
		"https://example.com/page", nil, func(n *html.Node) *Engine {
			if n.Data == "iframe" {
				return child
			}
			return nil
		})
	child = frameEngine(t, `<html><body></body></html>`, "https://captcha.example.net/c", parent, nil)

	run(t, parent, `
		window.got = [];
		const f = document.getElementById("f");
		f.addEventListener("load", () => got.push("load"));
		window.addEventListener("message", e => {
			got.push(e.origin, JSON.stringify(e.data), e.source === f.contentWindow, window.parent === window, window.top === window);
			e.source.postMessage({ reply: e.data.cookie }, e.origin);
		});`)
	run(t, child, `
		window.got = [];
		window.addEventListener("message", e => got.push(e.origin, e.data.reply, e.source === window.parent, e.source === window.top));
		window.parent.postMessage("ignored", "https://elsewhere.example.org");
		window.parent.postMessage({ cookie: "dd=1" }, "*");`)
	assert.Eventually(t, func() bool { return run(t, child, `got.length`) == int64(4) }, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, `https://captcha.example.net,{"cookie":"dd=1"},true,true,true`, run(t, parent, `got.join(",")`))
	assert.Equal(t, `https://example.com,dd=1,true,true`, run(t, child, `got.join(",")`))
	assert.Equal(t, true, run(t, child, `window.parent !== window && window.parent.top === window.parent && !window.parent.closed`))

	// the load event of the iframe, when its page has loaded
	n := parent.Document().FirstChild.LastChild.FirstChild // html > body > iframe
	require.Equal(t, "iframe", n.Data)
	parent.FrameLoaded(n)
	assert.Eventually(t, func() bool { return run(t, parent, `got.at(-1)`) == "load" }, 2*time.Second, 10*time.Millisecond)

	child.Close()
	assert.Equal(t, true, run(t, parent, `document.getElementById("f").contentWindow.closed`))
}

// TestURLOriginDefaultPort tests that origins and URLs have no default
// ports, and lower case schemes and hosts, as in browsers, so that a
// message posted to a target origin with the default port (as reCAPTCHA
// posts to https://www.google.com:443) is delivered.
func TestURLOriginDefaultPort(t *testing.T) {
	assert.Equal(t, "https://www.google.com", urlOrigin("https://www.google.com:443"))
	assert.Equal(t, "https://www.google.com", urlOrigin("HTTPS://WWW.Google.com:443/x"))
	assert.Equal(t, "http://a.com", urlOrigin("http://a.com:80/"))
	assert.Equal(t, "https://a.com:8443", urlOrigin("https://a.com:8443/"))
	assert.Equal(t, "http://[::1]:8080", urlOrigin("http://[::1]:8080/"))
	assert.Equal(t, "null", urlOrigin("about:blank"))
	e, err := New(nil, Options{PageURL: "https://www.google.com/"})
	require.NoError(t, err)
	defer e.Close()
	v, err := e.RunString(`const u = new URL("https://www.google.com:443/a?b"); [u.href, u.host, u.port, u.origin].join(" ")`)
	require.NoError(t, err)
	assert.Equal(t, "https://www.google.com/a?b www.google.com  https://www.google.com", v)
}

// TestFrameNames tests the windows of frames by the names (of the name
// attributes of their iframes) and indexes, as window[name], frames[i],
// and window.length, of this window and of others, as the challenge of
// reCAPTCHA posts to its checkbox by parent.frames[name], to the target
// origin with the default port (https://www.google.com:443), which was
// not delivered (the frames had no names, and the port was compared),
// and the name of the window of a frame, that of its iframe, from which
// reCAPTCHA gets that of its checkbox.
func TestFrameNames(t *testing.T) {
	var anchor, chal *Engine
	parent := frameEngine(t, `<html><body><iframe name="a-check" src="https://captcha.example.net/a"></iframe>`+
		`<div><iframe name="c-chal" src="https://captcha.example.net/c"></iframe></div></body></html>`,
		"https://example.com/page", nil, func(n *html.Node) *Engine {
			if nameAttr(n) == "a-check" {
				return anchor
			}
			return chal
		})
	anchor = frameEngine(t, `<html><body></body></html>`, "https://captcha.example.net/a", parent, nil)
	chal = frameEngine(t, `<html><body></body></html>`, "https://captcha.example.net/c", parent, nil)

	assert.Equal(t, "2 true true true true", run(t, parent, `const fs = document.getElementsByTagName("iframe");
		[window.length, window["a-check"] === fs[0].contentWindow, frames["c-chal"] === fs[1].contentWindow,
			window[1] === fs[1].contentWindow, window["no-such-frame"] === undefined].join(" ")`))
	// the name of the window of a frame is that of its iframe, until set
	assert.Equal(t, "c-chal", run(t, chal, `window.name`))
	assert.Equal(t, "", run(t, parent, `window.name`))
	assert.Equal(t, "renamed", run(t, anchor, `window.name = "renamed"; window.name`))
	run(t, anchor, `window.got = []; window.addEventListener("message", e => got.push(e.origin, e.data));`)
	assert.Equal(t, int64(2), run(t, chal, `parent.frames["a-check"].postMessage("hello", "https://captcha.example.net:443"); parent.length`))
	assert.Eventually(t, func() bool { return run(t, anchor, `got.join(",")`) == "https://captcha.example.net,hello" }, 2*time.Second, 10*time.Millisecond)
}
