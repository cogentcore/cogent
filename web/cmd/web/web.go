// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	_ "embed"
	"os"
	"path/filepath"

	"cogentcore.org/cogent/web"
	"cogentcore.org/core/core"
)

// appIcon is the icon of the app, which core build also sets, but go
// install does not, as for an app installed with AppGoSymlink.
//
//go:embed icon.svg
var appIcon string

func init() {
	core.TheApp.SetName("Cogent Web")
	web.Settings.File = filepath.Join(core.TheApp.AppDataDir(), "settings.toml") // makes the directory
	core.AddAppSettings(web.Settings)
}

func main() {
	core.AppIcon = appIcon
	b := core.NewBody("Cogent Web")
	br := web.InitBody(b)
	// the pages to open are the URLs or local files in the arguments,
	// each in a tab
	urls := os.Args[1:]
	if len(urls) == 0 {
		urls = []string{"https://google.com"}
	}
	for _, u := range urls {
		br.NewTab(web.LocationURL(u))
	}
	br.Tabs.SelectTabIndex(0)
	b.RunMainWindow()
}
