// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"

	"cogentcore.org/cogent/code/lang"
)

// Backend drives external language servers (gopls and others) over the
// [LSP] protocol, using a [Server] process
// for each language-server / workspace-root combination that is needed.
//
// Unlike the parse system, the language servers run out-of-process and
// do not depend on the [highlighting.UseParse] setting.
//
// [LSP]: https://microsoft.github.io/language-server-protocol/specification
// [highlighting.UseParse]: https://pkg.go.dev/cogentcore.org/core/text/highlighting#UseParse
type Backend struct {

	// Reg is the language server registry, which can be configured
	// with executable path overrides from settings.
	Reg *Registry

	// RootFunc returns the workspace root directory for a given file
	// path, e.g., the project root. If it returns "", the standard
	// [ServerConfig.RootDir] marker-based search is used instead.
	RootFunc func(fpath string) string

	// Verbose enables detailed logging of the backend.
	Verbose bool

	mu      sync.Mutex
	servers map[string]*Server        // serverKey -> Server
	docs    map[string]*docRef        // filename -> current doc tracking
	edits   map[string]*complete.Edit // pending completion edits, by Text
	diags   chan lang.FileDiags
	stopped bool
}

// docRef associates a [lines.Lines] buffer with its server and language.
type docRef struct {
	ln     *lines.Lines
	srv    *Server
	langID string
	fname  string // raw filename from the buffer, as the editor knows it
}

// NewBackend returns a new Backend driven by language servers,
// using the given executable path overrides (server ID -> path),
// which can be nil to use the standard PATH lookup.
func NewBackend(execPaths map[string]string) *Backend {
	return &Backend{
		Reg:     NewRegistry(execPaths),
		servers: make(map[string]*Server),
		docs:    make(map[string]*docRef),
		edits:   make(map[string]*complete.Edit),
		diags:   make(chan lang.FileDiags, 64),
	}
}

// Name returns the name of this backend: "lsp"
func (b *Backend) Name() string { return "lsp" }

// Stop shuts down all server processes and closes the diagnostics
// channel. It is safe to call more than once.
func (b *Backend) Stop() {
	b.mu.Lock()
	if b.stopped {
		b.mu.Unlock()
		return
	}
	b.stopped = true
	servers := make([]*Server, 0, len(b.servers))
	for _, srv := range b.servers {
		servers = append(servers, srv)
	}
	b.servers = make(map[string]*Server)
	b.docs = make(map[string]*docRef)
	close(b.diags)
	b.mu.Unlock()
	for _, srv := range servers {
		srv.Shutdown()
	}
}

// Diagnostics returns the channel receiving diagnostics for open files.
func (b *Backend) Diagnostics() <-chan lang.FileDiags { return b.diags }

// serverKey is the caching key for a server config at a workspace root.
func serverKey(cfg *ServerConfig, root string) string {
	return cfg.ID + "\x00" + root
}

// serverFor returns the [Server] to use for the given file,
// starting it asynchronously if it is not already running.
// It returns nil if there is no known server for the file's language.
func (b *Backend) serverFor(ln *lines.Lines) (*Server, *docRef, bool) {
	if ln == nil {
		return nil, nil, false
	}
	fname := ln.Filename()
	if fname == "" {
		return nil, nil, false
	}
	kn := ln.FileInfo().Known
	cfg, has := b.Reg.Server(kn)
	if !has {
		return nil, nil, false
	}
	root := ""
	if b.RootFunc != nil {
		root = b.RootFunc(fname)
	}
	if root == "" {
		root = cfg.RootDir(fname)
	}
	key := serverKey(cfg, root)

	b.mu.Lock()
	if b.stopped {
		b.mu.Unlock()
		return nil, nil, false
	}
	srv, has := b.servers[key]
	if !has {
		srv = NewServer(b.Reg, cfg, root)
		srv.Diags = b.handleDiags
		b.servers[key] = srv
		go func() {
			if err := srv.Start(context.Background()); err != nil {
				log.Println(err)
				b.mu.Lock()
				delete(b.servers, key)
				b.mu.Unlock()
				return
			}
			b.resync(srv)
		}()
	}
	ref := b.docs[normalizePath(fname)]
	if ref == nil || ref.srv != srv {
		ref = &docRef{ln: ln, srv: srv, langID: LanguageID(kn, fname), fname: fname}
		b.docs[normalizePath(fname)] = ref
	}
	b.mu.Unlock()
	return srv, ref, true
}

