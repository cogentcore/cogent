// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"cogentcore.org/core/base/fileinfo"
)

// CurrentOS is the platform string (runtime.GOOS style) used for
// install method selection. Tests can override it.
var CurrentOS = runtime.GOOS

// available reports whether the installer tool is usable: the tool
// exists on the PATH on the current platform. For other platforms we
// cannot check, so methods are listed unconditionally.
func available(tool, goos string) bool {
	if goos != CurrentOS {
		return true
	}
	_, err := exec.LookPath(tool)
	return err == nil
}

// methodPriority returns the preferred order of installer tools for
// the given platform: on macOS, homebrew is preferred when installed,
// since it manages everything consistently; elsewhere Go and node
// installs come first.
func methodPriority(goos string) []string {
	if goos == "darwin" {
		return []string{"brew", "go", "npm", "pip", "apt"}
	}
	return []string{"go", "npm", "pip", "brew", "apt"}
}

// Methods returns the installer commands available for the given
// platform (default: [CurrentOS]), in platform order of preference.
// Tools that are not found on the PATH (on the current platform)
// are excluded.
func (is *InstallSpec) Methods(goos string) []InstallMethod {
	if is == nil {
		return nil
	}
	if goos == "" {
		goos = CurrentOS
	}
	byTool := map[string]InstallMethod{}
	if is.GoModule != "" && available("go", goos) {
		byTool["go"] = InstallMethod{
			Tool: "go",
			Args: []string{"install", is.GoModule + "@latest"},
			Desc: "go install " + is.GoModule + "@latest",
		}
	}
	if len(is.Npm) > 0 && available("npm", goos) {
		args := append([]string{"install", "-g"}, is.Npm...)
		byTool["npm"] = InstallMethod{Tool: "npm", Args: args, Desc: "npm " + strings.Join(args, " ")}
	}
	if is.Pip != "" {
		switch {
		case available("pipx", goos):
			byTool["pip"] = InstallMethod{Tool: "pipx", Args: []string{"install", is.Pip}, Desc: "pipx install " + is.Pip}
		case available("pip", goos):
			byTool["pip"] = InstallMethod{Tool: "pip", Args: []string{"install", "--user", is.Pip}, Desc: "pip install --user " + is.Pip}
		}
	}
	if is.Brew != "" && (goos == "darwin" || goos == "linux") && available("brew", goos) {
		byTool["brew"] = InstallMethod{Tool: "brew", Args: []string{"install", is.Brew}, Desc: "brew install " + is.Brew}
	}
	if is.Apt != "" && goos == "linux" && available("apt-get", goos) {
		tool := "apt-get"
		args := []string{"install", "-y", is.Apt}
		if os.Geteuid() != 0 {
			tool = "sudo"
			args = append([]string{"apt-get"}, args...)
		}
		byTool["apt"] = InstallMethod{Tool: tool, Args: args, Desc: tool + " " + strings.Join(args, " ")}
	}
	var ms []InstallMethod
	for _, t := range methodPriority(goos) {
		if m, has := byTool[t]; has {
			ms = append(ms, m)
		}
	}
	return ms
}

// Instructions returns human-readable installation instructions for
// all methods, whether or not their tools are currently available,
// along with the documentation URL as the final line.
func (is *InstallSpec) Instructions(goos string) []string {
	if is == nil {
		return nil
	}
	if goos == "" {
		goos = CurrentOS
	}
	var ps []string
	if is.Brew != "" && (goos == "darwin" || goos == "linux") {
		ps = append(ps, "brew install "+is.Brew)
	}
	if is.GoModule != "" {
		ps = append(ps, "go install "+is.GoModule+"@latest")
	}
	if len(is.Npm) > 0 {
		ps = append(ps, "npm install -g "+strings.Join(is.Npm, " "))
	}
	if is.Pip != "" {
		ps = append(ps, "pipx install "+is.Pip+"  (or: pip install --user "+is.Pip+")")
	}
	if is.Apt != "" && goos == "linux" {
		ps = append(ps, "sudo apt-get install -y "+is.Apt)
	}
	if is.Docs != "" {
		ps = append(ps, "docs: "+is.Docs)
	}
	return ps
}

// ServerStatus is the install status of one language server.
type ServerStatus struct {

	// ID is the server ID.
	ID string

	// Desc is the human-readable description.
	Desc string

	// Langs are the language names it supports.
	Langs []string

	// Installed indicates the executable was found on the PATH
	// (or via a configured override).
	Installed bool

	// Path is the full path of the found executable, if installed.
	Path string

	// Version is the version from probing the executable, if installed
	// and it supports the --version flag.
	Version string

	// Methods are the installer commands available to install it.
	Methods []InstallMethod

	// Optional marks a fallback server that bulk installs skip
	// unless explicitly included.
	Optional bool
}

