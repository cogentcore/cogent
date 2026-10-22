// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"fmt"
	"image"
	"path/filepath"
	"strings"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/parse/syms"
	"cogentcore.org/core/text/search"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/text/textpos"

	"cogentcore.org/cogent/code/lang"
	"cogentcore.org/cogent/code/lsp"
)

// lspEditor returns the active editor if it has a buffer that uses
// the language server backend, and otherwise reports in the status
// bar that the given command requires one, returning nil.
func (cv *Code) lspEditor(cmd string) *TextEditor {
	tv := cv.ActiveEditor()
	if tv == nil || tv.Lines == nil {
		return nil
	}
	if !cv.LangSvc().LSPActive(tv.Lines) {
		cv.SetStatus(cmd + ": requires an LSP language server for this file")
		return nil
	}
	return tv
}

// cursorWordPos returns a position on the word at the cursor,
// preferring the end of the word, for definition-type queries.
func cursorWordPos(tv *TextEditor) (textpos.Pos, bool) {
	ln := tv.Lines
	pos := tv.CursorPos
	if tv.HasSelection() {
		if tv.SelectRegion.End.Line != tv.SelectRegion.Start.Line {
			return textpos.Pos{}, false
		}
		pos = tv.SelectRegion.End
	} else if !ln.IsWordEnd(pos) {
		w := ln.WordAt(pos)
		if w.Start.Line == pos.Line && w.End.Char > 0 {
			pos = w.End
		}
	}
	if !ln.IsValidLine(pos.Line) {
		return textpos.Pos{}, false
	}
	return pos, true
}

// wordAtCursor returns the word at the cursor for use as a rename
// placeholder, preferring a single-line selection if one exists.
// Returns "" if there is no word at the cursor.
func wordAtCursor(tv *TextEditor) string {
	ln := tv.Lines
	if tv.HasSelection() {
		r := tv.SelectRegion
		if r.Start.Line == r.End.Line && r.End.Char > r.Start.Char {
			return string(ln.Region(r.Start, r.End).ToBytes())
		}
		return ""
	}
	pos := tv.CursorPos
	if !ln.IsValidLine(pos.Line) {
		return ""
	}
	if ln.IsWordEnd(pos) && pos.Char > 0 {
		pos.Char-- // move inside the word so WordAt finds it
	}
	w := ln.WordAt(pos)
	if w.Start.Line != pos.Line || w.End.Char <= w.Start.Char {
		return ""
	}
	return string(ln.Region(w.Start, w.End).ToBytes())
}

// actionRegion returns the region to request code actions for:
// a single-line selection if there is one, otherwise the word at
// the cursor, falling back to the bare cursor position.
func actionRegion(tv *TextEditor) textpos.Region {
	if tv.HasSelection() && tv.SelectRegion.End.Line == tv.SelectRegion.Start.Line {
		return tv.SelectRegion
	}
	return wordRegion(tv.Lines, tv.CursorPos)
}

// wordRegion returns the region of the word at the given position,
// or the empty region at the position if there is no word.
func wordRegion(ln *lines.Lines, pos textpos.Pos) textpos.Region {
	if ln.IsValidLine(pos.Line) {
		if w := ln.WordAt(pos); w.Start.Line == pos.Line && w.End.Char >= w.Start.Char {
			return w
		}
	}
	return textpos.Region{Start: pos, End: pos}
}

////////   Lookup and quick fixes

// Lookup is the Ctrl+. action at the cursor: if there is an error with
// quick fixes from the language server there, it shows a menu of them,
// and otherwise it shows the completions for the text up to the cursor,
// headed by a go to definition item for the symbol at the cursor.
func (cv *Code) Lookup() { //types:add
	if tv := cv.ActiveEditor(); tv != nil && tv.Lines != nil {
		cv.LangSvc().Lookup(tv)
	}
}

// Lookup does [Code.Lookup] for the given editor, returning true if
// it showed quick fixes, and false if it offered completions.
func (ls *LangService) Lookup(tv *TextEditor) bool {
	if ls.LSPActive(tv.Lines) && ls.lookupMenu(tv, tv.CursorPos, true) {
		return true
	}
	ls.offerCompletion(tv)
	return false
}

