// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"bytes"
	"fmt"
	"image"
	"log/slog"
	"maps"
	"math"
	neturl "net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"cogentcore.org/cogent/web/script"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/tree"
	"golang.org/x/net/html"
)

// localStorage is the localStorage of the pages of the app, shared by its
// pages (by origin).
var localStorage = script.NewStorage()

// maxConsole is the number of console messages that a page keeps.
const maxConsole = 500

// pageScripts is the state of the scripts of a page (see [Page.Scripts]).
// Its fields other than the console are used in the tasks of its engine
// (with the lock of the page held, once it is shown).
type pageScripts struct {
	pg     *Page
	engine *script.Engine
	doc    *html.Node

	// base is the context that the page was first read with, whose
	// settings each reading of the document starts from.
	base *htmlcore.Context

	// ctx is the context that the document was last read with.
	ctx *htmlcore.Context

	// width is the width of the viewport that the document is read for.
	width float32

	// seen is the size of the viewport that the scripts last got, if
	// they have, for firing resize events when it changes.
	seen math32.Vector2

	// shown is whether the page is shown, before which it is not
	// rendered again (pending is whether it needs to be then).
	shown, pending bool

	// hovered is the element that the mouse is over, for the events of the
	// mouse moving over the elements (see [pageScripts.hoverAt]).
	hovered *html.Node

	// hoverSet has the elements that the mouse is over (the hovered
	// element and those it is in) for the :hover selectors of the page
	// (see [htmlcore.Context.Hovered]), only used in the tasks of the
	// engine (as the reading of the page).
	hoverSet map[*html.Node]bool

	// layoutMissed is whether the scripts got the box of an element before
	// the page had a layout (before it was shown), for which they get a
	// resize event once it has one.
	layoutMissed bool

	// pushed is the URL that the scripts set (with history.pushState)
	// before the page was shown, if any.
	pushed string

	// widgets has the widget of each node, once it is needed (for the
	// layout of the nodes), and laidOut whether the page has been laid
	// out since it was last rendered.
	widgets map[*html.Node]*core.WidgetBase
	laidOut bool

	// renders is the number of times that the page has been rendered
	// again after it was shown (for tests).
	renders    int
	renderTime time.Duration

	consoleMu sync.Mutex
	console   []string
}

// newPageScripts returns the scripts of the given parsed document of the
// page, which was gotten with the given context, for the given final URL.
func (pg *Page) newPageScripts(ctx *htmlcore.Context, doc *html.Node, final string) (*pageScripts, error) {
	ps := &pageScripts{pg: pg, doc: doc, base: ctx, width: ctx.MediaWidth}
	e, err := script.New(doc, script.Options{
		PageURL:       final,
		GetURL:        ctx.GetURL,
		Render:        ps.render,
		Layout:        ps.layout,
		ElementAt:     ps.elementAt,
		ComputedStyle: ps.computedStyle,
		Viewport:      ps.viewport,
		MediaMatches:  func(q string) bool { return ps.context().MediaMatches(q) },
		Navigate:      ps.navigate,
		Submit:        ps.submit,
		PushState:     ps.pushState,
		Storage:       localStorage,
		Cookies:       cookies,
		Log:           ps.log,
		Parent:        pg.parentEngine,
		FrameOf:       pg.frameEngine,
	})
	if err != nil {
		return nil, err
	}
	ps.engine = e
	return ps, nil
}

// context returns the context that the document was last read with.
func (ps *pageScripts) context() *htmlcore.Context {
	if ps.ctx != nil {
		return ps.ctx
	}
	return ps.base
}

// log records the given console message of the page.
func (ps *pageScripts) log(level slog.Level, msg string) {
	slog.Debug("web page script", "level", level, "msg", msg)
	ps.consoleMu.Lock()
	defer ps.consoleMu.Unlock()
	if len(ps.console) >= maxConsole {
		ps.console = ps.console[1:]
	}
	ps.console = append(ps.console, level.String()+": "+msg)
	go ps.pg.consoleChanged() // (not with consoleMu)
}

// read reads the document into a new frame for the content of the page,
// with a new context for the current width of the viewport.
func (ps *pageScripts) read() (*core.Frame, error) {
	ctx := ps.base.NewPage()
	ctx.PageURL = ps.engine.PageURL()
	ctx.GetURL = ps.base.GetURL
	ctx.ImageCache = ps.base.ImageCache
	ctx.MediaWidth = ps.width
	ctx.OpenURL = ps.openURL
	ctx.OpenLink = ps.openLink
	ctx.Checked = func(n *html.Node) bool { // the state from the user and the scripts
		if _, _, checked, ok := ps.engine.ControlState(n); ok {
			return checked
		}
		return htmlcore.HasAttr(n, "checked")
	}
	ctx.Hovered = func(n *html.Node) bool { return ps.hoverSet[n] }
	// the children of shadow hosts that are in no slot are not rendered
	ctx.Unrendered = ps.engine.Unassigned
	// details are toggled in the document, which is read again
	ctx.ToggleDetails = func(n *html.Node) bool {
		if ps.engine.Closed() { // in place, as the scripts were stopped
			return false
		}
		// the click on the summary is dispatched to the scripts, whose
		// default action toggles the details (once, unless prevented)
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			if k.Type == html.ElementNode && k.Data == "summary" {
				ps.engine.Dispatch(k, "click", nil, nil)
				break
			}
		}
		return true
	}
	ctx.WidgetHandlers = append(append([]func(core.Widget){}, ps.base.WidgetHandlers...), ps.handleWidget)
	content := newPageContent()
	err := htmlcore.ReadDocument(ctx, content, ps.doc)
	ps.ctx = ctx
	ps.widgets = nil
	ps.laidOut = false
	return content, err
}

