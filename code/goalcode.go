// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"unicode/utf8"

	"cogentcore.org/core/base/stringsx"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/lab/goal/transpile"
	"golang.org/x/tools/imports"
)

// goalCode is the Go code of a Goal file, as `goal build` makes it, but
// without formatting it, so that the Go code of each line of the Goal
// source is on its own line, after a header, which makes it possible
// to map the positions of the Go code back to the Goal source, for the
// language services that the Go language server (gopls) provides.
type goalCode struct {

	// code is the Go code.
	code string

	// goLines are the lines of the Go code.
	goLines []string

	// offset is the line of the Go code of the first line of the Goal
	// source: the number of lines of the header.
	offset int

	// src are the lines of the Goal source, with the map from the
	// columns of their Go code back to their own columns.
	src []transpile.SourceLine

	// imports are the import paths of the Go code, as goimports finds
	// them, with a name before the path if it has one (such as _).
	imports []string

	// errs are the errors of transpiling the Goal source.
	errs []*transpile.Error
}

// goalHeaderImports are the imports of the Go code of a Goal file
// without a package declaration, before goimports adds the others
// that it uses, and removes those that it does not use.
var goalHeaderImports = []string{
	`"cogentcore.org/lab/goal"`,
	`"cogentcore.org/lab/goal/goalib"`,
	`"cogentcore.org/lab/tensor"`,
	`_ "cogentcore.org/lab/tensor/tmath"`,
	`_ "cogentcore.org/lab/stats/stats"`,
	`_ "cogentcore.org/lab/stats/metric"`,
}

// transpileGoal returns the Go code of the given Goal source, for the
// Go file of the given name, as `goal build` makes it, but without
// formatting it (see [goalCode]). A Goal file without a package
// declaration is a script, whose code is in a main function, after the
// imports, which are found with goimports, as for `goal build`, if
// findImports is set. The given imports are used otherwise, or if it
// fails, which it can for code with errors.
func transpileGoal(src, gofile string, prevImports []string, findImports bool) *goalCode {
	lns := stringsx.SplitLines(src)
	hasPackage := false
	for _, ln := range lns {
		if strings.HasPrefix(ln, "package ") {
			hasPackage = true
			break
		}
	}
	st := transpile.NewState()
	st.FuncToVar = !hasPackage
	st.TranspileCode(src)
	st.AddChunk()
	gc := &goalCode{}
	for _, err := range st.Errors {
		if te, ok := err.(*transpile.Error); ok {
			gc.errs = append(gc.errs, te)
		}
	}

	// The Go code of each source line, which is one line of Go code.
	var body []string
	for ci, ch := range st.Chunks {
		cls := strings.Split(ch, "\n")
		srcs := st.ChunkSources[ci]
		for i, cl := range cls {
			body = append(body, cl)
			if i < len(srcs) {
				gc.src = append(gc.src, srcs[i])
			} else {
				gc.src = append(gc.src, transpile.SourceLine{})
			}
		}
	}
	for len(body) < len(lns) { // empty lines at the end
		body = append(body, "")
		gc.src = append(gc.src, transpile.SourceLine{Src: lns[len(gc.src)]})
	}

	// not marked as generated, which gopls warns about editing
	gen := "// The Go code of a Goal file, for the Go language server."
	if hasPackage {
		gc.offset = 1
		gc.setCode(gen, body, nil)
		return gc
	}
	header := func(imps []string) []string {
		hdr := []string{gen, "package main", "import ("}
		for _, imp := range imps {
			hdr = append(hdr, "\t"+imp)
		}
		return append(hdr, ")", "", "func main() {", "\tgoalrun := goal.NewGoal()", "\t_ = goalrun")
	}
	full := strings.Join(header(goalHeaderImports), "\n") + "\n" + strings.Join(body, "\n") + "\n}\n"
	var imps []string
	if findImports {
		imps = goImports(gofile, full)
	}
	switch {
	case imps != nil:
	case prevImports != nil:
		imps = prevImports
	default:
		imps = goalHeaderImports
	}
	gc.imports = imps
	hdr := header(imps)
	gc.offset = len(hdr)
	gc.setCode(strings.Join(hdr, "\n"), body, []string{"}"})
	return gc
}

// setCode sets the code from the given header, lines of the body,
// and lines after it.
func (gc *goalCode) setCode(header string, body, after []string) {
	gc.code = header + "\n" + strings.Join(append(body, after...), "\n") + "\n"
	gc.goLines = strings.Split(gc.code, "\n")
}

