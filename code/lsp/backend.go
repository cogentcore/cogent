// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
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
	applies chan []lang.FileEdits
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
		applies: make(chan []lang.FileEdits, 8),
	}
}

// Name returns the name of this backend: "lsp"
func (b *Backend) Name() string { return "lsp" }

// SetOptions sets additional per-server options (server ID -> options)
// on the registry, merged over the built-in defaults. See
// [Registry.SetOptions]: only servers started after this call see them,
// so pair it with [Backend.Restart] to apply to running servers.
func (b *Backend) SetOptions(opts map[string]map[string]any) {
	b.Reg.SetOptions(opts)
}

// Reconfigure prompts all running servers to re-fetch their settings
// from the client (workspace/didChangeConfiguration), picking up any
// options set with [Backend.SetOptions] after the servers started.
// Safe to call from a background goroutine.
func (b *Backend) Reconfigure() {
	b.mu.Lock()
	if b.stopped {
		b.mu.Unlock()
		return
	}
	servers := make([]*Server, 0, len(b.servers))
	for _, srv := range b.servers {
		servers = append(servers, srv)
	}
	refs := make([]*docRef, 0, len(b.docs))
	for _, ref := range b.docs {
		refs = append(refs, ref)
	}
	b.mu.Unlock()
	for _, srv := range servers {
		if !srv.Ready() {
			continue
		}
		if err := srv.DidChangeConfiguration(); err != nil {
			log.Printf("lsp: %s: didChangeConfiguration: %v\n", srv.Cfg.ID, err)
		}
	}
	// gopls applies re-fetched settings when it next rebuilds a snapshot,
	// so nudge each open document with a full-content didChange:
	for _, ref := range refs {
		srv := ref.srv
		if !srv.Ready() {
			continue
		}
		srv.DidChange(DocumentURIfromFilename(ref.fname), currentText(ref.ln))
	}
}

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
	close(b.applies)
	b.mu.Unlock()
	for _, srv := range servers {
		srv.Shutdown()
	}
}

// Diagnostics returns the channel receiving diagnostics for open files.
func (b *Backend) Diagnostics() <-chan lang.FileDiags { return b.diags }

// AppliedEdits returns the channel receiving per-file notifications
// for server-initiated workspace edits (workspace/applyEdit) that have
// already been applied by the backend, so that the editor layer can
// refresh the affected views.
func (b *Backend) AppliedEdits() <-chan []lang.FileEdits { return b.applies }

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
		srv.ApplyEdit = b.handleApplyEdit
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