// resync re-opens all documents tracked for a server that has
// just finished starting up, so they get their didOpen notifications.
func (b *Backend) resync(srv *Server) {
	b.mu.Lock()
	refs := make([]*docRef, 0, 4)
	for _, ref := range b.docs {
		if ref.srv == srv {
			refs = append(refs, ref)
		}
	}
	b.mu.Unlock()
	for _, ref := range refs {
		srv.Open(DocumentURIfromFilename(ref.ln.Filename()), ref.langID, currentText(ref.ln))
	}
}

// Supported returns whether there is a language server available
// for the language of the given buffer.
func (b *Backend) Supported(ln *lines.Lines) bool {
	if ln == nil {
		return false
	}
	kn := ln.FileInfo().Known
	if LanguageID(kn, ln.Filename()) == "" {
		return false
	}
	_, has := b.Reg.Server(kn)
	return has
}

// Open notifies the server that the file is open, sending its current
// contents, starting the server asynchronously if needed.
func (b *Backend) Open(ln *lines.Lines) {
	srv, ref, has := b.serverFor(ln)
	if !has {
		return
	}
	srv.Open(DocumentURIfromFilename(ln.Filename()), ref.langID, currentText(ln))
}

// Change syncs the current contents of the buffer to the server,
// opening the document first if needed (e.g., the file was renamed).
func (b *Backend) Change(ln *lines.Lines) {
	srv, ref, has := b.serverFor(ln)
	if !has {
		return
	}
	uri := DocumentURIfromFilename(ln.Filename())
	if !srv.isOpen(uri) {
		srv.Open(uri, ref.langID, currentText(ln))
		return
	}
	srv.DidChange(uri, currentText(ln))
}

// Save notifies the server that the file has been saved.
func (b *Backend) Save(ln *lines.Lines) {
	srv, ref, has := b.serverFor(ln)
	if !has {
		return
	}
	uri := DocumentURIfromFilename(ln.Filename())
	_ = ref
	// make sure latest text is synced before the save notification:
	srv.DidChange(uri, currentText(ln))
	srv.DidSave(uri)
}

// Close notifies the server that the file has been closed,
// shutting down the server if it is the last file using it.
func (b *Backend) Close(ln *lines.Lines) {
	if ln == nil {
		return
	}
	fname := ln.Filename()
	b.mu.Lock()
	ref := b.docs[normalizePath(fname)]
	delete(b.docs, normalizePath(fname))
	b.mu.Unlock()
	if ref == nil {
		return
	}
	ref.srv.Close(DocumentURIfromFilename(fname))
	b.maybeShutdown(ref.srv)
}

// maybeShutdown stops the given server if no open docs are still using it.
func (b *Backend) maybeShutdown(srv *Server) {
	b.mu.Lock()
	stillUsed := false
	for _, ref := range b.docs {
		if ref.srv == srv {
			stillUsed = true
			break
		}
	}
	// also keep if there are docs still referencing this root+cfg via
	// other key mappings; simple approach: check docs only.
	if !stillUsed {
		for k, s := range b.servers {
			if s == srv {
				delete(b.servers, k)
				break
			}
		}
	}
	b.mu.Unlock()
	if !stillUsed {
		srv.Shutdown()
	}
}

// isOpen reports whether the server is currently tracking an open
// document for the given URI.
func (s *Server) isOpen(uri DocumentURI) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, has := s.docs[uri]
	return has
}

