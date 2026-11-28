// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestDryRun scans a library of symbolic links to all of the real photos,
// with GoPix, when PHOTOS_DRYRUN is set to its directory, which is kept.
func TestDryRun(t *testing.T) {
	dir := os.Getenv("PHOTOS_DRYRUN")
	if dir == "" {
		t.Skip("PHOTOS_DRYRUN not set")
	}
	home, _ := os.UserHomeDir()
	src := filepath.Join(home, "Pictures", "CogentPhotos", "All")
	all := filepath.Join(dir, "All")
	require.NoError(t, os.MkdirAll(all, 0o755))
	ents, err := os.ReadDir(src)
	require.NoError(t, err)
	for _, e := range ents {
		os.Symlink(filepath.Join(src, e.Name()), filepath.Join(all, e.Name()))
	}
	lib, err := OpenLibrary(dir)
	require.NoError(t, err)
	defer lib.Close()
	start := time.Now()
	last := time.Now()
	require.NoError(t, lib.ScanGoPix(GoPixDir(), func(done, total int) {
		if time.Since(last) > 5*time.Second {
			t.Log(done, total, time.Since(start))
			last = time.Now()
		}
	}))
	t.Log("scan", time.Since(start), "photos", lib.Len())
	tags := lib.Tags()
	t.Log("tags", len(tags))
	renamed, setDates, noExif := 0, 0, 0
	for _, ph := range lib.Photos(nil) {
		if ph.OrigName != ph.Name {
			renamed++
		}
		if ph.DateSet {
			setDates++
		}
		if ph.Make == "" {
			noExif++
		}
	}
	t.Log("renamed", renamed, "dates from names", setDates, "no camera", noExif)
	start = time.Now()
	lib.Close()
	lib, err = OpenLibrary(dir)
	require.NoError(t, err)
	t.Log("reopen", time.Since(start), lib.Len())
	start = time.Now()
	require.NoError(t, lib.Scan(nil, nil))
	t.Log("rescan", time.Since(start), lib.Len())
}
