// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"bytes"
	"errors"
	"image"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
)

// exifSearchSize is how much of the start of a file is searched for its
// EXIF metadata first, which is near the start in JPEG files.
const exifSearchSize = 512 << 10

// readRawExif returns the raw EXIF metadata of the given file, or
// [exif.ErrNoExif] if it has none.
func readRawExif(filename string) ([]byte, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if ext := strings.ToLower(filenameExt(filename)); ext == ".heic" || ext == ".heif" {
		if raw := heicExif(f); len(raw) > 0 {
			return trimExifHeader(raw), nil
		}
		f.Seek(0, io.SeekStart)
		// some have it elsewhere: searched for below
	}
	buf := make([]byte, exifSearchSize)
	n, err := io.ReadFull(f, buf)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return nil, err
	}
	raw, err := exif.SearchAndExtractExif(buf[:n])
	if err == nil || n < exifSearchSize {
		return raw, err
	}
	all, err := os.ReadFile(filename) // somewhere later in a large file
	if err != nil {
		return nil, err
	}
	return exif.SearchAndExtractExif(all)
}

// trimExifHeader returns the given EXIF data without any "Exif\0\0"
// header and offset before the TIFF header that starts the EXIF data.
func trimExifHeader(raw []byte) []byte {
	for _, h := range [][]byte{[]byte("II*\x00"), []byte("MM\x00*")} {
		if i := bytes.Index(raw, h); i >= 0 && i < 16 {
			return raw[i:]
		}
	}
	return raw
}

// filenameExt returns the extension of the given file name.
func filenameExt(filename string) string {
	_, ext := splitExt(filename)
	return ext
}

// readExif sets the information of the given photo from the EXIF
// metadata of its file, and its size and modification time. The date is
// set from the file if it has no date in its metadata, unless set already.
func (ph *Photo) readExif(filename string) error {
	if isVideoFile(filename) {
		return ph.readVideo(filename)
	}
	st, err := os.Stat(filename)
	if err != nil {
		return err
	}
	ph.FileMod = st.ModTime()
	raw, err := readRawExif(filename)
	if err != nil {
		if !errors.Is(err, exif.ErrNoExif) {
			slog.Debug("photos: reading EXIF", "file", filename, "err", err)
		}
		if ph.DateTaken.IsZero() {
			ph.DateTaken = noZone(ph.FileMod)
		}
		if ph.Size == (image.Point{}) {
			ph.Size = imageSize(filename)
		}
		return nil
	}
	ph.parseExif(raw, filename)
	if ph.Size == (image.Point{}) {
		ph.Size = imageSize(filename)
	}
	return nil
}

// noZone returns the given time as its local time with no time zone (as
// UTC), as the dates of photos are.
func noZone(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.UTC)
}

