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
	"io"
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
		ge.NextViewFileNode(fn)
	}
}

// ExecCmdFile pops up a menu to select a command appropriate for the given node,
// and shows output in MainTab with name of command
func (fn *FileNode) ExecCmdFile() {
	gek, ok := fn.ParentByType(KiT_Gide, true)
	if ok {
		ge := gek.Embed(KiT_Gide).(*Gide)
		ge.ExecCmdFileNode(fn)
	}
}

// OpenNodes is a list of file nodes that have been opened for editing -- it
// is maintained in recency order -- most recent on top -- call Add every time
// a node is opened / visited for editing
type OpenNodes []*FileNode

// Add adds given node to list of open nodes -- if already on the list it is
// moved to the top
func (on *OpenNodes) Add(fn *FileNode) {
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
	for i, f := range *on {
		rp := f.FRoot.RelPath(f.FPath)
		rp = strings.TrimSuffix(rp, f.Nm)
		if rp != "" {
			sl[i] = f.Nm + " - " + rp
		} else {
			sl[i] = f.Nm
		}
	}
	return sl
}

//////////////////////////////////////////////////////////////////////////
//  Search

// FileSearch looks for a string (no regexp) within a file, in a
// case-sensitive way, returning number of occurences and specific match
// position list -- column positions are in bytes, not runes...
func FileSearch(filename string, find []byte) (int64, []giv.TextPos) {
	fp, err := os.Open(filename)
	if err != nil {
		log.Printf("gide.FileSearch file open error: %v\n", err)
		return 0, nil
	}
	defer fp.Close()
	return BufSearch(fp, find)
}

// BufSearch looks for a string (no regexp) within a byte buffer, in a
// case-sensitive way, returning number of occurences and specific match
// position list -- column positions are in bytes, not runes...
func BufSearch(reader io.Reader, find []byte) (int64, []giv.TextPos) {
	fsz := len(find)
	if fsz == 0 {
		return 0, nil
	}
	cnt := int64(0)
	var matches []giv.TextPos
	scan := bufio.NewScanner(reader)
	ln := 0
	for scan.Scan() {
		b := scan.Bytes()
		sz := len(b)
		ci := 0
		for ci < sz {
			i := bytes.Index(b[ci:], find)
			if i < 0 {
				break
			}
			i += ci
			ci += fsz
			matches = append(matches, giv.TextPos{ln, i})
			cnt++
		}
	}
	if err := scan.Err(); err != nil {
		log.Printf("gide.FileSearch error: %v\n", err)
	}
	return cnt, matches
}

// FileSearchCI looks for a string (no regexp) within a file, in a
// case-INsensitive way, returning number of occurences -- column positions
// are in bytes, not runes...
func FileSearchCI(filename string, find []byte) (int64, []giv.TextPos) {
	fp, err := os.Open(filename)
	if err != nil {
		log.Printf("gide.FileSearch file open error: %v\n", err)
		return 0, nil
	}
	defer fp.Close()
	return BufSearchCI(fp, find)
}

// BufSearchCI looks for a string (no regexp) within a file, in a
// case-INsensitive way, returning number of occurences -- column positions
// are in bytes, not runes...
func BufSearchCI(reader io.Reader, find []byte) (int64, []giv.TextPos) {
	fsz := len(find)
	if fsz == 0 {
		return 0, nil
	}
	find = bytes.ToLower(find)
	cnt := int64(0)
	var matches []giv.TextPos
	scan := bufio.NewScanner(reader)
	ln := 0
	for scan.Scan() {
		b := bytes.ToLower(scan.Bytes())
		sz := len(b)
		ci := 0
		for ci < sz {
			i := bytes.Index(b[ci:], find)
			if i < 0 {
				break
			}
			i += ci
			ci += fsz
			matches = append(matches, giv.TextPos{ln, i})
			cnt++
		}
	}
	if err := scan.Err(); err != nil {
		log.Printf("gide.FileSearch error: %v\n", err)
	}
	return cnt, matches
}

// FileSearchResults is used to report search results
type FileSearchResults struct {
	Node    *giv.FileNode
	Count   int64
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
			cnt, matches := FileSearchCI(string(sfn.FPath), []byte(find))
			if cnt > 0 {
				mls = append(mls, FileSearchResults{sfn, cnt, matches})
			}
		} else {
			cnt, matches := FileSearch(string(sfn.FPath), []byte(find))
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
			"label": "Exec Cmd",
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
