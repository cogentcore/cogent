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
	"net/http"
	"os"
	"os/exec"
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

	// cfg is the configuration for this server.
	Cfg *ServerConfig

	// root is the workspace root directory for this server.
	Root string

	// Diags receives published diagnostics, if set.
	// It must be set before calling [Start].
	Diags func(uri DocumentURI, diags []Diagnostic)

	// ApplyEdit receives server-initiated workspace edit requests
	// (workspace/applyEdit), if set. It must be set before calling
	// [Start]. It returns whether the edit was applied, and a reason
	// if not.
	ApplyEdit func(we WorkspaceEdit) (applied bool, reason string)

	// reg is the registry used to find the executable.
	reg *Registry

	// info is the server-provided name and version,
	// from the initialize response, available after Ready.
	info *ClientInfo

	conn *Conn
	cmd  *exec.Cmd

	mu    sync.Mutex
	state int
	err   error
	docs  map[DocumentURI]*docState
	caps  ServerCapabilities
}

// NewServer returns a new Server for the given config and workspace
// root directory, using the given registry to find the executable.
func NewServer(reg *Registry, cfg *ServerConfig, root string) *Server {
	return &Server{
		Cfg:  cfg,
		Root: root,
		reg:  reg,
		docs: make(map[DocumentURI]*docState),
	}
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

	s.mu.Lock()
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
	}
	s.mu.Unlock()
	log.Println(err)
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
		InitializationOptions: s.Cfg.InitOptions,
		Trace:                 "off",
	}
	params.Capabilities.Workspace = &WorkspaceClientCapabilities{
		ApplyEdit:        true,
		WorkspaceEdit:    &WorkspaceEditCapability{DocumentChanges: true},
		Configuration:    true,
		WorkspaceFolders: true,
	}
	params.Capabilities.Window = &WindowClientCapabilities{
		WorkDoneProgress: true,
	}
	td := &TextDocumentClientCapab{
		Synchronization:    &TextDocumentSyncCapability{DidSave: true},
		PublishDiagnostics: &PublishDiagnosticsCapability{RelatedInformation: true, VersionSupport: true},
		Completion: &CompletionClientCapability{
			CompletionItem: &CompletionItemCapability{
				SnippetSupport:      true,
				DocumentationFormat: []string{"markdown", "plaintext"},
				ResolveSupport:      &ResolveSupport{Properties: []string{"documentation"}},
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
	}
	params.Capabilities.TextDoc = td
	return params
}

// kill terminates the server process if needed.
func (s *Server) kill() {
	s.mu.Lock()
	cmd := s.cmd
	s.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}
}

// Shutdown gracefully shuts down the server, with a timeout
// after which the process is killed.
func (s *Server) Shutdown() {
	s.mu.Lock()
	st := s.state
	conn := s.conn
	s.state = stateStopped
	s.mu.Unlock()
	if st != stateReady && st != stateFailed {
		s.kill()
		return
	}
	if conn != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := conn.Call(ctx, "shutdown", nil, nil)
		if err != nil && !errors.Is(err, ErrServerClosed) {
			log.Printf("lsp: %s: shutdown: %v\n", s.Cfg.ID, err)
		}
		_ = conn.Notify("exit", nil)
		cancel()
	}
	done := make(chan struct{})
	go func() {
		s.kill()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
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
func (s *Server) DidChange(uri DocumentURI, text string) {
	s.mu.Lock()
	ds, has := s.docs[uri]
	if !has {
		ds = &docState{uri: uri, version: 1}
		s.docs[uri] = ds
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
	var raw json.RawMessage
	ctx, cancel := context.WithTimeout(context.Background(), defaultCallTimeout)
	defer cancel()
	if err := s.request(ctx, "textDocument/definition", posParams, &raw); err != nil {
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

// supportsPrepareRename reports whether the server advertised
// prepareProvider in its renameProvider capability.
func (s *Server) supportsPrepareRename() bool {
	s.mu.Lock()
	raw := s.caps.RenameProvider
	s.mu.Unlock()
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "false" {
		return false
	}
	if string(raw) == "true" {
		return false // plain capability: rename without prepare support
	}
	var rp struct {
		PrepareProvider bool `json:"prepareProvider"`
	}
	json.Unmarshal(raw, &rp)
	return rp.PrepareProvider
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
	if Verbose && len(raw) > 0 {
		dbg := string(raw)
		if len(dbg) > 600 {
			dbg = dbg[:600] + "..."
		}
		log.Printf("lsp: %s: rename raw: %s\n", s.Cfg.ID, dbg)
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
			Kind:           si.Kind,
			Deprecated:     si.Deprecated,
			Range:          si.Location.Range,
			SelectionRange: si.Location.Range,
		}
		// note: flat form includes file info which we track at this level:
		syms[i].Detail = si.ContainerName
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

////////   Server -> client downcalls (Handler interface)

// Request handles server-initiated requests, replying with results
// based on our minimal client capabilities, mirroring the downcall
// handling in the sample gopls client.
func (s *Server) Request(method string, params json.RawMessage) (any, *RPCError) {
	switch method {
	case "workspace/configuration":
		var cp ConfigurationParams
		if err := json.Unmarshal(params, &cp); err != nil {
			return nil, &RPCError{Code: http.StatusBadRequest, Message: err.Error()}
		}
		results := make([]any, len(cp.Items))
		for i, item := range cp.Items {
			if item.Section == "" || item.Section == s.Cfg.ID {
				results[i] = s.Cfg.InitOptions
			}
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
	case "workspace/applyEdit":
		var ap ApplyWorkspaceEditParams
		if err := json.Unmarshal(params, &ap); err != nil {
			return nil, &RPCError{Code: http.StatusBadRequest, Message: err.Error()}
		}
		if s.ApplyEdit == nil {
			return &ApplyWorkspaceEditResult{Applied: false, FailureReason: clientName + " cannot apply workspace edits right now"}, nil
		}
		applied, reason := s.ApplyEdit(ap.Edit)
		return &ApplyWorkspaceEditResult{Applied: applied, FailureReason: reason}, nil
	default:
		return nil, &RPCError{Code: http.StatusNotImplemented, Message: "method not handled: " + method}
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
		if err := json.Unmarshal(params, &pp); err == nil && Verbose {
			if kind := progressKind(pp.Value); kind == "report" {
				log.Printf("lsp: %s: progress: %s\n", s.Cfg.ID, progressMessage(pp.Value))
			}
		}
	default:
		// ignore other notifications
	}
}
