// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cogentcore.org/core/htmlcore"
	"github.com/dop251/goja"
)

// The media elements (video and audio) are not played, but their metadata
// (duration and size) is loaded, as browsers do for preload="metadata",
// which pages wait for to show the controls of their videos (as the
// chapters of the hero video of github.com).

// mediaHead is how much of the start of a media file is read for its
// metadata, and mediaMoovMax is the most of a metadata box at another
// place in the file (the end) that is read.
const (
	mediaHead    = 1 << 20
	mediaMoovMax = 8 << 20
)

// mediaMetadata is the metadata of a media file.
type mediaMetadata struct {
	// Duration is the duration in seconds.
	Duration float64

	// Width and Height are the size of its video track (0 for none).
	Width, Height int
}

// errNoMoov is returned by [parseMP4] when the data does not have the
// metadata box (moov) of an MP4 file, with where it is, if it is known.
type errNoMoov struct {
	offset, size int64
}

func (e *errNoMoov) Error() string { return "no moov box in the media data" }

// parseMP4 returns the metadata of the MP4 (or QuickTime) file whose first
// bytes are the given data, from its moov box. When that box is after the
// data, it returns an *errNoMoov with where it is.
func parseMP4(data []byte) (*mediaMetadata, error) {
	off := int64(0)
	for off+8 <= int64(len(data)) {
		size, typ, hdr := boxHeader(data[off:])
		if size == 0 { // to the end of the file
			size = int64(len(data)) - off
		}
		if size < hdr {
			return nil, errors.New("invalid MP4 box size")
		}
		if typ == "moov" {
			if off+size > int64(len(data)) {
				return nil, &errNoMoov{off, size}
			}
			return parseMoov(data[off+hdr : off+size])
		}
		if off == 0 && typ != "ftyp" {
			return nil, errors.New("not an MP4 file")
		}
		off += size
	}
	return nil, &errNoMoov{offset: off}
}

// boxHeader returns the size, type and header size of the MP4 box at the
// start of the given data (with at least 8 bytes).
func boxHeader(b []byte) (size int64, typ string, hdr int64) {
	size, typ, hdr = int64(binary.BigEndian.Uint32(b)), string(b[4:8]), 8
	if size == 1 && len(b) >= 16 {
		size, hdr = int64(binary.BigEndian.Uint64(b[8:])), 16
	}
	return
}

// parseMoov returns the metadata from the content of a moov box: the
// duration of its mvhd box, and the size of its first video track (tkhd).
func parseMoov(b []byte) (*mediaMetadata, error) {
	md := &mediaMetadata{}
	found := false
	eachBox(b, func(typ string, c []byte) {
		switch typ {
		case "mvhd":
			if d, ok := mvhdDuration(c); ok {
				md.Duration, found = d, true
			}
		case "trak":
			eachBox(c, func(typ string, c []byte) {
				if typ != "tkhd" || md.Width > 0 || len(c) < 8 {
					return
				}
				// the width and height are the last 8 bytes, 16.16 fixed point
				w := binary.BigEndian.Uint32(c[len(c)-8:]) >> 16
				h := binary.BigEndian.Uint32(c[len(c)-4:]) >> 16
				md.Width, md.Height = int(w), int(h)
			})
		}
	})
	if !found {
		return nil, errors.New("no mvhd box in the MP4 moov box")
	}
	return md, nil
}

// eachBox calls the given function with the type and content of each of
// the MP4 boxes in the given data.
func eachBox(b []byte, f func(typ string, content []byte)) {
	for len(b) >= 8 {
		size, typ, hdr := boxHeader(b)
		if size == 0 {
			size = int64(len(b))
		}
		if size < hdr || size > int64(len(b)) {
			return
		}
		f(typ, b[hdr:size])
		b = b[size:]
	}
}

// mvhdDuration returns the duration in seconds of the given content of an
// mvhd box (version 0 or 1).
func mvhdDuration(c []byte) (float64, bool) {
	if len(c) < 20 {
		return 0, false
	}
	var scale uint32
	var dur uint64
	if c[0] == 1 {
		if len(c) < 32 {
			return 0, false
		}
		scale, dur = binary.BigEndian.Uint32(c[20:]), binary.BigEndian.Uint64(c[24:])
	} else {
		scale, dur = binary.BigEndian.Uint32(c[12:]), uint64(binary.BigEndian.Uint32(c[16:]))
	}
	if scale == 0 {
		return 0, false
	}
	return float64(dur) / float64(scale), true
}

// loadMediaMetadata gets the metadata of the media file at the given URL:
// the start of it, and its moov box if that is after the start (and the
// server supports ranges).
func (e *Engine) loadMediaMetadata(u string) (*mediaMetadata, error) {
	get := func(from, n int64) ([]byte, bool, error) {
		resp, err := e.rangeRequest(u, from, n)
		if err != nil {
			return nil, false, err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return nil, false, fmt.Errorf("media: %s", resp.Status)
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, n))
		return data, resp.StatusCode == http.StatusPartialContent, err
	}
	data, ranged, err := get(0, mediaHead)
	if err != nil {
		return nil, err
	}
	md, err := parseMP4(data)
	var nm *errNoMoov
	if errors.As(err, &nm) && ranged && nm.offset > 0 {
		size := nm.size
		if size <= 0 || size > mediaMoovMax {
			size = mediaMoovMax
		}
		box, _, err := get(nm.offset, size)
		if err != nil {
			return nil, err
		}
		if len(box) >= 8 {
			if _, typ, hdr := boxHeader(box); typ == "moov" && int64(len(box)) >= hdr {
				return parseMoov(box[hdr:])
			}
		}
		return nil, errors.New("no moov box in the media file")
	}
	return md, err
}

// rangeRequest gets the given range of bytes of the file at the given URL
// (not with [Options.GetURL], which gets all of it, as for a page), with
// the headers of browsers and the cookies of the page for its origin.
func (e *Engine) rangeRequest(u string, from, n int64) (*http.Response, error) {
	if strings.HasPrefix(u, "blob:") {
		return e.blobResponse(u)
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	htmlcore.SetRequestHeaders(req)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", from, from+n-1))
	client := &http.Client{Timeout: 30 * time.Second}
	if origin(u) == origin(e.opts.PageURL) {
		client.Jar = e.opts.Cookies
	}
	return client.Do(req)
}

// setMediaFuncs sets the D functions for media elements.
func (e *Engine) setMediaFuncs(set func(name string, f func(call goja.FunctionCall) goja.Value)) {
	// mediaMetadata gets the metadata of the media file at the given URL
	// in another goroutine, and calls the callback of the given id with
	// its duration, width and height, and error (null if none).
	set("mediaMetadata", func(call goja.FunctionCall) goja.Value {
		cb := int(call.Argument(0).ToInteger())
		u, err := resolveURL(call.Argument(1).String(), e.baseURL())
		if err != nil {
			e.callBack(cb, 0, 0, 0, err.Error())
			return goja.Undefined()
		}
		go func() {
			md, err := e.loadMediaMetadata(u)
			if err != nil {
				e.callBack(cb, 0, 0, 0, err.Error())
				return
			}
			e.callBack(cb, md.Duration, md.Width, md.Height, nil)
		}()
		return goja.Undefined()
	})
}
