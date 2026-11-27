// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
	"strings"

	_ "image/gif"
	_ "image/png"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// ThumbSize is the maximum width and height of the thumbnails of the photos.
const ThumbSize = 256

// isHEIC returns whether the given file is a HEIC (HEIF) file.
func isHEIC(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".heic" || ext == ".heif"
}

// openImage opens the image in the given file, as stored (before its
// orientation): png, jpeg, gif, tiff, bmp, webp, or heic.
func openImage(filename string) (image.Image, error) {
	return openImageMax(filename, 0)
}

// openImageMax opens the image in the given file, as stored (before its
// orientation), scaled down to fit in the given maximum size if it is
// larger and that is faster, as for HEIC on the Mac, or its full size for 0.
func openImageMax(filename string, maxSize int) (image.Image, error) {
	if isHEIC(filename) {
		if runtime.GOOS == "darwin" {
			return openSips(filename, maxSize)
		}
		return decodeHEIC(filename)
	}
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(bufio.NewReader(f))
	return img, err
}

// openSips opens the given image with the sips command of the Mac, which
// uses the hardware decoder for HEIC, much faster than goheif, scaled
// down to fit in the given maximum size, unless 0.
func openSips(filename string, maxSize int) (image.Image, error) {
	tmp, err := os.CreateTemp("", "cogentphotos-*.jpg")
	if err != nil {
		return nil, err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())
	args := []string{"-s", "format", "jpeg", "-s", "formatOptions", "90"}
	if maxSize > 0 {
		args = append(args, "-Z", strconv.Itoa(maxSize))
	}
	args = append(args, filename, "--out", tmp.Name())
	if out, err := exec.Command("sips", args...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("sips: %w: %s", err, out)
	}
	return openThumb(tmp.Name())
}

// imageSize returns the size of the image in the given file, from its
// header, or zero if it cannot be read.
func imageSize(filename string) image.Point {
	if isHEIC(filename) {
		sz, _ := heicSize(filename)
		return sz
	}
	f, err := os.Open(filename)
	if err != nil {
		return image.Point{}
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(bufio.NewReader(f))
	if err != nil {
		return image.Point{}
	}
	return image.Pt(cfg.Width, cfg.Height)
}

// openPhoto opens the image of the given photo in the given file,
// upright: with its orientation and rotation applied.
// The image is scaled down to fit in the given maximum size if that is
// faster (see [openImageMax]).
func openPhoto(ph *Photo, filename string, maxSize int) (image.Image, error) {
	img, err := openImageMax(filename, maxSize)
	if err != nil {
		return nil, err
	}
	return orientImage(img, ph.Orient, ph.Rotation), nil
}

// orientImage returns the given image turned upright for the given EXIF
// orientation, and then rotated clockwise by the given degrees.
func orientImage(img image.Image, o Orientations, rotation int) image.Image {
	// the EXIF orientation is a transform; a rotation is one too:
	// 90 = 6, 180 = 3, 270 = 8, applied after.
	img = orientExif(img, int(o))
	switch rotation {
	case 90:
		img = orientExif(img, 6)
	case 180:
		img = orientExif(img, 3)
	case 270:
		img = orientExif(img, 8)
	}
	return img
}

// orientExif returns the given image turned upright for the given EXIF
// orientation: 2 mirrored, 3 turned 180°, 4 flipped, 5 transposed,
// 6 turned 90° clockwise, 7 transversed, and 8 turned 90° counterclockwise.
func orientExif(im image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return im
	}
	src := asRGBA(im)
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := range h {
		si := src.PixOffset(b.Min.X, b.Min.Y+y)
		for x := range w {
			var dx, dy int
			switch o {
			case 2:
				dx, dy = w-1-x, y
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dx, dy = x, h-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}
			di := dst.PixOffset(dx, dy)
			copy(dst.Pix[di:di+4], src.Pix[si:si+4])
			si += 4
		}
	}
	return dst
}

// asRGBA returns the given image as an [image.RGBA], converting it if needed.
func asRGBA(im image.Image) *image.RGBA {
	if rgba, ok := im.(*image.RGBA); ok {
		return rgba
	}
	b := im.Bounds()
	rgba := image.NewRGBA(b)
	draw.Draw(rgba, b, im, b.Min, draw.Src)
	return rgba
}

