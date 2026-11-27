// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"context"
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/cogent/photos/imagecapture"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeDevice is a device whose files are the given files, by their names.
type fakeDevice struct {
	files map[string]string
}

func (fd *fakeDevice) Download(f *imagecapture.File, dir, name string) (string, error) {
	to := filepath.Join(dir, name)
	return to, copyFile(to, fd.files[f.Name])
}

func (fd *fakeDevice) Close() {}

// TestImportDevice tests importing from a device: new photos and videos
// are imported, those of Live Photos are not, those here already are
// not, and those imported are not imported again.
func TestImportDevice(t *testing.T) {
	lib := smallLibrary(t, 2)
	home, _ := os.UserHomeDir()
	src := filepath.Join(home, "Pictures", "CogentPhotos", "All")
	heic := filepath.Join(src, "img_2023_05_08_00_21_26_n0.heic")
	mov := filepath.Join(src, "vid_2015_08_10_18_55_34_n0.mov")
	for _, f := range []string{heic, mov} {
		if _, err := os.Stat(f); err != nil {
			t.Skip("no real photos")
		}
	}
	existing := lib.Photos(nil)[0]
	fd := &fakeDevice{files: map[string]string{"IMG_0001.HEIC": heic, "IMG_0002.MOV": mov,
		"IMG_0001.MOV": mov, "IMG_0003.JPG": lib.PhotoFile(existing)}}
	files := []*imagecapture.File{
		{Name: "IMG_0001.HEIC", Size: 1, Date: time.Date(2023, 5, 8, 0, 21, 26, 0, time.Local), Width: 4032, Height: 3024},
		{Name: "IMG_0001.MOV", Size: 2, Movie: true}, // its Live Photo
		{Name: "IMG_0002.MOV", Size: 3, Movie: true, Date: time.Date(2015, 8, 10, 18, 55, 34, 0, time.Local)},
		{Name: "IMG_0003.JPG", Size: 4, Date: existing.DateTaken, Width: existing.Size.X, Height: existing.Size.Y}, // here already
		{Name: "IMG_0003.AAE", Size: 5},                                                                            // not a photo
	}
	di := lib.newOnDevice(fd, "Test iPhone", files)
	assert.Equal(t, 1, di.photos)
	assert.Equal(t, 1, di.videos)
	assert.Equal(t, 1, di.live, "the video of the Live Photo")
	assert.Equal(t, 1, di.already, "the photo here already")
	// canceled after the first file
	ctx, cancel := context.WithCancel(context.Background())
	first, err := lib.importDevice(ctx, di, []string{"2026_Phone"}, func(i, n int) {
		if i == 1 {
			cancel()
		}
	})
	require.NoError(t, err)
	require.Len(t, first, 1, "imported before it was canceled")
	di = lib.newOnDevice(fd, "Test iPhone", files)
	require.Len(t, di.files, 1, "the rest")
	phs, err := lib.importDevice(context.Background(), di, []string{"2026_Phone"}, nil)
	require.NoError(t, err)
	require.Len(t, phs, 1)
	phs = append(first, phs...)
	assert.Equal(t, 4, lib.Len())
	assert.Equal(t, 2, lib.Tags()["2026_Phone"])
	for _, ph := range phs {
		assert.FileExists(t, lib.PhotoFile(ph))
		assert.NotEqual(t, image.Point{}, ph.Size)
		t.Log(ph.Name, ph.OrigName, ph.GPS)
	}

	// again: nothing new, as they were imported
	di = lib.newOnDevice(fd, "Test iPhone", files)
	assert.Empty(t, di.files)
	assert.Equal(t, 3, di.already)
	// even when they are deleted here
	require.NoError(t, lib.Delete(phs...))
	di = lib.newOnDevice(fd, "Test iPhone", files)
	assert.Empty(t, di.files, "imported before")
}
