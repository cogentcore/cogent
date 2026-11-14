// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPrefetch tests that the resources of a page are gotten in parallel
// into its cache (with the web fonts of its style sheets), each once, and
// that one that is being gotten when the page asks for it is waited for,
// not gotten again.
func TestPrefetch(t *testing.T) {
	var mu sync.Mutex
	gets := map[string]int{}
	running, most := 0, 0
	rc := &resourceCache{get: func(u string) (*http.Response, error) {
		mu.Lock()
		gets[u]++
		running++
		most = max(most, running)
		mu.Unlock()
		time.Sleep(50 * time.Millisecond) // (as the latency of the network)
		mu.Lock()
		running--
		mu.Unlock()
		body, typ := "data", "image/png"
		if strings.HasSuffix(u, ".css") {
			body, typ = `@font-face { font-family: F; src: url(font.woff2) format("woff2") }`, "text/css"
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {typ}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	}}
	urls := []string{"https://example.com/css/site.css"}
	for i := range 20 {
		urls = append(urls, fmt.Sprintf("https://example.com/img%d.png", i))
	}
	start := time.Now()
	prefetch(urls, rc.Get, true, func() bool { return false })
	// (as the page is read, while it is being gotten)
	resp, err := rc.Get("https://example.com/img19.png")
	require.NoError(t, err)
	resp.Body.Close()
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return gets["https://example.com/css/font.woff2"] == 1 && running == 0 && len(gets) == 22
	}, 3*time.Second, 10*time.Millisecond, "the web font of the style sheet")
	mu.Lock()
	defer mu.Unlock()
	for u, n := range gets {
		assert.Equal(t, 1, n, "gotten once: %s", u)
	}
	assert.Greater(t, most, 10, "in parallel")
	assert.Less(t, time.Since(start), 500*time.Millisecond, "not one at a time (22 x 50ms)")
}
