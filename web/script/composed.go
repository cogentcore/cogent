// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"sync"

	"golang.org/x/net/html"
)

// The page is rendered as browsers render it, by its composed tree: the
// content of the shadow root of each element that has one is in its place
// (instead of its children), with the children that are assigned to each of
// its slots in the slot (instead of the fallback content of the slot). As
// the page is read into widgets (in a task, with no script running), the
// nodes of the document are linked as the composed tree (see
// [Engine.compose]), and then linked back as they were, so that the reader
// of pages (htmlcore) reads, styles and reads again the composed tree with
// the nodes of the document (as the posts of reddit.com, whose votes and
// actions are in shadow roots, which were not rendered).

// composition is the state of the composed tree of the last render.
type composition struct {
	mu sync.Mutex

	// scope has the host of the shadow root that each node of a shadow
	// tree is in (for the style sheets of shadow roots, which apply only to
	// them; see [Engine.ShadowScope]).
	scope map[*html.Node]*html.Node

	// parent has the parent of each node in the composed tree, where it is
	// not that of the document (see [Engine.ComposedParent]).
	parent map[*html.Node]*html.Node
}

// links are the links of a node, saved to restore them.
type links struct {
	n                               *html.Node
	parent, first, last, prev, next *html.Node
}

// save returns the links of the given node.
func save(n *html.Node) links {
	return links{n, n.Parent, n.FirstChild, n.LastChild, n.PrevSibling, n.NextSibling}
}

// restore links the node as it was.
func (l links) restore() {
	l.n.Parent, l.n.FirstChild, l.n.LastChild, l.n.PrevSibling, l.n.NextSibling = l.parent, l.first, l.last, l.prev, l.next
}

// compose links the nodes of the document as its composed tree, if it has
// any shadow roots, and returns the function that links them back as they
// were. It must be called in a task, as no script may run meanwhile.
func (e *Engine) compose() func() {
	if len(e.shadowRoots) == 0 {
		return func() {}
	}
	type assigned struct {
		host, root *html.Node
		slots      map[*html.Node][]*html.Node // the children assigned to each slot
	}
	// the assignments of the children of the hosts to the slots of their
	// shadow roots, and the scopes of the nodes, from the document as it is
	scope := map[*html.Node]*html.Node{}
	var as []assigned
	for host, root := range e.shadowRoots {
		for n := range root.Descendants() {
			scope[n] = host
		}
		a := assigned{host: host, root: root, slots: map[*html.Node][]*html.Node{}}
		for k := host.FirstChild; k != nil; k = k.NextSibling {
			name := ""
			switch k.Type {
			case html.ElementNode:
				name, _ = attr(k, "slot")
			case html.TextNode:
			default:
				continue
			}
			if s := firstSlot(root, name); s != nil {
				a.slots[s] = append(a.slots[s], k)
			}
		}
		as = append(as, a)
	}
	var saved []links
	parents := map[*html.Node]*html.Node{}
	for _, a := range as {
		saved = append(saved, save(a.host))
		for k := a.host.FirstChild; k != nil; k = k.NextSibling {
			saved = append(saved, save(k))
		}
		for k := a.root.FirstChild; k != nil; k = k.NextSibling {
			saved = append(saved, save(k))
		}
		// the content of the shadow root is in place of the children
		a.host.FirstChild, a.host.LastChild = a.root.FirstChild, a.root.LastChild
		for k := a.root.FirstChild; k != nil; k = k.NextSibling {
			k.Parent = a.host
			parents[k] = a.host
		}
		// the children assigned to each slot are in place of its fallback
		for s, ks := range a.slots {
			saved = append(saved, save(s))
			for _, k := range ks {
				saved = append(saved, save(k))
			}
			s.FirstChild, s.LastChild = ks[0], ks[len(ks)-1]
			for i, k := range ks {
				k.Parent = s
				k.PrevSibling, k.NextSibling = nil, nil
				if i > 0 {
					k.PrevSibling = ks[i-1]
				}
				if i < len(ks)-1 {
					k.NextSibling = ks[i+1]
				}
				parents[k] = s
			}
		}
	}
	e.composing = true
	e.composed.mu.Lock()
	e.composed.scope, e.composed.parent = scope, parents
	e.composed.mu.Unlock()
	return func() {
		e.composing = false
		for i := len(saved) - 1; i >= 0; i-- {
			saved[i].restore()
		}
	}
}

// firstSlot returns the first slot element of the given shadow tree of the
// given name ("" for the default slot), in tree order, or nil.
func firstSlot(root *html.Node, name string) *html.Node {
	for n := range root.Descendants() {
		if n.Type == html.ElementNode && n.Data == "slot" && slotName(n) == name {
			return n
		}
	}
	return nil
}

// ShadowScope returns the host of the shadow root whose tree the given node
// is in, or nil if it is in the document (not in a shadow tree), as of the
// last render of the page. The style sheets of a shadow tree apply only to
// it, and those of the document not to it.
func (e *Engine) ShadowScope(n *html.Node) *html.Node {
	e.composed.mu.Lock()
	defer e.composed.mu.Unlock()
	return e.composed.scope[n]
}

// ComposedParent returns the parent of the given node in the composed tree
// of the page (see [Engine.compose]), as of its last render: the host of a
// shadow root for a top-level node of it, or the slot of a node that is
// assigned to one, or else its parent in the document.
func (e *Engine) ComposedParent(n *html.Node) *html.Node {
	e.composed.mu.Lock()
	p, ok := e.composed.parent[n]
	e.composed.mu.Unlock()
	if ok {
		return p
	}
	return n.Parent
}

// slotName returns the name of the given slot ("" for the default slot).
func slotName(n *html.Node) string {
	name, _ := attr(n, "name")
	return name
}
