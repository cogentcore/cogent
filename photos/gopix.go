// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"bufio"
	"encoding/json"
	"image"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GoPix is the predecessor of Cogent Photos, which had its photos in
// ~/Pix/All, with the information about them in ~/Pix/info.json, and
// folders of symbolic links to them in ~/Pix, which are tags here.

// gopixInfo is the information about a photo in the info.json of GoPix.
type gopixInfo struct {
	Ext       string
	Desc      string
	FileMod   time.Time
	Number    int
	Size      image.Point
	Orient    string
	DateTaken time.Time
	GPSLoc    GPSCoord
	Exposure  Exposure
	Tags      map[string]string
}

// readGoPixInfo returns the photos in the given info.json file of GoPix,
// by their names (without extensions).
func readGoPixInfo(filename string) (map[string]*Photo, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	infos := map[string]*gopixInfo{}
	if err := json.NewDecoder(bufio.NewReader(f)).Decode(&infos); err != nil {
		return nil, err
	}
	res := make(map[string]*Photo, len(infos))
	for name, gi := range infos {
		ph := &Photo{Name: name, Ext: gi.Ext, Desc: gi.Desc, FileMod: gi.FileMod,
			Number: gi.Number, Size: gi.Size, DateTaken: noZone(gi.DateTaken),
			GPS: gi.GPSLoc, Exposure: gi.Exposure}
		ph.Orient.SetString(gi.Orient) // as named here
		if ph.Orient >= OrientUndef {
			ph.Orient = NoOrient
		}
		if gi.Tags != nil {
			ph.Make, ph.Model = strings.TrimSpace(gi.Tags["Make"]), strings.TrimSpace(gi.Tags["Model"])
		}
		res[name] = ph
	}
	return res, nil
}

// GoPixDir returns the directory of the photos of GoPix, ~/Pix, or ""
// if there is none.
func GoPixDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(home, "Pix")
	if st, err := os.Stat(filepath.Join(dir, "All")); err != nil || !st.IsDir() {
		return ""
	}
	return dir
}

// ScanGoPix scans the library (see [Library.Scan]) with the information
// about the photos from GoPix in the given directory, for those that are
// new, and then imports its folders as tags (see [Library.ImportGoPixFolders]).
func (lib *Library) ScanGoPix(dir string, progress ScanProgress) error {
	old, err := readGoPixInfo(filepath.Join(dir, "info.json"))
	if err != nil {
		slog.Error("photos: reading the information of GoPix", "err", err)
	}
	if err := lib.Scan(old, progress); err != nil {
		return err
	}
	return lib.ImportGoPixFolders(dir)
}

// ImportGoPixFolders adds tags to the photos for the folders of GoPix in
// the given directory: each folder other than All and Trash is a tag of
// the photos that it has links to, which are found by their names in
// GoPix (see [Photo.OrigName]).
func (lib *Library) ImportGoPixFolders(dir string) error {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	lib.mu.Lock()
	byOrig := make(map[string]*Photo, len(lib.photos))
	for _, ph := range lib.photos {
		byOrig[ph.Name] = ph
	}
	for _, ph := range lib.photos { // those renamed here win
		if ph.OrigName != "" {
			byOrig[ph.OrigName] = ph
		}
	}
	changed := map[*Photo]bool{}
	for _, e := range ents {
		tag := e.Name()
		if !e.IsDir() || tag == "All" || tag == "Trash" || strings.HasPrefix(tag, ".") {
			continue
		}
		links, err := os.ReadDir(filepath.Join(dir, tag))
		if err != nil {
			slog.Error("photos: reading GoPix folder", "folder", tag, "err", err)
			continue
		}
		missing := 0
		for _, l := range links {
			fn := l.Name()
			if l.Type()&os.ModeSymlink != 0 {
				if target, err := os.Readlink(filepath.Join(dir, tag, fn)); err == nil {
					fn = filepath.Base(target)
				}
			}
			if !isImageFile(fn) {
				continue
			}
			name, _ := splitExt(fn)
			ph := byOrig[name]
			if ph == nil {
				missing++
				continue
			}
			if ph.AddTag(tag) {
				changed[ph] = true
			}
		}
		if missing > 0 {
			slog.Info("photos: GoPix folder has photos not found", "folder", tag, "n", missing)
		}
	}
	var phs []*Photo
	for ph := range changed {
		phs = append(phs, ph)
	}
	lib.mu.Unlock()
	if err := lib.Save(phs...); err != nil {
		return err
	}
	return lib.store.setMeta("gopixImported", time.Now().Format(time.RFC3339))
}

// GoPixImported returns whether the folders of GoPix have been imported.
func (lib *Library) GoPixImported() bool {
	return lib.store.meta("gopixImported") != ""
}
