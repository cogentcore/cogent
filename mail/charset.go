// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"io"
	"strings"
	"unicode/utf8"

	"github.com/emersion/go-message"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"

	"github.com/emersion/go-message/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

func init() {
	// the charsets of messages whose sender did not know theirs
	for _, name := range []string{"unknown-8bit", "x-unknown", "unknown"} {
		charset.RegisterEncoding(name, unknown8bit{})
	}
	// names of Windows code pages that are not in the standard indexes
	charset.RegisterEncoding("cp932", japanese.ShiftJIS)
	charset.RegisterEncoding("windows-932", japanese.ShiftJIS)
	charset.RegisterEncoding("cp936", simplifiedchinese.GBK)
	charset.RegisterEncoding("cp949", korean.EUCKR)
	charset.RegisterEncoding("cp950", traditionalchinese.Big5)
	message.CharsetReader = charsetReader
}

// charsetReader returns a reader of the text in the given charset, as
// UTF-8, as [charset.Reader], for the name of the charset as senders put
// it, in quotes or with spaces around it.
func charsetReader(name string, input io.Reader) (io.Reader, error) {
	name = strings.Trim(strings.TrimSpace(name), `"'`)
	return charset.Reader(name, input)
}

// unknown8bit is the encoding of text in an unknown 8-bit charset: the
// valid UTF-8 is kept, and the other bytes are decoded as Windows-1252,
// the usual 8-bit charset of such messages.
type unknown8bit struct{}

func (unknown8bit) NewDecoder() *encoding.Decoder {
	return &encoding.Decoder{Transformer: utf8OrWindows1252{}}
}

func (unknown8bit) NewEncoder() *encoding.Encoder {
	return &encoding.Encoder{Transformer: encoding.UTF8Validator} // not used, as only read
}

// utf8OrWindows1252 is the decoder of [unknown8bit].
type utf8OrWindows1252 struct{ transform.NopResetter }

func (utf8OrWindows1252) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for nSrc < len(src) {
		r, size := utf8.DecodeRune(src[nSrc:])
		if r == utf8.RuneError && size <= 1 {
			if !atEOF && !utf8.FullRune(src[nSrc:]) {
				return nDst, nSrc, transform.ErrShortSrc // the rest may come
			}
			r, size = charmap.Windows1252.DecodeByte(src[nSrc]), 1
		}
		if nDst+utf8.RuneLen(r) > len(dst) {
			return nDst, nSrc, transform.ErrShortDst
		}
		nDst += utf8.EncodeRune(dst[nDst:], r)
		nSrc += size
	}
	return nDst, nSrc, nil
}
