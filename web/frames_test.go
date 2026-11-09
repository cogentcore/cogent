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
	"cogentcore.org/core/styles"
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