// parseExif sets the information of the photo from the given raw EXIF data.
func (ph *Photo) parseExif(raw []byte, filename string) {
	entries, _, err := exif.GetFlatExifDataUniversalSearch(raw, nil, false)
	if err != nil && len(entries) == 0 {
		slog.Debug("photos: parsing EXIF", "file", filename, "err", err)
	}
	lat, long := [4]float64{}, [4]float64{}
	var dto, dtd, dtp time.Time
	for i := range entries {
		e := &entries[i]
		val := e.FormattedFirst
		switch e.TagName {
		case "DateTimeOriginal":
			dto, _ = parseExifDate(val)
		case "DateTimeDigitized":
			dtd, _ = parseExifDate(val)
		case "DateTime":
			dtp, _ = parseExifDate(val)
		case "PixelYDimension":
			ph.Size.Y = entryInt(e)
		case "PixelXDimension":
			ph.Size.X = entryInt(e)
		case "Orientation":
			if e.IfdPath == "IFD" { // not that of the thumbnail
				ph.Orient = Orientations(entryInt(e))
			}
		case "ImageDescription":
			if ph.Desc == "" {
				ph.Desc = strings.TrimSpace(strings.Trim(val, "\x00"))
			}
		case "Make":
			ph.Make = strings.TrimSpace(strings.Trim(val, "\x00"))
		case "Model":
			ph.Model = strings.TrimSpace(strings.Trim(val, "\x00"))
		case "ExposureTime":
			ph.Exposure.Time = entryFloat(e)
		case "ISOSpeedRatings":
			ph.Exposure.ISOSpeed = entryFloat(e)
		case "ApertureValue":
			ph.Exposure.Aperture = entryFloat(e)
		case "FocalLength":
			ph.Exposure.FocalLen = entryFloat(e)
		case "FNumber":
			ph.Exposure.FStop = entryFloat(e)
		case "GPSLatitudeRef":
			lat[3] = 1
			if val == "S" {
				lat[3] = -1
			}
		case "GPSLatitude":
			copy(lat[:3], entryFloats(e))
		case "GPSLongitudeRef":
			long[3] = 1
			if val == "W" {
				long[3] = -1
			}
		case "GPSLongitude":
			copy(long[:3], entryFloats(e))
		case "GPSAltitude":
			ph.GPS.Alt = entryFloat(e)
		}
	}
	if ph.Orient < NoOrient || ph.Orient >= OrientUndef {
		ph.Orient = NoOrient
	}
	if !ph.DateSet || ph.DateTaken.IsZero() {
		switch {
		case !dto.IsZero():
			ph.DateTaken = dto
		case !dtd.IsZero():
			ph.DateTaken = dtd
		case !dtp.IsZero():
			ph.DateTaken = dtp
		default:
			ph.DateTaken = noZone(ph.FileMod)
		}
	}
	sign := func(v [4]float64) float64 {
		if v[3] == 0 {
			return 1
		}
		return v[3]
	}
	ph.GPS.Lat = sign(lat) * (lat[0] + lat[1]/60 + lat[2]/3600)
	ph.GPS.Long = sign(long) * (long[0] + long[1]/60 + long[2]/3600)
}

// exifTags returns all of the tags of the EXIF metadata of the given
// file, as name: value lines, for showing them.
func exifTags(filename string) ([][2]string, error) {
	raw, err := readRawExif(filename)
	if err != nil {
		return nil, err
	}
	entries, _, err := exif.GetFlatExifDataUniversalSearch(raw, nil, false)
	if err != nil && len(entries) == 0 {
		return nil, err
	}
	var res [][2]string
	for _, e := range entries {
		switch e.TagName {
		case "MakerNote", "UserComment", "ComponentsConfiguration":
			continue
		}
		val := e.FormattedFirst
		if len(val) > 200 {
			val = val[:200] + "…"
		}
		res = append(res, [2]string{e.TagName, val})
	}
	return res, nil
}

// parseExifDate parses a date in EXIF metadata, which can be in several
// formats besides the standard one.
func parseExifDate(ds string) (time.Time, error) {
	ds = strings.TrimSpace(strings.Trim(ds, "\x00"))
	dt, err := time.Parse("2006:01:02 15:04:05", ds)
	if err == nil {
		return dt, nil
	}
	switch len(ds) {
	case 10, 11: // 2014:09:268
		return time.Parse("2006:01:02", ds[:10])
	case 19: // 2006:11:27:21:33:00
		return time.Parse("2006:01:02:15:04:05", ds)
	}
	if len(ds) > 19 { // with a time zone or fractions
		return time.Parse("2006:01:02 15:04:05", ds[:19])
	}
	return dt, err
}

func entryInt(e *exif.ExifTag) int {
	return int(entryFloat(e))
}

func entryFloat(e *exif.ExifTag) float64 {
	if fs := entryFloats(e); len(fs) > 0 {
		return fs[0]
	}
	return 0
}

func entryFloats(e *exif.ExifTag) []float64 {
	var rf []float64
	switch vl := e.Value.(type) {
	case []uint32:
		for _, v := range vl {
			rf = append(rf, float64(v))
		}
	case []uint16:
		for _, v := range vl {
			rf = append(rf, float64(v))
		}
	case []int32:
		for _, v := range vl {
			rf = append(rf, float64(v))
		}
	case []exifcommon.Rational:
		for _, v := range vl {
			if v.Denominator != 0 {
				rf = append(rf, float64(v.Numerator)/float64(v.Denominator))
			} else {
				rf = append(rf, 0)
			}
		}
	case []exifcommon.SignedRational:
		for _, v := range vl {
			if v.Denominator != 0 {
				rf = append(rf, float64(v.Numerator)/float64(v.Denominator))
			} else {
				rf = append(rf, 0)
			}
		}
	}
	return rf
}