// render reads the document into new widgets after the scripts have
// changed it (without the lock of the page, as reading a page takes time,
// but while no script runs), and returns the function that shows them in
// the page (with the lock held).
func (ps *pageScripts) render() func() {
	if !ps.shown {
		ps.pending = true
		return nil
	}
	// the scripts get a resize event if the viewport has another size
	// than they got (as it may not have its size while the page loads)
	if ps.seen != (math32.Vector2{}) {
		if w, h := ps.viewportSize(); math32.Vec2(w, h) != ps.seen {
			ps.engine.Resized()
		}
	}
	if ps.layoutMissed {
		// the scripts got the boxes of elements before the page had a
		// layout (as the slider of the photos of the postings of
		// craigslist.org, which was as wide as nothing), and they get a
		// resize event to get them again, now that it has one
		ps.layoutMissed = false
		ps.engine.Resized()
	}
	start := time.Now()
	content, err := ps.read()
	if err != nil {
		ps.log(slog.LevelError, "reading the page: "+err.Error())
		return nil
	}
	ps.renders++
	ps.renderTime += time.Since(start)
	ctx := ps.ctx
	requested := ps.engine.TakeFocus() // which a script focused
	return func() {
		pg := ps.pg
		if pg.scripts != ps {
			return // another page is shown
		}
		if pg.media != nil {
			pg.media.ctx = ctx
			// the width is checked again for the media queries of the new
			// context (see Page.checkMedia), which may have more than the
			// one that was checked (as those of a style sheet that loads
			// after the page is shown, as of the postings of craigslist.org,
			// which were laid out for the width of the screen until the
			// window was resized)
			pg.checkedWidth = ctx.MediaWidth
		}
		// the element that has the focus keeps it in the new widgets, and
		// the elements that scroll keep their scroll positions (as the
		// sidebar of MDN, which scrolled to its top when an item opened)
		var focused *html.Node
		scrolls := map[*html.Node]math32.Vector2{}
		pg.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			if cwb.StateIs(states.Focused) {
				focused = htmlcore.NodeOf(cw)
			}
			if fr := core.AsFrame(cw); fr != nil && (fr.HasScroll[math32.X] || fr.HasScroll[math32.Y]) && fr.Geom.Scroll != (math32.Vector2{}) {
				if n := htmlcore.NodeOf(cw); n != nil {
					scrolls[n] = fr.Geom.Scroll
				}
			}
			return tree.Continue
		})
		if requested != nil {
			focused = requested
		}
		pageScroll := pg.Geom.Scroll
		pg.Title = ctx.Title
		pg.setContent(content)
		content.Defer(func() {
			restoreScrolls(content, scrolls)
			for d := math32.X; d <= math32.Y; d++ {
				if pg.HasScroll[d] && pg.Geom.Scroll.Dim(d) != pageScroll.Dim(d) {
					pg.Geom.Scroll.SetDim(d, pageScroll.Dim(d))
					pg.ScrollUpdateFromGeom(d)
				}
			}
		})
		if focused != nil {
			content.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
				if n, ok := cwb.Property("node").(*html.Node); ok && n == focused {
					cwb.SetFocus()
					if tf, ok := cw.(*core.TextField); ok {
						tf.SetCursorEnd() // to type on
					}
					return tree.Break
				}
				return tree.Continue
			})
		}
	}
}

// handleWidget sets up the given widget of the page (as it is read) for
// the events of its element: clicks for the elements that have listeners
// for them (or are in elements that do), and the values of form controls.
// handleTextClicks sends the clicks on the given text to the listeners of
// the element that it is in (and of the elements that that is in, and the
// document), as those that close the menus of Wikipedia on clicks outside
// of them. Its links open as they do.
func (ps *pageScripts) handleTextClicks(tx *core.Text) {
	n := htmlcore.NodeOf(tx)
	if n == nil || n.Type != html.ElementNode {
		return
	}
	// a text in a label of a control has the default action of clicking it
	// (as a label of a hidden checkbox that folds a thread of comments)
	inLabel := false
	for p := n; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.Data == "label" {
			inLabel = htmlcore.HasAttr(p, "for") || hasDescendant(p, "input")
			break
		}
	}
	if !inLabel && !ps.engine.HasListener(n, "click") {
		return
	}
	// a text in a link (as the items of the jump dialog of pkg.go.dev, a
	// link of a text and an i element): the click is dispatched to it, and
	// then the link is followed, unless it is prevented
	for a := n; a != nil; a = a.Parent {
		if a.Type == html.ElementNode && a.Data == "a" && htmlcore.HasAttr(a, "href") {
			href, link := htmlcore.GetAttr(a, "href"), a
			tx.OnClick(func(ev events.Event) {
				ev.SetHandled()
				ps.openLink(link, href, ev)
			})
			return
		}
	}
	details := summaryDetails(n)
	tx.OnClick(func(ev events.Event) {
		if ps.engine.Closed() {
			if details != nil { // the summary opens its details in place
				for p := core.AsWidget(tx.Parent); p != nil; p = core.AsWidget(p.Parent) {
					if pn := htmlcore.NodeOf(p.This.(core.Widget)); pn != nil && pn.Data == "summary" {
						p.Send(events.Click, ev)
						break
					}
				}
			}
			return
		}
		// (the default action of a click on the text of a summary, which
		// gets the click instead of it, toggles its details in the scripts)
		ps.engine.Dispatch(n, "click", ps.pointerProps(ev), nil)
	})
}

