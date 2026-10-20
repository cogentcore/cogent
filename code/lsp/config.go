// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"cogentcore.org/core/base/fileinfo"
)

// ServerConfig defines a language server: what binary to run,
// which languages it supports, and how to configure it.
type ServerConfig struct {

	// ID is the name of the server, e.g., "gopls",
	// also used as the key for executable path overrides.
	ID string

	// Desc is a human-readable description of the server.
	Desc string

	// Exec is the default name of the server executable
	// to find on the PATH.
	Exec string

	// Args are the arguments for running the server,
	// typically selecting stdio mode.
	Args []string

	// Langs lists the languages supported by this server.
	Langs []fileinfo.Known

	// RootMarkers are file names to look for when walking up the
	// directory tree from an opened file, to find the workspace root
	// directory for this server. The directory containing the first
	// marker found is used as the root. If none are found, the
	// directory of the file is used.
	RootMarkers []string

	// InitOptions are server-specific initialization options,
	// sent as the initializationOptions in the initialize request,
	// and returned for workspace/configuration requests.
	InitOptions map[string]any
}

// DefaultServers is the list of standard language server configurations,
// in order of preference when multiple servers support the same language.
var DefaultServers = []*ServerConfig{
	{
		ID:          "gopls",
		Desc:        "gopls, the official Go language server",
		Exec:        "gopls",
		Args:        []string{"serve"},
		Langs:       []fileinfo.Known{fileinfo.Go},
		RootMarkers: []string{"go.work", "go.mod"},
		InitOptions: map[string]any{
			// these are the CLI defaults from the sample gopls client:
			"analyses": map[string]any{
				"fillreturns":    true,
				"nonewvars":      true,
				"noresultvalues": true,
			},
			"completeUnimported": true,
			"usePlaceholders":    true,
			"staticcheck":        true,
		},
	},
	{
		ID:          "typescript-language-server",
		Desc:        "TypeScript / JavaScript language server",
		Exec:        "typescript-language-server",
		Args:        []string{"--stdio"},
		Langs:       []fileinfo.Known{fileinfo.TypeScript, fileinfo.JavaScript},
		RootMarkers: []string{"tsconfig.json", "jsconfig.json", "package.json"},
		InitOptions: map[string]any{},
	},
	{
		ID:          "pyright-langserver",
		Desc:        "Pyright Python language server",
		Exec:        "pyright-langserver",
		Args:        []string{"--stdio"},
		Langs:       []fileinfo.Known{fileinfo.Python},
		RootMarkers: []string{"pyproject.toml", "setup.py", "setup.cfg", "Pipfile", "requirements.txt"},
		InitOptions: map[string]any{},
	},
	{
		ID:          "pylsp",
		Desc:        "Python LSP server (pip install pylsp)",
		Exec:        "pylsp",
		Langs:       []fileinfo.Known{fileinfo.Python},
		RootMarkers: []string{"pyproject.toml", "setup.py", "setup.cfg", "Pipfile", "requirements.txt"},
		InitOptions: map[string]any{},
	},
	{
		ID:          "rust-analyzer",
		Desc:        "Rust analyzer language server",
		Exec:        "rust-analyzer",
		Langs:       []fileinfo.Known{fileinfo.Rust},
		RootMarkers: []string{"Cargo.toml"},
		InitOptions: map[string]any{},
	},
	{
		ID:          "clangd",
		Desc:        "clangd C / C++ language server",
		Exec:        "clangd",
		Langs:       []fileinfo.Known{fileinfo.C},
		RootMarkers: []string{"compile_commands.json"},
		InitOptions: map[string]any{},
	},
	{
		ID:          "bash-language-server",
		Desc:        "Bash language server",
		Exec:        "bash-language-server",
		Args:        []string{"start"},
		Langs:       []fileinfo.Known{fileinfo.Bash},
		InitOptions: map[string]any{},
	},
	{
		ID:          "vscode-json-language-server",
		Desc:        "JSON language server",
		Exec:        "vscode-json-language-server",
		Args:        []string{"--stdio"},
		Langs:       []fileinfo.Known{fileinfo.Json},
		InitOptions: map[string]any{},
	},
}

// langExtIDs maps language known types to LSP languageId strings.
var langExtIDs = map[fileinfo.Known]string{
	fileinfo.Go:         "go",
	fileinfo.TypeScript: "typescript",
	fileinfo.JavaScript: "javascript",
	fileinfo.Python:     "python",
	fileinfo.Rust:       "rust",
	fileinfo.C:          "c",
	fileinfo.Bash:       "shellscript",
	fileinfo.Json:       "json",
	fileinfo.Html:       "html",
	fileinfo.Css:        "css",
	fileinfo.Markdown:   "markdown",
	fileinfo.Toml:       "toml",
}

