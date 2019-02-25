// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"image/color"
	"log"
	"path/filepath"
	"sort"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/filecat"
)

// FileNode is Gide version of FileNode for FileTree view
type FileNode struct {
	giv.FileNode
}

var KiT_FileNode = kit.Types.AddType(&FileNode{}, FileNodeProps)

// ParentGide returns the Gide parent of given node
func ParentGide(kn ki.Ki) (Gide, bool) {
	if kn.IsRoot() {
		return nil, false
	}
	var ge Gide
	kn.FuncUpParent(0, kn, func(k ki.Ki, level int, d interface{}) bool {
		if kit.EmbedImplements(k.Type(), GideType) {
			ge = k.(Gide)
			return false
		}
		return true
	})
	return ge, ge != nil
}

// ViewFile pulls up this file in Gide
func (fn *FileNode) ViewFile() {
	if fn.IsDir() {
		log.Printf("FileNode Edit -- cannot edit directories!\n")
		return
	}
	ge, ok := ParentGide(fn.This())
	if ok {
		ge.NextViewFileNode(fn.This().Embed(giv.KiT_FileNode).(*giv.FileNode))
	}
}

// ExecCmdFile pops up a menu to select a command appropriate for the given node,
// and shows output in MainTab with name of command
func (fn *FileNode) ExecCmdFile() {
	ge, ok := ParentGide(fn.This())
	if ok {
		ge.ExecCmdFileNode(fn.This().Embed(giv.KiT_FileNode).(*giv.FileNode))
	}
}

