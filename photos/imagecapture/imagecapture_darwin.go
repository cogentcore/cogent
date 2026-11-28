// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin

package imagecapture

/*
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework ImageCaptureCore -framework Foundation

#include "imagecapture.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"
	"unsafe"
)

// Start starts looking for devices, which must be done before [Devices],
// at the start of the app, as it takes a moment to find them. The main
// thread must be running its run loop, as the event loop of the app does.
func Start() {
	C.icStart()
}

// Devices returns the camera devices that are connected, as cameras and phones.
func Devices() []*Device {
	n := int(C.icNumDevices())
	var res []*Device
	for i := range n {
		cn := C.icDeviceName(C.int(i))
		name := ""
		if cn != nil {
			name = C.GoString(cn)
			C.free(unsafe.Pointer(cn))
		}
		res = append(res, &Device{Name: name, index: i})
	}
	return res
}

// Device is a camera device, as a camera or phone.
type Device struct {

	// Name is the name of the device, as Randy's iPhone.
	Name string

	// index is its index in the list of devices.
	index int
}

// File is a photo or video on a device.
type File struct {

	// Name is the name of the file, as IMG_1234.HEIC.
	Name string

	// Size is the size of the file in bytes.
	Size int64

	// Date is the date the photo was taken, from its metadata, or the
	// date of the file, in the local time zone of the computer.
	Date time.Time

	// Width and Height are the size of the image in pixels.
	Width, Height int

	// Movie is whether it is a video, of the given Duration.
	Movie    bool
	Duration time.Duration

	// index is its index in the files of the device.
	index int
}

// Open opens the device and returns all of its photos and videos, waiting
// up to the given time for it to list them, which can take a while for
// many photos, the first time it is connected: an iPhone must be unlocked,
// and trust the computer. While it is locked, it waits for it to be
// unlocked, calling locked (if not nil) once, to ask for that.
func (d *Device) Open(timeout time.Duration, locked func()) ([]*File, error) {
	C.icOpen(C.int(d.index))
	deadline := time.Now().Add(timeout)
	wasLocked := false
	lastTry := time.Now()
	for {
		var cerr *C.char
		st := C.icState(C.int(d.index), &cerr)
		switch st {
		case 0:
			return nil, errors.New("the device is gone")
		case 2:
			return d.files(), nil
		case 3:
			err := C.GoString(cerr)
			C.free(unsafe.Pointer(cerr))
			return nil, fmt.Errorf("opening %s: %s", d.Name, err)
		case 4:
			if !wasLocked {
				wasLocked = true
				if locked != nil {
					locked()
				}
			}
			// in case it does not say when it is unlocked
			if time.Since(lastTry) > 3*time.Second {
				lastTry = time.Now()
				C.icOpen(C.int(d.index))
			}
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("%s did not list its photos: unlock it, and tap Trust if it asks", d.Name)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// files returns the files of the device, once it is ready.
func (d *Device) files() []*File {
	var n C.int
	cf := C.icFiles(C.int(d.index), &n)
	if cf == nil {
		return nil
	}
	defer C.icFreeFiles(cf, n)
	files := unsafe.Slice(cf, int(n))
	res := make([]*File, len(files))
	for k, f := range files {
		date := float64(f.exifDate)
		if date == 0 {
			date = float64(f.fileDate)
		}
		res[k] = &File{Name: C.GoString(f.name), Size: int64(f.size),
			Width: int(f.width), Height: int(f.height), Movie: f.movie != 0,
			Duration: time.Duration(float64(f.duration) * float64(time.Second)), index: k}
		if date != 0 {
			res[k].Date = time.Unix(int64(date), 0)
		}
	}
	return res
}

// Close closes the device.
func (d *Device) Close() {
	C.icClose(C.int(d.index))
}

// downloads are the channels of the results of the downloads in progress,
// by their ids, under downloadsMu.
var (
	downloads   = map[int64]chan downloadResult{}
	downloadsMu sync.Mutex
	downloadID  int64
)

type downloadResult struct {
	filename string
	err      error
}

//export goDownloaded
func goDownloaded(id C.longlong, filename *C.char, err *C.char) {
	var res downloadResult
	if err != nil {
		res.err = errors.New(C.GoString(err))
	} else {
		res.filename = C.GoString(filename)
	}
	downloadsMu.Lock()
	ch := downloads[int64(id)]
	delete(downloads, int64(id))
	downloadsMu.Unlock()
	if ch != nil {
		ch <- res
	}
}

// Download downloads the original file of the given photo or video into
// the given directory, as the given name, returning its path.
func (d *Device) Download(f *File, dir, name string) (string, error) {
	downloadsMu.Lock()
	downloadID++
	id := downloadID
	ch := make(chan downloadResult, 1)
	downloads[id] = ch
	downloadsMu.Unlock()
	cdir, cname := C.CString(dir), C.CString(name)
	defer C.free(unsafe.Pointer(cdir))
	defer C.free(unsafe.Pointer(cname))
	C.icDownload(C.int(d.index), C.int(f.index), cdir, cname, C.longlong(id))
	select {
	case res := <-ch:
		if res.err != nil {
			return "", fmt.Errorf("downloading %s: %w", f.Name, res.err)
		}
		if res.filename == "" {
			res.filename = name
		}
		return filepath.Join(dir, filepath.Base(res.filename)), nil
	case <-time.After(10 * time.Minute):
		return "", fmt.Errorf("downloading %s: timed out", f.Name)
	}
}

// RunMainLoop runs the given function in another goroutine, while it runs
// the run loop of the main thread, where ImageCaptureCore sends its
// messages, until the function returns, for programs without an event
// loop of their own, as tests and commands. It must be called on the
// main thread, which is locked to the main goroutine in its init.
func RunMainLoop(fun func()) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		fun()
	}()
	for {
		select {
		case <-done:
			return
		default:
			C.icRunLoop(0.05)
		}
	}
}
