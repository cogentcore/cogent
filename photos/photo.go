// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

//go:generate core generate

import (
	"fmt"
	"image"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Photo is the information about a photo, in the store of the photos,
// by its [Photo.Name]: that from the metadata (EXIF) of its file, and
// that added here, such as its tags.
type Photo struct {

	// Name is the name of the file of the photo, without its extension,
	// which is unique among the photos, and is the key of the photo in
	// the store. It is made from the date the photo was taken (see
	// [nameFor]), as img_2006_01_02_15_04_05_n0.
	Name string `edit:"-"`

	// Ext is the extension of the file, in lower case, as .jpg.
	Ext string `edit:"-"`

	// OrigName is the name of the file of the photo, without its
	// extension, before it was renamed by its date, for finding it by
	// that name, as when importing the folders of GoPix.
	OrigName string `edit:"-"`

	// Desc is a description of the photo.
	Desc string

	// Tags are the tags of the photo, which are like albums, except that
	// a photo can have any number of them. A tag that starts with a year
	// and an underscore, as 2017_Kyoto, is shown in that year.
	Tags []string

	// DateTaken is when the photo was taken, as the local time where it
	// was taken, with no time zone (as UTC), as in the metadata of photos.
	DateTaken time.Time

	// DateSet is whether [Photo.DateTaken] was set here (or by GoPix),
	// instead of being from the metadata, which it is then not set from
	// again.
	DateSet bool `edit:"-"`

	// Number is the number of the photo among those taken at the same
	// second, which is in its name.
	Number int `edit:"-"`

	// FileMod is the modification time of the file, when its metadata was
	// read, for reading it again when the file changes.
	FileMod time.Time `edit:"-"`

	// Size is the size of the image in pixels, as stored, before its
	// orientation.
	Size image.Point `edit:"-"`

	// Orient is the orientation of the image in its metadata.
	Orient Orientations `edit:"-"`

	// Rotation is the rotation of the photo set here, in degrees
	// clockwise (0, 90, 180, or 270), after its [Photo.Orient].
	Rotation int `edit:"-"`

	// GPS is the location where the photo was taken, if known.
	GPS GPSCoord `edit:"-"`

	// Exposure is the exposure of the photo.
	Exposure Exposure `edit:"-"`

	// Make and Model are those of the camera.
	Make, Model string `edit:"-"`

	// Trashed is when the photo was moved to the trash, or zero if it is
	// not in the trash. Photos in the trash are deleted for good after
	// [trashDays].
	Trashed time.Time `edit:"-"`

	// GoogleID is the ID of the photo in Google Photos, if it was
	// downloaded from there, so that it is not downloaded again.
	GoogleID string `edit:"-"`
}

// GPSCoord is a GPS position as decimal degrees.
type GPSCoord struct {

	// Lat is the latitude in decimal degrees, from -90 to 90.
	Lat float64

	// Long is the longitude in decimal degrees, from -180 to 180.
	Long float64

	// Alt is the altitude in meters.
	Alt float64
}

// IsZero returns whether there is no location.
func (g GPSCoord) IsZero() bool {
	return g.Lat == 0 && g.Long == 0
}

// Exposure is the standard information about the exposure of a photo.
type Exposure struct {
	Time     float64
	FStop    float64
	ISOSpeed float64
	FocalLen float64
	Aperture float64
}

// Filename returns the name of the file of the photo, without its directory.
func (ph *Photo) Filename() string {
	return ph.Name + ph.Ext
}

// HasTag returns whether the photo has the given tag.
func (ph *Photo) HasTag(tag string) bool {
	for _, t := range ph.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// AddTag adds the given tag, returning whether it was added (if not
// already there).
func (ph *Photo) AddTag(tag string) bool {
	if tag == "" || ph.HasTag(tag) {
		return false
	}
	ph.Tags = append(ph.Tags, tag)
	return true
}

// RemoveTag removes the given tag, returning whether it was there.
func (ph *Photo) RemoveTag(tag string) bool {
	for i, t := range ph.Tags {
		if t == tag {
			ph.Tags = append(ph.Tags[:i], ph.Tags[i+1:]...)
			return true
		}
	}
	return false
}

// DisplaySize returns the size of the photo as shown, after its
// orientation and rotation.
func (ph *Photo) DisplaySize() image.Point {
	sz := ph.Size
	if ph.Orient.Swaps() {
		sz.X, sz.Y = sz.Y, sz.X
	}
	if ph.Rotation == 90 || ph.Rotation == 270 {
		sz.X, sz.Y = sz.Y, sz.X
	}
	return sz
}

// Orientations are the orientations of images in EXIF metadata, with
// the values of the standard, which say how the pixels as stored are to
// be turned to be upright. The names are those of how the image is
// stored, as in GoPix.
type Orientations int32 //enums:enum

const (
	// NoOrient means no orientation was set: the image is upright.
	NoOrient Orientations = iota

	// Rotated0 means the image is upright as is.
	Rotated0

	// FlippedH means the image is flipped horizontally (mirrored).
	FlippedH

	// Rotated180 means the image is turned 180 degrees.
	Rotated180

	// FlippedV means the image is flipped vertically.
	FlippedV

	// FlippedHRotated90L means the image is flipped horizontally and
	// turned 90 degrees left (transposed).
	FlippedHRotated90L

	// Rotated90L means the image is turned 90 degrees left: it is shown
	// turned 90 degrees clockwise.
	Rotated90L

	// FlippedHRotated90R means the image is flipped horizontally and
	// turned 90 degrees right (transversed).
	FlippedHRotated90R

	// Rotated90R means the image is turned 90 degrees right: it is shown
	// turned 90 degrees counterclockwise.
	Rotated90R

	// OrientUndef is an undefined orientation.
	OrientUndef
)

// Swaps returns whether the orientation swaps the width and height.
func (o Orientations) Swaps() bool {
	return o >= FlippedHRotated90L && o <= Rotated90R
}

// DateFileFormat is the format of the dates in the names of the files of
// the photos, by the date they were taken.
const DateFileFormat = "2006_01_02_15_04_05"

// nameFor returns the name of the file (without extension) of a photo
// taken at the given date, with the given number among those taken in
// the same second.
func nameFor(date time.Time, n int) string {
	return fmt.Sprintf("img_%s_n%d", date.Format(DateFileFormat), n)
}

// nameRegexp matches the names of files made by [nameFor].
var nameRegexp = regexp.MustCompile(`^img_(\d{4}_\d{2}_\d{2}_\d{2}_\d{2}_\d{2})_n(\d+)$`)

// parseName returns the date and number in the given name of a file
// (without extension) made by [nameFor], and whether it is one.
func parseName(name string) (time.Time, int, bool) {
	m := nameRegexp.FindStringSubmatch(name)
	if m == nil {
		return time.Time{}, 0, false
	}
	date, err := time.Parse(DateFileFormat, m[1])
	if err != nil {
		return time.Time{}, 0, false
	}
	n, _ := strconv.Atoi(m[2])
	return date, n, true
}

// splitExt returns the name of the given file without its directory and
// extension, and its extension.
func splitExt(filename string) (string, string) {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext), ext
}

// imageExts are the extensions of the files of images, in lower case.
var imageExts = map[string]bool{".jpg": true, ".jpeg": true, ".png": true,
	".heic": true, ".heif": true, ".gif": true, ".tif": true, ".tiff": true,
	".bmp": true, ".webp": true}

// isImageFile returns whether the given file is an image, by its extension.
func isImageFile(filename string) bool {
	return imageExts[strings.ToLower(filepath.Ext(filename))]
}

// splitYearTag returns the year and the rest of the given tag, if it
// starts with a year and an underscore, as 2017_Kyoto, or "" and the tag.
func splitYearTag(tag string) (year, name string) {
	if len(tag) > 5 && tag[4] == '_' {
		if _, err := strconv.Atoi(tag[:4]); err == nil {
			return tag[:4], tag[5:]
		}
	}
	return "", tag
}
