// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSaveDOMFrames tests saving the document of a page with a frame,
// which is saved in a file of its own, and the errors of saving: without
// scripts, and without a Downloads directory. The files are in a
// temporary home directory.
func TestSaveDOMFrames(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	frame := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<html><body><script>document.body.append("in the frame")</script></body></html>`)
	}))
	defer frame.Close()
	b, pg := newScriptPage(t, map[string]string{
		"/": `<html><body><p>top</p><iframe src="` + frame.URL + `/f" width="100" height="50"></iframe><script>1</script></body></html>`,
	})
	core.RenderImage(b, func() { settle(pg) }, func() {
		require.True(t, eventually(func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			for _, fp := range pg.frames {
				return fp.scripts != nil
			}
			return false
		}), "the frame")
		for _, fp := range pg.frames {
			settle(fp)
		}
		dir, err := os.UserHomeDir()
		require.NoError(t, err)
		require.Equal(t, home, dir) // (the files are in the temporary directory)
		pg.saveDOM()                // no Downloads: an error
		require.NoError(t, os.MkdirAll(filepath.Join(home, "Downloads"), 0o755))
		pg.saveDOM()
	})
	files, err := filepath.Glob(filepath.Join(home, "Downloads", "example.com-*-frame1-127.0.0.1:*.html"))
	require.NoError(t, err)
	require.Len(t, files, 1)
	data, err := os.ReadFile(files[0])
	require.NoError(t, err)
	assert.Contains(t, string(data), "in the frame")

	b, pg = newScriptPageWith(t, map[string]string{"/": `<p>no scripts</p>`}, func(pg *Page) { pg.Scripts = false })
	core.RenderImage(b, func() { settle(pg); pg.saveDOM() })
}

// TestKeyProps tests the properties of the keyboard events of the scripts
// for keys: named keys, letters, digits, space, and other runes.
func TestKeyProps(t *testing.T) {
	k := func(code key.Codes, r rune) map[string]any {
		return keyProps(events.NewKey(events.KeyDown, r, code, 0))
	}
	p := k(key.CodeReturnEnter, 0)
	assert.Equal(t, "Enter", p["key"])
	assert.Equal(t, 13, p["keyCode"])
	p = k(key.CodeA, 'a')
	assert.Equal(t, "a", p["key"])
	assert.Equal(t, "KeyA", p["code"])
	assert.Equal(t, int('A'), p["keyCode"])
	assert.Equal(t, "Digit5", k(key.Code5, '5')["code"])
	assert.Equal(t, "Space", k(key.CodeSpacebar, ' ')["code"])
	p = k(0, '€')
	assert.Equal(t, "€", p["key"])
	assert.Equal(t, 0, p["keyCode"])
	assert.NotContains(t, keyProps(events.NewKey(events.KeyDown, 0, 0, 0)), "key")
}

// TestScriptsBlockAdsVerbose tests a page with scripts with the ads
// blocked and the verbose console: the requests that fail and those of
// typed resources are in the console, and the images of noscripts (as of
// lazy images) are set once the page is shown.
func TestScriptsBlockAdsVerbose(t *testing.T) {
	savedAds, savedVerbose := Settings.BlockAds, Settings.VerboseConsole
	Settings.BlockAds, Settings.VerboseConsole = true, true
	t.Cleanup(func() { Settings.BlockAds, Settings.VerboseConsole = savedAds, savedVerbose })
	b := core.NewBody()
	pg := NewPage(b)
	pg.Context.GetURL = func(u string) (*http.Response, error) {
		pu, _ := url.Parse(u)
		switch pu.Path {
		case "/fail.css":
			return nil, errors.New("no network")
		case "/s.css":
			return &http.Response{StatusCode: 200, Status: "200 OK", Request: &http.Request{URL: pu}, Header: http.Header{"Content-Type": {"text/css"}},
				Body: io.NopCloser(strings.NewReader("p { color: red }"))}, nil
		case "/i.png":
			var buf bytes.Buffer
			png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4)))
			return &http.Response{StatusCode: 200, Status: "200 OK", Request: &http.Request{URL: pu}, Header: http.Header{"Content-Type": {"image/png"}},
				Body: io.NopCloser(&buf)}, nil
		}
		return &http.Response{StatusCode: 200, Status: "200 OK", Request: &http.Request{URL: pu}, Body: io.NopCloser(strings.NewReader(
			`<html><head><link rel="stylesheet" href="/s.css"><link rel="stylesheet" href="/fail.css"></head><body><p>page</p><noscript><img src="/i.png"></noscript><script>1</script></body></html>`))}, nil
	}
	pg.OpenURL("https://example.com/", nil)
	core.RenderImage(b, func() { settle(pg) }, func() {
		require.True(t, eventually(func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			all := strings.Join(pg.ScriptConsole(), "\n")
			return strings.Contains(all, "fail.css") && strings.Contains(all, "s.css")
		}), "the console")
	})
}