// hoverAt dispatches the events of the mouse moving over the elements when
// the element under the given position in the window (outside of the page
// if it is not in it) differs from the one that it was over: the deepest
// widget there that is the box of an element (not of a pseudo-element),
// in the order of rendering (later on top).
func (ps *pageScripts) hoverAt(pos image.Point) {
	if ps.engine.Closed() {
		return
	}
	deepest, _ := ps.nodeAt(pos)
	if deepest == ps.hovered {
		return
	}
	from := ps.hovered
	ps.hovered = deepest
	x, y := ps.clientPos(pos)
	ps.engine.Hover(from, deepest, x, y)
	ps.engine.Post(func() { ps.setHovered(deepest) })
}

// nodeAt returns the deepest element of the page under the given position
// in the window (nil if it is not in the page), and whether the position
// is in the page of a frame of the page, whose iframe element it then
// returns, as the page in the frame gets the events of the mouse there,
// as in browsers.
func (ps *pageScripts) nodeAt(pos image.Point) (deepest *html.Node, framed bool) {
	var visit func(wb *core.WidgetBase)
	visit = func(wb *core.WidgetBase) {
		if n := htmlcore.NodeOf(wb.This.(core.Widget)); n != nil && n.Type == html.ElementNode && htmlcore.PseudoElement(n) == "" {
			deepest = n
		}
		kids := wb.Children
		for i := len(kids) - 1; i >= 0; i-- {
			kw := core.AsWidget(kids[i])
			if kw == nil || kw.StateIs(states.Invisible) || (kw.Styles.Opacity <= 0 && kw.Styles.Abilities == 0) {
				continue
			}
			if pos.In(kw.Geom.TotalBBox) {
				if _, isFrame := kw.This.(*Page); isFrame {
					framed = true // (of the iframe that it is in)
					return
				}
				visit(kw)
				return
			}
		}
	}
	if pos.In(ps.pg.Geom.TotalBBox) {
		visit(ps.pg.AsWidget())
	}
	return
}

// clientPos returns the given position in the window in the CSS px of the
// page, relative to its top left, as the scripts get the positions of the
// mouse (as clientX and clientY).
func (ps *pageScripts) clientPos(pos image.Point) (x, y float32) {
	dp := ps.pg.pxDots()
	if dp <= 0 {
		dp = 1
	}
	off := ps.pg.Geom.Pos.Content
	return (float32(pos.X) - off.X) / dp, (float32(pos.Y) - off.Y) / dp
}

// pressAt dispatches the events of a mouse button that is pressed (down)
// or released at the given position in the window to the element under it
// (pointerdown and mousedown, or pointerup and mouseup), as for the
// controls of scripts that act on them rather than on clicks (as the
// checkbox of reCAPTCHA, which did nothing).
func (ps *pageScripts) pressAt(ev events.Event, down bool) {
	if ps.engine.Closed() {
		return
	}
	n, framed := ps.nodeAt(ev.WindowPos())
	if n == nil || framed {
		return
	}
	ps.engine.Press(n, down, ps.pointerProps(ev))
}

// pointerProps returns the properties of a mouse event for the given event
// of the mouse: its modifier keys, its position, and its (left) button.
func (ps *pageScripts) pointerProps(ev events.Event) map[string]any {
	props := mouseProps(ev)
	x, y := ps.clientPos(ev.WindowPos())
	for _, k := range []string{"clientX", "pageX", "screenX", "x"} {
		props[k] = x
	}
	for _, k := range []string{"clientY", "pageY", "screenY", "y"} {
		props[k] = y
	}
	props["button"] = 0
	props["detail"] = 1
	return props
}

// setHovered sets the elements that the mouse is over to the given one and
// those it is in (in a task of the engine), reading the page again if the
// hover of any element that changes affects the styles of others (as the
// menu items of pkg.go.dev, whose submenus are shown on their :hover).
func (ps *pageScripts) setHovered(n *html.Node) {
	if ps.ctx == nil {
		return
	}
	// only the elements whose hover affects others are marked, as the
	// page is only read again for them (a link would otherwise keep its
	// :hover style when the page is read again for something else)
	set := map[*html.Node]bool{}
	for ; n != nil; n = n.Parent {
		if ps.ctx.HoverAffects(n) {
			set[n] = true
		}
	}
	changed := len(set) != len(ps.hoverSet)
	for n := range set {
		changed = changed || !ps.hoverSet[n]
	}
	ps.hoverSet = set
	if changed {
		ps.engine.Changed()
	}
}

