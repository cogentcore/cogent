// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/cogent/mail/contacts"
	"github.com/zalando/go-keyring"
)

// TestMain uses a keychain in memory instead of the real one, for the
// tokens of signing in, and a temporary file for the contacts settings.
func TestMain(m *testing.M) {
	keyring.MockInit()
	syncUpdateInterval = 0 // updated right away, for checking the lists after syncs
	// not the real settings of the contacts, as My card
	dir, err := os.MkdirTemp("", "mail-test")
	if err != nil {
		panic(err)
	}
	contacts.Settings.File = filepath.Join(dir, "contacts-settings.toml")
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
