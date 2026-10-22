// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"

	"cogentcore.org/cogent/code/lang"
)

// Tab names for the language service panels.
const (
	referencesTab = "References"
	problemsTab   = "Problems"
)

// locationItem is one item in a list of locations shown in a tab:
// a reference, or a problem (diagnostic).
type locationItem struct {
	loc lang.Location

	// sev is the severity of a problem, or "" for a reference.
	sev string

	// msg is the message for a problem.
	msg string
}

// locationLinkPrefix is the prefix of the link URLs for location items,
// followed by the index of the item.
const locationLinkPrefix = "location:"

// showLocations shows the given location items in the tab with the
// given name, under the given header line, grouped by file, with each
// location a link that opens it. References show the text of the line
// at the location, with the location highlighted, and problems show
// their severity and message. If the tab already exists, it is updated
// without selecting it, unless sel is set.
func (cv *Code) showLocations(tab, header string, items []locationItem, sel bool) {
	buf, has := cv.CmdBufs[tab]
	if sel || !has || cv.Tabs() == nil || cv.Tabs().TabByName(tab) == nil {
		b, tv, _ := cv.RecycleCmdTab(tab)
		if tv == nil {
			return
		}
		buf = b
		tv.AutoscrollOnInput = false
		tv.LinkHandler = func(tl *rich.Hyperlink) {
			cv.openLocationLink(tab, tl.URL)
		}
	} else {
		buf.SetText(nil)
	}
	sortLocationItems(items)
	cv.LangSvc().setLocationItems(tab, items)
	buf.AppendTextMarkup(locationsMarkup(cv, buf.FontStyle(), header, items))
}

// sortLocationItems sorts the location items by file and position.
func sortLocationItems(items []locationItem) {
	slices.SortStableFunc(items, func(a, b locationItem) int {
		return cmp.Or(strings.Compare(a.loc.Filename, b.loc.Filename),
			cmp.Compare(a.loc.Region.Start.Line, b.loc.Region.Start.Line),
			cmp.Compare(a.loc.Region.Start.Char, b.loc.Region.Start.Char))
	})
}

// locationsMarkup returns the text lines and markup for the given
// header and sorted location items, for [Code.showLocations].
func locationsMarkup(cv *Code, sty *rich.Style, header string, items []locationItem) ([][]rune, []rich.Text) {
	bold := sty.Clone().SetWeight(rich.Bold)
	link := sty.Clone().SetLinkStyle()
	match := sty.Clone().SetBackground(colors.ToUniform(colors.Scheme.Warn.Container))
	sevSty := map[string]*rich.Style{
		"error":   sty.Clone().SetFillColor(colors.ToUniform(colors.Scheme.Error.Base)),
		"warning": sty.Clone().SetFillColor(colors.ToUniform(colors.Scheme.Warn.Base)),
	}
	var lns [][]rune
	var mus []rich.Text
	add := func(ln []rune, mu rich.Text) {
		lns = append(lns, ln)
		mus = append(mus, mu)
	}
	add([]rune(header), rich.NewText(bold, []rune(header)))
	files := fileLineCache{cv: cv}
	for i, it := range items {
		fname := it.loc.Filename
		rel := relativePath(cv, fname)
		if i == 0 || items[i-1].loc.Filename != fname {
			n := 0
			for _, o := range items[i:] {
				if o.loc.Filename != fname {
					break
				}
				n++
			}
			hdr := []rune(fmt.Sprintf("%s: %d", rel, n))
			add(nil, rich.NewText(sty, nil))
			add(hdr, rich.NewText(bold, hdr))
		}
		st := it.loc.Region.Start
		pos := fmt.Sprintf("\t%s:%d:%d: ", rel, st.Line+1, st.Char+1)
		mu := rich.Text{}
		mu.AddLink(link, locationLinkPrefix+strconv.Itoa(i), pos)
		var txt []rune
		if it.sev != "" {
			ss := sevSty[it.sev]
			if ss == nil {
				ss = sty
			}
			txt = []rune(it.sev + ": " + it.msg)
			mu.AddSpan(ss, []rune(it.sev+": ")).AddSpanString(sty, it.msg)
		} else {
			txt = []rune(strings.ReplaceAll(files.line(fname, st.Line), "\t", " "))
			mst := min(st.Char, len(txt))
			med := len(txt)
			if it.loc.Region.End.Line == st.Line {
				med = max(mst, min(it.loc.Region.End.Char, len(txt)))
			}
			mu.AddSpan(sty, txt[:mst]).AddSpan(match, txt[mst:med]).AddSpan(sty, txt[med:])
		}
		add(append([]rune(pos), txt...), mu)
	}
	return lns, mus
}

