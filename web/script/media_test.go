// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mp4Box returns an MP4 box of the given type and content.
func mp4Box(typ string, content ...[]byte) []byte {
	c := bytes.Join(content, nil)
	b := binary.BigEndian.AppendUint32(nil, uint32(8+len(c)))
	return append(append(b, typ...), c...)
}

// testMP4 returns a small MP4 file with the given duration (in units of a
// timescale of 1000) and video size, with its moov box before or after a
// media data (mdat) box of the given size.
func testMP4(ms uint32, w, h uint16, moovLast bool, mdat int) []byte {
	mvhd := make([]byte, 100) // version 0
	binary.BigEndian.PutUint32(mvhd[12:], 1000)
	binary.BigEndian.PutUint32(mvhd[16:], ms)
	tkhd := make([]byte, 84)
	binary.BigEndian.PutUint32(tkhd[76:], uint32(w)<<16)
	binary.BigEndian.PutUint32(tkhd[80:], uint32(h)<<16)
	moov := mp4Box("moov", mp4Box("mvhd", mvhd), mp4Box("trak", mp4Box("tkhd", tkhd)))
	ftyp := mp4Box("ftyp", []byte("isom\x00\x00\x02\x00isom"))
	data := mp4Box("mdat", make([]byte, mdat))
	if moovLast {
		return bytes.Join([][]byte{ftyp, data, moov}, nil)
	}
	return bytes.Join([][]byte{ftyp, moov, data}, nil)
}

func TestParseMP4(t *testing.T) {
	md, err := parseMP4(testMP4(37700, 2560, 1726, false, 100))
	require.NoError(t, err)
	assert.Equal(t, mediaMetadata{Duration: 37.7, Width: 2560, Height: 1726}, *md)

	// the moov box after the media data: where it is
	f := testMP4(5000, 640, 480, true, 100)
	_, err = parseMP4(f[:200])
	var nm *errNoMoov
	require.True(t, errors.As(err, &nm))
	assert.Equal(t, int64(len(f))-nm.size, nm.offset, "at the end")

	_, err = parseMP4([]byte("<html>not a video</html>"))
	assert.Error(t, err)
}

// TestMediaMetadata tests that a video element loads the metadata of its
// source (served with ranges, its moov box at the start or at the end),
// as pages wait for (as the chapters of the video of github.com), and that
// its play() is rejected.
func TestMediaMetadata(t *testing.T) {
	files := map[string][]byte{
		"/start.mp4": testMP4(37700, 2560, 1726, false, 2*mediaHead),
		"/end.mp4":   testMP4(5000, 640, 480, true, 2*mediaHead),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, r.URL.Path, time.Time{}, bytes.NewReader(files[r.URL.Path]))
	}))
	defer srv.Close()
	e, _ := testEngine(t, `<html><body><video id="v" preload="metadata"></video><video id="w"><source src="`+srv.URL+`/end.mp4"></video></body></html>`, nil)
	run(t, e, `
		window.got = [];
		const v = document.getElementById("v");
		v.addEventListener("loadedmetadata", () => got.push([v.readyState, v.duration, v.videoWidth, v.videoHeight].join(" ")));
		v.src = "`+srv.URL+`/start.mp4";
		const w = document.getElementById("w");
		w.addEventListener("loadedmetadata", () => got.push([w.readyState >= w.HAVE_METADATA, w.duration, w.videoWidth].join(" ")));
		window.played = "";
		v.play().catch(e => { played = e.name; });
	`)
	assert.Eventually(t, func() bool {
		v, _ := e.RunString(`got.slice().sort().join(", ") + " " + played`)
		return v == "1 37.7 2560 1726, true 5 640 NotAllowedError"
	}, 5*time.Second, 20*time.Millisecond)
	assert.Equal(t, int64(0), run(t, e, `document.createElement("video").readyState`), "no source")
}