// Statuses returns the [ServerStatus] for all servers in the registry.
// If probe is true, installed executables are probed for their version
// (a short --version run; empty if unsupported).
func (rg *Registry) Statuses(goos string, probe bool) []*ServerStatus {
	if goos == "" {
		goos = CurrentOS
	}
	var out []*ServerStatus
	for _, sc := range rg.Servers {
		st := &ServerStatus{
			ID:       sc.ID,
			Desc:     sc.Desc,
			Optional: sc.Optional,
		}
		for _, ln := range sc.Langs {
			st.Langs = append(st.Langs, ln.String())
		}
		if p, err := rg.ExecPath(sc); err == nil {
			st.Installed = true
			st.Path = p
			if probe {
				st.Version = probeVersion(p, sc.VersionArgs)
			}
		}
		st.Methods = sc.Install.Methods(goos)
		out = append(out, st)
	}
	return out
}

// probeVersion runs the executable with its version args
// (default: --version) and returns the first line of output,
// or "" if it fails or times out.
func probeVersion(exe string, args []string) string {
	if len(args) == 0 {
		args = []string{"--version"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, args...)
	out, err := cmd.CombinedOutput()
	if err != nil && len(out) == 0 {
		return ""
	}
	ln := strings.TrimSpace(string(out))
	if i := strings.IndexByte(ln, '\n'); i >= 0 {
		ln = ln[:i]
	}
	return ln
}

// Select returns the server configs matching the given selection:
// explicit include IDs (in the given order) if any are given, otherwise
// all servers except [ServerConfig.Optional] ones; exclude IDs are then
// removed. It errors on unknown IDs.
func (rg *Registry) Select(include, exclude []string) ([]*ServerConfig, error) {
	byID := map[string]*ServerConfig{}
	for _, sc := range rg.Servers {
		byID[sc.ID] = sc
	}
	excl := map[string]bool{}
	for _, e := range exclude {
		if e == "" {
			continue
		}
		if _, has := byID[e]; !has {
			return nil, fmt.Errorf("lsp: unknown language server %q in exclude list", e)
		}
		excl[e] = true
	}
	var out []*ServerConfig
	if len(include) > 0 {
		for _, id := range include {
			if id == "" {
				continue
			}
			sc, has := byID[id]
			if !has {
				return nil, fmt.Errorf("lsp: unknown language server %q", id)
			}
			if !excl[id] {
				out = append(out, sc)
			}
		}
		return out, nil
	}
	for _, sc := range rg.Servers {
		if sc.Optional || excl[sc.ID] {
			continue
		}
		out = append(out, sc)
	}
	return out, nil
}

// Install runs the available installer methods for the server in order
// until one succeeds, streaming their output. It returns the command
// that succeeded. Use [InstallSpec.Methods] to preview the plan.
func Install(sc *ServerConfig, goos string) (*InstallMethod, error) {
	ms := sc.Install.Methods(goos)
	if len(ms) == 0 {
		return nil, fmt.Errorf("no automated install method available for %s on %s", sc.ID, goos)
	}
	var lastErr error
	for i := range ms {
		m := &ms[i]
		fmt.Printf("installing %s: %s\n", sc.ID, m.Desc)
		cmd := exec.Command(m.Tool, m.Args...)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			lastErr = fmt.Errorf("%s: %w", m.Desc, err)
			fmt.Fprintf(os.Stderr, "failed: %v\n", lastErr)
			continue
		}
		return m, nil
	}
	return nil, lastErr
}

// TestResult is the outcome of testing one language server:
// a full initialize handshake round trip against a small sample project.
type TestResult struct {

	// ID is the server ID.
	ID string

	// OK indicates the server started and completed the LSP
	// initialize handshake successfully.
	OK bool

	// Version is the server-reported version from the handshake,
	// if provided (may differ from the executable --version output).
	Version string

	// Caps lists the advertised capabilities, e.g. "hover", "rename".
	Caps []string

	// Symbols reports whether a sample documentSymbol request
	// succeeded (informational; does not affect OK).
	Symbols bool

	// Err is the failure message, if any.
	Err string
}

// sampleFile is one file of a test sample project.
type sampleFile struct {

	// Rel is the path relative to the workspace root.
	Rel string

	// Content is the file contents.
	Content string

	// Source marks the main source file to open with the server.
	Source bool
}

