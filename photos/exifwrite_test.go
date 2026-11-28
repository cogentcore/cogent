// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComposeOrient(t *testing.T) {
	assert.Equal(t, Rotated90L, composeOrient(NoOrient, 90)) // EXIF 6: shown turned clockwise
	assert.Equal(t, Rotated180, composeOrient(Rotated90L, 90))
	assert.Equal(t, Rotated0, composeOrient(Rotated90L, 270))
	assert.Equal(t, Rotated90R, composeOrient(Rotated0, 270))
	assert.Equal(t, FlippedH, composeOrient(FlippedH, 0))
}

func TestWriteExif(t *testing.T) {
	lib, dir := testLibrary(t)
	require.NoError(t, lib.Scan(nil, nil))
	byOrig := map[string]*Photo{}
	for _, ph := range lib.Photos(nil) {
		byOrig[ph.OrigName] = ph
	}

	// in place: an iPhone photo
	ph := byOrig["img_2015_03_27_11_19_51_n0"]
	require.NotNil(t, ph)
	before, err := os.ReadFile(lib.PhotoFile(ph))
	require.NoError(t, err)
	date := time.Date(1999, 1, 2, 3, 4, 5, 0, time.UTC)
	require.NoError(t, lib.SetDate(ph, date))
	file := lib.PhotoFile(ph)
	assert.Equal(t, "img_1999_01_02_03_04_05_n0.jpg", filepath.Base(file))
	after, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, len(before), len(after), "changed in place")
	diffs := 0
	for i := range before {
		if before[i] != after[i] {
			diffs++
		}
	}
	assert.Less(t, diffs, 60)
	assert.Equal(t, date, fileExif(file).DateTaken)

	// rotation into the orientation
	require.NoError(t, lib.Rotate(ph, 90))
	assert.Equal(t, 0, ph.Rotation)
	assert.Equal(t, Rotated90L, ph.Orient)
	assert.Equal(t, Rotated90L, fileExif(file).Orient)
	img, err := lib.Thumb(ph)
	require.NoError(t, err)
	assert.Greater(t, img.Bounds().Dy(), img.Bounds().Dx(), "portrait now")

	// no metadata: made new
	src := byOrig["1000061149"]
	require.NotNil(t, src)
	im, err := openImage(lib.PhotoFile(src))
	require.NoError(t, err)
	bare := filepath.Join(dir, "All", "scan.jpg")
	f, err := os.Create(bare)
	require.NoError(t, err)
	require.NoError(t, jpeg.Encode(f, im, nil))
	f.Close()
	require.NoError(t, lib.Scan(nil, nil))
	scan := lib.Photos(func(ph *Photo) bool { return ph.OrigName == "scan" })
	require.Len(t, scan, 1)
	date = time.Date(1965, 7, 4, 12, 0, 0, 0, time.UTC)
	require.NoError(t, lib.SetDate(scan[0], date))
	assert.Equal(t, date, fileExif(lib.PhotoFile(scan[0])).DateTaken)

	// a PNG: skipped, or converted
	png := byOrig["MountainPicture"]
	require.NotNil(t, png)
	require.NoError(t, lib.SetDate(png, time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)))
	n, skipped, err := lib.WriteMetadata([]*Photo{png}, false, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Equal(t, 1, skipped)
	n, _, err = lib.WriteMetadata([]*Photo{png}, true, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, ".jpg", png.Ext)
	assert.FileExists(t, lib.PhotoFile(png))
	assert.FileExists(t, filepath.Join(dir, "Trash", "Converted", "img_1970_01_01_00_00_00_n0.png"))
	assert.Equal(t, time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC), fileExif(lib.PhotoFile(png)).DateTaken)

	// all of them again: nothing to write
	n, _, err = lib.WriteMetadata(lib.Photos(nil), false, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	if os.Getenv("PHOTOS_KEEP") != "" {
		copyFile(filepath.Join(os.TempDir(), "photos-scan.jpg"), lib.PhotoFile(scan[0]))
		copyFile(filepath.Join(os.TempDir(), "photos-patched.jpg"), file)
	}
}
