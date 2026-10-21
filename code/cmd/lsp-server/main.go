// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// lsp-server manages the language servers used by Cogent's Code editor
// LSP backend: it lists, installs, and tests the standard set of
// supported servers (gopls, typescript-language-server, pyright,
// rust-analyzer, clangd, bash-language-server, vscode-json-language-server,
// and optional alternatives such as pylsp).
//
// By default, operations cover all non-optional servers for the current
// platform; explicit server IDs select a subset, and -x excludes IDs.
// Exit status is 0 on success and 1 if anything was missing or failed.
package main

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"cogentcore.org/core/cli"

	"cogentcore.org/cogent/code/lsp"
)

//go:generate core generate -add-funcs=true

type Config struct {

	// server IDs to operate on (default: all standard servers)
	Args []string `posarg:"all"`

	// comma-separated server IDs to exclude
	Exclude string `flag:"exclude,x"`

	// target platform: darwin, linux, or windows
	// (default: current platform)
	OS string

	// run installers without asking for confirmation
	Yes bool `flag:"yes,y"`

	// preview install commands without running them
	DryRun bool `flag:"dry-run,n"`

	// skip --version probing, for faster listing
	Fast bool `flag:"fast"`

	// test the servers after installing (install command)
	TestAfter bool `flag:"test,t"`

	// per-server timeout for the startup handshake,
	// as a Go duration string like 60s
	Timeout string `default:"60s"`
}

// List shows each supported language server with its languages,
// whether its executable is installed (and version, unless -fast),
// and how it can be installed if missing.
func List(c *Config) error {
	rg := newRegistry()
	remote := c.goos() != lsp.CurrentOS
	sts := rg.Statuses(c.goos(), !c.Fast && !remote)
	idw := 2
	for _, st := range sts {
		idw = max(idw, len(st.ID))
	}
	var missing int
	for _, st := range sts {
		opt := ""
		if st.Optional {
			opt = " (optional)"
		}
		if remote {
			if len(st.Methods) > 0 {
				fmt.Printf("%-*s  %-18s %s%s\n", idw, st.ID, strings.Join(st.Langs, ","), st.Methods[0].Desc, opt)
			} else {
				fmt.Printf("%-*s  %-18s no automated install for this platform%s\n", idw, st.ID, strings.Join(st.Langs, ","), opt)
			}
			continue
		}
		if st.Installed {
			ver := st.Version
			if ver == "" {
				ver = st.Path
			}
			fmt.Printf("%-*s  %-18s installed  %s%s\n", idw, st.ID, strings.Join(st.Langs, ","), ver, opt)
			continue
		}
		missing++
		if len(st.Methods) > 0 {
			fmt.Printf("%-*s  %-18s missing    %s%s\n", idw, st.ID, strings.Join(st.Langs, ","), st.Methods[0].Desc, opt)
			for _, m := range st.Methods[1:] {
				fmt.Printf("%-*s  %-18s or         %s\n", idw, "", "", m.Desc)
			}
		} else {
			fmt.Printf("%-*s  %-18s missing    no automated install for this platform%s\n", idw, st.ID, strings.Join(st.Langs, ","), opt)
		}
	}
	if missing > 0 && !remote {
		return fmt.Errorf("%d server(s) not installed", missing)
	}
	return nil
}

