// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
package gide provides the core Gide editor object.

Derived classes can extend the functionality for specific domains.

*/
package gide

import (
	"bufio"
	"bytes"
	"log"
	"os"
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
		ge.ViewFileNode(fn)
	}
}

// ExecCmdFile executes a command on the file
func (fn *FileNode) ExecCmdFile(cmdNm CmdName) {
	gek, ok := fn.ParentByType(KiT_Gide, true)
	if ok {
		ge := gek.Embed(KiT_Gide).(*Gide)
		ge.ExecCmdFileNode(cmdNm, fn)
	}
}

// FileSearch looks for a string (no regexp) within a file, in a
// case-sensitive way, returning number of occurances. adapted from:
// https://stackoverflow.com/questions/26709971/could-this-be-more-efficient-in-go
func FileSearch(filename string, pat []byte) int64 {
	cnt := int64(0)
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if bytes.Contains(scanner.Bytes(), pat) {
			cnt++
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("gide.FileSearch error: %v\n", err)
	}
	return cnt
}

// FileSearchCI looks for a string (no regexp) within a file, in a
// case-INsensitive way, returning number of occurances. adapted from:
// https://stackoverflow.com/questions/26709971/could-this-be-more-efficient-in-go
func FileSearchCI(filename string, pat []byte) int64 {
	pat = bytes.ToLower(pat)
	cnt := int64(0)
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f) // line at a time
	for scanner.Scan() {
		lcb := bytes.ToLower(scanner.Bytes())
		if bytes.Contains(lcb, pat) {
			cnt++
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("gide.FileSearchCI error: %v\n", err)
	}
	return cnt
}

// FileNodeCount is used to report counts by file node
type FileNodeCount struct {
	Node  *giv.FileNode
	Count int64
}

// FileTreeSearch returns list of all nodes starting at given node of given
// language(s) that contain the given string (non regexp version), sorted in
// descending order by number of occurrances -- ignoreCase transforms
// everything into lowercase
func FileTreeSearch(start *giv.FileNode, match string, ignoreCase bool) []FileNodeCount {
	mls := make([]FileNodeCount, 0)
	if ignoreCase {
		match = strings.ToLower(match)
	}
	start.FuncDownMeFirst(0, start, func(k ki.Ki, level int, d interface{}) bool {
		sfn := k.Embed(giv.KiT_FileNode).(*giv.FileNode)
		if ignoreCase {
			cnt := FileSearchCI(string(sfn.FPath), []byte(match))
			if cnt > 0 {
				mls = append(mls, FileNodeCount{sfn, cnt})
			}
		} else {
			cnt := FileSearch(string(sfn.FPath), []byte(match))
			if cnt > 0 {
				mls = append(mls, FileNodeCount{sfn, cnt})
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
			"label": "Exec Cmd",
			"Args": ki.PropSlice{
				{"Command", ki.Props{}},
			},
		}},
		{"DuplicateFile", ki.Props{
			"label": "Duplicate",
			"updtfunc": func(fni interface{}, act *gi.Action) {
				fn := fni.(ki.Ki).Embed(giv.KiT_FileNode).(*giv.FileNode)
				act.SetInactiveStateUpdt(fn.IsDir())
			},
		}},
		{"DeleteFile", ki.Props{
			"label":   "Delete",
			"desc":    "Ok to delete this file?  This is not undoable and is not moving to trash / recycle bin",
			"confirm": true,
			"updtfunc": func(fni interface{}, act *gi.Action) {
				fn := fni.(ki.Ki).Embed(giv.KiT_FileNode).(*giv.FileNode)
				act.SetInactiveStateUpdt(fn.IsDir())
			},
		}},
		{"RenameFile", ki.Props{
			"label": "Rename",
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