// restoreScrolls restores the given scroll positions of the elements that
// scroll in the given content (read again), after its layout.
func restoreScrolls(content *core.Frame, scrolls map[*html.Node]math32.Vector2) {
	content.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		fr := core.AsFrame(cw)
		if fr == nil {
			return tree.Continue
		}
		pos, ok := scrolls[htmlcore.NodeOf(cw)]
		if !ok {
			return tree.Continue
		}
		for d := math32.X; d <= math32.Y; d++ {
			if fr.HasScroll[d] {
				fr.Geom.Scroll.SetDim(d, pos.Dim(d))
				fr.ScrollUpdateFromGeom(d)
			}
		}
		return tree.Continue
	})
}

// summaryDetails returns the details element whose summary the given
// element is in (not in a link or button in it), or nil.
func summaryDetails(n *html.Node) *html.Node {
	for ; n != nil; n = n.Parent {
		if n.Type != html.ElementNode {
			continue
		}
		switch n.Data {
		case "a", "button", "input", "select", "textarea":
			return nil
		case "summary":
			if p := n.Parent; p != nil && p.Type == html.ElementNode && p.Data == "details" {
				return p
			}
			return nil
		}
	}
	return nil
}

func (ps *pageScripts) handleWidget(w core.Widget) {
	wb := w.AsWidget()
	if tx, ok := w.(*core.Text); ok {
		ps.handleTextClicks(tx)
		return
	}
	n, ok := wb.Property("node").(*html.Node)
	if !ok || n == nil || n.Type != html.ElementNode {
		return
	}
	if htmlcore.PseudoElement(n) != "" {
		// the clicks on a pseudo-element are those of its element (as the
		// chevrons of the sidebar of MDN, which took the clicks)
		return
	}
	e := ps.engine
	if form := ancestorForm(n); form != nil && e.HasListener(form, "submit") || isSubmitter(n) && e.HasListener(n, "click") {
		// the scripts get the click of a submit button, and the submit
		// event of the form, which they can prevent (as to send it
		// themselves); otherwise the form is submitted as it is in HTML
		switch {
		case isSubmitter(n):
			wb.OnClick(func(ev events.Event) {
				ev.SetHandled()
				e.Dispatch(n, "click", ps.pointerProps(ev), func(prevented bool) {
					if !prevented {
						e.Activate(n)
					}
				})
			})
			return
		case w.AsWidget().Property("tag") == "input":
			if tf, ok := w.(*core.TextField); ok {
				tf.OnFirst(events.KeyChord, func(ev events.Event) {
					if kf := keymap.Of(ev.KeyChord()); kf == keymap.Enter || kf == keymap.Accept {
						ev.SetHandled()
						e.Dispatch(n, "change", map[string]any{"value": tf.Text()}, func(bool) { e.Activate(n) })
					}
				})
			}
		}
	}
	// the state of the controls that the user or the scripts have changed
	if value, hasValue, checked, hasChecked := e.ControlState(n); hasValue || hasChecked {
		switch x := w.(type) {
		case *core.TextField:
			if hasValue {
				x.SetText(value)
			}
		case *core.Switch:
			if hasChecked {
				x.SetChecked(checked)
			}
		case *core.Chooser:
			if hasValue {
				x.SetCurrentValue(value)
			}
		}
	}
	switch x := w.(type) {
	case *core.TextField:
		// the keys typed in it: keydown before the field gets the key, and
		// keyup after it (with its text), as for the filters that pages
		// update on keyup (as the jump dialog of pkg.go.dev)
		x.OnFirst(events.KeyChord, func(ev events.Event) {
			props := keyProps(ev)
			e.Dispatch(n, "keydown", props, nil)
			x.Defer(func() {
				up := maps.Clone(props)
				up["value"] = x.Text()
				e.Dispatch(n, "keyup", up, nil)
			})
		})
		x.OnInput(func(ev events.Event) {
			e.Dispatch(n, "input", map[string]any{"value": x.Text()}, nil)
		})
		x.OnChange(func(ev events.Event) {
			e.Dispatch(n, "change", map[string]any{"value": x.Text()}, nil)
		})
		return
	case *core.Switch:
		x.OnChange(func(ev events.Event) {
			checked := x.IsChecked()
			e.Dispatch(n, "click", map[string]any{"checked": checked}, func(bool) {
				// the page is read again, for the :checked selectors (as the
				// CSS menus of Wikipedia, which show when a checkbox is checked)
				e.Changed()
				e.Dispatch(n, "input", nil, nil)
				e.Dispatch(n, "change", nil, nil)
			})
		})
		return
	case *core.Chooser:
		x.OnChange(func(ev events.Event) {
			e.Dispatch(n, "input", map[string]any{"value": fmt.Sprint(x.CurrentItem.Value)}, nil)
			e.Dispatch(n, "change", map[string]any{"value": fmt.Sprint(x.CurrentItem.Value)}, nil)
		})
		return
	case *core.Frame, *core.Button: // buttons that are not submitters (as the toggles of menus)
	default:
		return
	}
	// a label of a control has the default action of clicking it (as the
	// [-] labels of the comments of lobste.rs, of hidden checkboxes)
	label := n.Data == "label" && (htmlcore.HasAttr(n, "for") || hasDescendant(n, "input"))
	if n.Data == "a" || n.Data == "summary" || (!label && !e.HasListener(n, "click")) {
		// links open through openURL, and summaries dispatch their clicks
		// through Context.ToggleDetails (once)
		return
	}
	own := hasOwnClickHandler(n)
	wb.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Clickable)
		if own {
			s.Cursor = cursors.Pointer
		}
	})
	wb.FinalStyler(func(s *styles.Style) {
		if !own && !s.Is(states.Hovered) {
			// not highlighted when pressed, as it is only in an element with
			// a listener, but it shows a hover that it has (as the label of
			// a transparent checkbox over it, as the menus of Wikipedia)
			s.StateLayer = 0
		}
	})
	// a click in a link opens it, unless a listener prevents its default
	// (as in browsers), as for the boxes in a link (as the results of
	// craigslist), which get the clicks instead of the link
	href := ""
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.Data == "a" {
			href = attrOf(p, "href")
			break
		}
	}
	wb.OnClick(func(ev events.Event) {
		ev.SetHandled()
		if e.Closed() { // the scripts were stopped
			ps.followLink(href, ev)
			return
		}
		e.Dispatch(n, "click", ps.pointerProps(ev), func(prevented bool) {
			if !prevented {
				ps.followLink(href, ev)
			}
		})
	})
}