// currentText returns the full text of the buffer as a string.
// Note: all exported [lines.Lines] methods lock internally, and the
// Lines mutex is not reentrant, so we must not lock around them here.
func currentText(ln *lines.Lines) string {
	return string(ln.Text())
}

// DocumentURIfromFilename returns the file URI for a filename.
func DocumentURIfromFilename(fname string) DocumentURI {
	return URIFromPath(fname)
}

////////   Diagnostics

// handleDiags converts published diagnostics for a file into the
// backend-neutral [lang.FileDiags] and sends them to the channel.
// Called from the server read goroutine.
func (b *Backend) handleDiags(uri DocumentURI, diags []Diagnostic) {
	fname := uri.Path()
	b.mu.Lock()
	ref := b.docs[normalizePath(fname)]
	b.mu.Unlock()
	if ref == nil {
		return // only report for currently-tracked files
	}
	fd := lang.FileDiags{Filename: ref.fname}
	for _, d := range diags {
		fd.Diags = append(fd.Diags, lang.Diagnostic{
			Region:   regionFromLSP(ref.ln, d.Range),
			Msg:      cleanMessage(d.Message),
			Severity: diagSeverity(d.Severity),
			Source:   d.Source,
			Code:     codeString(d.Code),
		})
	}
	select {
	case b.diags <- fd:
	case <-time.After(time.Second):
		if b.Verbose {
			log.Printf("lsp: dropping diagnostics for %s: channel full\n", fname)
		}
	}
}

// cleanMessage collapses multi-line diagnostic messages onto one line,
// which works better for inline display in the editor.
func cleanMessage(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// diagSeverity converts LSP DiagnosticSeverity values (1..4) into the
// backend-neutral lang values (0..3). These enums are NOT numerically
// identical; do not cast directly.
func diagSeverity(s DiagnosticSeverity) lang.DiagSeverity {
	switch s {
	case SeverityError:
		return lang.DiagErr
	case SeverityWarning:
		return lang.DiagWarn
	case SeverityInformation:
		return lang.DiagInfo
	case SeverityHint:
		return lang.DiagHint
	}
	return lang.DiagErr
}

// normalizePath returns a canonical key for path-insensitive matching of
// server-reported file paths against buffer filenames, resolving symlinks
// (e.g. /var -> /private/var on macOS) so that URIs echoed back by the
// server match what the editor registered.
func normalizePath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = p
	}
	if r, err := filepath.EvalSymlinks(abs); err == nil {
		abs = r
	}
	return filepath.Clean(abs)
}

func codeString(code any) string {
	switch c := code.(type) {
	case nil:
		return ""
	case string:
		return c
	default:
		return ""
	}
}

// regionFromLSP converts an LSP range into a [textpos.Region] in the
// editor's rune units, using the given buffer for UTF-16 conversions.
func regionFromLSP(ln *lines.Lines, r Range) textpos.Region {
	stLn, stCh := posFromLSP(ln, r.Start)
	enLn, enCh := posFromLSP(ln, r.End)
	if enLn < stLn || (enLn == stLn && enCh < stCh) {
		enLn, enCh = stLn, stCh
	}
	return textpos.Region{
		Start: textpos.Pos{Line: stLn, Char: stCh},
		End:   textpos.Pos{Line: enLn, Char: enCh},
	}
}

// posFromLSP converts an LSP position to editor line, rune units,
// converting the character offset from UTF-16 code units.
// Lines methods lock internally; do not wrap them in Lock here.
func posFromLSP(ln *lines.Lines, p Position) (int, int) {
	nln := ln.NumLines()
	line := p.Line
	if line >= nln {
		line = max(0, nln-1)
	}
	if line < 0 {
		line = 0
	}
	txt := string(ln.Line(line))
	ch := lang.UTF16ToRune(txt, p.Character)
	if ch > len([]rune(txt)) {
		ch = len([]rune(txt))
	}
	return line, ch
}