// fileExtIDs maps file extensions to LSP languageId strings, used as a
// fallback when the Known type does not have an explicit mapping,
// following the approach of the sample gopls client.
var fileExtIDs = map[string]string{
	".go":   "go",
	".mod":  "go.mod",
	".sum":  "go.sum",
	".work": "go.work",
	".s":    "go.s",
	".ts":   "typescript",
	".tsx":  "typescriptreact",
	".js":   "javascript",
	".jsx":  "javascriptreact",
	".py":   "python",
	".rs":   "rust",
	".c":    "c",
	".h":    "cpp",
	".cpp":  "cpp",
	".sh":   "shellscript",
	".bash": "shellscript",
	".json": "json",
	".html": "html",
	".css":  "css",
	".md":   "markdown",
	".toml": "toml",
	".yaml": "yaml",
	".yml":  "yaml",
}

// LanguageID returns the LSP languageId for the given file, based on
// its Known language type, falling back on the file extension.
// Returns "" if there is no mapping.
func LanguageID(kn fileinfo.Known, fname string) string {
	if id, has := langExtIDs[kn]; has && id != "" {
		return id
	}
	if id, has := fileExtIDs[strings.ToLower(filepath.Ext(fname))]; has {
		return id
	}
	return ""
}

// Registry maintains the set of available language servers,
// with optional executable path overrides from settings.
type Registry struct {

	// Servers is the list of server configurations to consult,
	// defaulting to [DefaultServers].
	Servers []*ServerConfig

	// ExecPaths overrides the executable path for a server,
	// keyed by server [ServerConfig.ID].
	ExecPaths map[string]string

	mu        sync.Mutex
	langsOnce bool
	langMap   map[fileinfo.Known][]*ServerConfig
}

// NewRegistry returns a new Registry using the [DefaultServers],
// with the given optional executable path overrides
// (server ID -> full path or name).
func NewRegistry(execPaths map[string]string) *Registry {
	return &Registry{
		Servers:   DefaultServers,
		ExecPaths: execPaths,
	}
}

// configFor returns the configs for the given language,
// nil if no server configs claim that language.
func (rg *Registry) configFor(kn fileinfo.Known) []*ServerConfig {
	rg.mu.Lock()
	defer rg.mu.Unlock()
	if !rg.langsOnce {
		rg.langMap = make(map[fileinfo.Known][]*ServerConfig)
		for _, sc := range rg.Servers {
			for _, ln := range sc.Langs {
				rg.langMap[ln] = append(rg.langMap[ln], sc)
			}
		}
		rg.langsOnce = true
	}
	return rg.langMap[kn]
}

// Server returns the first available [ServerConfig] for the given
// language, checking executable path overrides and then the PATH
// for each config in preference order.
// It returns false if no server for that language can be found.
func (rg *Registry) Server(kn fileinfo.Known) (*ServerConfig, bool) {
	for _, sc := range rg.configFor(kn) {
		if _, err := rg.ExecPath(sc); err == nil {
			return sc, true
		}
	}
	return nil, false
}

// ExecPath returns the full executable path for the given server config,
// using any override path from settings, and otherwise searching the PATH.
func (rg *Registry) ExecPath(sc *ServerConfig) (string, error) {
	rg.mu.Lock()
	ovr, has := rg.ExecPaths[sc.ID]
	rg.mu.Unlock()
	if has && ovr != "" {
		if filepath.IsAbs(ovr) {
			return ovr, nil
		}
		if p, err := exec.LookPath(ovr); err == nil {
			return p, nil
		}
	}
	p, err := exec.LookPath(sc.Exec)
	if err != nil {
		return "", fmt.Errorf("lsp: language server %q executable %q not found in PATH: %w", sc.ID, sc.Exec, err)
	}
	return p, nil
}

// RootDir finds the workspace root directory for the server config,
// starting from the directory of the given file path, and walking up
// the directory tree looking for any of the [ServerConfig.RootMarkers]
// files. If no markers are given or found, it returns the directory of
// the file itself.
func (sc *ServerConfig) RootDir(fpath string) string {
	dir := filepath.Dir(fpath)
	if len(sc.RootMarkers) == 0 {
		return dir
	}
	cur := dir
	for {
		for _, mk := range sc.RootMarkers {
			st, err := os.Stat(filepath.Join(cur, mk))
			if err == nil && !st.IsDir() {
				return cur
			}
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return dir
		}
		cur = parent
	}
}
