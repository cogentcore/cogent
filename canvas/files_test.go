// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvas

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSaveOpen tests saving a drawing and opening it again, with its
// metadata, and saving it with a new name.
func TestSaveOpen(t *testing.T) {
	var fn string
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		fn = string(cv.Filename)
		selectNodes(t, cv, "rect59")
		cv.SetFillColor("#00ff00", true)
		assert.True(t, cv.EditState.Changed)
		cv.SVG.Grid = 24
		cv.SVG.SVG.Scale = 1.5
		require.NoError(t, cv.SaveDrawing())
		assert.False(t, cv.EditState.Changed)
		assert.Contains(t, RecentPaths, fn)
	})
	b, err := os.ReadFile(fn)
	require.NoError(t, err)
	assert.Contains(t, strings.ToLower(string(b)), "#00ff00")
	runCanvasFile(t, fn, func(cv *Canvas) {
		assert.Equal(t, "#00ff00", strings.ToLower(prop(t, cv, "rect59", "fill")))
		assert.Equal(t, float32(24), cv.SVG.Grid, "the grid from the metadata")
		as := filepath.Join(filepath.Dir(fn), "as.svg")
		require.NoError(t, cv.SaveDrawingAs(core.Filename(as)))
		assert.FileExists(t, as)
		assert.Equal(t, as, string(cv.Filename))
		assert.Error(t, cv.SaveDrawingAs(""))
	})
}

// runCanvasFile is [runCanvas] for a drawing file that is not in testdata.
func runCanvasFile(t *testing.T, fn string, funs ...func(cv *Canvas)) {
	t.Helper()
	b, cv := newCanvas()
	lf := []func(){core.Locked(b, func() { openDrawing(t, cv, fn) })}
	for _, f := range funs {
		lf = append(lf, core.Locked(b, func() { f(cv) }))
	}
	lf = append(lf, core.Locked(b, func() { closeSaved(cv) }))
	core.RenderImage(b, lf...)
}

// TestExport tests exporting a drawing as PNG and PDF.
func TestExport(t *testing.T) {
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		base := strings.TrimSuffix(string(cv.Filename), ".svg")
		require.NoError(t, cv.ExportPNG(320, 0))
		f, err := os.Open(base + ".png")
		require.NoError(t, err)
		img, err := png.Decode(f)
		f.Close()
		require.NoError(t, err)
		assert.Equal(t, 320, img.Bounds().Dx())
		require.NoError(t, cv.ExportPDF(0, false))
		assert.FileExists(t, base+".pdf")
		os.Remove(base + ".pdf")
		if _, err := exec.LookPath("inkscape"); err != nil {
			assert.Error(t, cv.ExportPDF(300, true), "no inkscape")
		} else {
			assert.NoError(t, cv.ExportPDF(300, true))
		}
		assert.NoFileExists(t, filepath.Join(filepath.Dir(base), "export_pdf.svg"), "removed")
	})
}

// TestAutosave tests autosaving a drawing, opening the autosave file,
// and deleting it when the drawing is saved.
func TestAutosave(t *testing.T) {
	defer core.TrackWindows()()
	var asfn string
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		asfn = cv.autoSaveFilename(cv.Filename)
		assert.Equal(t, "#shapes.svg#", filepath.Base(asfn))
		assert.False(t, cv.AutosaveCheck(cv.Filename))
	}, unlocked(func(b *core.Body, cv *Canvas) {
		require.NoError(t, cv.Autosave())
	}), func(cv *Canvas) {
		assert.FileExists(t, asfn)
		assert.True(t, cv.AutosaveCheck(cv.Filename))
		wd, _ := os.Getwd()
		require.NoError(t, cv.OpenDrawingCheck(cv.Filename)) // a dialog
		os.Chdir(wd)
	}, unlocked(func(b *core.Body, cv *Canvas) {
		var bt *core.Button
		core.Locked(b, func() {
			bt = core.ButtonByText(core.TopScene(b), "Open autosave")
		})()
		require.NotNil(t, bt)
		wd, _ := os.Getwd()
		core.Locked(bt, func() { bt.Send(events.Click) })()
		os.Chdir(wd)
	}), func(cv *Canvas) {
		assert.Equal(t, asfn, string(cv.Filename), "the autosave file is open")
		require.NoError(t, cv.SaveDrawingAs(core.Filename(filepath.Join(filepath.Dir(asfn), "shapes.svg"))))
		assert.NoFileExists(t, asfn, "deleted on save")
	})
	n := &Canvas{}
	n.Name = "x"
	assert.Equal(t, "#new_file_x.svg#", n.autoSaveFilename(""))
}

