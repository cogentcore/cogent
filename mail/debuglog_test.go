// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDebugLog tests that the debug log gets the debug messages, and
// that logging does not hang, with slog or the log package.
func TestDebugLog(t *testing.T) {
	old, flags, out := slog.Default(), log.Flags(), log.Writer()
	t.Cleanup(func() {
		slog.SetDefault(old)
		log.SetFlags(flags)
		log.SetOutput(out)
	})
	a := &App{dataDirectory: t.TempDir()}
	require.NoError(t, a.StartDebugLog())
	done := make(chan struct{})
	go func() {
		slog.Debug("signing in", "account", "a@x.test")
		log.Println("from the log package")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("logging hangs")
	}
	data, err := os.ReadFile(filepath.Join(a.dataDirectory, debugLogFile))
	require.NoError(t, err)
	assert.Contains(t, string(data), "signing in")
	assert.Contains(t, string(data), "a@x.test")
	assert.Contains(t, string(data), "from the log package")
}
