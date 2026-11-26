// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package contacts

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain keeps the settings of the tests in a temporary directory,
// not those of the user.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "contacts-test")
	if err != nil {
		panic(err)
	}
	Settings.File = filepath.Join(dir, "settings.toml")
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