// toLSPPos converts an editor line, rune position into an LSP position,
// converting character to UTF-16 code units.
func toLSPPos(ln *lines.Lines, pos textpos.Pos) Position {
	if !ln.IsValidLine(pos.Line) {
		return Position{Line: max(0, pos.Line)}
	}
	txt := string(ln.Line(pos.Line))
	return Position{Line: pos.Line, Character: lang.RuneToUTF16(txt, pos.Char)}
}

////////   Completion

// Completions returns completions for the given text at position,
// querying the language server with a short timeout.
func (b *Backend) Completions(ln *lines.Lines, text string, posLine, posChar int) complete.Matches {
	srv, _, has := b.serverFor(ln)
	if !has || !srv.Ready() {
		return complete.Matches{}
	}
	pos := toLSPPos(ln, textpos.Pos{Line: posLine, Char: posChar})
	params := &CompletionParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: DocumentURIfromFilename(ln.Filename())},
			Position:     pos,
		},
	}
	params.Context = completionContext(text)
	list, err := srv.Completion(params)
	if err != nil || list == nil {
		return complete.Matches{}
	}
	seed := complete.SeedSpace(text)
	ms := complete.Matches{Seed: seed}
	b.mu.Lock()
	b.edits = make(map[string]*complete.Edit)
	b.mu.Unlock()
	for _, item := range list.Items {
		cp := b.convertCompletion(ln, posLine, posChar, item)
		ms.Matches = append(ms.Matches, cp)
	}
	return ms
}

// completionContext determines the completion trigger kind from the text.
func completionContext(text string) *CompletionContext {
	tk := CompleteInvoked
	var ch string
	if len(text) > 0 {
		last := text[len(text)-1]
		if last == '.' || last == ':' || last == '/' {
			tk = CompleteTriggerCharacter
			ch = string(last)
		}
	}
	return &CompletionContext{TriggerKind: tk, TriggerCharacter: ch}
}

// convertCompletion converts one LSP CompletionItem into a
// [complete.Completion], recording the edit needed to apply it.
// posLine, posChar give the cursor position in editor units.
func (b *Backend) convertCompletion(ln *lines.Lines, posLine, posChar int, item CompletionItem) complete.Completion {
	newText := item.InsertText
	if newText == "" {
		newText = item.Label
	}
	adj := 0
	if item.InsertTextFormat == InsertTextSnippet {
		newText, adj = plainSnippet(item.InsertText)
	}
	if item.TextEdit != nil && item.TextEdit.NewText != "" {
		nt := item.TextEdit.NewText
		if item.InsertTextFormat == InsertTextSnippet {
			nt, _ = plainSnippet(nt)
		}
		newText = nt
	}
	cp := complete.Completion{
		Text:  newText,
		Label: item.Label,
		Icon:  completionIcon(item.Kind).Icon(),
		Desc:  completionDesc(item),
	}
	fd := forwardDelete(lineText(ln, posLine), posChar, newText)
	if item.TextEdit != nil {
		r := regionFromLSP(ln, item.TextEdit.Range)
		if r.Start.Line == posLine {
			d := r.End.Char - posChar
			if d > 0 {
				fd = d
			}
		}
	}
	b.mu.Lock()
	b.edits[cp.Text] = &complete.Edit{NewText: newText, ForwardDelete: fd, CursorAdjust: adj}
	b.mu.Unlock()
	return cp
}

// lineText returns the text of the given line as a string.
func lineText(ln *lines.Lines, line int) string {
	if !ln.IsValidLine(line) {
		return ""
	}
	return string(ln.Line(line))
}

