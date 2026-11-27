// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imagecapture

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func init() {
	runtime.LockOSThread() // the main thread, for the run loop
}

// TestMain runs the tests with the run loop of the main thread running,
// where ImageCaptureCore sends its messages.
func TestMain(m *testing.M) {
	code := 0
	RunMainLoop(func() { code = m.Run() })
	os.Exit(code)
}

// TestDevice lists the photos of a connected camera or phone, if any, and
// downloads the newest one.
func TestDevice(t *testing.T) {
	Start()
	var devs []*Device
	for range 30 { // finding them takes a moment
		if devs = Devices(); len(devs) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if len(devs) == 0 {
		t.Skip("no camera or phone connected")
	}
	d := devs[0]
	t.Log("device:", d.Name)
	start := time.Now()
	files, err := d.Open(5*time.Minute, func() { t.Log("unlock", d.Name) })
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	t.Log(len(files), "files, listed in", time.Since(start))
	var newest *File
	movies := 0
	for _, f := range files {
		if f.Movie {
			movies++
		}
		if newest == nil || f.Date.After(newest.Date) {
			newest = f
		}
	}
	t.Log(movies, "videos")
	for _, f := range files[max(0, len(files)-5):] {
		t.Logf("%s %d %v %dx%d movie %v %v", f.Name, f.Size, f.Date, f.Width, f.Height, f.Movie, f.Duration)
	}
	if newest == nil {
		return
	}
	dir := t.TempDir()
	start = time.Now()
	fn, err := d.Download(newest, dir, newest.Name)
	if err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(fn)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("downloaded", filepath.Base(fn), st.Size(), "bytes in", time.Since(start))
	if st.Size() != newest.Size {
		t.Errorf("downloaded %d bytes, not %d", st.Size(), newest.Size)
	}
	if os.Getenv("IMAGECAPTURE_KEEP") != "" {
		os.Rename(fn, filepath.Join(os.TempDir(), filepath.Base(fn)))
	}
}
