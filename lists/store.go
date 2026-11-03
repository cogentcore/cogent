// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lists

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/tree"
)

// The files of the lists, by their paths in [Lists.Dir], with slashes, as
// in a repository for syncing: the tree of the lists and groups, the list
// types, and a file for each list, named by its ID, with its items, so
// that the edits of different lists are in different files.
const (
	treeFile  = "tree.json"
	typesFile = "types.json"
	listsDir  = "lists"

	// oldListsFile is the file of all of the lists, with their items,
	// before the files of each list, which is moved to oldListsBackup
	// when it is converted (see [Lists.Open]).
	oldListsFile   = "lists.json"
	oldListsBackup = "lists.json.bak"
)

// listPath returns the path of the file of the list with the given ID.
func listPath(id string) string {
	return listsDir + "/" + id + ".json"
}

// isListsFile returns whether the given path is that of a file of the lists.
func isListsFile(path string) bool {
	if path == treeFile || path == typesFile {
		return true
	}
	name, ok := strings.CutPrefix(path, listsDir+"/")
	return ok && !strings.Contains(name, "/") && strings.HasSuffix(name, ".json")
}

// newID returns a new random ID, for a list or an item.
func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// treeNode is the saved form of a list or group in the tree of lists.
type treeNode struct {
	ID    string      `json:",omitempty"`
	Name  string      `json:",omitempty"`
	Group bool        `json:",omitempty"`
	Lists []*treeNode `json:",omitempty"`
}

// listFile is the saved form of a list, in its file.
type listFile struct {

	// Name is the name of the list, for reading the file,
	// which is that of the tree of lists.
	Name            string             `json:",omitempty"`
	Type            string             `json:",omitempty"`
	ColumnWidths    map[string]float32 `json:",omitempty"`
	Sort            string             `json:",omitempty"`
	HideGroupValues bool               `json:",omitempty"`
	Filtered        bool               `json:",omitempty"`
	Items           []*Item            `json:",omitempty"`
}

// listData is the saved form of a list or group, with the lists in it,
// in the [oldListsFile], before the files of each list, and for the
// standard lists.
type listData struct {
	Name            string
	Type            string             `json:",omitempty"`
	ColumnWidths    map[string]float32 `json:",omitempty"`
	Sort            string             `json:",omitempty"`
	HideGroupValues bool               `json:",omitempty"`
	Filtered        bool               `json:",omitempty"`
	Items           []*Item            `json:",omitempty"`
	Lists           []*listData        `json:",omitempty"`
}

// marshal returns the saved form of the given value, which is the same for
// the same value, as JSON with indents and a final newline, for files
// with diffs of lines.
func marshal(v any) []byte {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err) // the saved forms are always valid JSON
	}
	return append(b, '\n')
}

// Open opens the lists and list types in the given directory, starting
// with the standard list types and an empty list for the files that do
// not exist yet. The lists of the [oldListsFile] are converted to the
// files of each list.
func (ls *Lists) Open(dir string) error {
	ls.Dir = dir
	files, err := readFiles(dir)
	if err != nil {
		return err
	}
	if _, ok := files[treeFile]; !ok {
		var root listData
		err := jsonx.Open(&root, filepath.Join(dir, oldListsFile))
		switch {
		case err == nil:
			ls.loadOld(files, &root)
			if err := ls.Save(); err != nil {
				return err
			}
			if err := os.Rename(filepath.Join(dir, oldListsFile), filepath.Join(dir, oldListsBackup)); err != nil {
				return err
			}
			ls.Current = nil
			ls.Update()
			return nil
		case !errors.Is(err, fs.ErrNotExist):
			return err
		}
	}
	ls.load(files)
	ls.Current = nil
	ls.Update()
	return nil
}