// gotoDefCompletion and docsCompletion are the Text of the go to
// definition and docs completion items, which are handled by a select
// listener, without editing the text.
const (
	gotoDefCompletion = "\x00goto-definition"
	docsCompletion    = "\x00docs"
)

// offerCompletion shows the completions for the text up to the cursor,
// like the editor does while typing, for an explicit request, which
// for LSP files also includes completions with no seed, and a go to
// definition item (see [LangService.explicitMatches]).
func (ls *LangService) offerCompletion(tv *TextEditor) {
	c := tv.Complete
	ln := tv.Lines
	pos := tv.CursorPos
	if c == nil || !ln.IsValidLine(pos.Line) {
		return
	}
	rn := ln.Line(pos.Line)
	pos.Char = min(pos.Char, len(rn))
	if ls.LSPActive(ln) {
		ls.mu.Lock()
		ls.explicit, ls.explicitAt, ls.explicitLn = true, pos, ln
		ls.mu.Unlock()
	}
	c.SrcLn, c.SrcCh = pos.Line, pos.Char
	c.Show(tv, editorCursorPos(tv, pos), strings.TrimLeft(string(rn[:pos.Char]), " \t"))
}

// takeExplicit returns whether there is a pending explicit completion
// request for the given buffer and position, clearing it.
func (ls *LangService) takeExplicit(ln *lines.Lines, pos textpos.Pos) bool {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ok := ls.explicit && ls.explicitLn == ln && ls.explicitAt == pos
	ls.explicit = false
	return ok
}

// explicitMatches returns the completions for an explicit request at
// the given position, headed by a go to definition item for the symbol
// at the position, if it is defined elsewhere, and a docs item that
// shows its docs, if any.
func (ls *LangService) explicitMatches(ln *lines.Lines, text string, pos textpos.Pos) complete.Matches {
	ms := ls.lspB.CompletionsExplicit(ln, text, pos.Line, pos.Char)
	md := ls.lspB.Hover(ln, pos)
	tip := lsp.MarkdownText(md)
	var loc *lang.Location
	var items []complete.Completion
	if locs := ls.lspB.Definition(ln, pos); len(locs) > 0 &&
		!(lsp.SameFile(locs[0].Filename, ln.Filename()) && locs[0].Region.Start.Line == pos.Line) {
		loc = &locs[0]
		items = append(items, complete.Completion{
			Text:  gotoDefCompletion,
			Label: "Go to definition: " + locString(ls.cv, *loc),
			Icon:  icons.JumpToElement,
			Desc:  tip,
		})
	}
	if md != "" {
		items = append(items, complete.Completion{
			Text:  docsCompletion,
			Label: "Docs: " + docsSummary(tip),
			Icon:  icons.Description,
			Desc:  tip,
		})
	}
	ls.mu.Lock()
	ls.gotoLoc, ls.docsMD, ls.docsSymbol = loc, md, symbolAt(ln, pos)
	ls.mu.Unlock()
	ms.Matches = append(items, ms.Matches...)
	return ms
}

// symbolAt returns the name of the symbol (word) at the given position,
// which can be at the end of the word.
func symbolAt(ln *lines.Lines, pos textpos.Pos) string {
	if ln.IsWordEnd(pos) && pos.Char > 0 {
		pos.Char--
	}
	w := wordRegion(ln, pos)
	return string(ln.Region(w.Start, w.End).ToBytes())
}

// docsSummary returns a short summary of the given plain text docs,
// for a menu label: the first line, truncated.
func docsSummary(docs string) string {
	rn := []rune(firstLine(docs))
	if len(rn) > 60 {
		rn = append(rn[:60], '…')
	}
	return string(rn)
}

// lookupFunc returns the editor LookupFunc for the given buffer,
// which shows the [LangService.lookupMenu].
func (ls *LangService) lookupFunc(ln *lines.Lines) complete.LookupFunc {
	return func(data any, txt string, posLine, posChar int) complete.Lookup {
		if tv, _, has := ls.cv.EditorForLines(ln); has && tv != nil && tv.Scene != nil {
			ls.lookupMenu(tv, textpos.Pos{Line: posLine, Char: posChar}, false)
		}
		return complete.Lookup{}
	}
}

