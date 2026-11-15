// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCookieEditorEdits tests the editor of the cookies: filtering them
// by site, editing their values, and deleting those that are shown, in a
// jar of the test (saved in a temporary file).
func TestCookieEditorEdits(t *testing.T) {
	file := filepath.Join(t.TempDir(), "cookies.json")
	savedFile, savedDelay := cookiesFile, cookieSaveDelay
	cookiesFile = func() string { return file }
	cookieSaveDelay = time.Millisecond
	t.Cleanup(func() { cookiesFile, cookieSaveDelay = savedFile, savedDelay })

	pj := newPersistentJar()
	ex, _ := url.Parse("https://example.com/")
	other, _ := url.Parse("https://other.org/")
	pj.SetCookies(ex, []*http.Cookie{{Name: "a", Value: "1", MaxAge: 3600}, {Name: "b", Value: "2", MaxAge: 3600}})
	pj.SetCookies(other, []*http.Cookie{{Name: "o", Value: "x", MaxAge: 3600}})
	values := func(u *url.URL) map[string]string {
		m := map[string]string{}
		for _, c := range pj.Cookies(u) {
			m[c.Name] = c.Value
		}
		return m
	}

	b := core.NewBody()
	makeCookieEditor(b, pj, "example.com")
	var tf *core.TextField
	var table *core.Table
	for w := range b.AllWidgets() {
		switch x := w.(type) {
		case *core.TextField:
			if tf == nil {
				tf = x
			}
		case *core.Table:
			table = x
		}
	}
	require.NotNil(t, tf)
	require.NotNil(t, table)
	rows := func() []*CookieRow { return *table.Slice.(*[]*CookieRow) }
	core.RenderImage(b, core.Locked(b, func() {
		require.Len(t, rows(), 2, "the cookies of the site")
		rows()[0].Value = "changed"
		table.SendChange()
		assert.Equal(t, "changed", values(ex)[rows()[0].Name])

		tf.SetText("")
		tf.SendChange()
		assert.Len(t, rows(), 3, "all sites")
		core.ButtonByText(b.Scene, "Refresh").Send(events.Click)
		assert.Len(t, rows(), 3)

		tf.SetText("other.org")
		tf.SendChange()
		require.Len(t, rows(), 1)
		// (only the cookie of other.org, as checked)
		require.Equal(t, "o", rows()[0].Name)
		core.ButtonByText(b.Scene, "Delete shown").Send(events.Click)
		assert.Empty(t, rows())
		assert.Empty(t, values(other))
		assert.Len(t, values(ex), 2, "the others are kept")
	}))
}