// readFiles returns the files of the lists in the given directory by path,
// for those that exist.
func readFiles(dir string) (map[string][]byte, error) {
	files := map[string][]byte{}
	read := func(path string) error {
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
		if err == nil {
			files[path] = b
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	if err := read(treeFile); err != nil {
		return nil, err
	}
	if err := read(typesFile); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Join(dir, listsDir))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	for _, e := range entries {
		if path := listsDir + "/" + e.Name(); !e.IsDir() && isListsFile(path) {
			if err := read(path); err != nil {
				return nil, err
			}
		}
	}
	return files, nil
}

// writeFiles writes the given files of the lists by path in the given
// directory, those that are changed, and deletes the files of the lists
// that are not in them.
func writeFiles(dir string, files map[string][]byte) error {
	if err := os.MkdirAll(filepath.Join(dir, listsDir), 0o755); err != nil {
		return err
	}
	for _, path := range slices.Sorted(maps.Keys(files)) {
		fn := filepath.Join(dir, filepath.FromSlash(path))
		if old, err := os.ReadFile(fn); err == nil && bytes.Equal(old, files[path]) {
			continue
		}
		if err := os.WriteFile(fn, files[path], 0o644); err != nil {
			return err
		}
	}
	entries, err := os.ReadDir(filepath.Join(dir, listsDir))
	if err != nil {
		return err
	}
	for _, e := range entries {
		if path := listsDir + "/" + e.Name(); isListsFile(path) && files[path] == nil {
			if err := os.Remove(filepath.Join(dir, filepath.FromSlash(path))); err != nil {
				return err
			}
		}
	}
	return nil
}

// Save saves the lists and the list types in [Lists.Dir].
func (ls *Lists) Save() error {
	if ls.Dir == "" || ls.Root == nil {
		return nil
	}
	return writeFiles(ls.Dir, ls.files())
}

// files returns the saved files of the lists by path, with the IDs of
// the lists and items set.
func (ls *Lists) files() map[string][]byte {
	ls.setIDs()
	files := map[string][]byte{typesFile: marshal(ls.Types), treeFile: marshal(ls.Root.treeNode())}
	ls.Root.WalkDown(func(n tree.Node) bool {
		if l := n.(*List); !l.IsGroup() {
			files[listPath(l.ID)] = marshal(l.listFile())
		}
		return tree.Continue
	})
	return files
}

// setIDs sets the IDs of the lists and groups, and of the items of the
// lists, that do not have unique ones, as for new ones, and copies.
func (ls *Lists) setIDs() {
	seen := map[string]bool{}
	ls.Root.WalkDown(func(n tree.Node) bool {
		l := n.(*List)
		if l != ls.Root && (l.ID == "" || seen[l.ID]) {
			l.ID = newID()
		}
		seen[l.ID] = true
		setItemIDs(l.Items, map[string]bool{})
		return tree.Continue
	})
}

// setItemIDs sets the IDs of the given items, and those in them, that are
// not set, or are in the given IDs seen already.
func setItemIDs(items []*Item, seen map[string]bool) {
	for _, it := range items {
		if it.ID == "" || seen[it.ID] {
			it.ID = newID()
		}
		seen[it.ID] = true
		setItemIDs(it.Items, seen)
	}
}

// treeNode returns the saved form of the given list in the tree of lists.
func (ls *List) treeNode() *treeNode {
	tn := &treeNode{ID: ls.ID, Name: ls.Name, Group: ls.Group}
	for _, c := range ls.Children {
		tn.Lists = append(tn.Lists, c.(*List).treeNode())
	}
	return tn
}

// listFile returns the saved form of the given list in its file.
func (ls *List) listFile() *listFile {
	return &listFile{Name: ls.Name, Type: ls.TypeName, ColumnWidths: ls.ColumnWidths, Sort: ls.Sort, HideGroupValues: ls.HideGroupValues, Filtered: ls.Filtered, Items: ls.Items}
}

// load sets the lists and list types from the given files by path,
// starting with the standard list types and an empty list for the
// files that do not exist.
func (ls *Lists) load(files map[string][]byte) {
	ls.loadTypes(files)
	var root treeNode
	if b, ok := files[treeFile]; !ok || json.Unmarshal(b, &root) != nil {
		ls.Root = newList(&listData{Name: "Lists", Lists: []*listData{{Name: "To do", Type: "To do"}}})
	} else {
		ls.Root = ls.listFromFiles(&root, files)
	}
	ls.Root.Group = true
	ls.Root.ID = ""
	ls.setIDs()
	ls.keepUsedTypes()
}

// loadTypes sets the list types from the given files by path,
// which are the standard ones if there are none.
func (ls *Lists) loadTypes(files map[string][]byte) {
	ls.Types = nil
	if b, ok := files[typesFile]; ok {
		json.Unmarshal(b, &ls.Types)
	}
	if len(ls.Types) == 0 {
		ls.Types = standardTypes()
	}
}

// loadOld sets the lists from the given saved lists of the [oldListsFile],
// and the list types from the given files by path.
func (ls *Lists) loadOld(files map[string][]byte, root *listData) {
	ls.loadTypes(files)
	ls.Root = newList(root)
	ls.Root.Group = true
	ls.setIDs()
	ls.keepUsedTypes()
}

// listFromFiles returns a new list made from the given node of the tree of
// lists, with the given files by path. A list without a file has the
// first list type.
func (ls *Lists) listFromFiles(tn *treeNode, files map[string][]byte) *List {
	l := tree.New[List]()
	l.SetName(tn.Name)
	l.ID = tn.ID
	l.Group = tn.Group
	if !l.Group {
		var lf listFile
		if b, ok := files[listPath(tn.ID)]; ok {
			json.Unmarshal(b, &lf)
		}
		l.TypeName = lf.Type
		if l.TypeName == "" {
			l.TypeName = ls.Types[0].Name
		}
		l.ColumnWidths = lf.ColumnWidths
		l.Sort = lf.Sort
		l.HideGroupValues = lf.HideGroupValues
		l.Filtered = lf.Filtered
		l.Items = lf.Items
	}
	for _, c := range tn.Lists {
		cl := ls.listFromFiles(c, files)
		l.AddChild(cl)
		setUniqueName(cl, cl.Name)
	}
	return l
}

// newList returns a new list made from the given saved list.
func newList(ld *listData) *List {
	l := tree.New[List]()
	l.SetName(ld.Name)
	l.Group = ld.Type == ""
	l.TypeName = ld.Type
	l.ColumnWidths = ld.ColumnWidths
	l.Sort = ld.Sort
	l.HideGroupValues = ld.HideGroupValues
	l.Filtered = ld.Filtered
	l.Items = ld.Items
	for _, c := range ld.Lists {
		cl := newList(c)
		l.AddChild(cl)
		setUniqueName(cl, cl.Name)
	}
	return l
}

// listData returns the saved form of the given list, with the lists in it.
func (ls *List) listData() *listData {
	ld := &listData{Name: ls.Name, Type: ls.TypeName, ColumnWidths: ls.ColumnWidths, Sort: ls.Sort, HideGroupValues: ls.HideGroupValues, Filtered: ls.Filtered, Items: ls.Items}
	for _, c := range ls.Children {
		ld.Lists = append(ld.Lists, c.(*List).listData())
	}
	return ld
}
