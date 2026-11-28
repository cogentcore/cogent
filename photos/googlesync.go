// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"context"
	"errors"
	"fmt"
	"image"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// incomingDir is the directory in the library where photos are
// downloaded, before they are added to it.
const incomingDir = ".incoming"

// GoogleResult is the result of adding photos picked in Google Photos.
type GoogleResult struct {

	// Added are the photos added.
	Added []*Photo

	// Already is the number of photos that were already in the library:
	// downloaded before, or the same as one in it, by the date they were
	// taken and their size.
	Already int

	// Videos is the number of videos skipped, which are not handled yet.
	Videos int
}

func (r *GoogleResult) String() string {
	s := fmt.Sprintf("Added %d photos", len(r.Added))
	if r.Already > 0 {
		s += fmt.Sprintf("; %d were already here", r.Already)
	}
	if r.Videos > 0 {
		s += fmt.Sprintf("; skipped %d videos", r.Videos)
	}
	return s
}

// sameKey returns the key of a photo for finding the same photo from
// elsewhere: the date it was taken and its size, in either orientation.
func sameKey(date time.Time, size image.Point) string {
	w, h := size.X, size.Y
	if w > h {
		w, h = h, w
	}
	return fmt.Sprintf("%s %dx%d", date.Format(DateFileFormat), w, h)
}

// addGoogle downloads the given items picked in Google Photos and adds
// those that are not in the library already, renamed by their dates.
func (lib *Library) addGoogle(ctx context.Context, pc *pickerClient, items []*pickedItem, progress ScanProgress) (*GoogleResult, error) {
	res := &GoogleResult{}
	incoming := filepath.Join(lib.Dir, incomingDir)
	if err := os.MkdirAll(incoming, 0o755); err != nil {
		return res, err
	}
	defer os.RemoveAll(incoming)

	lib.mu.RLock()
	byGoogle := map[string]*Photo{}
	bySame := map[string]*Photo{}
	for _, ph := range lib.photos {
		if ph.GoogleID != "" {
			byGoogle[ph.GoogleID] = ph
		}
		if ph.Size != (image.Point{}) {
			bySame[sameKey(ph.DateTaken, ph.Size)] = ph
		}
	}
	lib.mu.RUnlock()

	var errs []error
	var known []*Photo // existing photos now with a Google ID
	for i, it := range items {
		if progress != nil {
			progress(i, len(items))
		}
		if ctx.Err() != nil {
			errs = append(errs, ctx.Err())
			break
		}
		if it.Type == "VIDEO" || strings.HasPrefix(it.MediaFile.MimeType, "video/") {
			res.Videos++
			continue
		}
		if byGoogle[it.ID] != nil {
			res.Already++
			continue
		}
		name := filepath.Base(it.MediaFile.Filename)
		if name == "." || name == "/" || name == "" || !isImageFile(name) {
			name = it.ID + extForMime(it.MediaFile.MimeType)
		}
		file := filepath.Join(incoming, name)
		if err := pc.download(ctx, it, file); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
			continue
		}
		if t := it.created(); !t.IsZero() { // the date, if it has none in its metadata
			os.Chtimes(file, t, t)
		}
		ph := &Photo{}
		if err := ph.readExif(file); err != nil {
			errs = append(errs, err)
			continue
		}
		if same := bySame[sameKey(ph.DateTaken, ph.Size)]; same != nil {
			res.Already++
			if same.GoogleID == "" {
				same.GoogleID = it.ID
				known = append(known, same)
			}
			os.Remove(file)
			continue
		}
		nph, err := lib.importFile(file, true)
		if errors.Is(err, errAlready) {
			res.Already++
			os.Remove(file)
			continue
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		nph.GoogleID = it.ID
		res.Added = append(res.Added, nph)
		bySame[sameKey(nph.DateTaken, nph.Size)] = nph
		byGoogle[it.ID] = nph
	}
	errs = append(errs, lib.Save(append(res.Added, known...)...))
	err := errors.Join(errs...)
	if err != nil {
		slog.Error("photos: adding photos from Google Photos", "err", err)
	}
	return res, err
}

// extForMime returns the extension of files of the given mime type of
// images, for those whose names are not known.
func extForMime(mime string) string {
	switch mime {
	case "image/heic", "image/heif":
		return ".heic"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/tiff":
		return ".tif"
	}
	return ".jpg"
}
