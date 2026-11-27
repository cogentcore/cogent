// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/textcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMessageImages tests the images of a message: one of its HTML at a
// cid: URL, one inline that the HTML does not show (as a photo from a
// phone), and an attached one, which are shown after the body and are
// attachments too.
func TestMessageImages(t *testing.T) {
	var pb bytes.Buffer
	require.NoError(t, png.Encode(&pb, image.NewRGBA(image.Rect(0, 0, 4, 3))))
	data := base64.StdEncoding.EncodeToString(pb.Bytes())
	part := func(header string) string {
		return "--B\r\n" + header + "Content-Transfer-Encoding: base64\r\n\r\n" + data + "\r\n"
	}
	msg := "From: me@example.com\r\nSubject: Photos\r\nMIME-Version: 1.0\r\n" +
		"Content-Type: multipart/related; boundary=\"B\"\r\n\r\n" +
		"--B\r\nContent-Type: text/html; charset=utf-8\r\n\r\n<p>Our logo: <img src=\"cid:logo@example.com\"></p>\r\n" +
		part("Content-Type: image/png\r\nContent-ID: <logo@example.com>\r\n") +
		part("Content-Type: image/png; name=\"photo.png\"\r\nContent-Disposition: inline; filename=\"photo.png\"\r\n") +
		part("Content-Type: image/png\r\nContent-Disposition: attachment; filename=\"chart.png\"\r\n") +
		"--B--\r\n"
	b, a := newTestApp(t, newIMAP(t))
	require.NoError(t, os.MkdirAll(a.messagesDir(testEmail), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(a.messagesDir(testEmail), "images"), []byte(msg), 0o600))
	cm := &CacheMessage{Filename: "images"}
	dmf := NewDisplayMessageFrame(b)
	dmf.Message = cm
	w := core.NewFrame(b)
	require.NoError(t, dmf.displayMessageContents(w))

	var names []string
	for _, at := range cm.parsed.attachments {
		names = append(names, at.Filename)
	}
	assert.ElementsMatch(t, []string{"photo.png", "chart.png"}, names, "not the logo, which the HTML shows")
	n := 0
	for k := range w.AllWidgets() {
		if _, ok := k.(*core.Image); ok {
			n++
		}
	}
	assert.Equal(t, 3, n, "the logo, and the photo and the chart after the body")

	ctx := messageImagesContext(htmlcore.NewContext(), "images/", map[string]*partImage{"logo@example.com": {contentType: "image/png", data: pb.Bytes()}})
	resp, err := ctx.GetURL()("cid:images/logo%40example.com")
	require.NoError(t, err)
	got, _ := io.ReadAll(resp.Body)
	assert.Equal(t, pb.Bytes(), got)
	assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))
	resp, err = ctx.GetURL()("cid:images/other")
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
}

// TestDropAttachments tests attaching files dropped on a message being
// written: an attachment of another message, dragged from it, as a copy
// that is removed with the message, and another file as it is.
func TestDropAttachments(t *testing.T) {
	_, a := newTestApp(t, newIMAP(t))
	at := &Attachment{Filename: "notes.txt", Data: []byte("some notes")}
	dragged, err := at.tempFile()
	require.NoError(t, err)
	other := filepath.Join(t.TempDir(), "other.txt")
	require.NoError(t, os.WriteFile(other, []byte("other"), 0o600))
	sm := &SendMessage{}
	ok, _ := a.dropAttachments(sm, &events.DragDrop{Data: mimedata.NewText("not a file")})
	assert.False(t, ok)

	ok, err = a.dropAttachments(sm, &events.DragDrop{Data: mimedata.NewText(dragged)})
	require.NoError(t, err)
	assert.True(t, ok)
	ok, _ = a.dropAttachments(sm, &events.DragDrop{Data: mimedata.NewText(other)})
	assert.True(t, ok)
	require.Len(t, sm.Attachments, 2)
	assert.True(t, a.isForwarded(sm.Attachments[0]), "a copy")
	assert.Equal(t, "notes.txt", filepath.Base(string(sm.Attachments[0])))
	data, err := os.ReadFile(string(sm.Attachments[0]))
	require.NoError(t, err)
	assert.Equal(t, "some notes", string(data))
	assert.Equal(t, core.Filename(other), sm.Attachments[1])
	a.removeForwarded(sm.Attachments)
	assert.NoFileExists(t, string(sm.Attachments[0]))
	assert.FileExists(t, other, "not removed")
}

