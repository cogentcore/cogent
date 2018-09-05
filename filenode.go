// Copyright (c) 2018, The gide / GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gide

import (
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/goki/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// FileNode represents a file in the file system -- the name of the node is
// the name of the file.  Folders have children containing further nodes.
type FileNode struct {
	ki.Node
	Ic      gi.IconName  `desc:"icon for this file"`
	FPath   gi.FileName  `desc:"full path to this file"`
	Size    giv.FileSize `desc:"size of the file in bytes"`
	Kind    string       `width:"20" max-width:"20" desc:"type of file / directory -- including MIME type"`
	Mode    os.FileMode  `desc:"file mode bits"`
	ModTime giv.FileTime `desc:"time that contents (only) were last modified"`
	Buf     *giv.TextBuf `json:"-" desc:"file buffer for editing"`
}

var KiT_FileNode = kit.Types.AddType(&FileNode{}, FileNodeProps)

var FileNodeProps = ki.Props{}

// IsDir returns true if file is a directory (folder)
func (fn *FileNode) IsDir() bool {
	return fn.Kind == "Folder"
}

// OpenPath reads all the files at given path into this tree -- uses config
// children to preserve extra info already stored about files. The root node
// represents the directory at the given path.
func (fn *FileNode) OpenPath(path string) {
	_, fnm := filepath.Split(path)
	fn.SetName(fnm)
	fn.FPath = gi.FileName(path)

	config := fn.ConfigOfFiles(path)
	mods, updt := fn.ConfigChildren(config, true) // unique names
	if mods {
		for _, sfk := range fn.Kids {
			sf := sfk.(*FileNode)
			fp := filepath.Join(path, sf.Nm)
			sf.UpdateNode(fp)
		}
		fn.UpdateEnd(updt)
	}
}

// ConfigOfFiles returns a type-and-name list for configuring nodes based on
// files immediately within given path
func (fn *FileNode) ConfigOfFiles(path string) kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	filepath.Walk(path, func(pth string, info os.FileInfo, err error) error {
		if err != nil {
			emsg := fmt.Sprintf("gide.FileNode ConfigFilesIn Path %q: Error: %v", path, err)
			log.Println(emsg)
			return nil // ignore
		}
		if pth == path { // proceed..
			return nil
		}
		_, fn := filepath.Split(pth)
		config.Add(KiT_FileNode, fn)
		if info.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})
	return config
}

// UpdateNode updates information in node based on file
func (fn *FileNode) UpdateNode(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		emsg := fmt.Errorf("gide.FileNode UpdateNode Path %q: Error: %v", path, err)
		log.Println(emsg)
		return emsg
	}
	fn.FPath = gi.FileName(path)
	fn.Size = giv.FileSize(info.Size())
	fn.Mode = info.Mode()
	fn.ModTime = giv.FileTime(info.ModTime())

	if info.IsDir() {
		fn.Kind = "Folder"
	} else {
		ext := filepath.Ext(fn.Nm)
		fn.Kind = mime.TypeByExtension(ext)
		fn.Kind = strings.TrimPrefix(fn.Kind, "application/") // long and unnec
	}
	fn.Ic = giv.FileKindToIcon(fn.Kind, fn.Nm)

	if fn.IsDir() {
		if !strings.HasPrefix(fn.Nm, ".") { // todo: pref
			fn.OpenPath(path) // keep going down..
		}
	}
	return nil
}

// OpenBuf opens the file in its buffer
func (fn *FileNode) OpenBuf() error {
	if fn.IsDir() {
		err := fmt.Errorf("gide.FileNode cannot open directory in editor: %v", fn.FPath)
		log.Println(err.Error())
		return err
	}
	fn.Buf = &giv.TextBuf{}
	fn.Buf.InitName(fn.Buf, fn.Nm)
	return fn.Buf.Open(fn.FPath)
}

// FindFile finds first node representing given file (false if not found) --
// looks for full path names that have the given string as their suffix, so
// you can include as much of the path (including whole thing) as is relevant
// to disambiguate.  See FilesMatching for a list of files that match a given
// string.
func (fn *FileNode) FindFile(fnm string) (*FileNode, bool) {
	var ffn *FileNode
	found := false
	fn.FuncDownMeFirst(0, fn, func(k ki.Ki, level int, d interface{}) bool {
		sfn := k.(*FileNode)
		if strings.HasSuffix(string(sfn.FPath), fnm) {
			ffn = sfn
			found = true
			return false
		}
		return true
	})
	return ffn, found
}
