// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/core"
	"golang.org/x/net/publicsuffix"
)

// cookiesFile returns the file that the cookies are saved in. It is a
// variable for tests.
var cookiesFile = func() string {
	return filepath.Join(core.TheApp.AppDataDir(), "web", "cookies.json")
}

// cookieSaveDelay is how long after a change the cookies are saved, so
// that the many cookies that a page sets are saved once.
var cookieSaveDelay = 2 * time.Second

// savedCookie is a cookie that is saved, with the URL that set it.
type savedCookie struct {
	URL      string
	Name     string
	Value    string
	Path     string `json:",omitempty"`
	Domain   string `json:",omitempty"`
	Expires  time.Time
	Secure   bool          `json:",omitempty"`
	HttpOnly bool          `json:",omitempty"`
	SameSite http.SameSite `json:",omitempty"`
}

// persistentJar is a cookie jar whose persistent cookies (those with an
// expiry, not session cookies) are saved in the [cookiesFile], as browsers
// do, so that sites see a returning visitor (and the choices made in
// their cookie consent forms are kept). It wraps a [cookiejar.Jar], which
// cannot list its cookies, recording the cookies that are set in it, and
// setting them in it again when they are loaded. It loads the cookies when
// it is first used.
type persistentJar struct {
	mu     sync.Mutex
	jar    *cookiejar.Jar
	loaded bool
	saved  map[string]savedCookie // by domain, path, and name
	// session are the session cookies (with no expiry), which are not
	// saved, by domain, path, and name, to list them (see [ShowCookies]).
	session map[string]savedCookie
	timer  *time.Timer
}

// newPersistentJar returns a new [persistentJar], with the public suffix
// list, so that sites cannot set cookies for public suffixes (as co.uk).
func newPersistentJar() *persistentJar {
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	return &persistentJar{jar: jar}
}

// cookieKey returns the key of the given cookie set by the given URL,
// which is the same for cookies that replace each other.
func cookieKey(u *url.URL, c *http.Cookie) string {
	domain := strings.ToLower(strings.TrimPrefix(c.Domain, "."))
	if domain == "" {
		domain = "=" + strings.ToLower(u.Hostname()) // host only
	}
	path := c.Path
	if path == "" || path[0] != '/' { // the default path
		path = u.Path
		if i := strings.LastIndex(path, "/"); i > 0 {
			path = path[:i]
		} else {
			path = "/"
		}
	}
	return domain + ";" + path + ";" + c.Name
}

// load loads the saved cookies, if they have not been loaded yet (with the
// lock held), setting those that have not expired in the jar.
func (pj *persistentJar) load() {
	if pj.loaded {
		return
	}
	pj.loaded = true
	var list []savedCookie
	err := jsonx.Open(&list, cookiesFile())
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		slog.Error("web: loading cookies", "err", err)
	}
	pj.saved = map[string]savedCookie{}
	pj.session = map[string]savedCookie{}
	now := time.Now()
	for _, sc := range list {
		u, err := url.Parse(sc.URL)
		if err != nil || !sc.Expires.After(now) {
			continue
		}
		c := &http.Cookie{Name: sc.Name, Value: sc.Value, Path: sc.Path, Domain: sc.Domain,
			Expires: sc.Expires, Secure: sc.Secure, HttpOnly: sc.HttpOnly, SameSite: sc.SameSite}
		pj.jar.SetCookies(u, []*http.Cookie{c})
		pj.saved[cookieKey(u, c)] = sc
	}
}

