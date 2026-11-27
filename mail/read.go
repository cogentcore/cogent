// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"bytes"
	"fmt"
	"html"
	"image/png"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
	"github.com/cogentcore/go-imap/v2"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
)

// readMessageParsed contains data parsed from the current message we are reading.
type readMessageParsed struct {

	// references is the References header.
	references []string

	// plain is the plain text body.
	plain string

	// attachments are the attachments.
	attachments []*Attachment

	// removed are the names of the attachments that were removed from it
	// (see [App.RemoveAttachments]).
	removed []string
}

// shownMessage is a message as shown, with its file, which is another
// once its attachments are removed, when it is shown again.
type shownMessage struct {
	message  *CacheMessage
	filename string
}

// Attachment represents an email attachment when reading a message.
type Attachment struct {
	Filename string
	Data     []byte

	// openedFile is the temporary file it was last opened from, if any.
	openedFile string
}

// DisplayMessageFrame is a frame that displays the metadata and contents of a message.
type DisplayMessageFrame struct {
	core.Frame
	Message *CacheMessage
}

func (dmf *DisplayMessageFrame) WidgetValue() any { return &dmf.Message }

func (dmf *DisplayMessageFrame) Init() {
	dmf.Frame.Init()
	dmf.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
		s.Direction = styles.Column
	})
	var header *core.Frame
	tree.AddChild(dmf, func(w *core.Frame) {
		header = w
		dmf.initHeader(w)
	})
	tree.AddChild(dmf, func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Grow.Set(1, 0)
		})
		w.Updater(func() {
			if dmf.Message.Filename == "" { // not downloaded yet
				text := "Downloading the message..."
				if theApp.imapClient[theApp.currentEmail] == nil {
					text = "Not downloaded yet: it will be once connected."
				}
				if w.Property("readMessage") != text {
					w.SetProperty("readMessage", text)
					w.DeleteChildren()
					core.NewText(w).SetType(core.TextSupporting).SetText(text)
				}
				theApp.downloadNow(theApp.currentEmail, dmf.Message)
				return
			}
			if (shownMessage{dmf.Message, dmf.Message.Filename}) != w.Property("readMessage") {
				core.ErrorSnackbar(w, dmf.displayMessageContents(w), "Error reading message")
				header.Update() // with the attachments, which are now parsed
			}
		})
	})
}

// messageImages is the cache of the decoded images of the messages, shared
// by all of them, so that showing a message again does not get its images
// again.
var messageImages = htmlcore.NewImageCache(200 << 20)

// messageContext returns a new context for reading the HTML (or markdown)
// of the given message, with a bar on the left of quoted text, in which a
// mailto: link opens a reply to the message to its addresses. Its images
// are gotten in the background, after it is shown, as there can be many.
func messageContext(cm *CacheMessage) *htmlcore.Context {
	bar := colors.AsHex(colors.ToUniform(colors.Scheme.Primary.Base))
	text := colors.AsHex(colors.ToUniform(colors.Scheme.OnSurfaceVariant))
	return htmlcore.NewContext().SetDelayedImageLoad(true).SetImageCache(messageImages).SetStyle(fmt.Sprintf(
		"blockquote { margin: 0.5em 0 0.5em 0.25em; padding-left: 0.75em; border-left: 3px solid %s; color: %s; }", bar, text)).
		SetOpenURL(func(u string, e events.Event) {
			if isMailto(u) {
				theApp.replyMailto(u, cm)
				return
			}
			core.TheApp.OpenURL(u)
		})
}

// headerDateFormat is the format of the date in the header of a message.
const headerDateFormat = "Mon, Jan 2, 2006 at 3:04 PM"

// addressLinks returns the given addresses as HTML, each a mailto: link
// with its name, if any, and address.
func addressLinks(as []imap.Address) string {
	var res []string
	for _, a := range as {
		addr := a.Addr()
		label := html.EscapeString(addr)
		if a.Name != "" && a.Name != addr {
			label = html.EscapeString(a.Name) + " &lt;" + label + "&gt;"
		}
		res = append(res, fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString("mailto:"+addr), label))
	}
	return strings.Join(res, ", ")
}

// otherLabels returns the labels of the given message other than the
// mailbox being shown and those that are not shown, in the order of the
// mailboxes.
func otherLabels(cm *CacheMessage) []string {
	var res []string
	for _, l := range cm.Labels {
		if l.Name != theApp.showLabel && !skipLabels[l.Name] && !theApp.isStarredLabel(l.Name) { // the star is shown
			res = append(res, l.Name)
		}
	}
	return sortedLabels(res)
}

