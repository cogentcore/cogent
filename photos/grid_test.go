// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"fmt"
	"image"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
)

// TestGridScrollClick tests that a photo clicked after scrolling far is
// the one under the mouse, which stays in view.
func TestGridScrollClick(t *testing.T) {
	b := core.NewBody()
	fr := core.NewFrame(b)
	fr.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Overflow.Set(styles.OverflowAuto)
		s.Min.Set(units.Dp(800), units.Dp(600))
		s.Max.Set(units.Dp(800), units.Dp(600))
	})
	g := NewGrid(fr)
	var phs []*Photo
	for i := range 200000 {
		phs = append(phs, &Photo{Name: fmt.Sprint(i), DateTaken: time.Unix(int64(i), 0)})
	}
	g.SetPhotos(phs)
	core.RenderImage(b, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		for _, frac := range []float32{0.001, 0.01, 0.1, 0.5, 0.9} {
			h := g.Geom.Size.Actual.Total.Y
			fr.ScrollDimToStart(math32.Y, int(fr.Geom.Pos.Content.Y+frac*h)+int(fr.Geom.Scroll.Y)*0)
			b.Scene.RenderWindow() // positions
			vis := fr.Geom.ContentBBox
			pt := image.Pt(vis.Min.X+g.cols/2*int(g.cell)+5, (vis.Min.Y+vis.Max.Y)/2)
			idx, ok := g.indexAt(pt)
			assert.True(t, ok)
			assert.True(t, pt.In(g.cellRect(idx)), "frac %v", frac)
			scroll := fr.Geom.Scroll.Y
			g.selectEvent(idx, events.SelectOne)
			assert.Equal(t, scroll, fr.Geom.Scroll.Y, "frac %v: the click scrolled", frac)
			t.Log(frac, idx, scroll, g.Geom.Pos.Content.Y, g.cell, g.cols, h)
		}
	})
}
