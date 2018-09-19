// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"log"
	"sort"
	"strings"

	"github.com/goki/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// FileNode is Gide version of FileNode for FileTree view
type FileNode struct {
	giv.FileNode
}

var KiT_FileNode = kit.Types.AddType(&FileNode{}, FileNodeProps)

// ViewFile pulls up this file in Gide
func (fn *FileNode) ViewFile() {
	if fn.IsDir() {
		log.Printf("FileNode Edit -- cannot edit directories!\n")
		return
	}
	gek, ok := fn.ParentByType(KiT_Gide, true)
	if ok {
		ge := gek.Embed(KiT_Gide).(*Gide)
		ge.NextViewFileNode(fn.This.Embed(giv.KiT_FileNode).(*giv.FileNode))
	}
}

// ExecCmdFile pops up a menu to select a command appropriate for the given node,
// and shows output in MainTab with name of command
func (fn *FileNode) ExecCmdFile() {
	gek, ok := fn.ParentByType(KiT_Gide, true)
	if ok {
		ge := gek.Embed(KiT_Gide).(*Gide)
		ge.ExecCmdFileNode(fn.This.Embed(giv.KiT_FileNode).(*giv.FileNode))
	}
}

// OpenNodes is a list of file nodes that have been opened for editing -- it
// is maintained in recency order -- most recent on top -- call Add every time
// a node is opened / visited for editing
type OpenNodes []*giv.FileNode

// Add adds given node to list of open nodes -- if already on the list it is
// moved to the top
func (on *OpenNodes) Add(fn *giv.FileNode) {
	sz := len(*on)

	for i, f := range *on {
		if f == fn {
			if i == 0 {
				return
			}
			copy((*on)[1:i+1], (*on)[0:i])
			(*on)[0] = fn
			return
		}
	}

	*on = append(*on, nil)
	if sz > 0 {
		copy((*on)[1:], (*on)[0:sz])
	}
	(*on)[0] = fn
}

// Strings returns a string list of nodes, with paths relative to proj root
func (on *OpenNodes) Strings() []string {
	sl := make([]string, len(*on))
	for i, fn := range *on {
		rp := fn.FRoot.RelPath(fn.FPath)
		rp = strings.TrimSuffix(rp, fn.Nm)
		if rp != "" {
			sl[i] = fn.Nm + " - " + rp
		} else {
			sl[i] = fn.Nm
		}
		if fn.IsChanged() {
			sl[i] += " *"
		}
	}
	return sl
}

// NChanged returns number of changed open files
func (on *OpenNodes) NChanged() int {
	cnt := 0
	for _, fn := range *on {
		if fn.IsChanged() {
			cnt++
		}
	}
	return cnt
}

//////////////////////////////////////////////////////////////////////////
//  Search

// FileSearchResults is used to report search results
type FileSearchResults struct {
	Node    *giv.FileNode
	Count   int
	Matches []giv.TextPos
}

// FileTreeSearch returns list of all nodes starting at given node of given
// language(s) that contain the given string (non regexp version), sorted in
// descending order by number of occurrances -- ignoreCase transforms
// everything into lowercase
func FileTreeSearch(start *giv.FileNode, find string, ignoreCase bool) []FileSearchResults {
	fsz := len(find)
	if fsz == 0 {
		return nil
	}
	mls := make([]FileSearchResults, 0)
	if ignoreCase {
		find = strings.ToLower(find)
	}
	start.FuncDownMeFirst(0, start, func(k ki.Ki, level int, d interface{}) bool {
		sfn := k.Embed(giv.KiT_FileNode).(*giv.FileNode)
		if ignoreCase {
			cnt, matches := giv.FileSearchCI(string(sfn.FPath), []byte(find))
			if cnt > 0 {
				mls = append(mls, FileSearchResults{sfn, cnt, matches})
			}
		} else {
			cnt, matches := giv.FileSearch(string(sfn.FPath), []byte(find))
			if cnt > 0 {
				mls = append(mls, FileSearchResults{sfn, cnt, matches})
			}
		}
		return true
	})
	sort.Slice(mls, func(i, j int) bool {
		return mls[i].Count > mls[j].Count
	})
	return mls
}

var FileNodeProps = ki.Props{
	"CtxtMenu": ki.PropSlice{
		{"ViewFile", ki.Props{
			"label": "View",
			"updtfunc": func(fni interface{}, act *gi.Action) {
				fn := fni.(ki.Ki).Embed(giv.KiT_FileNode).(*giv.FileNode)
				act.SetInactiveStateUpdt(fn.IsDir())
			},
		}},
		{"ExecCmdFile", ki.Props{
			"label": "Exec Cmd…",
		}},
		{"DuplicateFile", ki.Props{
			"label": "Duplicate",
			"updtfunc": func(fni interface{}, act *gi.Action) {
				fn := fni.(ki.Ki).Embed(giv.KiT_FileNode).(*giv.FileNode)
				act.SetInactiveStateUpdt(fn.IsDir())
			},
		}},
		{"DeleteFile", ki.Props{
			"label":   "Delete…",
			"desc":    "Ok to delete this file?  This is not undoable and is not moving to trash / recycle bin",
			"confirm": true,
			"updtfunc": func(fni interface{}, act *gi.Action) {
				fn := fni.(ki.Ki).Embed(giv.KiT_FileNode).(*giv.FileNode)
				act.SetInactiveStateUpdt(fn.IsDir())
			},
		}},
		{"RenameFile", ki.Props{
			"label": "Rename…",
			"desc":  "Rename file to new file name",
			"Args": ki.PropSlice{
				{"New Name", ki.Props{
					"default-field": "Name",
				}},
			},
		}},
		{"sep-open", ki.BlankProp{}},
		{"OpenDir", ki.Props{
			"desc": "open given directory to see files within",
			"updtfunc": func(fni interface{}, act *gi.Action) {
				fn := fni.(ki.Ki).Embed(giv.KiT_FileNode).(*giv.FileNode)
				act.SetActiveStateUpdt(fn.IsDir())
			},
		}},
	},
}
