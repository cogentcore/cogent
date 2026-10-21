// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file defines the subset of the Language Server Protocol (LSP)
// data structures that we use, following the specification:
// https://microsoft.github.io/language-server-protocol/specification
// The initialize handshake and client capability setup are modeled on
// the sample client provided with gopls:
// golang.org/x/tools/gopls/internal/cmd (BSD license).

package lsp

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"cogentcore.org/cogent/code/lang"
)

// DocumentURI represents an LSP URI, which is typically a file:// URL.
type DocumentURI string

// URIFromPath returns the file:// URI for the given file path.
func URIFromPath(path string) DocumentURI {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	return DocumentURI("file://" + filepath.ToSlash(abs))
}

// Path returns the file path for a file:// URI.
// Returns "" if this is not a file URI.
func (uri DocumentURI) Path() string {
	u, err := url.Parse(string(uri))
	if err != nil || u.Scheme != "file" {
		return ""
	}
	return filepath.FromSlash(u.Path)
}

// String returns the URI as a string.
func (uri DocumentURI) String() string { return string(uri) }

// Position is an LSP position: zero-based line and character,
// where the character offset is in UTF-16 code units.
type Position struct {

	// zero-based line number
	Line int `json:"line"`

	// zero-based character offset, in UTF-16 code units
	Character int `json:"character"`
}

// Range represents an interval between two [Position]s.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a location inside a document.
type Location struct {
	URI   DocumentURI `json:"uri"`
	Range Range       `json:"range"`
}

// LocationLink is an alternative form of a cross-file link.
type LocationLink struct {
	TargetURI            DocumentURI `json:"targetUri"`
	TargetRange          Range       `json:"targetRange"`
	TargetSelectionRange Range       `json:"targetSelectionRange"`
}

// TextEdit represents an edit to a text document.
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// TextDocumentIdentifier identifies a text document by URI.
type TextDocumentIdentifier struct {
	URI DocumentURI `json:"uri"`
}

// VersionedTextDocumentIdentifier identifies a document by URI
// and a version number, incremented for each change.
type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version int `json:"version"`
}

// TextDocumentItem is a text document transferred in full,
// used for the didOpen notification.
type TextDocumentItem struct {
	URI        DocumentURI `json:"uri"`
	LanguageID string      `json:"languageId"`
	Version    int         `json:"version"`
	Text       string      `json:"text"`
}

// TextDocumentPositionParams is the standard position-based request params.
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// RenameParams are the params for a textDocument/rename request.
type RenameParams struct {
	TextDocumentPositionParams
	NewName string `json:"newName"`
}

// PrepareRenameParams are the params for a textDocument/prepareRename
// request, which checks whether the symbol at a position can be renamed.
type PrepareRenameParams = TextDocumentPositionParams

////////   Initialize

// ClientInfo identifies the client to the server.
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// InitializeParams are the params for the initialize request.
type InitializeParams struct {

	// process ID of the client, or nil
	ProcessID *int `json:"processId,omitempty"`

	// rootUri is the root path of the workspace as a file URI.
	RootURI DocumentURI `json:"rootUri,omitempty"`

	// rootPath is the deprecated root path field.
	RootPath string `json:"rootPath,omitempty"`

	// initializationOptions holds language-server-specific options,
	// e.g., the "gopls" settings section for gopls.
	InitializationOptions any `json:"initializationOptions,omitempty"`

	// capabilities describes the client's capabilities.
	Capabilities ClientCapabilities `json:"capabilities"`

	// clientInfo identifies this client.
	ClientInfo *ClientInfo `json:"clientInfo,omitempty"`

	// trace sets the verbosity of the trace notifications.
	Trace string `json:"trace,omitempty"`
}

// ClientCapabilities describes what the client supports.
type ClientCapabilities struct {
	Workspace *WorkspaceClientCapabilities `json:"workspace,omitempty"`
	TextDoc   *TextDocumentClientCapab     `json:"textDocument,omitempty"`
	Window    *WindowClientCapabilities    `json:"window,omitempty"`
}