// Ready returns whether the language server for the given buffer
// exists and has completed its startup handshake, so that requests
// such as Hover and Definition can be answered.
func (b *Backend) Ready(ln *lines.Lines) bool {
	srv, _, has := b.serverFor(ln)
	return has && srv != nil && srv.Ready()
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
	// the send races with Stop closing the channel (servers can
	// publish at any time until they are shut down), so re-check
	// stopped under the lock around the send:
	b.mu.Lock()
	stopped := b.stopped
	if !stopped {
		select {
		case b.diags <- fd:
			b.mu.Unlock()
			return
		case <-time.After(time.Second):
		}
	}
	b.mu.Unlock()
	if !stopped && b.Verbose {
		log.Printf("lsp: dropping diagnostics for %s: channel full\n", fname)
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

// isIdentRune reports whether the rune can appear in an identifier.
func isIdentRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// completionSeed returns the seed for completions from the given line
// prefix: the trailing identifier PLUS any qualifying selector prefixes
// ("core.F" -> "core.F", not "F" as a plain whitespace seed would give),
// so that selecting a completion rewrites the whole qualified reference.
// It is rune-based: the text may be a byte prefix of the line ending
// mid-rune, and byte-slicing an incomplete rune would corrupt it.
func completionSeed(text string) string {
	rn := []rune(text)
	i := len(rn)
	for i > 0 && isIdentRune(rn[i-1]) {
		i--
	}
	for i > 0 && rn[i-1] == '.' {
		i--
		for i > 0 && isIdentRune(rn[i-1]) {
			i--
		}
	}
	return string(rn[i:])
}

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
	seed := completionSeed(text)
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

// signatureTriggerChars are the characters that trigger signature help
// in most language servers; gopls uses "(" and ",".
var signatureTriggerChars = []rune{'(', ',', '<'}

// IsSignatureTrigger reports whether the last character of the given
// text is one that should trigger a signature help request.
func IsSignatureTrigger(text string) bool {
	rn := []rune(text)
	if len(rn) == 0 {
		return false
	}
	return slices.Contains(signatureTriggerChars, rn[len(rn)-1])
}

// signatureContext determines the signature help trigger kind from the
// text and whether signature help was already showing (retrigger).
func signatureContext(text string, wasShowing bool) *SignatureHelpContext {
	ctx := &SignatureHelpContext{IsRetrigger: wasShowing}
	rn := []rune(text)
	if len(rn) > 0 && slices.Contains(signatureTriggerChars, rn[len(rn)-1]) {
		ctx.TriggerKind = SigHelpTriggerCharacterKind
		ctx.TriggerCharacter = string(rn[len(rn)-1])
	} else if wasShowing {
		ctx.TriggerKind = SigHelpContentChange
	} else {
		ctx.TriggerKind = SigHelpInvoked
	}
	return ctx
}

// activeParameter computes the index of the parameter the cursor is at,
// by counting top-level commas since the innermost unmatched open paren
// before the cursor. Returns -1 if the cursor is not inside a call.
// This is the standard client-side computation; some servers also
// provide ActiveParameter, which takes precedence when set.
func activeParameter(text string, charPos int) int {
	rn := []rune(text)
	if charPos > len(rn) {
		charPos = len(rn)
	}
	depth := 0
	paren := -1
	for i := charPos - 1; i >= 0; i-- {
		switch rn[i] {
		case ')', ']', '}':
			depth++
		case '(', '[', '{':
			if depth > 0 {
				depth--
			} else if rn[i] == '(' {
				paren = i
			} else {
				return -1 // crossed a non-paren bracket: not in a call
			}
		}
		if paren >= 0 {
			break
		}
	}
	if paren < 0 {
		return -1
	}
	idx := 0
	depth = 0
	for i := paren + 1; i < charPos; i++ {
		switch rn[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				idx++
			}
		case '"', '\'', '`': // skip string literal
			q := rn[i]
			i++
			for i < charPos && rn[i] != q {
				if rn[i] == '\\' {
					i++
				}
				i++
			}
		}
	}
	return idx
}

// SignatureHelp returns the signature help for the given text at the
// given editor position, or nil if the server has nothing to show.
// wasReTrigger indicates that signature help was already being shown,
// so that the server can do incremental updates.
func (b *Backend) SignatureHelp(ln *lines.Lines, pos textpos.Pos, wasReTrigger bool) (*SignatureHelp, int, error) {
	srv, _, has := b.serverFor(ln)
	if !has || !srv.Ready() {
		return nil, -1, nil
	}
	text := lineText(ln, pos.Line)
	params := &SignatureHelpParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: DocumentURIfromFilename(ln.Filename())},
			Position:     toLSPPos(ln, pos),
		},
	}
	params.Context = signatureContext(text[:min(pos.Char, len([]rune(text)))], wasReTrigger)
	sh, err := srv.SignatureHelp(params)
	if err != nil || sh == nil {
		return nil, -1, err
	}
	act := -1
	if sh.ActiveSignature != nil && *sh.ActiveSignature >= 0 && *sh.ActiveSignature < len(sh.Signatures) {
		act = *sh.ActiveSignature
	}
	if act < 0 && len(sh.Signatures) > 0 {
		act = 0
	}
	if act >= 0 {
		sig := &sh.Signatures[act]
		ap := activeParameter(text, pos.Char)
		if ap < 0 {
			ap = 0
		}
		if sh.ActiveParameter != nil && *sh.ActiveParameter >= 0 {
			ap = *sh.ActiveParameter
		}
		if ap >= len(sig.Parameters) {
			ap = len(sig.Parameters) - 1
		}
		sh.ActiveParameter = &ap
	}
	return sh, act, nil
}

// SignatureText formats the given (already active-signature-selected)
// signature help into a single-line display string, with the active
// parameter marked using the given prefix, suffix markers.
func SignatureText(sh *SignatureHelp, actSig, activeParam int, mark, unmark string) string {
	if sh == nil || actSig < 0 || actSig >= len(sh.Signatures) {
		return ""
	}
	sig := &sh.Signatures[actSig]
	if len(sig.Parameters) == 0 || activeParam < 0 || activeParam >= len(sig.Parameters) {
		return sig.Label
	}
	var sb strings.Builder
	prev := 0
	for i, p := range sig.Parameters {
		txt := p.LabelText(sig.Label)
		off := strings.Index(sig.Label[prev:], txt)
		if off < 0 {
			return sig.Label // label form doesn't match; show it plain
		}
		st, ed := prev+off, prev+off+len(txt)
		sb.WriteString(sig.Label[prev:st])
		if i == activeParam {
			sb.WriteString(mark)
		}
		sb.WriteString(txt)
		if i == activeParam {
			sb.WriteString(unmark)
		}
		prev = ed
	}
	sb.WriteString(sig.Label[prev:])
	return sb.String()
}

