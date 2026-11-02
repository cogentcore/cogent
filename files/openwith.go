// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"image"
	"path/filepath"

	"cogentcore.org/core/base/fileinfo/apps"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
)

// openWithMenuItems adds the context menu items for opening the given file,
// which is not a directory, with the apps of the system, in the directory
// tables and the file tree: Open with, a menu of the apps, with the default
// one first, and Change default app.
func openWithMenuItems(m *core.Scene, file string) {
	as, err := apps.For(file)
	if err != nil || len(as) == 0 {
		return
	}
	core.NewButton(m).SetText("Open with").SetIcon(icons.Open).
		SetMenu(func(m *core.Scene, pos image.Point) {
			for _, app := range as {
				text := app.Name
				if app.Default {
					text += " (default)"
				}
				core.NewButton(m).SetText(text).OnClick(func(e events.Event) {
					core.ErrorSnackbar(m, apps.OpenWith(file, app))
				})
			}
		}).SetTooltip("Open the file with one of the apps that can open it")
	core.NewButton(m).SetText("Change default app").SetIcon(icons.Edit).
		SetTooltip("Change the default app for this type of file").
		OnClick(func(e events.Event) {
			changeDefaultApp(m, file, as)
		})
}

// changeDefaultApp shows a dialog to choose the default app for the type of
// the given file from the given apps that can open it.
func changeDefaultApp(ctx core.Widget, file string, as []App) {
	d := core.NewBody("Default app")
	typ := filepath.Ext(file)
	if typ == "" {
		typ = filepath.Base(file)
	}
	core.NewText(d).SetType(core.TextSupporting).
		SetText("The default app for files like " + typ + ", which opens them in all apps")
	ch := core.NewChooser(d)
	for _, app := range as {
		ch.Items = append(ch.Items, core.ChooserItem{Value: app, Text: app.Name})
	}
	ch.SetCurrentIndex(0) // the default, if any, is first
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Set default").OnClick(func(e events.Event) {
			app := ch.CurrentItem.Value.(App)
			core.ErrorSnackbar(ctx, apps.SetDefault(file, app))
		})
	})
	d.RunDialog(ctx)
}

// App is an app of the system that opens files.
type App = apps.App
