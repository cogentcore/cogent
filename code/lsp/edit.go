// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"cmp"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/textpos"

	"cogentcore.org/cogent/code/lang"
)

////////   Formatting

// Format requests formatting edits from the server and applies them
// to the buffer. Returns true if any edits applied.
func (b *Backend) Format(ln *lines.Lines) bool {
	srv, ok := b.readyServer(ln)
	if !ok {
		return false
	}
	uri := docID(ln).URI
	srv.DidChange(uri, currentText(ln))
	edits, err := srv.Formatting(uri, FormattingOptions{TabSize: 4, InsertSpaces: false})
	if err != nil {
		return false
	}
	return applyTextEdits(ln, edits)
}

// sortEdits sorts the edits by start position, keeping the order of
// edits at the same position, which the spec says must be applied in
// the given order.
func sortEdits(edits []TextEdit) {
	slices.SortStableFunc(edits, func(a, b TextEdit) int {
		return cmp.Or(cmp.Compare(a.Range.Start.Line, b.Range.Start.Line),
			cmp.Compare(a.Range.Start.Character, b.Range.Start.Character))
	})
}

// applyTextEdits applies the given LSP edits to the buffer as one batch
// update, which is undone as one, in reverse position order so earlier
// positions stay valid: their regions can all be converted first.
// It only uses self-locking Lines methods, so it is safe from any
// goroutine (not under ln.Lock). It returns whether any edits were
// applied. The edits are sorted in place.
func applyTextEdits(ln *lines.Lines, edits []TextEdit) bool {
	sortEdits(edits)
	reps := make([]lines.Replacement, 0, len(edits))
	for _, e := range slices.Backward(edits) {
		reps = append(reps, lines.Replacement{Region: regionFromLSP(ln, e.Range), Text: e.NewText})
	}
	return ln.ReplaceTexts(reps, lines.ReplaceNoMatchCase) > 0
}

// convertEdits converts LSP edits to editor units for the buffer.
func convertEdits(ln *lines.Lines, edits []TextEdit) []lang.TextEdit {
	out := make([]lang.TextEdit, len(edits))
	for i, e := range edits {
		out[i] = lang.TextEdit{Region: regionFromLSP(ln, e.Range), NewText: e.NewText}
	}
	return out
}

////////   docText

// docText is a lightweight line-indexed view of file contents, for
// converting and applying LSP edits in files that are not open in an
// editor buffer, without the overhead of a [lines.Lines], which does
// syntax highlighting markup.
type docText struct {

	// text is the full text.
	text string

	// lines are the lines of the text, without line terminators.
	lines []string

	// starts are the byte offsets of the line starts in text.
	starts []int
}

// newDocText returns a new docText for the given text.
func newDocText(text string) *docText {
	dt := &docText{text: text}
	st := 0
	for {
		dt.starts = append(dt.starts, st)
		i := strings.IndexByte(text[st:], '\n')
		if i < 0 {
			dt.lines = append(dt.lines, text[st:])
			return dt
		}
		dt.lines = append(dt.lines, strings.TrimSuffix(text[st:st+i], "\r"))
		st += i + 1
	}
}

// readDocText returns a new docText for the contents of the given file.
func readDocText(fname string) (*docText, error) {
	b, err := os.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	return newDocText(string(b)), nil
}

// pos converts an LSP position to editor (rune) units.
// Positions past the last line map to the end of the text.
func (dt *docText) pos(p Position) textpos.Pos {
	switch {
	case p.Line < 0:
		return textpos.Pos{}
	case p.Line >= len(dt.lines):
		last := len(dt.lines) - 1
		return textpos.Pos{Line: last, Char: len([]rune(dt.lines[last]))}
	}
	return textpos.Pos{Line: p.Line, Char: lang.UTF16ToRune(dt.lines[p.Line], p.Character)}
}

// region converts an LSP range to editor (rune) units.
func (dt *docText) region(r Range) textpos.Region {
	st, ed := dt.pos(r.Start), dt.pos(r.End)
	if ed.IsLess(st) {
		ed = st
	}
	return textpos.Region{Start: st, End: ed}
}

