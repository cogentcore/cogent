// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/fileinfo"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/texteditor/textbuf"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/views"
)

// FileNode is Code version of FileNode for FileTree view
type FileNode struct {
	filetree.Node
}

func (fn *FileNode) OnInit() {
	fn.Node.OnInit()
	fn.AddContextMenu(fn.ContextMenu)
}

func (fn *FileNode) OnDoubleClick(e events.Event) {
	e.SetHandled()
	ge, ok := ParentCode(fn.This())
	if !ok {
		return
	}
	sels := fn.SelectedViews()
	if len(sels) > 0 {
		sn := filetree.AsNode(sels[len(sels)-1])
		if sn != nil {
			if sn.IsDir() {
				if !sn.HasChildren() {
					sn.OpenEmptyDir()
				} else {
					sn.ToggleClose()
				}
			} else {
				ge.FileNodeOpened(sn)
			}
		}
	}
}

// EditFile pulls up this file in Code
func (fn *FileNode) EditFile() {
	if fn.IsDir() {
		log.Printf("FileNode Edit -- cannot view (edit) directories!\n")
		return
	}
	ge, ok := ParentCode(fn.This())
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
	ge, ok := ParentCode(fn.This())
	if ok {
		ge.ProjSettings().RunExec = fn.FPath
		ge.ProjSettings().BuildDir = core.Filename(filepath.Dir(string(fn.FPath)))
	}
}

// ExecCmdFile pops up a menu to select a command appropriate for the given node,
// and shows output in MainTab with name of command
func (fn *FileNode) ExecCmdFile() { //gti:add
	ge, ok := ParentCode(fn.This())
	if ok {
		ge.ExecCmdFileNode(&fn.Node)
	} else {
		fmt.Println("no code!")
	}

}

// ExecCmdNameFile executes given command name on node
func (fn *FileNode) ExecCmdNameFile(cmdNm string) {
	ge, ok := ParentCode(fn.This())
	if ok {
		ge.ExecCmdNameFileNode(&fn.Node, CmdName(cmdNm), true, true)
	}
}

func (fn *FileNode) ContextMenu(m *core.Scene) {
	core.NewButton(m).SetText("Exec Cmd").SetIcon(icons.Terminal).
		SetMenu(CommandMenu(&fn.Node)).Style(func(s *styles.Style) {
		s.SetState(!fn.HasSelection(), states.Disabled)
	})
	views.NewFuncButton(m, fn.EditFiles).SetText("Edit").SetIcon(icons.Edit).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		})
	views.NewFuncButton(m, fn.SetRunExecs).SetText("Set Run Exec").SetIcon(icons.PlayArrow).
		Style(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || !fn.IsExec(), states.Disabled)
		})
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
	if fn.Buffer != nil {
		// fn.Buf.TextBufSig.Connect(fn.This(), func(recv, send tree.Node, sig int64, data any) {
		// 	if sig == int64(texteditor.BufClosed) {
		// 		fno, _ := recv.Embed(views.KiT_FileNode).(*filetree.Node)
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
			on.DeleteIndex(i)
			return true
		}
	}
	return false
}

// DeleteIndex deletes at given index
func (on *OpenNodes) DeleteIndex(idx int) {
	*on = append((*on)[:idx], (*on)[idx+1:]...)
}

