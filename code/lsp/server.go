// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Verbose enables logging of server log and progress messages.
var Verbose = false

// Version is the client version string reported to servers.
const Version = "0.1.0"

// clientName is the name we identify ourselves as to servers.
// Note: this identifies the client; it must not be used by a server
// to select behavior (see the comment in the sample gopls client).
const clientName = "cogent-code"

// Server state values:
const (
	stateNew      = iota // not started
	stateStarting        // startup in progress
	stateReady           // initialized and ready
	stateFailed          // failed to start or initialize
	stateStopped         // shut down
)

// defaultCallTimeout is the standard timeout for server requests,
// which can involve substantial computation, such as type checking.
const defaultCallTimeout = 15 * time.Second

// quickCallTimeout is the timeout for latency-sensitive requests
// that are made synchronously from the editor, like hover.
const quickCallTimeout = 2 * time.Second

// docState tracks the sync state of one document with a server.
type docState struct {
	uri     DocumentURI
	langID  string
	version int
	text    string // most recent text from the editor
	opened  bool   // didOpen has actually been sent
}

// Server is a running language server process,
// with a single workspace root directory. The request flow and
// initialization handshake are modeled on the sample client provided
// with gopls: golang.org/x/tools/gopls/internal/cmd (BSD license).
type Server struct {

	// Cfg is the configuration for this server.
	Cfg *ServerConfig

	// Root is the workspace root directory for this server.
	Root string

	// Diags receives published diagnostics, if set.
	// It must be set before calling [Start].
	Diags func(uri DocumentURI, diags []Diagnostic)

	// ApplyEdit receives server-initiated workspace edit requests
	// (workspace/applyEdit), if set. It must be set before calling
	// [Start]. It returns whether the edit was applied, and a reason
	// if not.
	ApplyEdit func(we WorkspaceEdit) (applied bool, reason string)

	// Progress receives the progress of server operations, if set:
	// the title of the operation, its current message (with any
	// percentage done), and whether it has ended, keyed by a token
	// that is unique to each operation. It is called from the server
	// connection goroutine and must not block.
	Progress func(token, title, msg string, done bool)

	// ShowDocument receives server requests to show a document
	// (window/showDocument), if set. It must be set before calling
	// [Start], and must not block. It returns whether it was shown.
	ShowDocument func(sd ShowDocumentParams) bool

	// reg is the registry used to find the executable.
	reg *Registry

	// info is the server-provided name and version,
	// from the initialize response, available after Ready.
	info *ClientInfo

	conn *Conn
	cmd  *exec.Cmd

	// options are the effective server options resolved at creation
	// from the registry (built-in InitOptions plus overrides).
	options map[string]any

	// progress is the title of each operation in progress, by token.
	progress map[string]string

	mu       sync.Mutex
	state    int
	err      error
	failTime time.Time // when the server failed, for restart throttling
	docs     map[DocumentURI]*docState
	caps     ServerCapabilities
}

// NewServer returns a new Server for the given config and workspace
// root directory, using the given registry to find the executable and
// resolve the effective server options (built-in InitOptions plus any
// registry-level overrides; see [Registry.SetOptions]).
func NewServer(reg *Registry, cfg *ServerConfig, root string) *Server {
	return &Server{
		Cfg:     cfg,
		Root:    root,
		reg:     reg,
		options: reg.optionsFor(cfg),
		docs:    make(map[DocumentURI]*docState),
	}
}

// Options returns the effective server options for this server: the
// registry-resolved options from startup, with any options set on the
// registry since then merged in (the latter win), so configuration
// re-fetches see live changes. The result is a private copy safe for
// the caller to modify.
func (s *Server) Options() map[string]any {
	return mergeOptions(s.options, s.reg.optionOverride(s.Cfg.ID))
}

// Ready returns whether the server has been started
// and completed the initialize handshake.
func (s *Server) Ready() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state == stateReady
}

// Info returns the server-provided name and version
// from the initialize response, or nil if not yet initialized.
func (s *Server) Info() *ClientInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.info
}

// Caps returns the server capabilities from the initialize response.
func (s *Server) Caps() ServerCapabilities {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.caps
}

