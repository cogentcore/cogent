// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"cogentcore.org/core/htmlcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSettingsUserAgent tests that the user agent of the settings is sent
// with the requests of pages and is the navigator.userAgent of their
// scripts, and that clearing it gives the default one.
func TestSettingsUserAgent(t *testing.T) {
	defer func() {
		Settings.UserAgent = ""
		Settings.Apply()
	}()
	Settings.UserAgent = " Test/1.0 "
	Settings.Apply()
	assert.Equal(t, "Test/1.0", Settings.UserAgent)

	got := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case got <- r.Header.Get("User-Agent"):
		default:
		}
	}))
	defer srv.Close()
	resp, err := htmlcore.NewContext().GetURL(srv.URL)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, "Test/1.0", <-got, "the header of requests")

	b, pg := newScriptPage(t, map[string]string{"/": `<html><body><p>page</p></body></html>`})
	b.RenderImage(func() { settle(pg) }, func() {
		require.NotNil(t, pg.scripts)
		v, err := pg.scripts.engine.RunString(`navigator.userAgent`)
		require.NoError(t, err)
		assert.Equal(t, "Test/1.0", v, "navigator.userAgent")
	})

	Settings.UserAgent = ""
	Settings.Apply()
	assert.Equal(t, defaultUserAgent, Settings.UserAgent, "the default when it is cleared")
	assert.Equal(t, defaultUserAgent, htmlcore.UserAgent)
}
