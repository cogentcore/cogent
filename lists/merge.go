// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"maps"
	"slices"
	"strings"
)

// The three-way merge of the files of the lists, for syncing: the files
// that are changed since the last sync, the base, both here (ours) and in
// the repository (theirs), are merged by the nodes of their trees, by ID:
// the lists and groups in the tree of lists, the items of each list, and
// the list types. A node has the properties that are changed on either
// side, and for those changed on both, those of the side that is changed
// last. A node that is added on either side is kept. One that is deleted
// on one side is deleted, unless it is changed on the other side. The
// nodes are in the order of the side that changed it, after their parents
// on the other side, which are kept, and are moved to the nearest of
// their ancestors that is kept.

// mergeNode is a node of a tree for a three-way merge.
type mergeNode struct {

	// parent is the ID of the parent of the node, or "" at the top.
	parent string

	// props are the values of the properties of the node by name,
	// as JSON, without those that are not set.
	props map[string]string
}

// equal returns whether the node has the same parent and properties
// as the given one.
func (n *mergeNode) equal(o *mergeNode) bool {
	return n.parent == o.parent && maps.Equal(n.props, o.props)
}

// mergeTree is a tree for a three-way merge.
type mergeTree struct {

	// nodes are the nodes by ID.
	nodes map[string]*mergeNode

	// children are the IDs of the children of the nodes by the ID of the
	// node, in order, with "" for those at the top.
	children map[string][]string
}

func newMergeTree() *mergeTree {
	return &mergeTree{nodes: map[string]*mergeNode{}, children: map[string][]string{}}
}

// add adds a node with the given ID, parent and properties,
// after the other children of the parent.
func (mt *mergeTree) add(id, parent string, props map[string]string) {
	mt.nodes[id] = &mergeNode{parent: parent, props: props}
	mt.children[parent] = append(mt.children[parent], id)
}

// pick returns the merged value of the given values of the base, ours and
// theirs, which is the one that is changed from the base, or for those
// that are changed on both sides, ours if oursWins, and otherwise theirs.
func pick[T comparable](base, ours, theirs T, oursWins bool) T {
	switch {
	case ours == theirs || theirs == base:
		return ours
	case ours == base:
		return theirs
	case oursWins:
		return ours
	}
	return theirs
}