// WorkspaceClientCapabilities are workspace-related client capabilities.
type WorkspaceClientCapabilities struct {
	ApplyEdit             bool                             `json:"applyEdit,omitempty"`
	WorkspaceEdit         *WorkspaceEditCapability         `json:"workspaceEdit,omitempty"`
	DidChangeWatchedFiles *DidChangeWatchedFilesCapability `json:"didChangeWatchedFiles,omitempty"`

	// DidChangeConfiguration must be non-nil for gopls to handle the
	// workspace/didChangeConfiguration notification at all (it gates
	// the handler on this capability being registered).
	DidChangeConfiguration *DidChangeConfigurationCapability `json:"didChangeConfiguration,omitempty"`
	Configuration          bool                              `json:"configuration,omitempty"`
	WorkspaceFolders       bool                              `json:"workspaceFolders,omitempty"`
}

// DidChangeConfigurationCapability describes didChangeConfiguration
// support, per the LSP didChangeConfiguration client capabilities.
type DidChangeConfigurationCapability struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// WorkspaceEditCapability describes workspace edit support.
type WorkspaceEditCapability struct {
	DocumentChanges    bool `json:"documentChanges,omitempty"`
	ResourceOperations bool `json:"resourceOperations,omitempty"`
}

// DidChangeWatchedFilesCapability describes file watching support.
type DidChangeWatchedFilesCapability struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// WindowClientCapabilities are window-related client capabilities.
type WindowClientCapabilities struct {
	WorkDoneProgress bool `json:"workDoneProgress,omitempty"`
}

// TextDocumentClientCapab are textDocument-related client capabilities.
// (the name is shortened from TextDocumentClientCapabilities for space.)
type TextDocumentClientCapab struct {
	Synchronization    *TextDocumentSyncCapability      `json:"synchronization,omitempty"`
	PublishDiagnostics *PublishDiagnosticsCapability    `json:"publishDiagnostics,omitempty"`
	Completion         *CompletionClientCapability      `json:"completion,omitempty"`
	Hover              *HoverClientCapability           `json:"hover,omitempty"`
	Definition         *LinkDefCapability               `json:"definition,omitempty"`
	References         *ReferencesClientCapability      `json:"references,omitempty"`
	DocumentSymbol     *DocumentSymbolClientCapability  `json:"documentSymbol,omitempty"`
	WorkspaceSymbol    *WorkspaceSymbolClientCapability `json:"workspaceSymbol,omitempty"`
	Formatting         *FormattingClientCapability      `json:"formatting,omitempty"`
	Rename             *RenameClientCapability          `json:"rename,omitempty"`
	CodeAction         *CodeActionClientCapability      `json:"codeAction,omitempty"`
	SignatureHelp      *SignatureHelpClientCapability   `json:"signatureHelp,omitempty"`
	InlayHint          *InlayHintClientCapability       `json:"inlayHint,omitempty"`
	SemanticTokens     *SemanticTokensClientCapability  `json:"semanticTokens,omitempty"`
}

// TextDocumentSyncCapability declares text sync support.
type TextDocumentSyncCapability struct {
	DidSave bool `json:"didSave,omitempty"`
}

// PublishDiagnosticsCapability declares diagnostic support.
type PublishDiagnosticsCapability struct {
	RelatedInformation bool `json:"relatedInformation,omitempty"`
	VersionSupport     bool `json:"versionSupport,omitempty"`
}

// CompletionClientCapability declares completion support.
type CompletionClientCapability struct {
	CompletionItem *CompletionItemCapability `json:"completionItem,omitempty"`
	ContextSupport bool                      `json:"contextSupport,omitempty"`
}

// CompletionItemCapability declares CompletionItem feature support.
type CompletionItemCapability struct {
	SnippetSupport      bool            `json:"snippetSupport,omitempty"`
	DocumentationFormat []string        `json:"documentationFormat,omitempty"`
	ResolveSupport      *ResolveSupport `json:"resolveSupport,omitempty"`
	InsertTextFormat    []int           `json:"insertTextFormatSupport,omitempty"`
}

// ResolveSupport declares which properties can be lazily resolved.
type ResolveSupport struct {
	Properties []string `json:"properties,omitempty"`
}

// HoverClientCapability declares hover support.
type HoverClientCapability struct {
	ContentFormat []string `json:"contentFormat,omitempty"`
}

// LinkDefCapability declares definition-link support.
type LinkDefCapability struct {
	LinkSupport bool `json:"linkSupport,omitempty"`
}

