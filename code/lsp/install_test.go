// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMethodsPlatformPriority(t *testing.T) {
	old := CurrentOS
	defer func() { CurrentOS = old }()
	// an OS that is never a target, so no local PATH probing happens
	// and the plans are deterministic everywhere:
	CurrentOS = "plan9"

	spec := &InstallSpec{
		GoModule: "example.com/x/y",
		Npm:      []string{"z"},
		Pip:      "p",
		Brew:     "y",
		Apt:      "y",
	}
	var tools []string
	norm := func(m InstallMethod) string {
		switch m.Tool {
		case "pipx", "pip":
			return "pip"
		case "sudo", "apt-get":
			return "apt"
		}
		return m.Tool
	}
	for _, m := range spec.Methods("darwin") {
		tools = append(tools, norm(m))
	}
	// homebrew first on mac; apt is linux-only:
	assert.Equal(t, []string{"brew", "go", "npm", "pip"}, tools)

	tools = nil
	for _, m := range spec.Methods("linux") {
		tools = append(tools, norm(m))
	}
	// go and node installs first elsewhere; brew last-but-present:
	assert.Equal(t, []string{"go", "npm", "pip", "brew", "apt"}, tools)

	// no brew on windows:
	tools = nil
	for _, m := range spec.Methods("windows") {
		tools = append(tools, norm(m))
	}
	assert.Equal(t, []string{"go", "npm", "pip"}, tools)

	// guard against the argv[0]-duplication bug on all platforms:
	for _, goos := range []string{"darwin", "linux", "windows"} {
		for _, m := range spec.Methods(goos) {
			if len(m.Args) > 0 {
				assert.NotEqual(t, m.Tool, m.Args[0], "%s: argv[0] must not repeat the tool: %s", goos, m.Desc)
			}
		}
	}
}

func TestMethodsNoBrewFallback(t *testing.T) {
	old := CurrentOS
	defer func() { CurrentOS = old }()
	CurrentOS = old // probe the real local platform
	if CurrentOS != "darwin" && CurrentOS != "linux" {
		t.Skip("brew preference test applies to darwin / linux only")
	}
	spec := &InstallSpec{GoModule: "example.com/x/y", Brew: "y"}
	ms := spec.Methods("")
	if brewExists() {
		if assert.NotEmpty(t, ms) {
			assert.Equal(t, "brew", ms[0].Tool, "brew must be preferred when installed")
		}
		return
	}
	for _, m := range ms {
		assert.NotEqual(t, "brew", m.Tool, "brew must be absent when not installed")
	}
	if len(ms) > 0 {
		assert.Equal(t, "go", ms[0].Tool, "go install is the fallback")
	}
}

func brewExists() bool {
	_, err := exec.LookPath("brew")
	return err == nil
}

func TestRegistrySelect(t *testing.T) {
	rg := NewRegistry(nil)
	all, err := rg.Select(nil, nil)
	assert.NoError(t, err)
	for _, sc := range all {
		assert.NotEqual(t, "pylsp", sc.ID, "optional servers are skipped by default")
	}

	inc, err := rg.Select([]string{"pylsp", "gopls"}, nil)
	assert.NoError(t, err)
	if assert.Len(t, inc, 2) {
		assert.Equal(t, "pylsp", inc[0].ID, "explicit include order is honored")
		assert.Equal(t, "gopls", inc[1].ID)
	}

	ex, err := rg.Select(nil, []string{"gopls"})
	assert.NoError(t, err)
	for _, sc := range ex {
		assert.NotEqual(t, "gopls", sc.ID)
	}

	// explicit include + exclude cancel out:
	both, err := rg.Select([]string{"gopls"}, []string{"gopls"})
	assert.NoError(t, err)
	assert.Empty(t, both)

	_, err = rg.Select([]string{"nope"}, nil)
	assert.Error(t, err)
	_, err = rg.Select(nil, []string{"nope"})
	assert.Error(t, err)
}

func TestNormalizeVersion(t *testing.T) {
	assert.Equal(t, "v0.17.1", normalizeVersion(`{"GoVersion":"go1.23.5","Main":{"Version":"v0.17.1"},"Version":"v0.17.1"}`))
	assert.Equal(t, "v1.2.3", normalizeVersion(`{"Version":"v1.2.3"}`))
	assert.Equal(t, "gopls v0.17.1", normalizeVersion("gopls v0.17.1\nmore lines\n"))
	long := normalizeVersion(string(make([]byte, 200)))
	assert.LessOrEqual(t, len(long), 83)
}
