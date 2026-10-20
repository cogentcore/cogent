// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/base/fileinfo"
)

// TestGopls exercises the client against a real gopls server,
// skipping if gopls is not available in PATH.
func TestGopls(t *testing.T) {
	exe, err := exec.LookPath("gopls")
	if err != nil {
		t.Skip("gopls not found in PATH:", err)
	}
	_ = exe

	// build a self-contained module in a temp dir, away from any
	// parent go.work files that could interfere with discovery:
	tmp := t.TempDir()
	gomod := "module demo\n\ngo 1.21\n"
	src := `package demo

import "fmt"

func Demo() {
	fmt.Pr
	x := undefinedThing()
	_ = x
}
`
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(gomod), 0644); err != nil {
		t.Fatal(err)
	}
	demoPath := filepath.Join(tmp, "demo.go")
	if err := os.WriteFile(demoPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	// ensure gopls doesn't pick up a parent workspace:
	t.Setenv("GOWORK", "off")

	cfg, has := NewRegistry(nil).Server(fileinfo.Go)
	if !has {
		t.Skip("no language server registry entry for Go found")
	}

	srv := NewServer(NewRegistry(nil), cfg, tmp)
	diags := make(chan PublishDiagnosticsParams, 16)
	srv.Diags = func(uri DocumentURI, ds []Diagnostic) {
		select {
		case diags <- PublishDiagnosticsParams{URI: uri, Diagnostics: ds}:
		default:
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("server start: %v", err)
	}
	defer srv.Shutdown()

	if !srv.Ready() {
		t.Fatal("server not ready after Start")
	}

	uri := URIFromPath(demoPath)
	srv.Open(uri, "go", src)

	// wait for diagnostics reporting the undefined error:
	var gotErr bool
	timeout := time.After(60 * time.Second)
	for !gotErr {
		select {
		case pd := <-diags:
			for _, d := range pd.Diagnostics {
				if d.Severity == SeverityError && len(d.Message) > 0 {
					gotErr = true
				}
			}
		case <-timeout:
			t.Fatal("timed out waiting for diagnostics")
		case <-srv.conn.Closed():
			t.Fatal("server connection closed while waiting for diagnostics")
		}
	}
	if !gotErr {
		t.Fatal("no error diagnostics received")
	}

	// completion: fmt.Pr should produce items (Println etc):
	list, err := srv.Completion(&CompletionParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: uri},
			Position:     Position{Line: 5, Character: 6},
		},
		Context: &CompletionContext{TriggerKind: CompleteTriggerCharacter, TriggerCharacter: "."},
	})
	if err != nil {
		t.Fatalf("completion: %v", err)
	}
	if list == nil || len(list.Items) == 0 {
		t.Fatal("no completion items for fmt.Pr")
	}
	foundPrintln := false
	for _, it := range list.Items {
		if it.Label == "Println" {
			foundPrintln = true
		}
	}
	if !foundPrintln {
		t.Errorf("Println not among %d completion items", len(list.Items))
	}

	// document symbols: should include the Demo function:
	syms, err := srv.DocumentSymbol(uri)
	if err != nil {
		t.Fatalf("documentSymbol: %v", err)
	}
	foundDemo := false
	for _, sy := range syms {
		if sy.Name == "Demo" {
			foundDemo = true
		}
	}
	if !foundDemo {
		t.Errorf("Demo not among document symbols: %v", syms)
	}

	// workspace symbols:
	wsyms, err := srv.WorkspaceSymbol("Demo")
	if err != nil {
		t.Fatalf("workspace/symbol: %v", err)
	}
	if len(wsyms) == 0 {
		t.Errorf("no workspace symbols for Demo")
	}

	// definition: Demo at line 4 should resolve to this file:
	locs, err := srv.Definition(&DefinitionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     Position{Line: 4, Character: 6},
	})
	if err != nil {
		t.Fatalf("definition: %v", err)
	}
	if len(locs) == 0 {
		t.Errorf("no definition locations for Demo")
	}
	for _, l := range locs {
		if l.URI.Path() == "" {
			t.Errorf("definition location with non-file URI: %v", l.URI)
		}
	}
	if len(locs) > 0 && filepath.Clean(locs[0].URI.Path()) != filepath.Clean(demoPath) {
		t.Errorf("definition of Demo resolved to %v, want %v", locs[0].URI.Path(), demoPath)
	}
	if locs[0].Range.Start.Line != 4 {
		t.Errorf("definition line %d, want 4", locs[0].Range.Start.Line)
	}

	// fix the file: didChange should eventually clear the error:
	fixed := `package demo

import "fmt"

func Demo() {
	fmt.Println("hi")
}
`
	srv.DidChange(uri, fixed)
	cleared := false
	timeout = time.After(60 * time.Second)
	for !cleared {
		select {
		case pd := <-diags:
			hasErr := false
			for _, d := range pd.Diagnostics {
				if d.Severity == SeverityError {
					hasErr = true
				}
			}
			if !hasErr {
				cleared = true
			}
		case <-timeout:
			t.Fatal("timed out waiting for diagnostics to clear")
		case <-srv.conn.Closed():
			t.Fatal("server connection closed while waiting for diagnostics clear")
		}
	}

	srv.Close(uri)
}
