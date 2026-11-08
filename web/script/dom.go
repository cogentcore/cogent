// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"bytes"
	"encoding/base64"
	"log/slog"
	"strings"

	"github.com/andybalholm/cascadia"
	"github.com/grafana/sobek"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// nodeIDs gives each node that the scripts use an id, which the prelude
// wraps in a JavaScript object (one for each node).
type nodeIDs struct {
	ids   map[*html.Node]int
	nodes []*html.Node
}

func (ni *nodeIDs) init() {
	ni.ids = map[*html.Node]int{}
	ni.nodes = []*html.Node{nil} // id 0 is no node
}

// id returns the id of the given node, or 0 for nil.
func (ni *nodeIDs) id(n *html.Node) int {
	if n == nil {
		return 0
	}
	if id, ok := ni.ids[n]; ok {
		return id
	}
	id := len(ni.nodes)
	ni.nodes = append(ni.nodes, n)
	ni.ids[n] = id
	return id
}

// node returns the node of the given id, or nil.
func (ni *nodeIDs) node(id int) *html.Node {
	if id <= 0 || id >= len(ni.nodes) {
		return nil
	}
	return ni.nodes[id]
}

// fragmentData is the data of the document nodes that are document
// fragments (as made by createDocumentFragment).
const fragmentData = "#document-fragment"

// changed records that the document has changed, so that the page is
// rendered again at the end of the task.
func (e *Engine) changed(n *html.Node) {
	if n == nil || connected(n, e.doc) {
		e.dirty = true
		if n != nil && len(e.changedNodes) < maxChangedNodes {
			e.changedNodes = append(e.changedNodes, n)
		} else {
			e.changedNodes = append(e.changedNodes, e.doc) // all of it
		}
	}
}

// maxChangedNodes is the number of changed nodes that are recorded (see
// [Engine.changedAround]), beyond which the whole document has changed.
const maxChangedNodes = 1000

// changedAround returns whether the given node, or a node that it is in
// or that is in it, has changed since the page was last rendered, which
// can change the box of its element (its size and position in it).
func (e *Engine) changedAround(n *html.Node) bool {
	for _, c := range e.changedNodes {
		if contains(c, n) || contains(n, c) {
			return true
		}
	}
	return false
}

// contains returns whether the given node is a or is in it.
func contains(a, n *html.Node) bool {
	for ; n != nil; n = n.Parent {
		if n == a {
			return true
		}
	}
	return false
}

// connected returns whether the given node is in the given document.
func connected(n, doc *html.Node) bool {
	for ; n != nil; n = n.Parent {
		if n == doc {
			return true
		}
	}
	return false
}