// ReferencesClientCapability declares references support.
type ReferencesClientCapability struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DocumentSymbolClientCapability declares document symbol support.
type DocumentSymbolClientCapability struct {
	HierarchicalDocumentSymbolSupport bool                  `json:"hierarchicalDocumentSymbolSupport,omitempty"`
	SymbolKind                        *SymbolKindCapability `json:"symbolKind,omitempty"`
}

// WorkspaceSymbolClientCapability declares workspace symbol support.
type WorkspaceSymbolClientCapability struct {
	SymbolKind *SymbolKindCapability `json:"symbolKind,omitempty"`
}

// SymbolKindCapability declares supported symbol kinds.
type SymbolKindCapability struct {
	ValueSet []int `json:"valueSet,omitempty"`
}

// FormattingClientCapability declares formatting support.
type FormattingClientCapability struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// RenameClientCapability declares rename support.
type RenameClientCapability struct {
	PrepareSupport          bool `json:"prepareSupport,omitempty"`
	HonorsChangeAnnotations bool `json:"honorsChangeAnnotations,omitempty"`
}

// WorkspaceSymbolParams are the params for a workspace/symbol request.
type WorkspaceSymbolParams struct {
	Query string `json:"query"`
}

// DocumentSymbolParams are the params for a textDocument/documentSymbol request.
type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// InitializeResult is the server's response to the initialize request.
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ClientInfo        `json:"serverInfo,omitempty"`
}

// ServerCapabilities describes what the server supports.
// Provider fields can be either a bool or a settings object
// in the protocol, so they are raw JSON with helper methods.
type ServerCapabilities struct {
	TextDocumentSync           rawJSON `json:"textDocumentSync,omitempty"`
	CompletionProvider         rawJSON `json:"completionProvider,omitempty"`
	HoverProvider              rawJSON `json:"hoverProvider,omitempty"`
	DefinitionProvider         rawJSON `json:"definitionProvider,omitempty"`
	ReferencesProvider         rawJSON `json:"referencesProvider,omitempty"`
	DocumentSymbolProvider     rawJSON `json:"documentSymbolProvider,omitempty"`
	WorkspaceSymbolProvider    rawJSON `json:"workspaceSymbolProvider,omitempty"`
	DocumentFormattingProvider rawJSON `json:"documentFormattingProvider,omitempty"`
	RenameProvider             rawJSON `json:"renameProvider,omitempty"`
	CodeActionProvider         rawJSON `json:"codeActionProvider,omitempty"`
	SignatureHelpProvider      rawJSON `json:"signatureHelpProvider,omitempty"`
	InlayHintProvider          rawJSON `json:"inlayHintProvider,omitempty"`
	SemanticTokensProvider     rawJSON `json:"semanticTokensProvider,omitempty"`
}

// rawJSON holds raw JSON for union-typed protocol fields,
// such as capability providers that can be a bool or an object.
type rawJSON = json.RawMessage

// jsonEnabled reports whether a provider capability field is present
// (either true, or an object with settings).
func jsonEnabled(r rawJSON) bool {
	return len(r) > 0 && string(r) != "false" && string(r) != "null"
}

// jsonBoolField extracts a bool field from an object-form capability
// value, or the bool itself if that is the form used.
func jsonBoolField(r rawJSON, name string) bool {
	if string(r) == "true" {
		return true
	}
	var obj map[string]any
	if err := json.Unmarshal(r, &obj); err != nil {
		return false
	}
	b, _ := obj[name].(bool)
	return b
}

////////   Text document sync

// DidOpenTextDocumentParams are sent when a file is opened.
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// DidChangeTextDocumentParams are sent when the content of a file changes.
// With full sync, the ContentChanges have exactly one element with the
// full text of the file, and no Range.
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// TextDocumentContentChangeEvent describes one content change.
// A nil Range means that the Text replaces the entire document.
type TextDocumentContentChangeEvent struct {
	Range *Range `json:"range,omitempty"`
	Text  string `json:"text"`
}

// DidSaveTextDocumentParams are sent when a file is saved.
type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Text         *string                `json:"text,omitempty"`
}

// DidCloseTextDocumentParams are sent when a file is closed.
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

////////   Diagnostics

// DiagnosticSeverity indicates how severe a diagnostic is.
type DiagnosticSeverity int

