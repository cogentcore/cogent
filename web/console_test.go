// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// consoleTexts returns the texts of the messages of the console panel.
func consoleTexts(b *core.Body, br *Browser) string {
	b.AsyncLock()
	defer b.AsyncUnlock()
	var ts []string
	br.Console.list.WalkDown(func(n tree.Node) bool {
		if tx, ok := n.(*core.Text); ok {
			ts = append(ts, tx.Text)
		}
		return tree.Continue
	})
	return strings.Join(ts, "|")
}

// TestConsolePanel tests the console panel of the browser: hidden at
// first, it shows the messages of the page of the current tab when it is
// shown, with those that come after, and those of another tab when it is
// selected, and it is cleared and closed.
func TestConsolePanel(t *testing.T) {
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.X.Dp(800); s.Min.Y.Dp(600) })
	br := NewBrowser(b)
	br.PageFunc = func(pg *Page) {
		pg.Context.GetURL = func(u string) (*http.Response, error) {
			pu, _ := url.Parse(u)
			r := strings.NewReader(`<html><body><p>page</p><script>console.log("hello from ` + pu.Path + `"); console.error("oops"); setTimeout(() => console.warn("later"), 300);</script></body></html>`)
			return &http.Response{StatusCode: 200, Status: "200 OK", Request: &http.Request{URL: pu}, Body: io.NopCloser(r)}, nil
		}
	}
	one := br.NewTab("https://example.com/one")
	br.NewTab("https://example.com/two")
	b.RenderImage(func() {
		for _, pg := range br.Pages() {
			settle(pg)
		}
		b.AsyncLock()
		assert.False(t, br.ConsoleShown())
		assert.True(t, br.Splits.ChildIsCollapsed(1), "the console is hidden at first")
		br.ShowConsole(true)
		assert.False(t, br.Splits.ChildIsCollapsed(1))
		b.AsyncUnlock()
		// the messages of the page, and one that comes later
		require.True(t, eventually(func() bool { return strings.Contains(consoleTexts(b, br), "WARN: later") }), consoleTexts(b, br))
		assert.Equal(t, "INFO: hello from /two|ERROR: oops|WARN: later", consoleTexts(b, br))
		b.AsyncLock()
		br.Tabs.SelectTabIndex(0)
		b.AsyncUnlock()
		assert.True(t, eventually(func() bool { return strings.Contains(consoleTexts(b, br), "hello from /one") }), consoleTexts(b, br))
		b.AsyncLock()
		assert.Equal(t, one, br.CurrentPage())
		one.clearConsole()
		br.Console.refresh()
		b.AsyncUnlock()
		assert.Equal(t, "No messages", consoleTexts(b, br))
		b.AsyncLock()
		br.ShowConsole(false)
		assert.False(t, br.ConsoleShown())
		assert.True(t, br.Splits.ChildIsCollapsed(1))
		b.AsyncUnlock()
	})
}