// setLocationItems records the location items shown in the given tab,
// for opening their links.
func (ls *LangService) setLocationItems(tab string, items []locationItem) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	if ls.locItems == nil {
		ls.locItems = make(map[string][]locationItem)
	}
	ls.locItems[tab] = items
}

// openLocationLink opens the location of the given location link URL
// in the given tab.
func (cv *Code) openLocationLink(tab, url string) {
	idx, err := strconv.Atoi(strings.TrimPrefix(url, locationLinkPrefix))
	if err != nil {
		return
	}
	ls := cv.LangSvc()
	ls.mu.Lock()
	items := ls.locItems[tab]
	ls.mu.Unlock()
	if idx >= 0 && idx < len(items) {
		cv.openLoc(items[idx].loc)
	}
}

// relativePath returns the path of the file relative to the project
// root for display, or its directory and name if it is outside of it.
func relativePath(cv *Code, fname string) string {
	if rel, err := filepath.Rel(string(cv.Settings.ProjectRoot), fname); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return filepath.Join(filepath.Base(filepath.Dir(fname)), filepath.Base(fname))
}

// fileLineCache returns the text of lines in files, from the open
// buffer if the file is open, and otherwise from the file contents,
// which are read once.
type fileLineCache struct {
	cv    *Code
	files map[string][]string
}

// line returns the text of the given line of the given file.
func (fc *fileLineCache) line(fname string, line int) string {
	if ln := fc.cv.GetOpenFile(fname); ln != nil {
		if ln.IsValidLine(line) {
			return string(ln.Line(line))
		}
		return ""
	}
	if fc.files == nil {
		fc.files = make(map[string][]string)
	}
	lns, has := fc.files[fname]
	if !has {
		if b, err := os.ReadFile(fname); err == nil {
			lns = strings.Split(string(b), "\n")
		}
		fc.files[fname] = lns
	}
	if line >= 0 && line < len(lns) {
		return strings.TrimSuffix(lns[line], "\r")
	}
	return ""
}

////////   References

// FindReferences lists all references to the symbol at the cursor,
// using the language server, in the References tab, where they can be
// clicked to go to them.
func (cv *Code) FindReferences() { //types:add
	tv := cv.lspEditor("Find references")
	if tv == nil {
		return
	}
	pos, ok := cursorWordPos(tv)
	if !ok {
		return
	}
	cv.SetStatus("Finding references...")
	locs := cv.LangSvc().lspB.References(tv.Lines, pos)
	if len(locs) == 0 {
		cv.SetStatus("Find references: none found")
		return
	}
	cv.showReferences(symbolAt(tv.Lines, pos), locs)
}

// showReferences shows the given references to the given symbol
// in the References tab.
func (cv *Code) showReferences(sym string, locs []lang.Location) {
	items := make([]locationItem, len(locs))
	files := map[string]bool{}
	for i, loc := range locs {
		items[i] = locationItem{loc: loc}
		files[loc.Filename] = true
	}
	header := fmt.Sprintf("References to %s: %d in %d file%s", sym, len(locs), len(files), plural(len(files)))
	cv.showLocations(referencesTab, header, items, true)
	cv.SetStatus(header)
}

////////   Problems

// ShowProblems shows the problems (errors and warnings) reported by the
// language servers for all of the open files in the Problems tab, where
// they can be clicked to go to them. The tab is updated as the problems
// change, while it is open.
func (cv *Code) ShowProblems() { //types:add
	ls := cv.LangSvc()
	ls.mu.Lock()
	ls.problemsShown = true
	ls.mu.Unlock()
	ls.updateProblems(true)
}