// lookupMenu shows a completer-style chooser menu at the given position
// in the editor: the quick fixes available from the language server for
// any errors at the position (applied on click), followed by the symbol
// lookup action: showing hover docs if the symbol is defined in the
// current file, and opening the defining file otherwise. If fixesOnly,
// the menu is only shown if there are quick fixes. It returns whether
// the menu was shown.
func (ls *LangService) lookupMenu(tv *TextEditor, pos textpos.Pos, fixesOnly bool) bool {
	ln := tv.Lines
	if !ln.IsValidLine(pos.Line) {
		return false
	}
	diags := ls.diagsAt(ln, pos)
	if fixesOnly && len(diags) == 0 {
		return false
	}
	if tv.Complete != nil {
		tv.Complete.Cancel() // like the completer does before offering
	}
	ls.Change(ln) // make sure the server has the current text

	var items []core.ChooserItem
	if len(diags) > 0 {
		actions, _ := ls.lspB.CodeActionsQuick(ln, diags[0].Region, diags)
		for _, ca := range actions {
			if ca.IsQuickFix() || len(ca.Diagnostics) > 0 {
				items = append(items, core.ChooserItem{Text: "Fix: " + actionTitle(&ca), Tooltip: ca.Description(), Value: ca})
			}
		}
	}
	if fixesOnly && len(items) == 0 {
		return false
	}
	docs := ls.lspB.Hover(ln, pos)
	tip := lsp.MarkdownText(docs)
	if locs := ls.lspB.Definition(ln, pos); len(locs) > 0 && !lsp.SameFile(locs[0].Filename, ln.Filename()) {
		items = append(items, core.ChooserItem{Text: "Go to definition: " + locString(ls.cv, locs[0]), Tooltip: tip, Value: locs[0]})
	}
	if docs != "" {
		items = append(items, core.ChooserItem{Text: "Docs: " + docsSummary(tip), Tooltip: tip, Value: docs})
	}
	if len(items) == 0 {
		ls.cv.SetStatus("Lookup: nothing available at cursor")
		return false
	}
	core.NewMenu(func(m *core.Scene, mp image.Point) {
		for _, it := range items {
			core.NewButton(m).SetText(it.Text).SetTooltip(it.Tooltip).OnClick(func(e events.Event) {
				switch v := it.Value.(type) {
				case lsp.CodeAction:
					ls.applyCodeAction(ln, &v)
				case lang.Location:
					ls.cv.openLoc(v)
				case string:
					docsDialog(tv, docsWindow+": "+symbolAt(ln, pos), v)
				}
			})
		}
	}, tv, editorCursorPos(tv, pos)).Run()
	return true
}

// actionTitle returns the title of the code action for display,
// with any disabled reason or preferred status.
func actionTitle(ca *lsp.CodeAction) string {
	switch {
	case ca.Disabled != nil:
		return ca.Title + " -- " + ca.Disabled.Reason
	case ca.IsPreferred:
		return ca.Title + " (preferred)"
	}
	return ca.Title
}

// firstLine returns the first line of the given text, for compact menu
// labels from multi-line hover docs.
func firstLine(s string) string {
	ln, _, _ := strings.Cut(s, "\n")
	return ln
}

// applyCodeAction runs the given code action,
// reporting the result in the status bar.
func (ls *LangService) applyCodeAction(ln *lines.Lines, ca *lsp.CodeAction) {
	if ca.Disabled != nil {
		ls.cv.SetStatus("Quick fix: " + ca.Disabled.Reason)
		return
	}
	out, err := ls.lspB.RunCodeAction(ln, ca)
	ls.renderEdited(out)
	if err != nil {
		ls.cv.SetStatus("Quick fix: " + err.Error())
		return
	}
	if len(out) == 0 { // e.g., showed a document
		ls.cv.SetStatus("Done: " + ca.Title)
		return
	}
	n := 0
	for _, f := range out {
		n += len(f.Edits)
	}
	ls.cv.SetStatus(fmt.Sprintf("Applied: %s (%d edits in %d files)", ca.Title, n, len(out)))
}

