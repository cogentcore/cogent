// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/lab/goal/goalfmt"
)

// The language services for Goal files come from goalls, the language
// server for Goal files, which is a proxy for gopls, for the Go code that
// Goal files are transpiled to (see cogentcore.org/lab/goal/goalls). It
// is used instead of gopls for Go files when it is installed. Goal files
// have the Go file type, for their highlighting, but they are never sent
// to gopls itself, which cannot parse them.

// isGoalFile returns whether the given buffer is of a Goal file, which
// has the .goal extension (its file type can be Go, for highlighting).
func isGoalFile(ln *lines.Lines) bool {
	return ln != nil && (ln.FileInfo().Known == fileinfo.Goal || strings.EqualFold(filepath.Ext(ln.Filename()), ".goal"))
}

// goalServer returns whether the language server for Go files is goalls,
// which is the server for Goal files.
func (ls *LangService) goalServer() bool {
	sc, ok := ls.lspB.Reg.Server(fileinfo.Go)
	return ok && sc.ID == "goalls"
}

// formatGoal formats the given saved Goal file and fixes its imports,
// as goimports does for Go files, with goalfmt, writing it if it changed,
// and returns whether it did.
func formatGoal(fname string) (bool, error) {
	src, err := os.ReadFile(fname)
	if err != nil {
		return false, err
	}
	res, err := goalfmt.Source(fname, src, nil)
	if err != nil || bytes.Equal(src, res) {
		return false, err
	}
	return true, os.WriteFile(fname, res, 0666)
}
