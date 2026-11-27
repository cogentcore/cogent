// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"image"
	"image/draw"
	"log/slog"
	"runtime"
	"slices"
	"strings"
	"sync"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/units"
	xdraw "golang.org/x/image/draw"
)

// photosMime is the mime type of the names of photos being dragged,
// one per line.
const photosMime = "application/x-cogent-photos"

// Grid is a grid of the thumbnails of photos, which draws only those that
// are visible, so that it can show any number of them. It is in a
// scrolling frame. Photos are selected as in a list, with the mouse and
// keys, and dragged onto tags to tag them, or out of the app as files.
type Grid struct {
	core.WidgetBase

	// Photos are the photos shown.
	Photos []*Photo `set:"-"`

	// Current is the index of the current photo, the last one selected,
	// or -1 for none.
	Current int `set:"-"`

	// Selected are the indexes of the selected photos.
	Selected map[int]bool `set:"-"`

	// CellSize is the size of the cells of the thumbnails.
	CellSize units.Value

	// lib is the library of the photos.
	lib *Library

	// cols is the number of columns, and cell the size of each cell in
	// dots, from the last layout.
	cols int
	cell float32

	// thumbs are the loaded thumbnails, by the names of the photos, under
	// thumbsMu, and loading are those being loaded.
	thumbs   map[string]image.Image
	loading  map[string]bool
	thumbsMu sync.Mutex

	// scaled are the thumbnails scaled to the size of the cells, scaledSize.
	scaled     map[string]image.Image
	scaledSize int

	// queue are the photos whose thumbnails are to be loaded, the last one
	// first, as the most recently shown, and wake wakes the loaders.
	queue []*Photo
	wake  chan struct{}

	// renderPending is whether a render is pending for loaded thumbnails.
	renderPending bool
}

func (g *Grid) Init() {
	g.WidgetBase.Init()
	g.Current = -1
	g.Selected = map[int]bool{}
	g.CellSize = units.Dp(136)
	g.thumbs = map[string]image.Image{}
	g.loading = map[string]bool{}
	g.scaled = map[string]image.Image{}
	g.wake = make(chan struct{}, 1)
	for range max(2, runtime.NumCPU()-2) {
		go g.loader()
	}
	g.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Focusable, abilities.Clickable, abilities.DoubleClickable, abilities.Draggable, abilities.Selectable)
		s.Grow.Set(1, 0)
		s.Min.X.Set(g.CellSize.Value, g.CellSize.Unit)
	})
	g.OnClick(func(e events.Event) {
		g.SetFocus()
		idx, ok := g.indexAt(e.Pos())
		if !ok {
			return
		}
		g.selectEvent(idx, e.SelectMode())
	})
	g.On(events.DragStart, g.dragStart)
	g.On(events.KeyChord, g.keyInput)
}

// SetPhotos sets the photos shown, with none selected.
func (g *Grid) SetPhotos(phs []*Photo) {
	g.Photos = phs
	g.Current = -1
	g.Selected = map[int]bool{}
	g.NeedsLayout()
}

// SelectedPhotos returns the selected photos, in order.
func (g *Grid) SelectedPhotos() []*Photo {
	var idxs []int
	for i := range g.Selected {
		if i < len(g.Photos) {
			idxs = append(idxs, i)
		}
	}
	slices.Sort(idxs)
	res := make([]*Photo, len(idxs))
	for i, idx := range idxs {
		res[i] = g.Photos[idx]
	}
	return res
}

// CurrentPhoto returns the current photo, or nil.
func (g *Grid) CurrentPhoto() *Photo {
	if g.Current < 0 || g.Current >= len(g.Photos) {
		return nil
	}
	return g.Photos[g.Current]
}

// SelectIndex makes the photo at the given index the only one selected,
// and the current one, scrolling to it, and sends a Select event.
func (g *Grid) SelectIndex(idx int) {
	g.selectEvent(idx, events.SelectOne)
}

