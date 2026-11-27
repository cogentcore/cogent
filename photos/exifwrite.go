// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
	jpegstructure "github.com/dsoprea/go-jpeg-image-structure/v2"
)

// The dates and orientation of photos are written into the EXIF
// metadata of their JPEG files, so that other apps, such as Google
// Photos, see the dates set here, as for scans of old photos. Tags that
// are already in the file are changed in place, which changes nothing
// else in it; otherwise the metadata is made again with go-exif. Each
// file is written to a temporary file, which is checked before it
// replaces the file.

// isJPEG returns whether the given file is a JPEG file, by its extension.
func isJPEG(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".jpg" || ext == ".jpeg"
}

// exifDate returns the given date in the format of EXIF.
func exifDate(t time.Time) string {
	return t.Format("2006:01:02 15:04:05")
}

// composeOrient returns the EXIF orientation that is the given
// orientation followed by the given rotation in degrees clockwise.
func composeOrient(o Orientations, rotation int) Orientations {
	if o == NoOrient || o >= OrientUndef {
		o = Rotated0
	}
	if rotation%360 == 0 {
		return o
	}
	// a small image with all different pixels shows the transform
	src := image.NewRGBA(image.Rect(0, 0, 3, 2))
	for i := range src.Pix {
		src.Pix[i] = uint8(i)
	}
	want := asRGBA(orientImage(src, o, rotation))
	for c := Rotated0; c < OrientUndef; c++ {
		got := asRGBA(orientExif(src, int(c)))
		if got.Bounds() == want.Bounds() && bytes.Equal(got.Pix, want.Pix) {
			return c
		}
	}
	return o
}

// needsExif returns whether the metadata in the file of the photo does
// not have its date and orientation, given those read from the file.
func (ph *Photo) needsExif(file *Photo) bool {
	if !file.DateTaken.Equal(ph.DateTaken) || ph.Rotation%360 != 0 {
		return true
	}
	o := ph.Orient
	if o == NoOrient {
		o = Rotated0
	}
	fo := file.Orient
	if fo == NoOrient {
		fo = Rotated0
	}
	return o != fo
}

// fileExif returns the date and orientation in the metadata of the given
// file, as a Photo, with no date if it has none.
func fileExif(filename string) *Photo {
	fp := &Photo{DateSet: true} // no date from the file time
	if raw, err := readRawExif(filename); err == nil {
		fp.parseExif(raw, filename)
	}
	return fp
}

// writeExif writes the date of the given photo, and its orientation with
// its rotation, into the metadata of its JPEG file. After this, the
// rotation of the photo is 0, in its orientation. It does nothing if the
// file already has them.
func writeExif(ph *Photo, filename string) error {
	if !isJPEG(filename) {
		return fmt.Errorf("%s: only the metadata of JPEG files can be written", filepath.Base(filename))
	}
	if !ph.needsExif(fileExif(filename)) {
		return nil
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	orient := composeOrient(ph.Orient, ph.Rotation)
	date := exifDate(ph.DateTaken)
	out, ok := patchExif(data, date, uint16(orient))
	if !ok {
		out, err = rebuildExif(data, date, uint16(orient))
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(filename), err)
		}
	}
	if err := replaceChecked(filename, out, ph.DateTaken, orient); err != nil {
		return err
	}
	ph.Orient, ph.Rotation = orient, 0
	if st, err := os.Stat(filename); err == nil {
		ph.FileMod = st.ModTime()
	}
	return nil
}

// replaceChecked replaces the given file with the given data, after
// checking that it is a JPEG image of the same size as the file, with the
// given date and orientation in its metadata.
func replaceChecked(filename string, data []byte, date time.Time, orient Orientations) error {
	base := filepath.Base(filename)
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("%s: the written image is not valid: %w", base, err)
	}
	if old := imageSize(filename); old != (image.Point{}) && old != image.Pt(cfg.Width, cfg.Height) {
		return fmt.Errorf("%s: the written image has another size", base)
	}
	tmp := filename + ".tmp-exif"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	fp := &Photo{DateSet: true}
	if raw, err := readRawExif(tmp); err == nil {
		fp.parseExif(raw, tmp)
	}
	fo := fp.Orient
	if fo == NoOrient {
		fo = Rotated0
	}
	if !fp.DateTaken.Equal(date) || fo != orient {
		os.Remove(tmp)
		return fmt.Errorf("%s: the written metadata has date %v and orientation %v, instead of %v and %v", base, fp.DateTaken, fo, date, orient)
	}
	if st, err := os.Stat(filename); err == nil {
		os.Chmod(tmp, st.Mode())
	}
	return os.Rename(tmp, filename)
}