// TestDrawingSize tests the physical size of the drawing, resizing it to
// its contents, and adding an image.
func TestDrawingSize(t *testing.T) {
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		sz := &PhysicalSize{}
		sz.SetFromSVG(cv.SVG)
		assert.Equal(t, Img1280x720, sz.StandardSize)
		assert.Equal(t, units.UnitPx, sz.Units)
		require.NoError(t, sz.SetStandardSize(A4))
		cv.SetPhysicalSize(sz)
		assert.Equal(t, math32.Vec2(210, 297), cv.SVG.Root().ViewBox.Size)
		assert.Equal(t, units.UnitMm, cv.SSVG().PhysicalWidth.Unit)
		cv.SetPhysicalSize(&PhysicalSize{}) // a zero size is the standard one
		cv.SetPhysicalSize(nil)

		cv.ResizeToContents()
		assert.Less(t, cv.SVG.Root().ViewBox.Size.X, float32(1280), "fits the contents")

		fn := filepath.Join(t.TempDir(), "img.png")
		f, err := os.Create(fn)
		require.NoError(t, err)
		img := image.NewRGBA(image.Rect(0, 0, 8, 8))
		img.Set(1, 1, color.RGBA{255, 0, 0, 255})
		require.NoError(t, png.Encode(f, img))
		f.Close()
		n := count[*svg.Image](cv)
		require.NoError(t, cv.AddImage(core.Filename(fn), 0, 0))
		assert.Equal(t, n+1, count[*svg.Image](cv))
		assert.Error(t, cv.AddImage("nonexistent.png", 0, 0))
	})
}

// TestPhysicalSize tests the standard physical sizes.
func TestPhysicalSize(t *testing.T) {
	var ps PhysicalSize
	ps.Defaults()
	assert.Equal(t, math32.Vec2(1280, 720), ps.Size)
	ps.StandardSize = USLetter
	ps.Update()
	assert.Equal(t, units.UnitPt, ps.Units)
	assert.Equal(t, USLetter, MatchStandardSize(792, 612, units.UnitPt), "landscape")
	assert.Equal(t, CustomSize, MatchStandardSize(100, 100, units.UnitPx))
	ps.StandardSize = CustomSize
	ps.Size.Set(5, 5)
	ps.Update()
	assert.Equal(t, math32.Vec2(5, 5), ps.Size, "custom is kept")
	assert.Error(t, ps.SetStandardSize(StandardSizes(1000)))
}

// TestSettings tests the default settings and saving and opening them,
// with the recent paths and splits.
func TestSettings(t *testing.T) {
	se := &SettingsData{}
	se.Defaults()
	assert.True(t, se.ShowGrid)
	assert.Equal(t, 3, se.SnapZone)
	se.Update()
	se.File = filepath.Join(t.TempDir(), "settings.toml")
	rp := RecentPaths
	defer func() { RecentPaths = rp }()
	RecentPaths = core.FilePaths{"a.svg"}
	Splits[0] = 0.2
	require.NoError(t, se.Save())
	RecentPaths = nil
	Splits[0] = 0.1
	se.SnapZone = 1
	require.NoError(t, se.Open())
	assert.Equal(t, 3, se.SnapZone)
	assert.Equal(t, core.FilePaths{"a.svg"}, RecentPaths)
	assert.Equal(t, float32(0.2), Splits[0])
	Splits[0] = 0.15
	SaveSplits()
}