// defineDOM defines the Go functions of the DOM on __dom (e.jsDOM),
// which the prelude uses.
func (e *Engine) defineDOM() {
	vm := e.vm
	d := e.jsDOM
	ni := &e.nodes
	node := func(v sobek.Value) *html.Node { return ni.node(int(v.ToInteger())) }
	id := func(n *html.Node) sobek.Value { return vm.ToValue(ni.id(n)) }
	set := func(name string, f func(call sobek.FunctionCall) sobek.Value) {
		d.Set(name, f)
	}
	arg := func(call sobek.FunctionCall, i int) sobek.Value { return call.Argument(i) }
	str := func(call sobek.FunctionCall, i int) string { return call.Argument(i).String() }
	syntaxError := func(msg string) *sobek.Object {
		return vm.NewGoError(&domException{name: "SyntaxError", msg: msg})
	}
	compile := func(sel string) cascadia.Matcher {
		g, err := cascadia.ParseGroup(sel)
		if err != nil {
			// the brackets and parentheses that are open at the end are
			// closed, as in CSS (as for '[class^="x"')
			if closed := closeSelector(sel); closed != sel {
				g, err = cascadia.ParseGroup(closed)
			}
		}
		if err != nil {
			panic(syntaxError("'" + sel + "' is not a valid selector"))
		}
		return g
	}

	set("doc", func(call sobek.FunctionCall) sobek.Value { return id(e.doc) })
	set("type", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		if n == nil {
			return vm.ToValue(0)
		}
		switch n.Type {
		case html.ElementNode:
			return vm.ToValue(1)
		case html.TextNode:
			return vm.ToValue(3)
		case html.CommentNode:
			return vm.ToValue(8)
		case html.DocumentNode:
			if n.Data == fragmentData {
				return vm.ToValue(11)
			}
			return vm.ToValue(9)
		case html.DoctypeNode:
			return vm.ToValue(10)
		}
		return vm.ToValue(0)
	})
	// ns returns the namespace of the element ("svg" or "math" for foreign
	// elements, otherwise "").
	set("ns", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		if n == nil {
			return vm.ToValue("")
		}
		return vm.ToValue(n.Namespace)
	})
	// attrName returns the given name of an attribute of the given node:
	// lowercase for HTML elements, and as it is for foreign (SVG) ones,
	// as the DOM does (as for viewBox).
	attrName := func(n *html.Node, name string) string {
		if n.Namespace != "" {
			return name
		}
		return strings.ToLower(name)
	}
	set("name", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		if n == nil {
			return vm.ToValue("")
		}
		return vm.ToValue(n.Data)
	})
	rel := func(f func(n *html.Node) *html.Node) func(call sobek.FunctionCall) sobek.Value {
		return func(call sobek.FunctionCall) sobek.Value {
			n := node(arg(call, 0))
			if n == nil {
				return vm.ToValue(0)
			}
			return id(f(n))
		}
	}
	set("parent", rel(func(n *html.Node) *html.Node { return n.Parent }))
	set("first", rel(func(n *html.Node) *html.Node { return n.FirstChild }))
	set("last", rel(func(n *html.Node) *html.Node { return n.LastChild }))
	set("next", rel(func(n *html.Node) *html.Node { return n.NextSibling }))
	set("prev", rel(func(n *html.Node) *html.Node { return n.PrevSibling }))
	set("children", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		var ids []any
		if n != nil {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				ids = append(ids, ni.id(c))
			}
		}
		return vm.NewArray(ids...)
	})
	set("attr", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		if n == nil {
			return sobek.Null()
		}
		name := attrName(n, str(call, 1))
		for _, a := range n.Attr {
			if a.Key == name {
				return vm.ToValue(a.Val)
			}
		}
		return sobek.Null()
	})
	set("setAttr", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		if n == nil {
			return sobek.Undefined()
		}
		name, val := attrName(n, str(call, 1)), str(call, 2)
		for i, a := range n.Attr {
			if a.Key == name {
				if a.Val != val {
					n.Attr[i].Val = val
					e.changed(n)
				}
				return sobek.Undefined()
			}
		}
		n.Attr = append(n.Attr, html.Attribute{Key: name, Val: val})
		e.changed(n)
		return sobek.Undefined()
	})
	set("removeAttr", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		if n == nil {
			return sobek.Undefined()
		}
		name := attrName(n, str(call, 1))
		for i, a := range n.Attr {
			if a.Key == name {
				n.Attr = append(n.Attr[:i], n.Attr[i+1:]...)
				e.changed(n)
				break
			}
		}
		return sobek.Undefined()
	})
	set("attrs", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		var kv []any
		if n != nil {
			for _, a := range n.Attr {
				kv = append(kv, a.Key, a.Val)
			}
		}
		return vm.NewArray(kv...)
	})
	set("data", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		if n == nil {
			return vm.ToValue("")
		}
		return vm.ToValue(n.Data)
	})
	set("setData", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		if n != nil && (n.Type == html.TextNode || n.Type == html.CommentNode) {
			if s := str(call, 1); s != n.Data {
				n.Data = s
				e.changed(n)
			}
		}
		return sobek.Undefined()
	})
	set("text", func(call sobek.FunctionCall) sobek.Value {
		return vm.ToValue(textContent(node(arg(call, 0))))
	})
	set("setText", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		if n == nil {
			return sobek.Undefined()
		}
		if n.Type == html.TextNode || n.Type == html.CommentNode {
			n.Data = str(call, 1)
		} else {
			removeChildren(n)
			if s := str(call, 1); s != "" {
				n.AppendChild(&html.Node{Type: html.TextNode, Data: s})
			}
		}
		e.changed(n)
		return sobek.Undefined()
	})
	// create creates an element with the given name, in the given
	// namespace ("svg" or "math", whose names keep their case, as
	// linearGradient, or "" for HTML).
	set("create", func(call sobek.FunctionCall) sobek.Value {
		tag, ns := str(call, 0), str(call, 1)
		if ns != "svg" && ns != "math" {
			ns = ""
			tag = strings.ToLower(tag)
		}
		return id(&html.Node{Type: html.ElementNode, Data: tag, Namespace: ns, DataAtom: atom.Lookup([]byte(strings.ToLower(tag)))})
	})
	set("createText", func(call sobek.FunctionCall) sobek.Value {
		return id(&html.Node{Type: html.TextNode, Data: str(call, 0)})
	})
	set("createComment", func(call sobek.FunctionCall) sobek.Value {
		return id(&html.Node{Type: html.CommentNode, Data: str(call, 0)})
	})
	set("createFragment", func(call sobek.FunctionCall) sobek.Value {
		return id(&html.Node{Type: html.DocumentNode, Data: fragmentData})
	})
	set("insert", func(call sobek.FunctionCall) sobek.Value {
		parent, child, before := node(arg(call, 0)), node(arg(call, 1)), node(arg(call, 2))
		if parent == nil || child == nil {
			return sobek.Undefined()
		}
		if before != nil && before.Parent != parent {
			panic(vm.NewGoError(&domException{name: "NotFoundError", msg: "the node before which to insert is not a child of this node"}))
		}
		for p := parent; p != nil; p = p.Parent {
			if p == child {
				panic(vm.NewGoError(&domException{name: "HierarchyRequestError", msg: "the new child contains the parent"}))
			}
		}
		var kids []*html.Node
		if child.Type == html.DocumentNode && child.Data == fragmentData {
			for c := child.FirstChild; c != nil; c = c.NextSibling {
				kids = append(kids, c)
			}
		} else {
			kids = []*html.Node{child}
		}
		for _, k := range kids {
			if k == before {
				continue
			}
			if k.Parent != nil {
				e.changed(k.Parent)
				k.Parent.RemoveChild(k)
			}
			parent.InsertBefore(k, before)
		}
		e.changed(parent)
		for _, k := range kids {
			e.runInserted(k)
		}
		return sobek.Undefined()
	})
	set("remove", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		if n != nil && n.Parent != nil {
			e.changed(n.Parent)
			n.Parent.RemoveChild(n)
		}
		return sobek.Undefined()
	})
	set("clone", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		if n == nil {
			return vm.ToValue(0)
		}
		return id(cloneNode(n, arg(call, 1).ToBoolean()))
	})
	set("inner", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		var b bytes.Buffer
		if n != nil {
			serializeChildren(&b, n)
		}
		return vm.ToValue(b.String())
	})
	set("outer", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		var b bytes.Buffer
		if n != nil {
			serialize(&b, n)
		}
		return vm.ToValue(b.String())
	})
	set("setInner", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		if n == nil {
			return sobek.Undefined()
		}
		kids := e.parseFragment(str(call, 1), n)
		removeChildren(n)
		for _, k := range kids {
			n.AppendChild(k)
		}
		e.changed(n)
		return sobek.Undefined()
	})
	// parse parses the given HTML in the context of the given element,
	// into a new fragment, whose id it returns.
	set("parse", func(call sobek.FunctionCall) sobek.Value {
		ctx := node(arg(call, 1))
		frag := &html.Node{Type: html.DocumentNode, Data: fragmentData}
		for _, k := range e.parseFragment(str(call, 0), ctx) {
			frag.AppendChild(k)
		}
		return id(frag)
	})
	set("parseDocument", func(call sobek.FunctionCall) sobek.Value {
		doc, err := html.Parse(strings.NewReader(str(call, 0)))
		if err != nil {
			return vm.ToValue(0)
		}
		return id(doc)
	})
	set("query", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		if n == nil {
			return vm.ToValue(0)
		}
		defer scoped(n)()
		sel := compile(scopeSelector(str(call, 1), n))
		var found *html.Node
		walk(n, func(c *html.Node) bool {
			if c != n && c.Type == html.ElementNode && sel.Match(c) {
				found = c
				return false
			}
			return true
		})
		return id(found)
	})
	set("queryAll", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		var ids []any
		if n != nil {
			defer scoped(n)()
			sel := compile(scopeSelector(str(call, 1), n))
			walk(n, func(c *html.Node) bool {
				if c != n && c.Type == html.ElementNode && sel.Match(c) {
					ids = append(ids, ni.id(c))
				}
				return true
			})
		}
		return vm.NewArray(ids...)
	})
	set("matches", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		if n == nil || n.Type != html.ElementNode {
			return vm.ToValue(false)
		}
		defer scoped(n)()
		return vm.ToValue(compile(scopeSelector(str(call, 1), n)).Match(n))
	})
	set("byId", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		want := str(call, 1)
		var found *html.Node
		if n != nil {
			walk(n, func(c *html.Node) bool {
				if c.Type == html.ElementNode {
					for _, a := range c.Attr {
						if a.Key == "id" && a.Val == want {
							found = c
							return false
						}
					}
				}
				return true
			})
		}
		return id(found)
	})
	set("computed", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		if n == nil || e.opts.ComputedStyle == nil {
			return vm.ToValue("")
		}
		return vm.ToValue(e.opts.ComputedStyle(n, str(call, 1)))
	})
	set("rect", func(call sobek.FunctionCall) sobek.Value {
		n := node(arg(call, 0))
		if n == nil || e.opts.Layout == nil {
			return sobek.Null()
		}
		// The page is rendered first if the element may have changed
		// (and otherwise its box as last rendered is current), as
		// rendering a page is costly and scripts get boxes often.
		if e.dirty && e.changedAround(n) {
			e.renderIfDirty(nil, nil) // the lock is held by the task
		}
		x, y, w, h, ok := e.opts.Layout(n)
		if !ok && e.dirty {
			e.renderIfDirty(nil, nil) // as for an element that has no box yet
			x, y, w, h, ok = e.opts.Layout(n)
		}
		if !ok {
			return sobek.Null()
		}
		return vm.NewArray(x, y, w, h)
	})
	set("viewport", func(call sobek.FunctionCall) sobek.Value {
		w, h := float32(1024), float32(768)
		if e.opts.Viewport != nil {
			w, h = e.opts.Viewport()
		}
		return vm.NewArray(w, h)
	})
	set("mediaMatches", func(call sobek.FunctionCall) sobek.Value {
		if e.opts.MediaMatches == nil {
			return vm.ToValue(false)
		}
		return vm.ToValue(e.opts.MediaMatches(str(call, 0)))
	})
	set("log", func(call sobek.FunctionCall) sobek.Value {
		level := slog.LevelInfo
		switch str(call, 0) {
		case "error":
			level = slog.LevelError
		case "warn":
			level = slog.LevelWarn
		case "debug":
			level = slog.LevelDebug
		}
		e.logf(level, "%s", str(call, 1))
		return sobek.Undefined()
	})
	set("setTimer", func(call sobek.FunctionCall) sobek.Value {
		return vm.ToValue(e.setTimer(arg(call, 0).ToFloat(), arg(call, 1).ToBoolean()))
	})
	set("clearTimer", func(call sobek.FunctionCall) sobek.Value {
		e.clearTimer(int(arg(call, 0).ToInteger()))
		return sobek.Undefined()
	})
	set("btoa", func(call sobek.FunctionCall) sobek.Value {
		s := str(call, 0)
		b := make([]byte, 0, len(s))
		for _, r := range s {
			if r > 255 {
				panic(vm.NewGoError(&domException{name: "InvalidCharacterError", msg: "the string has characters outside of Latin1"}))
			}
			b = append(b, byte(r))
		}
		return vm.ToValue(base64.StdEncoding.EncodeToString(b))
	})
	set("atob", func(call sobek.FunctionCall) sobek.Value {
		s := strings.Map(func(r rune) rune {
			if r == ' ' || r == '\n' || r == '\t' || r == '\r' || r == '\f' {
				return -1
			}
			return r
		}, str(call, 0))
		s = strings.TrimRight(s, "=")
		b, err := base64.RawStdEncoding.DecodeString(s)
		if err != nil {
			panic(vm.NewGoError(&domException{name: "InvalidCharacterError", msg: "the string is not valid base64"}))
		}
		rs := make([]rune, len(b))
		for i, c := range b {
			rs[i] = rune(c)
		}
		return vm.ToValue(string(rs))
	})
	// importModule gets the module of the given specifier (relative to the
	// given URL of the importing script, or the page if "") bundled with the modules that it imports, in another goroutine,
	// and then runs it and calls the callback of the given id with its
	// exports (or null and an error).
	set("importModule", func(call sobek.FunctionCall) sobek.Value {
		cb := int(arg(call, 0).ToInteger())
		base := str(call, 2)
		if base == "" {
			base = e.baseURL()
		}
		u, err := e.resolveModule(str(call, 1), base)
		if err != nil {
			e.callBack(cb, nil, err.Error())
			return sobek.Undefined()
		}
		go func() {
			var defs []moduleDef
			var err error
			if !e.moduleDefined(u) {
				defs, err = e.moduleGraph(u, nil, "")
			}
			if err != nil {
				e.callBack(cb, nil, err.Error())
				return
			}
			e.Post(func() {
				e.defineModules(defs)
				// the module is evaluated by the callback (requireModule),
				// so that its errors reject the import
				_, err := e.callback(sobek.Undefined(), vm.ToValue(cb), vm.ToValue(u), sobek.Null())
				e.reportError(err)
			})
		}()
		return sobek.Undefined()
	})
	// lower returns the given code rewritten without the features of
	// JavaScript that sobek does not support, or null if it is not valid.
	set("lower", func(call sobek.FunctionCall) sobek.Value {
		code, err := lower(str(call, 0), "Function")
		if err != nil {
			return sobek.Null()
		}
		return vm.ToValue(strings.TrimRight(strings.TrimSpace(code), ";"))
	})
	// listenerAdded records that a listener for events for which the
	// widgets are set up (as clicks) was added to the given node, so that
	// the page is rendered again.
	set("listenerAdded", func(call sobek.FunctionCall) sobek.Value {
		e.changed(node(arg(call, 0)))
		return sobek.Undefined()
	})
	set("currentScript", func(call sobek.FunctionCall) sobek.Value { return id(e.current) })
	set("loading", func(call sobek.FunctionCall) sobek.Value { return vm.ToValue(e.loading) })
	set("write", func(call sobek.FunctionCall) sobek.Value {
		e.write(str(call, 0))
		return sobek.Undefined()
	})
	e.defineNet()
}

