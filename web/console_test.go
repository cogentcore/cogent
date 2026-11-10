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

// TestConsoleVerbose tests that the console shows what the scripts of a
// page do when it is verbose (see [SettingsData.VerboseConsole]): their
// requests, the properties of the browser that they get that it does not
// have, and the messages that they post, and none of that when it is not.
func TestConsoleVerbose(t *testing.T) {
	for _, verbose := range []bool{true, false} {
		Settings.VerboseConsole = verbose
		b, pg := newScriptPage(t, map[string]string{
			"/": `<html><body><script>
				fetch("/data.json").then(r => r.text()).then(() => { document.body.dataset.done = "1"; });
				const a = navigator.notInBrowsers;
				const b = typeof notDefinedAnywhere;
				const c = document.body.notAProperty;
				window.postMessage({ hello: "there" }, "*");
				// (a task that holds the page, as it gets a box, for long)
				setTimeout(() => { document.body.getBoundingClientRect(); const t = Date.now(); while (Date.now() - t < 150) {} }, 10);
				</script></body></html>`,
			"/data.json": `{"a": 1}`,
		})
		b.RenderImage(func() { settle(pg) }, func() {
			var all string
			ok := eventually(func() bool {
				all = strings.Join(pg.ScriptConsole(), "\n")
				return !verbose || strings.Contains(all, "data.json")
			})
			if !verbose {
				assert.NotContains(t, all, "missing:")
				assert.NotContains(t, all, "data.json")
				return
			}
			assert.True(t, ok, all)
			assert.Contains(t, all, "DEBUG: page https://example.com/: 1 scripts", "the page, as the first message")
			assert.Contains(t, all, "GET https://example.com/data.json (0 bytes): 200, 8 bytes")
			assert.Contains(t, all, "missing: navigator.notInBrowsers")
			assert.Contains(t, all, "missing: window.notDefinedAnywhere")
			assert.Contains(t, all, "missing: <body>.notAProperty")
			assert.Contains(t, all, `postMessage to https://example.com: {"hello":"there"}`)
			assert.True(t, eventually(func() bool {
				return strings.Contains(strings.Join(pg.ScriptConsole(), "\n"), "a task of the scripts held the page for")
			}), "a long task")
		})
	}
	Settings.VerboseConsole = false
}

// TestConsolePanelVerbose tests that the console panel shows the messages
// of a verbose console (see [SettingsData.VerboseConsole]) of a page that
// is opened after it is set.
func TestConsolePanelVerbose(t *testing.T) {
	Settings.VerboseConsole = true
	defer func() { Settings.VerboseConsole = false }()
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.X.Dp(800); s.Min.Y.Dp(600) })
	br := NewBrowser(b)
	br.PageFunc = func(pg *Page) {
		pg.Context.GetURL = func(u string) (*http.Response, error) {
			pu, _ := url.Parse(u)
			r := strings.NewReader(`<html><body><p>page</p><script>fetch("/data.txt")</script></body></html>`)
			return &http.Response{StatusCode: 200, Status: "200 OK", Request: &http.Request{URL: pu}, Body: io.NopCloser(r)}, nil
		}
	}
	br.NewTab("https://example.com/one")
	b.RenderImage(func() {
		for _, pg := range br.Pages() {
			settle(pg)
		}
		b.AsyncLock()
		br.ShowConsole(true)
		b.AsyncUnlock()
		assert.True(t, eventually(func() bool { return strings.Contains(consoleTexts(b, br), "DEBUG: GET https://example.com/data.txt") }), consoleTexts(b, br))
	})
}

// TestConsoleShownBySplitter tests that the console panel shows the
// messages of the page when it is shown by dragging the splitter of the
// browser (by its splits), not by ShowConsole, as it never did.
func TestConsoleShownBySplitter(t *testing.T) {
	b := core.NewBody()
	b.Styler(func(s *styles.Style) { s.Min.X.Dp(800); s.Min.Y.Dp(600) })
	br := NewBrowser(b)
	br.PageFunc = func(pg *Page) {
		pg.Context.GetURL = func(u string) (*http.Response, error) {
			pu, _ := url.Parse(u)
			r := strings.NewReader(`<html><body><p>page</p><script>console.log("hello"); setTimeout(() => console.log("later"), 500);</script></body></html>`)
			return &http.Response{StatusCode: 200, Status: "200 OK", Request: &http.Request{URL: pu}, Body: io.NopCloser(r)}, nil
		}
	}
	br.NewTab("https://example.com/")
	b.RenderImage(func() {
		for _, pg := range br.Pages() {
			settle(pg)
		}
		b.AsyncLock()
		br.Splits.SetSplits(0.7, 0.3) // (as by dragging the splitter)
		br.Splits.NeedsLayout()
		b.AsyncUnlock()
	}, func() {
		// (the splits give the panel its size as they render, after the layout)
		b.AsyncLock()
		br.Splits.NeedsLayout()
		b.AsyncUnlock()
	}, func() {
		assert.True(t, eventually(func() bool { return strings.Contains(consoleTexts(b, br), "INFO: hello") }), consoleTexts(b, br))
		assert.True(t, eventually(func() bool { return strings.Contains(consoleTexts(b, br), "INFO: later") }), "and the messages that come later: "+consoleTexts(b, br))
	})
}