// offset returns the byte offset in the text of the given LSP position.
func (dt *docText) offset(p Position) int {
	switch {
	case p.Line < 0:
		return 0
	case p.Line >= len(dt.lines):
		return len(dt.text)
	}
	line := dt.lines[p.Line]
	rpos := lang.UTF16ToRune(line, p.Character)
	n := 0
	for i := range line {
		if n == rpos {
			return dt.starts[p.Line] + i
		}
		n++
	}
	return dt.starts[p.Line] + len(line)
}

// rangeText returns the text in the given LSP range.
func (dt *docText) rangeText(r Range) string {
	st := dt.offset(r.Start)
	return dt.text[st:max(st, dt.offset(r.End))]
}

// apply returns the text with the given edits applied. Per the spec,
// edits must not overlap: any that do are skipped. The edits are
// sorted in place.
func (dt *docText) apply(edits []TextEdit) string {
	sortEdits(edits)
	var sb strings.Builder
	prev := 0
	for _, e := range edits {
		st := dt.offset(e.Range.Start)
		ed := max(st, dt.offset(e.Range.End))
		if st < prev {
			log.Printf("lsp: skipping overlapping edit at %v\n", e.Range.Start)
			continue
		}
		sb.WriteString(dt.text[prev:st])
		sb.WriteString(e.NewText)
		prev = ed
	}
	sb.WriteString(dt.text[prev:])
	return sb.String()
}

// convertEdits converts LSP edits to editor units for the text.
func (dt *docText) convertEdits(edits []TextEdit) []lang.TextEdit {
	out := make([]lang.TextEdit, len(edits))
	for i, e := range edits {
		out[i] = lang.TextEdit{Region: dt.region(e.Range), NewText: e.NewText}
	}
	return out
}

////////   Workspace edits

// handleApplyEdit implements server-initiated workspace/applyEdit
// requests. Edits are applied synchronously: applying only uses
// self-locking Lines methods and disk io, the same safety pattern as
// diagnostics handling. Affected views are refreshed asynchronously
// via the [Backend.AppliedEdits] channel.
func (b *Backend) handleApplyEdit(we WorkspaceEdit) (bool, string) {
	out := b.ApplyWorkspaceEdit(&we)
	if len(out) == 0 {
		return false, "no edits applied"
	}
	b.mu.Lock()
	if b.inCommand {
		b.cmdEdits = append(b.cmdEdits, out...)
	}
	b.mu.Unlock()
	b.sendMu.RLock()
	defer b.sendMu.RUnlock()
	if !b.closed {
		select {
		case b.applies <- out:
		default:
			log.Println("lsp: applyEdit refresh notification dropped: buffer full")
		}
	}
	return true, ""
}

// fileEdits is the normalized per-file group of LSP edits,
// merged from the changes map and documentChanges shapes of a
// WorkspaceEdit.
type fileEdits struct {
	uri   DocumentURI
	edits []TextEdit
}

// normalizeWE flattens a WorkspaceEdit into per-file edit groups,
// merging entries for the same document across both the changes map
// and the documentChanges array, and dropping empty groups.
func normalizeWE(we *WorkspaceEdit) []fileEdits {
	var fes []fileEdits
	idx := map[DocumentURI]int{}
	add := func(uri DocumentURI, edits []TextEdit) {
		if len(edits) == 0 || uri == "" { // resource operations have no edits
			return
		}
		if i, has := idx[uri]; has {
			fes[i].edits = append(fes[i].edits, edits...)
			return
		}
		idx[uri] = len(fes)
		fes = append(fes, fileEdits{uri: uri, edits: slices.Clone(edits)})
	}
	for uri, es := range we.Changes {
		add(DocumentURI(uri), es)
	}
	for _, de := range we.DocumentChanges {
		add(de.TextDocument.URI, de.Edits)
	}
	return fes
}