// TestKeys tests the editing keys in the drawing.
func TestKeys(t *testing.T) {
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		root := cv.SVG.Root()
		n0 := len(root.Children)
		selectNodes(t, cv, "rect59")
		core.SendKeyFunction(cv.SVG, keymap.Duplicate)
		assert.Len(t, root.Children, n0+1, "duplicate")
		core.SendKeyFunction(cv.SVG, keymap.Undo)
		assert.Len(t, root.Children, n0, "undo")
		core.SendKeyFunction(cv.SVG, keymap.Redo)
		assert.Len(t, root.Children, n0+1, "redo")
		selectNodes(t, cv, "rect59")
		core.SendKeyFunction(cv.SVG, keymap.Copy)
		core.SendKeyFunction(cv.SVG, keymap.Paste)
		assert.Len(t, root.Children, n0+2, "paste")
		selectNodes(t, cv, "rect59")
		core.SendKeyFunction(cv.SVG, keymap.Cut)
		assert.Len(t, root.Children, n0+1, "cut")
		selectNodes(t, cv, "rect318")
		core.SendKeyFunction(cv.SVG, keymap.Delete)
		assert.Len(t, root.Children, n0, "delete")
	})
}

// TestZoom tests zooming the drawing with the scroll wheel and the
// toolbar buttons.
func TestZoom(t *testing.T) {
	var pt image.Point
	var sc float32
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		pt = center(node(t, cv, "rect59"))
		sc = cv.SSVG().Scale
	}, unlocked(func(b *core.Body, cv *Canvas) {
		sysEvents(b).MouseMove(pt)
		sysEvents(b).Scroll(pt, math32.Vec2(0, -10))
	}), func(cv *Canvas) {
		assert.NotEqual(t, sc, cv.SSVG().Scale, "zoomed")
		core.ButtonByText(cv.Scene, "Zoom page").Send(events.Click)
		assert.Equal(t, float32(1), cv.SSVG().Scale)
		core.ButtonByText(cv.Scene, "Zoom all").Send(events.Click)
		assert.Greater(t, cv.SSVG().Scale, float32(1), "the contents are smaller than the page")
	})
}

// TestDropItems tests dropping items copied from a drawing on it, which
// adds them at the drop position.
func TestDropItems(t *testing.T) {
	runCanvas(t, "shapes.svg", func(cv *Canvas) {
		root := cv.SVG.Root()
		n0 := len(root.Children)
		var md mimedata.Mimes
		cv.tree.FindSyncNode(node(t, cv, "rect59")).MimeData(&md)
		require.True(t, md.HasType(fileinfo.DataJSON))
		pt := image.Pt(900, 900)
		de := events.NewExternalDrop(events.Drop, events.Left, pt, 0, md)
		de.WhereLocal = pt // as the event manager does for a widget
		cv.SVG.HandleEvent(de)
		assert.Len(t, root.Children, n0+1, "dropped")
		sel := cv.EditState.FirstSelected()
		require.NotNil(t, sel)
		assert.InDelta(t, 900, sel.AsNodeBase().BBox.Min.X, 2, "at the drop position")

		cv.SVG.dragItems = []svg.Node{node(t, cv, "rect318")}
		cv.SVG.HandleEvent(events.NewExternalDrop(events.DropDeleteSource, events.Left, pt, 0, nil))
		assert.Len(t, root.Children, n0, "the source items of a move are deleted")
		assert.NotNil(t, cv.SVG.selectionImage())
	})
}

// TestContextMenus tests making the context menus of the drawing and
// the tree.
func TestContextMenus(t *testing.T) {
	runCanvas(t, "seplays.svg", func(cv *Canvas) {
		m := core.NewScene()
		cv.SVG.contextMenu(m, image.Point{})
		assert.NotNil(t, core.ButtonByText(m, "Paste"), "nothing selected")
		selectNodes(t, cv, "rect847", "ellipse318")
		m = core.NewScene()
		cv.SVG.contextMenu(m, image.Point{})
		assert.NotNil(t, core.ButtonByText(m, "Group"))
		cv.SelectGroup()
		m = core.NewScene()
		cv.SVG.contextMenu(m, image.Point{})
		assert.NotNil(t, core.ButtonByText(m, "Ungroup"))
		core.ButtonByText(m, "Select in tree").Send(events.Click)

		m = core.NewScene()
		treeNode(t, cv, "Layer1").contextMenu(m, image.Point{})
		assert.NotNil(t, core.ButtonByText(m, "Set as current"), "a layer")
		m = core.NewScene()
		tn := tree.ChildByType[*Tree](treeNode(t, cv, "Layer1"))
		if tn != nil {
			tn.contextMenu(m, image.Point{})
			assert.Nil(t, core.ButtonByText(m, "Set as current"))
		}
	})
}
