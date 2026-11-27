// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !darwin

package photos

import (
	"bufio"
	"image"
	"os"

	"github.com/jdeng/goheif"
)

// HEIC files are decoded with goheif (with its C decoder, libde265),
// except on the Mac, which has its own (see heic_darwin.go).

// decodeHEIC returns the image in the given HEIC file.
func decodeHEIC(filename string) (image.Image, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return goheif.Decode(bufio.NewReader(f))
}

// heicSize returns the size of the image in the given HEIC file.
func heicSize(filename string) (image.Point, error) {
	f, err := os.Open(filename)
	if err != nil {
		return image.Point{}, err
	}
	defer f.Close()
	cfg, err := goheif.DecodeConfig(bufio.NewReader(f))
	return image.Pt(cfg.Width, cfg.Height), err
}

// heicExif returns the EXIF metadata of the given HEIC file, or nil if
// it is not found, for searching for it in the file.
func heicExif(f *os.File) []byte {
	raw, err := goheif.ExtractExif(f)
	if err != nil {
		return nil
	}
	return raw
}
