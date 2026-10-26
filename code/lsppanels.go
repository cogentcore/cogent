// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"cmp"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textcore"
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
	ls := cv.LangSvc()
	var ed *textcore.Editor // existing editor to keep the view of
	var scroll float32
	var cursor textpos.Pos
	if sel || !has || cv.tabEditor(tab) == nil {
		b, tv, _ := cv.RecycleCmdTab(tab)
		if tv == nil {
			return
		}
		buf = b
		tv.AutoscrollOnInput = false
		ls.mu.Lock()
		delete(ls.locCurrent, tab) // a new listing
		ls.mu.Unlock()
		tv.LinkHandler = func(tl *rich.Hyperlink) {
			cv.openLocationLink(tab, tl.URL)
		}
	} else {
		ed = cv.tabEditor(tab)
		scroll, cursor = ed.ScrollPos(), ed.CursorPos
		buf.SetText(nil)
	}
	sortLocationItems(items)
	lns, mus, itemLines := locationsMarkup(cv, buf.FontStyle(), header, items)
	ls.setLocationItems(tab, items, itemLines)
	if ed != nil {
		// the output editor styling turns autoscroll back on, which would
		// go to the end on the Input from appending:
		ed.AutoscrollOnInput = false
	}
	buf.AppendTextMarkup(lns, mus)
	if ed == nil {
		return
	}
	// keep the view where it was, with the current item visible:
	ed.CursorPos = buf.ValidPos(cursor)
	ed.SetScrollPos(scroll)
	ls.mu.Lock()
	cur, has := ls.locCurrent[tab]
	ls.mu.Unlock()
	if !has {
		return
	}
	i := slices.Index(items, cur)
	if i < 0 {
		// gone, e.g., fixed: show the next one, but keep the current one,
		// as servers can briefly publish no problems for a changed file
		i = nextLocationIndex(items, cur)
	}
	cv.highlightLocation(tab, i)
}

// tabEditor returns the editor of the tab with the given name,
// or nil if there is none.
func (cv *Code) tabEditor(tab string) *textcore.Editor {
	if cv.Tabs() == nil {
		return nil
	}
	if fr := cv.Tabs().TabByName(tab); fr != nil && fr.AsTree().HasChildren() {
		ed, _ := fr.AsTree().Child(0).(*textcore.Editor)
		return ed
	}
	return nil
}

// compareLocationItems compares location items by file and position.
func compareLocationItems(a, b locationItem) int {
	return cmp.Or(strings.Compare(a.loc.Filename, b.loc.Filename),
		cmp.Compare(a.loc.Region.Start.Line, b.loc.Region.Start.Line),
		cmp.Compare(a.loc.Region.Start.Char, b.loc.Region.Start.Char))
}

// sortLocationItems sorts the location items by file and position.
func sortLocationItems(items []locationItem) {
	slices.SortStableFunc(items, compareLocationItems)
}

// nextLocationIndex returns the index of the first of the sorted items
// after the given item, which is not among them, or of the last item if
// there is none after it, or -1 if there are no items.
func nextLocationIndex(items []locationItem, it locationItem) int {
	for i, o := range items {
		if compareLocationItems(o, it) > 0 {
			return i
		}
	}
	return len(items) - 1
}

// setCurrentLocation makes the location item with the given index the
// current one in the given tab, highlighting its line and making it
// visible. Must be called from the event loop.
func (cv *Code) setCurrentLocation(tab string, i int) {
	ls := cv.LangSvc()
	ls.mu.Lock()
	items := ls.locItems[tab]
	if i < 0 || i >= len(items) {
		ls.mu.Unlock()
		return
	}
	ls.locCurrent[tab] = items[i]
	ls.mu.Unlock()
	cv.highlightLocation(tab, i)
}

// highlightLocation highlights the line of the location item with the
// given index in the given tab, and makes it visible.
// Must be called from the event loop.
func (cv *Code) highlightLocation(tab string, i int) {
	ls := cv.LangSvc()
	ls.mu.Lock()
	lines := ls.locLines[tab]
	ls.mu.Unlock()
	if i < 0 || i >= len(lines) {
		return
	}
	ed := cv.tabEditor(tab)
	if ed == nil || ed.Lines == nil {
		return
	}
	ln := lines[i]
	ed.SetCursorShow(textpos.Pos{Line: ln})
	ed.Highlights = []textpos.Region{{Start: textpos.Pos{Line: ln}, End: textpos.Pos{Line: ln, Char: ed.Lines.LineLen(ln)}}}
	ed.NeedsRender()
}

