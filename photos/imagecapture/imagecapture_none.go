// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !darwin

package imagecapture

import (
	"errors"
	"time"
)

// Start does nothing where there is no ImageCaptureCore.
func Start() {}

// Devices returns none where there is no ImageCaptureCore.
func Devices() []*Device { return nil }

// Device is a camera device, as a camera or phone.
type Device struct {
	Name string
}

// File is a photo or video on a device.
type File struct {
	Name          string
	Size          int64
	Date          time.Time
	Width, Height int
	Movie         bool
	Duration      time.Duration
}

var errNone = errors.New("importing from devices is only on the Mac")

func (d *Device) Open(timeout time.Duration, locked func()) ([]*File, error) { return nil, errNone }
func (d *Device) Close()                                                     {}
func (d *Device) Download(f *File, dir, name string) (string, error) {
	return "", errNone
}

// RunMainLoop runs the given function.
func RunMainLoop(fun func()) { fun() }