// followLink opens the given URL of a link (if not ""), or runs its
// script for a javascript: URL (unless the scripts were stopped).
func (ps *pageScripts) followLink(href string, ev events.Event) {
	if href == "" {
		return
	}
	if js, ok := cutPrefixFold(strings.TrimSpace(href), "javascript:"); ok {
		if !ps.engine.Closed() {
			ps.engine.Eval(js)
		}
		return
	}
	ps.pg.OpenURL(href, ev)
}

// cutPrefixFold is [strings.CutPrefix], ignoring the case of the prefix.
func cutPrefixFold(s, prefix string) (string, bool) {
	if len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix) {
		return s[len(prefix):], true
	}
	return s, false
}

// attrOf returns the value of the given attribute of the given node,
// or "" if it does not have it.
func attrOf(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key && a.Namespace == "" {
			return a.Val
		}
	}
	return ""
}

// hasOwnClickHandler returns whether the given element has an onclick
// attribute, as for the pointer cursor (listeners are not known here).
func hasOwnClickHandler(n *html.Node) bool {
	for _, a := range n.Attr {
		if a.Key == "onclick" {
			return true
		}
	}
	return n.Data == "button"
}

// mouseProps returns the properties of a mouse event for the given event.
func mouseProps(ev events.Event) map[string]any {
	return map[string]any{
		"ctrlKey": ev.HasAnyModifier(key.Control), "metaKey": ev.HasAnyModifier(key.Meta),
		"shiftKey": ev.HasAnyModifier(key.Shift), "altKey": ev.HasAnyModifier(key.Alt),
	}
}

// hasDescendant returns whether the given node has an element of the
// given tag in it.
func hasDescendant(n *html.Node, tag string) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (c.Data == tag || hasDescendant(c, tag)) {
			return true
		}
	}
	return false
}

// keyProps returns the properties of a KeyboardEvent for the given key
// event: its key, code, and keyCode (and which), and its modifiers.
func keyProps(ev events.Event) map[string]any {
	props := mouseProps(ev)
	kc, ok := ev.(*events.Key)
	if !ok {
		return props
	}
	named := map[key.Codes][2]any{
		key.CodeReturnEnter: {"Enter", 13}, key.CodeKeypadEnter: {"Enter", 13}, key.CodeEscape: {"Escape", 27},
		key.CodeTab: {"Tab", 9}, key.CodeBackspace: {"Backspace", 8}, key.CodeDelete: {"Delete", 46},
		key.CodeUpArrow: {"ArrowUp", 38}, key.CodeDownArrow: {"ArrowDown", 40},
		key.CodeLeftArrow: {"ArrowLeft", 37}, key.CodeRightArrow: {"ArrowRight", 39},
		key.CodeHome: {"Home", 36}, key.CodeEnd: {"End", 35}, key.CodePageUp: {"PageUp", 33}, key.CodePageDown: {"PageDown", 34},
	}
	if v, ok := named[kc.Code]; ok {
		props["key"], props["code"], props["keyCode"] = v[0], v[0], v[1]
		return props
	}
	if r := kc.Rune; r > 0 {
		props["key"] = string(r)
		code := int(unicode.ToUpper(r))
		if code > 0xff {
			code = 0
		}
		props["keyCode"] = code
		switch {
		case unicode.IsLetter(r) && r < 0x80:
			props["code"] = "Key" + string(unicode.ToUpper(r))
		case unicode.IsDigit(r) && r < 0x80:
			props["code"] = "Digit" + string(r)
		case r == ' ':
			props["code"] = "Space"
		}
	}
	return props
}

