// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"image"
	"os"
	"strconv"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVideoDate(t *testing.T) {
	d, ok := videoDate(map[string]string{"com.apple.quicktime.creationdate": "2015-08-10T18:55:44+0800",
		"creation_time": "2015-08-10T10:55:44.000000Z"}, GPSCoord{})
	assert.True(t, ok)
	assert.Equal(t, time.Date(2015, 8, 10, 18, 55, 44, 0, time.UTC), d, "the local time of the Apple date")
	d, _ = videoDate(map[string]string{"creation_time": "2021-10-26T01:25:24.000000Z"}, GPSCoord{Lat: 38.5499, Long: -121.7557})
	assert.Equal(t, time.Date(2021, 10, 25, 17, 25, 24, 0, time.UTC), d, "UTC turned local by the longitude")
	d, _ = videoDate(map[string]string{"creation_time": "2005-08-27T17:57:27.000000Z"}, GPSCoord{})
	assert.Equal(t, time.Date(2005, 8, 27, 17, 57, 27, 0, time.UTC), d, "the local time of old cameras")
	_, ok = videoDate(map[string]string{"creation_time": "1970-01-01T00:00:00.000000Z"}, GPSCoord{})
	assert.False(t, ok)
	d, ok = nameDate("/x/20140817_130059.mp4")
	assert.True(t, ok)
	assert.Equal(t, time.Date(2014, 8, 17, 13, 0, 59, 0, time.UTC), d)
	_, ok = nameDate("PXL_20211026_012524123.mp4")
	assert.False(t, ok)
	_, ok = nameDate("img_2015_03_02_14_23_50_n0.mp4")
	assert.False(t, ok)
	g, ok := parseISO6709("+39.9170+116.4026+061.502/")
	assert.True(t, ok)
	assert.Equal(t, GPSCoord{39.917, 116.4026, 61.502}, g)
	assert.Equal(t, "0:22", formatLength(22400*time.Millisecond))
	assert.Equal(t, "1:02:03", formatLength(time.Hour+2*time.Minute+3*time.Second))
}

// TestRealVideos reads the videos of each kind in the real library, if
// it is there, and makes their thumbnails, saved in the temporary
// directory for looking at them.
func TestRealVideos(t *testing.T) {
	home, _ := os.UserHomeDir()
	src := filepath.Join(home, "Pictures", "CogentPhotos", "All")
	ents, err := os.ReadDir(src)
	if err != nil {
		t.Skip("no real library")
	}
	seen := map[string]bool{}
	n := 0
	for _, e := range ents {
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if !isVideoFile(e.Name()) || seen[ext] {
			continue
		}
		seen[ext] = true
		fn := filepath.Join(src, e.Name())
		ph := &Photo{Ext: ext}
		start := time.Now()
		require.NoError(t, ph.readVideo(fn))
		read := time.Since(start)
		assert.False(t, ph.DateTaken.IsZero(), e.Name()) // (some have no date but a bad file date)
		assert.NotZero(t, ph.Size.X, e.Name())
		assert.NotZero(t, ph.Duration, e.Name())
		start = time.Now()
		tf := filepath.Join(t.TempDir(), "thumb.jpg")
		img, err := makeVideoThumb(ph, fn, tf)
		require.NoError(t, err, e.Name())
		t.Log(e.Name(), ph.DateTaken, ph.Size, ph.Duration, ph.Make, ph.Model, ph.GPS, "read", read, "thumb", time.Since(start), img.Bounds())
		imagex.Save(img, filepath.Join(os.TempDir(), "photos-video-"+strings.TrimPrefix(ext, ".")+".png"))
		n++
	}
	if n == 0 {
		t.Skip("no videos")
	}
}

// TestDrawConcurrent tests that the dates and play marks of thumbnails
// are drawn safely in parallel, as the thumbnails are made (run with -race).
func TestDrawConcurrent(t *testing.T) {
	var phs []*Photo
	for i := range 64 {
		phs = append(phs, &Photo{Name: strconv.Itoa(i)})
	}
	parallel(phs, func(ph *Photo) {
		img := image.NewRGBA(image.Rect(0, 0, 256, 192))
		drawDate(img, time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC))
		drawPlayMark(img, 75*time.Second, 1)
	})
}

// TestVideoNames tests that videos are named vid_, and those named img_
// before are renamed so when scanned, with their tags and thumbnails.
func TestVideoNames(t *testing.T) {
	lib, dir := testLibrary(t)
	home, _ := os.UserHomeDir()
	src := filepath.Join(home, "Pictures", "CogentPhotos", "All")
	ents, _ := os.ReadDir(src)
	var vfile string
	for _, e := range ents {
		if strings.HasSuffix(e.Name(), ".3gp") {
			vfile = filepath.Join(src, e.Name())
			break
		}
	}
	if vfile == "" {
		t.Skip("no video")
	}
	require.NoError(t, copyFile(filepath.Join(dir, "All", "clip.3gp"), vfile))
	require.NoError(t, lib.Scan(nil, nil))
	vids := lib.Photos((*Photo).IsVideo)
	require.Len(t, vids, 1)
	v := vids[0]
	assert.True(t, strings.HasPrefix(v.Name, "vid_"), v.Name)

	// as named before: img_
	old := "img" + strings.TrimPrefix(v.Name, "vid")
	vname := v.Name
	lib.mu.Lock()
	require.NoError(t, lib.renameTo(v, old, v.Ext))
	lib.mu.Unlock()
	require.NoError(t, lib.store.rename(vname, v))
	require.NoError(t, lib.AddTag("2012_CampingMiscVideo", v))
	_, err := lib.Thumb(v)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(dir, "All", old+".3gp"))

	require.NoError(t, lib.Scan(nil, nil))
	assert.Equal(t, vname, v.Name)
	assert.FileExists(t, filepath.Join(dir, "All", vname+".3gp"))
	assert.FileExists(t, lib.thumbFile(v))
	assert.Nil(t, lib.Photo(old))
	require.NoError(t, lib.Close())
	lib2, err := OpenLibrary(dir)
	require.NoError(t, err)
	defer lib2.Close()
	v2 := lib2.Photo(vname)
	require.NotNil(t, v2, "stored by its new name")
	assert.Equal(t, []string{"2012_CampingMiscVideo"}, v2.Tags)
	assert.Equal(t, len(realPhotos)+1, lib2.Len())
	res, err := lib2.Search("kind:video")
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{vname: true}, res)
}
