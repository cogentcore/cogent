// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

func TestTranspileGoal(t *testing.T) {
	src := "// test\nx := 10\nfor i := range x {\n\tfmt.Println(\"héllo\", i)\n}\nls -la\n"
	gc := transpileGoal(src, filepath.Join(t.TempDir(), "test.go"), nil, true)
	// a script is in a main function, with the imports it uses
	assert.Contains(t, gc.imports, `"fmt"`)
	assert.NotContains(t, gc.imports, `"cogentcore.org/lab/goal/goalib"`)
	assert.Equal(t, "func main() {", gc.goLines[gc.offset-3])
	// each line of the source is a line of the Go code
	assert.Equal(t, "x := 10", gc.goLines[gc.offset+1])
	assert.Equal(t, `fmt.Println("héllo", i)`, gc.goLines[gc.offset+3])
	assert.Equal(t, `goalrun.Run("ls", "-la")`, gc.goLines[gc.offset+5])
	assert.Equal(t, "}", gc.goLines[gc.offset+7])
	assert.Empty(t, gc.errs)

	// positions are mapped both ways, in runes, through the tokens,
	// including the indentation that the Go code does not have
	pos := func(ln, ch int) textpos.Pos { return textpos.Pos{Line: ln, Char: ch} }
	goal := []textpos.Pos{pos(1, 0), pos(1, 5), pos(3, 1), pos(3, 6), pos(3, 22), pos(3, 24)}
	gos := []textpos.Pos{pos(1, 0), pos(1, 5), pos(3, 0), pos(3, 5), pos(3, 21), pos(3, 23)}
	for i, gp := range goal {
		want := gos[i]
		want.Line += gc.offset
		p, ok := gc.toGo(gp)
		assert.True(t, ok)
		assert.Equal(t, want, p, "toGo %v", gp)
		p, ok = gc.toGoal(want)
		assert.True(t, ok)
		assert.Equal(t, gp, p, "toGoal %v", want)
	}
	// a position at a space is after the token before it, as for
	// the type of x, after it
	p, ok := gc.toGoal(pos(gc.offset+1, 1))
	assert.True(t, ok)
	assert.Equal(t, pos(1, 1), p)
	// the code of a command is at its start
	p, ok = gc.toGoal(pos(gc.offset+5, 10))
	assert.True(t, ok)
	assert.Equal(t, pos(5, 0), p)
	// the header and the end are not in the source
	_, ok = gc.toGoal(pos(gc.offset-1, 0))
	assert.False(t, ok)
	_, ok = gc.toGoal(pos(gc.offset+7, 0))
	assert.False(t, ok)
}

func TestTranspileGoalPackage(t *testing.T) {
	src := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hi\")\n}\n"
	gc := transpileGoal(src, filepath.Join(t.TempDir(), "test.go"), nil, true)
	// a file with a package has its own imports and no main function
	assert.Equal(t, 1, gc.offset)
	assert.Equal(t, "package main", gc.goLines[1])
	assert.Equal(t, strings.Count(src, "\n")+1, gc.numGoalLines())
	assert.Equal(t, "}", gc.goLines[gc.offset+6])
	p, ok := gc.toGoal(textpos.Pos{Line: 6, Char: 4})
	assert.True(t, ok)
	assert.Equal(t, textpos.Pos{Line: 5, Char: 5}, p)
}

func TestTranspileGoalErrors(t *testing.T) {
	src := "x := 1\nfor {\n"
	gc := transpileGoal(src, filepath.Join(t.TempDir(), "test.go"), []string{`"fmt"`}, true)
	// the imports of the code with errors are the given ones
	assert.Equal(t, []string{`"fmt"`}, gc.imports)
	assert.Equal(t, "x := 1", gc.goLines[gc.offset])
}