// QuickFix prompts for and applies a quick fix or refactoring from the
// language server at the cursor or selection, like other editors'
// Ctrl+. (Ctrl+period). The diagnostics cached for the position are
// sent as context, which is what turns the quick fixes on: organize
// imports, missing returns, error-check wrapping, typo fixes, etc,
// along with refactorings like extract function/variable.
func (cv *Code) QuickFix() { //types:add
	tv := cv.lspEditor("Quick fix")
	if tv == nil {
		return
	}
	ln := tv.Lines
	ls := cv.LangSvc()
	reg := actionRegion(tv)
	cv.SetStatus("Quick fix: consulting server...")
	actions, err := ls.lspB.CodeActions(ln, reg, ls.diagsIn(ln, reg))
	if err != nil {
		cv.SetStatus("Quick fix: " + err.Error())
		return
	}
	if len(actions) == 0 {
		cv.SetStatus("Quick fix: no actions available here")
		return
	}
	lsp.SortCodeActions(actions)
	d := core.NewBody("Quick fix / refactor")
	c := core.NewChooser(d)
	items := make([]core.ChooserItem, len(actions))
	for i := range actions {
		items[i] = core.ChooserItem{Text: actionTitle(&actions[i]), Tooltip: actions[i].Description()}
	}
	c.SetItems(items...)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Apply").OnClick(func(e events.Event) {
			if idx := c.CurrentIndex; idx >= 0 && idx < len(actions) {
				ls.applyCodeAction(ln, &actions[idx])
			}
		})
	})
	d.RunWindowDialog(tv)
}

// codeActionsMenu adds the code actions available at the cursor or
// selection in the given editor to the given menu: quick fixes,
// refactorings, and source actions such as browsing the assembly or
// documentation. This is used for a submenu, which is populated when
// it is opened.
func (ls *LangService) codeActionsMenu(m *core.Scene, tv *TextEditor) {
	ln := tv.Lines
	reg := actionRegion(tv)
	actions, err := ls.lspB.CodeActionsQuick(ln, reg, ls.diagsIn(ln, reg))
	if err != nil || len(actions) == 0 {
		core.NewButton(m).SetText("No code actions here").SetState(true, states.Disabled)
		return
	}
	lsp.SortCodeActions(actions)
	for _, ca := range actions {
		core.NewButton(m).SetText(actionTitle(&ca)).SetTooltip(ca.Description()).
			SetState(ca.Disabled != nil, states.Disabled).OnClick(func(e events.Event) {
			ls.applyCodeAction(ln, &ca)
		})
	}
}

////////   Navigation

// GotoDefinition jumps to the definition of the symbol at the cursor,
// using the active language backend.
func (cv *Code) GotoDefinition() { //types:add
	cv.gotoLocations("Go to definition", (*lsp.Backend).Definition)
}

// GotoTypeDefinition jumps to the definition of the type of the symbol
// at the cursor, e.g., the type of a variable, using the language server.
func (cv *Code) GotoTypeDefinition() { //types:add
	cv.gotoLocations("Go to type definition", (*lsp.Backend).TypeDefinition)
}

// FindImplementations lists the implementations of the interface or
// method at the cursor, or the interfaces that the type at the cursor
// implements, using the language server, in a menu for selection.
func (cv *Code) FindImplementations() { //types:add
	cv.gotoLocations("Find implementations", (*lsp.Backend).Implementations)
}

// gotoLocations does the given location request for the symbol at the
// cursor, going to the location if there is one, and otherwise showing
// a menu of the locations. The cmd is the name of the command.
func (cv *Code) gotoLocations(cmd string, req func(*lsp.Backend, *lines.Lines, textpos.Pos) []lang.Location) {
	tv := cv.lspEditor(cmd)
	if tv == nil {
		return
	}
	pos, ok := cursorWordPos(tv)
	if !ok {
		return
	}
	cv.SetStatus(cmd + "...")
	locs := req(cv.LangSvc().lspB, tv.Lines, pos)
	switch len(locs) {
	case 0:
		cv.SetStatus(cmd + ": none found")
	case 1:
		cv.openLoc(locs[0])
	default:
		cv.locationsMenu(tv, locs)
	}
}

// locationsMenu shows a menu of the given locations in the editor,
// opening the selected one.
func (cv *Code) locationsMenu(tv *TextEditor, locs []lang.Location) {
	texts := make([]string, len(locs))
	for i, lc := range locs {
		texts[i] = locString(cv, lc)
	}
	m := core.NewMenuFromStrings(texts, texts[0], func(idx int) {
		cv.openLoc(locs[idx])
	})
	core.NewMenuStage(m, tv, tv.ContextMenuPos(nil)).Run()
}

