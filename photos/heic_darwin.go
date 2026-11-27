// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"fmt"
	"image"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// On the Mac, HEIC files are decoded with sips, which uses its hardware
// decoder, much faster than goheif, which is not built here, with its C
// decoder (see heic_goheif.go).

// decodeHEIC returns the image in the given HEIC file.
func decodeHEIC(filename string) (image.Image, error) {
	return openSips(filename, 0)
}

// heicSize returns the size of the image in the given HEIC file.
func heicSize(filename string) (image.Point, error) {
	out, err := exec.Command("sips", "-g", "pixelWidth", "-g", "pixelHeight", filename).Output()
	if err != nil {
		return image.Point{}, err
	}
	var sz image.Point
	for _, line := range strings.Split(string(out), "\n") {
		name, val, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		v, _ := strconv.Atoi(strings.TrimSpace(val))
		switch name {
		case "pixelWidth":
			sz.X = v
		case "pixelHeight":
			sz.Y = v
		}
	}
	if sz.X == 0 || sz.Y == 0 {
		return sz, fmt.Errorf("no size of %s from sips", filename)
	}
	return sz, nil
}

// heicExif returns nil, for searching for the EXIF metadata in the file,
// which finds it in HEIC files.
func heicExif(f *os.File) []byte {
	return nil
}