// goImports returns the imports of the given Go code of the given file
// after goimports processes it, adding the imports that it uses and
// removing those that it does not use, or nil if that fails.
func goImports(gofile, code string) []string {
	res, err := imports.Process(gofile, []byte(code), &imports.Options{Comments: true, FormatOnly: false})
	if err != nil {
		return nil
	}
	f, err := parser.ParseFile(token.NewFileSet(), gofile, res, parser.ImportsOnly)
	if err != nil {
		return nil
	}
	imps := []string{}
	for _, is := range f.Imports {
		imp := is.Path.Value
		if is.Name != nil {
			imp = is.Name.Name + " " + imp
		}
		if _, err := strconv.Unquote(is.Path.Value); err == nil {
			imps = append(imps, imp)
		}
	}
	return imps
}

// numGoalLines returns the number of lines of the Goal source.
func (gc *goalCode) numGoalLines() int {
	return len(gc.src)
}

// toGo returns the position in the Go code of the given position in
// the Goal source, and whether it has one.
func (gc *goalCode) toGo(pos textpos.Pos) (textpos.Pos, bool) {
	if pos.Line < 0 || pos.Line >= len(gc.src) {
		return textpos.Pos{}, false
	}
	gl := gc.offset + pos.Line
	if gl >= len(gc.goLines) {
		return textpos.Pos{}, false
	}
	sl := &gc.src[pos.Line]
	gtxt := gc.goLines[gl]
	if len(sl.Cols) == 0 { // the same code
		return textpos.Pos{Line: gl, Char: min(pos.Char, utf8.RuneCountInString(gtxt))}, true
	}
	b := runeToByte(sl.Src, pos.Char)
	// The Go code of the source token that the position is in, which
	// is the last one that starts at or before it, at its offset in it.
	best := -1
	for i, c := range sl.Cols {
		if c > 0 && int(c)-1 <= b && (best < 0 || c > sl.Cols[best]) {
			best = i
		}
	}
	if best < 0 {
		return textpos.Pos{Line: gl}, true
	}
	end := best
	for end < len(sl.Cols) && sl.Cols[end] == sl.Cols[best] {
		end++
	}
	gb := min(best+b-(int(sl.Cols[best])-1), end, len(gtxt))
	return textpos.Pos{Line: gl, Char: utf8.RuneCountInString(gtxt[:gb])}, true
}

// toGoal returns the position in the Goal source of the given position
// in the Go code, and whether it has one: the code that the transpiler
// adds before and after the code of the source, such as the imports,
// does not.
func (gc *goalCode) toGoal(pos textpos.Pos) (textpos.Pos, bool) {
	ln := pos.Line - gc.offset
	if ln < 0 || ln >= len(gc.src) || pos.Line >= len(gc.goLines) {
		return textpos.Pos{}, false
	}
	sl := &gc.src[ln]
	gtxt := gc.goLines[pos.Line]
	nsrc := utf8.RuneCountInString(sl.Src)
	if len(sl.Cols) == 0 { // the same code
		return textpos.Pos{Line: ln, Char: min(pos.Char, nsrc)}, true
	}
	i := runeToByte(gtxt, pos.Char)
	if i >= len(sl.Cols) { // at the end
		return textpos.Pos{Line: ln, Char: nsrc}, true
	}
	if i > 0 && gtxt[i] == ' ' { // the spaces are after the token before
		p, ok := gc.toGoal(textpos.Pos{Line: pos.Line, Char: pos.Char - 1})
		p.Char = min(p.Char+1, nsrc)
		return p, ok
	}
	c := sl.Cols[i]
	if c == 0 {
		return textpos.Pos{Line: ln}, true
	}
	// at the offset of the position in the Go code of its token
	i0 := i
	for i0 > 0 && sl.Cols[i0-1] == c {
		i0--
	}
	b := min(int(c)-1+i-i0, len(sl.Src))
	return textpos.Pos{Line: ln, Char: utf8.RuneCountInString(sl.Src[:b])}, true
}

// toGoalRegion returns the region in the Goal source of the given region
// in the Go code, and whether it has one, which it does if its start does.
func (gc *goalCode) toGoalRegion(reg textpos.Region) (textpos.Region, bool) {
	st, ok := gc.toGoal(reg.Start)
	if !ok {
		return textpos.Region{}, false
	}
	ed, ok := gc.toGoal(reg.End)
	if !ok || ed.IsLess(st) {
		ed = st
	}
	return textpos.Region{Start: st, End: ed}, true
}

// runeToByte returns the byte offset in s of the given rune index.
func runeToByte(s string, r int) int {
	if r <= 0 {
		return 0
	}
	n := 0
	for i := range s {
		if n == r {
			return i
		}
		n++
	}
	return len(s)
}
