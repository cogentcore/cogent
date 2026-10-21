// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"cmp"
	"fmt"
	"image"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
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

// Lookup shows the lookup / quick-fix chooser menu at the cursor, the
// Ctrl+. action: quick fixes and refactorings from the language server,
// plus symbol lookup (docs or go-to-definition). For LSP-backed files
// it uses the language-server lookup; otherwise it falls back to the
// editor's own parse-based lookup.
func (cv *Code) Lookup() { //types:add
	tv := cv.ActiveEditor()
	if tv == nil || tv.Lines == nil {
		return
	}
	ls := cv.LangSvc()
	if !ls.LSPActive(tv.Lines) {
		tv.Lookup()
		return
	}
	if pos, ok := cursorWordPos(tv); ok {
		ls.lookupMenu(tv, pos)
	}
}

// lookupFunc returns the editor LookupFunc for the given buffer,
// which shows the [LangService.lookupMenu].
func (ls *LangService) lookupFunc(ln *lines.Lines) complete.LookupFunc {
	return func(data any, txt string, posLine, posChar int) complete.Lookup {
		if tv, _, has := ls.cv.EditorForLines(ln); has && tv != nil && tv.Scene != nil {
			ls.lookupMenu(tv, textpos.Pos{Line: posLine, Char: posChar})
		}
		return complete.Lookup{}
	}
}

// lookupMenu shows a completer-style chooser menu at the given position
// in the editor: the quick fixes / refactorings available from the
// language server (applied on click), followed by the symbol lookup
// action: showing hover docs if the symbol is defined in the current
// file, and opening the defining file otherwise.
func (ls *LangService) lookupMenu(tv *TextEditor, pos textpos.Pos) {
	ln := tv.Lines
	if !ln.IsValidLine(pos.Line) {
		return
	}
	if tv.Complete != nil {
		tv.Complete.Cancel() // like the completer does before offering
	}
	ls.Change(ln) // make sure the server has the current text

	actions, _ := ls.lspB.CodeActionsQuick(ln, wordRegion(ln, pos), ls.diagsAt(ln, pos))
	var items []core.ChooserItem
	for _, ca := range actions {
		items = append(items, core.ChooserItem{Text: "Fix: " + actionTitle(&ca), Value: ca})
	}
	if locs := ls.lspB.Definition(ln, pos); len(locs) > 0 && locs[0].Filename != ln.Filename() {
		items = append(items, core.ChooserItem{Text: "Go to definition: " + locString(ls.cv, locs[0]), Value: locs[0]})
	} else if docs := ls.lspB.Hover(ln, pos); docs != "" {
		items = append(items, core.ChooserItem{Text: "Docs: " + firstLine(docs), Value: docs})
	}
	if len(items) == 0 {
		ls.cv.SetStatus("Lookup: nothing available at cursor")
		return
	}
	core.NewMenu(func(m *core.Scene, mp image.Point) {
		for _, it := range items {
			core.NewButton(m).SetText(it.Text).OnClick(func(e events.Event) {
				switch v := it.Value.(type) {
				case lsp.CodeAction:
					ls.applyCodeAction(ln, &v)
				case lang.Location:
					ls.cv.openLoc(v)
				case string:
					textcore.TextDialog(tv, "Docs", v)
				}
			})
		}
	}, tv, editorCursorPos(tv, pos)).Run()
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
	// preferred, enabled actions first for a useful default selection:
	slices.SortStableFunc(actions, func(a, b lsp.CodeAction) int {
		return cmp.Or(compareBool(a.Disabled == nil, b.Disabled == nil), compareBool(a.IsPreferred, b.IsPreferred))
	})
	d := core.NewBody("Quick fix / refactor")
	c := core.NewChooser(d)
	items := make([]core.ChooserItem, len(actions))
	for i := range actions {
		items[i] = core.ChooserItem{Text: actionTitle(&actions[i])}
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

// compareBool orders true before false.
func compareBool(a, b bool) int {
	switch {
	case a == b:
		return 0
	case a:
		return -1
	}
	return 1
}

////////   Navigation

// GotoDefinition jumps to the definition of the symbol at the cursor,
// using the active language backend.
func (cv *Code) GotoDefinition() { //types:add
	tv := cv.lspEditor("Goto definition")
	if tv == nil {
		return
	}
	pos, ok := cursorWordPos(tv)
	if !ok {
		return
	}
	cv.SetStatus("Goto definition...")
	locs := cv.LangSvc().lspB.Definition(tv.Lines, pos)
	switch len(locs) {
	case 0:
		cv.SetStatus("Goto definition: not found")
	case 1:
		cv.openLoc(locs[0])
	default:
		cv.locationsMenu(tv, locs)
	}
}

// FindReferences lists all references to the symbol at the cursor,
// using the active language backend, in a menu for selection.
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
	cv.locationsMenu(tv, locs)
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
	nm := lc.Filename
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
	var sb strings.Builder
	for i, sig := range sh.Signatures {
		mark, unmark := "", ""
		if i == act {
			mark, unmark = "[", "]"
		}
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(lsp.SignatureText(sh, i, sh.ActiveParam(), mark, unmark))
		if sig.Documentation != nil && sig.Documentation.Value != "" {
			sb.WriteString("\n" + sig.Documentation.Value)
		}
	}
	textcore.TextDialog(tv, "Signature help", sb.String())
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
	textcore.TextDialog(tv, fmt.Sprintf("Inlay hints for %s (%d)", filepath.Base(ln.Filename()), n), sb.String())
}
