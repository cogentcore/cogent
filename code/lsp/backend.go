// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"context"
	"log"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

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

	// ShowDocument, if set, shows a document at the request of a server
	// (e.g., gopls "Browse" code actions): an external resource such as
	// a web page to open in a browser if loc is nil, or otherwise the
	// given location in a file. It is called from the server connection
	// goroutine and must not block. It returns whether it was shown.
	ShowDocument func(uri string, loc *lang.Location) bool

	// Progress, if set, is called with a short message describing the
	// most recent operation in progress in the servers (e.g., gopls
	// loading packages), or "" when there are none. It is called from
	// server connection goroutines and must not block.
	Progress func(msg string)

	// Verbose enables detailed logging of the backend.
	Verbose bool

	mu      sync.Mutex
	servers map[string]*Server        // serverKey -> Server
	docs    map[string]*docRef        // normalized filename -> doc tracking
	edits   map[string]*complete.Edit // completion edits, by Completion.Text

	// cmdEdits collects the edits applied by server-initiated
	// workspace/applyEdit requests while inCommand is set, i.e., during
	// the resolve or command requests of a code action
	// (see [Backend.captureEdits]).
	cmdEdits  []lang.FileEdits
	inCommand bool

	// progress are the messages of the operations in progress,
	// in order of their most recent update, for [Backend.Progress].
	progress []progressMsg

	stopped bool

	diags   chan lang.FileDiags
	applies chan []lang.FileEdits

	// done is closed by Stop, releasing any blocked channel sends.
	done chan struct{}

	// sendMu is read-locked around channel sends, and write-locked by
	// Stop to close the channels, so that sends never race the close.
	sendMu sync.RWMutex
	closed bool
}

// docRef associates a [lines.Lines] buffer with its server and language.
type docRef struct {
	ln     *lines.Lines
	srv    *Server
	langID string
	fname  string // raw filename from the buffer, as the editor knows it
}

// restartDelay is the minimum time after a server fails (to start, or
// by exiting unexpectedly) before another attempt is made to start it.
const restartDelay = 10 * time.Second

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
		done:    make(chan struct{}),
	}
}

// SetOptions sets additional per-server options (server ID -> options)
// on the registry, merged over the built-in defaults. See
// [Registry.SetOptions]: servers started after this call get them as
// initialization options; use [Backend.Reconfigure] to apply them to
// running servers.
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
	servers := b.serverList()
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
		if ref.srv.Ready() {
			ref.srv.ForceDidChange(URIFromPath(ref.fname), currentText(ref.ln))
		}
	}
}

// Stop shuts down all server processes and closes the
// [Backend.Diagnostics] and [Backend.AppliedEdits] channels.
// The backend cannot be used after this. It is safe to call more
// than once.
func (b *Backend) Stop() {
	b.mu.Lock()
	if b.stopped {
		b.mu.Unlock()
		return
	}
	b.stopped = true
	servers := b.serverList()
	b.servers = make(map[string]*Server)
	b.docs = make(map[string]*docRef)
	b.mu.Unlock()
	close(b.done)
	b.sendMu.Lock()
	b.closed = true
	close(b.diags)
	close(b.applies)
	b.sendMu.Unlock()
	for _, srv := range servers {
		srv.Shutdown()
	}
}

// serverList returns the current servers. Must be called under mu.
func (b *Backend) serverList() []*Server {
	servers := make([]*Server, 0, len(b.servers))
	for _, srv := range b.servers {
		servers = append(servers, srv)
	}
	return servers
}

// Diagnostics returns the channel receiving diagnostics for open files,
// which is closed by [Backend.Stop].
func (b *Backend) Diagnostics() <-chan lang.FileDiags { return b.diags }

// AppliedEdits returns the channel receiving per-file notifications
// for server-initiated workspace edits (workspace/applyEdit) that have
// already been applied by the backend, so that the editor layer can
// refresh the affected views. It is closed by [Backend.Stop].
func (b *Backend) AppliedEdits() <-chan []lang.FileEdits { return b.applies }

