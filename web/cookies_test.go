// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
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