// samples maps languages to small sample projects for [TestServer],
// giving the server a realistic workspace with a root marker.
var samples = map[fileinfo.Known][]sampleFile{
	fileinfo.Go: {
		{Rel: "go.mod", Content: "module lspservertest\n\ngo 1.21\n"},
		{Rel: "main.go", Content: "package main\n\nfunc main() {\n\thello()\n}\n\nfunc hello() {\n}\n", Source: true},
	},
	fileinfo.TypeScript: {
		{Rel: "package.json", Content: "{\"name\": \"lspservertest\"}\n"},
		{Rel: "index.ts", Content: "export function hello(): string {\n\treturn \"hi\"\n}\n", Source: true},
	},
	fileinfo.JavaScript: {
		{Rel: "package.json", Content: "{\"name\": \"lspservertest\"}\n"},
		{Rel: "index.js", Content: "export function hello() {\n\treturn \"hi\"\n}\n", Source: true},
	},
	fileinfo.Python: {
		{Rel: "requirements.txt", Content: ""},
		{Rel: "main.py", Content: "def hello() -> str:\n    return \"hi\"\n", Source: true},
	},
	fileinfo.Rust: {
		{Rel: "Cargo.toml", Content: "[package]\nname = \"lspservertest\"\nversion = \"0.1.0\"\nedition = \"2021\"\n"},
		{Rel: "src/main.rs", Content: "fn main() {\n\thello();\n}\n\nfn hello() {\n}\n", Source: true},
	},
	fileinfo.C: {
		{Rel: "main.c", Content: "#include <stdio.h>\n\nint main(void) {\n\tprintf(\"hi\\n\");\n\treturn 0;\n}\n", Source: true},
	},
	fileinfo.Bash: {
		{Rel: "main.sh", Content: "#!/bin/bash\nhello() {\n\techo hi\n}\nhello\n", Source: true},
	},
	fileinfo.Json: {
		{Rel: "data.json", Content: "{\n\t\"hello\": \"world\"\n}\n", Source: true},
	},
}

// TestServer starts the server in a temporary workspace containing a
// small sample project for its first language, verifies the initialize
// handshake, tries a documentSymbol request for extra assurance, and
// shuts it down.
func TestServer(sc *ServerConfig, timeout time.Duration) TestResult {
	tr := TestResult{ID: sc.ID}
	rg := NewRegistry(nil)
	if _, err := rg.ExecPath(sc); err != nil {
		tr.Err = err.Error()
		return tr
	}
	if len(sc.Langs) == 0 {
		tr.Err = "no languages configured"
		return tr
	}
	kn := sc.Langs[0]
	fs, has := samples[kn]
	if !has {
		tr.Err = fmt.Sprintf("no test sample for language %v", kn)
		return tr
	}
	root, err := os.MkdirTemp("", "lspserver-test")
	if err != nil {
		tr.Err = err.Error()
		return tr
	}
	defer os.RemoveAll(root)
	var sample *sampleFile
	for i := range fs {
		p := filepath.Join(root, fs[i].Rel)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			tr.Err = err.Error()
			return tr
		}
		if err := os.WriteFile(p, []byte(fs[i].Content), 0644); err != nil {
			tr.Err = err.Error()
			return tr
		}
		if fs[i].Source {
			sample = &fs[i]
		}
	}
	srv := NewServer(rg, sc, root)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := srv.Start(ctx); err != nil {
		tr.Err = err.Error()
		return tr
	}
	defer srv.Shutdown()
	tr.OK = true
	if info := srv.Info(); info != nil {
		tr.Version = normalizeVersion(info.Version)
		if tr.Version == "" {
			tr.Version = normalizeVersion(info.Name)
		}
	}
	tr.Caps = CapsList(srv.Caps())
	if sample != nil && srv.Ready() {
		uri := URIFromPath(filepath.Join(root, sample.Rel))
		langID := LanguageID(kn, sample.Rel)
		srv.Open(uri, langID, sample.Content)
		if syms, err := srv.DocumentSymbol(uri); err == nil {
			tr.Symbols = len(syms) > 0
		}
	}
	return tr
}

// normalizeVersion extracts a clean version string, unwrapping the
// build-info JSON some servers (e.g., gopls) return as their version,
// and limiting what remains to one short line.
func normalizeVersion(v string) string {
	if i := strings.IndexByte(v, '\n'); i >= 0 {
		v = strings.TrimSpace(v[:i])
	}
	if strings.HasPrefix(v, "{") {
		var o struct {
			Version string `json:"Version"`
			Main    struct {
				Version string `json:"Version"`
			} `json:"Main"`
		}
		if err := json.Unmarshal([]byte(v), &o); err == nil {
			if o.Main.Version != "" {
				return o.Main.Version
			}
			if o.Version != "" {
				return o.Version
			}
		}
	}
	if len(v) > 80 {
		v = v[:80] + "..."
	}
	return v
}

// CapsList returns the names of the capabilities the server advertised,
// for reporting.
func CapsList(caps ServerCapabilities) []string {
	var ps []string
	add := func(name string, v rawJSON) {
		if jsonEnabled(v) {
			ps = append(ps, name)
		}
	}
	add("sync", caps.TextDocumentSync)
	add("completion", caps.CompletionProvider)
	add("hover", caps.HoverProvider)
	add("definition", caps.DefinitionProvider)
	add("references", caps.ReferencesProvider)
	add("documentSymbol", caps.DocumentSymbolProvider)
	add("workspaceSymbol", caps.WorkspaceSymbolProvider)
	add("formatting", caps.DocumentFormattingProvider)
	add("rename", caps.RenameProvider)
	add("codeAction", caps.CodeActionProvider)
	add("signatureHelp", caps.SignatureHelpProvider)
	add("inlayHint", caps.InlayHintProvider)
	add("semanticTokens", caps.SemanticTokensProvider)
	return ps
}