// openURL opens the given URL of a link that is clicked, after its click
// event is dispatched to its element (the first a element of the document
// for the URL), unless a listener prevents its default.
func (ps *pageScripts) openURL(url string, ev events.Event) {
	pg := ps.pg
	js := strings.HasPrefix(strings.ToLower(strings.TrimSpace(url)), "javascript:")
	if ps.engine.Closed() {
		// the scripts were stopped (as for running too long): links are
		// followed as without scripts
		if !js {
			pg.OpenURL(url, ev)
		}
		return
	}
	anchor := ps.anchorFor(url)
	if anchor == nil {
		if js {
			ps.engine.Eval(strings.TrimSpace(url)[len("javascript:"):])
			return
		}
		pg.OpenURL(url, ev)
		return
	}
	ps.openLink(anchor, url, ev)
}

// openLink dispatches a click to the given link element, and then opens
// the given URL of it (or runs it, for javascript:), unless the click is
// prevented. It is used for the clicks on links (see
// [htmlcore.Context.OpenLink]), whose own element gets the click (as the
// items of the jump dialog of pkg.go.dev, which close it on their click,
// not the first link to the same URL in the page).
func (ps *pageScripts) openLink(link *html.Node, url string, ev events.Event) {
	pg := ps.pg
	js := strings.HasPrefix(strings.ToLower(strings.TrimSpace(url)), "javascript:")
	if ps.engine.Closed() {
		if !js {
			pg.OpenURL(url, ev)
		}
		return
	}
	props := map[string]any{}
	if ev != nil {
		props = ps.pointerProps(ev)
	}
	ps.engine.Dispatch(link, "click", props, func(prevented bool) {
		if prevented {
			return
		}
		if js {
			ps.engine.Eval(strings.TrimSpace(url)[len("javascript:"):])
			return
		}
		pg.OpenURL(url, ev)
	})
}

// anchorFor returns the first a element of the document whose href is the
// given URL (resolved), or nil if there is none.
func (ps *pageScripts) anchorFor(url string) *html.Node {
	base := ps.engine.PageURL()
	var found *html.Node
	var walk func(n *html.Node) bool
	walk = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key != "href" {
					continue
				}
				if a.Val == url {
					found = n
					return false
				}
				if u, err := htmlcore.ParseRelativeURL(a.Val, base); err == nil && u.String() == url {
					found = n
					return false
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if !walk(c) {
				return false
			}
		}
		return true
	}
	walk(ps.doc)
	return found
}

// navigate opens the given URL, as for location.href (in another
// goroutine than that of the scripts).
func (ps *pageScripts) navigate(url string, replace bool) {
	pg := ps.pg
	if !pg.AsyncLockIfShown() {
		return
	}
	defer pg.AsyncUnlock()
	if pg.scripts != ps {
		return
	}
	if replace && pg.Current < len(pg.History) {
		pg.open(url, historyAt, pg.Current, 0)
		return
	}
	pg.OpenURL(url, nil)
}

// submit opens the page of the given URL with a POST request with the
// given body (as for a form), in another goroutine than that of the scripts.
func (ps *pageScripts) submit(url, contentType string, body []byte) {
	pg := ps.pg
	if !pg.AsyncLockIfShown() {
		return
	}
	defer pg.AsyncUnlock()
	if pg.scripts != ps {
		return
	}
	pg.openPost(url, &postRequest{contentType: contentType, body: body}, historyNew, 0, 0)
}

// pushState changes the URL of the page without opening it, as for
// history.pushState (in a task).
func (ps *pageScripts) pushState(url string, replace bool) {
	if !ps.shown {
		ps.pushed = url
		return
	}
	pg := ps.pg
	if pg.scripts != ps || pg.Current >= len(pg.History) {
		return
	}
	pg.History[pg.Current] = url
	pg.Context.PageURL = url
	pg.opened(url)
}

// viewport returns the size of the viewport of the page in CSS px, for
// the scripts, recording it (see pageScripts.seen).
func (ps *pageScripts) viewport() (float32, float32) {
	w, h := ps.viewportSize()
	ps.seen = math32.Vec2(w, h)
	return w, h
}

// viewportSize returns the width and height of the viewport in CSS px.
func (ps *pageScripts) viewportSize() (float32, float32) {
	pg := ps.pg
	w := ps.width
	h := float32(768)
	if dp := pg.pxDots(); dp > 0 {
		if fr := core.AsFrame(pg.Parent); fr != nil && fr.Geom.Size.Alloc.Content.Y > 0 {
			h = fr.Geom.Size.Alloc.Content.Y / dp
		}
	}
	return w, h
}

