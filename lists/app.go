// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package lists is Cogent Lists, an app for lists, such as to-do and
// shopping lists, in groups, with the items of each list in categories,
// and the fields of the items given by the type of the list. The apps
// for desktop and mobile are in cmd/lists and cmd/mobile.
package lists

//go:generate core generate

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
)

// NewApp makes the app in the given body, with the lists in the given
// directory, or the app data directory if it is empty, with the toolbar
// in the top bar, and syncing once the body is shown.
func NewApp(b *core.Body, dir string) *Lists {
	if dir == "" {
		dir = core.TheApp.AppDataDir()
	}
	ls := NewLists(b)
	theLists = ls
	errors.Log(ls.Open(dir))
	b.OnShow(func(e events.Event) {
		ls.StartSync()
	})
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(ls.MakeToolbar)
	})
	return ls
}