// Err returns any startup or initialization error.
func (s *Server) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// Start launches the server process and runs the LSP initialize
// handshake, blocking until the server is ready or an error occurs.
// It is a no-op if the server is already started.
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	switch s.state {
	case stateReady:
		s.mu.Unlock()
		return nil
	case stateFailed, stateStopped:
		err := s.err
		s.mu.Unlock()
		return err
	case stateStarting:
		s.mu.Unlock()
		return fmt.Errorf("lsp: %s server is starting", s.Cfg.ID)
	}
	s.state = stateStarting
	s.mu.Unlock()

	exe, err := s.reg.ExecPath(s.Cfg)
	if err != nil {
		s.fail(err)
		return err
	}
	cmd := exec.Command(exe, s.Cfg.Args...)
	cmd.Dir = s.Root
	cmd.Env = os.Environ()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		s.fail(err)
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.fail(err)
		return err
	}
	if Verbose {
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Start(); err != nil {
		werr := fmt.Errorf("lsp: error starting %s: %w", s.Cfg.ID, err)
		s.fail(werr)
		return werr
	}

	s.mu.Lock()
	s.cmd = cmd
	s.conn = NewConn(stdout, stdin)
	s.conn.SetHandler(s)
	s.mu.Unlock()

	cctx, cancel := context.WithTimeout(ctx, defaultCallTimeout)
	defer cancel()
	// the serve loop must live as long as the connection, not the
	// initialize handshake, so it gets its own background context:
	go s.conn.Serve(context.Background())

	if err := s.initialize(cctx); err != nil {
		werr := fmt.Errorf("lsp: %s: %w", s.Cfg.ID, err)
		s.fail(werr)
		s.kill()
		return werr
	}

	go s.watchExit()

	s.mu.Lock()
	if s.state != stateStarting { // shut down while starting
		s.mu.Unlock()
		return fmt.Errorf("lsp: %s server was stopped", s.Cfg.ID)
	}
	s.state = stateReady
	pending := make([]*docState, 0, len(s.docs))
	for _, ds := range s.docs {
		if !ds.opened {
			pending = append(pending, ds)
		}
	}
	s.mu.Unlock()

	for _, ds := range pending {
		s.didOpenDoc(ds)
	}
	return nil
}

// fail marks the server as having failed.
func (s *Server) fail(err error) {
	s.mu.Lock()
	if s.state != stateStopped {
		s.state = stateFailed
		s.err = err
		s.failTime = time.Now()
	}
	s.mu.Unlock()
	log.Println(err)
}

// failedSince returns how long ago the server failed,
// and false if it has not failed.
func (s *Server) failedSince() (time.Duration, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != stateFailed {
		return 0, false
	}
	return time.Since(s.failTime), true
}

// watchExit marks the server as failed if its connection closes
// (e.g., the process crashes) while it is supposed to be running.
func (s *Server) watchExit() {
	<-s.conn.Closed()
	s.mu.Lock()
	running := s.state == stateReady
	s.mu.Unlock()
	if running {
		s.fail(fmt.Errorf("lsp: %s: server exited unexpectedly", s.Cfg.ID))
		s.kill()
	}
}

// initialize does the two-call LSP client/server handshake,
// based on initParams in the sample gopls client.
func (s *Server) initialize(ctx context.Context) error {
	params := s.InitParams()
	var res InitializeResult
	if err := s.conn.Call(ctx, "initialize", params, &res); err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}
	s.mu.Lock()
	s.caps = res.Capabilities
	s.info = res.ServerInfo
	s.mu.Unlock()
	if err := s.conn.Notify("initialized", &struct{}{}); err != nil {
		return fmt.Errorf("initialized: %w", err)
	}
	return nil
}

// configResult returns the settings object to reply for a given
// configuration section. Servers send this request after initialized
// and again after didChangeConfiguration when the client advertises
// workspace.configuration support; the reply must be the language
// server's own settings object (the "gopls" section in gopls's case).
// It reflects the current effective options including any SetOptions
// applied after the server started. An unknown section gets null,
// meaning "use defaults".
func (s *Server) configResult(section string) any {
	if section == "" || section == s.Cfg.ID || strings.HasPrefix(section, s.Cfg.ID+".") {
		return s.Options()
	}
	return nil
}