const (
	// SeverityError is a serious error that prevents compilation.
	SeverityError DiagnosticSeverity = 1

	// SeverityWarning should be addressed.
	SeverityWarning DiagnosticSeverity = 2

	// SeverityInformation is informational.
	SeverityInformation DiagnosticSeverity = 3

	// SeverityHint is a hint for improvement.
	SeverityHint DiagnosticSeverity = 4
)

// Diagnostic represents one diagnostic message for a file.
type Diagnostic struct {
	Range    Range              `json:"range"`
	Severity DiagnosticSeverity `json:"severity,omitempty"`
	Code     any                `json:"code,omitempty"`
	Source   string             `json:"source,omitempty"`
	Message  string             `json:"message"`
}

// PublishDiagnosticsParams are sent by the server
// with the current diagnostics for a file.
type PublishDiagnosticsParams struct {
	URI         DocumentURI  `json:"uri"`
	Version     int          `json:"version,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

////////   Completion

// CompletionContext provides context for a completion request.
type CompletionContext struct {
	TriggerKind      int    `json:"triggerKind"`
	TriggerCharacter string `json:"triggerCharacter,omitempty"`
}

// Completion trigger kinds:
const (
	// CompleteInvoked means completion was explicitly requested.
	CompleteInvoked = 1

	// CompleteTriggerCharacter means it was triggered by a character
	// like "." in the text.
	CompleteTriggerCharacter = 2

	// CompleteTriggerForIncompleteCompletions re-triggers
	// an incomplete completion list.
	CompleteTriggerForIncompleteCompletions = 3

	// CompleteContentChange means the completion list is being updated
	// because the user kept typing; servers filter their candidates
	// against the typed prefix for this kind.
	CompleteContentChange = 4
)

// CompletionParams are the params for a textDocument/completion request.
type CompletionParams struct {
	TextDocumentPositionParams
	Context *CompletionContext `json:"context,omitempty"`
}

// CompletionList is a possibly-incomplete list of completions.
type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

// InsertTextFormat indicates the format of InsertText.
type InsertTextFormat int

const (
	// InsertTextPlain is plain text.
	InsertTextPlain InsertTextFormat = 1

	// InsertTextSnippet is a snippet with ${n:placeholder} forms.
	InsertTextSnippet InsertTextFormat = 2
)

// CompletionItemKind is the LSP kind of a completion item,
// which overlaps with [SymbolKind].
type CompletionItemKind int

// CompletionItem represents one completion option.
type CompletionItem struct {
	Label            string             `json:"label"`
	Kind             CompletionItemKind `json:"kind,omitempty"`
	Detail           string             `json:"detail,omitempty"`
	Documentation    *MarkupContent     `json:"documentation,omitempty"`
	SortText         string             `json:"sortText,omitempty"`
	FilterText       string             `json:"filterText,omitempty"`
	InsertText       string             `json:"insertText,omitempty"`
	InsertTextFormat InsertTextFormat   `json:"insertTextFormat,omitempty"`
	TextEdit         *TextEdit          `json:"textEdit,omitempty"`
	Deprecated       bool               `json:"deprecated,omitempty"`
}

// CompletionResolveParams requests resolution of extra fields
// for a completion item.
type CompletionResolveParams = CompletionItem

////////   Hover

// MarkupContent is text with a markup kind: "" or "plaintext",
// or "markdown".
type MarkupContent struct {
	Kind  string `json:"kind,omitempty"`
	Value string `json:"value"`
}

// HoverParams are the params for a textDocument/hover request.
type HoverParams = TextDocumentPositionParams

// Hover is the result of a textDocument/hover request.
// Contents can be a MarkupContent, a plain string, or an array
// of strings, so it is raw JSON.
type Hover struct {
	Contents rawJSON `json:"contents"`
	Range    *Range  `json:"range,omitempty"`
}

////////   Definitions and references

// DefinitionParams are the params for a textDocument/definition request.
type DefinitionParams = TextDocumentPositionParams

// ReferenceParams are the params for a textDocument/references request.
type ReferenceParams struct {
	TextDocumentPositionParams
	Context ReferenceContext `json:"context"`
}

// ReferenceContext controls whether the declaration itself is included.
type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

////////   Symbols

// SymbolKind is the LSP kind of a symbol.
type SymbolKind int

// DocumentSymbol represents a symbol hierarchy entry.
type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           SymbolKind       `json:"kind"`
	Deprecated     bool             `json:"deprecated,omitempty"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

// SymbolInformation is a flat symbol entry, from workspace/symbol.
type SymbolInformation struct {
	Name          string     `json:"name"`
	Kind          SymbolKind `json:"kind"`
	Deprecated    bool       `json:"deprecated,omitempty"`
	ContainerName string     `json:"containerName,omitempty"`
	Location      Location   `json:"location"`
}

////////   Formatting and workspace edits

// DocumentFormattingParams are the params
// for a textDocument/formatting request.
type DocumentFormattingParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Options      FormattingOptions      `json:"options"`
}