// locationsMarkup returns the text lines and markup for the given
// header and sorted location items, for [Code.showLocations], and the
// line of each item.
func locationsMarkup(cv *Code, sty *rich.Style, header string, items []locationItem) ([][]rune, []rich.Text, []int) {
	bold := sty.Clone().SetWeight(rich.Bold)
	link := sty.Clone().SetLinkStyle()
	match := sty.Clone().SetBackground(colors.ToUniform(colors.Scheme.Warn.Container))
	sevSty := map[string]*rich.Style{
		"error":   sty.Clone().SetFillColor(colors.ToUniform(colors.Scheme.Error.Base)),
		"warning": sty.Clone().SetFillColor(colors.ToUniform(colors.Scheme.Warn.Base)),
	}
	var lns [][]rune
	var mus []rich.Text
	itemLines := make([]int, len(items))
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
		itemLines[i] = len(lns)
		add(append([]rune(pos), txt...), mu)
	}
	return lns, mus, itemLines
}

// setLocationItems records the location items shown in the given tab,
// and their lines, for opening their links.
func (ls *LangService) setLocationItems(tab string, items []locationItem, lines []int) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	if ls.locItems == nil {
		ls.locItems = make(map[string][]locationItem)
		ls.locLines = make(map[string][]int)
		ls.locCurrent = make(map[string]locationItem)
	}
	ls.locItems[tab] = items
	ls.locLines[tab] = lines
}

// openLocationLink opens the location of the given location link URL
// in the given tab, making it the current one.
func (cv *Code) openLocationLink(tab, url string) {
	idx, err := strconv.Atoi(strings.TrimPrefix(url, locationLinkPrefix))
	if err != nil {
		return
	}
	cv.openLocationItem(tab, idx)
}