// forwardDelete computes the number of runes past the cursor that the
// given completion text should overwrite (e.g., an existing open paren),
// following the same logic as the golang CompleteEdit.
func forwardDelete(line string, cursorPos int, newText string) int {
	if cursorPos >= len(line) {
		return 0
	}
	s2 := line[cursorPos:]
	for i, c := range s2 {
		if c == '(' {
			return i
		}
		isAlnum := c == '_' || isLetterOrDigitRune(c)
		if !isAlnum {
			return i
		}
	}
	return len(s2)
}

func isLetterOrDigitRune(c rune) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// plainSnippet strips snippet placeholders from an insertText, returning
// the plain text to insert and the cursor adjustment relative to the end
// (negative), using the position of $0 or the first placeholder, which is
// where the cursor should be placed after insertion.
func plainSnippet(snip string) (text string, cursorAdjust int) {
	var sb strings.Builder
	cursorAt := -1
	i := 0
	lowest := 0x7FFFFFFF
	for i < len(snip) {
		c := snip[i]
		if c != '$' {
			sb.WriteByte(c)
			i++
			continue
		}
		// escaped dollar
		if i+1 < len(snip) && snip[i+1] == '$' {
			sb.WriteByte('$')
			i += 2
			continue
		}
		i++ // consume $
		if i < len(snip) && snip[i] == '{' {
			j := i + 1
			numStr := ""
			for j < len(snip) && snip[j] >= '0' && snip[j] <= '9' {
				numStr += string(snip[j])
				j++
			}
			n, _ := atoi(numStr)
			ph := ""
			if j < len(snip) && snip[j] == ':' {
				k := j + 1
				depth := 1
				start := k
				for k < len(snip) && depth > 0 {
					if snip[k] == '{' {
						depth++
					} else if snip[k] == '}' {
						depth--
					}
					if depth > 0 {
						k++
					}
				}
				ph = snip[start:k]
				j = k
			}
			if j < len(snip) && snip[j] == '}' {
				j++
			}
			i = j
			sb.WriteString(ph)
			if n == 0 {
				cursorAt = sb.Len()
			} else if n < lowest {
				lowest = n
				if cursorAt < 0 {
					cursorAt = sb.Len()
				}
			}
			continue
		}
		// simple $N form
		j := i
		for j < len(snip) && snip[j] >= '0' && snip[j] <= '9' {
			j++
		}
		if j > i {
			n, _ := atoi(snip[i:j])
			if n == 0 {
				cursorAt = sb.Len()
			}
			i = j
			continue
		}
		sb.WriteByte('$')
	}
	if cursorAt < 0 {
		return sb.String(), 0
	}
	return sb.String(), cursorAt - sb.Len()
}

func atoi(s string) (int, error) {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n, nil
}

// CompleteEdit returns the recorded edit for a selected completion,
// falling back to overwriting the current word if the edit was not
// recorded (e.g., after an editor restart between completion and select).
func (b *Backend) CompleteEdit(ln *lines.Lines, text string, cursorPos int, comp complete.Completion, seed string) complete.Edit {
	b.mu.Lock()
	ed, has := b.edits[comp.Text]
	b.mu.Unlock()
	if has {
		return *ed
	}
	fd := forwardDelete(text, cursorPos, comp.Text)
	return complete.Edit{NewText: comp.Text, ForwardDelete: fd}
}

// completionIcon returns the icon token for a completion item kind,
// mapping the LSP CompletionItemKind onto token.Tokens. Completion
// kinds and symbol kinds share the same numbering for the overlap,
// so this uses the standard symbol kind map.
func completionIcon(kind CompletionItemKind) token.Tokens {
	return symbolToken(int(kind))
}

// symbolToken maps the LSP SymbolKind numeric value onto token.Tokens.
// Note: we deliberately keep our own map rather than reuse the one in
// the vestigial cogentcore.org/core/text/parse/lsp package, which is
// unused elsewhere and panics at package init (nil reverse-map), and
// would otherwise couple us to specific core versions.
func symbolToken(kind int) token.Tokens {
	if tk, has := symbolKindToken[kind]; has {
		return tk
	}
	return token.None
}

