// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNames(t *testing.T) {
	d := time.Date(2017, 2, 21, 13, 4, 5, 0, time.UTC)
	n := nameFor("img", d, 2)
	assert.Equal(t, "img_2017_02_21_13_04_05_n2", n)
	pd, pn, ok := parseName(n)
	assert.True(t, ok)
	assert.Equal(t, d, pd)
	assert.Equal(t, 2, pn)
	_, _, ok = parseName("IMG_5563")
	assert.False(t, ok)

	assert.Equal(t, "Harvard 85 86", splitWords("Harvard85-86"))
	assert.Equal(t, "Alpine VRBOposs", splitWords("AlpineVRBOposs"))
}

// realPhotos are photos of the real library used in the tests, if it is
// there, by their names in the tests (as before they were renamed), with
// the names they may have now.
var realPhotos = []string{"IMG_0073_1.HEIC", "1000061149.jpg", "MountainPicture.png", "img_2015_03_27_11_19_51_n0.jpg"}

var realRenamed = map[string]string{"IMG_0073_1.HEIC": "img_2023_05_08_00_21_26_n0.heic",
	"1000061149.jpg": "img_2026_01_26_19_09_44_n0.jpg", "MountainPicture.png": "img_2024_01_21_14_53_36_n0.png"}

// realPhoto returns the file of the given real photo, which may be renamed.
func realPhoto(src, name string) string {
	fn := filepath.Join(src, name)
	if _, err := os.Stat(fn); err != nil {
		if r, ok := realRenamed[name]; ok {
			return filepath.Join(src, r)
		}
	}
	return fn
}

// testLibrary returns a new library in a temporary directory with copies
// of [realPhotos], skipping the test if they are not there.
func testLibrary(t *testing.T) (*Library, string) {
	home, _ := os.UserHomeDir()
	src := filepath.Join(home, "Pictures", "CogentPhotos", "All")
	for _, fn := range realPhotos {
		if _, err := os.Stat(realPhoto(src, fn)); err != nil {
			t.Skip("no real photos to test with")
		}
	}
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "All"), 0o755))
	for _, fn := range realPhotos {
		require.NoError(t, copyFile(filepath.Join(dir, "All", fn), realPhoto(src, fn)))
	}
	lib, err := OpenLibrary(dir)
	require.NoError(t, err)
	lib.ThumbDir = filepath.Join(dir, "thumbs") // not the real ones
	t.Cleanup(func() { lib.Close() })
	return lib, dir
}

func TestScan(t *testing.T) {
	lib, dir := testLibrary(t)
	// a GoPix folder of links
	pix := filepath.Join(t.TempDir(), "Pix")
	require.NoError(t, os.MkdirAll(filepath.Join(pix, "All"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(pix, "2017_Kyoto"), 0o755))
	require.NoError(t, os.Symlink("../All/IMG_0073_1.HEIC", filepath.Join(pix, "2017_Kyoto", "IMG_0073_1.HEIC")))
	require.NoError(t, os.Symlink("../All/img_2015_03_27_11_19_51_n0.jpg", filepath.Join(pix, "2017_Kyoto", "img_2015_03_27_11_19_51_n0.jpg")))

	start := time.Now()
	require.NoError(t, lib.ScanGoPix(pix, nil))
	t.Log("scan", time.Since(start))
	phs := lib.Photos(nil)
	require.Len(t, phs, len(realPhotos))
	for _, ph := range phs {
		_, _, ok := parseName(ph.Name)
		assert.True(t, ok, ph.Name)
		assert.FileExists(t, filepath.Join(dir, "All", ph.Filename()))
		t.Log(ph.Name, ph.Ext, ph.OrigName, ph.DateTaken, ph.Size, ph.Orient, ph.Make, ph.Model, ph.GPS, ph.Tags)
	}
	heic := lib.Photos(func(ph *Photo) bool { return ph.OrigName == "IMG_0073_1" })
	require.Len(t, heic, 1)
	assert.Equal(t, ".heic", heic[0].Ext)
	assert.Equal(t, []string{"2017/Kyoto"}, heic[0].Tags)
	assert.Equal(t, time.Date(2023, 5, 8, 0, 21, 26, 0, time.UTC), heic[0].DateTaken)
	assert.Equal(t, 2, lib.Tags()["2017/Kyoto"])

	res, err := lib.Search("kyo")
	require.NoError(t, err)
	assert.Len(t, res, 2)
	res, err = lib.Search("2023 may")
	require.NoError(t, err)
	assert.Len(t, res, 1)

	start = time.Now()
	img, err := lib.Thumb(heic[0])
	require.NoError(t, err)
	t.Log("heic thumb", time.Since(start), img.Bounds())
	assert.Equal(t, ThumbSize, max(img.Bounds().Dx(), img.Bounds().Dy()))

	// reopened: the same
	require.NoError(t, lib.Close())
	lib2, err := OpenLibrary(dir)
	require.NoError(t, err)
	lib2.ThumbDir = lib.ThumbDir
	defer lib2.Close()
	require.NoError(t, lib2.Scan(nil, nil))
	assert.Equal(t, len(realPhotos), lib2.Len())
	ph := lib2.Photo(heic[0].Name)
	require.NotNil(t, ph)
	assert.Equal(t, []string{"2017/Kyoto"}, ph.Tags)

	// trash and restore
	require.NoError(t, lib2.Trash(ph))
	assert.FileExists(t, filepath.Join(dir, "Trash", ph.Filename()))
	require.NoError(t, lib2.Scan(nil, nil)) // still there, in the trash
	assert.Equal(t, len(realPhotos), lib2.Len())
	require.NoError(t, lib2.Restore(ph))
	assert.FileExists(t, filepath.Join(dir, "All", ph.Filename()))

	// set date renames
	require.NoError(t, lib2.SetDate(ph, time.Date(1999, 1, 2, 3, 4, 5, 0, time.UTC)))
	assert.Equal(t, "img_1999_01_02_03_04_05_n0", ph.Name)
	assert.FileExists(t, filepath.Join(dir, "All", ph.Filename()))
	res, err = lib2.Search("1999")
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{ph.Name: true}, res)
}