// selectEvent selects the photo at the given index for the given mode,
// making it the current one, and sends a Select event.
func (g *Grid) selectEvent(idx int, mode events.SelectModes) {
	if len(g.Photos) == 0 {
		return
	}
	idx = max(0, min(idx, len(g.Photos)-1))
	switch mode {
	case events.ExtendContinuous:
		from := g.Current
		if from < 0 {
			from = idx
		}
		for i := min(from, idx); i <= max(from, idx); i++ {
			g.Selected[i] = true
		}
	case events.ExtendOne:
		if g.Selected[idx] {
			delete(g.Selected, idx)
		} else {
			g.Selected[idx] = true
		}
	default:
		g.Selected = map[int]bool{idx: true}
	}
	g.Current = idx
	g.scrollToIndex(idx)
	g.NeedsRender()
	g.Send(events.Select)
}

// SelectAll selects all of the photos.
func (g *Grid) SelectAll() {
	for i := range g.Photos {
		g.Selected[i] = true
	}
	g.NeedsRender()
	g.Send(events.Select)
}

// cellRect returns the box of the cell of the photo at the given index,
// in scene coordinates.
func (g *Grid) cellRect(idx int) image.Rectangle {
	if g.cols == 0 {
		return image.Rectangle{}
	}
	pos := g.Geom.Pos.Content
	r, c := idx/g.cols, idx%g.cols
	x, y := pos.X+float32(c)*g.cell, pos.Y+float32(r)*g.cell
	return image.Rect(int(x), int(y), int(x+g.cell), int(y+g.cell))
}

// scrollToIndex scrolls the frame of the grid so that the photo at the
// given index is visible.
func (g *Grid) scrollToIndex(idx int) {
	if fr, ok := g.Parent.(*core.Frame); ok && g.cols > 0 {
		fr.ScrollToBox(g.cellRect(idx))
	}
}

// indexAt returns the index of the photo at the given position in scene
// coordinates.
func (g *Grid) indexAt(pt image.Point) (int, bool) {
	if g.cols == 0 || g.cell == 0 {
		return -1, false
	}
	rel := math32.FromPoint(pt).Sub(g.Geom.Pos.Content)
	if rel.X < 0 || rel.Y < 0 {
		return -1, false
	}
	c, r := int(rel.X/g.cell), int(rel.Y/g.cell)
	if c >= g.cols {
		return -1, false
	}
	idx := r*g.cols + c
	return idx, idx < len(g.Photos)
}

// layoutCells sets the number of columns and size of the cells for the
// given width, returning the height of the grid.
func (g *Grid) layoutCells(width float32) float32 {
	cell := g.CellSize.Dots
	if cell <= 0 {
		cell = 128
	}
	g.cols = max(1, int(width/cell))
	g.cell = math32.Floor(width / float32(g.cols))
	rows := (len(g.Photos) + g.cols - 1) / g.cols
	return float32(rows) * g.cell
}

func (g *Grid) SizeUp() {
	g.WidgetBase.SizeUp()
	g.CellSize.ToDots(&g.Styles.UnitContext)
	sz := &g.Geom.Size
	h := g.layoutCells(max(sz.Actual.Content.X, g.CellSize.Dots))
	sz.Actual.Content.Y = h
	sz.SetTotalFromContent(&sz.Actual)
}

// SizeDown sets the height of the grid from its width, which is only
// known now.
func (g *Grid) SizeDown(iter int) bool {
	redo := g.WidgetBase.SizeDown(iter)
	sz := &g.Geom.Size
	h := g.layoutCells(max(sz.Actual.Content.X, sz.Alloc.Content.X))
	if sz.Actual.Content.Y == h {
		return redo
	}
	sz.Actual.Content.Y = h
	sz.SetTotalFromContent(&sz.Actual)
	return true
}

// SizeFinal sets the columns for the final width.
func (g *Grid) SizeFinal() {
	g.WidgetBase.SizeFinal()
	g.layoutCells(g.Geom.Size.Actual.Content.X)
}

