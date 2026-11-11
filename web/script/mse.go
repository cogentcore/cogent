// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/dop251/goja"
)

// Media Source Extensions (MSE): the players of web pages (as hls.js,
// dash.js, and those of YouTube) fetch the media themselves, and append it
// to the SourceBuffers of a MediaSource, whose object URL is the source of
// a video element. A [MediaSource] keeps the media appended to its
// buffers, as fragmented MP4 (an init segment, then fragments of moof and
// mdat boxes, whose times are parsed for their buffered ranges), and is the
// source of the media of the video for the player of the page
// (video.Source): its streams, one for each buffer (as one for the video
// and one for the audio), are its init segment and its fragments in the
// order of their times, from a time, which wait for the fragments that
// are not appended yet, until the end of the stream. WebM is not supported
// (isTypeSupported is false for it, so that players use MP4).

// MediaSource is a MediaSource of a page (Media Source Extensions), the
// source of the media of a video element whose source is its object URL.
type MediaSource struct {
	mu   sync.Mutex
	cond *sync.Cond

	buffers  []*sourceBuffer
	duration float64 // in seconds, NaN if not set
	ended    bool
}

// sourceBuffer is a SourceBuffer of a [MediaSource], of the media of one
// type (as video/mp4), appended as fragmented MP4.
type sourceBuffer struct {
	src  *MediaSource
	mime string

	// pending are the bytes appended that are not a whole box yet.
	pending []byte

	// init is the init segment (ftyp and moov boxes), and inits counts
	// them, for the segments of each.
	init  []byte
	inits int

	// timescales and durations are the timescales and default sample
	// durations of the tracks of the init segment, by track ID.
	timescales, durations map[uint32]uint32

	// segments are the fragments (moof and mdat boxes) appended, in the
	// order of their times, and cur the one that is being appended (its
	// moof, before its mdat).
	segments []*segment
	cur      *segment

	// offset is the timestampOffset of the buffer, in seconds, which is
	// added to the times of the fragments appended (in their tfdt boxes).
	offset float64
}

// segment is a fragment of a [sourceBuffer] (a moof and its mdat).
type segment struct {
	start, end float64 // in seconds
	data       []byte
	init       int // the number of the init segment that it is of
}

// newMediaSource returns a new [MediaSource].
func newMediaSource() *MediaSource {
	ms := &MediaSource{duration: math.NaN()}
	ms.cond = sync.NewCond(&ms.mu)
	return ms
}

// Duration returns the duration of the media (as the duration of the
// MediaSource set by the page), or 0 if it is not known.
func (ms *MediaSource) Duration() time.Duration {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if math.IsNaN(ms.duration) || math.IsInf(ms.duration, 0) {
		return 0
	}
	return time.Duration(ms.duration * float64(time.Second))
}

// Streams returns readers of the streams of the media of the buffers,
// from the given time (see video.Source), once each buffer has its init
// segment.
func (ms *MediaSource) Streams(start time.Duration) ([]io.ReadCloser, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	// (the buffers are added before their media is appended)
	for len(ms.buffers) == 0 || slices.ContainsFunc(ms.buffers, func(b *sourceBuffer) bool { return b.init == nil }) {
		if ms.ended {
			return nil, errors.New("no media in the MediaSource")
		}
		ms.cond.Wait()
	}
	rs := make([]io.ReadCloser, len(ms.buffers))
	for i, b := range ms.buffers {
		rs[i] = &streamReader{buf: b, start: start.Seconds()}
	}
	return rs, nil
}

// endOfStream ends the stream of the media.
func (ms *MediaSource) endOfStream() {
	ms.mu.Lock()
	ms.ended = true
	ms.mu.Unlock()
	ms.cond.Broadcast()
}

// addBuffer adds a new buffer of the given type.
func (ms *MediaSource) addBuffer(mime string) *sourceBuffer {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	b := &sourceBuffer{src: ms, mime: mime, timescales: map[uint32]uint32{}, durations: map[uint32]uint32{}}
	ms.buffers = append(ms.buffers, b)
	ms.ended = false
	return b
}

