// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"cogentcore.org/core/core"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPersistentCookies tests that the persistent cookies (not session
// cookies) are saved and loaded again, with their attributes, and that
// deleted and cleared cookies are not.
func TestPersistentCookies(t *testing.T) {
	file := filepath.Join(t.TempDir(), "web", "cookies.json")
	savedFile, savedDelay := cookiesFile, cookieSaveDelay
	cookiesFile = func() string { return file }
	cookieSaveDelay = time.Millisecond
	t.Cleanup(func() { cookiesFile, cookieSaveDelay = savedFile, savedDelay })

	waitSaved := func(pj *persistentJar) {
		t.Helper()
		assert.Eventually(t, func() bool {
			pj.mu.Lock()
			defer pj.mu.Unlock()
			return pj.timer == nil
		}, 2*time.Second, 5*time.Millisecond)
	}
	u, _ := url.Parse("https://www.example.com/a/b")
	pj := newPersistentJar()
	pj.SetCookies(u, []*http.Cookie{
		{Name: "consent", Value: "yes", Domain: "example.com", Path: "/", MaxAge: 3600, Secure: true},
		{Name: "session", Value: "s"},
		{Name: "gone", Value: "g", Expires: time.Now().Add(time.Hour)},
		{Name: "public", Value: "p", Domain: "com", MaxAge: 3600}, // a public suffix: refused
	})
	pj.SetCookies(u, []*http.Cookie{{Name: "gone", Value: "", MaxAge: -1}})
	waitSaved(pj)
	fi, err := os.Stat(file)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), fi.Mode().Perm())

	names := func(pj *persistentJar, u *url.URL) map[string]string {
		m := map[string]string{}
		for _, c := range pj.Cookies(u) {
			m[c.Name] = c.Value
		}
		return m
	}
	other, _ := url.Parse("https://mail.example.com/")
	plain, _ := url.Parse("http://mail.example.com/")
	again := newPersistentJar()
	assert.Equal(t, map[string]string{"consent": "yes"}, names(again, u))
	assert.Equal(t, map[string]string{"consent": "yes"}, names(again, other), "domain cookie")
	assert.Empty(t, names(again, plain), "secure cookie")

	again.Clear()
	waitSaved(again)
	assert.Empty(t, names(newPersistentJar(), u))
}

// TestCookieEditor tests the cookie editor: it lists the saved and the
// session cookies of a site, and editing the value of a row sets the
// cookie, and deleting a row (as in its table) deletes it, as to be a new
// visitor of google.com after it took the pages of the app for those of a
// bot (with a page of status 429).
func TestCookieEditor(t *testing.T) {
	file := filepath.Join(t.TempDir(), "web", "cookies.json")
	savedFile := cookiesFile
	cookiesFile = func() string { return file }
	t.Cleanup(func() { cookiesFile = savedFile })

	g, _ := url.Parse("https://www.google.com/search")
	o, _ := url.Parse("https://other.org/")
	pj := newPersistentJar()
	exp := time.Date(2036, 1, 2, 12, 0, 0, 0, time.Local) // fixed, for the image of the editor
	pj.SetCookies(g, []*http.Cookie{
		{Name: "NID", Value: "flagged", Domain: ".google.com", Path: "/", Expires: exp},
		{Name: "AEC", Value: "a", Domain: ".google.com", Path: "/", Expires: exp},
		{Name: "sess", Value: "s", Path: "/"},
	})
	pj.SetCookies(o, []*http.Cookie{{Name: "x", Value: "1", MaxAge: 3600}})

	rows := cookieRows(pj, "google.com")
	names := func(rs []*CookieRow) []string {
		var ns []string
		for _, r := range rs {
			ns = append(ns, r.Name)
		}
		return ns
	}
	assert.Equal(t, []string{"AEC", "NID", "sess"}, names(rows), "the saved and session cookies of the site")
	assert.Len(t, cookieRows(pj, ""), 4, "all sites")

	edited := []*CookieRow{rows[0], rows[2]} // NID deleted
	edited[0].Value = "b"
	applyCookieRows(pj, rows, edited)
	sent := map[string]string{}
	for _, c := range pj.Cookies(g) {
		sent[c.Name] = c.Value
	}
	assert.Equal(t, map[string]string{"AEC": "b", "sess": "s"}, sent, "NID deleted, AEC edited")
	assert.Equal(t, []string{"AEC", "sess"}, names(cookieRows(pj, "google")))
	assert.Len(t, pj.Cookies(o), 1, "the cookie of the other site is kept")

	b := core.NewBody()
	makeCookieEditor(b, pj, "google")
	b.AssertRender(t, "cookie-editor")
}