// layout returns the position and size of the box of the given node in
// CSS px, relative to the top left of the page, laying out the page if it
// has not been since it was rendered (in a task, with the lock held).
func (ps *pageScripts) layout(n *html.Node) (x, y, w, h float32, ok bool) {
	pg := ps.pg
	if !ps.shown || pg.scripts != ps || pg.Scene == nil {
		// the page has no layout yet (it is shown once it has loaded), and
		// the scripts get a resize event once it has one (see render)
		ps.layoutMissed = true
		return
	}
	wb := ps.widgetOf(n)
	if wb == nil {
		if ps.renders == 0 {
			// an element that the scripts made while the page loads, which
			// has no widget until the page is rendered (see layoutMissed)
			ps.layoutMissed = true
		}
		return
	}
	if !ps.laidOut {
		pg.Scene.LayoutScene()
		ps.laidOut = true
	}
	if wb.Styles.Display == styles.DisplayNone || wb.StateIs(states.Invisible) {
		return
	}
	dp := pg.pxDots()
	if dp <= 0 {
		return
	}
	pos := wb.Geom.Pos.Total.Sub(pg.Geom.Pos.Content)
	size := wb.Geom.Size.Actual.Total
	return pos.X / dp, pos.Y / dp, size.X / dp, size.Y / dp, true
}

// elementAt returns the element that a click at the given position in CSS
// px (relative to the top left of the page, as for layout) goes to: that
// of the widget that gets the events there (see core.Scene.WidgetAt), or
// of the nearest widget that it is in that has an element of the page.
func (ps *pageScripts) elementAt(x, y float32) *html.Node {
	pg := ps.pg
	if !ps.shown || pg.scripts != ps || pg.Scene == nil {
		return nil
	}
	dp := pg.pxDots()
	if dp <= 0 {
		return nil
	}
	pos := pg.Geom.Pos.Content.Add(math32.Vec2(x, y).MulScalar(dp))
	w := pg.Scene.WidgetAt(pos.ToPointFloor())
	for w != nil {
		wb := w.AsWidget()
		if wb == pg.AsWidget() {
			return nil
		}
		if n, ok := wb.Property("node").(*html.Node); ok && n != nil && n.Parent != nil {
			for n != nil && n.Type != html.ElementNode {
				n = n.Parent // the element of a text
			}
			if n != nil {
				return n
			}
		}
		p, _ := wb.Parent.(core.Widget)
		w = p
	}
	return nil
}

// colorProperties are the CSS properties whose computed values are
// colors, which browsers report as rgb() or rgba().
var colorProperties = map[string]bool{
	"color": true, "background-color": true, "border-color": true, "border-top-color": true,
	"border-right-color": true, "border-bottom-color": true, "border-left-color": true,
	"outline-color": true, "fill": true, "stroke": true, "text-decoration-color": true,
	"caret-color": true, "column-rule-color": true, "stop-color": true,
}

// computedStyle returns the computed value of the given CSS property of the
// given element, as browsers report it, from the style sheets of the page
// as it was last read (see [script.Options.ComputedStyle]): colors as rgb()
// (black for the default color), and font sizes in px, from its widget.
func (ps *pageScripts) computedStyle(n *html.Node, prop string) string {
	v := ps.context().ComputedValue(n, prop)
	switch {
	case prop == "font-size":
		if wb := ps.widgetOf(n); wb != nil {
			if dp := ps.pg.pxDots(); dp > 0 && wb.Styles.Font.Size.Dots > 0 {
				return strconv.FormatFloat(float64(wb.Styles.Font.Size.Dots/dp), 'f', -1, 32) + "px"
			}
		}
	case colorProperties[prop]:
		if v == "" {
			if prop != "color" {
				return ""
			}
			v = "black"
		}
		return cssColor(v)
	}
	return v
}

// cssColor returns the given CSS color as browsers report computed colors:
// rgb(r, g, b), or rgba(r, g, b, a) if it is not opaque, or the color
// itself if it is not one that can be parsed (as a gradient).
func cssColor(v string) string {
	c, err := colors.FromString(v)
	if err != nil {
		return v
	}
	r, g, b, a := colors.AsRGBA(c).R, colors.AsRGBA(c).G, colors.AsRGBA(c).B, colors.AsRGBA(c).A
	if a == 255 {
		return fmt.Sprintf("rgb(%d, %d, %d)", r, g, b)
	}
	if a == 0 {
		return "rgba(0, 0, 0, 0)"
	}
	// the colors are premultiplied by alpha
	un := func(x uint8) int { return int(math32.Round(float32(x) * 255 / float32(a))) }
	// the alpha with 2 decimals if they give the same 8 bit alpha, else 3, as Chrome does
	af := math.Round(float64(a)/255*100) / 100
	if uint8(math.Round(af*255)) != a {
		af = math.Round(float64(a)/255*1000) / 1000
	}
	return fmt.Sprintf("rgba(%d, %d, %d, %s)", un(r), un(g), un(b), strconv.FormatFloat(af, 'f', -1, 64))
}

// widgetOf returns the widget of the given element, if it has one.
func (ps *pageScripts) widgetOf(n *html.Node) *core.WidgetBase {
	pg := ps.pg
	if !ps.shown || pg.scripts != ps || pg.Scene == nil {
		return nil
	}
	if ps.widgets == nil {
		ps.widgets = map[*html.Node]*core.WidgetBase{}
		pg.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			if nd, ok := cwb.Property("node").(*html.Node); ok && nd != nil {
				if _, has := ps.widgets[nd]; !has {
					ps.widgets[nd] = cwb
				}
			}
			return tree.Continue
		})
	}
	return ps.widgets[n]
}

