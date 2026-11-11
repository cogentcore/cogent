// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"cogentcore.org/cogent/web/script"
	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"golang.org/x/net/html"
	"strings"
)

// The frames of a page: each iframe element of the page that is shown has
// a [Page] of its own in its box (see htmlcore's handleIframe), with its
// own history and scripts, as in browsers, as the captchas of bot checks,
// which are pages of the sites that check in frames of the pages that they
// block. The page in a frame is kept while the iframe element is in the
// document, as the widgets of the page are read again (as after its
// scripts change it), and its scripts see the scripts of the page around
// it as another window (window.parent), to which they post messages (see
// the script package).

// frame is the state of a page that is in a frame of another page.
type frame struct {
	// parent is the page that the frame is in.
	parent *Page

	// node is the iframe element of the frame.
	node *html.Node

	// src is the URL that was opened in the frame, from the src of the
	// iframe (which scripts can change), or "" for none.
	src string

	// engine is the engine of the scripts of the parent page when the
	// frame was made, if it has scripts, which gets the load events of the
	// iframe and the messages of the frame.
	engine *script.Engine
}

// setContent shows the given content (of a page that has been read) in
// the page, in place of the content before it, keeping the pages in the
// frames of the iframes that are still in it (with the lock held).
func (pg *Page) setContent(content *core.Frame) {
	opens := pg.attachFrames(content)
	pg.attachVideos(content)
	pg.DeleteChildren()
	pg.AddChild(content)
	// the pages of the frames are opened in the scene of the page
	for _, fp := range pg.frames {
		fp.setShown()
	}
	for _, open := range opens {
		open()
	}
	pg.Update()
}

// attachFrames moves the pages of the frames of the iframes of the given
// content into the boxes of their iframes in it, making the pages of new
// iframes, and closes the frames of the iframes that are no longer in it.
// It returns the functions that open the pages of the frames whose
// sources have changed (as new ones), to call once the content is in the
// page.
func (pg *Page) attachFrames(content *core.Frame) (opens []func()) {
	seen := map[*html.Node]bool{}
	content.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		// (the element of the widget itself: NodeOf of the content of the
		// page of a frame is the iframe of the page that it is in, which
		// was opened again in it, in frames in frames, as the challenge of
		// reCAPTCHA, which then had no name, nor its checkbox in its parent)
		n, _ := cwb.Property("node").(*html.Node)
		if n == nil || n.Data != "iframe" {
			return tree.Continue
		}
		if seen[n] {
			return tree.Break // (a box of the same element, as of a re-read)
		}
		seen[n] = true
		src := pg.frameSource(n)
		fp := pg.frames[n]
		if fp == nil {
			fp = pg.newFrame(n)
		}
		tree.MoveToParent(fp, cw)
		if open := pg.frameOpens[n]; open != nil {
			// a page opened in it before it had a page (in place of its src)
			delete(pg.frameOpens, n)
			fp.frame.src = src
			opens = append(opens, func() { open(fp) })
		} else if src != fp.frame.src {
			fp.frame.src = src
			opens = append(opens, func() {
				if src == "" {
					fp.closeFrames()
					fp.showScripts(nil)
					fp.DeleteChildren()
					fp.frameLoaded()
				} else {
					fp.OpenURL(src, nil)
				}
			})
		}
		return tree.Break // its children are the frame
	})
	for n, fp := range pg.frames {
		if !seen[n] {
			fp.closeFrame()
			delete(pg.frames, n)
		}
	}
	for n := range pg.frameOpens {
		if !seen[n] && n.Parent == nil { // (an iframe that is gone)
			delete(pg.frameOpens, n)
		}
	}
	return opens
}

// frameSource returns the absolute URL of the page of the given iframe
// element, or "" for none (as about:blank).
func (pg *Page) frameSource(n *html.Node) string {
	src := htmlcore.GetAttr(n, "src")
	if src == "" || src == "about:blank" {
		return ""
	}
	u, err := htmlcore.ParseRelativeURL(src, pg.Context.PageURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "" // (javascript: and data: URLs are not opened)
	}
	return u.String()
}

