// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
)

// debugLogFile is the name of the file of the debug log in the directory
// of the data of the app (see [App.StartDebugLog]).
const debugLogFile = "debug.log"

// StartDebugLog starts writing the log of the app, with its debug
// messages, to the debug log file in the directory of its data, after any
// earlier log, as well as the regular log at its usual level, for
// finding problems such as with signing in. It never logs the tokens.
// It must be called after [App.StartTesting], if used.
func (a *App) StartDebugLog() error {
	dir := a.dataDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	fn := filepath.Join(dir, debugLogFile)
	f, err := os.OpenFile(fn, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	file := slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug})
	// Not the default handler of slog, which writes with the log package,
	// whose output SetDefault sends back to the new handler: a deadlock.
	stderr := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(&teeHandler{file, stderr}))
	slog.Info("mail: writing the debug log", "file", fn)
	return nil
}

// teeHandler is a [slog.Handler] that sends the records to two handlers,
// each at its own level.
type teeHandler struct {
	a, b slog.Handler
}

func (h *teeHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.a.Enabled(ctx, l) || h.b.Enabled(ctx, l)
}

func (h *teeHandler) Handle(ctx context.Context, r slog.Record) error {
	var err error
	if h.a.Enabled(ctx, r.Level) {
		err = h.a.Handle(ctx, r.Clone())
	}
	if h.b.Enabled(ctx, r.Level) {
		if e := h.b.Handle(ctx, r.Clone()); err == nil {
			err = e
		}
	}
	return err
}

func (h *teeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &teeHandler{h.a.WithAttrs(attrs), h.b.WithAttrs(attrs)}
}

func (h *teeHandler) WithGroup(name string) slog.Handler {
	return &teeHandler{h.a.WithGroup(name), h.b.WithGroup(name)}
}