// SetCookies sets the given cookies of the given URL in the jar, and
// records the persistent ones to be saved (or removes those that are
// deleted or expired).
func (pj *persistentJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	pj.mu.Lock()
	defer pj.mu.Unlock()
	pj.load()
	pj.jar.SetCookies(u, cookies)
	now := time.Now()
	changed := false
	for _, c := range cookies {
		key := cookieKey(u, c)
		expires := c.Expires
		if c.MaxAge > 0 {
			expires = now.Add(time.Duration(c.MaxAge) * time.Second)
		}
		sc := savedCookie{URL: u.Scheme + "://" + u.Host + u.Path, Name: c.Name, Value: c.Value,
			Path: c.Path, Domain: c.Domain, Secure: c.Secure, HttpOnly: c.HttpOnly, SameSite: c.SameSite}
		if c.MaxAge < 0 || expires.IsZero() || !expires.After(now) {
			if _, ok := pj.saved[key]; ok { // deleted, expired, or now a session cookie
				delete(pj.saved, key)
				changed = true
			}
			if c.MaxAge == 0 && expires.IsZero() {
				pj.session[key] = sc
			} else {
				delete(pj.session, key)
			}
			continue
		}
		delete(pj.session, key)
		sc.Expires = expires
		pj.saved[key] = sc
		changed = true
	}
	if changed {
		pj.saveLater()
	}
}

// Cookies returns the cookies to send to the given URL.
func (pj *persistentJar) Cookies(u *url.URL) []*http.Cookie {
	pj.mu.Lock()
	defer pj.mu.Unlock()
	pj.load()
	return pj.jar.Cookies(u)
}

// Clear removes all of the cookies, including those saved.
func (pj *persistentJar) Clear() {
	pj.mu.Lock()
	defer pj.mu.Unlock()
	pj.loaded = true
	pj.jar, _ = cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	pj.saved = map[string]savedCookie{}
	pj.session = map[string]savedCookie{}
	pj.saveLater()
}

// list returns the cookies that have not expired, saved and session ones,
// with their keys, by domain and name.
func (pj *persistentJar) list() map[string]savedCookie {
	pj.mu.Lock()
	defer pj.mu.Unlock()
	pj.load()
	now := time.Now()
	all := map[string]savedCookie{}
	for k, sc := range pj.saved {
		if sc.Expires.After(now) {
			all[k] = sc
		}
	}
	for k, sc := range pj.session {
		all[k] = sc
	}
	return all
}

// remove removes the given cookie (as listed by list).
func (pj *persistentJar) remove(sc savedCookie) {
	u, err := url.Parse(sc.URL)
	if err != nil {
		return
	}
	pj.SetCookies(u, []*http.Cookie{{Name: sc.Name, Path: sc.Path, Domain: sc.Domain, MaxAge: -1}})
}

// setValue sets the value of the given cookie (as listed by list).
func (pj *persistentJar) setValue(sc savedCookie, value string) {
	u, err := url.Parse(sc.URL)
	if err != nil {
		return
	}
	pj.SetCookies(u, []*http.Cookie{{Name: sc.Name, Value: value, Path: sc.Path, Domain: sc.Domain,
		Expires: sc.Expires, Secure: sc.Secure, HttpOnly: sc.HttpOnly, SameSite: sc.SameSite}})
}

// saveLater saves the cookies after the [cookieSaveDelay] (with the lock
// held), unless they are already about to be saved.
func (pj *persistentJar) saveLater() {
	if pj.timer != nil {
		return
	}
	pj.timer = time.AfterFunc(cookieSaveDelay, pj.save)
}

// save saves the persistent cookies that have not expired.
func (pj *persistentJar) save() {
	pj.mu.Lock()
	pj.timer = nil
	now := time.Now()
	list := []savedCookie{}
	for key, sc := range pj.saved {
		if sc.Expires.After(now) {
			list = append(list, sc)
		} else {
			delete(pj.saved, key)
		}
	}
	pj.mu.Unlock()
	file := cookiesFile()
	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		slog.Error("web: saving cookies", "err", err)
		return
	}
	if err := jsonx.Save(list, file); err != nil {
		slog.Error("web: saving cookies", "err", err)
		return
	}
	os.Chmod(file, 0600) // cookies can sign in to accounts
}

// ClearCookies removes all of the cookies of the pages, including those
// saved.
func ClearCookies() {
	cookies.Clear()
}