// InitParams returns the initialize request params for this server,
// based on initParams from the sample gopls client.
func (s *Server) InitParams() *InitializeParams {
	pid := os.Getpid()
	params := &InitializeParams{
		ProcessID: &pid,
		RootURI:   URIFromPath(s.Root),
		RootPath:  s.Root,
		ClientInfo: &ClientInfo{
			Name:    clientName,
			Version: Version,
		},
		InitializationOptions: s.options,
		Trace:                 "off",
	}
	params.Capabilities.Workspace = &WorkspaceClientCapabilities{
		ApplyEdit:        true,
		WorkspaceEdit:    &WorkspaceEditCapability{DocumentChanges: true, ResourceOperations: []string{"create"}},
		Configuration:    true,
		WorkspaceFolders: true,
		// needed for gopls to run its didChangeConfiguration handler
		// (settings re-fetch) when we send that notification:
		DidChangeConfiguration: &DidChangeConfigurationCapability{},
	}
	params.Capabilities.Window = &WindowClientCapabilities{
		WorkDoneProgress: true,
		ShowDocument:     &ShowDocumentCapability{Support: s.ShowDocument != nil},
	}
	td := &TextDocumentClientCapab{
		Synchronization:    &TextDocumentSyncCapability{DidSave: true},
		PublishDiagnostics: &PublishDiagnosticsCapability{RelatedInformation: true, VersionSupport: true},
		Diagnostic:         &DiagnosticClientCapability{},
		Completion: &CompletionClientCapability{
			CompletionItem: &CompletionItemCapability{
				SnippetSupport:      true,
				DocumentationFormat: []string{"markdown", "plaintext"},
				// note: no ResolveSupport, as completionItem/resolve is not
				// used, so that servers include documentation in the items.
			},
			ContextSupport: true,
		},
		Hover:      &HoverClientCapability{ContentFormat: []string{"markdown", "plaintext"}},
		Definition: &LinkDefCapability{LinkSupport: false},
		References: &ReferencesClientCapability{},
		DocumentSymbol: &DocumentSymbolClientCapability{
			HierarchicalDocumentSymbolSupport: true,
		},
		WorkspaceSymbol: &WorkspaceSymbolClientCapability{},
		Formatting:      &FormattingClientCapability{},
		Rename:          &RenameClientCapability{PrepareSupport: true, HonorsChangeAnnotations: true},
		CodeAction: &CodeActionClientCapability{
			CodeActionLiteralSupport: &CodeActionLiteralSupport{
				CodeActionKind: CodeActionKindValueSet{ValueSet: []string{
					"quickfix", "refactor", "refactor.extract", "refactor.inline",
					"refactor.rewrite", "source", "source.organizeImports", "source.fixAll",
				}},
			},
			IsPreferredSupport: true,
			DataSupport:        true,
			ResolveSupport:     &ResolveSupport{Properties: []string{"edit"}},
		},
		SignatureHelp: &SignatureHelpClientCapability{ContextSupport: true},
		InlayHint:     &InlayHintClientCapability{},
		SemanticTokens: &SemanticTokensClientCapability{
			Requests:       SemanticTokensRequestsClientCapability{Full: true},
			TokenTypes:     clientSemanticTokenTypes,
			TokenModifiers: clientSemanticTokenModifiers,
			Formats:        []string{"relative"},
		},
	}
	params.Capabilities.TextDoc = td
	return params
}

// clientSemanticTokenTypes are the semantic token types we can render,
// declared in the initialize capabilities so servers encode them.
// The order here defines our own legend; servers map their own legend
// indexes to the names, so ours is authoritative for decoding.
var clientSemanticTokenTypes = []string{
	"namespace", "type", "class", "enum", "interface", "struct",
	"typeParameter", "parameter", "variable", "property", "enumMember",
	"event", "function", "method", "macro", "keyword", "modifier",
	"comment", "string", "number", "regexp", "operator", "decorator",
}

// clientSemanticTokenModifiers are the semantic token modifiers
// we understand, in the order of our declared legend.
var clientSemanticTokenModifiers = []string{
	"declaration", "definition", "readonly", "static", "deprecated",
	"abstract", "async", "modification", "documentation", "defaultLibrary",
}

// shutdownTimeout is how long to wait for each stage of a graceful
// server shutdown before killing the process.
const shutdownTimeout = 3 * time.Second

// kill terminates the server process if needed.
func (s *Server) kill() {
	s.mu.Lock()
	cmd := s.cmd
	s.cmd = nil
	s.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}

// Shutdown gracefully shuts down the server with the shutdown request
// and exit notification, killing the process if it does not exit
// within a timeout.
func (s *Server) Shutdown() {
	s.mu.Lock()
	st := s.state
	conn, cmd := s.conn, s.cmd
	s.state = stateStopped
	s.cmd = nil
	s.mu.Unlock()
	if cmd == nil {
		return
	}
	if st != stateReady || conn == nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	err := conn.Call(ctx, "shutdown", nil, nil)
	cancel()
	if err != nil && !errors.Is(err, ErrServerClosed) {
		log.Printf("lsp: %s: shutdown: %v\n", s.Cfg.ID, err)
	}
	_ = conn.Notify("exit", nil)
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(shutdownTimeout):
		_ = cmd.Process.Kill()
		<-done
	}
}

// request makes a request to the server, unless it is not ready,
// in which case it returns an error immediately.
func (s *Server) request(ctx context.Context, method string, params, result any) error {
	s.mu.Lock()
	st, conn := s.state, s.conn
	s.mu.Unlock()
	if st != stateReady || conn == nil {
		return fmt.Errorf("lsp: %s server is not ready", s.Cfg.ID)
	}
	return conn.Call(ctx, method, params, result)
}

// notify sends a notification to the server if it is ready.
func (s *Server) notify(method string, params any) error {
	s.mu.Lock()
	st, conn := s.state, s.conn
	s.mu.Unlock()
	if st != stateReady || conn == nil {
		return fmt.Errorf("lsp: %s server is not ready", s.Cfg.ID)
	}
	return conn.Notify(method, params)
}