// FormattingOptions describe the client's formatting settings.
type FormattingOptions struct {
	TabSize      int  `json:"tabSize"`
	InsertSpaces bool `json:"insertSpaces"`
}

// WorkspaceEdit represents changes across multiple files.
// Servers use either Changes (a map of URI to edits) or the more
// expressive DocumentChanges (with explicit document versions).
type WorkspaceEdit struct {
	Changes         map[string][]TextEdit `json:"changes,omitempty"`
	DocumentChanges []DocumentEdit        `json:"documentChanges,omitempty"`
}

// DocumentEdit is an edit to specific versioned documents,
// as used in WorkspaceEdit.DocumentChanges. The spec form has a
// single Edit, but some servers (e.g. gopls) send multiple edits
// per document in an Edits array instead.
type DocumentEdit struct {
	TextDocument VersionedTextDocumentIdentifier `json:"textDocument"`
	Edit         *TextEdit                       `json:"edit,omitempty"`
	Edits        []TextEdit                      `json:"edits,omitempty"`
}

////////   Server -> client messages

// ShowMessageParams are sent by the server to display a message.
type ShowMessageParams struct {
	Type    MessageType `json:"type"`
	Message string      `json:"message"`
}

// MessageType indicates the importance of a message.
type MessageType int

// LogMessageParams are sent by the server for logging.
type LogMessageParams struct {
	Type    MessageType `json:"type"`
	Message string      `json:"message"`
}

// ShowMessageRequestParams requests that the user pick an action.
type ShowMessageRequestParams struct {
	ShowMessageParams
	Actions []MessageActionItem `json:"actions,omitempty"`
}

// MessageActionItem is one action option in a message request.
type MessageActionItem struct {
	Title string `json:"title"`
}

// RegistrationParams dynamically registers server capabilities.
type RegistrationParams struct {
	Registrations []Registration `json:"registrations"`
}

// Registration is one dynamic capability registration.
type Registration struct {
	ID     string `json:"id"`
	Method string `json:"method"`
}

// UnregistrationParams removes dynamic capability registrations.
type UnregistrationParams struct {
	Unregistrations []Registration `json:"unregistrations"`
}

// WorkspaceFoldersParams is the response to workspace/workspaceFolders.
type WorkspaceFoldersParams struct {
	URI  DocumentURI `json:"uri"`
	Name string      `json:"name"`
}

// ConfigurationParams are requests for client configuration sections.
type ConfigurationParams struct {
	Items []ConfigurationItem `json:"items"`
}

// ConfigurationItem identifies one configuration section.
type ConfigurationItem struct {
	ScopeURI DocumentURI `json:"scopeUri,omitempty"`
	Section  string      `json:"section,omitempty"`
}

// ProgressParams report server progress.
type ProgressParams struct {
	Token any `json:"token"`

	// Value is a WorkDoneProgressBegin/Report/End object,
	// which all have a "kind" field.
	Value rawJSON `json:"value"`
}

// progressKind extracts the kind field ("begin", "report", "end")
// from a progress value.
func progressKind(v rawJSON) string {
	var obj struct {
		Kind string `json:"kind"`
	}
	json.Unmarshal(v, &obj)
	return obj.Kind
}

// progressMessage extracts the message field from a progress value.
func progressMessage(v rawJSON) string {
	var obj struct {
		Message string `json:"message"`
	}
	json.Unmarshal(v, &obj)
	return obj.Message
}

// ApplyWorkspaceEditParams requests that the client apply an edit
// sent by the server, e.g. as part of a refactoring.
type ApplyWorkspaceEditParams struct {
	Edit  WorkspaceEdit `json:"edit"`
	Label string        `json:"label,omitempty"`
}

// ApplyWorkspaceEditResult reports whether an edit was applied.
type ApplyWorkspaceEditResult struct {
	Applied       bool   `json:"applied"`
	FailureReason string `json:"failureReason,omitempty"`
}