// domException is an error of a DOM operation, which the prelude makes a
// DOMException of its name.
type domException struct {
	name, msg string
}

func (d *domException) Error() string { return d.name + ": " + d.msg }

// walk calls f for n and its descendants in document order, until f
// returns false; it returns false if it stopped.
func walk(n *html.Node, f func(n *html.Node) bool) bool {
	if !f(n) {
		return false
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if !walk(c, f) {
			return false
		}
	}
	return true
}

// textContent returns the text of the given node and its descendants.
func textContent(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.TextNode || n.Type == html.CommentNode {
		return n.Data
	}
	var sb strings.Builder
	walk(n, func(c *html.Node) bool {
		if c.Type == html.TextNode {
			sb.WriteString(c.Data)
		}
		return true
	})
	return sb.String()
}

// removeChildren removes the children of the given node.
func removeChildren(n *html.Node) {
	for c := n.FirstChild; c != nil; c = n.FirstChild {
		n.RemoveChild(c)
	}
}

// cloneNode returns a copy of the given node, with copies of its
// descendants if deep.
func cloneNode(n *html.Node, deep bool) *html.Node {
	c := &html.Node{Type: n.Type, DataAtom: n.DataAtom, Data: n.Data, Namespace: n.Namespace,
		Attr: append([]html.Attribute(nil), n.Attr...)}
	if deep {
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			c.AppendChild(cloneNode(k, true))
		}
	}
	return c
}