// openLocationItem opens the location of the location item with the
// given index in the given tab, making it the current one.
func (cv *Code) openLocationItem(tab string, idx int) {
	ls := cv.LangSvc()
	ls.mu.Lock()
	items := ls.locItems[tab]
	ls.mu.Unlock()
	if idx < 0 || idx >= len(items) {
		return
	}
	cv.setCurrentLocation(tab, idx)
	cv.openLoc(items[idx].loc)
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
// language servers for the files in the given scope in the Problems tab,
// where they can be clicked to go to them. The tab is updated as the
// problems change, while it is open.
func (cv *Code) ShowProblems(scope Locations) { //types:add
	cv.Settings.ProblemsLoc = scope
	ls := cv.LangSvc()
	ls.mu.Lock()
	ls.problemsShown = true
	ls.mu.Unlock()
	ls.updateProblems(true)
	ls.pullProblems()
}

// updateProblems updates the Problems tab if it is being shown, selecting
// it if sel. Must be called from the event loop or under AsyncLock.
func (ls *LangService) updateProblems(sel bool) {
	cv := ls.cv
	ls.mu.Lock()
	shown := ls.problemsShown
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
	inScope, desc := cv.problemsScope(cv.Settings.ProblemsLoc)
	ls.mu.Lock()
	var items []locationItem
	files := slices.Collect(maps.Keys(ls.diagCache))
	for fname := range ls.pulled {
		if _, has := ls.diagCache[fname]; !has {
			files = append(files, fname)
		}
	}
	for _, fname := range files {
		if !inScope(fname) {
			continue
		}
		diags := ls.diagCache[fname]
		if pd, has := ls.pulled[fname]; has && cv.GetOpenFile(fname) == nil {
			diags = mergeDiags(pd, diags)
		}
		for _, d := range filterIgnored(diags, ls.ignore) {
			if !showSeverity(d.Severity, ls.hints) {
				continue
			}
			items = append(items, locationItem{loc: lang.Location{Filename: fname, Region: d.Region}, sev: diagWord(d.Severity), msg: d.Msg})
		}
	}
	ls.mu.Unlock()
	header := "Problems in " + desc + ": " + diagSummaryText(items)
	cv.showLocations(problemsTab, header, items, sel)
}

// mergeDiags returns the pulled diagnostics of a file plus those of the
// published ones that are not among them.
func mergeDiags(pulled, published []lang.Diagnostic) []lang.Diagnostic {
	out := slices.Clone(pulled)
	for _, d := range published {
		if !slices.ContainsFunc(pulled, func(p lang.Diagnostic) bool {
			return p.Msg == d.Msg && p.Region.Start == d.Region.Start
		}) {
			out = append(out, d)
		}
	}
	return out
}

// pullProblems requests the diagnostics of the files that are not open
// in the directories of the Problems tab scope, in the background, from
// the servers that provide them on request, and then updates the tab.
// Servers such as gopls only publish some of the problems of files that
// are not open: type errors, but not the results of the analyzers.
// The All scope only uses the published problems, as it would take too
// long. Must be called from the event loop.
func (ls *LangService) pullProblems() {
	cv := ls.cv
	var dirs []string
	switch cv.Settings.ProblemsLoc {
	case Dir:
		if tv := cv.ActiveEditor(); tv != nil && tv.Lines != nil {
			dirs = []string{filepath.Dir(tv.Lines.Filename())}
		}
	case Open:
		if cv.Files != nil {
			dirs = cv.Files.OpenPaths()
		}
	}
	// an open buffer with a language server, for each file extension:
	servers := map[string]*lines.Lines{}
	for _, ln := range cv.OpenFiles.Values {
		if ext := filepath.Ext(ln.Filename()); servers[ext] == nil && ls.LSPActive(ln) {
			servers[ext] = ln
		}
	}
	files := map[string][]string{}
	for _, dir := range dirs {
		des, _ := os.ReadDir(dir)
		for _, de := range des {
			fname := filepath.Join(dir, de.Name())
			ext := filepath.Ext(fname)
			if de.IsDir() || servers[ext] == nil || cv.GetOpenFile(fname) != nil {
				continue
			}
			files[ext] = append(files[ext], fname)
		}
	}
	if len(files) == 0 {
		return
	}
	go func() {
		var fds []lang.FileDiags
		for ext, fnames := range files {
			fds = append(fds, ls.lspB.FileDiagnostics(servers[ext], fnames)...)
		}
		ls.mu.Lock()
		for _, fd := range fds {
			if len(fd.Diags) == 0 {
				delete(ls.pulled, fd.Filename)
			} else {
				ls.pulled[fd.Filename] = fd.Diags
			}
		}
		ls.mu.Unlock()
		if cv.Scene == nil {
			return
		}
		cv.AsyncLock()
		ls.updateProblems(false)
		cv.AsyncUnlock()
	}()
}

// problemsScope returns a function that reports whether the problems of
// a file are shown for the given scope, with the same meaning as for
// [Code.Find], and a description of the scope.
func (cv *Code) problemsScope(scope Locations) (func(fname string) bool, string) {
	afile := ""
	if tv := cv.ActiveEditor(); tv != nil && tv.Lines != nil {
		afile = tv.Lines.Filename()
	}
	switch scope {
	case File:
		return func(fname string) bool { return fname == afile }, relativePath(cv, afile)
	case Dir:
		adir := filepath.Dir(afile)
		return func(fname string) bool { return afile != "" && filepath.Dir(fname) == adir }, relativePath(cv, adir)
	case Open:
		dirs := map[string]bool{}
		if cv.Files != nil {
			for _, p := range cv.Files.OpenPaths() {
				dirs[filepath.Clean(p)] = true
			}
		}
		return func(fname string) bool { return dirs[filepath.Dir(fname)] }, "open directories"
	}
	root := ""
	if cv.Files != nil {
		root = filepath.Clean(string(cv.Files.Filepath)) + string(filepath.Separator)
	}
	return func(fname string) bool { return root != "" && strings.HasPrefix(fname, root) }, "all files"
}

// NextProblem goes to the current problem in the Problems tab, if the
// cursor is not already on it in the active editor, and otherwise to the
// next one. The current problem is the last one clicked or gone to, and
// when it goes away, e.g., after applying a fix for it, NextProblem goes
// to the next one, which is highlighted. The Problems tab is shown if it is not already.
func (cv *Code) NextProblem() { //types:add
	ls := cv.LangSvc()
	if cv.tabEditor(problemsTab) == nil {
		cv.ShowProblems(cv.Settings.ProblemsLoc)
	}
	ls.mu.Lock()
	items := ls.locItems[problemsTab]
	cur, has := ls.locCurrent[problemsTab]
	ls.mu.Unlock()
	if len(items) == 0 {
		cv.SetStatus("No problems")
		return
	}
	if !has {
		cv.openLocationItem(problemsTab, 0)
		return
	}
	i := slices.Index(items, cur)
	if i < 0 { // gone, e.g., fixed
		cv.openLocationItem(problemsTab, nextLocationIndex(items, cur))
		return
	}
	if tv := cv.ActiveEditor(); tv == nil || tv.Lines == nil || tv.Lines.Filename() != cur.loc.Filename || tv.CursorPos.Line != cur.loc.Region.Start.Line {
		cv.openLocationItem(problemsTab, i) // not there yet
		return
	}
	if i+1 >= len(items) {
		cv.SetStatus("No more problems")
		return
	}
	cv.openLocationItem(problemsTab, i+1)
}

// diagSummaryText returns a summary of the numbers of errors, warnings,
// and info and hints, if any, in the given problem items.
func diagSummaryText(items []locationItem) string {
	nerr, nwarn, nhint := 0, 0, 0
	for _, it := range items {
		switch it.sev {
		case "error":
			nerr++
		case "warning":
			nwarn++
		default:
			nhint++
		}
	}
	if nerr+nwarn+nhint == 0 {
		return "none"
	}
	s := fmt.Sprintf("%d error%s, %d warning%s", nerr, plural(nerr), nwarn, plural(nwarn))
	if nhint > 0 {
		s += fmt.Sprintf(", %d hint%s", nhint, plural(nhint))
	}
	return s
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