// TestDragAttachment tests dragging the button of an attachment with the
// mouse and dropping it on a message being written, which attaches it.
func TestDragAttachment(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(500), units.Dp(400)) })
	ab := NewAttachmentButton(b)
	ab.Attachment = &Attachment{Filename: "notes.txt", Data: []byte("some notes")}
	target := core.NewFrame(b)
	target.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(300), units.Dp(150)) })
	opened := 0
	oldOpen := openURL
	openURL = func(string) { opened++ }
	t.Cleanup(func() { openURL = oldOpen })
	sm := &SendMessage{}
	dropped := 0
	a.attachDropped(sm, func(e events.Event, err error) {
		assert.NoError(t, err)
		dropped++
	}, target)
	core.RenderImage(b, func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		b.AsyncLock()
		fp := ab.Geom.TotalBBox.Min.Add(ab.Geom.TotalBBox.Size().Div(2))
		tbb := target.Geom.TotalBBox
		tp := tbb.Min.Add(tbb.Size().Div(2))
		b.AsyncUnlock()
		src.MouseButton(events.MouseDown, events.Left, fp, 0)
		for i := 1; i <= 20; i++ {
			src.MouseMove(fp.Add(tp.Sub(fp).Mul(i).Div(20)))
			time.Sleep(30 * time.Millisecond)
		}
		for range 5 {
			src.MouseMove(tp)
			time.Sleep(30 * time.Millisecond)
		}
		src.MouseButton(events.MouseUp, events.Left, tp, 0)
		time.Sleep(200 * time.Millisecond)
	}, core.Locked(b, func() {
		assert.Equal(t, 1, dropped)
		require.Len(t, sm.Attachments, 1)
		assert.Equal(t, "notes.txt", filepath.Base(string(sm.Attachments[0])))
		assert.True(t, a.isForwarded(sm.Attachments[0]), "a copy")
		assert.Zero(t, opened, "not opened, as by a click")
	}))
}

// TestImageAttachment tests the files of the images of messages that are
// dragged: the image of the message for a cid: URL, with its name, and
// otherwise a PNG named from its URL.
func TestImageAttachment(t *testing.T) {
	images := map[string]*partImage{"logo@x": {contentType: "image/jpeg", name: "logo.jpg", data: []byte("jpeg")}}
	im := core.NewImage(core.NewBody())
	at, err := imageAttachment(im, "cid:m/logo%40x", "m/", images)
	require.NoError(t, err)
	assert.Equal(t, &Attachment{Filename: "logo.jpg", Data: []byte("jpeg")}, at, "that of the message")
	at, err = imageAttachment(im, "https://example.com/a/photo.jpg?w=100", "m/", images)
	require.NoError(t, err)
	assert.Nil(t, at, "not loaded yet")
	im.SetImage(image.NewRGBA(image.Rect(0, 0, 3, 2)))
	at, err = imageAttachment(im, "https://example.com/a/photo.jpg?w=100", "m/", images)
	require.NoError(t, err)
	assert.Equal(t, "photo.png", at.Filename)
	got, err := png.Decode(bytes.NewReader(at.Data))
	require.NoError(t, err)
	assert.Equal(t, image.Rect(0, 0, 3, 2), got.Bounds())
}

// TestExternalDropAttachment tests dropping a file from another app (as
// the Finder) on the editor of a message being written, which attaches it
// as it is, instead of inserting its name.
func TestExternalDropAttachment(t *testing.T) {
	b, a := newTestApp(t, newIMAP(t))
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(400), units.Dp(300)) })
	target := textcore.NewEditor(b)
	target.Styler(func(s *styles.Style) { s.Grow.Set(1, 1) })
	sm := &SendMessage{}
	a.attachDropped(sm, func(e events.Event, err error) { assert.NoError(t, err) }, target)
	file := filepath.Join(t.TempDir(), "report.pdf")
	require.NoError(t, os.WriteFile(file, []byte("%PDF"), 0o600))
	core.RenderImage(b, func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		b.AsyncLock()
		bb := target.Geom.TotalBBox
		b.AsyncUnlock()
		pt := bb.Min.Add(bb.Size().Div(2))
		md := mimedata.NewText(file)
		for range 3 {
			src.DragExternal(events.DragMove, pt, 0, md, false)
			time.Sleep(30 * time.Millisecond)
		}
		src.DragExternal(events.Drop, pt, 0, md, false)
		time.Sleep(200 * time.Millisecond)
	}, core.Locked(b, func() {
		assert.Equal(t, []core.Filename{core.Filename(file)}, sm.Attachments, "as it is")
		assert.Empty(t, strings.TrimSpace(target.Lines.String()), "not its name")
	}))
}
