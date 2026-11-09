// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"image/color"
	"strings"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// widgetWithID returns the widget of the element of the given id in the
// page, or nil (with the lock held).
func widgetWithID(pg *Page, id string) *core.WidgetBase {
	var found *core.WidgetBase
	pg.WalkDown(func(n tree.Node) bool {
		if wb := core.AsWidget(n); wb != nil {
			if v, ok := wb.Property("id").(string); ok && v == id {
				found = wb
				return tree.Break
			}
		}
		return tree.Continue
	})
	return found
}

// TestScriptsRestyle tests that changes of the styles of elements by
// scripts (of their style and class attributes) are made by styling their
// widgets again, without reading the page into new widgets, and that the
// boxes that the scripts get then are of the new styles; and that a change
// that needs it (of a display) reads the page again.
func TestScriptsRestyle(t *testing.T) {
	b, pg := newScriptPage(t, map[string]string{
		"/": `<html><head><style>
			.box { width: 100px; height: 20px; background: blue }
			.red { background: red }
			.sub { display: none }
			.open .sub { display: block }
			</style></head><body>
			<div id="box" class="box">box</div>
			<div id="menu"><div class="sub">sub item</div></div>
			</body></html>`,
	})
	b.RenderImage(func() { settle(pg) }, func() {
		ps := pg.scripts
		require.NotNil(t, ps)
		b.AsyncLock()
		renders, restyles := ps.renders, ps.restyles
		b.AsyncUnlock()

		v, err := ps.engine.RunString(`const box = document.getElementById("box");
			box.style.width = "300px";
			box.classList.add("red");
			box.getBoundingClientRect().width`)
		require.NoError(t, err)
		assert.InDelta(t, 300, v, 1, "the box that the script gets is of the new width")
		ps.engine.Wait()
		b.AsyncLock()
		assert.Equal(t, renders, ps.renders, "the page is not read again")
		assert.Greater(t, ps.restyles, restyles, "it is restyled")
		wb := widgetWithID(pg, "box")
		require.NotNil(t, wb)
		assert.InDelta(t, 300*pg.pxDots(), wb.Geom.Size.Actual.Total.X, 2)
		assert.Equal(t, color.RGBA{255, 0, 0, 255}, colors.AsRGBA(colors.ToUniform(wb.Styles.Background)))
		b.AsyncUnlock()

		// a change of the display of an element is a change of its widgets
		_, err = ps.engine.RunString(`document.getElementById("menu").classList.add("open")`)
		require.NoError(t, err)
		assert.True(t, eventually(func() bool { return strings.Contains(pageTexts(b, pg), "sub item") }), "the item is shown")
		b.AsyncLock()
		assert.Greater(t, ps.renders, renders, "the page is read again")
		b.AsyncUnlock()
	})
}
