// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"encoding/json"
	"strings"

	"golang.org/x/net/html"
)

// importMap is the import map of a page (of its script elements of type
// importmap), which maps the specifiers of modules (as "react") to URLs.
type importMap struct {
	Imports map[string]string            `json:"imports"`
	Scopes  map[string]map[string]string `json:"scopes"`
}

// readImportMaps adds the import maps of the document to that of the
// engine, resolving their URLs relative to the base URL of the page.
func (e *Engine) readImportMaps() {
	base := e.baseURL()
	walk(e.doc, func(n *html.Node) bool {
		if n.Type != html.ElementNode || n.Data != "script" {
			return true
		}
		if typ, _ := attr(n, "type"); !strings.EqualFold(strings.TrimSpace(typ), "importmap") {
			return true
		}
		var m importMap
		if err := json.Unmarshal([]byte(textContent(n)), &m); err != nil {
			return true
		}
		if e.imports == nil {
			e.imports = &importMap{Imports: map[string]string{}, Scopes: map[string]map[string]string{}}
		}
		resolve := func(dst, src map[string]string) {
			for k, v := range src {
				if u, err := resolveURL(v, base); err == nil {
					dst[k] = u
				}
			}
		}
		resolve(e.imports.Imports, m.Imports)
		for scope, imports := range m.Scopes {
			s, err := resolveURL(scope, base)
			if err != nil {
				continue
			}
			if e.imports.Scopes[s] == nil {
				e.imports.Scopes[s] = map[string]string{}
			}
			resolve(e.imports.Scopes[s], imports)
		}
		return true
	})
}

// resolveModule returns the URL of the module of the given specifier that
// is imported by the module at the given URL: mapped by the import map of
// the page (the longest matching scope of the importer first, then the
// imports), or relative to the importer.
func (e *Engine) resolveModule(spec, importer string) (string, error) {
	if m := e.imports; m != nil {
		scope := ""
		for s := range m.Scopes {
			if strings.HasPrefix(importer, s) && len(s) > len(scope) {
				scope = s
			}
		}
		if scope != "" {
			if u, ok := mapSpecifier(m.Scopes[scope], spec); ok {
				return u, nil
			}
		}
		if u, ok := mapSpecifier(m.Imports, spec); ok {
			return u, nil
		}
	}
	return resolveURL(spec, importer)
}

// mapSpecifier maps the given specifier with the given imports: an exact
// match, or the longest prefix that ends with a slash.
func mapSpecifier(imports map[string]string, spec string) (string, bool) {
	if u, ok := imports[spec]; ok {
		return u, true
	}
	best := ""
	for k := range imports {
		if strings.HasSuffix(k, "/") && strings.HasPrefix(spec, k) && len(k) > len(best) {
			best = k
		}
	}
	if best == "" {
		return "", false
	}
	return imports[best] + spec[len(best):], true
}
