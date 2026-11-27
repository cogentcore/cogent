// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"fmt"
	"strings"
	"sync"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/tree"
)

// viewMaxSize is the maximum size of the images of photos shown, which
// is enough for the screen.
const viewMaxSize = 3000

// Viewer shows a photo, with a line of information about it below it.
// The left and right keys go to the previous and next photos in the grid.
type Viewer struct {
	core.Frame

	// Photo is the photo shown.
	Photo *Photo `set:"-"`

	// app is the app.
	app *App

	image *core.Image
	info  *core.Text

	// gen is the number of the photo being loaded, so that only the
	// latest one is shown, under mu.
	gen int
	mu  sync.Mutex
}

func (v *Viewer) Init() {
	v.Frame.Init()
	v.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.SetAbilities(true, abilities.Focusable)
		s.Background = colors.Scheme.SurfaceContainerLowest
		s.Grow.Set(1, 1)
		s.Align.Items = styles.Center
	})
	v.OnClick(func(e events.Event) { v.SetFocus() })
	v.On(events.KeyChord, func(e events.Event) {
		g := v.app.grid
		switch keymap.Of(e.KeyChord()) {
		case keymap.MoveRight, keymap.MoveDown:
			e.SetHandled()
			g.SelectIndex(g.Current + 1)
		case keymap.MoveLeft, keymap.MoveUp:
			e.SetHandled()
			g.SelectIndex(g.Current - 1)
		}
	})
	tree.AddChild(v, func(w *core.Image) {
		v.image = w
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 1)
			s.Min.X.Zero()
			s.Min.Y.Zero()
			s.ObjectFit = styles.FitContain
		})
	})
	tree.AddChild(v, func(w *core.Text) {
		v.info = w
		w.SetType(core.TextSupporting)
		w.Updater(func() {
			w.SetText(v.infoText())
		})
	})
}

// infoText returns the line of information about the photo shown.
func (v *Viewer) infoText() string {
	ph := v.Photo
	if ph == nil {
		return ""
	}
	var parts []string
	parts = append(parts, ph.DateTaken.Format("Mon Jan 2, 2006 3:04 PM"))
	if len(ph.Tags) > 0 {
		var tags []string
		for _, t := range ph.Tags {
			y, n := splitYearTag(t)
			tags = append(tags, strings.TrimSpace(y+" "+n))
		}
		parts = append(parts, strings.Join(tags, ", "))
	}
	if ph.Desc != "" {
		parts = append(parts, ph.Desc)
	}
	sz := ph.DisplaySize()
	if sz.X > 0 {
		parts = append(parts, fmt.Sprintf("%d×%d", sz.X, sz.Y))
	}
	if cam := strings.TrimSpace(ph.Make + " " + ph.Model); cam != "" {
		parts = append(parts, cam)
	}
	parts = append(parts, ph.Filename())
	return strings.Join(parts, " · ")
}

// SetPhoto shows the given photo, loading its image in the background,
// with its thumbnail shown first.
func (v *Viewer) SetPhoto(ph *Photo) {
	v.Photo = ph
	v.mu.Lock()
	v.gen++
	gen := v.gen
	v.mu.Unlock()
	if ph == nil || v.app == nil || v.app.lib == nil {
		v.image.SetImage(nil)
		v.Update()
		return
	}
	g := v.app.grid
	g.thumbsMu.Lock()
	thumb := g.thumbs[ph.Name]
	g.thumbsMu.Unlock()
	v.image.SetImage(thumb)
	v.Update()
	lib := v.app.lib
	go func() {
		img, err := openPhoto(ph, lib.PhotoFile(ph), viewMaxSize)
		if err == nil {
			img = scaleToFit(img, viewMaxSize)
		}
		v.mu.Lock()
		current := gen == v.gen
		v.mu.Unlock()
		if !current {
			return
		}
		v.AsyncLock()
		defer v.AsyncUnlock()
		if err != nil {
			core.ErrorSnackbar(v, err, "Error opening "+ph.Filename())
			return
		}
		v.image.SetImage(img)
		v.image.NeedsRender()
	}()
}
