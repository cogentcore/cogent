// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDOMEdges tests the edge cases of the DOM: invalid selectors and
// selectors that are closed, the types of nodes, clones, the text of
// comments, insertions that are not allowed, the console levels, btoa and
// atob of invalid strings, the serialization of doctypes, and the state of
// loading.
func TestDOMEdges(t *testing.T) {
	e, logs := testEngine(t, `<!DOCTYPE html><html><body><!--c--><div id="a" class="x"><p>One</p></div></body></html>`, nil)
	assert.Equal(t, "SyntaxError 12 true", run(t, e, `try { document.querySelector("!!") } catch (err) { [err.name, err.code, err instanceof DOMException].join(" ") }`))
	assert.Equal(t, "a", run(t, e, `document.querySelector('[class^="x"').id`), "closed")
	assert.Equal(t, "8 10 c", run(t, e, `const c = document.body.firstChild; [c.nodeType, document.doctype.nodeType, c.textContent].join(" ")`))
	assert.Equal(t, "d", run(t, e, `c.textContent = "d"; c.data`))
	assert.Equal(t, "DIV 1 0", run(t, e, `
		const a = document.getElementById("a");
		const deep = a.cloneNode(true), shallow = a.cloneNode(false);
		[deep.tagName, deep.childNodes.length, shallow.childNodes.length].join(" ")`))
	assert.Equal(t, "NotFoundError", run(t, e, `try { a.insertBefore(document.createElement("i"), document.body) } catch (err) { err.name }`))
	assert.Equal(t, "HierarchyRequestError", run(t, e, `try { a.firstChild.appendChild(a) } catch (err) { err.name }`))
	assert.Equal(t, "InvalidCharacterError InvalidCharacterError", run(t, e, `
		const r = [];
		try { btoa("€") } catch (err) { r.push(err.name) }
		try { atob("!!!") } catch (err) { r.push(err.name) }
		r.join(" ")`))
	assert.Equal(t, "ab", run(t, e, `atob(" YW\nI= ")`))
	assert.Contains(t, run(t, e, `new XMLSerializer().serializeToString(document)`), "<!DOCTYPE html>")
	assert.Equal(t, "loading", run(t, e, `document.readyState`), "not loaded")
	run(t, e, `console.error("e1"); console.warn("w1"); console.debug("d1")`)
	e.Wait()
	assert.Contains(t, *logs, "ERROR: e1")
	assert.Contains(t, *logs, "WARN: w1")
	assert.Contains(t, *logs, "DEBUG: d1")
	assert.Equal(t, "6", run(t, e, `String(new Function("a", "async function* g() { yield a } let s = 0; return s + 6")(1))`))
}

// TestDynamicImportErrors tests the errors of dynamic imports: of their
// specifiers, and of the modules that are not found.
func TestDynamicImportErrors(t *testing.T) {
	e, _ := testEngine(t, `<html><body></body></html>`, map[string]string{"/ok.js": `export const v = 1;`})
	e.Eval(`window.r = []; // (as a script, whose dynamic imports are rewritten)
		const imp = s => import(s);
		imp("http://[bad/x.js").catch(err => r.push("bad"));
		imp("/none.js").catch(err => r.push("none"));
		imp("/ok.js").then(m => r.push("ok " + m.v));`)
	e.Wait()
	require.Eventually(t, func() bool {
		v, _ := e.RunString(`r.slice().sort().join()`)
		return v == "bad,none,ok 1"
	}, 5*time.Second, 10*time.Millisecond)
}