// initHeader initializes the header of the message: the names of the
// fields in bold, with their values, and its other labels, which go to
// their mailboxes when clicked, and its attachments.
func (dmf *DisplayMessageFrame) initHeader(w *core.Frame) {
	w.Styler(func(s *styles.Style) {
		s.Display = styles.Grid
		s.Columns = 2
		s.Grow.Set(1, 0)
		s.Gap.Set(units.Em(1), units.Em(0.25))
	})
	w.Maker(func(p *tree.Plan) {
		cm := dmf.Message
		field := func(name string, value func(p *tree.Plan)) {
			tree.AddAt(p, name+"-label", func(w *core.Text) {
				w.SetText(name)
				w.Styler(func(s *styles.Style) {
					s.Font.Weight = rich.Bold
					s.Align.Self = styles.Start
				})
			})
			value(p)
		}
		// htmlText adds a field with the given HTML, as of the message
		// when it is updated, if it is not empty
		htmlText := func(name string, value func(cm *CacheMessage) string) {
			if value(cm) == "" {
				return
			}
			field(name, func(p *tree.Plan) {
				tree.AddAt(p, name, func(w *core.Text) {
					w.Styler(func(s *styles.Style) {
						s.Grow.Set(1, 0) // wrapping in the width of the header
					})
					// a mailto: link of an address replies to the message
					w.HandleTextClick(func(tl *rich.Hyperlink, e events.Event) {
						if isMailto(tl.URL) {
							theApp.replyMailto(tl.URL, dmf.Message)
							return
						}
						core.TheApp.OpenURL(tl.URL)
					})
					w.Updater(func() { w.SetText(value(dmf.Message)) })
				})
			})
		}
		text := func(name string, value func(cm *CacheMessage) string) {
			htmlText(name, func(cm *CacheMessage) string { return html.EscapeString(value(cm)) })
		}
		addresses := func(name string, as func(cm *CacheMessage) []imap.Address) {
			htmlText(name, func(cm *CacheMessage) string { return addressLinks(as(cm)) })
		}
		addresses("From", func(cm *CacheMessage) []imap.Address { return cm.From })
		addresses("To", func(cm *CacheMessage) []imap.Address { return cm.To })
		addresses("Cc", func(cm *CacheMessage) []imap.Address { return cm.Cc })
		text("Subject", func(cm *CacheMessage) string { return cm.Subject })
		text("Date", func(cm *CacheMessage) string { return cm.Date.Local().Format(headerDateFormat) })
		if cm.isLocal() {
			text("Status", func(cm *CacheMessage) string { return theApp.draftStatus(cm) })
			addresses("Bcc", func(cm *CacheMessage) []imap.Address { return cm.Bcc })
		}
		if labels := otherLabels(cm); len(labels) > 0 {
			field("Labels", func(p *tree.Plan) {
				tree.AddAt(p, "Labels", func(w *core.Frame) {
					w.Styler(func(s *styles.Style) {
						s.Wrap = true
						s.Grow.Set(1, 0)
						s.Gap.Set(units.Em(0.25))
					})
					w.Maker(func(p *tree.Plan) {
						for _, label := range otherLabels(dmf.Message) {
							tree.AddAt(p, label, func(w *core.Button) {
								w.SetType(core.ButtonTonal).SetText(friendlyLabelName(label)).SetIcon(icons.Label)
								w.SetTooltip("Show the messages of " + friendlyLabelName(label))
								w.OnClick(func(e events.Event) {
									theApp.showMailbox(theApp.currentEmail, label)
								})
							})
						}
					})
				})
			})
		}
		if len(cm.parsed.attachments) > 0 || len(cm.parsed.removed) > 0 {
			field("Attachments", func(p *tree.Plan) {
				tree.AddAt(p, "Attachments", func(w *core.Frame) {
					w.Styler(func(s *styles.Style) {
						s.Wrap = true
						s.Grow.Set(1, 0)
						s.Gap.Set(units.Em(0.25))
					})
					w.Maker(func(p *tree.Plan) {
						for i, at := range dmf.Message.parsed.attachments {
							tree.AddAt(p, strconv.Itoa(i), func(w *AttachmentButton) {
								w.Updater(func() { w.Attachment = at })
							})
						}
						for i, name := range dmf.Message.parsed.removed {
							tree.AddAt(p, "removed-"+strconv.Itoa(i), func(w *core.Text) {
								w.SetType(core.TextBodySmall).SetText(name + " (removed)")
								w.SetTooltip("This attachment was removed from the message")
								w.Styler(func(s *styles.Style) {
									s.Color = colors.Scheme.OnSurfaceVariant
									s.Align.Self = styles.Center
								})
							})
						}
					})
				})
			})
		}
	})
}

