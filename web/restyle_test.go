// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"image/color"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/styles"
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
// that needs it (of a display) reads the changed elements again (see
// [TestScriptsReread]).
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
		renders, restyles, rereads := ps.renders, ps.restyles, ps.rereads
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
		assert.Greater(t, ps.rereads, rereads, "the changed elements are read again")
		b.AsyncUnlock()
	})
}

// TestScriptsKeepContainerSizes tests that the elements in containers of a
// page that is read again after its scripts change it are styled for the
// sizes of the containers from the start (see
// [htmlcore.Context.KeepContainerSizes]), not for the viewport, as the
// panels of repositories of github.com, of font-size: 1cqw, which were
// shown much larger for a moment each time that the page was read again.
func TestScriptsKeepContainerSizes(t *testing.T) {
	var mu sync.Mutex
	var sizes []float32 // of the font of the text, as each widget of it is first styled
	b, pg := newScriptPageWith(t, map[string]string{
		"/": `<html><body><div style="container-type: inline-size; width: 200px">` +
			`<div id="text" style="font-size: 10cqw">text</div></div><div id="other">other</div></body></html>`,
	}, func(pg *Page) {
		pg.Context.WidgetHandlers = append(pg.Context.WidgetHandlers, func(w core.Widget) {
			if htmlcore.GetAttr(htmlcore.NodeOf(w), "id") != "text" {
				return
			}
			first := true
			w.AsWidget().FinalStyler(func(s *styles.Style) {
				if first {
					first = false
					mu.Lock()
					sizes = append(sizes, s.Font.Size.Value)
					mu.Unlock()
				}
			})
		})
	})
	b.RenderImage(func() { settle(pg) }, func() {
		ps := pg.scripts
		require.NotNil(t, ps)
		// (a change of the html element itself, for which all of it is read again)
		_, err := ps.engine.RunString(`document.documentElement.setAttribute("lang", "fr");
			document.getElementById("other").textContent = "changed"`)
		require.NoError(t, err)
		assert.True(t, eventually(func() bool { return strings.Contains(pageTexts(b, pg), "changed") }), "the page is read again")
		mu.Lock()
		defer mu.Unlock()
		require.GreaterOrEqual(t, len(sizes), 2)
		assert.InDelta(t, 20, sizes[len(sizes)-1], 0.5, "10%% of the width of its container from the start: %v", sizes)
	})
}

// TestScriptsReread tests that changes of the document by scripts (of the
// text of an element, of the children of another, and of an attribute of
// another) are made by reading only the changed elements again, keeping
// the widgets of the others (see [htmlcore.Context.Reread]), without
// reading the page again, and that the page of a frame whose box is made
// new is kept, in its new box.
func TestScriptsReread(t *testing.T) {
	frame := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<html><body><p>in the frame</p></body></html>`)
	}))
	defer frame.Close()
	b, pg := newScriptPage(t, map[string]string{
		"/": `<html><body><div id="wrap"><p id="out">before</p><ul id="list"><li id="a">a</li></ul>
			<div id="box"><iframe src="` + frame.URL + `/frame" width="300" height="100"></iframe></div></div>
			<p id="other">other</p></body></html>`,
	})
	b.RenderImage(func() { settle(pg) }, func() {
		require.True(t, eventually(func() bool { return strings.Contains(pageTexts(b, pg), "in the frame") }), pageTexts(b, pg))
		ps := pg.scripts
		require.NotNil(t, ps)
		b.AsyncLock()
		renders, rereads := ps.renders, ps.rereads
		other, a := widgetWithID(pg, "other"), widgetWithID(pg, "a")
		require.Len(t, pg.frames, 1)
		var fp *Page
		for _, f := range pg.frames {
			fp = f
		}
		b.AsyncUnlock()
		require.NotNil(t, other)
		require.NotNil(t, a)

		_, err := ps.engine.RunString(`document.getElementById("out").textContent = "after";
			const li = document.createElement("li");
			li.textContent = "b";
			document.getElementById("list").appendChild(li);
			document.getElementById("box").setAttribute("title", "a new box")`)
		require.NoError(t, err)
		require.True(t, eventually(func() bool {
			txt := pageTexts(b, pg)
			return strings.Contains(txt, "after") && strings.Contains(txt, "b")
		}), pageTexts(b, pg))
		ps.engine.Wait()
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Equal(t, renders, ps.renders, "the page is not read again")
		assert.Greater(t, ps.rereads, rereads, "the changed elements are read again")
		assert.Same(t, other, widgetWithID(pg, "other"), "the widget of an element that has not changed is kept")
		assert.Same(t, a, widgetWithID(pg, "a"), "the widget of an item of the list that has not changed is kept")
		assert.Len(t, pg.frames, 1)
		assert.Same(t, fp, pg.frames[htmlcore.NodeOf(fp.Parent.(core.Widget))], "the page of the frame is kept")
		assert.Equal(t, "a new box", widgetWithID(pg, "box").Property("title"), "the box is made new")
		assert.Contains(t, pageTextsLocked(pg), "in the frame")
	})
}

// pageTextsLocked returns the texts of the widgets of the page, with the
// lock held.
func pageTextsLocked(pg *Page) string {
	var ts []string
	pg.WalkDown(func(n tree.Node) bool {
		if tx, ok := n.(*core.Text); ok {
			ts = append(ts, tx.Text)
		}
		return tree.Continue
	})
	return strings.Join(ts, "|")
}

// TestScriptsTransitionEnd tests that a change of a class of an element by
// a script that changes a property with a transition dispatches the
// transitionend event when the transition would end (Google sign-in waits
// for it to show its next step, which was never shown).
func TestScriptsTransitionEnd(t *testing.T) {
	b, pg := newScriptPage(t, map[string]string{
		"/": `<html><head><style>
			.view { transform: translateX(0); transition: transform 50ms ease-out }
			.view.away { transform: translateX(-100px) }
			</style></head><body><div id="v" class="view">view</div></body></html>`,
	})
	b.RenderImage(func() { settle(pg) }, func() {
		ps := pg.scripts
		require.NotNil(t, ps)
		_, err := ps.engine.RunString(`window.ended = "";
			const v = document.getElementById("v");
			v.addEventListener("transitionend", e => { window.ended += e.propertyName + ":" + (e instanceof TransitionEvent) + ":" + e.elapsedTime; });
			v.classList.add("away"); v.getBoundingClientRect();`)
		require.NoError(t, err)
		assert.True(t, eventually(func() bool {
			v, _ := ps.engine.RunString(`window.ended`)
			return v == "transform:true:0.05"
		}), "transitionend")
	})
}
