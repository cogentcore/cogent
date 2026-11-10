// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// resourceCache caches the resources of a page (its HTML, style sheets and
// images) by URL, so that the page can be read again without getting them
// again, as for another width of its viewport (see [Page.SizeFinal]).
// Responses of error statuses, and errors, are cached too, so that they
// are not gotten again for each read of the page.
type resourceCache struct {
	// get gets the resources that are not cached yet.
	get func(url string) (*http.Response, error)

	mu    sync.Mutex
	items map[string]*cachedResource
}

// cachedResource is a resource in a [resourceCache].
type cachedResource struct {
	status int
	header http.Header
	url    *neturl.URL // of the request, after redirects
	body   []byte
	err    error // the error getting it, if any
}

// Get returns the response for the given URL, from the cache if it is in it.
func (rc *resourceCache) Get(url string) (*http.Response, error) {
	rc.mu.Lock()
	r := rc.items[url]
	rc.mu.Unlock()
	if r == nil {
		resp, err := rc.get(url)
		if err != nil {
			if errors.Is(err, errStopped) {
				return nil, err
			}
			// a resource that can not be gotten is not tried again for the
			// page (each of the reads of the page that its scripts make
			// got all of those of nytimes.com again, for seconds)
			rc.mu.Lock()
			if rc.items == nil {
				rc.items = map[string]*cachedResource{}
			}
			rc.items[url] = &cachedResource{err: err}
			rc.mu.Unlock()
			return nil, err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		// (also those of an error status, which are not gotten again)
		r = &cachedResource{status: resp.StatusCode, header: resp.Header, body: body}
		if resp.Request != nil {
			r.url = resp.Request.URL
		}
		rc.mu.Lock()
		if rc.items == nil {
			rc.items = map[string]*cachedResource{}
		}
		rc.items[url] = r
		rc.mu.Unlock()
	}
	if r.err != nil {
		return nil, r.err
	}
	resp := &http.Response{StatusCode: r.status, Status: http.StatusText(r.status), Header: r.header,
		Body: io.NopCloser(bytes.NewReader(r.body))}
	if r.url != nil {
		resp.Request = &http.Request{URL: r.url}
	}
	return resp, nil
}

// save writes the cached resources to the given directory, with an index
// of their URLs (index.tsv: file, URL, status and content type), for the
// page to be read again offline (see [Page.saveDOM]), up to the given
// total size.
func (rc *resourceCache) save(dir string, maxBytes int) (int, error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if len(rc.items) == 0 {
		return 0, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}
	urls := make([]string, 0, len(rc.items))
	for u := range rc.items {
		urls = append(urls, u)
	}
	sort.Strings(urls)
	var index strings.Builder
	n, total := 0, 0
	for _, u := range urls {
		r := rc.items[u]
		if r.err != nil {
			fmt.Fprintf(&index, "-\t%s\terror\t%s\n", u, strings.ReplaceAll(r.err.Error(), "\t", " "))
			continue
		}
		if total+len(r.body) > maxBytes {
			continue
		}
		file := fmt.Sprintf("%04d", n)
		if err := os.WriteFile(filepath.Join(dir, file), r.body, 0o644); err != nil {
			return n, err
		}
		total += len(r.body)
		n++
		fmt.Fprintf(&index, "%s\t%s\t%d\t%s\n", file, u, r.status, r.header.Get("Content-Type"))
	}
	return n, os.WriteFile(filepath.Join(dir, "index.tsv"), []byte(index.String()), 0o644)
}