// ExecCmdNameFile executes given command name on node
func (fn *FileNode) ExecCmdNameFile(cmdNm string) {
	ge, ok := ParentGide(fn.This())
	if ok {
		ge.ExecCmdNameFileNode(fn.This().Embed(giv.KiT_FileNode).(*giv.FileNode), CmdName(cmdNm), true, true)
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

// DeleteDeleted deletes deleted nodes on list
func (on *OpenNodes) DeleteDeleted() {
	sz := len(*on)
	for i := sz - 1; i >= 0; i-- {
		fn := (*on)[i]
		if fn.This() == nil || fn.FRoot == nil || fn.IsDeleted() {
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
		if fn.IsChanged() {
			sl[i] += " *"
		}
	}
	return sl
}

// ByStringName returns the open node with given strings name
func (on *OpenNodes) ByStringName(name string) *giv.FileNode {
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
// descending order by number of occurrences -- ignoreCase transforms
// everything into lowercase
func FileTreeSearch(start *giv.FileNode, find string, ignoreCase bool, loc FindLoc, activeDir string, langs []filecat.Supported) []FileSearchResults {
	fsz := len(find)
	if fsz == 0 {
		return nil
	}
	mls := make([]FileSearchResults, 0)
	start.FuncDownMeFirst(0, start, func(k ki.Ki, level int, d interface{}) bool {
		sfn := k.Embed(giv.KiT_FileNode).(*giv.FileNode)
		if sfn.IsDir() && !sfn.IsOpen() {
			return false // don't go down into closed directories!
		}
		if sfn.IsDir() || sfn.IsExec() || sfn.Info.Kind == "octet-stream" || sfn.IsAutoSave() {
			return true
		}
		if !filecat.IsMatchList(langs, sfn.Info.Sup) {
			return true
		}
		if loc == FindLocDir {
			cdir, _ := filepath.Split(string(sfn.FPath))
			if activeDir != cdir {
				return true
			}
		} else if loc == FindLocNotTop {
			if level == 1 {
				return true
			}
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
	if ft.This() == ft.RootView.This() {
		ge, ok := ParentGide(ft.SrcNode.Ptr)
		if !ok {
			return nil
		}
		return AvailCmds.FilterCmdNames(filecat.NoSupport, ge.VersCtrl())
	}
	fn := ft.FileNode()
	if fn == nil {
		return nil
	}
	ge, ok := ParentGide(fn.This())
	if !ok {
		return nil
	}
	lang := filecat.NoSupport
	if fn != nil {
		lang = fn.Info.Sup
	}
	cmds := AvailCmds.FilterCmdNames(lang, ge.VersCtrl())
	return cmds
}

// ExecCmdFiles calls given command on selected files
func (ft *FileTreeView) ExecCmdFiles(cmdNm string) {
	sels := ft.SelectedViews()
	if len(sels) > 1 {
		CmdWaitOverride = true // force wait mode
	}
	for i := len(sels) - 1; i >= 0; i-- {
		sn := sels[i]
		ftv := sn.Embed(KiT_FileTreeView).(*FileTreeView)
		if ftv.This() == ft.RootView.This() {
			if ft.SrcNode.Ptr == nil {
				continue
			}
			ftr := ft.SrcNode.Ptr.(*giv.FileTree)
			ge, ok := ParentGide(ftr)
			if ok {
				ge.ExecCmdNameFileName(string(ftr.FPath), CmdName(cmdNm), true, true)
			}
		} else {
			fn := ftv.FileNode()
			if fn != nil {
				fn.ExecCmdNameFile(cmdNm)
			}
		}
	}
	if CmdWaitOverride {
		CmdWaitOverride = false
	}
}

// FileTreeInactiveDirFunc is an ActionUpdateFunc that inactivates action if node is a dir
var FileTreeInactiveDirFunc = giv.ActionUpdateFunc(func(fni interface{}, act *gi.Action) {
	ft := fni.(ki.Ki).Embed(KiT_FileTreeView).(*FileTreeView)
	fn := ft.FileNode()
	if fn != nil {
		act.SetInactiveState(fn.IsDir())
	}
})

// FileTreeActiveDirFunc is an ActionUpdateFunc that activates action if node is a dir
var FileTreeActiveDirFunc = giv.ActionUpdateFunc(func(fni interface{}, act *gi.Action) {
	ft := fni.(ki.Ki).Embed(KiT_FileTreeView).(*FileTreeView)
	fn := ft.FileNode()
	if fn != nil {
		act.SetActiveState(fn.IsDir())
	}
})

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
	".notinvcs": ki.Props{
		"color": "#ce4252",
	},
	".modified": ki.Props{
		"color": "#4b7fd1",
	},
	".added": ki.Props{
		"color": "#52af36",
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
			"label":    "View",
			"updtfunc": FileTreeInactiveDirFunc,
		}},
		{"ShowFileInfo", ki.Props{
			"label": "File Info",
		}},
		{"ExecCmdFiles", ki.Props{
			"label":        "Exec Cmd",
			"submenu-func": giv.SubMenuFunc(FileTreeViewExecCmds),
			"Args": ki.PropSlice{
				{"Cmd Name", ki.Props{}},
			},
		}},
		{"DuplicateFiles", ki.Props{
			"label":    "Duplicate",
			"updtfunc": FileTreeInactiveDirFunc,
			"shortcut": gi.KeyFunDuplicate,
		}},
		{"DeleteFiles", ki.Props{
			"label":    "Delete",
			"desc":     "Ok to delete file(s)?  This is not undoable and is not moving to trash / recycle bin",
			"shortcut": gi.KeyFunDelete,
		}},
		{"RenameFiles", ki.Props{
			"label": "Rename",
			"desc":  "Rename file to new file name",
		}},
		{"sep-open", ki.BlankProp{}},
		{"OpenDirs", ki.Props{
			"label":    "Open Dir",
			"desc":     "open given folder to see files within",
			"updtfunc": FileTreeActiveDirFunc,
		}},
		{"NewFile", ki.Props{
			"label":    "New File...",
			"desc":     "make a new file in this folder",
			"shortcut": gi.KeyFunInsert,
			"updtfunc": FileTreeActiveDirFunc,
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"width": 60,
				}},
				{"Add To Version Control", ki.Props{}},
			},
		}},
		{"NewFolder", ki.Props{
			"label":    "New Folder...",
			"desc":     "make a new folder within this folder",
			"shortcut": gi.KeyFunInsertAfter,
			"updtfunc": FileTreeActiveDirFunc,
			"Args": ki.PropSlice{
				{"Folder Name", ki.Props{
					"width": 60,
				}},
			},
		}},
		{"sep-vcs", ki.BlankProp{}},
		{"AddToVcs", ki.Props{
			//"label":    "Add To Git",
			"desc":       "Add file to version control git/svn",
			"updtfunc":   giv.FileTreeActiveNotInVcsFunc,
			"label-func": giv.VcsLabelFunc,
		}},
		{"RemoveFromVcs", ki.Props{
			//"label":    "Remove From Version Control",
			"desc":       "Remove file from version control git/svn",
			"updtfunc":   giv.FileTreeActiveInVcsFunc,
			"label-func": giv.VcsLabelFunc,
		}},
		{"CommitToVcs", ki.Props{
			"desc":       "Commit file to version control",
			"updtfunc":   giv.FileTreeActiveInVcsModifiedFunc,
			"label-func": giv.VcsLabelFunc,
		}},
		{"RevertVcs", ki.Props{
			"label":    "Revert",
			"desc":     "Revert file to last commit",
			"updtfunc": giv.FileTreeActiveInVcsModifiedFunc,
		}},
	},
}