// removeBuffer removes the given buffer.
func (ms *MediaSource) removeBuffer(b *sourceBuffer) {
	ms.mu.Lock()
	ms.buffers = slices.DeleteFunc(ms.buffers, func(x *sourceBuffer) bool { return x == b })
	ms.mu.Unlock()
	ms.cond.Broadcast()
}

// append appends the given data to the buffer: whole boxes of the init
// segment and of fragments, keeping the rest for the next.
func (b *sourceBuffer) append(data []byte) error {
	ms := b.src
	ms.mu.Lock()
	defer func() {
		ms.mu.Unlock()
		ms.cond.Broadcast()
	}()
	b.pending = append(b.pending, data...)
	for {
		size, typ, ok := topBox(b.pending)
		if !ok || int64(len(b.pending)) < size {
			return nil // (the rest of the box is appended later)
		}
		box := bytes.Clone(b.pending[:size])
		b.pending = b.pending[size:]
		switch typ {
		case "ftyp":
			b.init = box
			b.inits++
		case "moov":
			b.init = append(b.init, box...)
			b.parseInit(box)
		case "moof":
			if b.offset != 0 {
				b.offsetFragment(box)
			}
			seg := &segment{data: box, init: b.inits}
			seg.start, seg.end = b.fragmentTimes(box)
			b.cur = seg
		case "mdat":
			if b.cur == nil {
				continue
			}
			b.cur.data = append(b.cur.data, box...)
			b.addSegment(b.cur)
			b.cur = nil
		} // (styp, sidx, emsg, prft, free... are not needed)
	}
}

// topBox returns the size and type of the box at the start of the given
// data, and whether its header is there.
func topBox(b []byte) (size int64, typ string, ok bool) {
	if len(b) < 8 {
		return 0, "", false
	}
	size, typ = int64(binary.BigEndian.Uint32(b)), string(b[4:8])
	switch size {
	case 1: // 64-bit size
		if len(b) < 16 {
			return 0, "", false
		}
		size = int64(binary.BigEndian.Uint64(b[8:]))
	case 0: // to the end
		size = int64(len(b))
	}
	if size < 8 {
		size = 8 // (invalid: skipped)
	}
	return size, typ, true
}

// addSegment adds the given segment in the order of times, replacing the
// segments that it overlaps (as appended again after a seek, or of
// another quality).
func (b *sourceBuffer) addSegment(seg *segment) {
	const eps = 0.001
	b.segments = slices.DeleteFunc(b.segments, func(s *segment) bool {
		return s.start < seg.end-eps && seg.start < s.end-eps
	})
	i := sort.Search(len(b.segments), func(i int) bool { return b.segments[i].start >= seg.start })
	b.segments = slices.Insert(b.segments, i, seg)
}

// remove removes the segments within the given range of times.
func (b *sourceBuffer) remove(start, end float64) {
	ms := b.src
	ms.mu.Lock()
	defer ms.mu.Unlock()
	b.segments = slices.DeleteFunc(b.segments, func(s *segment) bool {
		return s.start >= start-0.001 && s.end <= end+0.001
	})
}

// abort drops the bytes appended that are not a whole box yet.
func (b *sourceBuffer) abort() {
	ms := b.src
	ms.mu.Lock()
	defer ms.mu.Unlock()
	b.pending, b.cur = nil, nil
}

// buffered returns the ranges of the times of the segments (merging those
// that are less than a tenth of a second apart).
func (b *sourceBuffer) buffered() [][2]float64 {
	ms := b.src
	ms.mu.Lock()
	defer ms.mu.Unlock()
	var rs [][2]float64
	for _, s := range b.segments {
		if n := len(rs); n > 0 && s.start <= rs[n-1][1]+0.1 {
			rs[n-1][1] = max(rs[n-1][1], s.end)
			continue
		}
		rs = append(rs, [2]float64{s.start, s.end})
	}
	return rs
}

