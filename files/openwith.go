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
// tables and the file tree: Open with, a menu of the apps that can open it,
// with the default one first, and Other..., to choose from all of the apps,
// and Change default app.
func openWithMenuItems(m *core.Scene, file string) {
	as, err := apps.For(file)
	if err != nil {
		return // not supported
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
			if len(as) > 0 {
				core.NewSeparator(m)
			}
			core.NewButton(m).SetText(otherApps).
				SetTooltip("Open the file with one of all of the apps").
				OnClick(func(e events.Event) {
					openWithOther(m, file)
				})
		}).SetTooltip("Open the file with one of the apps that can open it")
	core.NewButton(m).SetText("Change default app").SetIcon(icons.Edit).
		SetTooltip("Change the default app for this type of file").
		OnClick(func(e events.Event) {
			changeDefaultApp(m, file, as)
		})
}

// otherApps is the text of the choices of all of the apps.
const otherApps = "Other..."

// newAppChooser makes a chooser of the given apps, with an Other... item
// at the end, which replaces them with all of the apps, or of all of the
// apps if there are none. It can be typed in to find an app.
func newAppChooser(parent core.Widget, as []App) *core.Chooser {
	ch := core.NewChooser(parent).SetEditable(true)
	setApps := func(as []App, other bool) {
		ch.Items = ch.Items[:0]
		for _, app := range as {
			ch.Items = append(ch.Items, core.ChooserItem{Value: app, Text: app.Name})
		}
		if other {
			ch.Items = append(ch.Items, core.ChooserItem{Value: otherApps, Text: otherApps,
				Tooltip: "Choose from all of the apps"})
		}
		if len(ch.Items) > 0 {
			ch.SetCurrentIndex(0) // the default, if any, is first
		}
	}
	all := func() {
		all, err := apps.All()
		core.ErrorSnackbar(ch, err)
		setApps(all, false)
	}
	if len(as) == 0 {
		all()
	} else {
		setApps(as, true)
	}
	ch.OnChange(func(e events.Event) {
		if ch.CurrentItem.Value == otherApps {
			all()
			ch.Update()
		}
	})
	return ch
}

// chosenApp returns the app chosen in the given chooser of [newAppChooser].
func chosenApp(ch *core.Chooser) (App, bool) {
	app, ok := ch.CurrentItem.Value.(App)
	return app, ok
}

// openWithOther shows a dialog to choose one of all of the apps
// to open the given file with.
func openWithOther(ctx core.Widget, file string) {
	d := core.NewBody("Open with")
	core.NewText(d).SetType(core.TextSupporting).
		SetText("The app to open " + filepath.Base(file) + " with: it may not open this type of file")
	ch := newAppChooser(d, nil)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Open").OnClick(func(e events.Event) {
			if app, ok := chosenApp(ch); ok {
				core.ErrorSnackbar(ctx, apps.OpenWith(file, app))
			}
		})
	})
	d.RunDialog(ctx)
}

// changeDefaultApp shows a dialog to choose the default app for the type of
// the given file from the given apps that can open it, or from all of them.
func changeDefaultApp(ctx core.Widget, file string, as []App) {
	d := core.NewBody("Default app")
	typ := filepath.Ext(file)
	if typ == "" {
		typ = filepath.Base(file)
	}
	core.NewText(d).SetType(core.TextSupporting).
		SetText("The default app for files like " + typ + ", which opens them in all apps")
	ch := newAppChooser(d, as)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Set default").OnClick(func(e events.Event) {
			if app, ok := chosenApp(ch); ok {
				core.ErrorSnackbar(ctx, apps.SetDefault(file, app))
			}
		})
	})
	d.RunDialog(ctx)
}

// App is an app of the system that opens files.
type App = apps.App