// DidChangeConfiguration sends the workspace/didChangeConfiguration
// notification. gopls ignores the params and re-fetches settings via
// workspace/configuration, which gets our current options (see
// [Server.Options], re-resolved from the registry so post-startup
// SetOptions changes are included); we send the settings object too
// for maximal compatibility with servers that read settings directly
// from the notification.
//
// Note: gopls applies the re-fetched settings to its session options,
// and updates folder/view options only when it next rebuilds a snapshot
// (a didChange or didSave event). Callers should therefore follow this
// with a Change sync of open buffers, or expect the new settings to
// apply on the next edit.
func (s *Server) DidChangeConfiguration() error {
	return s.notify("workspace/didChangeConfiguration", map[string]any{s.Cfg.ID: s.Options()})
}

////////   Document sync

// Open adds the given document to the server's tracking,
// sending textDocument/didOpen with the given current text if the
// server is already ready, or queuing it for sending when it starts.
func (s *Server) Open(uri DocumentURI, langID, text string) {
	s.mu.Lock()
	ds, has := s.docs[uri]
	if !has {
		ds = &docState{uri: uri, langID: langID, version: 1, text: text}
		s.docs[uri] = ds
	} else if !ds.opened {
		ds.text = text
	}
	ready := s.state == stateReady
	s.mu.Unlock()
	if ready {
		s.didOpenDoc(ds)
	}
}

// didOpenDoc sends the didOpen notification for a tracked document,
// using its most recent text.
func (s *Server) didOpenDoc(ds *docState) {
	s.mu.Lock()
	if ds.opened || s.state != stateReady {
		s.mu.Unlock()
		return
	}
	ds.opened = true
	version, langID, text := ds.version, ds.langID, ds.text
	s.mu.Unlock()

	params := &DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        ds.uri,
			LanguageID: langID,
			Version:    version,
			Text:       text,
		},
	}
	if err := s.notify("textDocument/didOpen", params); err != nil {
		s.mu.Lock()
		ds.opened = false
		s.mu.Unlock()
	}
}

// DidChange sends the full current text of a tracked document
// as a textDocument/didChange notification, bumping its version.
// If the document has not yet been opened because the server was
// starting, this records the new text for the eventual didOpen.
// Nothing is sent if the server already has this text.
func (s *Server) DidChange(uri DocumentURI, text string) {
	s.didChange(uri, text, false)
}

// ForceDidChange is [Server.DidChange] that sends the text even if it
// is unchanged, e.g., to make the server rebuild its state.
func (s *Server) ForceDidChange(uri DocumentURI, text string) {
	s.didChange(uri, text, true)
}

func (s *Server) didChange(uri DocumentURI, text string, force bool) {
	s.mu.Lock()
	ds, has := s.docs[uri]
	if !has {
		ds = &docState{uri: uri, version: 1}
		s.docs[uri] = ds
	} else if !force && ds.opened && ds.text == text {
		s.mu.Unlock()
		return // already in sync
	}
	ds.version++
	ds.text = text
	ready := s.state == stateReady && ds.opened
	version := ds.version
	s.mu.Unlock()
	if !ready {
		return
	}
	params := &DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: TextDocumentIdentifier{URI: uri},
			Version:                version,
		},
		// full sync: a single change with no range replaces the doc:
		ContentChanges: []TextDocumentContentChangeEvent{{Text: text}},
	}
	_ = s.notify("textDocument/didChange", params)
}

// DidSave sends a textDocument/didSave notification.
func (s *Server) DidSave(uri DocumentURI) {
	_ = s.notify("textDocument/didSave", &DidSaveTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	})
}

// isOpen reports whether the server is currently tracking
// a document for the given URI.
func (s *Server) isOpen(uri DocumentURI) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, has := s.docs[uri]
	return has
}

// Close removes the document from tracking,
// sending textDocument/didClose if it was opened.
func (s *Server) Close(uri DocumentURI) {
	s.mu.Lock()
	ds, has := s.docs[uri]
	if has {
		delete(s.docs, uri)
	}
	opened := has && ds.opened
	s.mu.Unlock()
	if !opened {
		return
	}
	_ = s.notify("textDocument/didClose", &DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	})
}

////////   Language feature requests

// Completion sends a textDocument/completion request,
// returning the resulting list. Uses a short timeout, appropriate
// for interactive use from the editor.
func (s *Server) Completion(posParams *CompletionParams) (*CompletionList, error) {
	var raw json.RawMessage
	ctx, cancel := context.WithTimeout(context.Background(), quickCallTimeout)
	defer cancel()
	if err := s.request(ctx, "textDocument/completion", posParams, &raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	// result can be a plain array or a CompletionList:
	var list CompletionList
	if err := json.Unmarshal(raw, &list); err != nil {
		var items []CompletionItem
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, fmt.Errorf("lsp: bad completion result: %w", err)
		}
		list.Items = items
	}
	return &list, nil
}