// jpegExifSegment returns the start and end of the TIFF data of the EXIF
// metadata in the given JPEG data, after its Exif header.
func jpegExifSegment(b []byte) (start, end int, ok bool) {
	if len(b) < 4 || b[0] != 0xFF || b[1] != 0xD8 {
		return 0, 0, false
	}
	i := 2
	for i+4 <= len(b) {
		if b[i] != 0xFF {
			return 0, 0, false
		}
		marker := b[i+1]
		if marker == 0xFF { // padding
			i++
			continue
		}
		if marker == 0xD8 || (marker >= 0xD0 && marker <= 0xD7) || marker == 0x01 {
			i += 2
			continue
		}
		if marker == 0xDA || marker == 0xD9 {
			return 0, 0, false
		}
		n := int(binary.BigEndian.Uint16(b[i+2:]))
		end := i + 2 + n
		if n < 2 || end > len(b) {
			return 0, 0, false
		}
		seg := b[i+4 : end]
		if marker == 0xE1 && bytes.HasPrefix(seg, []byte("Exif\x00\x00")) {
			return i + 4 + 6, end, true
		}
		i = end
	}
	return 0, 0, false
}

// patchExif changes the dates and the orientation in the EXIF metadata
// of the given JPEG data in place, returning a copy, if they are all
// there to change: DateTimeOriginal is needed, and Orientation unless it
// is upright. The other dates are changed if they are there.
func patchExif(data []byte, date string, orient uint16) ([]byte, bool) {
	start, end, ok := jpegExifSegment(data)
	if !ok {
		return nil, false
	}
	out := bytes.Clone(data)
	t := out[start:end]
	if len(t) < 8 {
		return nil, false
	}
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return nil, false
	}
	// entry returns the offset in t of each 12-byte entry of the IFD at off
	entries := func(off int) []int {
		if off < 8 || off+2 > len(t) {
			return nil
		}
		n := int(bo.Uint16(t[off:]))
		var res []int
		for k := range n {
			e := off + 2 + 12*k
			if e+12 > len(t) {
				return res
			}
			res = append(res, e)
		}
		return res
	}
	// setASCII sets the 20-byte date of the ASCII entry at e, if it is one
	setASCII := func(e int) bool {
		if bo.Uint16(t[e+2:]) != 2 || bo.Uint32(t[e+4:]) != 20 {
			return false
		}
		off := int(bo.Uint32(t[e+8:]))
		if off+20 > len(t) {
			return false
		}
		copy(t[off:off+19], date)
		t[off+19] = 0
		return true
	}
	gotOrig, gotOrient := false, orient == 1
	exifIFD := 0
	for _, e := range entries(int(bo.Uint32(t[4:]))) {
		switch bo.Uint16(t[e:]) {
		case 0x0132: // DateTime
			setASCII(e)
		case 0x0112: // Orientation, a SHORT
			if bo.Uint16(t[e+2:]) == 3 && bo.Uint32(t[e+4:]) == 1 {
				bo.PutUint16(t[e+8:], orient)
				gotOrient = true
			}
		case 0x8769: // the Exif IFD
			exifIFD = int(bo.Uint32(t[e+8:]))
		}
	}
	for _, e := range entries(exifIFD) {
		switch bo.Uint16(t[e:]) {
		case 0x9003: // DateTimeOriginal
			gotOrig = setASCII(e)
		case 0x9004: // DateTimeDigitized
			setASCII(e)
		}
	}
	return out, gotOrig && gotOrient
}

// rebuildExif makes the EXIF metadata of the given JPEG data again, with
// the given dates and orientation, keeping the rest of it, or new
// metadata if it has none.
func rebuildExif(data []byte, date string, orient uint16) (out []byte, err error) {
	defer func() { // go-exif panics on some errors
		if r := recover(); r != nil {
			err = fmt.Errorf("writing the metadata: %v", r)
		}
	}()
	jmp := jpegstructure.NewJpegMediaParser()
	intfc, err := jmp.ParseBytes(data)
	if err != nil {
		return nil, err
	}
	sl := intfc.(*jpegstructure.SegmentList)
	rootIb, err := sl.ConstructExifBuilder()
	if err != nil {
		if !errors.Is(err, exif.ErrNoExif) && !strings.Contains(err.Error(), "no exif") {
			return nil, err
		}
		im, err := exifcommon.NewIfdMappingWithStandard()
		if err != nil {
			return nil, err
		}
		rootIb = exif.NewIfdBuilder(im, exif.NewTagIndex(), exifcommon.IfdStandardIfdIdentity, exifcommon.EncodeDefaultByteOrder)
	}
	if err := rootIb.SetStandardWithName("DateTime", date); err != nil {
		return nil, err
	}
	if err := rootIb.SetStandardWithName("Orientation", []uint16{orient}); err != nil {
		return nil, err
	}
	exifIb, err := exif.GetOrCreateIbFromRootIb(rootIb, "IFD/Exif")
	if err != nil {
		return nil, err
	}
	for _, name := range []string{"DateTimeOriginal", "DateTimeDigitized"} {
		if err := exifIb.SetStandardWithName(name, date); err != nil {
			return nil, err
		}
	}
	if err := sl.SetExif(rootIb); err != nil {
		return nil, err
	}
	var b bytes.Buffer
	if err := sl.Write(&b); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
