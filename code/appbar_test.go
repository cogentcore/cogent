// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"image"
	"slices"
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// menuTexts returns the texts of the buttons of the menu made by the
// given function, including those of its submenus.
func menuTexts(fun func(m *core.Scene, pos image.Point)) []string {
	m := core.NewScene()
	fun(m, image.Point{})
	var txts []string
	for cw := range m.AllWidgets() {
		var bt *core.Button
		switch x := cw.(type) {
		case *core.FuncButton:
			bt = &x.Button
		case *core.Button:
			bt = x
		}
		if bt == nil {
			continue
		}
		txts = append(txts, bt.Text)
		if bt.Menu != nil {
			txts = append(txts, menuTexts(bt.Menu)...)
		}
	}
	return txts
}

// TestCodeMenus tests the toolbar of Code, its overflow menu, the context
// menus of the editors and the file tree, and the items of the search of
// the app bar.
func TestCodeMenus(t *testing.T) {
	tmp := tempDir(t)
	path := writeFiles(t, tmp, map[string]string{
		"a.txt":     "some text\n",
		"sub/b.txt": "b\n",
	})
	b, cv := newTestCode(t, tmp)
	b.AddTopBar(func(bar *core.Frame) {
		tb := core.NewToolbar(bar)
		tb.Maker(cv.MakeToolbar)
		tb.AddOverflowMenu(cv.OverflowMenu)
	})
	core.AssertRender(t, b, "menus", func() {
		locked(b, func() {
			tb := core.ButtonByText(b.Scene, "Save")
			assert.NotNil(t, tb, "the toolbar")
			assert.NotNil(t, core.ButtonByText(b.Scene, "Find"))

			over := menuTexts(cv.OverflowMenu)
			for _, txt := range []string{"New project", "Count words region", "Help"} {
				assert.Contains(t, over, txt)
			}

			tv, _, _ := cv.ViewFileInIndex(path("a.txt"), 0)
			tv.SetCursorShow(textpos.Pos{Char: 2})
			ctx := menuTexts(tv.ContextMenu)
			assert.Contains(t, ctx, "Copy")
			assert.Contains(t, ctx, "Lookup")

			var items []core.ChooserItem
			cv.MenuSearch(&items)
			texts := make([]string, len(items))
			for i, it := range items {
				texts[i] = it.Text
			}
			assert.Contains(t, texts, "a.txt:a.txt")
			i := slices.IndexFunc(texts, func(s string) bool { return strings.HasPrefix(s, "sub:") })
			require.GreaterOrEqual(t, i, 0, "%v", texts)
			items[i].Func() // opens the directory
			items[slices.Index(texts, "a.txt:a.txt")].Func()

			fn := cv.FileNodeForFile(path("a.txt"))
			require.NotNil(t, fn)
			fm := menuTexts(fn.This.(*FileNode).ContextMenu)
			assert.NotEmpty(t, fm)
		})
	})
}
