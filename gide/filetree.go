// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"goki.dev/fi"
	"goki.dev/gi/v2/filetree"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/texteditor/textbuf"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/icons"
	"goki.dev/ki/v2"
)

// FileNode is Gide version of FileNode for FileTree view
type FileNode struct {
	filetree.Node
}

func (fn *FileNode) OnInit() {
	fn.HandleFileNodeEvents()
	fn.FileNodeStyles()
	fn.CustomContextMenu = fn.GideContextMenu
}

// EditFile pulls up this file in Gide
func (fn *FileNode) EditFile() {
	if fn.IsDir() {
		log.Printf("FileNode Edit -- cannot view (edit) directories!\n")
		return
	}
	ge, ok := ParentGide(fn.This())
	if ok {
		ge.NextViewFileNode(&fn.Node)
	}
}

// SetRunExec sets executable as the RunExec executable that will be run with Run / Debug buttons
func (fn *FileNode) SetRunExec() {
	if !fn.IsExec() {
		log.Printf("FileNode SetRunExec -- only works for executable files!\n")
		return
	}
	ge, ok := ParentGide(fn.This())
	if ok {
		ge.ProjPrefs().RunExec = fn.FPath
		ge.ProjPrefs().BuildDir = gi.FileName(filepath.Dir(string(fn.FPath)))
	}
}

// ExecCmdFile pops up a menu to select a command appropriate for the given node,
// and shows output in MainTab with name of command
func (fn *FileNode) ExecCmdFile() { //gti:add
	ge, ok := ParentGide(fn.This())
	if ok {
		ge.ExecCmdFileNode(&fn.Node)
	} else {
		fmt.Println("no gide!")
	}

}

// ExecCmdNameFile executes given command name on node
func (fn *FileNode) ExecCmdNameFile(cmdNm string) {
	ge, ok := ParentGide(fn.This())
	if ok {
		ge.ExecCmdNameFileNode(&fn.Node, CmdName(cmdNm), true, true)
	}
}

func (fn *FileNode) GideContextMenu(m *gi.Scene) {
	gi.NewButton(m).SetText("Exec Cmd").SetIcon(icons.Terminal).
		SetMenu(CommandMenu(&fn.Node)).Style(func(s *styles.Style) {
		s.SetState(!fn.HasSelection(), states.Disabled)
	})
	giv.NewFuncButton(m, fn.EditFiles).SetText("Edit").SetIcon(icons.Edit).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		})
	giv.NewFuncButton(m, fn.SetRunExecs).SetText("Set Run Exec").SetIcon(icons.PlayArrow).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || !fn.IsExec(), states.Disabled)
		})
	gi.NewSeparator(m)
	fn.FileNodeContextMenu(m)
}

/////////////////////////////////////////////////////////////////////
//   OpenNodes

// OpenNodes is a list of file nodes that have been opened for editing -- it
// is maintained in recency order -- most recent on top -- call Add every time
// a node is opened / visited for editing
type OpenNodes []*filetree.Node

// Add adds given node to list of open nodes -- if already on the list it is
// moved to the top -- returns true if actually added.
// Connects to fn.TextBuf signal and auto-closes when buffer closes.
func (on *OpenNodes) Add(fn *filetree.Node) bool {
	added := on.AddImpl(fn)
	if !added {
		return added
	}
	if fn.Buf != nil {
		// fn.Buf.TextBufSig.Connect(fn.This(), func(recv, send ki.Ki, sig int64, data any) {
		// 	if sig == int64(texteditor.BufClosed) {
		// 		fno, _ := recv.Embed(giv.KiT_FileNode).(*filetree.Node)
		// 		on.Delete(fno)
		// 	}
		// })
	}
	return added
}

// AddImpl adds given node to list of open nodes -- if already on the list it is
// moved to the top -- returns true if actually added.
func (on *OpenNodes) AddImpl(fn *filetree.Node) bool {
	sz := len(*on)

	for i, f := range *on {
		if f == fn {
			if i == 0 {
				return false
			}
			copy((*on)[1:i+1], (*on)[0:i])
			(*on)[0] = fn
			return false
		}
	}

	*on = append(*on, nil)
	if sz > 0 {
		copy((*on)[1:], (*on)[0:sz])
	}
	(*on)[0] = fn
	return true
}

// Delete deletes given node in list of open nodes, returning true if found and deleted
func (on *OpenNodes) Delete(fn *filetree.Node) bool {
	for i, f := range *on {
		if f == fn {
			on.DeleteIdx(i)
			return true
		}
	}
	return false
}

// DeleteIdx deletes at given index
func (on *OpenNodes) DeleteIdx(idx int) {
	*on = append((*on)[:idx], (*on)[idx+1:]...)
}

// DeleteDeleted deletes deleted nodes on list
func (on *OpenNodes) DeleteDeleted() {
	sz := len(*on)
	for i := sz - 1; i >= 0; i-- {
		fn := (*on)[i]
		if fn.This() == nil || fn.FRoot == nil || fn.Is(ki.Deleted) {
			on.DeleteIdx(i)
		}
	}
}

