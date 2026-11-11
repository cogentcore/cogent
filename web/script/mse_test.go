// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// appendFile appends the given file to a new buffer of the given media
// source in pieces of the given size, as players append what they fetch.
func appendFile(t *testing.T, ms *MediaSource, file string, piece int) *sourceBuffer {
	data, err := os.ReadFile(file)
	require.NoError(t, err)
	b := ms.addBuffer("video/mp4; codecs=\"avc1.64000a\"")
	for len(data) > 0 {
		n := min(piece, len(data))
		require.NoError(t, b.append(data[:n]))
		data = data[n:]
	}
	return b
}

func TestMSEBuffer(t *testing.T) {
	// ../testdata/frag-video.mp4 is 10 frames at 10 per second, in
	// fragments of 3 (key frame) frames
	ms := newMediaSource()
	b := appendFile(t, ms, "../testdata/frag-video.mp4", 333)
	require.Len(t, b.segments, 4)
	for i, want := range [][2]float64{{0, 0.3}, {0.3, 0.6}, {0.6, 0.9}, {0.9, 1}} {
		assert.InDelta(t, want[0], b.segments[i].start, 0.001, i)
		assert.InDelta(t, want[1], b.segments[i].end, 0.001, i)
	}
	assert.Len(t, b.buffered(), 1)
	assert.InDelta(t, 1.0, b.buffered()[0][1], 0.001)
	assert.NotEmpty(t, b.init)

	// a stream from 0.5 is the init segment and the fragments from the
	// one at it, until the end of the stream
	ms.endOfStream()
	rs, err := ms.Streams(500 * time.Millisecond)
	require.NoError(t, err)
	require.Len(t, rs, 1)
	got, err := io.ReadAll(rs[0])
	require.NoError(t, err)
	want := append(bytes.Clone(b.init), b.segments[1].data...)
	want = append(want, b.segments[2].data...)
	want = append(want, b.segments[3].data...)
	assert.Equal(t, want, got)

	// removing the first two fragments
	b.remove(0, 0.6)
	assert.Len(t, b.segments, 2)
	assert.InDelta(t, 0.6, b.buffered()[0][0], 0.001)

	// the audio: its fragments are of the timescale of its track
	ms2 := newMediaSource()
	a := appendFile(t, ms2, "../testdata/frag-audio.mp4", 1000)
	require.Len(t, a.segments, 4)
	assert.InDelta(t, 1.0, a.segments[3].end, 0.03)
}

func TestMSEStreamWaits(t *testing.T) {
	// A stream waits for the fragments that are not appended yet, and
	// ends when it is closed.
	ms := newMediaSource()
	data, err := os.ReadFile("../testdata/frag-video.mp4")
	require.NoError(t, err)
	b := ms.addBuffer("video/mp4")
	require.NoError(t, b.append(data[:777+28])) // the init segment (ftyp and moov)
	rs, err := ms.Streams(0)
	require.NoError(t, err)
	got := make(chan int)
	go func() {
		n, _ := io.ReadAll(rs[0])
		got <- len(n)
	}()
	time.Sleep(50 * time.Millisecond)
	require.NoError(t, b.append(data[777+28:]))
	time.Sleep(50 * time.Millisecond)
	rs[0].Close()
	select {
	case n := <-got:
		assert.Greater(t, n, 777+28, "the fragments appended after")
	case <-time.After(time.Second):
		t.Fatal("the stream did not end when closed")
	}
	assert.True(t, mseTypeSupported(`video/mp4; codecs="avc1.42E01E"`))
	assert.True(t, mseTypeSupported(`audio/mp4; codecs="mp4a.40.2"`))
	assert.False(t, mseTypeSupported(`video/webm; codecs="vp9"`))
}

func TestMSETimestampOffset(t *testing.T) {
	// The timestampOffset is added to the times of the fragments appended
	// after it is set (in their tfdt boxes, which the decoder reads).
	ms := newMediaSource()
	data, err := os.ReadFile("../testdata/frag-video.mp4")
	require.NoError(t, err)
	b := ms.addBuffer("video/mp4")
	b.offset = 10
	require.NoError(t, b.append(data))
	require.Len(t, b.segments, 4)
	assert.InDelta(t, 10.0, b.segments[0].start, 0.001)
	assert.InDelta(t, 11.0, b.segments[3].end, 0.001)
	// parsed again from the data, as the decoder reads it
	start, _ := b.fragmentTimes(b.segments[1].data[:binary.BigEndian.Uint32(b.segments[1].data)])
	assert.InDelta(t, 10.3, start, 0.001)

	// and back (as -10 for a stream whose times start at 10)
	ms2 := newMediaSource()
	b2 := ms2.addBuffer("video/mp4")
	b2.offset = -0.3
	require.NoError(t, b2.append(data))
	// (the first is not before 0, and the second, now at 0, replaces it)
	require.Len(t, b2.segments, 3)
	assert.InDelta(t, 0.0, b2.segments[0].start, 0.001)
	assert.InDelta(t, 0.3, b2.segments[1].start, 0.001)
}
