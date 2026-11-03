// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build goscript

package lists

import "net/http"

// syncHTTP is the HTTP client for syncing, which is the default one in
// JavaScript, where it has no timeout: the browser has its own.
var syncHTTP *http.Client