func (g *Grid) Render() {
	g.WidgetBase.Render()
	vis := g.Geom.ContentBBox
	if vis.Empty() || g.cols == 0 || len(g.Photos) == 0 {
		return
	}
	pc := &g.Scene.Painter
	pos := g.Geom.Pos.Content
	cell := g.cell
	gap := math32.Max(3, math32.Round(cell/32))
	inner := int(cell - 2*gap)
	if inner != g.scaledSize {
		g.scaled = map[string]image.Image{}
		g.scaledSize = inner
	}
	r0 := max(0, int((float32(vis.Min.Y)-pos.Y)/cell))
	r1 := int((float32(vis.Max.Y)-pos.Y)/cell) + 1
	var need []*Photo
	for r := r0; r <= r1; r++ {
		for c := range g.cols {
			idx := r*g.cols + c
			if idx >= len(g.Photos) {
				break
			}
			ph := g.Photos[idx]
			x, y := pos.X+float32(c)*cell, pos.Y+float32(r)*cell
			img := g.scaledThumb(ph, inner)
			if img == nil {
				need = append(need, ph)
				pc.Fill.Color = colors.Scheme.SurfaceContainerHigh
				pc.Stroke.Color = nil
				pc.Rectangle(x+gap, y+gap, cell-2*gap, cell-2*gap)
				pc.Draw()
				continue
			}
			isz := img.Bounds().Size()
			ix := int(x+gap) + (inner-isz.X)/2
			iy := int(y+gap) + (inner-isz.Y)/2
			dst := image.Rect(ix, iy, ix+isz.X, iy+isz.Y)
			if g.Selected[idx] { // a frame around it
				pc.Fill.Color = colors.Scheme.Select.Container
				if idx == g.Current {
					pc.Fill.Color = colors.Scheme.Primary.Base
				}
				pc.Stroke.Color = nil
				pc.RoundedRectangle(float32(ix)-gap, float32(iy)-gap, float32(isz.X)+2*gap, float32(isz.Y)+2*gap, gap)
				pc.Draw()
			}
			clip := dst.Intersect(vis)
			if clip.Empty() {
				continue
			}
			pc.DrawImage(img, img, clip, clip.Min.Sub(dst.Min), draw.Over)
		}
	}
	if len(need) > 0 {
		g.load(need)
	}
}

// scaledThumb returns the thumbnail of the given photo scaled to fit in
// the given size, or nil if it is not loaded yet.
func (g *Grid) scaledThumb(ph *Photo, size int) image.Image {
	if img, ok := g.scaled[ph.Name]; ok {
		return img
	}
	g.thumbsMu.Lock()
	thumb := g.thumbs[ph.Name]
	g.thumbsMu.Unlock()
	if thumb == nil {
		return nil
	}
	b := thumb.Bounds()
	w, h := b.Dx(), b.Dy()
	var nw, nh int
	if w >= h {
		nw, nh = size, max(1, h*size/w)
	} else {
		nw, nh = max(1, w*size/h), size
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), thumb, b, draw.Src, nil)
	if len(g.scaled) > 2000 {
		g.scaled = map[string]image.Image{}
	}
	g.scaled[ph.Name] = dst
	return dst
}

// Forget forgets the thumbnail of the given photo, as after it is rotated,
// so that it is loaded again.
func (g *Grid) Forget(ph *Photo) {
	g.thumbsMu.Lock()
	delete(g.thumbs, ph.Name)
	g.thumbsMu.Unlock()
	delete(g.scaled, ph.Name)
	g.NeedsRender()
}

// load queues the thumbnails of the given photos to be loaded.
func (g *Grid) load(phs []*Photo) {
	g.thumbsMu.Lock()
	for _, ph := range slices.Backward(phs) { // the first ones are loaded first
		if !g.loading[ph.Name] {
			g.loading[ph.Name] = true
			g.queue = append(g.queue, ph)
		}
	}
	g.thumbsMu.Unlock()
	select {
	case g.wake <- struct{}{}:
	default:
	}
}