// DeleteDeleted deletes deleted nodes on list
func (on *OpenNodes) DeleteDeleted() {
	sz := len(*on)
	for i := sz - 1; i >= 0; i-- {
		fn := (*on)[i]
		if fn.This() == nil || fn.FRoot == nil {
			on.DeleteIndex(i)
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

// FindPath finds node for given path, nil if not found
func (on *OpenNodes) FindPath(path string) *filetree.Node {
	for _, f := range *on {
		if f.FPath == core.Filename(path) {
			return f
		}
	}
	return nil
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
// language(s) that contain the given string, sorted in descending order by number
// of occurrences -- ignoreCase transforms everything into lowercase
func FileTreeSearch(ge Code, start *filetree.Node, find string, ignoreCase, regExp bool, loc FindLoc, activeDir string, langs []fileinfo.Known) []FileSearchResults {
	fb := []byte(find)
	fsz := len(find)
	if fsz == 0 {
		return nil
	}
	if loc == FindLocAll {
		return FindAll(ge, start, find, ignoreCase, regExp, langs)
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
	start.WalkDown(func(k tree.Node) bool {
		sfn := filetree.AsNode(k)
		if sfn == nil {
			return tree.Continue
		}
		if sfn.IsDir() && !sfn.IsOpen() {
			// fmt.Printf("dir: %v closed\n", sfn.FPath)
			return tree.Break // don't go down into closed directories!
		}
		if sfn.IsDir() || sfn.IsExec() || sfn.Info.Kind == "octet-stream" || sfn.IsAutoSave() {
			// fmt.Printf("dir: %v opened\n", sfn.Nm)
			return tree.Continue
		}
		if int(sfn.Info.Size) > core.SystemSettings.BigFileSize {
			return tree.Continue
		}
		if strings.HasSuffix(sfn.Nm, ".code") { // exclude self
			return tree.Continue
		}
		if !fileinfo.IsMatchList(langs, sfn.Info.Known) {
			return tree.Continue
		}
		if loc == FindLocDir {
			cdir, _ := filepath.Split(string(sfn.FPath))
			if activeDir != cdir {
				return tree.Continue
			}
		} else if loc == FindLocNotTop {
			// if level == 1 { // todo
			// 	return tree.Continue
			// }
		}
		var cnt int
		var matches []textbuf.Match
		if sfn.IsOpen() && sfn.Buffer != nil {
			if regExp {
				cnt, matches = sfn.Buffer.SearchRegexp(re)
			} else {
				cnt, matches = sfn.Buffer.Search(fb, ignoreCase, false)
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
		return tree.Continue
	})
	sort.Slice(mls, func(i, j int) bool {
		return mls[i].Count > mls[j].Count
	})
	return mls
}

// FindAll returns list of all files (regardless of what is currently open)
// starting at given node of given language(s) that contain the given string,
// sorted in descending order by number of occurrences. ignoreCase transforms
// everything into lowercase.
func FindAll(ge Code, start *filetree.Node, find string, ignoreCase, regExp bool, langs []fileinfo.Known) []FileSearchResults {
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
	spath := string(start.FPath) // note: is already Abs
	filepath.Walk(spath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == ".git" {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		if int(info.Size()) > core.SystemSettings.BigFileSize {
			return nil
		}
		if strings.HasSuffix(info.Name(), ".code") { // exclude self
			return nil
		}
		if len(langs) > 0 {
			mtyp, _, err := fileinfo.MimeFromFile(path)
			if err != nil {
				return nil
			}
			known := fileinfo.MimeKnown(mtyp)
			if !fileinfo.IsMatchList(langs, known) {
				return nil
			}
		}
		ofn := ge.CurOpenNodes().FindPath(path)
		var cnt int
		var matches []textbuf.Match
		if ofn != nil && ofn.Buffer != nil {
			if regExp {
				cnt, matches = ofn.Buffer.SearchRegexp(re)
			} else {
				cnt, matches = ofn.Buffer.Search(fb, ignoreCase, false)
			}
		} else {
			if regExp {
				cnt, matches = textbuf.SearchFileRegexp(path, re)
			} else {
				cnt, matches = textbuf.SearchFile(path, fb, ignoreCase)
			}
		}
		if cnt > 0 {
			if ofn != nil {
				mls = append(mls, FileSearchResults{ofn, cnt, matches})
			} else {
				sfn, found := start.FindFile(path)
				if found {
					mls = append(mls, FileSearchResults{sfn, cnt, matches})
				} else {
					fmt.Println("file not found in FindFile:", path)
				}
			}
		}
		return nil
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
	ge, ok := ParentCode(fn.This())
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
			views.CallFunc(sn, sn.RenameFile)
		}
	})
}