// InlayHints returns the inlay hints the server provides for the
// current contents of the buffer within the given region.
// Note: the editor does not render these; this is for programmatic use.
func (b *Backend) InlayHints(ln *lines.Lines, reg textpos.Region) ([]InlayHint, error) {
	srv, _, has := b.serverFor(ln)
	if !has || !srv.Ready() {
		return nil, nil
	}
	params := &InlayHintParams{
		TextDocument: TextDocumentIdentifier{URI: DocumentURIfromFilename(ln.Filename())},
		Range:        toLSPRange(ln, reg),
	}
	return srv.InlayHint(params)
}

// SemanticToken is a decoded semantic token in editor units:
// a rune-based line/char region with the legend type name and a
// bit set of legend modifier names.
type SemanticToken struct {
	Start      textpos.Pos
	End        textpos.Pos
	Type       string
	Modifiers  []string
	Deprecated bool
}

// SemanticTokensFull returns the decoded semantic tokens for the given
// buffer, in editor (rune) units. Returns nil if the server does not
// support semantic tokens or the request fails.
func (b *Backend) SemanticTokensFull(ln *lines.Lines) ([]SemanticToken, error) {
	srv, _, has := b.serverFor(ln)
	if !has || !srv.Ready() {
		return nil, nil
	}
	legend := srv.SemanticTokensLegend()
	if legend == nil {
		return nil, nil
	}
	st, err := srv.SemanticTokensFull(DocumentURIfromFilename(ln.Filename()))
	if err != nil || st == nil {
		return nil, err
	}
	return decodeSemanticTokens(ln, legend, st)
}

// decodeSemanticTokens converts the relative-encoded token data into
// absolute editor-unit tokens. Data is five ints per token:
// deltaLine, deltaStartUTF16, length, type, modifiers, where the first
// four are relative to the previous token's start.
func decodeSemanticTokens(ln *lines.Lines, legend *SemanticTokensLegend, st *SemanticTokens) ([]SemanticToken, error) {
	n := len(st.Data) / 5
	if len(st.Data)%5 != 0 {
		return nil, fmt.Errorf("lsp: semanticTokens data length %d not a multiple of 5", len(st.Data))
	}
	out := make([]SemanticToken, 0, n)
	line, char16 := 0, 0
	for i := 0; i < n; i++ {
		d := st.Data[i*5 : i*5+5]
		line += d[0]
		if d[0] == 0 {
			char16 += d[1]
		} else {
			char16 = d[1]
		}
		if !ln.IsValidLine(line) {
			continue
		}
		txt := string(ln.Line(line))
		stChar := lang.UTF16ToRune(txt, char16)
		edChar := lang.UTF16ToRune(txt, char16+d[2])
		typ := ""
		if d[3] >= 0 && d[3] < len(legend.TokenTypes) {
			typ = legend.TokenTypes[d[3]]
		}
		var mods []string
		modsBit := d[4]
		for mi := 0; mi < len(legend.TokenModifiers) && modsBit > 0; mi++ {
			if modsBit&1 != 0 {
				mods = append(mods, legend.TokenModifiers[mi])
			}
			modsBit >>= 1
		}
		out = append(out, SemanticToken{
			Start:      textpos.Pos{Line: line, Char: stChar},
			End:        textpos.Pos{Line: line, Char: edChar},
			Type:       typ,
			Modifiers:  mods,
			Deprecated: slices.Contains(mods, "deprecated"),
		})
	}
	return out, nil
}