// scaleToFit returns the given image scaled down to fit in the given
// maximum width and height, keeping its aspect ratio, or itself if it fits.
func scaleToFit(img image.Image, maxSize int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxSize && h <= maxSize {
		return img
	}
	var nw, nh int
	if w >= h {
		nw, nh = maxSize, max(1, h*maxSize/w)
	} else {
		nw, nh = max(1, w*maxSize/h), maxSize
	}
	// halve with a box filter first, which is fast, down to twice the size
	for w >= 4*nw && h >= 4*nh {
		img = halve(img)
		b = img.Bounds()
		w, h = b.Dx(), b.Dy()
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	return dst
}

// halve returns the given image at half of its size, by averaging each
// 2x2 block of pixels.
func halve(im image.Image) *image.RGBA {
	src := asRGBA(im)
	b := src.Bounds()
	w, h := b.Dx()/2, b.Dy()/2
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		s0 := src.PixOffset(b.Min.X, b.Min.Y+2*y)
		s1 := s0 + src.Stride
		di := dst.PixOffset(0, y)
		for range w {
			for c := range 4 {
				sum := int(src.Pix[s0+c]) + int(src.Pix[s0+4+c]) + int(src.Pix[s1+c]) + int(src.Pix[s1+4+c])
				dst.Pix[di+c] = uint8(sum / 4)
			}
			s0 += 8
			s1 += 8
			di += 4
		}
	}
	return dst
}

// makeThumb makes the thumbnail of the given photo, in the given file,
// saving it as a JPEG file in the given file of the thumbnail.
func makeThumb(ph *Photo, filename, thumbFile string) (image.Image, error) {
	img, err := openImageMax(filename, ThumbSize)
	if err != nil {
		return nil, err
	}
	if ph.Size == (image.Point{}) {
		ph.Size = imageSize(filename)
	}
	img = scaleToFit(img, ThumbSize)
	img = orientImage(img, ph.Orient, ph.Rotation)
	img = drawDate(img, ph.DateTaken)
	if err := os.MkdirAll(filepath.Dir(thumbFile), 0o755); err != nil {
		return img, err
	}
	tmp := thumbFile + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return img, err
	}
	err = jpeg.Encode(f, img, &jpeg.Options{Quality: 85})
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return img, err
	}
	return img, os.Rename(tmp, thumbFile)
}

// openThumb opens the thumbnail in the given file.
func openThumb(thumbFile string) (image.Image, error) {
	f, err := os.Open(thumbFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return jpeg.Decode(bufio.NewReader(f))
}

// thumbFace is the font face of the dates drawn on the thumbnails.
var thumbFace = sync.OnceValue(func() font.Face {
	f, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: 12, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil
	}
	return face
})

// drawDate draws the given date at the top left of the given thumbnail,
// in white or black, as is darker or lighter there, as GoPix did.
func drawDate(img image.Image, date time.Time) image.Image {
	face := thumbFace()
	if face == nil || date.IsZero() {
		return img
	}
	rgba := asRGBA(img)
	b := rgba.Bounds()
	clr := color.Black
	if averageGrey(rgba, image.Rect(b.Min.X+5, b.Min.Y+5, b.Min.X+100, b.Min.Y+25)) < 0.5 {
		clr = color.White
	}
	d := &font.Drawer{Dst: rgba, Src: image.NewUniform(clr), Face: face,
		Dot: fixed.P(b.Min.X+5, b.Min.Y+5+face.Metrics().Ascent.Ceil())}
	d.DrawString(date.Format("2006:01:02"))
	return rgba
}

// averageGrey returns the average brightness of the given region of the
// given image, from 0 to 1.
func averageGrey(img *image.RGBA, r image.Rectangle) float32 {
	r = r.Intersect(img.Bounds())
	sum, n := 0, 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		i := img.PixOffset(r.Min.X, y)
		for x := r.Min.X; x < r.Max.X; x++ {
			sum += int(img.Pix[i]) + int(img.Pix[i+1]) + int(img.Pix[i+2])
			n += 3
			i += 4
		}
	}
	if n == 0 {
		return 0
	}
	return float32(sum) / float32(n) / 255
}
