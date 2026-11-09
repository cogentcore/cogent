// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/tree"
)

// ConsolePanel shows the console messages of the scripts of the page of
// the current tab of a [Browser] (and of the pages in its frames, and its
// workers), below its tabs, in a split that can be resized, as the
// console of the developer tools of browsers. It is updated as messages
// come (see [Browser.ShowConsole]).
type ConsolePanel struct {
	core.Frame

	// browser is the browser that the panel is in.
	browser *Browser

	// list is the frame of the messages.
	list *core.Frame

	// msgs are the messages that are shown.
	msgs []string

	// pending is whether an update of the panel is scheduled.
	pending atomic.Bool
}

// consoleDelay is how long the panel waits after a message before it is
// updated, so that the messages that come together are shown together.
const consoleDelay = 150 * time.Millisecond

func (cp *ConsolePanel) Init() {
	cp.Frame.Init()
	cp.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
		s.Padding.Zero()
		s.Gap.Zero()
		s.Border.Width.Top.Dp(1)
		s.Border.Color.Top = colors.Scheme.OutlineVariant
	})
	bar := core.NewFrame(cp)
	bar.Styler(func(s *styles.Style) {
		s.Align.Items = styles.Center
		s.Padding.Set(units.Dp(2), units.Dp(8))
		s.Grow.Set(1, 0)
	})
	core.NewText(bar).SetText("Console").SetType(core.TextTitleSmall)
	core.NewStretch(bar)
	core.NewButton(bar).SetIcon(icons.Delete).SetType(core.ButtonAction).SetTooltip("Clear the console").OnClick(func(e events.Event) {
		if pg := cp.browser.CurrentPage(); pg != nil {
			pg.clearConsole()
		}
		cp.refresh()
	})
	core.NewButton(bar).SetIcon(icons.Close).SetType(core.ButtonAction).SetTooltip("Close the console").OnClick(func(e events.Event) {
		cp.browser.ShowConsole(false)
	})
	cp.list = core.NewFrame(cp)
	cp.list.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
		s.Overflow.Set(styles.OverflowAuto)
		s.Padding.Set(units.Dp(0), units.Dp(8))
		s.Gap.Zero()
	})
	cp.list.Maker(func(p *tree.Plan) {
		if len(cp.msgs) == 0 {
			tree.AddAt(p, "none", func(w *core.Text) {
				w.SetText("No messages")
				w.Styler(func(s *styles.Style) { s.Color = colors.Scheme.OnSurfaceVariant })
			})
			return
		}
		for i := range cp.msgs {
			// (the widgets are kept by index as the messages change, so
			// they get their message when they are updated)
			msg := func() string {
				if i < len(cp.msgs) {
					return cp.msgs[i]
				}
				return ""
			}
			tree.AddAt(p, strconv.Itoa(i), func(w *core.Text) {
				w.Styler(func(s *styles.Style) {
					s.Font.Family = rich.Monospace
					s.Text.WhiteSpace = text.WhiteSpacePreWrap
					s.Grow.Set(1, 0)
					s.Padding.Set(units.Dp(2), units.Dp(0))
					s.Border.Width.Bottom.Dp(1)
					s.Border.Color.Bottom = colors.Scheme.OutlineVariant
					switch consoleLevel(msg()) {
					case "ERROR":
						s.Color = colors.Scheme.Error.Base
					case "WARN":
						s.Color = colors.Uniform(colors.FromRGB(0xb0, 0x70, 0x00))
					case "DEBUG":
						s.Color = colors.Scheme.OnSurfaceVariant
					}
				})
				w.Updater(func() { w.SetText(msg()) })
			})
		}
	})
}

// consoleLevel returns the level of the given console message (as ERROR),
// after the URL of its frame, if any.
func consoleLevel(m string) string {
	if strings.HasPrefix(m, "[frame ") {
		if _, rest, ok := strings.Cut(m, "] "); ok {
			m = rest
		}
	}
	level, _, _ := strings.Cut(m, ":")
	return level
}

// refresh shows the messages of the page of the current tab (with the
// lock held), scrolled to the last one.
func (cp *ConsolePanel) refresh() {
	cp.msgs = nil
	if pg := cp.browser.CurrentPage(); pg != nil {
		cp.msgs = pg.ScriptConsole()
	}
	cp.list.Update()
	cp.list.Defer(func() { cp.list.ScrollDimToContentEnd(1) })
}

// changed schedules an update of the panel for new messages (from any
// goroutine), if it is shown.
func (cp *ConsolePanel) changed() {
	if !cp.browser.ConsoleShown() || cp.pending.Swap(true) {
		return
	}
	time.AfterFunc(consoleDelay, func() {
		cp.pending.Store(false)
		if !cp.AsyncLockIfShown() {
			return
		}
		defer cp.AsyncUnlock()
		cp.refresh()
	})
}

// ShowConsole shows or hides the console panel of the browser, below its
// tabs (in a split of a fifth of its height when it is shown).
func (br *Browser) ShowConsole(show bool) {
	// (the splits set this as they render, after the layout, which would
	// not give the panel its size the first time)
	br.Console.SetState(!show, states.Invisible)
	if show {
		br.Splits.SetSplits(0.75, 0.25)
		br.consoleShown.Store(true)
		br.Console.refresh()
	} else {
		br.Splits.SetSplits(1, 0)
		br.consoleShown.Store(false)
	}
	br.Splits.NeedsLayout()
}

// ConsoleShown returns whether the console panel of the browser is shown.
func (br *Browser) ConsoleShown() bool {
	return br.consoleShown.Load()
}

// clearConsole clears the console messages of the page and its frames.
func (pg *Page) clearConsole() {
	if ps := pg.scripts; ps != nil {
		ps.consoleMu.Lock()
		ps.console = nil
		ps.consoleMu.Unlock()
	}
	for _, fp := range pg.frames {
		fp.clearConsole()
	}
}

// consoleChanged tells the console panel that the page shows its messages
// in, if any, that it has new ones (from any goroutine).
func (pg *Page) consoleChanged() {
	if f := pg.topPage().onConsole; f != nil {
		f()
	}
}
