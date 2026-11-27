// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cogentcore.org/core/video"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Videos are in the library with the photos, named by their dates in the
// same way, with their information from the metadata of their files,
// read with FFmpeg (cogentcore.org/core/video), which also decodes a
// frame of each for its thumbnail, which has a play mark and its length.

// videoExts are the extensions of the files of videos, in lower case.
var videoExts = map[string]bool{".mov": true, ".mp4": true, ".m4v": true,
	".avi": true, ".3gp": true, ".mpg": true, ".mpeg": true, ".mts": true,
	".m2ts": true, ".mkv": true, ".wmv": true, ".webm": true}

// isVideoFile returns whether the given file is a video, by its extension.
func isVideoFile(filename string) bool {
	return videoExts[strings.ToLower(filepath.Ext(filename))]
}

// isMediaFile returns whether the given file is a photo or a video, by
// its extension.
func isMediaFile(filename string) bool {
	return isImageFile(filename) || isVideoFile(filename)
}

// IsVideo returns whether the photo is a video.
func (ph *Photo) IsVideo() bool {
	return videoExts[strings.ToLower(ph.Ext)]
}

// readVideo sets the information of the given video from the metadata
// of its file: its date, size, length, camera, and location. The date is
// from the file time if it has none, unless set already.
func (ph *Photo) readVideo(filename string) error {
	st, err := os.Stat(filename)
	if err != nil {
		return err
	}
	ph.FileMod = st.ModTime()
	m, err := video.OpenMedia(filename, nil)
	if err != nil {
		slog.Debug("photos: reading video", "file", filename, "err", err)
		if ph.DateTaken.IsZero() {
			ph.DateTaken = noZone(ph.FileMod)
		}
		return nil
	}
	defer m.Close()
	ph.Size = image.Pt(m.Width, m.Height) // (turned by its rotation)
	ph.Orient = NoOrient
	ph.Duration = m.Duration
	md := m.Metadata
	first := func(keys ...string) string {
		for _, k := range keys {
			if v := strings.TrimSpace(md[k]); v != "" {
				return v
			}
		}
		return ""
	}
	ph.Make = first("com.apple.quicktime.make", "make")
	ph.Model = first("com.apple.quicktime.model", "model")
	if gps, ok := parseISO6709(first("com.apple.quicktime.location.ISO6709", "location")); ok {
		ph.GPS = gps
	}
	if !ph.DateSet || ph.DateTaken.IsZero() {
		if date, ok := nameDate(filename); ok && md["com.apple.quicktime.creationdate"] == "" {
			ph.DateTaken = date
		} else if date, ok := videoDate(md, ph.GPS); ok {
			ph.DateTaken = date
		} else {
			ph.DateTaken = noZone(ph.FileMod)
		}
	}
	return nil
}

// videoDate returns the date a video was taken, as the local time where
// it was taken, from its metadata: the date of the Apple metadata, which
// is in that local time; else its creation_time, which is in UTC, turned
// into local time by the longitude of its location (to the nearest hour,
// without daylight time), if it has one, as phones do, and otherwise taken
// as the local time, as that of old cameras is.
func videoDate(md map[string]string, gps GPSCoord) (time.Time, bool) {
	for _, k := range []string{"com.apple.quicktime.creationdate", "date"} {
		v := strings.TrimSpace(md[k])
		for _, layout := range []string{"2006-01-02T15:04:05-0700", time.RFC3339, "2006-01-02T15:04:05"} {
			if t, err := time.Parse(layout, v); err == nil && validDate(t) {
				return noZone(t), true
			}
		}
	}
	v := strings.TrimSpace(md["creation_time"])
	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil || !validDate(t) {
		return time.Time{}, false
	}
	t = t.UTC()
	if !gps.IsZero() {
		t = t.Add(time.Duration(math.Round(gps.Long/15)) * time.Hour)
	}
	return noZone(t), true
}

// nameDateRegexp matches the local date and time in the names of the
// files of the cameras of phones, as 20140817_130059 or VID_20140817_130059.
var nameDateRegexp = regexp.MustCompile(`(?:^|[_-])((?:19|20)\d{6})[_-](\d{6})(?:\D|$)`)