// EditPreview is one proposed workspace edit, resolved against the
// current contents of the target file, for review before applying.
type EditPreview struct {

	// Filename is the normalized path of the target file.
	Filename string

	// URI is the document uri the server used for this file.
	URI DocumentURI

	// Open indicates the file was open in a tracked editor buffer
	// when the preview was computed.
	Open bool

	// Region is the target region in editor (rune) units,
	// relative to the file contents at preview time. Set the Region
	// Time to keep it valid across subsequent edits via AdjustRegion.
	Region textpos.Region

	// Old is the current text in the region, for display and for
	// verifying the region before applying.
	Old string

	// New is the replacement text.
	New string

	// LineText is the text of the first line of the region,
	// for display as context.
	LineText string
}

// PreviewWorkspaceEdit converts a workspace edit into per-edit previews
// resolved against current file contents, without applying anything.
// It returns the previews sorted by filename and position.
func (b *Backend) PreviewWorkspaceEdit(we *WorkspaceEdit) []*EditPreview {
	if we == nil {
		return nil
	}
	var out []*EditPreview
	for _, fe := range normalizeWE(we) {
		fname := fe.uri.Path()
		if fname == "" {
			continue
		}
		norm := normalizePath(fname)
		if ln := b.openLines(fname); ln != nil {
			for _, e := range fe.edits {
				r := regionFromLSP(ln, e.Range)
				out = append(out, &EditPreview{
					Filename: norm,
					URI:      fe.uri,
					Open:     true,
					Region:   r,
					Old:      string(ln.Region(r.Start, r.End).ToBytes()),
					New:      e.NewText,
					LineText: lineText(ln, r.Start.Line),
				})
			}
			continue
		}
		dt, err := readDocText(fname)
		if err != nil {
			log.Printf("lsp: preview: %v\n", err)
			continue
		}
		for _, e := range fe.edits {
			r := dt.region(e.Range)
			out = append(out, &EditPreview{
				Filename: norm,
				URI:      fe.uri,
				Region:   r,
				Old:      dt.rangeText(e.Range),
				New:      e.NewText,
				LineText: dt.lines[r.Start.Line],
			})
		}
	}
	slices.SortStableFunc(out, func(a, b *EditPreview) int {
		return cmp.Or(strings.Compare(a.Filename, b.Filename),
			cmp.Compare(a.Region.Start.Line, b.Region.Start.Line),
			cmp.Compare(a.Region.Start.Char, b.Region.Start.Char))
	})
	return out
}

// ApplyWorkspaceEdit applies a workspace edit from a language server:
// to the live buffers for files that are open, and by rewriting the
// files on disk otherwise (open buffers are left modified for the user
// to save, as in other editors). It returns the edits actually applied,
// per file, keyed by normalized filename.
func (b *Backend) ApplyWorkspaceEdit(we *WorkspaceEdit) []lang.FileEdits {
	if we == nil {
		return nil
	}
	var out []lang.FileEdits
	for _, de := range we.DocumentChanges {
		if de.Kind != "create" {
			continue
		}
		fname, err := createFile(de)
		if err != nil {
			log.Printf("lsp: applyEdit: %v\n", err)
		} else if fname != "" {
			out = append(out, lang.FileEdits{Filename: normalizePath(fname)})
		}
	}
	for _, fe := range normalizeWE(we) {
		fname := fe.uri.Path()
		if fname == "" {
			continue
		}
		b.mu.Lock()
		ref := b.docs[normalizePath(fname)]
		b.mu.Unlock()
		var edits []lang.TextEdit
		if ref != nil {
			edits = convertEdits(ref.ln, fe.edits)
			if !applyTextEdits(ref.ln, fe.edits) {
				continue
			}
			ref.srv.DidChange(fe.uri, currentText(ref.ln))
		} else {
			var err error
			if edits, err = applyFileEdits(fname, fe.edits); err != nil {
				log.Printf("lsp: applyEdit: %v\n", err)
				continue
			}
		}
		norm := normalizePath(fname)
		if i := slices.IndexFunc(out, func(f lang.FileEdits) bool { return f.Filename == norm }); i >= 0 {
			out[i].Edits = append(out[i].Edits, edits...) // just created
			continue
		}
		out = append(out, lang.FileEdits{Filename: norm, Edits: edits})
	}
	slices.SortFunc(out, func(a, b lang.FileEdits) int { return strings.Compare(a.Filename, b.Filename) })
	return out
}