// Install installs the selected language servers (default: all missing
// non-optional servers for the platform), trying each available
// installer method in order. -n previews the plan; -y skips the
// confirmation; -t tests each server after installing.
func Install(c *Config) error {
	rg := newRegistry()
	goos := c.goos()
	if goos != lsp.CurrentOS {
		return fmt.Errorf("can only install servers on the current platform (%s), not %s; use list to see plans", lsp.CurrentOS, goos)
	}
	scs, err := rg.Select(c.Args, c.exclude())
	if err != nil {
		return err
	}
	var todo []*lsp.ServerConfig
	for _, sc := range scs {
		if _, err := rg.ExecPath(sc); err == nil {
			if !slices.Contains(c.Args, sc.ID) { // explicitly included: reinstall
				fmt.Printf("%s: already installed, skipping\n", sc.ID)
				continue
			}
			fmt.Printf("%s: reinstalling (explicitly requested)\n", sc.ID)
		}
		todo = append(todo, sc)
	}
	if len(todo) == 0 {
		fmt.Println("all selected servers are already installed")
		return nil
	}
	fmt.Printf("install plan for %s:\n", goos)
	anyMethod := false
	for _, sc := range todo {
		ms := sc.Install.Methods(goos)
		if len(ms) == 0 {
			fmt.Printf("  %s: NO AUTOMATED METHOD\n", sc.ID)
			for _, in := range sc.Install.Instructions(goos) {
				fmt.Printf("      %s\n", in)
			}
			continue
		}
		anyMethod = true
		fmt.Printf("  %s: %s\n", sc.ID, ms[0].Desc)
		for _, m := range ms[1:] {
			fmt.Printf("      fallback: %s\n", m.Desc)
		}
	}
	if !anyMethod {
		return fmt.Errorf("no automated install methods available")
	}
	if !c.Yes && !c.DryRun {
		fmt.Print("run these installs? [y/N] ")
		if !confirmed() {
			fmt.Println("aborted")
			return nil
		}
	}
	var failed []string
	var done []*lsp.ServerConfig
	for _, sc := range todo {
		if c.DryRun {
			continue
		}
		m, err := lsp.Install(sc, goos)
		if err != nil {
			fmt.Printf("%s: FAILED (%v)\n", sc.ID, err)
			failed = append(failed, sc.ID)
			continue
		}
		fmt.Printf("%s: installed via %s\n", sc.ID, m.Desc)
		done = append(done, sc)
	}
	if c.TestAfter && len(done) > 0 {
		if err := testServers(done, c.timeout()); err != nil {
			return err
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("failed to install: %s", strings.Join(failed, ", "))
	}
	return nil
}

// Test starts each selected installed server, runs the full LSP
// initialize handshake against a small sample project, and reports
// the version and advertised capabilities.
func Test(c *Config) error {
	if c.OS != "" && c.OS != lsp.CurrentOS {
		return fmt.Errorf("can only test servers on the current platform (%s), not %s", lsp.CurrentOS, c.OS)
	}
	rg := newRegistry()
	scs, err := rg.Select(c.Args, c.exclude())
	if err != nil {
		return err
	}
	return testServers(scs, c.timeout())
}

func testServers(scs []*lsp.ServerConfig, timeout time.Duration) error {
	rg := newRegistry()
	var failed []string
	var skipped []string
	for _, sc := range scs {
		if _, err := rg.ExecPath(sc); err != nil {
			skipped = append(skipped, sc.ID)
			continue
		}
		fmt.Printf("testing %s ...\n", sc.ID)
		tr := lsp.TestServer(sc, timeout)
		if !tr.OK {
			fmt.Printf("  %s: FAIL: %s\n", sc.ID, tr.Err)
			failed = append(failed, sc.ID)
			continue
		}
		ver := tr.Version
		if ver == "" {
			ver = "unknown version"
		}
		fmt.Printf("  %s: OK (%s)\n", sc.ID, ver)
		fmt.Printf("      capabilities: %s\n", strings.Join(tr.Caps, ", "))
		if !tr.Symbols {
			fmt.Printf("      note: no symbols returned for sample file (may be normal)\n")
		}
	}
	for _, id := range skipped {
		fmt.Printf("  %s: SKIPPED (not installed)\n", id)
	}
	if len(failed) > 0 {
		return fmt.Errorf("servers failed testing: %s", strings.Join(failed, ", "))
	}
	if len(skipped) > 0 {
		return fmt.Errorf("servers not installed, skipped: %s", strings.Join(skipped, ", "))
	}
	return nil
}

func newRegistry() *lsp.Registry {
	return lsp.NewRegistry(nil)
}

func (c *Config) goos() string {
	if c.OS != "" {
		return c.OS
	}
	return lsp.CurrentOS
}

func (c *Config) exclude() []string {
	var ps []string
	for _, s := range strings.Split(c.Exclude, ",") {
		if s = strings.TrimSpace(s); s != "" {
			ps = append(ps, s)
		}
	}
	return ps
}

func (c *Config) timeout() time.Duration {
	d, err := time.ParseDuration(c.Timeout)
	if err != nil || d <= 0 {
		d = 60 * time.Second
	}
	return d
}

func confirmed() bool {
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(sc.Text())) {
	case "y", "yes":
		return true
	}
	return false
}

func main() { //types:skip
	opts := cli.DefaultOptions("lsp-server", "List, install, and test the language servers supported by Cogent's Code editor LSP backend.")
	opts.PrintSuccess = false
	cli.Run(opts, &Config{}, List, Install, Test)
}