// parseFragment parses the given HTML in the context of the given
// element (the body if nil or not an element).
func (e *Engine) parseFragment(s string, ctx *html.Node) []*html.Node {
	if ctx == nil || ctx.Type != html.ElementNode {
		ctx = &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	}
	kids, err := html.ParseFragmentWithOptions(strings.NewReader(s), ctx, html.ParseOptionEnableScripting(true))
	if err != nil {
		return nil
	}
	return kids
}

// scopeAttr is the attribute that marks the element that a selector with
// :scope is relative to (see [scopeSelector]).
const scopeAttr = "data-cc-scope"

// scopeSelector returns the given selector with :scope (which cascadia
// does not support) as the attribute that marks the given node (see
// [scoped]), or :root for a document.
func scopeSelector(sel string, n *html.Node) string {
	if !strings.Contains(sel, ":scope") {
		return sel
	}
	if n.Type != html.ElementNode {
		return strings.ReplaceAll(sel, ":scope", ":root")
	}
	return strings.ReplaceAll(sel, ":scope", "["+scopeAttr+"]")
}

// scoped marks the given element with the attribute of :scope (without
// changing the document for rendering), returning the function that
// removes it.
func scoped(n *html.Node) func() {
	if n.Type != html.ElementNode {
		return func() {}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: scopeAttr})
	return func() {
		for i, a := range n.Attr {
			if a.Key == scopeAttr {
				n.Attr = append(n.Attr[:i], n.Attr[i+1:]...)
				return
			}
		}
	}
}

