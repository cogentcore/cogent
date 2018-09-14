// Copyright (c) 2018, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
package gide provides the core Gide editor object.

Derived classes can extend the functionality for specific domains.

*/
package gide

import (
	"log"

	"github.com/goki/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// FileNode is Gide version of FileNode for FileTree view
type FileNode struct {
	giv.FileNode
	Langs LangNames `desc:"languages associated with this file"`
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
