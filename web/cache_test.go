// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResourceCacheFailures tests that the responses of error statuses and
// the errors of the resources of a page are cached, so that they are not
// gotten again for each read of the page (as those of nytimes.com, which
// its scripts read again many times), except a stop, and that the cache
// is saved with its index.
func TestResourceCacheFailures(t *testing.T) {
	gets := map[string]int{}
	rc := &resourceCache{get: func(u string) (*http.Response, error) {
		gets[u]++
		switch u {
		case "https://x/missing":
			return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("gone")), Header: http.Header{}}, nil
		case "https://x/down":
			return nil, errors.New("connection refused")
		case "https://x/stopped":
			return nil, errStopped
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok")), Header: http.Header{"Content-Type": {"text/css"}}}, nil
	}}
	for range 3 {
		resp, err := rc.Get("https://x/missing")
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode)
		_, err = rc.Get("https://x/down")
		assert.Error(t, err)
		rc.Get("https://x/stopped")
		rc.Get("https://x/a.css")
	}
	assert.Equal(t, map[string]int{"https://x/missing": 1, "https://x/down": 1, "https://x/stopped": 3, "https://x/a.css": 1}, gets)

	dir := t.TempDir()
	n, err := rc.save(dir, 1<<20)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	index, err := os.ReadFile(filepath.Join(dir, "index.tsv"))
	require.NoError(t, err)
	assert.Contains(t, string(index), "\thttps://x/a.css\t200\ttext/css")
	assert.Contains(t, string(index), "-\thttps://x/down\terror\tconnection refused")
}