// createFile performs the given create file resource operation,
// returning the name of the file if it was created.
func createFile(de DocumentEdit) (string, error) {
	fname := de.URI.Path()
	if fname == "" {
		return "", fmt.Errorf("create: not a file: %q", de.URI)
	}
	if _, err := os.Stat(fname); err == nil && (de.Options == nil || !de.Options.Overwrite) {
		return "", nil // exists: ignored
	}
	if err := os.MkdirAll(filepath.Dir(fname), 0755); err != nil {
		return "", err
	}
	return fname, os.WriteFile(fname, nil, 0644)
}

// applyFileEdits applies the given edits to the file on disk,
// returning the edits in editor units.
func applyFileEdits(fname string, edits []TextEdit) ([]lang.TextEdit, error) {
	st, err := os.Stat(fname)
	if err != nil {
		return nil, err
	}
	dt, err := readDocText(fname)
	if err != nil {
		return nil, err
	}
	out := dt.convertEdits(edits)
	if err := os.WriteFile(fname, []byte(dt.apply(edits)), st.Mode().Perm()); err != nil {
		return nil, err
	}
	return out, nil
}

////////   Rename

// RenameEdits requests the workspace edit for renaming the symbol at
// pos to newName. It returns an error if there is nothing to rename.
func (b *Backend) RenameEdits(ln *lines.Lines, pos textpos.Pos, newName string) (*WorkspaceEdit, error) {
	srv, ok := b.readyServer(ln)
	if !ok {
		return nil, fmt.Errorf("language server not ready for %s", filepath.Base(ln.Filename()))
	}
	pp := posParams(ln, pos)
	srv.DidChange(pp.TextDocument.URI, currentText(ln))
	// note: prepareRename is not used to validate the position, because
	// it is advisory: it returns null both for genuinely non-renameable
	// positions and for recoverable conditions such as package errors or
	// symbols that are not yet analyzed, while rename itself answers with
	// a specific reason on failure.
	we, err := srv.Rename(&RenameParams{TextDocumentPositionParams: pp, NewName: newName})
	if err != nil {
		return nil, err
	}
	if we == nil {
		return nil, fmt.Errorf("cannot rename the symbol at the cursor (nothing to rename)")
	}
	return we, nil
}

// Rename renames the symbol at pos to newName across the workspace,
// applying the resulting edits to open buffers and files on disk
// (see [Backend.ApplyWorkspaceEdit]). It returns the edits applied,
// per file.
func (b *Backend) Rename(ln *lines.Lines, pos textpos.Pos, newName string) ([]lang.FileEdits, error) {
	we, err := b.RenameEdits(ln, pos, newName)
	if err != nil {
		return nil, err
	}
	out := b.ApplyWorkspaceEdit(we)
	if len(out) == 0 {
		return nil, fmt.Errorf("rename produced no edits")
	}
	return out, nil
}

////////   Code actions

// sevToLSP converts an editor severity to the LSP diagnostic severity.
func sevToLSP(s lang.DiagSeverity) DiagnosticSeverity {
	switch s {
	case lang.DiagWarn:
		return SeverityWarning
	case lang.DiagInfo:
		return SeverityInformation
	case lang.DiagHint:
		return SeverityHint
	}
	return SeverityError
}

// toLSPDiags converts cached editor diagnostics into LSP form for the
// code action context.
func toLSPDiags(ln *lines.Lines, diags []lang.Diagnostic) []Diagnostic {
	out := make([]Diagnostic, len(diags))
	for i, d := range diags {
		out[i] = Diagnostic{
			Range:    toLSPRange(ln, d.Region),
			Severity: sevToLSP(d.Severity),
			Source:   d.Source,
			Message:  d.Msg,
		}
		if d.Code != "" {
			out[i].Code = d.Code
		}
	}
	return out
}