func TestPortraitThumb(t *testing.T) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, "Pictures", "CogentPhotos", "All")
	fn := filepath.Join(dir, "IMG_8090.HEIC")
	if _, err := os.Stat(fn); err != nil { // renamed
		fn = filepath.Join(dir, "img_2022_01_12_13_47_05_n0.heic")
		if (&Photo{}).readExif(fn) != nil || fileExif(fn).Orient != Rotated90L {
			fn = filepath.Join(dir, "img_2022_01_12_13_47_05_n1.heic")
		}
	}
	if _, err := os.Stat(fn); err != nil {
		t.Skip("no real photo")
	}
	ph := &Photo{}
	require.NoError(t, ph.readExif(fn))
	t.Log(ph.Orient, ph.Size, ph.DateTaken)
	assert.Equal(t, Rotated90L, ph.Orient)
	tf := filepath.Join(t.TempDir(), "thumb.jpg")
	start := time.Now()
	img, err := makeThumb(ph, fn, tf)
	require.NoError(t, err)
	t.Log("thumb", time.Since(start))
	assert.Equal(t, 192, img.Bounds().Dx())
	assert.Equal(t, 256, img.Bounds().Dy())
	copyFile(filepath.Join(os.TempDir(), "cogentphotos-portrait.jpg"), tf)
}

// TestImportAgain tests that adding files adds them renamed by their dates,
// with the tags, and adding them again does not add them twice, but tags
// them, and that files are copied, not moved.
func TestImportAgain(t *testing.T) {
	lib := smallLibrary(t, 3)
	src := t.TempDir()
	var files []string
	for i, ph := range lib.Photos(nil) {
		fn := filepath.Join(src, fmt.Sprintf("new%d.jpg", i))
		require.NoError(t, copyFile(fn, lib.PhotoFile(ph)))
		files = append(files, fn)
	}
	// two the same as photos in the library, and a new one
	newer := filepath.Join(src, "newer.jpg")
	require.NoError(t, copyFile(newer, files[0]))
	d := time.Date(2021, 5, 6, 7, 8, 9, 0, time.Local)
	os.Chtimes(newer, d, d)
	phs, err := lib.Import([]string{files[0], files[1], newer}, false, []string{"2021_Dropped"}, nil)
	require.NoError(t, err)
	require.Len(t, phs, 1, "only the new one")
	assert.Equal(t, "img_2021_05_06_07_08_09_n0", phs[0].Name)
	assert.FileExists(t, newer, "copied, not moved")
	assert.Equal(t, 4, lib.Len())
	assert.Equal(t, 3, lib.Tags()["2021/Dropped"], "the ones already there are tagged too")
	phs, err = lib.Import([]string{newer}, false, nil, nil)
	require.NoError(t, err)
	assert.Len(t, phs, 0)
	assert.Equal(t, 4, lib.Len())
}

// TestRemoveGoPixThumbs tests that the thumbnails copied from GoPix are
// removed once, for making them again, and not the others.
func TestRemoveGoPixThumbs(t *testing.T) {
	lib := smallLibrary(t, 2)
	require.NoError(t, lib.store.setMeta(goPixThumbsKey, "")) // as before
	home := t.TempDir()
	t.Setenv("HOME", home)
	cache, err := os.UserCacheDir()
	require.NoError(t, err)
	gopix := filepath.Join(cache, "GoPix", "thumbs")
	require.NoError(t, os.MkdirAll(gopix, 0o755))
	phs := lib.Photos(nil)
	for i, ph := range phs {
		ph.OrigName = fmt.Sprintf("IMG_%d.JPG", i)
		require.NoError(t, os.WriteFile(filepath.Join(gopix, ph.OrigName+".jpg"), []byte("gopix"), 0o644))
	}
	copied, made := lib.thumbFile(phs[0]), lib.thumbFile(phs[1])
	require.NoError(t, os.MkdirAll(filepath.Dir(copied), 0o755))
	require.NoError(t, os.WriteFile(copied, []byte("gopix"), 0o644))
	require.NoError(t, os.WriteFile(made, []byte("made here"), 0o644))

	require.NoError(t, lib.removeGoPixThumbs())
	assert.NoFileExists(t, copied, "the copy is removed")
	assert.FileExists(t, made, "the one made here is kept")

	require.NoError(t, os.WriteFile(copied, []byte("gopix"), 0o644))
	require.NoError(t, lib.removeGoPixThumbs())
	assert.FileExists(t, copied, "only once")
}
