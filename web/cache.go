// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"bytes"
	"io"
	"net/http"
	neturl "net/url"
	"sync"
)

// resourceCache caches the resources of a page (its HTML, style sheets and
// images) by URL, so that the page can be read again without getting them
// again, as for another width of its viewport (see [Page.SizeFinal]).
// Only successful responses are cached.
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
}

// Get returns the response for the given URL, from the cache if it is in it.
func (rc *resourceCache) Get(url string) (*http.Response, error) {
	rc.mu.Lock()
	r := rc.items[url]
	rc.mu.Unlock()
	if r == nil {
		resp, err := rc.get(url)
		if err != nil || resp.StatusCode != http.StatusOK {
			return resp, err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
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
	resp := &http.Response{StatusCode: r.status, Status: http.StatusText(r.status), Header: r.header,
		Body: io.NopCloser(bytes.NewReader(r.body))}
	if r.url != nil {
		resp.Request = &http.Request{URL: r.url}
	}
	return resp, nil
}