// CodeActions returns the quick fixes and refactorings the server
// offers for the given region. The cached diags are passed as context,
// which is what enables the quick fixes: servers generally only offer
// fixes for diagnostics included in the request.
func (b *Backend) CodeActions(ln *lines.Lines, reg textpos.Region, diags []lang.Diagnostic) ([]CodeAction, error) {
	return b.codeActions(ln, reg, diags, false)
}

// CodeActionsQuick is CodeActions with the short, latency-sensitive
// timeout for hover and other interactive paths.
func (b *Backend) CodeActionsQuick(ln *lines.Lines, reg textpos.Region, diags []lang.Diagnostic) ([]CodeAction, error) {
	return b.codeActions(ln, reg, diags, true)
}

func (b *Backend) codeActions(ln *lines.Lines, reg textpos.Region, diags []lang.Diagnostic, quick bool) ([]CodeAction, error) {
	srv, ok := b.readyServer(ln)
	if !ok {
		return nil, fmt.Errorf("language server not ready for %s", filepath.Base(ln.Filename()))
	}
	if !srv.supportsCodeAction() {
		return nil, nil
	}
	uri := docID(ln).URI
	srv.DidChange(uri, currentText(ln))
	params := &CodeActionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Range:        toLSPRange(ln, reg),
		Context: CodeActionContext{
			Diagnostics: toLSPDiags(ln, diags),
			TriggerKind: CodeActionTriggerInvoked,
		},
	}
	if quick {
		return srv.CodeActionQuick(params)
	}
	return srv.CodeAction(params)
}

// RunCodeAction applies a code action: inline edits directly, resolving
// deferred edits via codeAction/resolve, and otherwise running the
// action's command, which can return a WorkspaceEdit, or apply its edits
// through workspace/applyEdit requests (as gopls does). Workspace edits
// are applied to open buffers and files on disk, as for renames
// (see [Backend.ApplyWorkspaceEdit]). It returns the edits applied,
// which can be none for a command that does something else, such as
// showing a document.
func (b *Backend) RunCodeAction(ln *lines.Lines, ca *CodeAction) ([]lang.FileEdits, error) {
	srv, ok := b.readyServer(ln)
	if !ok {
		return nil, fmt.Errorf("language server not ready for %s", filepath.Base(ln.Filename()))
	}
	var out []lang.FileEdits
	if ca.Edit == nil {
		// gopls can run the command for the action on resolve,
		// applying its edits through workspace/applyEdit:
		var err error
		out = b.captureEdits(func() { err = srv.ResolveCodeAction(ca) })
		if err != nil && ca.Command == nil {
			return out, fmt.Errorf("code action %q: %w", ca.Title, err)
		}
	}
	if ca.Edit != nil {
		out = append(out, b.ApplyWorkspaceEdit(ca.Edit)...)
	}
	if ca.Command != nil && ca.Command.Command != "" {
		cmdOut, err := b.runCommand(srv, ca.Command)
		if err != nil {
			return out, fmt.Errorf("code action %q: %w", ca.Title, err)
		}
		return append(out, cmdOut...), nil
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("code action %q produced no edits", ca.Title)
	}
	return out, nil
}

// captureEdits calls the given function, which makes a request to a
// server, returning the edits applied by any workspace/applyEdit
// requests that the server makes while handling it.
func (b *Backend) captureEdits(fun func()) []lang.FileEdits {
	b.mu.Lock()
	b.inCommand = true
	b.cmdEdits = nil
	b.mu.Unlock()
	fun()
	b.mu.Lock()
	defer b.mu.Unlock()
	out := b.cmdEdits
	b.inCommand = false
	b.cmdEdits = nil
	return out
}

