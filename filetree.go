// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"image/color"
	"log"
	"sort"
	"strings"

	"github.com/goki/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/units"
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

// ExecCmdNameFile executes given command name on node
func (fn *FileNode) ExecCmdNameFile(cmdNm string) {
	gek, ok := fn.ParentByType(KiT_Gide, true)
	if ok {
		ge := gek.Embed(KiT_Gide).(*Gide)
		ge.ExecCmdNameFileNode(fn.This.Embed(giv.KiT_FileNode).(*giv.FileNode), CmdName(cmdNm), true, true)
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

// Delete deletes given node in list of open nodes, returning true if found and deleted
func (on *OpenNodes) Delete(fn *giv.FileNode) bool {
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
	Matches []giv.FileSearchMatch
}

// FileTreeSearch returns list of all nodes starting at given node of given
// language(s) that contain the given string (non regexp version), sorted in
// descending order by number of occurrances -- ignoreCase transforms
// everything into lowercase
func FileTreeSearch(start *giv.FileNode, find string, ignoreCase bool, langs LangNames) []FileSearchResults {
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
		if sfn.IsDir() && !sfn.IsOpen() {
			return false // don't go down into closed directories!
		}
		if sfn.IsDir() || sfn.IsExec() || sfn.Info.Kind == "octet-stream" || sfn.IsAutoSave() {
			return true
		}
		if !LangNamesMatchFilename(sfn.Nm, langs) {
			return true
		}
		var cnt int
		var matches []giv.FileSearchMatch
		if sfn.IsOpen() && sfn.Buf != nil {
			cnt, matches = sfn.Buf.Search([]byte(find), ignoreCase)
		} else {
			cnt, matches = giv.FileSearch(string(sfn.FPath), []byte(find), ignoreCase)
		}
		if cnt > 0 {
			mls = append(mls, FileSearchResults{sfn, cnt, matches})
		}
		return true
	})
	sort.Slice(mls, func(i, j int) bool {
		return mls[i].Count > mls[j].Count
	})
	return mls
}

var FileNodeProps = ki.Props{
	"CallMethods": ki.PropSlice{
		{"RenameFile", ki.Props{
			"label": "Rename",
			"desc":  "Rename file to new file name",
			"Args": ki.PropSlice{
				{"New Name", ki.Props{
					"default-field": "Name",
					"width":         60,
				}},
			},
		}},
	},
}

/////////////////////////////////////////////////////////////////////////
// FileTreeView is the Gide version of the FileTreeView

// FileTreeView is a TreeView that knows how to operate on FileNode nodes
type FileTreeView struct {
	giv.FileTreeView
}

var KiT_FileTreeView = kit.Types.AddType(&FileTreeView{}, nil)

func init() {
	kit.Types.SetProps(KiT_FileTreeView, FileTreeViewProps)
}

// FileNode returns the SrcNode as a *gide* FileNode
func (ft *FileTreeView) FileNode() *FileNode {
	fn := ft.SrcNode.Ptr.Embed(KiT_FileNode)
	if fn == nil {
		return nil
	}
	return fn.(*FileNode)
}

// ViewFiles calls ViewFile on selected files
func (ft *FileTreeView) ViewFiles() {
	sels := ft.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		sn := sels[i]
		ftv := sn.Embed(KiT_FileTreeView).(*FileTreeView)
		fn := ftv.FileNode()
		if fn != nil {
			fn.ViewFile()
		}
	}
}

// FileTreeViewExecCmds gets list of available commands for given file node, as a submenu-func
func FileTreeViewExecCmds(it interface{}, vp *gi.Viewport2D) []string {
	ft, ok := it.(ki.Ki).Embed(KiT_FileTreeView).(*FileTreeView)
	if !ok {
		return nil
	}
	gek, ok := ft.ParentByType(KiT_Gide, true)
	if !ok {
		return nil
	}
	ge := gek.Embed(KiT_Gide).(*Gide)
	fnm := ft.SrcNode.Ptr.Name()
	langs := LangNamesForFilename(fnm)
	cmds := AvailCmds.FilterCmdNames(langs, ge.Prefs.VersCtrl)
	return cmds
}

// ExecCmdFiles calls given command on selected files
func (ft *FileTreeView) ExecCmdFiles(cmdNm string) {
	sels := ft.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		sn := sels[i]
		ftv := sn.Embed(KiT_FileTreeView).(*FileTreeView)
		fn := ftv.FileNode()
		if fn != nil {
			fn.ExecCmdNameFile(cmdNm)
		}
	}
}

