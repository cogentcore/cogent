// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"cmp"
	"net/url"
	"slices"
	"strings"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// CookieRow is a cookie in the cookie editor (see [ShowCookies]): its
// value can be edited, and deleting its row deletes it.
type CookieRow struct {

	// Site is the domain of the cookie (or the host that set it).
	Site string `edit:"-"`

	// Name is the name of the cookie.
	Name string `edit:"-"`

	// Value is the value of the cookie.
	Value string `width:"40"`

	// Path is the path of the cookie, if it is not the default.
	Path string `edit:"-"`

	// Expires is when the cookie expires, or zero for a session cookie,
	// which is not saved.
	Expires time.Time `edit:"-"`

	// cookie is the cookie as listed by the jar, and key is its key.
	cookie savedCookie
	key    string
}

// cookieRows returns the rows of the cookies of the given jar whose site
// contains the given filter (all of them if it is empty), by site and name.
func cookieRows(pj *persistentJar, filter string) []*CookieRow {
	filter = strings.ToLower(strings.TrimSpace(filter))
	var rows []*CookieRow
	for key, sc := range pj.list() {
		site := strings.TrimPrefix(sc.Domain, ".")
		if site == "" {
			if u, err := url.Parse(sc.URL); err == nil {
				site = u.Hostname()
			}
		}
		if filter != "" && !strings.Contains(strings.ToLower(site), filter) {
			continue
		}
		rows = append(rows, &CookieRow{Site: site, Name: sc.Name, Value: sc.Value, Path: sc.Path, Expires: sc.Expires, cookie: sc, key: key})
	}
	slices.SortFunc(rows, func(a, b *CookieRow) int {
		return cmp.Or(cmp.Compare(a.Site, b.Site), cmp.Compare(a.Name, b.Name), cmp.Compare(a.Path, b.Path))
	})
	return rows
}

// applyCookieRows applies the edits of the given rows to the jar: the
// cookies of the rows that were listed before and are no longer are
// removed, and those whose values changed are set.
func applyCookieRows(pj *persistentJar, before, after []*CookieRow) {
	kept := map[string]bool{}
	for _, r := range after {
		kept[r.key] = true
		if r.Value != r.cookie.Value {
			pj.setValue(r.cookie, r.Value)
			r.cookie.Value = r.Value
		}
	}
	for _, r := range before {
		if !kept[r.key] {
			pj.remove(r.cookie)
		}
	}
}

// ShowCookies shows the cookies of the pages in a new window, those of the
// sites that contain the given site (as the host of the current page), or
// all of them if it is empty, in a table in which their values can be
// edited and they can be deleted (as to be a new visitor of a site, as of
// google.com after it has taken its pages for those of a bot).
func ShowCookies(site string) {
	b := core.NewBody("Cookies")
	makeCookieEditor(b, cookies, site)
	b.RunWindow()
}

// makeCookieEditor makes the cookie editor of [ShowCookies] in the given
// body, for the cookies of the given jar.
func makeCookieEditor(b *core.Body, pj *persistentJar, site string) {
	filter := site
	var rows, listed []*CookieRow
	load := func() {
		rows = cookieRows(pj, filter)
		listed = slices.Clone(rows)
	}
	load()
	var table *core.Table
	reload := func() {
		load()
		table.SetSlice(&rows)
		table.Update()
	}
	tf := core.NewTextField(b).SetPlaceholder("Site (all sites if empty)").SetText(filter)
	tf.Styler(func(s *styles.Style) { s.Min.X.Em(30) })
	tf.OnChange(func(e events.Event) {
		filter = tf.Text()
		reload()
	})
	table = core.NewTable(b).SetSlice(&rows)
	table.OnChange(func(e events.Event) {
		applyCookieRows(pj, listed, rows)
		listed = slices.Clone(rows)
	})
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(func(p *tree.Plan) {
			tree.Add(p, func(w *core.Button) {
				w.SetText("Delete shown").SetIcon(icons.Delete).SetTooltip("Delete the cookies that are shown")
				w.OnClick(func(e events.Event) {
					for _, r := range rows {
						pj.remove(r.cookie)
					}
					reload()
				})
			})
			tree.Add(p, func(w *core.Button) {
				w.SetText("Refresh").SetIcon(icons.Refresh).SetTooltip("List the cookies again, as pages have set them")
				w.OnClick(func(e events.Event) { reload() })
			})
		})
	})
}