// Strings returns a string list of nodes, with paths relative to proj root
func (on *OpenNodes) Strings() []string {
	on.DeleteDeleted()
	sl := make([]string, len(*on))
	for i, fn := range *on {
		rp := fn.FRoot.RelPath(fn.FPath)
		rp = strings.TrimSuffix(rp, fn.Nm)
		if rp != "" {
			sl[i] = fn.Nm + " - " + rp
		} else {
			sl[i] = fn.Nm
		}
		if fn.IsNotSaved() {
			sl[i] += " *"
		}
	}
	return sl
}

// ByStringName returns the open node with given strings name
func (on *OpenNodes) ByStringName(name string) *filetree.Node {
	sl := on.Strings()
	for i, ns := range sl {
		if ns == name {
			return (*on)[i]
		}
	}
	return nil
}

// NChanged returns number of changed open files
func (on *OpenNodes) NChanged() int {
	cnt := 0
	for _, fn := range *on {
		if fn.IsNotSaved() {
			cnt++
		}
	}
	return cnt
}

//////////////////////////////////////////////////////////////////////////
//  Search

// FileSearchResults is used to report search results
type FileSearchResults struct {
	Node    *filetree.Node
	Count   int
	Matches []textbuf.Match
}

// FileTreeSearch returns list of all nodes starting at given node of given
// language(s) that contain the given string (non regexp version), sorted in
// descending order by number of occurrences -- ignoreCase transforms
// everything into lowercase
func FileTreeSearch(start *filetree.Node, find string, ignoreCase, regExp bool, loc FindLoc, activeDir string, langs []fi.Known) []FileSearchResults {
	fb := []byte(find)
	fsz := len(find)
	if fsz == 0 {
		return nil
	}
	var re *regexp.Regexp
	var err error
	if regExp {
		re, err = regexp.Compile(find)
		if err != nil {
			log.Println(err)
			return nil
		}
	}
	mls := make([]FileSearchResults, 0)
	start.WalkPre(func(k ki.Ki) bool {
		sfn := filetree.AsNode(k)
		if sfn == nil {
			return ki.Continue
		}
		if sfn.IsDir() && !sfn.IsOpen() {
			// fmt.Printf("dir: %v closed\n", sfn.FPath)
			return ki.Break // don't go down into closed directories!
		}
		if sfn.IsDir() || sfn.IsExec() || sfn.Info.Kind == "octet-stream" || sfn.IsAutoSave() {
			// fmt.Printf("dir: %v opened\n", sfn.Nm)
			return ki.Continue
		}
		if int(sfn.Info.Size) > gi.Prefs.Params.BigFileSize {
			return ki.Continue
		}
		if strings.HasSuffix(sfn.Nm, ".gide") { // exclude self
			return ki.Continue
		}
		if !fi.IsMatchList(langs, sfn.Info.Known) {
			return ki.Continue
		}
		if loc == FindLocDir {
			cdir, _ := filepath.Split(string(sfn.FPath))
			if activeDir != cdir {
				return ki.Continue
			}
		} else if loc == FindLocNotTop {
			// if level == 1 { // todo
			// 	return ki.Continue
			// }
		}
		var cnt int
		var matches []textbuf.Match
		if sfn.IsOpen() && sfn.Buf != nil {
			if regExp {
				cnt, matches = sfn.Buf.SearchRegexp(re)
			} else {
				cnt, matches = sfn.Buf.Search(fb, ignoreCase, false)
			}
		} else {
			if regExp {
				cnt, matches = textbuf.SearchFileRegexp(string(sfn.FPath), re)
			} else {
				cnt, matches = textbuf.SearchFile(string(sfn.FPath), fb, ignoreCase)
			}
		}
		if cnt > 0 {
			mls = append(mls, FileSearchResults{sfn, cnt, matches})
		}
		return ki.Continue
	})
	sort.Slice(mls, func(i, j int) bool {
		return mls[i].Count > mls[j].Count
	})
	return mls
}

// EditFiles calls EditFile on selected files
func (fn *FileNode) EditFiles() { //gti:add
	sels := fn.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		sn := sels[i].This().(*FileNode)
		sn.EditFile()
	}
}

// SetRunExecs sets executable as the RunExec executable that will be run with Run / Debug buttons
func (fn *FileNode) SetRunExecs() { //gti:add
	sels := fn.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		sn := sels[i].This().(*FileNode)
		sn.SetRunExec()
	}
}

// RenameFiles calls RenameFile on any selected nodes
func (fn *FileNode) RenameFiles() {
	ge, ok := ParentGide(fn.This())
	if !ok {
		return
	}
	ge.SaveAllCheck(true, func() {
		var nodes []*FileNode
		sels := fn.SelectedViews()
		for i := len(sels) - 1; i >= 0; i-- {
			sn := sels[i].This().(*FileNode)
			nodes = append(nodes, sn)
		}
		ge.CloseOpenNodes(nodes) // close before rename because we are async after this
		for _, sn := range nodes {
			giv.CallFunc(sn, sn.RenameFile)
		}
	})
}