// openLoc opens a file at the given location, selecting the region.
func (cv *Code) openLoc(lc lang.Location) {
	if _, ok := cv.OpenFileAtRegion(lc.Filename, lc.Region); !ok {
		cv.SetStatus("Could not open file: " + lc.Filename)
	}
}

// locString returns a display string for a location,
// with file:line plus a trimmed preview of the line.
func locString(cv *Code, lc lang.Location) string {
	// files outside the project (e.g., in GOROOT) show their directory
	// name, and deep paths within the project are abbreviated:
	nm := filepath.Join(filepath.Base(filepath.Dir(lc.Filename)), filepath.Base(lc.Filename))
	if rel, err := filepath.Rel(string(cv.Settings.ProjectRoot), lc.Filename); err == nil && !strings.HasPrefix(rel, "..") {
		nm = rel
		if dir := filepath.Dir(rel); strings.Count(dir, string(filepath.Separator)) > 1 {
			nm = filepath.Join("...", filepath.Base(dir), filepath.Base(rel))
		}
	}
	txt := ""
	if ln := cv.GetOpenFile(lc.Filename); ln != nil && ln.IsValidLine(lc.Region.Start.Line) {
		rn := []rune(strings.TrimSpace(string(ln.Line(lc.Region.Start.Line))))
		if len(rn) > 60 {
			rn = append(rn[:60], '…')
		}
		txt = string(rn)
	}
	if txt != "" {
		return fmt.Sprintf("%s:%d: %s", nm, lc.Region.Start.Line+1, txt)
	}
	return fmt.Sprintf("%s:%d", nm, lc.Region.Start.Line+1)
}

////////   Symbols

// WorkspaceSymbolPrompt prompts for a symbol name and shows matching
// symbols from the entire workspace project, using the language server.
func (cv *Code) WorkspaceSymbolPrompt() { //types:add
	tv := cv.lspEditor("Workspace symbols")
	if tv == nil {
		return
	}
	d := core.NewBody("Workspace symbols")
	tf := core.NewTextField(d).SetPlaceholder("symbol name to search")
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Search").OnClick(func(e events.Event) {
			cv.workspaceSymbolsMenu(tv, tf.Text())
		})
	})
	d.RunWindowDialog(tv)
}

// workspaceSymbolsMenu shows a menu of the workspace symbols matching
// the query, selecting the chosen one.
func (cv *Code) workspaceSymbolsMenu(tv *TextEditor, query string) {
	sl := cv.LangSvc().lspB.WorkspaceSymbols(tv.Lines, query)
	if len(sl) == 0 {
		cv.SetStatus(fmt.Sprintf("Workspace symbols: none found matching %q", query))
		return
	}
	items := langSymItems(cv, sl)
	core.NewMenu(func(m *core.Scene, mp image.Point) {
		for _, it := range items {
			core.NewButton(m).SetText(it.Text).SetIcon(it.Icon).OnClick(func(e events.Event) {
				it.Func()
			})
		}
	}, tv, tv.ContextMenuPos(nil)).Run()
}

// langSymItems returns chooser items for the given symbols,
// which select the symbol when chosen.
func langSymItems(cv *Code, sl []lang.Symbol) []core.ChooserItem {
	items := make([]core.ChooserItem, len(sl))
	for i, sy := range sl {
		txt := sy.Name
		if sy.Detail != "" {
			txt += "  [" + sy.Detail + "]"
		}
		items[i] = core.ChooserItem{
			Text: txt,
			Icon: sy.Kind.Icon(),
			Func: func() {
				SelectSymbol(cv, LangSymbolToSyms(sy))
			},
		}
	}
	return items
}

// LangSymbolToSyms converts a [lang.Symbol] to the [syms.Symbol]
// representation used by the symbols panels, recursively.
func LangSymbolToSyms(sy lang.Symbol) syms.Symbol {
	s := syms.Symbol{
		Name:      sy.Name,
		Detail:    sy.Detail,
		Kind:      sy.Kind,
		Filename:  sy.Filename,
		Region:    sy.Region,
		SelectReg: sy.Region,
	}
	for _, ch := range sy.Children {
		c := LangSymbolToSyms(ch)
		s.Children.Add(&c)
	}
	return s
}

