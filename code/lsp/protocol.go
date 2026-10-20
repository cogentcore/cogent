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
	Configuration         bool                             `json:"configuration,omitempty"`
	WorkspaceFolders      bool                             `json:"workspaceFolders,omitempty"`
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
type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes,omitempty"`
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

// ApplyWorkspaceEditParams requests that the client apply an edit;
// we do not currently support this, and always reject.
type ApplyWorkspaceEditParams struct {
	Edit WorkspaceEdit `json:"edit"`
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
