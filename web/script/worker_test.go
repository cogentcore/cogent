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

// TestWorker tests web workers (as reCAPTCHA uses, whose checkbox did
// nothing without them): a worker of a blob: URL and one of a URL with
// importScripts, the global scope of a worker (without window and
// document), messages both ways (with typed arrays and maps, which JSON
// does not keep), the console of a worker in that of the page, and
// terminate.
func TestWorker(t *testing.T) {
	e, logs := testEngine(t, `<html><body></body></html>`, map[string]string{
		"/w.js":   `importScripts("/lib.js"); onmessage = e => postMessage({ echo: e.data, lib: lib(), scope: typeof window + " " + typeof document + " " + typeof self.importScripts });`,
		"/lib.js": `self.lib = () => "lib"; console.log("in worker");`,
	})
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	run(t, e, `
		window.got = [];
		const src = "onmessage = e => { const a = e.data; postMessage([a.bytes instanceof Uint8Array, a.bytes[2], a.map.get('k'), typeof a.u]); };";
		const bw = new Worker(URL.createObjectURL(new Blob([src], { type: "text/javascript" })));
		bw.onmessage = e => got.push("blob:" + e.data.join("/"));
		bw.postMessage({ bytes: new Uint8Array([1, 2, 3]), map: new Map([["k", "v"]]), u: undefined });
		const w = new Worker("/w.js");
		w.addEventListener("message", e => { got.push("url:" + e.data.echo + "/" + e.data.lib + "/" + e.data.scope); w.terminate(); });
		w.postMessage("hi");`)
	var got any
	assert.Eventually(t, func() bool {
		got = run(t, e, `got.sort().join(" ")`)
		return strings.Count(got.(string), ":") == 2
	}, 3*time.Second, 10*time.Millisecond)
	assert.Equal(t, "blob:true/3/v/undefined url:hi/lib/undefined undefined function", got)
	e.Wait()
	require.NotEmpty(t, *logs)
	assert.Contains(t, *logs, "INFO: [worker https://example.com/w.js] in worker")
}

// TestTransferPorts tests message ports that are transferred to a worker
// and to the window of a frame (as reCAPTCHA talks to them, which it timed
// out waiting for without them): the moved port is in the ports of the
// message, its partner talks to it both ways, and a message posted to it
// before it is made waits for it.
func TestTransferPorts(t *testing.T) {
	e, logs := testEngine(t, `<html><body></body></html>`, map[string]string{
		"/w.js": `onmessage = e => { const p = e.ports[0]; p.onmessage = m => p.postMessage("worker got " + m.data); p.postMessage("hello from worker"); };`,
	})
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	run(t, e, `
		window.got = [];
		const c = new MessageChannel();
		c.port1.onmessage = m => got.push(m.data);
		c.port1.postMessage("early");
		new Worker("/w.js").postMessage("port", [c.port2]);`)
	assert.Eventually(t, func() bool {
		return run(t, e, `got.join("|")`) == "hello from worker|worker got early"
	}, 3*time.Second, 10*time.Millisecond, run(t, e, `got.join("|")`))

	// to the window of a frame, and back over the port
	child := frameEngine(t, `<html><body></body></html>`, "https://captcha.example.net/c", e, nil)
	run(t, child, `
		window.got = [];
		window.addEventListener("message", m => { got.push(m.data + ":" + m.ports.length); m.ports[0].postMessage("pong"); });`)
	run(t, e, `
		window.got2 = [];
		const c2 = new MessageChannel();
		c2.port1.onmessage = m => got2.push(m.data);`)
	e.Post(func() { e.opts.FrameOf = func(*html.Node) *Engine { return child } }) // (in a task)
	e.Wait()
	run(t, e, `document.body.appendChild(document.createElement("iframe")).contentWindow.postMessage("ping", "*", [c2.port2]);`)
	assert.Eventually(t, func() bool { return run(t, e, `got2.join("|")`) == "pong" }, 3*time.Second, 10*time.Millisecond)
	assert.Equal(t, "ping:1", run(t, child, `got.join("|")`))
	e.Wait()
	for _, l := range *logs {
		assert.True(t, strings.HasPrefix(l, "DEBUG:"), l)
	}
}

// TestWindowPostMessageArray tests that postMessage(data, [port]) of a
// window takes the array as its options, as browsers do, so that no port
// is transferred (reCAPTCHA posts its worker setup so in windows too, and
// took its own port for that of its frame, which did not answer).
func TestWindowPostMessageArray(t *testing.T) {
	e, logs := testEngine(t, `<html><body></body></html>`, nil)
	loaded := make(chan bool, 1)
	e.Load(func() { loaded <- true })
	<-loaded
	run(t, e, `
		window.got = [];
		window.addEventListener("message", m => got.push(m.data + ":" + m.ports.length));
		const c = new MessageChannel();
		postMessage("array", [c.port2]);
		postMessage("third", "*", [new MessageChannel().port2]);`)
	assert.Eventually(t, func() bool { return run(t, e, `got.join(" ")`) == "array:0 third:1" }, 3*time.Second, 10*time.Millisecond, run(t, e, `got.join(" ")`))
	e.Wait()
	assert.Empty(t, *logs)
}