// symbolKindToken maps LSP SymbolKind values onto token.Tokens.
var symbolKindToken = map[int]token.Tokens{
	1:  token.NameOther, // File: no dedicated token for files
	2:  token.NameModule,
	3:  token.NameNamespace,
	4:  token.NamePackage,
	5:  token.NameClass,
	6:  token.NameMethod,
	7:  token.NameProperty,
	8:  token.NameField,
	9:  token.NameConstructor,
	10: token.NameEnum,
	11: token.NameInterface,
	12: token.NameFunction,
	13: token.NameVar,
	14: token.NameConstant,
	15: token.LitStr,
	16: token.LitNum,
	17: token.LiteralBool,
	18: token.NameArray,
	19: token.NameObject,
	20: token.NameTag,
	21: token.None,
	22: token.NameEnumMember,
	23: token.NameStruct,
	24: token.NameEvent,
	25: token.Operator,
	26: token.NameTypeParam,
}

// completionDesc returns a description for a completion item,
// combining detail and documentation.
func completionDesc(item CompletionItem) string {
	desc := item.Detail
	if item.Documentation != nil && item.Documentation.Value != "" {
		if desc != "" {
			desc += "\n\n"
		}
		desc += item.Documentation.Value
	}
	const maxDesc = 500
	if utf8.RuneCountInString(desc) > maxDesc {
		desc = string([]rune(desc)[:maxDesc]) + "…"
	}
	return desc
}

////////   Hover / definition / references

// Hover returns hover documentation for the symbol at the given position.
func (b *Backend) Hover(ln *lines.Lines, pos textpos.Pos) string {
	srv, _, has := b.serverFor(ln)
	if !has || !srv.Ready() {
		return ""
	}
	params := &HoverParams{
		TextDocument: TextDocumentIdentifier{URI: DocumentURIfromFilename(ln.Filename())},
		Position:     toLSPPos(ln, pos),
	}
	txt, ok := srv.Hover(params)
	if !ok {
		return ""
	}
	return txt
}

// Definition returns definition locations for the symbol at the position.
func (b *Backend) Definition(ln *lines.Lines, pos textpos.Pos) []lang.Location {
	srv, _, has := b.serverFor(ln)
	if !has || !srv.Ready() {
		return nil
	}
	params := &DefinitionParams{
		TextDocument: TextDocumentIdentifier{URI: DocumentURIfromFilename(ln.Filename())},
		Position:     toLSPPos(ln, pos),
	}
	locs, err := srv.Definition(params)
	if err != nil {
		return nil
	}
	return b.convertLocations(locs)
}

// References returns reference locations for the symbol at the position.
func (b *Backend) References(ln *lines.Lines, pos textpos.Pos) []lang.Location {
	srv, _, has := b.serverFor(ln)
	if !has || !srv.Ready() {
		return nil
	}
	params := &ReferenceParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: DocumentURIfromFilename(ln.Filename())},
			Position:     toLSPPos(ln, pos),
		},
		Context: ReferenceContext{IncludeDeclaration: true},
	}
	locs, err := srv.References(params)
	if err != nil {
		return nil
	}
	return b.convertLocations(locs)
}

// convertLocations converts LSP locations into [lang.Location],
// converting UTF-16 ranges using the target file's contents, which is
// read from disk if the file is not currently open in the editor.
func (b *Backend) convertLocations(locs []Location) []lang.Location {
	out := make([]lang.Location, 0, len(locs))
	for _, loc := range locs {
		fname := loc.URI.Path()
		if fname == "" {
			continue
		}
		out = append(out, lang.Location{
			Filename: fname,
			Region:   b.regionForFile(fname, loc.Range),
		})
	}
	return out
}