// openScripts opens the given page (gotten with the given context from
// the given final URL) with its scripts: it parses it with scripting,
// shows it as it is parsed (as browsers show pages while their scripts
// load), and then runs its scripts, after which (and as they change it)
// it is rendered again, with the given change to the history. It returns
// false if its scripts cannot be run, for the page to be read without them.
func (pg *Page) openScripts(ctx *htmlcore.Context, page []byte, final string, mode historyMode, index int, cache *resourceCache, canceled func() bool) bool {
	ctx.Scripting = true
	ctx.ImageCache = htmlcore.NewImageCache(64 << 20)
	doc, err := htmlcore.ParseHTML(ctx, bytes.NewReader(page))
	if err != nil {
		ctx.Scripting = false
		return false
	}
	ps, err := pg.newPageScripts(ctx, doc, final)
	if err != nil {
		slog.Error("web: running the scripts of the page", "url", final, "err", err)
		ctx.Scripting = false
		return false
	}
	content, err := ps.read() // no script has run yet
	if canceled() {
		ps.engine.Close()
		return true
	}
	<-pg.shown // AsyncLock is not safe before then
	pg.AsyncLock()
	defer pg.AsyncUnlock()
	if canceled() {
		ps.engine.Close()
		return true
	}
	pg.loading = ""
	if err != nil {
		core.ErrorSnackbar(pg, err, "Error opening page")
		pg.opened(pg.currentURL())
		ps.engine.Close()
		pg.frameLoaded()
		return true
	}
	// a page that redirects is replaced by the page that it redirects to
	if ref := ps.ctx.RefreshURL; ref != "" {
		if u, err := htmlcore.ParseRelativeURL(ref, final); err == nil && u.String() != final {
			ps.engine.Close()
			pg.open(u.String(), mode, index, 1)
			return true
		}
	}
	pg.setHistory(final, mode, index)
	pg.Context.PageURL = final
	pg.Title = ps.ctx.Title
	pg.setMedia(&pageMedia{ctx: ps.ctx, html: page, url: final, cache: cache})
	pg.showScripts(ps)
	pg.setContent(content)
	if u, err := neturl.Parse(final); err == nil && u.Fragment != "" {
		pg.Defer(func() { pg.scrollToFragment(u.Fragment) })
	} else if mode == historyNew {
		pg.scrollToTop()
	}
	pg.opened(final)
	ps.engine.Load(pg.frameLoaded) // rendered again after the load event
	return true
}

// showScripts makes the given scripts those of the page that is shown,
// stopping those of the page before (with the lock of the page held, in a
// task of the given scripts).
func (pg *Page) showScripts(ps *pageScripts) {
	if pg.scripts != nil && pg.scripts != ps {
		pg.scripts.engine.Close()
	}
	pg.scripts = ps
	if ps == nil {
		return
	}
	ps.shown = true
	ps.engine.SetLock(pg.AsyncLockIfShown, pg.AsyncUnlock)
	if ps.pending {
		ps.pending = false
		ps.engine.Changed()
	}
}

// ScriptConsole returns the console messages of the scripts of the page
// that is shown (and their errors), oldest first, followed by those of the
// pages in its frames, each marked with the URL of its frame (as the
// captchas of bot checks, whose errors were not shown).
func (pg *Page) ScriptConsole() []string {
	var msgs []string
	if ps := pg.scripts; ps != nil {
		ps.consoleMu.Lock()
		msgs = append(msgs, ps.console...)
		ps.consoleMu.Unlock()
	}
	for _, fp := range pg.frames {
		for _, m := range fp.ScriptConsole() {
			if !strings.HasPrefix(m, "[frame ") {
				src, _, _ := strings.Cut(fp.frame.src, "?") // (without its long query)
				m = "[frame " + src + "] " + m
			}
			msgs = append(msgs, m)
		}
	}
	return msgs
}

// showConsole shows the console messages of the scripts of the page.
func (pg *Page) showConsole() {
	msgs := pg.ScriptConsole()
	all := strings.Join(msgs, "\n")
	if all == "" {
		all = "No messages"
	}
	d := core.NewBody("Script console")
	core.NewText(d).SetText(pg.currentURL()).Styler(func(s *styles.Style) { s.Font.Size.Dp(12) })
	tx := core.NewText(d).SetText(html.EscapeString(all))
	tx.Styler(func(s *styles.Style) {
		s.Font.Family = rich.Monospace
		s.Text.WhiteSpace = text.WhiteSpacePreWrap
	})
	d.AddOKOnly().RunWindowDialog(pg)
}

// ancestorForm returns the form element that the given node is in, if any.
func ancestorForm(n *html.Node) *html.Node {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.Data == "form" {
			return p
		}
	}
	return nil
}

// isSubmitter returns whether the given element submits its form when it
// is clicked (a submit button).
func isSubmitter(n *html.Node) bool {
	typ := ""
	for _, a := range n.Attr {
		if a.Key == "type" {
			typ = strings.ToLower(strings.TrimSpace(a.Val))
		}
	}
	switch n.Data {
	case "button":
		return typ == "" || typ == "submit"
	case "input":
		return typ == "submit" || typ == "image"
	}
	return false
}
