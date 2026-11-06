// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"cogentcore.org/core/htmlcore"
)

// cookies are the cookies of the pages of the app, shared by its pages:
// they are sent with the requests of pages and their resources, and are
// those of the scripts of pages (document.cookie and fetch).
var cookies, _ = cookiejar.New(nil)

// httpClient is the client of the requests of pages and their resources,
// with the [cookies], and a timeout so that a server that does not respond
// does not stop the reading of a page.
var httpClient = &http.Client{Jar: cookies, Timeout: 30 * time.Second}

// postRequest is the body of a POST request that opens a page (as for the
// submission of a form), and its content type.
type postRequest struct {
	contentType string
	body        []byte
}

// withCookies returns a function that gets URLs as the given one does,
// except for http and https URLs, which it gets with the [cookies].
func withCookies(get func(url string) (*http.Response, error)) func(url string) (*http.Response, error) {
	return func(url string) (*http.Response, error) {
		if !strings.HasPrefix(url, "http:") && !strings.HasPrefix(url, "https:") {
			return get(url)
		}
		return doRequest(http.MethodGet, url, nil)
	}
}

// doRequest makes the given request with the [httpClient], with the
// [htmlcore.UserAgent], and with the given body, if not nil.
func doRequest(method, url string, post *postRequest) (*http.Response, error) {
	var body *bytes.Reader
	if post != nil {
		body = bytes.NewReader(post.body)
	} else {
		body = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", htmlcore.UserAgent)
	if post != nil {
		req.Header.Set("Content-Type", post.contentType)
	}
	return httpClient.Do(req)
}

// postPage opens the page of the given URL with the given POST request,
// returning an error (and closing the body) if its status is not OK, as
// [htmlcore.Get] does.
func postPage(url string, post *postRequest) (*http.Response, error) {
	resp, err := doRequest(http.MethodPost, url, post)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return resp, fmt.Errorf("got error status %q (code %d)", resp.Status, resp.StatusCode)
	}
	return resp, nil
}