// markupFlatten extracts plain text from Hover contents or
// completion documentation, which can be a MarkupContent, a string,
// or an array of strings / {language,value} objects.
func markupFlatten(raw rawJSON) (string, string) {
	if len(raw) == 0 {
		return "", ""
	}
	var mc MarkupContent
	if err := json.Unmarshal(raw, &mc); err == nil && mc.Value != "" {
		return mc.Kind, mc.Value
	}
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return "plaintext", str
	}
	var arr []rawJSON
	if err := json.Unmarshal(raw, &arr); err == nil {
		var sb strings.Builder
		kind := "plaintext"
		for _, el := range arr {
			var es string
			if err := json.Unmarshal(el, &es); err == nil {
				sb.WriteString(es)
				sb.WriteString("\n")
				continue
			}
			var mc2 struct {
				Language string `json:"language"`
				Value    string `json:"value"`
			}
			if err := json.Unmarshal(el, &mc2); err == nil && mc2.Value != "" {
				fmt.Fprintf(&sb, "```%s\n%s\n```\n", mc2.Language, mc2.Value)
				kind = "markdown"
			}
		}
		return kind, sb.String()
	}
	return "", ""
}

////////   Code actions

// Command is a simple command identifier with arguments, as used in
// code actions and other results. Some servers return commands instead
// of (or in addition to) workspace edits.
type Command struct {

	// Title is the human-readable title.
	Title string `json:"title"`

	// Command is the command identifier, e.g. "gopls.organize_imports".
	Command string `json:"command"`

	// Arguments are command-specific arguments.
	Arguments []any `json:"arguments,omitempty"`
}

// CodeActionKindValueSet lists the action kinds the client understands.
// Servers omit literal code action support unless this is declared.
type CodeActionKindValueSet struct {
	ValueSet []string `json:"valueSet"`
}

// CodeActionLiteralSupport declares kind-based code action support.
type CodeActionLiteralSupport struct {
	CodeActionKind CodeActionKindValueSet `json:"codeActionKind"`
}

// CodeActionClientCapability declares code action support.
type CodeActionClientCapability struct {
	DynamicRegistration      bool                      `json:"dynamicRegistration,omitempty"`
	CodeActionLiteralSupport *CodeActionLiteralSupport `json:"codeActionLiteralSupport,omitempty"`
	IsPreferredSupport       bool                      `json:"isPreferredSupport,omitempty"`
	DataSupport              bool                      `json:"dataSupport,omitempty"`
	ResolveSupport           *ResolveSupport           `json:"resolveSupport,omitempty"`
}

// Code action context trigger kinds.
const (
	// CodeActionTriggerInvoked means explicitly requested by the user.
	CodeActionTriggerInvoked = 1

	// CodeActionTriggerAutomatic means triggered by a keystroke or save.
	CodeActionTriggerAutomatic = 2
)

// CodeActionContext carries the diagnostics the actions should address,
// which is what turns quick fixes on: servers generally only return
// quick fixes for diagnostics included here.
type CodeActionContext struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
	TriggerKind int          `json:"triggerKind,omitempty"`
	Only        []string     `json:"only,omitempty"`
}

// CodeActionParams requests the code actions in a document range.
type CodeActionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      CodeActionContext      `json:"context"`
}

// CodeActionDisabled explains why an action cannot be applied.
type CodeActionDisabled struct {
	Reason string `json:"reason"`
}

