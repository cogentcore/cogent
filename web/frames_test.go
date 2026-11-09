// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFrameCaptcha tests a page that a site blocks with a captcha in a
// frame from another origin, as the bot checks of DataDome (as of
// nytimes.com): the script of the page writes the iframe, the page in it
// is shown in its box, and when it is solved (by a click on its button),
// it posts a cookie to the page, which sets it and reloads, and the site
// then serves the page.
func TestFrameCaptcha(t *testing.T) {
	// the cookie is of this run, as cookies are of hosts, not ports
	token := fmt.Sprintf("solved%d", time.Now().UnixNano())
	captcha := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<html><body><p>slide to solve</p><button id="solve">solve</button>
			<script>
				document.getElementById("solve").addEventListener("click", () => {
					window.parent.postMessage(JSON.stringify({ cookie: "dd=`+token+`; path=/" }), "*");
				});
			</script></body></html>`)
	}))
	defer captcha.Close()
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("dd"); err == nil && c.Value == token {
			io.WriteString(w, `<html><body><p>the news</p></body></html>`)
			return
		}
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, `<html><body><p id="msg">please wait</p>
			<script>
				window.addEventListener("message", e => {
					if (e.origin !== "`+captcha.URL+`") return;
					document.cookie = JSON.parse(e.data).cookie;
					location.reload();
				});
				const f = document.createElement("div");
				f.innerHTML = '<iframe src="`+captcha.URL+`/c" width="100%" height="200" frameborder="0"></iframe>';
				document.body.appendChild(f);
				f.firstChild.addEventListener("load", () => { document.getElementById("msg").textContent = "frame loaded"; });
			</script></body></html>`)
	}))
	defer site.Close()

	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.X.Dp(800); s.Min.Y.Dp(600) })
	pg := NewPage(b)
	pg.OpenURL(site.URL+"/", nil)
	b.RenderImage(func() { settle(pg) }, func() {
		// the captcha is shown in the frame of the page, which is told that
		// it loaded
		require.True(t, eventually(func() bool {
			txt := pageTexts(b, pg)
			return strings.Contains(txt, "slide to solve") && strings.Contains(txt, "frame loaded")
		}), pageTexts(b, pg))
		b.AsyncLock()
		var fp *Page
		for _, f := range pg.frames {
			fp = f
		}
		b.AsyncUnlock()
		require.NotNil(t, fp)
		assert.Equal(t, captcha.URL+"/c", fp.currentURL())
		require.True(t, eventually(func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return fp.scripts != nil
		}))
		_, err := fp.scripts.engine.RunString(`document.getElementById("solve").click()`)
		require.NoError(t, err)
		// the page sets the cookie from the captcha, and reloads
		assert.True(t, eventually(func() bool { return strings.Contains(pageTexts(b, pg), "the news") }), pageTexts(b, pg))
		b.AsyncLock()
		assert.Empty(t, pg.frames, "the frame is closed with its page")
		b.AsyncUnlock()
	})
}

// TestFrameMouseEvents tests the events of the mouse in a page in a frame,
// of a zoomed page: the element under the mouse gets pointerdown,
// mousedown, mouseup and click (in that order) at the position of the mouse
// in its box (its clientX and clientY in the coordinates of its bounding
// rect), and the page around the frame sees the iframe hovered, not the
// elements of the other page. The checkbox of reCAPTCHA acts on mousedown
// and mouseup, which were not dispatched, and the zoom was applied twice
// to the positions of the mouse in frames, so that it was hovered to the
// right of it.
func TestFrameMouseEvents(t *testing.T) {
	tempSites(t)
	inner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<html><body style="margin: 0"><div style="padding: 10px 0 0 30px"><span id="box" style="display: inline-block; width: 40px; height: 40px"></span></div><p id="out">events</p>
			<script>
				const b = document.getElementById("box"), o = document.getElementById("out");
				for (const ty of ["pointerdown", "mousedown", "mouseup", "click"]) b.addEventListener(ty, e => {
					const r = b.getBoundingClientRect(), in_ = e.clientX > r.left && e.clientX < r.right && e.clientY > r.top && e.clientY < r.bottom;
					o.textContent += " " + ty + (in_ ? ":in" : ":out");
				});
				o.textContent = "ready";
			</script></body></html>`)
	}))
	defer inner.Close()
	outer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<html><body style="margin: 0"><div style="padding: 80px 0 0 120px"><iframe src="`+inner.URL+`/" width="300" height="150" frameborder="0"></iframe></div><p id="pout">parent</p>
			<script>document.addEventListener("mouseover", e => { document.getElementById("pout").textContent += " over:" + (e.target.id || e.target.localName); });</script></body></html>`)
	}))
	defer outer.Close()
	SetSite(outer.URL+"/", func(ss *SiteSettings) { ss.Zoom = 1.25 })

	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.X.Dp(800); s.Min.Y.Dp(600) })
	pg := NewPage(b)
	pg.OpenURL(outer.URL+"/", nil)
	b.RenderImage(func() {
		settle(pg)
		var box *core.WidgetBase
		require.True(t, eventually(func() bool {
			if !strings.Contains(pageTexts(b, pg), "ready") {
				return false
			}
			b.AsyncLock()
			defer b.AsyncUnlock()
			pg.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
				if cwb.Property("id") == "box" && cwb.Geom.TotalBBox.Dx() > 0 {
					box = cwb
				}
				return tree.Continue
			})
			return box != nil
		}))
		b.AsyncLock()
		bb := box.Geom.TotalBBox
		b.AsyncUnlock()
		c := bb.Min.Add(bb.Size().Div(2))
		ev := b.Scene.RenderWindow().SystemWindow.Events()
		ev.MouseMove(c)
		ev.MouseButton(events.MouseDown, events.Left, c, 0)
		ev.MouseButton(events.MouseUp, events.Left, c, 0)
		assert.True(t, eventually(func() bool {
			return strings.Contains(pageTexts(b, pg), "ready pointerdown:in mousedown:in mouseup:in click:in")
		}), pageTexts(b, pg))
		assert.Contains(t, pageTexts(b, pg), "parent over:iframe")
		assert.NotContains(t, pageTexts(b, pg), "over:box")
	})
}