////////   Formatting and rename

// LangFormat formats the current file using the active language
// backend, if it supports formatting (e.g., gopls for Go).
func (cv *Code) LangFormat() { //types:add
	tv := cv.lspEditor("Format")
	if tv == nil {
		return
	}
	ln := tv.Lines
	ls := cv.LangSvc()
	// run synchronously on the UI goroutine, as for GotoDefinition and
	// FindReferences: Format applies edits with self-locking Lines
	// methods, and the server round trip is bounded by its own timeout.
	if !ls.lspB.Format(ln) {
		cv.SetStatus("Format: no edits")
		return
	}
	ls.Change(ln) // keep server in sync with new text
	tv.NeedsRender()
	cv.SetStatus("Formatted: " + filepath.Base(ln.Filename()))
}

// RenameSymbol prompts for a new name and renames the symbol at the
// cursor across the workspace, using the language server. The proposed
// edits are shown in the Find tab for review before applying: click a
// result to jump to it, Replace for one at a time, or All for all
// (open buffers are edited, unopened files are opened; save to write).
func (cv *Code) RenameSymbol() { //types:add
	tv := cv.lspEditor("Rename")
	if tv == nil {
		return
	}
	pos, ok := cursorWordPos(tv)
	if !ok {
		return
	}
	word := wordAtCursor(tv)
	if word == "" {
		cv.SetStatus("Rename: no symbol at cursor")
		return
	}
	d := core.NewBody("Rename " + word + " to")
	tf := core.NewTextField(d).SetPlaceholder("new name")
	tf.SetText(word)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Rename").OnClick(func(e events.Event) {
			newName := strings.TrimSpace(tf.Text())
			if newName != "" && newName != word {
				cv.LangSvc().previewRename(tv.Lines, pos, word, newName)
			}
		})
	})
	d.RunWindowDialog(tv)
}

// previewRename computes the workspace edits that rename the symbol at
// pos to newName, and shows them in the Find tab for review: clicking
// a result jumps to it, Replace applies one at a time, and All applies
// them all (with confirmation). This reuses [FindPanel] in rename mode.
func (ls *LangService) previewRename(ln *lines.Lines, pos textpos.Pos, old, newName string) {
	cv := ls.cv
	// synchronous, as for GotoDefinition: bounded server timeouts.
	cv.SetStatus("Renaming " + old + "...")
	we, err := ls.lspB.RenameEdits(ln, pos, newName)
	if err != nil {
		cv.SetStatus("Rename: " + err.Error())
		return
	}
	prev := ls.lspB.PreviewWorkspaceEdit(we)
	if len(prev) == 0 {
		cv.SetStatus("Rename: nothing to rename")
		return
	}
	tv := cv.Tabs()
	if tv == nil {
		return
	}
	fv := core.RecycleTabWidget[FindPanel](tv, "Find")
	fv.UpdateTree()
	fbuf, _ := cv.RecycleCmdBuf("Find")
	fv.TextEditor().SetLines(fbuf)
	fv.Time = time.Now()
	fv.Renaming = true
	fv.Renamed = 0
	fv.OnReplace = func(etv *TextEditor) {
		ls.Change(etv.Lines) // keep the language server in sync
	}
	fp := &cv.Settings.Find
	fp.Find = old
	fp.Replace = newName
	fp.IgnoreCase = false
	res := renameResults(prev)
	fv.ShowResults(res)
	cv.FocusOnPanel(TabsIndex)
	cv.SetStatus(fmt.Sprintf("Rename %s to %s: review %d edits in %d files -- Replace steps through, All applies", old, newName, len(prev), len(res)))
}