// completionContext determines the completion trigger kind from the text.
func completionContext(text string) *CompletionContext {
	seed := completionSeed(text)
	rn := []rune(seed)
	if len(rn) > 0 && rn[len(rn)-1] == '.' {
		// just typed a selector: members of the qualifier
		return &CompletionContext{TriggerKind: CompleteTriggerCharacter, TriggerCharacter: "."}
	}
	if len(rn) > 0 {
		// typing an identifier: gopls only filters its full candidate
		// list when the request is a content-change trigger; Invoked
		// returns everything unfiltered.
		return &CompletionContext{TriggerKind: CompleteContentChange}
	}
	return &CompletionContext{TriggerKind: CompleteInvoked}
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
	// The seed the user has typed (possibly qualified, e.g. "core.F")
	// must be accounted for: the editor replaces the seed back from the
	// cursor with the edit text. Servers disagree about how much of the
	// seed their text includes: if the item's TextEdit range starts
	// before the cursor, the difference is how many seed runes it
	// already contains, so drop exactly that many from the text and keep
	// the rest of the prefix -- which the editor's seed replacement
	// restores ("core.F" + "Frame" -> "core.Frame", and "strings.S" +
	// "strings.Split" or "Split" -> "strings.Split"). Without a usable
	// TextEdit range, the text is taken as-is relative to the cursor.
	// Note: seed here is the line prefix up to the cursor, not the full
	// line, which may continue past it.
	lt := lineText(ln, posLine)
	rn := []rune(lt)
	seed := completionSeed(string(rn[:min(posChar, len(rn))]))
	repl := len([]rune(seed)) // how many seed runes the server's text replaces
	if item.TextEdit != nil {
		r := regionFromLSP(ln, item.TextEdit.Range)
		if r.Start.Line == posLine && r.Start.Char <= posChar {
			repl = posChar - r.Start.Char
		}
	}
	if repl > 0 && repl < len([]rune(seed)) {
		sr := []rune(seed)
		newText = string(sr[:len(sr)-repl]) + newText
	} else if repl >= len([]rune(seed)) && strings.HasPrefix(newText, seed) {
		newText = newText[len(seed):]
	}
	cp := complete.Completion{
		Text:  newText,
		Label: item.Label,
		Icon:  completionIcon(item.Kind).Icon(),
		Desc:  completionDesc(item),
	}
	// forward delete is relative to what the server's edit range ends
	// at (the cursor for gopls member completions), not to the end of
	// our full-seed replacement span:
	fdSrc := newText
	if item.TextEdit != nil {
		r := regionFromLSP(ln, item.TextEdit.Range)
		if r.Start.Line == posLine && r.End.Char >= posChar {
			rnr := []rune(lt)
			fdSrc = string(rnr[min(posChar, len(rnr)):min(r.End.Char, len(rnr))]) + newText
		}
	}
	fd := forwardDelete(lt, posChar, fdSrc)
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
// falling back to the same seed-normalization the conversion used if
// the edit was not recorded (e.g., after an editor restart between
// completion and select).
func (b *Backend) CompleteEdit(ln *lines.Lines, text string, cursorPos int, comp complete.Completion, seed string) complete.Edit {
	b.mu.Lock()
	ed, has := b.edits[comp.Text]
	b.mu.Unlock()
	if has {
		return *ed
	}
	newText := comp.Text
	// same normalization as convertCompletion, from the seed alone
	// (no server item available here): the editor replaces the seed,
	// so text that already starts with the whole seed keeps only its
	// selector prefix portion.
	if strings.HasPrefix(newText, seed) && strings.Contains(seed, ".") {
		dot := strings.LastIndexByte(seed, '.')
		newText = seed[:dot+1] + newText[len(seed):]
	}
	fd := forwardDelete(text, cursorPos, newText)
	return complete.Edit{NewText: newText, ForwardDelete: fd}
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
	return applyTextEdits(ln, edits)
}

// applyTextEdits applies the given LSP edits to the buffer, in reverse
// position order so earlier positions stay valid. It only uses
// self-locking Lines methods, so it is safe from any goroutine
// (not under ln.Lock). It returns whether any edits were applied.
func applyTextEdits(ln *lines.Lines, edits []TextEdit) bool {
	if len(edits) == 0 {
		return false
	}
	sort.SliceStable(edits, func(i, j int) bool {
		a, bi := edits[i].Range.Start, edits[j].Range.Start
		if a.Line != bi.Line {
			return a.Line > bi.Line
		}
		return a.Character > bi.Character
	})
	did := false
	for _, e := range edits {
		r := regionFromLSP(ln, e.Range)
		if !ln.IsValidPos(r.Start) || !ln.IsValidPos(r.End) {
			log.Printf("lsp: skipping edit at invalid position %v in %s\n", r.Start, ln.Filename())
			continue
		}
		d := ln.DeleteText(r.Start, r.End)
		var ins *textpos.Edit
		if e.NewText != "" {
			ins = ln.InsertText(r.Start, []rune(e.NewText))
		}
		if d != nil || ins != nil {
			did = true
		}
	}
	return did
}

////////   Workspace edits and rename

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
	// guard the send against Stop closing the channel concurrently:
	b.mu.Lock()
	if !b.stopped {
		select {
		case b.applies <- out:
		default:
			log.Printf("lsp: applyEdit refresh notification dropped: buffer full\n")
		}
	}
	b.mu.Unlock()
	return true, ""
}