// serverKey is the caching key for a server config at a workspace root.
func serverKey(cfg *ServerConfig, root string) string {
	return cfg.ID + "\x00" + root
}

// serverFor returns the [Server] to use for the given file, along with
// its document tracking record, starting the server asynchronously if
// it is not already running, and making sure the document is tracked
// by it. It returns false if there is no known server for the file's
// language, or the backend has been stopped.
func (b *Backend) serverFor(ln *lines.Lines) (*Server, *docRef, bool) {
	if ln == nil {
		return nil, nil, false
	}
	fname := ln.Filename()
	if fname == "" {
		return nil, nil, false
	}
	key := normalizePath(fname)
	b.mu.Lock()
	if b.stopped {
		b.mu.Unlock()
		return nil, nil, false
	}
	// fast path: an already-tracked document with a usable server
	if ref := b.docs[key]; ref != nil && ref.ln == ln && b.usable(ref.srv) {
		b.mu.Unlock()
		return ref.srv, ref, true
	}
	b.mu.Unlock()

	// the registry lookup and root search touch the file system,
	// so they are done outside the lock:
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
	skey := serverKey(cfg, root)

	b.mu.Lock()
	if b.stopped {
		b.mu.Unlock()
		return nil, nil, false
	}
	srv := b.servers[skey]
	if srv == nil || !b.usable(srv) {
		srv = NewServer(b.Reg, cfg, root)
		srv.Diags = b.handleDiags
		srv.ApplyEdit = b.handleApplyEdit
		if b.ShowDocument != nil {
			srv.ShowDocument = b.handleShowDocument
		}
		if b.Progress != nil {
			srv.Progress = func(token, title, msg string, done bool) {
				b.handleProgress(cfg.ID, token, title, msg, done)
			}
		}
		b.servers[skey] = srv
		go srv.Start(context.Background()) // errors are logged and recorded
	}
	ref := &docRef{ln: ln, srv: srv, langID: LanguageID(kn, fname), fname: fname}
	b.docs[key] = ref
	b.mu.Unlock()

	// queued until the server is ready, if it is still starting:
	srv.Open(URIFromPath(fname), ref.langID, currentText(ln))
	return srv, ref, true
}

// usable returns whether the given server is the current one for its
// key, and has not failed long enough ago to warrant a restart.
// Must be called under mu.
func (b *Backend) usable(srv *Server) bool {
	if b.servers[serverKey(srv.Cfg, srv.Root)] != srv {
		return false
	}
	since, failed := srv.failedSince()
	return !failed || since < restartDelay
}

// readyServer returns the server for the given buffer if it is ready
// to answer requests, and false otherwise.
func (b *Backend) readyServer(ln *lines.Lines) (*Server, bool) {
	srv, _, has := b.serverFor(ln)
	if !has || !srv.Ready() {
		return nil, false
	}
	return srv, true
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
	_, ok := b.readyServer(ln)
	return ok
}

// Open notifies the server that the file is open, sending its current
// contents, starting the server asynchronously if needed.
func (b *Backend) Open(ln *lines.Lines) {
	b.serverFor(ln)
}

// Change syncs the current contents of the buffer to the server,
// opening the document first if needed (e.g., the file was renamed).
func (b *Backend) Change(ln *lines.Lines) {
	srv, _, has := b.serverFor(ln)
	if !has {
		return
	}
	srv.DidChange(URIFromPath(ln.Filename()), currentText(ln))
}

// Save notifies the server that the file has been saved,
// after syncing its latest text.
func (b *Backend) Save(ln *lines.Lines) {
	srv, _, has := b.serverFor(ln)
	if !has {
		return
	}
	uri := URIFromPath(ln.Filename())
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
	key := normalizePath(fname)
	b.mu.Lock()
	ref := b.docs[key]
	delete(b.docs, key)
	b.mu.Unlock()
	if ref == nil {
		return
	}
	ref.srv.Close(URIFromPath(fname))
	b.maybeShutdown(ref.srv)
}