// nameDate returns the local date and time in the name of the given file,
// as of the videos of Android phones (20140817_130059.mp4), whose metadata
// has the time in UTC, except for Pixel phones (PXL_), whose names do too.
func nameDate(filename string) (time.Time, bool) {
	name, _ := splitExt(filename)
	if strings.HasPrefix(name, "PXL_") {
		return time.Time{}, false
	}
	m := nameDateRegexp.FindStringSubmatch(name)
	if m == nil {
		return time.Time{}, false
	}
	t, err := time.Parse("20060102150405", m[1]+m[2])
	if err != nil || !validDate(t) {
		return time.Time{}, false
	}
	return t, true
}

// validDate returns whether the given date in the metadata of a video is
// a real one, not one of the zero dates of cameras that do not set it.
func validDate(t time.Time) bool {
	return t.Year() > 1980 && t.Before(time.Now().Add(48*time.Hour))
}

// iso6709 matches a location in ISO 6709, as +39.9170+116.4026+061.502/.
var iso6709 = regexp.MustCompile(`^([+-]\d+(?:\.\d+)?)([+-]\d+(?:\.\d+)?)([+-]\d+(?:\.\d+)?)?`)

// parseISO6709 returns the location in the given text in ISO 6709, as in
// the metadata of the videos of phones.
func parseISO6709(s string) (GPSCoord, bool) {
	m := iso6709.FindStringSubmatch(s)
	if m == nil {
		return GPSCoord{}, false
	}
	var g GPSCoord
	g.Lat, _ = strconv.ParseFloat(m[1], 64)
	g.Long, _ = strconv.ParseFloat(m[2], 64)
	if m[3] != "" {
		g.Alt, _ = strconv.ParseFloat(m[3], 64)
	}
	return g, !g.IsZero()
}

// videoFrame returns a frame of the given video, at the given size, a
// second into it if it is longer than two seconds (the first frames are
// often dark), or its first frame.
func videoFrame(m *video.Media, width, height int) (*image.RGBA, error) {
	start := time.Duration(0)
	if m.Duration > 2*time.Second {
		start = time.Second
	}
	for {
		img, err := videoFrameAt(m, start, width, height)
		if img != nil || start == 0 {
			return img, err
		}
		start = 0 // as for a wrong duration
	}
}

// videoFrameAt returns the first frame of the given video at or after the
// given time, at the given size.
func videoFrameAt(m *video.Media, start time.Duration, width, height int) (*image.RGBA, error) {
	done := make(chan struct{})
	defer close(done)
	frames, _, errs, err := m.DecodeFrom(start, width, height, false, done)
	if err != nil {
		return nil, err
	}
	for f := range frames {
		if f.Image != nil {
			img := image.NewRGBA(f.Image.Bounds()) // (the decoder reuses it)
			copy(img.Pix, f.Image.Pix)
			return img, nil
		}
	}
	for err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return nil, errors.New("video: no frames")
}

// fitSize returns the given size scaled down to fit in the given maximum
// width and height, keeping its aspect ratio.
func fitSize(sz image.Point, maxSize int) image.Point {
	if sz.X <= maxSize && sz.Y <= maxSize {
		return sz
	}
	if sz.X >= sz.Y {
		return image.Pt(maxSize, max(1, sz.Y*maxSize/sz.X))
	}
	return image.Pt(max(1, sz.X*maxSize/sz.Y), maxSize)
}

// openVideoFrame returns a frame of the given video, upright, fitting in
// the given maximum size, as for showing it before it plays.
func openVideoFrame(ph *Photo, filename string, maxSize int) (image.Image, error) {
	m, err := video.OpenMedia(filename, nil)
	if err != nil {
		return nil, err
	}
	defer m.Close()
	sz := fitSize(image.Pt(m.Width, m.Height), maxSize)
	if sz.X == 0 || sz.Y == 0 {
		return nil, fmt.Errorf("%s: no video", filepath.Base(filename))
	}
	img, err := videoFrame(m, sz.X, sz.Y)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(filename), err)
	}
	return orientImage(img, NoOrient, ph.Rotation), nil
}

