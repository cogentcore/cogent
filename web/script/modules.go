// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/evanw/esbuild/pkg/api"
)

// The modules of a page are each transformed by esbuild into a function
// in the CommonJS format (which goja runs), and defined once in the
// registry of the prelude (__defineModule) by URL, with the URLs of the
// modules that it imports. A module is evaluated when it is first
// imported (required), so that the modules that the module scripts of a
// page share are evaluated once, as in browsers (bundling each script on
// its own would evaluate them for each, defining custom elements again).

// moduleDef is the definition of a module: its URL and the script that
// defines it in the registry.
type moduleDef struct {
	url, script string
}

// importsSpec returns whether the given source of a module imports the
// given specifier (with import or export from), as its require calls
// after it is transformed by esbuild are then imports.
func importsSpec(src, spec string) bool {
	for _, q := range []string{`"`, `'`} {
		lit := q + spec + q
		for i := strings.Index(src, lit); i >= 0; {
			before := strings.TrimRight(src[:i], " \t\n(")
			if strings.HasSuffix(before, "from") || strings.HasSuffix(before, "import") {
				return true
			}
			j := strings.Index(src[i+len(lit):], lit)
			if j < 0 {
				break
			}
			i += len(lit) + j
		}
	}
	return false
}

// requireCall matches the imports of a module transformed by esbuild:
// calls of require, not of methods named require (as a.require("util")
// of lodash in a chunk of GitHub, which was fetched as a module).
var requireCall = regexp.MustCompile(`(?:^|[^.\w$])require\(("(?:[^"\\]|\\.)*")\)`)

// moduleGraph returns the definitions of the module at the given URL
// (with the given code, if not nil, as for a module script in the page,
// and the given import.meta.url, if not "") and of the modules that it
// imports, directly or not, that are not defined in the registry yet.
// The modules are gotten in parallel.
func (e *Engine) moduleGraph(entry string, code *string, metaURL string) ([]moduleDef, error) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	var defs []moduleDef
	var firstErr error
	seen := map[string]bool{}
	var load func(u string, code *string, meta string)
	load = func(u string, code *string, meta string) {
		defer wg.Done()
		var src string
		if code != nil {
			src = *code
		} else {
			var err error
			if src, err = e.getScript(u); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
		}
		script, deps, err := e.transformModule(src, u, meta)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return
		}
		defs = append(defs, moduleDef{url: u, script: script})
		for _, d := range deps {
			if !seen[d] && !e.moduleDefined(d) {
				seen[d] = true
				wg.Add(1)
				go load(d, nil, "")
			}
		}
	}
	seen[entry] = true
	wg.Add(1)
	go load(entry, code, metaURL)
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	// a stable order, for tests (they are evaluated when required)
	sort.Slice(defs, func(i, j int) bool { return defs[i].url < defs[j].url })
	return defs, nil
}

// moduleDefined returns whether the module at the given URL has been
// defined in the registry.
func (e *Engine) moduleDefined(u string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.modules[u]
}

// defineModules runs the given definitions of modules (in a task).
func (e *Engine) defineModules(defs []moduleDef) {
	for _, d := range defs {
		if e.moduleDefined(d.url) {
			continue // by another script meanwhile
		}
		e.run(d.url, d.script)
		e.mu.Lock()
		if e.modules == nil {
			e.modules = map[string]bool{}
		}
		e.modules[d.url] = true
		e.mu.Unlock()
	}
}

// transformModule returns the given module source (of the given URL) as
// a script that defines it in the registry, and the URLs of the modules
// that it imports. Its import.meta.url is the given one, if not "", or
// its URL.
func (e *Engine) transformModule(src, u, metaURL string) (string, []string, error) {
	if metaURL == "" {
		metaURL = u
	}
	path := strings.ToLower(strings.Split(strings.Split(u, "#")[0], "?")[0])
	loader := api.LoaderJS
	switch {
	case strings.HasSuffix(path, ".css"):
		src, loader = "", api.LoaderJS // style sheets are not applied
	case strings.HasSuffix(path, ".json"):
		loader = api.LoaderJSON
	default:
		src = rewriteImports(src, u)
		src = importMetaURL.ReplaceAllLiteralString(src, quoteJS(metaURL))
	}
	res := api.Transform(src, api.TransformOptions{
		Loader:     loader,
		Format:     api.FormatCommonJS,
		Target:     api.ESNext,
		Supported:  unsupported,
		Sourcefile: u,
		LogLevel:   api.LogLevelSilent,
	})
	if len(res.Errors) > 0 {
		return "", nil, buildError(res.Errors)
	}
	out := string(res.Code)
	// the calls of require in the module itself are not imports (as the
	// optional dependency of framer-motion in a chunk of GitHub,
	// require("@emotion/is-prop-valid") in a try, which failed the module
	// as it was not found): they throw when they run, as in browsers
	own := map[string]bool{}
	for _, m := range requireCall.FindAllStringSubmatch(src, -1) {
		var spec string
		if json.Unmarshal([]byte(m[1]), &spec) == nil && !importsSpec(src, spec) {
			own[spec] = true
		}
	}
	deps := map[string]string{}
	var urls []string
	for _, m := range requireCall.FindAllStringSubmatch(out, -1) {
		var spec string
		if json.Unmarshal([]byte(m[1]), &spec) != nil || own[spec] {
			continue
		}
		if _, ok := deps[spec]; ok {
			continue
		}
		r, err := e.resolveModule(spec, u)
		if err != nil {
			continue // the error is thrown when it is required
		}
		deps[spec] = r
		urls = append(urls, r)
	}
	dj, _ := json.Marshal(deps)
	// on one line before the code, so that its lines are those of errors
	script := "__defineModule(" + quoteJS(u) + ", " + string(dj) + ", function (exports, require, module) { \"use strict\"; " + out + "\n});"
	return script, urls, nil
}
