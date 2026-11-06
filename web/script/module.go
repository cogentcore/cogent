// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// unsupported are the features of JavaScript that goja does not support,
// which esbuild rewrites in terms of others.
var unsupported = map[string]bool{
	"async-generator": false,
	"for-await":       false,
	"top-level-await": false,
	"import-meta":     false,
}

// bundleModule returns the given module script (with the given name, its
// URL if external) as a classic script that goja can run: esbuild bundles
// it with the modules that it imports (gotten with GetURL, relative to the
// module that imports them), as a function that is called at once.
func (e *Engine) bundleModule(code, name string) (string, error) {
	return e.bundle(code, name, "")
}

// bundle bundles the given module code (with the given name or URL), as
// a script that sets the given global variable to its exports, if not "".
func (e *Engine) bundle(code, name, global string) (string, error) {
	base := name
	if !strings.Contains(base, "://") {
		base = e.baseURL()
	}
	plugin := api.Plugin{
		Name: "url",
		Setup: func(b api.PluginBuild) {
			b.OnResolve(api.OnResolveOptions{Filter: ".*"}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				importer := args.Importer
				if importer == "" || importer == "<stdin>" || !strings.Contains(importer, "://") {
					importer = base
				}
				u, err := e.resolveModule(args.Path, importer)
				if err != nil {
					return api.OnResolveResult{}, err
				}
				return api.OnResolveResult{Path: u, Namespace: "url"}, nil
			})
			b.OnLoad(api.OnLoadOptions{Filter: ".*", Namespace: "url"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				path := strings.ToLower(strings.Split(args.Path, "?")[0])
				if strings.HasSuffix(path, ".css") {
					empty := ""
					return api.OnLoadResult{Contents: &empty, Loader: api.LoaderEmpty}, nil
				}
				src, err := e.getScript(args.Path)
				if err != nil {
					return api.OnLoadResult{}, fmt.Errorf("getting %s: %w", args.Path, err)
				}
				loader := api.LoaderJS
				if strings.HasSuffix(path, ".json") {
					loader = api.LoaderJSON
				} else {
					src = rewriteImports(src, args.Path)
					src = importMetaURL.ReplaceAllLiteralString(src, quoteJS(args.Path))
				}
				return api.OnLoadResult{Contents: &src, Loader: loader}, nil
			})
		},
	}
	res := api.Build(api.BuildOptions{
		// the dynamic imports are done at run time (by the prelude), as esbuild
		// would match those of computed paths with local files
		Stdin:      &api.StdinOptions{Contents: rewriteImports(code, base), Sourcefile: name, Loader: api.LoaderJS},
		Bundle:     true,
		Format:     api.FormatIIFE,
		Platform:   api.PlatformBrowser,
		Target:     api.ESNext,
		Supported:  unsupported,
		Plugins:    []api.Plugin{plugin},
		GlobalName: global,
		// the URL of the module (as webpack finds the URLs of its
		// chunks from it), which is the URL of the page for a module
		// in the page
		Define:   map[string]string{"import.meta.url": quoteJS(base)},
		Write:    false,
		LogLevel: api.LogLevelSilent,
	})
	if len(res.Errors) > 0 {
		return "", buildError(res.Errors)
	}
	if len(res.OutputFiles) == 0 {
		return "", errors.New("no output")
	}
	return string(res.OutputFiles[0].Contents), nil
}

// moduleSyntax matches the import and export statements of modules.
var moduleSyntax = regexp.MustCompile(`(?m)^\s*(import\s*[\w{*'"]|export\s+(default|const|let|var|function|class|async|\{|\*))`)

// dynamicImport matches the dynamic imports of scripts (import(...)),
// which goja does not support, but not import.meta or method calls.
var dynamicImport = regexp.MustCompile(`(^|[^.\w$])import\s*\(`)

// importMetaURL matches import.meta.url, which is the URL of each module.
var importMetaURL = regexp.MustCompile(`\bimport\.meta\.url\b`)

// rewriteImports rewrites the dynamic imports of the given script as
// calls of __dynamicImport (of the prelude), which bundles the module,
// with the URL of the script that they are relative to (the base URL of
// the document if "").
func rewriteImports(src, base string) string {
	if !strings.Contains(src, "import") {
		return src
	}
	if !strings.Contains(base, "://") {
		base = ""
	}
	call := "__dynamicImport(" + quoteJS(base) + ", "
	var sb strings.Builder
	last := 0
	for _, m := range dynamicImport.FindAllStringSubmatchIndex(src, -1) {
		open := m[1] - 1 // the (
		if isMethod(src, open) {
			continue // a method named import, as import(e) { ... }
		}
		sb.WriteString(src[last:m[3]])
		sb.WriteString(call)
		last = m[1]
	}
	if last == 0 {
		return src
	}
	sb.WriteString(src[last:])
	return sb.String()
}

// isMethod returns whether the parentheses that start at the given index
// of the given script are the parameters of a method, followed by its body
// ({), and not the arguments of a call.
func isMethod(src string, open int) bool {
	depth := 0
	var quote byte
	for i := open; i < len(src); i++ {
		c := src[i]
		if quote != 0 {
			if c == '\\' {
				i++
			} else if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '"', '\'', '`':
			quote = c
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				rest := strings.TrimLeft(src[i+1:], " \t\r\n")
				return strings.HasPrefix(rest, "{")
			}
		}
	}
	return false
}

// bundleURL returns the module at the given URL, bundled with the modules
// that it imports as a script that sets the given global variable to the
// exports of the module (for a dynamic import).
func (e *Engine) bundleURL(u, global string) (string, error) {
	entry := "export * from " + quoteJS(u) + "; export { default } from " + quoteJS(u) + ";"
	code, err := e.bundle(entry, u, global)
	if err != nil {
		// a module without a default export
		code, err = e.bundle("export * from "+quoteJS(u)+";", u, global)
	}
	return code, err
}

// quoteJS returns the given string as a JavaScript string literal.
func quoteJS(s string) string {
	return "'" + strings.NewReplacer("\\", "\\\\", "'", "\\'", "\n", "\\n").Replace(s) + "'"
}

// lower returns the given classic script with the features of JavaScript
// that goja does not support rewritten (as when it does not compile).
func lower(code, name string) (string, error) {
	res := api.Transform(code, api.TransformOptions{
		Sourcefile: name,
		Loader:     api.LoaderJS,
		Target:     api.ESNext,
		Supported:  unsupported,
		LogLevel:   api.LogLevelSilent,
	})
	if len(res.Errors) > 0 {
		return "", buildError(res.Errors)
	}
	return string(res.Code), nil
}

// buildError returns an error of the given messages of esbuild.
func buildError(msgs []api.Message) error {
	var parts []string
	for i, m := range msgs {
		if i >= 3 {
			break
		}
		s := m.Text
		if m.Location != nil {
			s = fmt.Sprintf("%s:%d:%d: %s", m.Location.File, m.Location.Line, m.Location.Column, m.Text)
		}
		parts = append(parts, s)
	}
	return errors.New(strings.Join(parts, "; "))
}
