// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"io"
	"net/http"
	"strings"
	"sync"

	"cogentcore.org/core/htmlcore"
)

// prefetchWorkers is the number of resources of a page that are gotten at
// once by [prefetch].
const prefetchWorkers = 24

// prefetch gets the given resources of a page (see
// [htmlcore.Context.PrefetchURLs]) in parallel with the given function of
// its cache, in which they are as the page is read, and the web fonts and
// imports of the style sheets that it gets (see [htmlcore.StyleSheetURLs]),
// until the page is canceled: as the preload scanners of browsers, as
// reading a page gets its resources one at a time (the first read of
// volkskrant.nl got 255 of them in 8s, each in 20-100ms, before it was
// shown). The resources that are being gotten as the page is read are
// waited for, not gotten again (see [resourceCache.Get]).
//
// The images that it gets are decoded into the given cache (if not nil),
// in parallel, as they would be one at a time as the page is read.
func prefetch(urls []string, get func(url string) (*http.Response, error), webFonts bool, images *htmlcore.ImageCache, canceled func() bool) {
	if len(urls) == 0 {
		return
	}
	var mu sync.Mutex
	queue := append([]string{}, urls...)
	seen := map[string]bool{}
	for _, u := range urls {
		seen[u] = true
	}
	next := func() (string, bool) {
		mu.Lock()
		defer mu.Unlock()
		if len(queue) == 0 || canceled() {
			return "", false
		}
		u := queue[0]
		queue = queue[1:]
		return u, true
	}
	add := func(us []string) {
		mu.Lock()
		defer mu.Unlock()
		for _, u := range us {
			if !seen[u] {
				seen[u] = true
				queue = append(queue, u)
			}
		}
	}
	var wg sync.WaitGroup
	var worker func()
	worker = func() {
		defer wg.Done()
		for {
			u, ok := next()
			if !ok {
				return
			}
			resp, err := get(u)
			if err != nil {
				continue
			}
			ct := strings.ToLower(resp.Header.Get("Content-Type"))
			if images != nil && strings.HasPrefix(ct, "image/") && !strings.Contains(ct, "svg") {
				if b, err := io.ReadAll(resp.Body); err == nil {
					images.Decode(u, b)
				}
			}
			if strings.Contains(ct, "css") {
				// its imports and web fonts, as it is added to the page
				b, err := io.ReadAll(resp.Body)
				if err == nil {
					more := htmlcore.StyleSheetURLs(string(b), u, webFonts)
					add(more)
					// (more workers, if the others have stopped)
					for range min(len(more), prefetchWorkers) {
						wg.Add(1)
						go worker()
					}
				}
			}
			resp.Body.Close()
		}
	}
	for range min(len(urls), prefetchWorkers) {
		wg.Add(1)
		go worker()
	}
}