// makeVideoThumb makes the thumbnail of the given video, in the given
// file, saving it in the given file of the thumbnail: a frame of it, with
// its date, and a play mark with its length.
func makeVideoThumb(ph *Photo, filename, thumbFile string) (image.Image, error) {
	img, err := openVideoFrame(ph, filename, ThumbSize)
	if err != nil {
		return nil, err
	}
	img = drawDate(img, ph.DateTaken)
	img = drawPlayMark(img, ph.Duration, 1)
	if err := os.MkdirAll(filepath.Dir(thumbFile), 0o755); err != nil {
		return img, err
	}
	tmp := thumbFile + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return img, err
	}
	err = jpeg.Encode(f, img, &jpeg.Options{Quality: 85})
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return img, err
	}
	return img, os.Rename(tmp, thumbFile)
}

// drawPlayMark draws a play mark at the bottom left of the given image,
// a white triangle in a dark circle, with the given length of the video
// after it if it is known, at the given scale (1 for thumbnails).
func drawPlayMark(img image.Image, length time.Duration, scale float64) image.Image {
	rgba := asRGBA(img)
	b := rgba.Bounds()
	r := 11 * scale
	cx, cy := float64(b.Min.X)+r+5*scale, float64(b.Max.Y)-r-5*scale
	dark := color.RGBA{0, 0, 0, 150}
	for y := int(cy - r); y <= int(cy+r); y++ {
		for x := int(cx - r); x <= int(cx+r); x++ {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			if dx*dx+dy*dy > r*r || !image.Pt(x, y).In(b) {
				continue
			}
			// the triangle: equilateral, pointing right, centered by its area
			tx, ty := dx+r*0.12, dy
			in := tx > -r*0.35 && tx < r*0.5 && math.Abs(ty) < (r*0.5-tx)*0.577
			if in {
				rgba.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
			} else {
				blend(rgba, x, y, dark)
			}
		}
	}
	face := thumbFace()
	if length <= 0 || face == nil || scale != 1 {
		return rgba
	}
	s := formatLength(length)
	x := int(cx + r + 4)
	thumbFaceMu.Lock()
	defer thumbFaceMu.Unlock()
	y := int(cy) + face.Metrics().Ascent.Ceil()/2 - 1
	for _, d := range []image.Point{{1, 1}, {-1, 0}, {0, -1}, {1, 0}, {0, 1}} { // an outline
		(&font.Drawer{Dst: rgba, Src: image.NewUniform(color.RGBA{0, 0, 0, 200}), Face: face, Dot: fixed.P(x+d.X, y+d.Y)}).DrawString(s)
	}
	(&font.Drawer{Dst: rgba, Src: image.NewUniform(color.White), Face: face, Dot: fixed.P(x, y)}).DrawString(s)
	return rgba
}

// blend blends the given color over the pixel at x, y.
func blend(img *image.RGBA, x, y int, c color.RGBA) {
	draw.Draw(img, image.Rect(x, y, x+1, y+1), image.NewUniform(c), image.Point{}, draw.Over)
}

// formatLength returns the given length of a video as m:ss, or h:mm:ss.
func formatLength(d time.Duration) string {
	s := int(d.Round(time.Second).Seconds())
	if s >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
	}
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

// videoThumbsVersion is the version of the thumbnails of videos, which
// are made again when it changes (see [Library.updateVideoThumbs]):
// 2 has the play mark of the right shape.
const videoThumbsVersion = "2"

// updateVideoThumbs removes the thumbnails of the videos if they are of
// another version (see [videoThumbsVersion]), for making them again.
func (lib *Library) updateVideoThumbs() error {
	if lib.store.meta("videoThumbs") == videoThumbsVersion {
		return nil
	}
	for _, ph := range lib.Photos((*Photo).IsVideo) {
		os.Remove(lib.thumbFile(ph))
	}
	return lib.store.setMeta("videoThumbs", videoThumbsVersion)
}