// CodeAction is a quick fix or refactoring offered by a server.
// The changes are either provided inline in Edit, retrievable via
// codeAction/resolve (Data set, Edit nil), or require running Command.
type CodeAction struct {

	// Title is the human-readable title, e.g. "Organize imports".
	Title string `json:"title"`

	// Kind is the action kind: "quickfix", "refactor",
	// "refactor.extract", "source.organizeImports", etc.
	// Empty for plain Command results.
	Kind string `json:"kind,omitempty"`

	// Diagnostics are the problems this action addresses.
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`

	// IsPreferred marks the recommended fix for the diagnostics.
	IsPreferred bool `json:"isPreferred,omitempty"`

	// Disabled, if set, marks the action un-appliable with a reason.
	Disabled *CodeActionDisabled `json:"disabled,omitempty"`

	// Edit is the inline workspace edit, if provided.
	Edit *WorkspaceEdit `json:"edit,omitempty"`

	// Command is an associated command to run, when edits are not
	// provided inline or additional work is required.
	Command *Command `json:"command,omitempty"`

	// Data is an opaque token echoed back to codeAction/resolve.
	Data rawJSON `json:"data,omitempty"`
}

////////   Signature help

// SignatureHelpClientCapability declares signature help support.
type SignatureHelpClientCapability struct {
	ContextSupport bool `json:"contextSupport,omitempty"`
}

// SignatureHelpTriggerCharacter is a character that should trigger
// a signature help request, e.g. "(" or ",".
type SignatureHelpTriggerCharacter = string

// Signature help context trigger kinds.
const (
	// SigHelpInvoked means explicitly requested by the user.
	SigHelpInvoked = 1

	// SigHelpTriggerCharacter means it was triggered by a trigger
	// character such as "(" or ",".
	SigHelpTriggerCharacterKind = 2

	// SigHelpContentChange means it was triggered by a content change
	// while signature help was already showing.
	SigHelpContentChange = 3
)

// SignatureHelpContext provides context for a signature help request.
type SignatureHelpContext struct {
	TriggerKind      int    `json:"triggerKind"`
	TriggerCharacter string `json:"triggerCharacter,omitempty"`

	// IsRetrigger is true when signature help was already showing and
	// more content was typed or a cursor movement occurred.
	IsRetrigger bool `json:"isRetrigger"`

	// ActiveSignatureHelp is the previously computed signature help,
	// which servers can use for incremental updates. We do not
	// currently send it back, but the field exists in the protocol.
	ActiveSignatureHelp *SignatureHelp `json:"activeSignatureHelp,omitempty"`
}

// SignatureHelpParams are the params for a textDocument/signatureHelp
// request.
type SignatureHelpParams struct {
	TextDocumentPositionParams
	Context *SignatureHelpContext `json:"context,omitempty"`
}

// ParameterInformation describes one parameter of a signature.
type ParameterInformation struct {

	// Label is either the parameter label as a string, or a
	// [start, end) pair of UTF-16 offsets into the signature label,
	// so it is raw JSON. Use LabelText with the signature label
	// to resolve either form.
	Label rawJSON `json:"label"`

	// Documentation is optional human-readable documentation.
	Documentation *MarkupContent `json:"documentation,omitempty"`
}

// LabelText returns the plain text of a parameter label, resolving the
// offset form against the given signature label if needed.
func (pi *ParameterInformation) LabelText(sigLabel string) string {
	var s string
	if err := json.Unmarshal(pi.Label, &s); err == nil {
		return s
	}
	var offs []int
	if err := json.Unmarshal(pi.Label, &offs); err == nil && len(offs) == 2 {
		rn := []rune(sigLabel)
		st := lang.UTF16ToRune(sigLabel, offs[0])
		ed := lang.UTF16ToRune(sigLabel, offs[1])
		if st >= 0 && ed <= len(rn) && st < ed {
			return string(rn[st:ed])
		}
	}
	return ""
}

// SignatureInformation describes one overloaded signature of a callable.
type SignatureInformation struct {

	// Label is the signature label, e.g. "Read(f *os.File, p []byte) (n int, err error)".
	Label string `json:"label"`

	// Documentation is optional human-readable documentation.
	Documentation *MarkupContent `json:"documentation,omitempty"`

	// Parameters are the parameters of this signature.
	Parameters []ParameterInformation `json:"parameters,omitempty"`

	// ActiveParameter is a server-suggested active parameter index,
	// which most servers leave to the client context to compute.
	ActiveParameter *int `json:"activeParameter,omitempty"`
}

// SignatureHelp is the result of a textDocument/signatureHelp request.
type SignatureHelp struct {
	Signatures      []SignatureInformation `json:"signatures"`
	ActiveSignature *int                   `json:"activeSignature,omitempty"`
	ActiveParameter *int                   `json:"activeParameter,omitempty"`
}

////////   Inlay hints

// InlayHintClientCapability declares inlay hint support.
type InlayHintClientCapability struct {
	ResolveSupport *ResolveSupport `json:"resolveSupport,omitempty"`
}

// InlayHintKind indicates the kind of an inlay hint.
type InlayHintKind int

const (
	// InlayHintType is a type hint, e.g. ": []byte" after a variable.
	InlayHintType InlayHintKind = 1

	// InlayHintParameter is a parameter-name hint, e.g. "f:" before
	// an argument in a function call.
	InlayHintParameter InlayHintKind = 2
)

// InlayHintLabelPart is one segment of a structured inlay hint label.
type InlayHintLabelPart struct {
	Value         string         `json:"value"`
	Tooltip       *MarkupContent `json:"tooltip,omitempty"`
	Location      *Location      `json:"location,omitempty"`
	Command       *Command       `json:"command,omitempty"`
	MarkupContent *MarkupContent `json:"markupContent,omitempty"`
}

// InlayHintParams are the params for a textDocument/inlayHint request.
type InlayHintParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
}

// InlayHint is a virtual text annotation at a position. Note: the
// client currently has no way to render these inline in the editor;
// they are fetched and decoded for programmatic use only.
type InlayHint struct {

	// Position is where the hint is anchored.
	Position Position `json:"position"`

	// Label is either a plain string or an array of label parts,
	// so it is raw JSON. Use LabelText to get the plain text.
	Label rawJSON `json:"label"`

	// Kind distinguishes parameter from type hints.
	Kind InlayHintKind `json:"kind,omitempty"`

	// TextEdits, if present, insert the label text into the document
	// on accept; we do not apply these.
	TextEdits []TextEdit `json:"textEdits,omitempty"`

	// PaddingLeft, PaddingRight add whitespace around the hint.
	PaddingLeft  bool `json:"paddingLeft,omitempty"`
	PaddingRight bool `json:"paddingRight,omitempty"`
}

// LabelText returns the plain text of an inlay hint label, joining the
// parts of the array form if needed.
func (ih *InlayHint) LabelText() string {
	var s string
	if err := json.Unmarshal(ih.Label, &s); err == nil {
		return s
	}
	var parts []InlayHintLabelPart
	if err := json.Unmarshal(ih.Label, &parts); err == nil {
		var sb strings.Builder
		for _, p := range parts {
			sb.WriteString(p.Value)
		}
		return sb.String()
	}
	return ""
}

////////   Semantic tokens

// SemanticTokensClientCapability declares semantic token support.
// The legend arrays must be declared with the exact types the client
// renders; servers only return tokens from this set.
type SemanticTokensClientCapability struct {

	// Requests lists the supported request forms.
	Requests SemanticTokensRequestsClientCapability `json:"requests"`

	// TokenTypes are the token types we understand, in the order of
	// the legend the server uses to encode them.
	TokenTypes []string `json:"tokenTypes"`

	// TokenModifiers are the token modifiers we understand.
	TokenModifiers []string `json:"tokenModifiers"`

	// Formats are the encodings we support; only "relative" is used.
	Formats []string `json:"formats"`
}

// SemanticTokensRequestsClientCapability declares which token requests
// the client supports.
type SemanticTokensRequestsClientCapability struct {
	Range bool `json:"range,omitempty"`
	Full  bool `json:"full,omitempty"`
}

// SemanticTokensLegend is the server's encoding legend, mapping token
// type and modifier indexes to their names.
type SemanticTokensLegend struct {
	TokenTypes     []string `json:"tokenTypes"`
	TokenModifiers []string `json:"tokenModifiers"`
}

// SemanticTokensParams are the params for a
// textDocument/semanticTokens/full request.
type SemanticTokensParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// SemanticTokens is the result of a textDocument/semanticTokens/full
// request. Data is a flat array of five integers per token:
// deltaLine, deltaStartChar, length, tokenType, tokenModifiers (bit set),
// with positions relative to the previous token, in UTF-16 code units,
// and tokenModifiers relative to the legend in [ServerCapabilities].
type SemanticTokens struct {
	ResultID string `json:"resultId,omitempty"`
	Data     []int  `json:"data"`
}

// semanticTokensProviderLegend extracts the legend from the raw
// semanticTokensProvider capability, which can be a bool (no legend,
// no support) or an object with a legend field.
func semanticTokensProviderLegend(r rawJSON) *SemanticTokensLegend {
	var obj struct {
		Legend SemanticTokensLegend `json:"legend"`
	}
	if err := json.Unmarshal(r, &obj); err != nil {
		return nil
	}
	if len(obj.Legend.TokenTypes) == 0 {
		return nil
	}
	return &obj.Legend
}
