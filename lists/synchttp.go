// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !goscript

package lists

import (
	"net/http"
	"time"
)

// syncHTTP is the HTTP client for syncing, with a timeout,
// so that a sync does not wait forever for a lost connection.
var syncHTTP = &http.Client{Timeout: 30 * time.Second}