// maybeShutdown stops the given server if no open docs are still using it.
func (b *Backend) maybeShutdown(srv *Server) {
	b.mu.Lock()
	for _, ref := range b.docs {
		if ref.srv == srv {
			b.mu.Unlock()
			return
		}
	}
	if key := serverKey(srv.Cfg, srv.Root); b.servers[key] == srv {
		delete(b.servers, key)
	}
	b.mu.Unlock()
	srv.Shutdown()
}

// currentText returns the full text of the buffer as a string.
// Note: all exported [lines.Lines] methods lock internally, and the
// Lines mutex is not reentrant, so we must not lock around them here.
func currentText(ln *lines.Lines) string {
	return string(ln.Text())
}

// docID returns the document identifier for the given buffer.
func docID(ln *lines.Lines) TextDocumentIdentifier {
	return TextDocumentIdentifier{URI: URIFromPath(ln.Filename())}
}

// posParams returns the standard position request params for the
// given buffer and editor position.
func posParams(ln *lines.Lines, pos textpos.Pos) TextDocumentPositionParams {
	return TextDocumentPositionParams{TextDocument: docID(ln), Position: toLSPPos(ln, pos)}
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

// SameFile returns whether the two file paths refer to the same file,
// resolving symlinks, e.g., for comparing paths reported by servers
// with buffer filenames.
func SameFile(a, b string) bool {
	return normalizePath(a) == normalizePath(b)
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
	b.sendMu.RLock()
	defer b.sendMu.RUnlock()
	if b.closed {
		return
	}
	select {
	case b.diags <- fd:
	case <-b.done:
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
	case SeverityWarning:
		return lang.DiagWarn
	case SeverityInformation:
		return lang.DiagInfo
	case SeverityHint:
		return lang.DiagHint
	}
	return lang.DiagErr
}

// codeString returns the diagnostic code, which can be a string
// or a number in the protocol, as a string.
func codeString(code any) string {
	switch c := code.(type) {
	case string:
		return c
	case float64:
		return strconv.FormatFloat(c, 'f', -1, 64)
	}
	return ""
}

// progressMsg is the message for one operation in progress.
type progressMsg struct {
	key, msg string
}

// handleProgress updates the operations in progress for the given
// server operation, and calls [Backend.Progress] with the message for
// the most recently updated one.
func (b *Backend) handleProgress(server, token, title, msg string, done bool) {
	key := server + "\x00" + token
	text := server + ": " + title
	if msg != "" {
		text += ": " + msg
	}
	b.mu.Lock()
	b.progress = slices.DeleteFunc(b.progress, func(p progressMsg) bool { return p.key == key })
	if !done {
		b.progress = append(b.progress, progressMsg{key: key, msg: text})
	}
	cur := ""
	if n := len(b.progress); n > 0 {
		cur = b.progress[n-1].msg
	}
	b.mu.Unlock()
	b.Progress(cur)
}

// handleShowDocument implements server window/showDocument requests,
// using [Backend.ShowDocument].
func (b *Backend) handleShowDocument(sd ShowDocumentParams) bool {
	fname := sd.URI.Path()
	if sd.External || fname == "" {
		return b.ShowDocument(string(sd.URI), nil)
	}
	loc := &lang.Location{Filename: fname}
	if sd.Selection != nil {
		loc.Region = b.regionForFile(fname, *sd.Selection)
	}
	return b.ShowDocument(string(sd.URI), loc)
}

////////   Positions

// regionFromLSP converts an LSP range into a [textpos.Region] in the
// editor's rune units, using the given buffer for UTF-16 conversions.
func regionFromLSP(ln *lines.Lines, r Range) textpos.Region {
	st := posFromLSP(ln, r.Start)
	ed := posFromLSP(ln, r.End)
	if ed.IsLess(st) {
		ed = st
	}
	return textpos.Region{Start: st, End: ed}
}

// posFromLSP converts an LSP position to editor line, rune units,
// converting the character offset from UTF-16 code units.
// Positions past the last line map to the end of the buffer.
// Lines methods lock internally; do not wrap them in Lock here.
func posFromLSP(ln *lines.Lines, p Position) textpos.Pos {
	nln := ln.NumLines()
	if nln == 0 || p.Line < 0 {
		return textpos.Pos{}
	}
	if p.Line >= nln {
		return textpos.Pos{Line: nln - 1, Char: ln.LineLen(nln - 1)}
	}
	return textpos.Pos{Line: p.Line, Char: lang.UTF16ToRune(string(ln.Line(p.Line)), p.Character)}
}

// toLSPPos converts an editor line, rune position into an LSP position,
// converting character to UTF-16 code units.
func toLSPPos(ln *lines.Lines, pos textpos.Pos) Position {
	if !ln.IsValidLine(pos.Line) {
		return Position{Line: max(0, pos.Line)}
	}
	return Position{Line: pos.Line, Character: lang.RuneToUTF16(string(ln.Line(pos.Line)), pos.Char)}
}

// toLSPRange converts an editor (rune) region to an LSP (UTF-16) range
// for the given buffer.
func toLSPRange(ln *lines.Lines, reg textpos.Region) Range {
	return Range{Start: toLSPPos(ln, reg.Start), End: toLSPPos(ln, reg.End)}
}

// lineText returns the text of the given line as a string.
func lineText(ln *lines.Lines, line int) string {
	if !ln.IsValidLine(line) {
		return ""
	}
	return string(ln.Line(line))
}

////////   Hover / definition / references

// Hover returns hover documentation for the symbol at the given position.
func (b *Backend) Hover(ln *lines.Lines, pos textpos.Pos) string {
	srv, ok := b.readyServer(ln)
	if !ok {
		return ""
	}
	params := posParams(ln, pos)
	txt, _ := srv.Hover(&params)
	return txt
}

// Definition returns definition locations for the symbol at the position.
func (b *Backend) Definition(ln *lines.Lines, pos textpos.Pos) []lang.Location {
	return b.locations(ln, pos, (*Server).Definition)
}

// TypeDefinition returns the locations of the definition of the type
// of the symbol at the position.
func (b *Backend) TypeDefinition(ln *lines.Lines, pos textpos.Pos) []lang.Location {
	return b.locations(ln, pos, (*Server).TypeDefinition)
}

// Implementations returns the locations of the implementations of the
// interface or method at the position, or of the interfaces that the
// type at the position implements.
func (b *Backend) Implementations(ln *lines.Lines, pos textpos.Pos) []lang.Location {
	return b.locations(ln, pos, (*Server).Implementation)
}

// locations returns the locations from the given server position
// request for the symbol at the position.
func (b *Backend) locations(ln *lines.Lines, pos textpos.Pos, req func(*Server, *TextDocumentPositionParams) ([]Location, error)) []lang.Location {
	srv, ok := b.readyServer(ln)
	if !ok {
		return nil
	}
	params := posParams(ln, pos)
	locs, err := req(srv, &params)
	if err != nil {
		return nil
	}
	return b.convertLocations(locs)
}

// References returns reference locations for the symbol at the position.
func (b *Backend) References(ln *lines.Lines, pos textpos.Pos) []lang.Location {
	srv, ok := b.readyServer(ln)
	if !ok {
		return nil
	}
	params := &ReferenceParams{
		TextDocumentPositionParams: posParams(ln, pos),
		Context:                    ReferenceContext{IncludeDeclaration: true},
	}
	locs, err := srv.References(params)
	if err != nil {
		return nil
	}
	return b.convertLocations(locs)
}

// convertLocations converts LSP locations into [lang.Location],
// converting UTF-16 ranges using the target file's contents.
func (b *Backend) convertLocations(locs []Location) []lang.Location {
	rc := b.newRegionConverter()
	out := make([]lang.Location, 0, len(locs))
	for _, loc := range locs {
		fname := loc.URI.Path()
		if fname == "" {
			continue
		}
		out = append(out, lang.Location{Filename: fname, Region: rc.region(fname, loc.Range)})
	}
	return out
}

// regionConverter converts LSP ranges into editor units for any file,
// using the open buffer if available, and otherwise the file contents
// read from disk, which are cached for the life of the converter.
type regionConverter struct {
	b     *Backend
	files map[string]*docText
}

func (b *Backend) newRegionConverter() *regionConverter {
	return &regionConverter{b: b, files: make(map[string]*docText)}
}

// region returns the given range in the given file in editor units.
func (rc *regionConverter) region(fname string, r Range) textpos.Region {
	if ln := rc.b.openLines(fname); ln != nil {
		return regionFromLSP(ln, r)
	}
	dt, has := rc.files[fname]
	if !has {
		dt, _ = readDocText(fname) // nil on error
		rc.files[fname] = dt
	}
	if dt == nil { // best effort, correct for ASCII
		return textpos.Region{
			Start: textpos.Pos{Line: r.Start.Line, Char: r.Start.Character},
			End:   textpos.Pos{Line: r.End.Line, Char: r.End.Character},
		}
	}
	return dt.region(r)
}

// regionForFile converts an LSP range in the given file into editor units,
// using the open buffer if available, otherwise reading the file from disk.
func (b *Backend) regionForFile(fname string, r Range) textpos.Region {
	return b.newRegionConverter().region(fname, r)
}

// openLines returns the open buffer tracked for the given file, or nil.
func (b *Backend) openLines(fname string) *lines.Lines {
	b.mu.Lock()
	defer b.mu.Unlock()
	if ref := b.docs[normalizePath(fname)]; ref != nil {
		return ref.ln
	}
	return nil
}

// Occurrences returns the regions of the occurrences of the symbol at the
// position in the buffer, for highlighting them, or nil if there is no
// symbol there.
func (b *Backend) Occurrences(ln *lines.Lines, pos textpos.Pos) []textpos.Region {
	srv, ok := b.readyServer(ln)
	if !ok || !jsonEnabled(srv.Caps().DocumentHighlightProvider) {
		return nil
	}
	params := posParams(ln, pos)
	srv.DidChange(params.TextDocument.URI, currentText(ln))
	hs, err := srv.DocumentHighlight(&params)
	if err != nil {
		return nil
	}
	regs := make([]textpos.Region, len(hs))
	for i, h := range hs {
		regs[i] = regionFromLSP(ln, h.Range)
	}
	return regs
}

////////   Symbols

// Symbols returns the hierarchical document symbols for the file.
func (b *Backend) Symbols(ln *lines.Lines) []lang.Symbol {
	srv, ok := b.readyServer(ln)
	if !ok {
		return nil
	}
	syms, err := srv.DocumentSymbol(docID(ln).URI)
	if err != nil {
		return nil
	}
	return convertDocSymbols(ln, syms, ln.Filename())
}

// convertDocSymbols converts the hierarchical LSP document symbols into
// our [lang.Symbol] tree.
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
	srv, ok := b.readyServer(ln)
	if !ok {
		return nil
	}
	infos, err := srv.WorkspaceSymbol(query)
	if err != nil {
		return nil
	}
	rc := b.newRegionConverter()
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
			Region:   rc.region(fname, si.Location.Range),
		})
	}
	return out
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

////////   Selection ranges

// SelectionRanges returns the nested regions around the given position,
// from the innermost (e.g., the word) to the outermost (e.g., the whole
// function), for expanding and shrinking the selection.
func (b *Backend) SelectionRanges(ln *lines.Lines, pos textpos.Pos) []textpos.Region {
	srv, ok := b.readyServer(ln)
	if !ok {
		return nil
	}
	uri := docID(ln).URI
	srv.DidChange(uri, currentText(ln))
	rs, err := srv.SelectionRange(uri, toLSPPos(ln, pos))
	if err != nil {
		return nil
	}
	regs := make([]textpos.Region, len(rs))
	for i, r := range rs {
		regs[i] = regionFromLSP(ln, r)
	}
	return regs
}

////////   Inlay hints

// InlayHints returns the inlay hints the server provides for the
// current contents of the buffer within the given region.
// Note: the editor does not render these; this is for programmatic use.
func (b *Backend) InlayHints(ln *lines.Lines, reg textpos.Region) ([]InlayHint, error) {
	srv, ok := b.readyServer(ln)
	if !ok {
		return nil, nil
	}
	return srv.InlayHint(&InlayHintParams{TextDocument: docID(ln), Range: toLSPRange(ln, reg)})
}
