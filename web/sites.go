// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"errors"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/core"
)

// SiteSettings are the settings of the pages of a site (a host, as
// www.example.com), which the browser keeps for it, as the zoom of its
// pages. The zero value is the default for each setting.
type SiteSettings struct {

	// Zoom is the zoom factor of the pages of the site, if positive
	// (1 is 100%); otherwise, they are not zoomed.
	Zoom float32 `json:",omitempty"`
}

// isDefault returns whether all of the settings are the defaults, so that
// the site does not need to be saved.
func (ss *SiteSettings) isDefault() bool {
	return *ss == SiteSettings{}
}

// zoom returns the zoom factor of the site (1 if it is not zoomed).
func (ss *SiteSettings) zoom() float32 {
	if ss.Zoom > 0 {
		return ss.Zoom
	}
	return 1
}

// sites has the settings of all of the sites that have settings other
// than the defaults, by host, which are saved in [sitesFile] when they
// are changed, and loaded from it when they are first used.
var sites struct {
	sync.Mutex

	// loaded is whether the sites have been loaded from the file.
	loaded bool

	// settings are the settings of the sites, by host.
	settings map[string]SiteSettings
}

// sitesFile returns the file that the settings of the sites are saved in.
// It is a variable for tests.
var sitesFile = func() string {
	return filepath.Join(core.TheApp.AppDataDir(), "web", "sites.json")
}

// siteOf returns the site (host) of the given URL, in lowercase, or ""
// if it has none (as a file or data URL).
func siteOf(u string) string {
	pu, err := url.Parse(u)
	if err != nil {
		return ""
	}
	return strings.ToLower(pu.Hostname())
}

// loadSites loads the settings of the sites, if they have not been
// loaded yet (with the lock of sites held).
func loadSites() {
	if sites.loaded {
		return
	}
	sites.loaded = true
	sites.settings = map[string]SiteSettings{}
	err := jsonx.Open(&sites.settings, sitesFile())
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		slog.Error("web: loading the settings of sites", "err", err)
	}
	if sites.settings == nil {
		sites.settings = map[string]SiteSettings{}
	}
}

// Site returns the settings of the site of the given URL.
func Site(u string) SiteSettings {
	sites.Lock()
	defer sites.Unlock()
	loadSites()
	return sites.settings[siteOf(u)]
}

// SetSite changes the settings of the site of the given URL with the given
// function, and saves the settings of all of the sites. Sites with
// default settings are not saved. It does nothing for URLs without a site.
func SetSite(u string, change func(ss *SiteSettings)) {
	site := siteOf(u)
	if site == "" {
		return
	}
	sites.Lock()
	defer sites.Unlock()
	loadSites()
	ss := sites.settings[site]
	change(&ss)
	if ss.isDefault() {
		delete(sites.settings, site)
	} else {
		sites.settings[site] = ss
	}
	file := sitesFile()
	if err := os.MkdirAll(filepath.Dir(file), 0750); err != nil {
		slog.Error("web: saving the settings of sites", "err", err)
		return
	}
	if err := jsonx.Save(sites.settings, file); err != nil {
		slog.Error("web: saving the settings of sites", "err", err)
	}
}
