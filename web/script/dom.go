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
	"github.com/dop251/goja"
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
	}
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
	node := func(v goja.Value) *html.Node { return ni.node(int(v.ToInteger())) }
	id := func(n *html.Node) goja.Value { return vm.ToValue(ni.id(n)) }
	set := func(name string, f func(call goja.FunctionCall) goja.Value) {
		d.Set(name, f)
	}
	arg := func(call goja.FunctionCall, i int) goja.Value { return call.Argument(i) }
	str := func(call goja.FunctionCall, i int) string { return call.Argument(i).String() }
	syntaxError := func(msg string) *goja.Object {
		return vm.NewGoError(&domException{name: "SyntaxError", msg: msg})
	}
	compile := func(sel string) cascadia.Matcher {
		g, err := cascadia.ParseGroup(sel)
		if err != nil {
			panic(syntaxError("'" + sel + "' is not a valid selector"))
		}
		return g
	}

	set("doc", func(call goja.FunctionCall) goja.Value { return id(e.doc) })
	set("type", func(call goja.FunctionCall) goja.Value {
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
	set("name", func(call goja.FunctionCall) goja.Value {
		n := node(arg(call, 0))
		if n == nil {
			return vm.ToValue("")
		}
		return vm.ToValue(n.Data)
	})
	rel := func(f func(n *html.Node) *html.Node) func(call goja.FunctionCall) goja.Value {
		return func(call goja.FunctionCall) goja.Value {
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
	set("children", func(call goja.FunctionCall) goja.Value {
		n := node(arg(call, 0))
		var ids []any
		if n != nil {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				ids = append(ids, ni.id(c))
			}
		}
		return vm.NewArray(ids...)
	})
	set("attr", func(call goja.FunctionCall) goja.Value {
		n := node(arg(call, 0))
		if n == nil {
			return goja.Null()
		}
		name := strings.ToLower(str(call, 1))
		for _, a := range n.Attr {
			if a.Key == name {
				return vm.ToValue(a.Val)
			}
		}
		return goja.Null()
	})
	set("setAttr", func(call goja.FunctionCall) goja.Value {
		n := node(arg(call, 0))
		if n == nil {
			return goja.Undefined()
		}
		name, val := strings.ToLower(str(call, 1)), str(call, 2)
		for i, a := range n.Attr {
			if a.Key == name {
				if a.Val != val {
					n.Attr[i].Val = val
					e.changed(n)
				}
				return goja.Undefined()
			}
		}
		n.Attr = append(n.Attr, html.Attribute{Key: name, Val: val})
		e.changed(n)
		return goja.Undefined()
	})
	set("removeAttr", func(call goja.FunctionCall) goja.Value {
		n := node(arg(call, 0))
		if n == nil {
			return goja.Undefined()
		}
		name := strings.ToLower(str(call, 1))
		for i, a := range n.Attr {
			if a.Key == name {
				n.Attr = append(n.Attr[:i], n.Attr[i+1:]...)
				e.changed(n)
				break
			}
		}
		return goja.Undefined()
	})
	set("attrs", func(call goja.FunctionCall) goja.Value {
		n := node(arg(call, 0))
		var kv []any
		if n != nil {
			for _, a := range n.Attr {
				kv = append(kv, a.Key, a.Val)
			}
		}
		return vm.NewArray(kv...)
	})
	set("data", func(call goja.FunctionCall) goja.Value {
		n := node(arg(call, 0))
		if n == nil {
			return vm.ToValue("")
		}
		return vm.ToValue(n.Data)
	})
	set("setData", func(call goja.FunctionCall) goja.Value {
		n := node(arg(call, 0))
		if n != nil && (n.Type == html.TextNode || n.Type == html.CommentNode) {
			if s := str(call, 1); s != n.Data {
				n.Data = s
				e.changed(n)
			}
		}
		return goja.Undefined()
	})
	set("text", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(textContent(node(arg(call, 0))))
	})
	set("setText", func(call goja.FunctionCall) goja.Value {
		n := node(arg(call, 0))
		if n == nil {
			return goja.Undefined()
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
		return goja.Undefined()
	})
	set("create", func(call goja.FunctionCall) goja.Value {
		tag := strings.ToLower(str(call, 0))
		return id(&html.Node{Type: html.ElementNode, Data: tag, DataAtom: atom.Lookup([]byte(tag))})
	})
	set("createText", func(call goja.FunctionCall) goja.Value {
		return id(&html.Node{Type: html.TextNode, Data: str(call, 0)})
	})
	set("createComment", func(call goja.FunctionCall) goja.Value {
		return id(&html.Node{Type: html.CommentNode, Data: str(call, 0)})
	})
	set("createFragment", func(call goja.FunctionCall) goja.Value {
		return id(&html.Node{Type: html.DocumentNode, Data: fragmentData})
	})
	set("insert", func(call goja.FunctionCall) goja.Value {
		parent, child, before := node(arg(call, 0)), node(arg(call, 1)), node(arg(call, 2))
		if parent == nil || child == nil {
			return goja.Undefined()
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
		return goja.Undefined()
	})
	set("remove", func(call goja.FunctionCall) goja.Value {
		n := node(arg(call, 0))
		if n != nil && n.Parent != nil {
			e.changed(n.Parent)
			n.Parent.RemoveChild(n)
		}
		return goja.Undefined()
	})
	set("clone", func(call goja.FunctionCall) goja.Value {
		n := node(arg(call, 0))
		if n == nil {
			return vm.ToValue(0)
		}
		return id(cloneNode(n, arg(call, 1).ToBoolean()))
	})
	set("inner", func(call goja.FunctionCall) goja.Value {
		n := node(arg(call, 0))
		var b bytes.Buffer
		if n != nil {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				html.Render(&b, c)
			}
		}
		return vm.ToValue(b.String())
	})
	set("outer", func(call goja.FunctionCall) goja.Value {
		n := node(arg(call, 0))
		var b bytes.Buffer
		if n != nil {
			html.Render(&b, n)
		}
		return vm.ToValue(b.String())
	})
	set("setInner", func(call goja.FunctionCall) goja.Value {
		n := node(arg(call, 0))
		if n == nil {
			return goja.Undefined()
		}
		kids := e.parseFragment(str(call, 1), n)
		removeChildren(n)
		for _, k := range kids {
			n.AppendChild(k)
		}
		e.changed(n)
		return goja.Undefined()
	})
	// parse parses the given HTML in the context of the given element,
	// into a new fragment, whose id it returns.
	set("parse", func(call goja.FunctionCall) goja.Value {
		ctx := node(arg(call, 1))
		frag := &html.Node{Type: html.DocumentNode, Data: fragmentData}
		for _, k := range e.parseFragment(str(call, 0), ctx) {
			frag.AppendChild(k)
		}
		return id(frag)
	})
	set("parseDocument", func(call goja.FunctionCall) goja.Value {
		doc, err := html.Parse(strings.NewReader(str(call, 0)))
		if err != nil {
			return vm.ToValue(0)
		}
		return id(doc)
	})
	set("query", func(call goja.FunctionCall) goja.Value {
		n := node(arg(call, 0))
		if n == nil {
			return vm.ToValue(0)
		}
		sel := compile(str(call, 1))
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
	set("queryAll", func(call goja.FunctionCall) goja.Value {
		n := node(arg(call, 0))
		var ids []any
		if n != nil {
			sel := compile(str(call, 1))
			walk(n, func(c *html.Node) bool {
				if c != n && c.Type == html.ElementNode && sel.Match(c) {
					ids = append(ids, ni.id(c))
				}
				return true
			})
		}
		return vm.NewArray(ids...)
	})
	set("matches", func(call goja.FunctionCall) goja.Value {
		n := node(arg(call, 0))
		if n == nil || n.Type != html.ElementNode {
			return vm.ToValue(false)
		}
		return vm.ToValue(compile(str(call, 1)).Match(n))
	})
	set("byId", func(call goja.FunctionCall) goja.Value {
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
	set("rect", func(call goja.FunctionCall) goja.Value {
		n := node(arg(call, 0))
		if n == nil || e.opts.Layout == nil {
			return goja.Null()
		}
		e.renderIfDirty()
		x, y, w, h, ok := e.opts.Layout(n)
		if !ok {
			return goja.Null()
		}
		return vm.NewArray(x, y, w, h)
	})
	set("viewport", func(call goja.FunctionCall) goja.Value {
		w, h := float32(1024), float32(768)
		if e.opts.Viewport != nil {
			w, h = e.opts.Viewport()
		}
		return vm.NewArray(w, h)
	})
	set("mediaMatches", func(call goja.FunctionCall) goja.Value {
		if e.opts.MediaMatches == nil {
			return vm.ToValue(false)
		}
		return vm.ToValue(e.opts.MediaMatches(str(call, 0)))
	})
	set("log", func(call goja.FunctionCall) goja.Value {
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
		return goja.Undefined()
	})
	set("setTimer", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(e.setTimer(arg(call, 0).ToFloat(), arg(call, 1).ToBoolean()))
	})
	set("clearTimer", func(call goja.FunctionCall) goja.Value {
		e.clearTimer(int(arg(call, 0).ToInteger()))
		return goja.Undefined()
	})
	set("btoa", func(call goja.FunctionCall) goja.Value {
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
	set("atob", func(call goja.FunctionCall) goja.Value {
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
	set("currentScript", func(call goja.FunctionCall) goja.Value { return id(e.current) })
	set("loading", func(call goja.FunctionCall) goja.Value { return vm.ToValue(e.loading) })
	set("write", func(call goja.FunctionCall) goja.Value {
		e.write(str(call, 0))
		return goja.Undefined()
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