// newFrame returns a new page for the frame of the given iframe element
// of this page.
func (pg *Page) newFrame(n *html.Node) *Page {
	fp := NewPage()
	fp.frame = &frame{parent: pg, node: n}
	fp.Styler(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowAuto) // it scrolls in its frame
		if strings.EqualFold(strings.TrimSpace(htmlcore.GetAttr(n, "scrolling")), "no") {
			s.Overflow.Set(styles.OverflowHidden)
		}
	})
	if pg.scripts != nil {
		fp.frame.engine = pg.scripts.engine
	}
	fp.Scripts = pg.Scripts
	fp.OpenNewTab = pg.OpenNewTab
	if pg.frames == nil {
		pg.frames = map[*html.Node]*Page{}
	}
	pg.frames[n] = fp
	return fp
}

// closeFrame stops the page of a frame (its loading and scripts, and those
// of its frames) and deletes it.
func (pg *Page) closeFrame() {
	pg.load.Add(1) // cancels the opening of a page
	pg.loading = ""
	pg.stopMediaTimer()
	pg.closeFrames()
	pg.closeVideos()
	pg.showScripts(nil)
	pg.Delete()
}

// closeFrames closes the frames of the page.
func (pg *Page) closeFrames() {
	for n, fp := range pg.frames {
		fp.closeFrame()
		delete(pg.frames, n)
	}
}

// frameEngine returns the engine of the scripts of the page in the frame
// of the given iframe element of this page, if it has one (for the
// contentWindow of the iframe). It is called in the tasks of the scripts
// of this page (with the lock held).
func (pg *Page) frameEngine(n *html.Node) *script.Engine {
	fp := pg.frames[n]
	if fp == nil || fp.scripts == nil {
		return nil
	}
	return fp.scripts.engine
}

// parentEngine returns the engine of the scripts of the page that this
// page is in a frame of, if any (for window.parent).
func (pg *Page) parentEngine() *script.Engine {
	if pg.frame == nil {
		return nil
	}
	return pg.frame.engine
}

// frameLoaded fires the load event of the iframe of the page, if it is in
// a frame of a page with scripts, as its page has loaded (or failed to).
func (pg *Page) frameLoaded() {
	if e := pg.parentEngine(); e != nil {
		e.FrameLoaded(pg.frame.node)
	}
}

// topPage returns the page that this one is in a frame of (through the
// frames that it is in), or itself if it is not in a frame.
func (pg *Page) topPage() *Page {
	for pg.frame != nil {
		pg = pg.frame.parent
	}
	return pg
}

// AsyncLock locks the widgets of the page for a goroutine (see
// [core.WidgetBase.AsyncLock]) with the top page that it is in, as the
// page of a frame is moved to the new widgets of its parent page (with
// the lock held) whenever the parent is read again, which sets its scene.
func (pg *Page) AsyncLock() {
	pg.topPage().Frame.AsyncLock()
}

// AsyncLockIfShown is [core.WidgetBase.AsyncLockIfShown] with the top
// page (see [Page.AsyncLock]).
func (pg *Page) AsyncLockIfShown() bool {
	return pg.topPage().Frame.AsyncLockIfShown()
}

// AsyncUnlock unlocks the lock of [Page.AsyncLock].
func (pg *Page) AsyncUnlock() {
	pg.topPage().Frame.AsyncUnlock()
}

// targetPage returns the page that a link (or form) of this page with the
// given target opens its page in, as in browsers: the page itself for none
// (or _self), the page that it is in a frame of for _parent, the top page
// for _top, and the page of the frame of the iframe of that name (in the
// pages of the top page) for a name, or nil for _blank (a new tab). A
// name of no frame is the page itself. (The links of the account menu of
// Google, in a frame, are of _top, which opened them in the frame.)
func (pg *Page) targetPage(target string) *Page {
	switch strings.ToLower(strings.TrimSpace(target)) {
	case "", "_self":
		return pg
	case "_blank":
		return nil
	case "_parent":
		if pg.frame != nil {
			return pg.frame.parent
		}
		return pg
	case "_top":
		return pg.topPage()
	}
	if fp := pg.topPage().namedFrame(target); fp != nil {
		return fp
	}
	return pg
}

// namedFrame returns the page of the frame of the iframe with the given
// name in this page or the pages of its frames, or nil.
func (pg *Page) namedFrame(name string) *Page {
	for n, fp := range pg.frames {
		if nameAttr(n) == name {
			return fp
		}
		if f := fp.namedFrame(name); f != nil {
			return f
		}
	}
	return nil
}

// nameAttr returns the name attribute of the given element.
func nameAttr(n *html.Node) string {
	return htmlcore.GetAttr(n, "name")
}