// loader loads the thumbnails in the queue, the last ones first.
func (g *Grid) loader() {
	for {
		g.thumbsMu.Lock()
		n := len(g.queue)
		if n == 0 {
			g.thumbsMu.Unlock()
			<-g.wake
			continue
		}
		ph := g.queue[n-1]
		g.queue = g.queue[:n-1]
		if n > 1 { // more for the others
			select {
			case g.wake <- struct{}{}:
			default:
			}
		}
		g.thumbsMu.Unlock()
		if g.lib == nil || g.This == nil {
			return
		}
		img, err := g.lib.Thumb(ph)
		if err != nil {
			slog.Error("photos: thumbnail", "photo", ph.Name, "err", err)
			img = image.NewRGBA(image.Rect(0, 0, 1, 1)) // not again
		}
		g.thumbsMu.Lock()
		if len(g.thumbs) > 5000 { // only those in view are needed
			g.thumbs = map[string]image.Image{}
		}
		g.thumbs[ph.Name] = img
		delete(g.loading, ph.Name)
		pending := g.renderPending
		g.renderPending = true
		g.thumbsMu.Unlock()
		if !pending {
			g.AsyncLock()
			g.thumbsMu.Lock()
			g.renderPending = false
			g.thumbsMu.Unlock()
			g.NeedsRender()
			g.AsyncUnlock()
		}
	}
}

// keyInput handles the keys for moving through the photos, and selecting
// them.
func (g *Grid) keyInput(e events.Event) {
	if len(g.Photos) == 0 {
		return
	}
	mode := events.SelectModeBits(e.Modifiers())
	if mode == events.ExtendOne { // as with Command: moving
		mode = events.SelectOne
	}
	cur := max(g.Current, 0)
	switch keymap.Of(e.KeyChord()) {
	case keymap.MoveRight:
		e.SetHandled()
		g.selectEvent(cur+1, mode)
	case keymap.MoveLeft:
		e.SetHandled()
		g.selectEvent(cur-1, mode)
	case keymap.MoveDown:
		e.SetHandled()
		g.selectEvent(cur+g.cols, mode)
	case keymap.MoveUp:
		e.SetHandled()
		g.selectEvent(cur-g.cols, mode)
	case keymap.PageDown:
		e.SetHandled()
		g.selectEvent(cur+g.cols*g.visibleRows(), mode)
	case keymap.PageUp:
		e.SetHandled()
		g.selectEvent(cur-g.cols*g.visibleRows(), mode)
	case keymap.Home, keymap.DocHome:
		e.SetHandled()
		g.selectEvent(0, mode)
	case keymap.End, keymap.DocEnd:
		e.SetHandled()
		g.selectEvent(len(g.Photos)-1, mode)
	case keymap.SelectAll:
		e.SetHandled()
		g.SelectAll()
	case keymap.Enter, keymap.Accept:
		e.SetHandled()
		g.Send(events.DoubleClick)
	}
}

// visibleRows returns the number of rows visible in the frame of the grid.
func (g *Grid) visibleRows() int {
	if fr, ok := g.Parent.(*core.Frame); ok && g.cell > 0 {
		return max(1, int(float32(fr.Geom.ContentBBox.Dy())/g.cell))
	}
	return 1
}

// dragStart starts dragging the selected photos, or the one under the
// mouse if it is not selected.
func (g *Grid) dragStart(e events.Event) {
	idx, ok := g.indexAt(e.StartPos())
	if !ok {
		return
	}
	if !g.Selected[idx] {
		g.selectEvent(idx, events.SelectOne)
	}
	phs := g.SelectedPhotos()
	names := make([]string, len(phs))
	for i, ph := range phs {
		names[i] = ph.Name
	}
	md := mimedata.NewMime(photosMime, []byte(strings.Join(names, "\n")))
	g.Scene.Events.DragStart(g.This.(core.Widget), md, e)
	if img := g.scaledThumb(g.Photos[idx], g.scaledSize); img != nil {
		g.Scene.Events.SetDragImage(img)
	}
	e.SetHandled()
}

// DragFiles returns the files of the photos being dragged, for dragging
// them out of the app (see [core.FileDragger]).
func (g *Grid) DragFiles() []string {
	var files []string
	for _, ph := range g.SelectedPhotos() {
		files = append(files, g.lib.PhotoFile(ph))
	}
	return files
}

// draggedPhotos returns the photos in the given drag data, if any.
func (lib *Library) draggedPhotos(md mimedata.Mimes) []*Photo {
	var res []*Photo
	for _, d := range md {
		if d.Type != photosMime {
			continue
		}
		for _, name := range strings.Split(string(d.Data), "\n") {
			if ph := lib.Photo(name); ph != nil {
				res = append(res, ph)
			}
		}
	}
	return res
}