// Hover sends a textDocument/hover request, returning flattened
// text (markdown if the server provided it) and whether anything
// was found.
func (s *Server) Hover(posParams *HoverParams) (string, bool) {
	var hv Hover
	ctx, cancel := context.WithTimeout(context.Background(), quickCallTimeout)
	defer cancel()
	if err := s.request(ctx, "textDocument/hover", posParams, &hv); err != nil {
		return "", false
	}
	_, val := markupFlatten(hv.Contents)
	return val, val != ""
}

// Definition sends a textDocument/definition request, returning
// locations in either Location or LocationLink form.
func (s *Server) Definition(posParams *DefinitionParams) ([]Location, error) {
	return s.locations("textDocument/definition", posParams)
}

// TypeDefinition sends a textDocument/typeDefinition request,
// returning the locations of the type of the symbol at the position.
func (s *Server) TypeDefinition(posParams *TextDocumentPositionParams) ([]Location, error) {
	return s.locations("textDocument/typeDefinition", posParams)
}

// Implementation sends a textDocument/implementation request,
// returning the locations of the implementations of the interface
// or method at the position, or the interfaces that a type implements.
func (s *Server) Implementation(posParams *TextDocumentPositionParams) ([]Location, error) {
	return s.locations("textDocument/implementation", posParams)
}

// locations sends a position request with the given method that returns
// locations in either Location or LocationLink form.
func (s *Server) locations(method string, posParams *TextDocumentPositionParams) ([]Location, error) {
	var raw json.RawMessage
	ctx, cancel := context.WithTimeout(context.Background(), defaultCallTimeout)
	defer cancel()
	if err := s.request(ctx, method, posParams, &raw); err != nil {
		return nil, err
	}
	return decodeLocations(raw)
}

// decodeLocations handles both Location and LocationLink forms.
func decodeLocations(raw json.RawMessage) ([]Location, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var locs []Location
	if err := json.Unmarshal(raw, &locs); err == nil {
		return locs, nil
	}
	var links []LocationLink
	if err := json.Unmarshal(raw, &links); err != nil {
		var one Location
		if err := json.Unmarshal(raw, &one); err != nil {
			return nil, fmt.Errorf("lsp: bad location result: %w", err)
		}
		return []Location{one}, nil
	}
	locs = make([]Location, len(links))
	for i, lk := range links {
		locs[i] = Location{URI: lk.TargetURI, Range: lk.TargetSelectionRange}
	}
	return locs, nil
}

// References sends a textDocument/references request.
func (s *Server) References(posParams *ReferenceParams) ([]Location, error) {
	var locs []Location
	ctx, cancel := context.WithTimeout(context.Background(), defaultCallTimeout)
	defer cancel()
	if err := s.request(ctx, "textDocument/references", posParams, &locs); err != nil {
		return nil, err
	}
	return locs, nil
}

// PrepareRename sends a textDocument/prepareRename request, checking
// whether the symbol at the position can be renamed. It returns the
// valid rename range and a suggested placeholder name, which can be
// empty. ok is false if the symbol cannot be renamed at that position.
// err is a method-not-found [RPCError] if the server lacks the feature.
func (s *Server) PrepareRename(posParams *PrepareRenameParams) (rng *Range, placeholder string, ok bool, err error) {
	var raw json.RawMessage
	ctx, cancel := context.WithTimeout(context.Background(), quickCallTimeout)
	defer cancel()
	if err := s.request(ctx, "textDocument/prepareRename", posParams, &raw); err != nil {
		return nil, "", false, err
	}
	return parsePrepareRename(raw)
}