// streamReader reads a stream of a [sourceBuffer]: its init segment, then
// its segments in the order of their times, from the one at its start time,
// waiting for those that are not appended yet, until the end of the stream.
type streamReader struct {
	buf   *sourceBuffer
	start float64

	// data is what is left to be read of the init segment or segment that
	// is being read, and last the segment that was read last.
	data     []byte
	sentInit bool
	last     *segment
	closed   bool
}

func (r *streamReader) Read(p []byte) (int, error) {
	ms := r.buf.src
	ms.mu.Lock()
	defer ms.mu.Unlock()
	for len(r.data) == 0 {
		if r.closed {
			return 0, io.EOF
		}
		if !r.sentInit {
			r.data, r.sentInit = r.buf.init, true
			continue
		}
		if seg := r.next(); seg != nil {
			r.data, r.last = seg.data, seg
			continue
		}
		if ms.ended {
			return 0, io.EOF
		}
		ms.cond.Wait()
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

// next returns the next segment to read, if it is appended: first the one
// at the start time (or the first after it), then the one after the last
// one read, in the order of their times.
func (r *streamReader) next() *segment {
	segs := r.buf.segments
	if r.last == nil {
		for _, s := range segs {
			if s.end > r.start+0.001 { // (the one at it, or after it)
				return s
			}
		}
		return nil
	}
	for _, s := range segs {
		if s.start > r.last.start+0.001 {
			return s
		}
	}
	return nil
}

func (r *streamReader) Close() error {
	ms := r.buf.src
	ms.mu.Lock()
	r.closed = true
	ms.mu.Unlock()
	ms.cond.Broadcast()
	return nil
}

// parseInit parses the timescales and default sample durations of the
// tracks of the given moov box.
func (b *sourceBuffer) parseInit(moov []byte) {
	forBoxes(moov[8:], func(typ string, body []byte) {
		switch typ {
		case "trak":
			var id uint32
			forBoxes(body, func(typ string, body []byte) {
				switch typ {
				case "tkhd":
					if len(body) >= 24 {
						if body[0] == 1 {
							id = binary.BigEndian.Uint32(body[20:])
						} else {
							id = binary.BigEndian.Uint32(body[12:])
						}
					}
				case "mdia":
					forBoxes(body, func(typ string, body []byte) {
						if typ == "mdhd" && len(body) >= 24 {
							if body[0] == 1 {
								b.timescales[id] = binary.BigEndian.Uint32(body[20:])
							} else {
								b.timescales[id] = binary.BigEndian.Uint32(body[12:])
							}
						}
					})
				}
			})
		case "mvex":
			forBoxes(body, func(typ string, body []byte) {
				if typ == "trex" && len(body) >= 16 {
					b.durations[binary.BigEndian.Uint32(body[4:])] = binary.BigEndian.Uint32(body[12:])
				}
			})
		}
	})
}

// offsetFragment adds the timestampOffset of the buffer to the base media
// decode times (tfdt) of the tracks of the given moof box, in place (as
// players set it to start the times of a stream at 0, as hls.js does for
// one whose times start at 10 seconds).
func (b *sourceBuffer) offsetFragment(moof []byte) {
	forBoxes(moof[8:], func(typ string, body []byte) {
		if typ != "traf" {
			return
		}
		var id uint32
		forBoxes(body, func(typ string, body []byte) {
			switch {
			case typ == "tfhd" && len(body) >= 8:
				id = binary.BigEndian.Uint32(body[4:])
			case typ == "tfdt" && len(body) >= 8:
				ts := float64(b.timescales[id])
				if ts <= 0 {
					ts = 90000
				}
				d := int64(math.Round(b.offset * ts))
				if body[0] == 1 && len(body) >= 12 {
					v := max(int64(binary.BigEndian.Uint64(body[4:]))+d, 0)
					binary.BigEndian.PutUint64(body[4:], uint64(v))
				} else {
					v := max(int64(binary.BigEndian.Uint32(body[4:]))+d, 0)
					binary.BigEndian.PutUint32(body[4:], uint32(min(v, math.MaxUint32)))
				}
			}
		})
	})
}

// fragmentTimes returns the start and end times in seconds of the given
// moof box, of its first track: its base media decode time (tfdt), and the
// durations of its samples (trun, or the defaults of tfhd and trex).
func (b *sourceBuffer) fragmentTimes(moof []byte) (start, end float64) {
	done := false
	forBoxes(moof[8:], func(typ string, body []byte) {
		if typ != "traf" || done {
			return
		}
		done = true
		var id, defDur uint32
		var base uint64
		var total uint64
		forBoxes(body, func(typ string, body []byte) {
			if len(body) < 8 {
				return
			}
			flags := binary.BigEndian.Uint32(body) & 0xffffff
			switch typ {
			case "tfhd":
				id = binary.BigEndian.Uint32(body[4:])
				defDur = b.durations[id]
				off := 8
				if flags&0x01 != 0 {
					off += 8 // base data offset
				}
				if flags&0x02 != 0 {
					off += 4 // sample description index
				}
				if flags&0x08 != 0 && len(body) >= off+4 {
					defDur = binary.BigEndian.Uint32(body[off:])
				}
			case "tfdt":
				if body[0] == 1 && len(body) >= 12 {
					base = binary.BigEndian.Uint64(body[4:])
				} else {
					base = uint64(binary.BigEndian.Uint32(body[4:]))
				}
			case "trun":
				count := binary.BigEndian.Uint32(body[4:])
				off := 8
				if flags&0x01 != 0 {
					off += 4 // data offset
				}
				if flags&0x04 != 0 {
					off += 4 // first sample flags
				}
				per := 0
				for _, f := range []uint32{0x100, 0x200, 0x400, 0x800} {
					if flags&f != 0 {
						per += 4
					}
				}
				for i := uint32(0); i < count; i++ {
					d := uint64(defDur)
					if flags&0x100 != 0 && len(body) >= off+4 {
						d = uint64(binary.BigEndian.Uint32(body[off:]))
					}
					total += d
					off += per
				}
			}
		})
		ts := float64(b.timescales[id])
		if ts <= 0 {
			ts = 90000
		}
		start = float64(base) / ts
		end = start + float64(total)/ts
	})
	return start, end
}

// forBoxes calls the given function for each box in the given data
// (the body of a box), with its type and body.
func forBoxes(data []byte, f func(typ string, body []byte)) {
	for len(data) >= 8 {
		size, typ, ok := topBox(data)
		if !ok || size > int64(len(data)) {
			return
		}
		hdr := int64(8)
		if binary.BigEndian.Uint32(data) == 1 {
			hdr = 16
		}
		f(typ, data[hdr:size])
		data = data[size:]
	}
}

// mediaTypeMP4 matches the types of MP4 media (with any codecs).
var mediaTypeMP4 = regexp.MustCompile(`(?i)^\s*(video|audio)/mp4\b`)

// closeMediaSources ends the streams of the MediaSources of the page, as
// it is closed, so that its videos do not wait for them.
func (e *Engine) closeMediaSources() {
	e.mu.Lock()
	sources := e.mediaSources
	e.mu.Unlock()
	for _, ms := range sources {
		ms.endOfStream()
	}
}

// mseTypeSupported returns whether media of the given type (with codecs)
// can be appended to a MediaSource: fragmented MP4 of the video and audio
// codecs that FFmpeg decodes (not WebM, so that players use MP4).
func mseTypeSupported(mime string) bool {
	return mediaTypeMP4.MatchString(mime)
}

// MediaSource returns the MediaSource of the given object URL (blob:) of
// the page, if any.
func (e *Engine) MediaSource(url string) *MediaSource {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.mediaSources[url]
}

// setMSEFuncs sets the D functions of MediaSources (see mse.go).
func (e *Engine) setMSEFuncs(set func(name string, f func(call goja.FunctionCall) goja.Value)) {
	vm := e.vm
	sources := map[int]*MediaSource{}
	buffers := map[int]*sourceBuffer{}
	ids := 0
	set("mseTypeSupported", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(mseTypeSupported(call.Argument(0).String()))
	})
	// mseCreate makes a new MediaSource, returning its id and object URL.
	set("mseCreate", func(call goja.FunctionCall) goja.Value {
		ids++
		ms := newMediaSource()
		sources[ids] = ms
		e.mu.Lock()
		if e.mediaSources == nil {
			e.mediaSources = map[string]*MediaSource{}
		}
		e.timerID++
		u := fmt.Sprintf("blob:%s/mse-%d-%d", origin(e.opts.PageURL), time.Now().UnixNano(), e.timerID)
		e.mediaSources[u] = ms
		e.mu.Unlock()
		return vm.ToValue([]any{ids, u})
	})
	set("mseAddBuffer", func(call goja.FunctionCall) goja.Value {
		ms := sources[int(call.Argument(0).ToInteger())]
		if ms == nil {
			return vm.ToValue(0)
		}
		ids++
		buffers[ids] = ms.addBuffer(call.Argument(1).String())
		return vm.ToValue(ids)
	})
	set("mseRemoveBuffer", func(call goja.FunctionCall) goja.Value {
		if b := buffers[int(call.Argument(0).ToInteger())]; b != nil {
			b.src.removeBuffer(b)
		}
		return goja.Undefined()
	})
	// mseAppend appends the given data to the buffer, returning an error
	// message, or "".
	set("mseAppend", func(call goja.FunctionCall) goja.Value {
		b := buffers[int(call.Argument(0).ToInteger())]
		if b == nil {
			return vm.ToValue("no such SourceBuffer")
		}
		if err := b.append(bytes.Clone(bytesOf(call.Argument(1)))); err != nil {
			return vm.ToValue(err.Error())
		}
		return vm.ToValue("")
	})
	set("mseSetOffset", func(call goja.FunctionCall) goja.Value {
		if b := buffers[int(call.Argument(0).ToInteger())]; b != nil {
			b.src.mu.Lock()
			b.offset = call.Argument(1).ToFloat()
			b.src.mu.Unlock()
		}
		return goja.Undefined()
	})
	set("mseRemove", func(call goja.FunctionCall) goja.Value {
		if b := buffers[int(call.Argument(0).ToInteger())]; b != nil {
			b.remove(call.Argument(1).ToFloat(), call.Argument(2).ToFloat())
		}
		return goja.Undefined()
	})
	set("mseAbort", func(call goja.FunctionCall) goja.Value {
		if b := buffers[int(call.Argument(0).ToInteger())]; b != nil {
			b.abort()
		}
		return goja.Undefined()
	})
	// mseBuffered returns the buffered ranges of the buffer, as an array
	// of their starts and ends.
	set("mseBuffered", func(call goja.FunctionCall) goja.Value {
		var r []any
		if b := buffers[int(call.Argument(0).ToInteger())]; b != nil {
			for _, rg := range b.buffered() {
				r = append(r, rg[0], rg[1])
			}
		}
		return vm.ToValue(r)
	})
	set("mseSetDuration", func(call goja.FunctionCall) goja.Value {
		if ms := sources[int(call.Argument(0).ToInteger())]; ms != nil {
			ms.mu.Lock()
			ms.duration = call.Argument(1).ToFloat()
			ms.mu.Unlock()
		}
		return goja.Undefined()
	})
	set("mseEnd", func(call goja.FunctionCall) goja.Value {
		if ms := sources[int(call.Argument(0).ToInteger())]; ms != nil {
			ms.endOfStream()
		}
		return goja.Undefined()
	})
	set("mseRevoke", func(call goja.FunctionCall) goja.Value {
		e.mu.Lock()
		delete(e.mediaSources, call.Argument(0).String())
		e.mu.Unlock()
		return goja.Undefined()
	})
}