var FileTreeViewProps = ki.Props{
	"indent":           units.NewValue(2, units.Ch),
	"spacing":          units.NewValue(.5, units.Ch),
	"border-width":     units.NewValue(0, units.Px),
	"border-radius":    units.NewValue(0, units.Px),
	"padding":          units.NewValue(0, units.Px),
	"margin":           units.NewValue(1, units.Px),
	"text-align":       gi.AlignLeft,
	"vertical-align":   gi.AlignTop,
	"color":            &gi.Prefs.Colors.Font,
	"background-color": "inherit",
	".exec": ki.Props{
		"font-weight": gi.WeightBold,
	},
	".open": ki.Props{
		"font-style": gi.FontItalic,
	},
	"#icon": ki.Props{
		"width":   units.NewValue(1, units.Em),
		"height":  units.NewValue(1, units.Em),
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
		"fill":    &gi.Prefs.Colors.Icon,
		"stroke":  &gi.Prefs.Colors.Font,
	},
	"#branch": ki.Props{
		"icon":             "widget-wedge-down",
		"icon-off":         "widget-wedge-right",
		"margin":           units.NewValue(0, units.Px),
		"padding":          units.NewValue(0, units.Px),
		"background-color": color.Transparent,
		"max-width":        units.NewValue(.8, units.Em),
		"max-height":       units.NewValue(.8, units.Em),
	},
	"#space": ki.Props{
		"width": units.NewValue(.5, units.Em),
	},
	"#label": ki.Props{
		"margin":    units.NewValue(0, units.Px),
		"padding":   units.NewValue(0, units.Px),
		"min-width": units.NewValue(16, units.Ch),
	},
	"#menu": ki.Props{
		"indicator": "none",
	},
	giv.TreeViewSelectors[giv.TreeViewActive]: ki.Props{},
	giv.TreeViewSelectors[giv.TreeViewSel]: ki.Props{
		"background-color": &gi.Prefs.Colors.Select,
	},
	giv.TreeViewSelectors[giv.TreeViewFocus]: ki.Props{
		"background-color": &gi.Prefs.Colors.Control,
	},
	"CtxtMenuActive": ki.PropSlice{
		{"ViewFiles", ki.Props{
			"label": "View",
			"updtfunc": func(fni interface{}, act *gi.Action) {
				ft := fni.(ki.Ki).Embed(KiT_FileTreeView).(*FileTreeView)
				fn := ft.FileNode()
				if fn != nil {
					act.SetInactiveStateUpdt(fn.IsDir())
				}
			},
		}},
		{"ExecCmdFiles", ki.Props{
			"label":        "Exec Cmd",
			"submenu-func": giv.SubMenuFunc(FileTreeViewExecCmds),
			"Args": ki.PropSlice{
				{"Cmd Name", ki.Props{}},
			},
		}},
		{"DuplicateFiles", ki.Props{
			"label": "Duplicate",
			"updtfunc": func(fni interface{}, act *gi.Action) {
				ft := fni.(ki.Ki).Embed(KiT_FileTreeView).(*FileTreeView)
				fn := ft.FileNode()
				if fn != nil {
					act.SetInactiveStateUpdt(fn.IsDir())
				}
			},
		}},
		{"DeleteFiles", ki.Props{
			"label":   "Delete",
			"desc":    "Ok to delete file(s)?  This is not undoable and is not moving to trash / recycle bin",
			"confirm": true,
			"updtfunc": func(fni interface{}, act *gi.Action) {
				ft := fni.(ki.Ki).Embed(KiT_FileTreeView).(*FileTreeView)
				fn := ft.FileNode()
				if fn != nil {
					act.SetInactiveStateUpdt(fn.IsDir())
				}
			},
		}},
		{"RenameFiles", ki.Props{
			"label": "Rename",
			"desc":  "Rename file to new file name",
		}},
		{"sep-open", ki.BlankProp{}},
		{"OpenDirs", ki.Props{
			"label": "Open Dir",
			"desc":  "open given folder to see files within",
			"updtfunc": func(fni interface{}, act *gi.Action) {
				ft := fni.(ki.Ki).Embed(KiT_FileTreeView).(*FileTreeView)
				fn := ft.FileNode()
				if fn != nil {
					act.SetActiveStateUpdt(fn.IsDir())
				}
			},
		}},
		{"NewFile", ki.Props{
			"label": "New File...",
			"desc":  "make a new file in this folder",
			"updtfunc": func(fni interface{}, act *gi.Action) {
				ft := fni.(ki.Ki).Embed(KiT_FileTreeView).(*FileTreeView)
				fn := ft.FileNode()
				if fn != nil {
					act.SetActiveStateUpdt(fn.IsDir())
				}
			},
		}},
	},
}
