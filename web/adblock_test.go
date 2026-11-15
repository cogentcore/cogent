// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andybalholm/cascadia"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdBlock tests that the requests to the hosts of ads and tracking
// (and their subdomains) are blocked if BlockAds, and others are not.
func TestAdBlock(t *testing.T) {
	for host, blocked := range map[string]bool{
		"securepubads.g.doubleclick.net": true, "pagead2.googlesyndication.com": true,
		"www.googletagmanager.com": true, "c.amazon-adsystem.com": true, "warp.media.net": true,
		"DoubleClick.NET": true, "www.nytimes.com": false, "static01.nyt.com": false,
		"notdoubleclick.net": false, "www.google.com": false, "geo.captcha-delivery.com": false,
	} {
		assert.Equal(t, blocked, adHost(host), host)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
	defer srv.Close()
	old := Settings.BlockAds
	defer func() { Settings.BlockAds = old }()
	Settings.BlockAds = true
	b := adBlocker{}
	req, _ := http.NewRequest("GET", "https://securepubads.g.doubleclick.net/tag/js/gpt.js", nil)
	_, err := b.RoundTrip(req)
	assert.ErrorIs(t, err, errBlocked)
	req, _ = http.NewRequest("GET", srv.URL, nil)
	resp, err := b.RoundTrip(req)
	require.NoError(t, err)
	resp.Body.Close()
	Settings.BlockAds = false
	req, _ = http.NewRequest("GET", "https://securepubads.g.doubleclick.net/tag/js/gpt.js", nil)
	req.URL.Host = srv.Listener.Addr().String() // (not blocked: to the test server)
	req.URL.Scheme = "http"
	resp, err = b.RoundTrip(req)
	require.NoError(t, err)
	resp.Body.Close()
}

// TestAdStyles tests that the selectors of the styles that hide the slots
// of ads parse (a selector that does not drops its rule, as :has(> x),
// with the others).
func TestAdStyles(t *testing.T) {
	for rule := range strings.SplitSeq(adStyles, "}") {
		sel, _, ok := strings.Cut(rule, "{")
		if !ok {
			continue
		}
		_, err := cascadia.ParseGroup(strings.TrimSpace(sel))
		assert.NoError(t, err, sel)
	}
}