// closeSelector returns the given selector with the brackets, parentheses
// and strings that are open at its end closed, as CSS does.
func closeSelector(sel string) string {
	var open []byte
	var quote byte
	for i := 0; i < len(sel); i++ {
		c := sel[i]
		switch {
		case quote != 0:
			if c == '\\' {
				i++
			} else if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '[':
			open = append(open, ']')
		case c == '(':
			open = append(open, ')')
		case (c == ']' || c == ')') && len(open) > 0 && open[len(open)-1] == c:
			open = open[:len(open)-1]
		}
	}
	if quote != 0 {
		sel += string(quote)
	}
	for i := len(open) - 1; i >= 0; i-- {
		sel += string(open[i])
	}
	return sel
}

// rawTextElements are the elements whose text is serialized as it is.
var rawTextElements = map[string]bool{"script": true, "style": true, "xmp": true, "iframe": true,
	"noembed": true, "noframes": true, "plaintext": true, "noscript": true}

// voidElements are the elements that have no end tag.
var voidElements = map[string]bool{"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true, "source": true, "track": true,
	"wbr": true, "basefont": true, "bgsound": true, "frame": true, "keygen": true, "param": true}

// serialize writes the HTML of the given node, as browsers serialize it
// (for innerHTML and outerHTML): unlike [html.Render], the text of script
// and style elements and the quotes of text are not escaped.
func serialize(b *bytes.Buffer, n *html.Node) {
	switch n.Type {
	case html.ElementNode:
		b.WriteByte('<')
		b.WriteString(n.Data)
		for _, a := range n.Attr {
			b.WriteByte(' ')
			if a.Namespace != "" {
				b.WriteString(a.Namespace)
				b.WriteByte(':')
			}
			b.WriteString(a.Key)
			b.WriteString(`="`)
			b.WriteString(escapeText(a.Val, true))
			b.WriteByte('"')
		}
		b.WriteByte('>')
		if n.Namespace == "" && voidElements[n.Data] {
			return
		}
		serializeChildren(b, n)
		b.WriteString("</")
		b.WriteString(n.Data)
		b.WriteByte('>')
	case html.TextNode:
		if p := n.Parent; p != nil && p.Type == html.ElementNode && p.Namespace == "" && rawTextElements[p.Data] {
			b.WriteString(n.Data)
		} else {
			b.WriteString(escapeText(n.Data, false))
		}
	case html.CommentNode:
		b.WriteString("<!--")
		b.WriteString(n.Data)
		b.WriteString("-->")
	case html.DoctypeNode:
		b.WriteString("<!DOCTYPE ")
		b.WriteString(n.Data)
		b.WriteByte('>')
	case html.DocumentNode:
		serializeChildren(b, n)
	}
}

// serializeChildren writes the HTML of the children of the given node.
func serializeChildren(b *bytes.Buffer, n *html.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		serialize(b, c)
	}
}

// escapeText escapes the given text (or attribute value) as browsers do
// when they serialize HTML.
func escapeText(s string, attr bool) string {
	if !strings.ContainsAny(s, "&<>\"\u00a0") {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '&':
			b.WriteString("&amp;")
		case r == '\u00a0':
			b.WriteString("&nbsp;")
		case r == '"' && attr:
			b.WriteString("&quot;")
		case (r == '<' || r == '>') && !attr:
			if r == '<' {
				b.WriteString("&lt;")
			} else {
				b.WriteString("&gt;")
			}
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