// updateProblems updates the Problems tab if it is being shown, selecting
// it if sel. Must be called from the event loop or under AsyncLock.
func (ls *LangService) updateProblems(sel bool) {
	cv := ls.cv
	ls.mu.Lock()
	shown := ls.problemsShown
	var items []locationItem
	for fname, diags := range ls.diagCache {
		for _, d := range diags {
			if d.Severity != lang.DiagErr && d.Severity != lang.DiagWarn {
				continue
			}
			items = append(items, locationItem{loc: lang.Location{Filename: fname, Region: d.Region}, sev: diagWord(d.Severity), msg: d.Msg})
		}
	}
	ls.mu.Unlock()
	if !shown {
		return
	}
	if !sel && (cv.Tabs() == nil || cv.Tabs().TabByName(problemsTab) == nil) {
		ls.mu.Lock()
		ls.problemsShown = false // tab was closed
		ls.mu.Unlock()
		return
	}
	header := "Problems in open files: " + diagSummaryText(items)
	cv.showLocations(problemsTab, header, items, sel)
}

// diagSummaryText returns a summary of the numbers of errors and
// warnings in the given problem items.
func diagSummaryText(items []locationItem) string {
	nerr, nwarn := 0, 0
	for _, it := range items {
		if it.sev == "error" {
			nerr++
		} else {
			nwarn++
		}
	}
	if nerr+nwarn == 0 {
		return "none"
	}
	return fmt.Sprintf("%d error%s, %d warning%s", nerr, plural(nerr), nwarn, plural(nwarn))
}

////////   Occurrences

// occurrenceDelay is how long the cursor must rest on a symbol before
// its occurrences are highlighted, so that moving through the text does
// not generate a request for each position.
const occurrenceDelay = 200 * time.Millisecond

// scheduleOccurrences highlights the occurrences of the symbol at the
// cursor in the editor after a short delay, if the cursor is still there.
// It is called from the event loop after each key and Input event.
func (ls *LangService) scheduleOccurrences(tv *TextEditor) {
	ln := tv.Lines
	if ln == nil || tv.Scene == nil || !ls.LSPActive(ln) {
		return
	}
	pos := tv.CursorPos
	ls.mu.Lock()
	ls.occurGen++
	gen := ls.occurGen
	ls.mu.Unlock()
	time.AfterFunc(occurrenceDelay, func() {
		current := func() bool {
			ls.mu.Lock()
			defer ls.mu.Unlock()
			return ls.occurGen == gen
		}
		if !current() {
			return
		}
		regs := ls.occurrences(tv, ln, pos)
		tv.AsyncLock()
		defer tv.AsyncUnlock()
		if current() && tv.Lines == ln && tv.CursorPos == pos {
			ls.setOccurrences(tv, regs)
		}
	})
}

// occurrences returns the regions of the occurrences of the symbol at
// the given position, if they should be highlighted, and nil otherwise:
// when there is a selection, or no symbol, or only one occurrence.
func (ls *LangService) occurrences(tv *TextEditor, ln *lines.Lines, pos textpos.Pos) []textpos.Region {
	if tv.HasSelection() {
		return nil
	}
	regs := ls.lspB.Occurrences(ln, pos)
	if len(regs) < 2 {
		return nil
	}
	return regs
}

// setOccurrences sets the given occurrence regions as the highlights of
// the editor, unless the highlights are being used for something else,
// such as Find results, which are only replaced once they are reset,
// or Find results are showing in the Find tab.
// Must be called from the event loop or under AsyncLock.
func (ls *LangService) setOccurrences(tv *TextEditor, regs []textpos.Region) {
	if tv.ISearch.On || tv.QReplace.On || ls.findShowing() {
		return
	}
	ls.mu.Lock()
	defer ls.mu.Unlock()
	owned := len(tv.Highlights) == 0 || (ls.occurEditor == tv && slices.Equal(tv.Highlights, ls.occurRegs))
	if !owned {
		return
	}
	ls.occurEditor, ls.occurRegs = tv, regs
	tv.Highlights = slices.Clone(regs)
	tv.NeedsRender()
}

// replaceableHighlights marks the current highlights of the editor as
// replaceable by occurrence highlighting, e.g., the highlight of the
// target region of a jump to a location, which then lasts until the
// cursor rests somewhere else.
func (ls *LangService) replaceableHighlights(tv *TextEditor) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.occurEditor, ls.occurRegs = tv, slices.Clone(tv.Highlights)
}

// findShowing returns whether the Find tab is the current tab.
func (ls *LangService) findShowing() bool {
	tabs := ls.cv.Tabs()
	if tabs == nil {
		return false
	}
	w, _ := tabs.CurrentTab()
	return w != nil && w.AsTree().Name == "Find"
}