// parsePrepareRename parses the prepareRename result, which can be
// null (not renameable), a Range, a PrepareRenameResult {range,
// placeholder}, or a MarkupRange {range, markup}.
func parsePrepareRename(raw json.RawMessage) (rng *Range, placeholder string, ok bool, err error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, "", false, nil
	}
	var obj struct {
		Start       json.RawMessage `json:"start"`
		Range       json.RawMessage `json:"range"`
		Placeholder string          `json:"placeholder"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, "", false, fmt.Errorf("lsp: bad prepareRename result: %w", err)
	}
	switch {
	case len(obj.Start) > 0: // plain Range
		var r Range
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, "", false, fmt.Errorf("lsp: bad prepareRename result: %w", err)
		}
		return &r, obj.Placeholder, true, nil
	case len(obj.Range) > 0: // wrapped in "range" field
		var r Range
		if err := json.Unmarshal(obj.Range, &r); err != nil {
			return nil, "", false, fmt.Errorf("lsp: bad prepareRename result: %w", err)
		}
		return &r, obj.Placeholder, true, nil
	}
	return nil, "", false, nil
}

// Rename sends a textDocument/rename request, returning the resulting
// workspace edit, or nil if there is nothing to change.
func (s *Server) Rename(params *RenameParams) (*WorkspaceEdit, error) {
	var raw json.RawMessage
	ctx, cancel := context.WithTimeout(context.Background(), defaultCallTimeout)
	defer cancel()
	if err := s.request(ctx, "textDocument/rename", params, &raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var we WorkspaceEdit
	if err := json.Unmarshal(raw, &we); err != nil {
		return nil, fmt.Errorf("lsp: bad rename result: %w", err)
	}
	return &we, nil
}

// supportsCodeAction reports whether the server offers code actions.
func (s *Server) supportsCodeAction() bool {
	return jsonEnabled(s.Caps().CodeActionProvider)
}

// codeActionResolveProvider reports whether the server fills in action
// edits lazily via codeAction/resolve.
func (s *Server) codeActionResolveProvider() bool {
	return jsonObjectBool(s.Caps().CodeActionProvider, "resolveProvider")
}

// CodeAction sends a textDocument/codeAction request, returning the
// offered quick fixes and refactorings.
func (s *Server) CodeAction(params *CodeActionParams) ([]CodeAction, error) {
	return s.codeAction(params, defaultCallTimeout)
}

// CodeActionQuick is CodeAction with the quick, latency-sensitive
// timeout, for use on interactive paths such as hover.
func (s *Server) CodeActionQuick(params *CodeActionParams) ([]CodeAction, error) {
	return s.codeAction(params, quickCallTimeout)
}

func (s *Server) codeAction(params *CodeActionParams, timeout time.Duration) ([]CodeAction, error) {
	var raw json.RawMessage
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := s.request(ctx, "textDocument/codeAction", params, &raw); err != nil {
		return nil, err
	}
	return parseCodeActions(raw)
}

// parseCodeActions decodes a codeAction result, which is a mixed array
// of CodeAction literals and plain Commands (legacy form, per the
// union type in the spec).
func parseCodeActions(raw json.RawMessage) ([]CodeAction, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("lsp: bad codeAction result: %w", err)
	}
	var out []CodeAction
	for _, it := range items {
		var probe struct {
			Command json.RawMessage `json:"command"`
		}
		if err := json.Unmarshal(it, &probe); err != nil {
			continue
		}
		if len(probe.Command) > 0 && probe.Command[0] == '"' {
			// plain Command form: command is a string id, not an object
			var cmd Command
			if err := json.Unmarshal(it, &cmd); err == nil {
				out = append(out, CodeAction{Title: cmd.Title, Kind: "command", Command: &cmd})
			}
			continue
		}
		var ca CodeAction
		if err := json.Unmarshal(it, &ca); err == nil && ca.Title != "" {
			out = append(out, ca)
		}
	}
	return out, nil
}

// ResolveCodeAction fills in deferred parts (the edit) of an action via
// codeAction/resolve, when the server supports it. It is a no-op
// otherwise.
func (s *Server) ResolveCodeAction(ca *CodeAction) error {
	if !s.codeActionResolveProvider() {
		return nil
	}
	var out CodeAction
	ctx, cancel := context.WithTimeout(context.Background(), defaultCallTimeout)
	defer cancel()
	if err := s.request(ctx, "codeAction/resolve", ca, &out); err != nil {
		return err
	}
	if out.Edit != nil {
		ca.Edit = out.Edit
	}
	if out.Command != nil {
		ca.Command = out.Command
	}
	return nil
}

// ExecuteCommand sends a workspace/executeCommand request, returning
// the raw result; many editing commands return a WorkspaceEdit.
func (s *Server) ExecuteCommand(command string, args []any) (json.RawMessage, error) {
	params := struct {
		Command   string `json:"command"`
		Arguments []any  `json:"arguments,omitempty"`
	}{Command: command, Arguments: args}
	var raw json.RawMessage
	ctx, cancel := context.WithTimeout(context.Background(), defaultCallTimeout)
	defer cancel()
	if err := s.request(ctx, "workspace/executeCommand", params, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// SignatureHelp sends a textDocument/signatureHelp request.
// Uses the quick timeout: it is on the typing hot path.
func (s *Server) SignatureHelp(params *SignatureHelpParams) (*SignatureHelp, error) {
	var sh SignatureHelp
	ctx, cancel := context.WithTimeout(context.Background(), quickCallTimeout)
	defer cancel()
	if err := s.request(ctx, "textDocument/signatureHelp", params, &sh); err != nil {
		return nil, err
	}
	if len(sh.Signatures) == 0 {
		return nil, nil
	}
	return &sh, nil
}

// DocumentHighlight sends a textDocument/documentHighlight request,
// returning the occurrences of the symbol at the position in the document.
func (s *Server) DocumentHighlight(posParams *TextDocumentPositionParams) ([]DocumentHighlight, error) {
	var hs []DocumentHighlight
	ctx, cancel := context.WithTimeout(context.Background(), quickCallTimeout)
	defer cancel()
	if err := s.request(ctx, "textDocument/documentHighlight", posParams, &hs); err != nil {
		return nil, err
	}
	return hs, nil
}

// SelectionRange sends a textDocument/selectionRange request for the
// given position, returning the ranges around it, innermost first.
func (s *Server) SelectionRange(uri DocumentURI, pos Position) ([]Range, error) {
	var srs []SelectionRange
	ctx, cancel := context.WithTimeout(context.Background(), quickCallTimeout)
	defer cancel()
	params := &SelectionRangeParams{TextDocument: TextDocumentIdentifier{URI: uri}, Positions: []Position{pos}}
	if err := s.request(ctx, "textDocument/selectionRange", params, &srs); err != nil {
		return nil, err
	}
	var rs []Range
	if len(srs) > 0 {
		for sr := &srs[0]; sr != nil; sr = sr.Parent {
			rs = append(rs, sr.Range)
		}
	}
	return rs, nil
}

// InlayHint sends a textDocument/inlayHint request for the given range.
// Note: the editor does not currently render inlay hints; this is for
// programmatic use (CLI, tooling).
func (s *Server) InlayHint(params *InlayHintParams) ([]InlayHint, error) {
	var hints []InlayHint
	ctx, cancel := context.WithTimeout(context.Background(), defaultCallTimeout)
	defer cancel()
	if err := s.request(ctx, "textDocument/inlayHint", params, &hints); err != nil {
		return nil, err
	}
	return hints, nil
}

// supportsPullDiagnostics reports whether the server provides the
// diagnostics of files on request (textDocument/diagnostic).
func (s *Server) supportsPullDiagnostics() bool {
	return jsonEnabled(s.Caps().DiagnosticProvider)
}

// DocumentDiagnostic sends a textDocument/diagnostic request for the
// file, which need not be open, returning its current diagnostics.
func (s *Server) DocumentDiagnostic(uri DocumentURI) ([]Diagnostic, error) {
	var rep DocumentDiagnosticReport
	ctx, cancel := context.WithTimeout(context.Background(), defaultCallTimeout)
	defer cancel()
	params := &DocumentDiagnosticParams{TextDocument: TextDocumentIdentifier{URI: uri}}
	if err := s.request(ctx, "textDocument/diagnostic", params, &rep); err != nil {
		return nil, err
	}
	return rep.Items, nil
}

// SemanticTokensLegend returns the legend the server advertised for
// semantic tokens, or nil if it does not support them.
func (s *Server) SemanticTokensLegend() *SemanticTokensLegend {
	s.mu.Lock()
	raw := s.caps.SemanticTokensProvider
	s.mu.Unlock()
	return semanticTokensProviderLegend(raw)
}

// SemanticTokensFull sends a textDocument/semanticTokens/full request.
func (s *Server) SemanticTokensFull(uri DocumentURI) (*SemanticTokens, error) {
	var st SemanticTokens
	ctx, cancel := context.WithTimeout(context.Background(), defaultCallTimeout)
	defer cancel()
	params := &SemanticTokensParams{TextDocument: TextDocumentIdentifier{URI: uri}}
	if err := s.request(ctx, "textDocument/semanticTokens/full", params, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

// DocumentSymbol sends a textDocument/documentSymbol request,
// returning hierarchical symbols if the server supports them,
// converting the flat form otherwise.
func (s *Server) DocumentSymbol(uri DocumentURI) ([]DocumentSymbol, error) {
	var raw json.RawMessage
	ctx, cancel := context.WithTimeout(context.Background(), defaultCallTimeout)
	defer cancel()
	params := &DocumentSymbolParams{TextDocument: TextDocumentIdentifier{URI: uri}}
	if err := s.request(ctx, "textDocument/documentSymbol", params, &raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var syms []DocumentSymbol
	if err := json.Unmarshal(raw, &syms); err == nil {
		return syms, nil
	}
	var infos []SymbolInformation
	if err := json.Unmarshal(raw, &infos); err != nil {
		return nil, fmt.Errorf("lsp: bad documentSymbol result: %w", err)
	}
	syms = make([]DocumentSymbol, len(infos))
	for i, si := range infos {
		syms[i] = DocumentSymbol{
			Name:           si.Name,
			Detail:         si.ContainerName,
			Kind:           si.Kind,
			Deprecated:     si.Deprecated,
			Range:          si.Location.Range,
			SelectionRange: si.Location.Range,
		}
	}
	return syms, nil
}

// WorkspaceSymbol sends a workspace/symbol request.
func (s *Server) WorkspaceSymbol(query string) ([]SymbolInformation, error) {
	var infos []SymbolInformation
	ctx, cancel := context.WithTimeout(context.Background(), defaultCallTimeout)
	defer cancel()
	if err := s.request(ctx, "workspace/symbol", &WorkspaceSymbolParams{Query: query}, &infos); err != nil {
		return nil, err
	}
	return infos, nil
}

// Formatting sends a textDocument/formatting request,
// returning the text edits to apply.
func (s *Server) Formatting(uri DocumentURI, opts FormattingOptions) ([]TextEdit, error) {
	var edits []TextEdit
	ctx, cancel := context.WithTimeout(context.Background(), defaultCallTimeout)
	defer cancel()
	params := &DocumentFormattingParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Options:      opts,
	}
	if err := s.request(ctx, "textDocument/formatting", params, &edits); err != nil {
		return nil, err
	}
	return edits, nil
}

// handleProgress handles a $/progress notification.
// It is only called from the connection read goroutine.
func (s *Server) handleProgress(token string, wp *WorkDoneProgress) {
	if s.progress == nil {
		s.progress = make(map[string]string)
	}
	if wp.Kind == "begin" {
		s.progress[token] = wp.Title
	}
	title := s.progress[token]
	done := wp.Kind == "end"
	if done {
		delete(s.progress, token)
	}
	msg := wp.Message
	if wp.Percentage != nil {
		msg = strings.TrimSpace(fmt.Sprintf("%s %d%%", msg, *wp.Percentage))
	}
	if Verbose {
		log.Printf("lsp: %s: progress: %s %s: %s\n", s.Cfg.ID, wp.Kind, title, msg)
	}
	if s.Progress != nil {
		s.Progress(token, title, msg, done)
	}
}

////////   Server -> client downcalls (Handler interface)

// Request handles server-initiated requests, replying with results
// based on our minimal client capabilities, mirroring the downcall
// handling in the sample gopls client.
func (s *Server) Request(method string, params json.RawMessage) (any, *RPCError) {
	switch method {
	case "workspace/configuration":
		var cp ConfigurationParams
		if err := json.Unmarshal(params, &cp); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: err.Error()}
		}
		results := make([]any, len(cp.Items))
		for i, item := range cp.Items {
			// the configuration request is the server re-delivering
			// the settings we sent as initializationOptions; reply
			// per-section, with null (defaults) for unknown sections:
			results[i] = s.configResult(item.Section)
		}
		return results, nil
	case "workspace/workspaceFolders":
		return []WorkspaceFoldersParams{{URI: URIFromPath(s.Root), Name: s.Root}}, nil
	case "client/registerCapability", "client/unregisterCapability":
		// accept dynamic registrations; we don't act on them yet
		return nil, nil
	case "window/workDoneProgress/create":
		return nil, nil
	case "window/showMessageRequest":
		// no action items for now
		return nil, nil
	case "window/showDocument":
		var sd ShowDocumentParams
		if err := json.Unmarshal(params, &sd); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: err.Error()}
		}
		shown := s.ShowDocument != nil && s.ShowDocument(sd)
		return &ShowDocumentResult{Success: shown}, nil
	case "workspace/applyEdit":
		var ap ApplyWorkspaceEditParams
		if err := json.Unmarshal(params, &ap); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: err.Error()}
		}
		if s.ApplyEdit == nil {
			return &ApplyWorkspaceEditResult{Applied: false, FailureReason: clientName + " cannot apply workspace edits right now"}, nil
		}
		applied, reason := s.ApplyEdit(ap.Edit)
		return &ApplyWorkspaceEditResult{Applied: applied, FailureReason: reason}, nil
	default:
		return nil, &RPCError{Code: CodeMethodNotFound, Message: "method not handled: " + method}
	}
}

// Notify handles server-initiated notifications.
func (s *Server) Notify(method string, params json.RawMessage) {
	switch method {
	case "textDocument/publishDiagnostics":
		var pd PublishDiagnosticsParams
		if err := json.Unmarshal(params, &pd); err != nil {
			log.Printf("lsp: bad publishDiagnostics: %v\n", err)
			return
		}
		if s.Diags != nil {
			s.Diags(pd.URI, pd.Diagnostics)
		}
	case "window/showMessage":
		var mp ShowMessageParams
		if err := json.Unmarshal(params, &mp); err == nil {
			log.Printf("lsp: %s: message: %s\n", s.Cfg.ID, mp.Message)
		}
	case "window/logMessage":
		var lp LogMessageParams
		if err := json.Unmarshal(params, &lp); err == nil && Verbose {
			log.Printf("lsp: %s: %s\n", s.Cfg.ID, lp.Message)
		}
	case "$/progress":
		var pp ProgressParams
		var wp WorkDoneProgress
		if json.Unmarshal(params, &pp) != nil || json.Unmarshal(pp.Value, &wp) != nil {
			return
		}
		s.handleProgress(fmt.Sprint(pp.Token), &wp)
	default:
		// ignore other notifications
	}
}