// pendingEdits is the normalized per-file group of LSP edits,
// merged from the changes map and documentChanges shapes of a
// WorkspaceEdit.
type pendingEdits struct {
	uri   DocumentURI
	edits []TextEdit
}

// normalizeWE flattens a WorkspaceEdit into per-file edit groups,
// merging entries for the same document across both the changes map
// and the documentChanges array.
func normalizeWE(we *WorkspaceEdit) []pendingEdits {
	var ps []pendingEdits
	seen := map[DocumentURI]int{}
	add := func(uri DocumentURI, edits ...TextEdit) {
		if i, has := seen[uri]; has {
			ps[i].edits = append(ps[i].edits, edits...)
			return
		}
		seen[uri] = len(ps)
		ps = append(ps, pendingEdits{uri: uri, edits: edits})
	}
	for uri, es := range we.Changes {
		add(DocumentURI(uri), es...)
	}
	for _, de := range we.DocumentChanges {
		if de.Edit != nil {
			add(de.TextDocument.URI, *de.Edit)
		}
		add(de.TextDocument.URI, de.Edits...)
	}
	return ps
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
	for _, p := range normalizeWE(we) {
		if len(p.edits) == 0 {
			continue
		}
		fname := p.uri.Path()
		if fname == "" {
			continue
		}
		norm := normalizePath(fname)
		b.mu.Lock()
		ref := b.docs[norm]
		b.mu.Unlock()
		open := ref != nil
		var ln *lines.Lines
		if open {
			ln = ref.ln
		} else {
			content, err := os.ReadFile(fname)
			if err != nil {
				log.Printf("lsp: preview: %v\n", err)
				continue
			}
			ln, _ = lines.NewLinesFromBytes(fname, 80, content)
			if ln == nil {
				log.Printf("lsp: preview: cannot parse %s\n", fname)
				continue
			}
		}
		for _, e := range p.edits {
			r := regionFromLSP(ln, e.Range)
			if !ln.IsValidPos(r.Start) || !ln.IsValidPos(r.End) {
				log.Printf("lsp: preview: skipping edit at invalid position %v in %s\n", r.Start, fname)
				continue
			}
			out = append(out, &EditPreview{
				Filename: norm,
				URI:      p.uri,
				Open:     open,
				Region:   r,
				Old:      string(ln.Region(r.Start, r.End).ToBytes()),
				New:      e.NewText,
				LineText: string(ln.Line(r.Start.Line)),
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Filename != out[j].Filename {
			return out[i].Filename < out[j].Filename
		}
		a, b := out[i].Region.Start, out[j].Region.Start
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Char < b.Char
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
	ps := normalizeWE(we)
	var out []lang.FileEdits
	for _, p := range ps {
		if len(p.edits) == 0 {
			continue
		}
		fname := p.uri.Path()
		if fname == "" {
			continue
		}
		b.mu.Lock()
		ref := b.docs[normalizePath(fname)]
		b.mu.Unlock()
		if ref != nil {
			edits := convertEdits(ref.ln, p.edits)
			if applyTextEdits(ref.ln, p.edits) {
				ref.srv.DidChange(p.uri, currentText(ref.ln))
				log.Printf("lsp: applyEdit: %s: %d edits applied to open buffer\n", fname, len(edits))
				out = append(out, lang.FileEdits{Filename: normalizePath(fname), Edits: edits})
			}
			continue
		}
		content, err := os.ReadFile(fname)
		if err != nil {
			log.Printf("lsp: applyEdit: %v\n", err)
			continue
		}
		ln2, _ := lines.NewLinesFromBytes(fname, 80, content)
		if ln2 == nil {
			log.Printf("lsp: applyEdit: cannot parse %s\n", fname)
			continue
		}
		edits := convertEdits(ln2, p.edits)
		if !applyTextEdits(ln2, p.edits) {
			continue
		}
		newTxt := ln2.String()
		if err := os.WriteFile(fname, []byte(newTxt), 0644); err != nil {
			log.Printf("lsp: applyEdit: %v\n", err)
			continue
		}
		log.Printf("lsp: applyEdit: %s: %d edits written (%d->%d bytes)\n", fname, len(edits), len(content), len(newTxt))
		out = append(out, lang.FileEdits{Filename: normalizePath(fname), Edits: edits})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Filename < out[j].Filename })
	return out
}

// convertEdits converts LSP edits to editor units for the buffer.
func convertEdits(ln *lines.Lines, edits []TextEdit) []lang.TextEdit {
	out := make([]lang.TextEdit, 0, len(edits))
	for _, e := range edits {
		out = append(out, lang.TextEdit{Region: regionFromLSP(ln, e.Range), NewText: e.NewText})
	}
	return out
}

// RenameEdits requests the workspace edit for renaming the symbol at
// pos to newName, validating with prepareRename first if the server
// supports it. It returns nil if there is nothing to rename.
func (b *Backend) RenameEdits(ln *lines.Lines, pos textpos.Pos, newName string) (*WorkspaceEdit, error) {
	srv, _, has := b.serverFor(ln)
	if !has || !srv.Ready() {
		return nil, fmt.Errorf("language server not ready for %s", filepath.Base(ln.Filename()))
	}
	uri := DocumentURIfromFilename(ln.Filename())
	srv.DidChange(uri, currentText(ln))
	lpp := &TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     toLSPPos(ln, pos),
	}
	if srv.supportsPrepareRename() {
		_, _, ok, err := srv.PrepareRename(lpp)
		if err != nil {
			return nil, err
		}
		if !ok {
			// prepareRename is advisory: it returns null both for
			// genuinely non-renameable positions and for recoverable
			// conditions such as package errors or symbols that are
			// not yet analyzed. Ask rename itself, which answers with
			// a specific reason on failure, rather than stopping here
			// with a generic message.
			log.Printf("lsp: prepareRename: nothing renameable at %v in %s; consulting rename\n",
				pos, filepath.Base(ln.Filename()))
		}
	}
	we, err := srv.Rename(&RenameParams{TextDocumentPositionParams: *lpp, NewName: newName})
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

// toLSPRange converts an editor (rune) region to an LSP (utf-16) range
// for the given buffer.
func toLSPRange(ln *lines.Lines, reg textpos.Region) Range {
	return Range{Start: toLSPPos(ln, reg.Start), End: toLSPPos(ln, reg.End)}
}

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
	out := make([]Diagnostic, 0, len(diags))
	for _, d := range diags {
		ld := Diagnostic{
			Range:    toLSPRange(ln, d.Region),
			Severity: sevToLSP(d.Severity),
			Code:     d.Code,
			Source:   d.Source,
		}
		// Code can be numeric in the protocol; string is acceptable:
		ld.Message = d.Msg
		out = append(out, ld)
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
	srv, _, has := b.serverFor(ln)
	if !has || !srv.Ready() {
		return nil, fmt.Errorf("language server not ready for %s", filepath.Base(ln.Filename()))
	}
	if !srv.supportsCodeAction() {
		return nil, nil
	}
	uri := DocumentURIfromFilename(ln.Filename())
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
// deferred edits via codeAction/resolve, and running the action's
// command when it returns a WorkspaceEdit. The workspace edit is then
// applied to open buffers and files on disk, as for renames
// (see [Backend.ApplyWorkspaceEdit]). It returns the edits applied.
func (b *Backend) RunCodeAction(ln *lines.Lines, ca *CodeAction) ([]lang.FileEdits, error) {
	srv, _, has := b.serverFor(ln)
	if !has || !srv.Ready() {
		return nil, fmt.Errorf("language server not ready for %s", filepath.Base(ln.Filename()))
	}
	if ca.Edit == nil {
		if err := srv.ResolveCodeAction(ca); err != nil {
			log.Printf("lsp: codeAction resolve %q: %v\n", ca.Title, err)
		}
	}
	if ca.Edit == nil && ca.Command != nil && ca.Command.Command != "" {
		raw, err := srv.ExecuteCommand(ca.Command.Command, ca.Command.Arguments)
		if err != nil {
			return nil, fmt.Errorf("code action %q: %w", ca.Title, err)
		}
		if len(raw) > 0 && string(raw) != "null" {
			var we WorkspaceEdit
			if err := json.Unmarshal(raw, &we); err == nil {
				ca.Edit = &we
			}
		}
	}
	if ca.Edit == nil {
		return nil, fmt.Errorf("code action %q did not provide edits", ca.Title)
	}
	out := b.ApplyWorkspaceEdit(ca.Edit)
	if len(out) == 0 {
		return nil, fmt.Errorf("code action %q produced no edits", ca.Title)
	}
	return out, nil
}