// merge3 returns the three-way merge of the given trees, the base and
// ours and theirs, which are changed from it, with ours for the
// properties that are changed on both sides if oursWins.
func merge3(base, ours, theirs *mergeTree, oursWins bool) *mergeTree {
	nodes := map[string]*mergeNode{}
	lost := map[string]bool{} // the nodes moved by the side that does not win
	ids := slices.Sorted(maps.Keys(ours.nodes))
	for id := range theirs.nodes {
		if ours.nodes[id] == nil {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)

	// the nodes, with their properties
	for _, id := range ids {
		b, o, t := base.nodes[id], ours.nodes[id], theirs.nodes[id]
		if o == nil || t == nil { // added on one side, or deleted on the other
			n := o
			if n == nil {
				n = t
			}
			if b == nil || !n.equal(b) {
				nodes[id] = &mergeNode{parent: n.parent, props: maps.Clone(n.props)}
			}
			continue
		}
		if b == nil {
			b = &mergeNode{}
		}
		m := &mergeNode{parent: pick(b.parent, o.parent, t.parent, oursWins), props: map[string]string{}}
		loser := t.parent
		if !oursWins {
			loser = o.parent
		}
		lost[id] = m.parent != b.parent && m.parent == loser
		for _, props := range []map[string]string{b.props, o.props, t.props} {
			for k := range props {
				if v := pick(b.props[k], o.props[k], t.props[k], oursWins); v != "" {
					m.props[k] = v
				}
			}
		}
		nodes[id] = m
	}
	reparent(nodes, lost, base, ours, theirs)

	// the order of the children of each node
	res := &mergeTree{nodes: nodes, children: map[string][]string{}}
	byParent := map[string][]string{}
	for _, id := range ids {
		if n := nodes[id]; n != nil {
			byParent[n.parent] = append(byParent[n.parent], id)
		}
	}
	for parent, kids := range byParent {
		seq := func(mt *mergeTree) []string {
			return slices.DeleteFunc(slices.Clone(mt.children[parent]), func(id string) bool {
				return nodes[id] == nil || nodes[id].parent != parent
			})
		}
		so, st, sb := seq(ours), seq(theirs), seq(base)
		first, second := st, so
		if slices.Equal(st, sb) || (!slices.Equal(so, sb) && oursWins) {
			first, second = so, st
		}
		order := slices.Clone(first)
		insert := func(src []string) {
			for i, id := range src {
				if slices.Contains(order, id) {
					continue
				}
				at := 0
				if i > 0 {
					at = slices.Index(order, src[i-1]) + 1
				}
				order = slices.Insert(order, at, id)
			}
		}
		insert(second)
		insert(kids) // those that are moved here from removed parents
		res.children[parent] = order
	}
	return res
}

// reparent moves the given merged nodes whose parents are not kept to
// their nearest ancestors that are, in the given trees, and moves to the
// top a node that is in itself, by the moves on both sides, of those
// moved by the side that does not win, the given lost nodes, if any.
func reparent(nodes map[string]*mergeNode, lost map[string]bool, trees ...*mergeTree) {
	parentOf := func(id string) string {
		for _, mt := range trees {
			if n := mt.nodes[id]; n != nil {
				return n.parent
			}
		}
		return ""
	}
	for _, id := range slices.Sorted(maps.Keys(nodes)) {
		n := nodes[id]
		p := n.parent
		for range len(nodes) + 1 {
			if p == "" || nodes[p] != nil {
				break
			}
			p = parentOf(p)
		}
		if p != "" && nodes[p] == nil {
			p = ""
		}
		n.parent = p
	}
	for _, id := range slices.Sorted(maps.Keys(nodes)) {
		var path []string // the ancestors of the node, until one repeats
		seen := map[string]bool{}
		p := id
		for ; p != "" && !seen[p]; p = nodes[p].parent {
			seen[p] = true
			path = append(path, p)
		}
		if p == "" {
			continue
		}
		cycle := path[slices.Index(path, p):]
		slices.Sort(cycle)
		move := cycle[0]
		if i := slices.IndexFunc(cycle, func(id string) bool { return lost[id] }); i >= 0 {
			move = cycle[i]
		}
		nodes[move].parent = ""
	}
}

// mergeBytes returns the merge of the given contents of a file, the base,
// ours and theirs, if they do not need a merge of their nodes: if they are
// the same on both sides, or the same as the base on one side.
func mergeBytes(base, ours, theirs []byte) ([]byte, bool) {
	switch {
	case bytes.Equal(ours, theirs) || bytes.Equal(theirs, base):
		return ours, true
	case bytes.Equal(ours, base):
		return theirs, true
	}
	return nil, false
}

// mergeFiles returns the three-way merge of the given files of the lists by
// path, the base and ours and theirs, which are changed from it, with ours
// for the values that are changed on both sides if oursWins.
func mergeFiles(base, ours, theirs map[string][]byte, oursWins bool) map[string][]byte {
	res := map[string][]byte{}

	// the list types
	if b, ok := mergeBytes(base[typesFile], ours[typesFile], theirs[typesFile]); ok {
		if b != nil {
			res[typesFile] = b
		}
	} else {
		typesTree := func(b []byte) *mergeTree {
			var types []*ListType
			json.Unmarshal(b, &types)
			mt := newMergeTree()
			for _, lt := range types {
				mt.add(lt.Name, "", map[string]string{"type": string(marshal(lt))})
			}
			return mt
		}
		mt := merge3(typesTree(base[typesFile]), typesTree(ours[typesFile]), typesTree(theirs[typesFile]), oursWins)
		var types []*ListType
		for _, id := range mt.children[""] {
			lt := &ListType{}
			json.Unmarshal([]byte(mt.nodes[id].props["type"]), lt)
			types = append(types, lt)
		}
		res[typesFile] = marshal(types)
	}

	// the tree of lists, in which a list that is deleted on one side is
	// kept if its file is changed on the other side
	listsTree := func(files map[string][]byte) (*mergeTree, *treeNode) {
		var root treeNode
		json.Unmarshal(files[treeFile], &root)
		mt := newMergeTree()
		var add func(tn *treeNode, parent string)
		add = func(tn *treeNode, parent string) {
			props := map[string]string{"Name": string(marshal(tn.Name))}
			if tn.Group {
				props["Group"] = "true"
			} else if b, ok := files[listPath(tn.ID)]; ok {
				h := sha1.Sum(b)
				props["file"] = hex.EncodeToString(h[:])
			}
			mt.add(tn.ID, parent, props)
			for _, c := range tn.Lists {
				add(c, tn.ID)
			}
		}
		for _, c := range root.Lists {
			add(c, "")
		}
		return mt, &root
	}
	bt, _ := listsTree(base)
	ot, oroot := listsTree(ours)
	tt, _ := listsTree(theirs)
	mt := merge3(bt, ot, tt, oursWins)
	root := &treeNode{Name: oroot.Name, Group: true}
	var lists []*treeNode // the lists, which have files
	var build func(parent string) []*treeNode
	build = func(parent string) []*treeNode {
		var tns []*treeNode
		for _, id := range mt.children[parent] {
			n := mt.nodes[id]
			tn := &treeNode{ID: id, Group: n.props["Group"] == "true"}
			json.Unmarshal([]byte(n.props["Name"]), &tn.Name)
			tn.Lists = build(id)
			if !tn.Group {
				lists = append(lists, tn)
			}
			tns = append(tns, tn)
		}
		return tns
	}
	root.Lists = build("")
	res[treeFile] = marshal(root)

	// the files of the lists
	for _, tn := range lists {
		path := listPath(tn.ID)
		if b, ok := mergeBytes(base[path], ours[path], theirs[path]); ok {
			if b != nil {
				res[path] = b
			}
			continue
		}
		// a list that is deleted on one side is that of the other
		if ours[path] == nil || theirs[path] == nil {
			res[path] = ours[path]
			if res[path] == nil {
				res[path] = theirs[path]
			}
			continue
		}
		res[path] = marshal(mergeList(base[path], ours[path], theirs[path], tn.Name, oursWins))
	}
	return res
}

// mergeList returns the three-way merge of the given files of a list, the
// base and ours and theirs, with the given name, from the tree of lists.
func mergeList(base, ours, theirs []byte, name string, oursWins bool) *listFile {
	var lfs [3]listFile
	var props [3]map[string]json.RawMessage
	var trees [3]*mergeTree
	for i, b := range [][]byte{base, ours, theirs} {
		json.Unmarshal(b, &lfs[i])
		lfs[i].Name = ""
		items := lfs[i].Items
		lfs[i].Items = nil
		json.Unmarshal(marshal(lfs[i]), &props[i])
		trees[i] = newMergeTree()
		trees[i].addItems(items, "")
	}

	// the settings of the list
	merged := map[string]json.RawMessage{}
	for _, p := range props {
		for k := range p {
			if v := pick(string(props[0][k]), string(props[1][k]), string(props[2][k]), oursWins); v != "" {
				merged[k] = json.RawMessage(v)
			}
		}
	}
	lf := &listFile{}
	json.Unmarshal(marshal(merged), lf)
	lf.Name = name
	lf.Items = merge3(trees[0], trees[1], trees[2], oursWins).items("")
	return lf
}

// valuePrefix is the prefix of the names of the properties of the merge
// nodes of items for their values, after which is the name of the field.
const valuePrefix = "v:"

// addItems adds the given items, and those in them, with the given parent.
func (mt *mergeTree) addItems(items []*Item, parent string) {
	for _, it := range items {
		props := map[string]string{}
		for name, v := range it.Values {
			props[valuePrefix+name] = string(v)
		}
		if it.Closed {
			props["Closed"] = "true"
		}
		mt.add(it.ID, parent, props)
		mt.addItems(it.Items, it.ID)
	}
}

// items returns the items that are the children of the given parent.
func (mt *mergeTree) items(parent string) []*Item {
	var items []*Item
	for _, id := range mt.children[parent] {
		n := mt.nodes[id]
		it := &Item{ID: id, Values: map[string]json.RawMessage{}, Closed: n.props["Closed"] == "true"}
		for k, v := range n.props {
			if name, ok := strings.CutPrefix(k, valuePrefix); ok {
				it.Values[name] = json.RawMessage(v)
			}
		}
		it.Items = mt.items(id)
		items = append(items, it)
	}
	return items
}