// renameResults converts rename edit previews into [search.Results]
// for display and stepwise application in the Find panel. The previews
// must be sorted by filename, as [lsp.Backend.PreviewWorkspaceEdit]
// returns them, so that the matches group by file.
func renameResults(prev []*lsp.EditPreview) []search.Results {
	var res []search.Results
	var cur *search.Results
	for _, it := range prev {
		if cur == nil || cur.Filepath != it.Filename {
			res = append(res, search.Results{Filepath: it.Filename})
			cur = &res[len(res)-1]
		}
		txt := []rune(it.LineText)
		st := min(it.Region.Start.Char, len(txt))
		ed := it.Region.End.Char
		if it.Region.End.Line != it.Region.Start.Line || ed > len(txt) {
			ed = len(txt) // multi-line region: show to end of first line
		}
		m := textpos.Match{Region: it.Region, Text: txt}
		if st <= ed {
			m.TextMatch = textpos.Range{Start: st, End: ed}
		}
		cur.Matches = append(cur.Matches, m)
		cur.Count++
	}
	return res
}

////////   Signature help and inlay hints

// ShowSignatureHelp explicitly requests signature help at the cursor
// and shows all of the signatures in a dialog, for cases where the
// automatic display is not desired or the cursor is not in a call
// context.
func (cv *Code) ShowSignatureHelp() { //types:add
	tv := cv.lspEditor("Signature help")
	if tv == nil || !tv.Lines.IsValidLine(tv.CursorPos.Line) {
		return
	}
	ls := cv.LangSvc()
	ls.Change(tv.Lines)
	sh, act, err := ls.lspB.SignatureHelp(tv.Lines, tv.CursorPos, false)
	if err != nil {
		cv.SetStatus("Signature help: " + err.Error())
		return
	}
	if sh == nil {
		cv.SetStatus("Signature help: no callable at cursor")
		return
	}
	// markdown, with the signatures as code:
	var sb strings.Builder
	for i, sig := range sh.Signatures {
		mark, unmark := "", ""
		if i == act {
			mark, unmark = "[", "]"
		}
		if i > 0 {
			sb.WriteString("\n---\n")
		}
		sb.WriteString("```go\n" + lsp.SignatureText(sh, i, sh.ActiveParam(), mark, unmark) + "\n```\n")
		if sig.Documentation != nil && sig.Documentation.Value != "" {
			sb.WriteString("\n" + sig.Documentation.Value + "\n")
		}
	}
	name, _, _ := strings.Cut(sh.Signatures[max(act, 0)].Label, "(")
	docsDialog(tv, signatureWindow+": "+name, sb.String())
}

// goplsInlayHints are the gopls inlay hint categories to enable when
// the InlayHints setting is on, from gopls internal settings
// (ui.inlayhint.hints map). gopls only returns hints for categories
// explicitly enabled, and its default is an empty map.
var goplsInlayHints = map[string]any{
	"parameterNames":         true,
	"assignVariableTypes":    true,
	"constantValues":         true,
	"rangeVariableTypes":     true,
	"compositeLiteralTypes":  true,
	"compositeLiteralFields": true,
	"functionTypeParameters": true,
	"ignoredError":           true,
}

// serverOptions returns the additional per-server options for the
// given inlay-hints setting (server ID -> options). Only gopls is
// configured currently; other servers gate hints differently and
// return none, which is a valid empty result.
func serverOptions(inlayHints bool) map[string]map[string]any {
	if !inlayHints {
		return nil
	}
	return map[string]map[string]any{
		"gopls": {"hints": goplsInlayHints},
	}
}

// SetInlayHints turns the inlay hints setting on or off, applying it to
// the running language servers: already-running gopls re-fetches its
// settings through workspace/configuration (gopls supports dynamic
// configuration), so no restart is needed for it; servers that only
// read options at startup pick the change up when they next start.
// The server work runs in the background.
func (cv *Code) SetInlayHints(on bool) { //types:add
	if cv.Settings.InlayHints == on {
		return
	}
	cv.Settings.InlayHints = on
	ls := cv.LangSvc()
	go func() {
		ls.lspB.SetOptions(serverOptions(on))
		ls.lspB.Reconfigure()
	}()
	cv.SetStatus(fmt.Sprintf("Inlay hints: %v (displayed by the Show Inlay Hints command)", on))
}