// displayMessageContents updates the given frame to display the contents of
// the current message, if it does not already.
func (dmf *DisplayMessageFrame) displayMessageContents(w *core.Frame) error {
	w.SetProperty("readMessage", shownMessage{dmf.Message, dmf.Message.Filename})
	w.DeleteChildren()

	f, err := os.Open(filepath.Join(theApp.messagesDir(theApp.currentEmail), dmf.Message.Filename))
	if err != nil {
		return err
	}
	defer f.Close()

	mr, err := mail.CreateReader(f)
	if err != nil {
		return err
	}

	refs, err := mr.Header.MsgIDList("References")
	if err != nil {
		return err
	}
	dmf.Message.parsed.references = refs

	dmf.Message.parsed.attachments = nil
	dmf.Message.parsed.removed = nil
	dmf.Message.parsed.plain = ""
	var htmls []string
	images := map[string]*partImage{} // by their Content-IDs
	var order []string                // the IDs of the images, in order
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if im, err := readPartImage(p); err != nil {
			return err
		} else if im != nil { // inline or attached, shown in the message
			id := im.id
			if id == "" {
				id = "part-" + strconv.Itoa(len(images))
			}
			images[id] = im
			order = append(order, id)
			continue
		}
		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			ct, _, err := h.ContentType()
			if err != nil {
				return err
			}

			if name := h.Get(removedHeader); name != "" { // shown with the attachments
				if dec, err := new(mime.WordDecoder).DecodeHeader(name); err == nil {
					name = dec
				}
				dmf.Message.parsed.removed = append(dmf.Message.parsed.removed, name)
				continue
			}
			switch {
			case ct == "text/plain":
				b, err := io.ReadAll(p.Body)
				if err != nil {
					return err
				}
				// all of the text parts, in order, as Apple Mail sends
				// text, then attachments, then more (often blank) text
				if text := string(b); strings.TrimSpace(text) != "" {
					if dmf.Message.parsed.plain != "" {
						dmf.Message.parsed.plain += "\n\n"
					}
					dmf.Message.parsed.plain += text
				}
			case ct == "text/html":
				h, err := io.ReadAll(p.Body)
				if err != nil {
					return err
				}
				htmls = append(htmls, string(h))
			}
		case *mail.AttachmentHeader:
			fname, err := h.Filename()
			if err != nil {
				return err
			}
			at := &Attachment{Filename: fname}
			at.Data, err = io.ReadAll(p.Body)
			if err != nil {
				return err
			}
			dmf.Message.parsed.attachments = append(dmf.Message.parsed.attachments, at)
		}
	}

	// the images that the HTML does not show are attachments, shown after
	// the body, as photos sent from phones
	all := strings.Join(htmls, "")
	var shown []string
	for _, id := range order {
		if im := images[id]; im.id == "" || !strings.Contains(all, "cid:"+id) {
			dmf.Message.parsed.attachments = append(dmf.Message.parsed.attachments, &Attachment{Filename: im.name, Data: im.data})
			shown = append(shown, id)
		}
	}

	prefix := dmf.Message.Filename + "/" // unique, as images are cached by URL
	ctx := messageImagesContext(messageContext(dmf.Message), prefix, images)
	for _, h := range htmls {
		if err := htmlcore.ReadHTMLString(ctx, w, messageHTML(cidSrc.ReplaceAllString(h, "${1}cid:"+prefix))); err != nil {
			return err
		}
	}
	// we only handle the plain version if there is no HTML version
	if len(htmls) == 0 {
		err := htmlcore.ReadMDString(ctx, w, dmf.Message.parsed.plain)
		if err != nil {
			return err
		}
	}
	if len(shown) > 0 {
		var b strings.Builder
		for _, id := range shown {
			fmt.Fprintf(&b, `<p><img src="cid:%s" alt="%s" style="max-width: 100%%"></p>`, html.EscapeString(prefix+id), html.EscapeString(images[id].name))
		}
		if err := htmlcore.ReadHTMLString(ctx, w, b.String()); err != nil {
			return err
		}
	}
	return nil
}

// dragImage makes the given image of a message draggable, as a file, to a
// message being written or other apps: that of the message, for an image
// at a cid: URL with the given prefix (see [messageImagesContext]), and
// otherwise a PNG of it.
func dragImage(im *core.Image, prefix string, images map[string]*partImage) {
	im.Styler(func(s *styles.Style) { s.SetAbilities(true, abilities.Draggable) })
	im.On(events.DragStart, func(e events.Event) {
		src, _ := im.Property("src").(string)
		at, err := imageAttachment(im, src, prefix, images)
		if err != nil || at == nil {
			return
		}
		fname, err := at.tempFile()
		if err != nil {
			core.ErrorSnackbar(im, err, "Error dragging image")
			return
		}
		e.SetHandled()
		im.Scene.Events.DragStart(im.This.(core.Widget), mimedata.NewText(fname), e)
		im.Scene.Events.SetDragFiles(fname)
	})
}

