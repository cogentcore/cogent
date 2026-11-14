// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cogentcore.org/cogent/code/lang"
	"cogentcore.org/core/text/lines"
)

// menuSymbolsWait is how long the menu search waits for the symbols of a
// package that it does not have yet, before it is shown without them.
const menuSymbolsWait = 250 * time.Millisecond

// symbolCache has the symbols of packages for the menu search, which
// gets them on the UI thread as it is opened, so that it never waits for
// the language server, which can take seconds for a package that it has
// not loaded yet, or while it loads the workspace again. The cached
// symbols are used as they are, and are fetched again in the background
// for the next search.
type symbolCache struct {
	mu   sync.Mutex
	pkgs map[string]*cachedSymbols
}

// cachedSymbols are the symbols of a package in a [symbolCache].
type cachedSymbols struct {
	syms []lang.Symbol

	// have is whether syms have been fetched.
	have bool

	// loading is closed when the fetch in progress is done, if any.
	loading chan struct{}
}

// get returns the symbols with the given key, fetched by the given
// function. It returns any cached symbols immediately, and fetches them
// again in the background (once at a time). Otherwise, it waits for them
// for at most the given time, and returns nil if they are not there yet.
func (sc *symbolCache) get(key string, wait time.Duration, fetch func() []lang.Symbol) []lang.Symbol {
	sc.mu.Lock()
	if sc.pkgs == nil {
		sc.pkgs = make(map[string]*cachedSymbols)
	}
	cs := sc.pkgs[key]
	if cs == nil {
		cs = &cachedSymbols{}
		sc.pkgs[key] = cs
	}
	syms, have, done := cs.syms, cs.have, cs.loading
	if done == nil {
		done = make(chan struct{})
		cs.loading = done
		go func() {
			syms := fetch()
			sc.mu.Lock()
			cs.syms, cs.have, cs.loading = syms, true, nil
			sc.mu.Unlock()
			close(done)
		}()
	}
	sc.mu.Unlock()
	if have {
		return syms
	}
	select {
	case <-done:
	case <-time.After(wait):
		return nil
	}
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return cs.syms
}

// menuSymbols returns the symbols of the package of the given buffer from
// its language server, for the menu search (see [symbolCache]).
func (ls *LangService) menuSymbols(ln *lines.Lines) []lang.Symbol {
	fname := ln.Filename()
	key := filepath.Dir(fname) + "|" + filepath.Ext(fname)
	if strings.HasSuffix(fname, "_test.go") { // (they include the tests)
		key += "|test"
	}
	return ls.menuSyms.get(key, menuSymbolsWait, func() []lang.Symbol {
		return ls.lspB.PackageSymbols(ln)
	})
}