// IsQuickFix returns whether the code action is a quick fix for a
// problem, as opposed to a refactoring or source action.
func (ca *CodeAction) IsQuickFix() bool {
	return ca.Kind == "quickfix" || strings.HasPrefix(ca.Kind, "quickfix.")
}

// SortCodeActions sorts the code actions for presentation: enabled
// before disabled, then quick fixes (preferred first), refactorings,
// source actions, and finally general documentation, such as the
// gopls feature documentation.
func SortCodeActions(acts []CodeAction) {
	rank := func(ca *CodeAction) int {
		switch {
		case ca.Disabled != nil:
			return 6
		case ca.IsQuickFix() && ca.IsPreferred:
			return 0
		case ca.IsQuickFix():
			return 1
		case strings.HasPrefix(ca.Kind, "refactor"):
			return 2
		case strings.HasPrefix(ca.Kind, "source"):
			return 3
		case strings.HasPrefix(ca.Kind, "gopls.doc"):
			return 5
		}
		return 4
	}
	slices.SortStableFunc(acts, func(a, b CodeAction) int { return cmp.Compare(rank(&a), rank(&b)) })
}

// codeActionKindDescs describe what code actions of each kind do,
// including the gopls-specific kinds, by kind prefix.
var codeActionKindDescs = []struct{ kind, desc string }{
	{"quickfix", "Fixes the problem"},
	{"refactor.extract", "Refactoring: extracts the selected code into a new function, method, variable, or constant"},
	{"refactor.inline", "Refactoring: inlines the function call or variable at the cursor"},
	{"refactor.rewrite", "Refactoring: rewrites the code at the cursor in an equivalent form"},
	{"refactor.move", "Refactoring: moves the code at the cursor"},
	{"refactor", "Refactoring: restructures the code without changing its behavior"},
	{"source.organizeImports", "Adds missing imports and removes unused ones"},
	{"source.fixAll", "Applies all of the automatic fixes for the file"},
	{"source.addTest", "Adds a test for this function to the corresponding _test.go file"},
	{"source.doc", "Opens the documentation for this symbol in a web browser"},
	{"source.assembly", "Opens the assembly listing for this function in a web browser"},
	{"source.freesymbols", "Opens a report of the free symbols referenced by the selected code in a web browser"},
	{"source.splitPackage", "Opens a tool for splitting this package in a web browser"},
	{"source.toggleCompilerOptDetails", "Toggles the display of compiler optimization details as diagnostics"},
	{"gopls.doc.features", "Opens the gopls feature documentation in a web browser"},
	{"source", "Source action: applies to the whole file"},
}

// Description returns a description of what the code action does,
// based on its kind, including the messages of any diagnostics that
// it fixes, for use as a tooltip.
func (ca *CodeAction) Description() string {
	desc := ""
	for _, kd := range codeActionKindDescs {
		if ca.Kind == kd.kind || strings.HasPrefix(ca.Kind, kd.kind+".") {
			desc = kd.desc
			break
		}
	}
	if desc == "" {
		desc = "Code action"
		if ca.Kind != "" {
			desc += " of kind " + ca.Kind
		}
	}
	for _, d := range ca.Diagnostics {
		desc += ": " + cleanMessage(d.Message)
	}
	if ca.Disabled != nil {
		desc += " (not available: " + ca.Disabled.Reason + ")"
	}
	return desc
}

// runCommand executes the given command on the server, returning the
// edits applied, either from a WorkspaceEdit result, or from the
// workspace/applyEdit requests the server makes while executing it.
func (b *Backend) runCommand(srv *Server, cmd *Command) ([]lang.FileEdits, error) {
	var raw json.RawMessage
	var err error
	out := b.captureEdits(func() { raw, err = srv.ExecuteCommand(cmd.Command, cmd.Arguments) })
	if err != nil {
		return out, err
	}
	if len(raw) > 0 && string(raw) != "null" {
		var we WorkspaceEdit
		if json.Unmarshal(raw, &we) == nil {
			out = append(out, b.ApplyWorkspaceEdit(&we)...)
		}
	}
	return out, nil
}
