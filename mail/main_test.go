// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"os"
	"testing"

	"github.com/zalando/go-keyring"
)

// TestMain uses a keychain in memory instead of the real one, for the
// tokens of signing in.
func TestMain(m *testing.M) {
	keyring.MockInit()
	syncUpdateInterval = 0 // updated right away, for checking the lists after syncs
	os.Exit(m.Run())
}