// ShowInlayHints requests the language server's inlay hints for the
// current file and lists them in a dialog, as a display-only overview:
// nothing is inserted into the buffer (the per-hint TextEdits that would
// insert the label text are ignored). Requires the InlayHints setting to
// be on for servers like gopls that gate hints behind settings.
func (cv *Code) ShowInlayHints() { //types:add
	tv := cv.lspEditor("Inlay hints")
	if tv == nil {
		return
	}
	if !cv.Settings.InlayHints {
		cv.SetStatus("Inlay hints: enable the InlayHints setting first (servers gate these behind settings)")
		return
	}
	ln := tv.Lines
	ls := cv.LangSvc()
	ls.Change(ln)
	hints, err := ls.lspB.InlayHints(ln, textpos.Region{End: textpos.Pos{Line: ln.NumLines()}})
	if err != nil {
		cv.SetStatus("Inlay hints: " + err.Error())
		return
	}
	var sb strings.Builder
	n := 0
	for _, h := range hints {
		if txt := h.LabelText(); txt != "" {
			fmt.Fprintf(&sb, "%d:%d: %s\n", h.Position.Line+1, h.Position.Character+1, txt)
			n++
		}
	}
	if n == 0 {
		cv.SetStatus("Inlay hints: none returned by the server for this file")
		return
	}
	textcore.TextDialog(tv, fmt.Sprintf("%s: %s (%d)", inlayHintsWindow, filepath.Base(ln.Filename()), n), sb.String())
}

////////   Selection

// ExpandSelection expands the selection to the next larger syntactic
// element around it, using the language server: word, expression,
// statement, block, function, etc.
func (cv *Code) ExpandSelection() { //types:add
	if tv := cv.lspEditor("Expand selection"); tv != nil && !cv.LangSvc().ExpandSelection(tv) {
		cv.SetStatus("Expand selection: no larger element")
	}
}

// ShrinkSelection shrinks the selection back to the next smaller
// syntactic element, after it has been expanded with [Code.ExpandSelection].
func (cv *Code) ShrinkSelection() { //types:add
	if tv := cv.lspEditor("Shrink selection"); tv != nil {
		cv.LangSvc().ShrinkSelection(tv)
	}
}

// selectionChain returns the recorded chain of nested selection regions
// and the index of the selected one, if it is for the given editor and
// selection, and otherwise nil.
func (ls *LangService) selectionChain(tv *TextEditor, sel textpos.Region) ([]textpos.Region, int) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	if ls.selEditor != tv || ls.selIndex >= len(ls.selChain) || ls.selChain[ls.selIndex] != sel {
		return nil, -1
	}
	return ls.selChain, ls.selIndex
}

// ExpandSelection does [Code.ExpandSelection] in the given editor,
// returning false if there is no larger element.
func (ls *LangService) ExpandSelection(tv *TextEditor) bool {
	sel := tv.SelectRegion
	if !tv.HasSelection() {
		sel = textpos.Region{Start: tv.CursorPos, End: tv.CursorPos}
	}
	chain, idx := ls.selectionChain(tv, sel)
	if chain == nil {
		chain = ls.lspB.SelectionRanges(tv.Lines, sel.Start)
	}
	for i := idx + 1; i < len(chain); i++ {
		if r := chain[i]; regionContains(r, sel) && r != sel {
			ls.setSelection(tv, chain, i)
			return true
		}
	}
	return false
}

// ShrinkSelection does [Code.ShrinkSelection] in the given editor.
func (ls *LangService) ShrinkSelection(tv *TextEditor) {
	chain, idx := ls.selectionChain(tv, tv.SelectRegion)
	switch {
	case chain == nil: // not an expanded selection
	case idx == 0:
		tv.SelectReset()
		tv.NeedsRender()
	default:
		ls.setSelection(tv, chain, idx-1)
	}
}

// setSelection selects the region at the given index in the given chain
// of nested selection regions in the editor, and records the chain.
func (ls *LangService) setSelection(tv *TextEditor, chain []textpos.Region, idx int) {
	ls.mu.Lock()
	ls.selChain, ls.selEditor, ls.selIndex = chain, tv, idx
	ls.mu.Unlock()
	reg := chain[idx]
	tv.SetCursorShow(reg.End)
	tv.SelectRegion = reg
	tv.NeedsRender()
}

// regionContains returns whether region a contains region b.
func regionContains(a, b textpos.Region) bool {
	return !b.Start.IsLess(a.Start) && !a.End.IsLess(b.End)
}