// regionForFile converts an LSP range in the given file into editor units,
// using the open buffer if available, otherwise reading the file from disk.
func (b *Backend) regionForFile(fname string, r Range) textpos.Region {
	b.mu.Lock()
	ref := b.docs[normalizePath(fname)]
	b.mu.Unlock()
	if ref != nil {
		return regionFromLSP(ref.ln, r)
	}
	naive := textpos.Region{
		Start: textpos.Pos{Line: r.Start.Line, Char: r.Start.Character},
		End:   textpos.Pos{Line: r.End.Line, Char: r.End.Character},
	}
	content, err := os.ReadFile(fname)
	if err != nil {
		return naive
	}
	ln2, _ := lines.NewLinesFromBytes(fname, 80, content)
	if ln2 == nil {
		return naive
	}
	return regionFromLSP(ln2, r)
}

////////   Symbols

// Symbols returns the hierarchical document symbols for the file.
func (b *Backend) Symbols(ln *lines.Lines) []lang.Symbol {
	srv, _, has := b.serverFor(ln)
	if !has || !srv.Ready() {
		return nil
	}
	syms, err := srv.DocumentSymbol(DocumentURIfromFilename(ln.Filename()))
	if err != nil {
		return nil
	}
	fname := ln.Filename()
	return convertDocSymbols(ln, syms, fname)
}

// convertDocSymbols flattens the hierarchical LSP document symbols into
// our [lang.Symbol] tree, attaching children to their parents.
func convertDocSymbols(ln *lines.Lines, syms []DocumentSymbol, fname string) []lang.Symbol {
	out := make([]lang.Symbol, 0, len(syms))
	for _, ds := range syms {
		sy := lang.Symbol{
			Name:     ds.Name,
			Detail:   ds.Detail,
			Kind:     symbolToken(int(ds.Kind)),
			Filename: fname,
			Region:   regionFromLSP(ln, ds.SelectionRange),
		}
		if len(ds.Children) > 0 {
			sy.Children = convertDocSymbols(ln, ds.Children, fname)
		}
		out = append(out, sy)
	}
	return out
}

// WorkspaceSymbols searches the workspace for symbols matching query,
// using the server appropriate for the given file's language.
func (b *Backend) WorkspaceSymbols(ln *lines.Lines, query string) []lang.Symbol {
	srv, _, has := b.serverFor(ln)
	if !has || !srv.Ready() {
		return nil
	}
	infos, err := srv.WorkspaceSymbol(query)
	if err != nil {
		return nil
	}
	out := make([]lang.Symbol, 0, len(infos))
	for _, si := range infos {
		fname := si.Location.URI.Path()
		if fname == "" {
			continue
		}
		out = append(out, lang.Symbol{
			Name:     si.Name,
			Detail:   si.ContainerName,
			Kind:     symbolToken(int(si.Kind)),
			Filename: fname,
			Region:   b.regionForFile(fname, si.Location.Range),
		})
	}
	return out
}

////////   Formatting

// Format requests formatting edits from the server and applies them
// to the buffer in reverse order. Returns true if any edits applied.
func (b *Backend) Format(ln *lines.Lines) bool {
	srv, _, has := b.serverFor(ln)
	if !has || !srv.Ready() {
		return false
	}
	uri := DocumentURIfromFilename(ln.Filename())
	srv.DidChange(uri, currentText(ln))
	opts := FormattingOptions{TabSize: 4, InsertSpaces: false}
	edits, err := srv.Formatting(uri, opts)
	if err != nil || len(edits) == 0 {
		return false
	}
	// apply in reverse so earlier positions stay valid:
	sort.SliceStable(edits, func(i, j int) bool {
		a, bi := edits[i].Range.Start, edits[j].Range.Start
		if a.Line != bi.Line {
			return a.Line > bi.Line
		}
		return a.Character > bi.Character
	})
	for _, e := range edits {
		r := regionFromLSP(ln, e.Range)
		ln.DeleteText(r.Start, r.End)
		if e.NewText != "" {
			ln.InsertText(r.Start, []rune(e.NewText))
		}
	}
	return true
}