// imageAttachment returns the given image of a message, at the given URL,
// as an attachment (see [dragImage]), or nil if it has no image yet.
func imageAttachment(im *core.Image, src, prefix string, images map[string]*partImage) (*Attachment, error) {
	if rest, ok := strings.CutPrefix(src, "cid:"); ok {
		id, _ := strings.CutPrefix(rest, prefix)
		if un, err := url.PathUnescape(id); err == nil {
			id = un
		}
		if pi := images[id]; pi != nil {
			return &Attachment{Filename: pi.name, Data: pi.data}, nil
		}
	}
	if im.Image == nil {
		return nil, nil
	}
	var b bytes.Buffer
	if err := png.Encode(&b, im.Image); err != nil {
		return nil, err
	}
	name := "image"
	if u, err := url.Parse(src); err == nil && !strings.HasPrefix(src, "data:") {
		if base := path.Base(u.Path); base != "." && base != "/" && base != "" {
			name = strings.TrimSuffix(base, path.Ext(base))
		}
	}
	return &Attachment{Filename: name + ".png", Data: b.Bytes()}, nil
}

// partImage is an image in a part of a message, inline or attached, shown
// in the message (see [messageImagesContext]), with its Content-ID, if any.
type partImage struct {
	contentType, name, id string
	data                  []byte
}

// readPartImage returns the image of the given part of a message, if it
// is one, with a name: its file name, or one of its type.
func readPartImage(p *mail.Part) (*partImage, error) {
	var h message.Header
	switch ph := p.Header.(type) {
	case *mail.InlineHeader:
		h = ph.Header
	case *mail.AttachmentHeader:
		h = ph.Header
	default:
		return nil, nil
	}
	ct, params, err := h.ContentType()
	if err != nil || !strings.HasPrefix(ct, "image/") {
		return nil, nil
	}
	data, err := io.ReadAll(p.Body)
	if err != nil {
		return nil, err
	}
	name := params["name"]
	if _, dp, err := h.ContentDisposition(); err == nil && dp["filename"] != "" {
		name = dp["filename"]
	}
	if name == "" {
		name = "image." + strings.TrimPrefix(ct, "image/")
	}
	return &partImage{contentType: ct, name: name, id: strings.Trim(h.Get("Content-Id"), "<> "), data: data}, nil
}

// cidSrc matches the start of a cid: URL of the src of an image in HTML.
var cidSrc = regexp.MustCompile(`(?i)(\bsrc\s*=\s*["']?)cid:`)

// messageImagesContext returns the given context for reading a message,
// getting the images of its parts at cid: URLs with the given prefix (the
// file of the message, as images are cached by their URLs, and other
// messages can have the same Content-IDs) from the given images, by their
// Content-IDs.
func messageImagesContext(ctx *htmlcore.Context, prefix string, images map[string]*partImage) *htmlcore.Context {
	ctx.AddWidgetHandler(func(w core.Widget) {
		if im, ok := w.(*core.Image); ok {
			dragImage(im, prefix, images)
		}
	})
	get := ctx.GetURL()
	return ctx.SetGetURL(func(u string) (*http.Response, error) {
		rest, ok := strings.CutPrefix(u, "cid:")
		if !ok {
			return get(u)
		}
		id, _ := strings.CutPrefix(rest, prefix)
		if un, err := url.PathUnescape(id); err == nil {
			id = un
		}
		im := images[id]
		if im == nil {
			return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found", Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK",
			Header: http.Header{"Content-Type": {im.contentType}},
			Body:   io.NopCloser(bytes.NewReader(im.data)), ContentLength: int64(len(im.data))}, nil
	})
}

// bodyTag and styleAttr are the <body> tag of HTML and a style attribute.
var (
	bodyTag   = regexp.MustCompile(`(?i)<body\b[^>]*>`)
	styleAttr = regexp.MustCompile(`(?i)\sstyle\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)`)
)

// messageHTML returns the given HTML of a message as it is shown, without
// the style of its <body>, as Gmail and other mail programs show it, as
// messages are written for: as the max-width of the body of messages of
// Facebook, whose content is wider, which was shown to the right of it.
func messageHTML(h string) string {
	return bodyTag.ReplaceAllStringFunc(h, func(tag string) string {
		return styleAttr.ReplaceAllString(tag, "")
	})
}
