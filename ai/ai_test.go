// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// texts returns the text of all of the text widgets in the given body.
func texts(b *core.Body) string {
	var s strings.Builder
	for w := range b.AllWidgets() {
		if tx, ok := w.(*core.Text); ok {
			s.WriteString(tx.Text + "\n")
		}
	}
	return s.String()
}

// TestChat tests asking a question with a suggestion, and showing the
// answer as it is streamed from the server, and the errors of the server.
func TestChat(t *testing.T) {
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		for _, tok := range []string{"Call ", "it ", "with ()."} {
			w.Write([]byte(`data: {"choices":[{"delta":{"content":"` + tok + `"}}]}` + "\n"))
		}
		w.Write([]byte("data: {\ndata: [DONE]\n"))
	}))
	defer srv.Close()
	old := ollamaURL
	ollamaURL = srv.URL
	defer func() { ollamaURL = old }()
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("http_proxy", "")
	t.Cleanup(core.TrackWindows())

	b := core.NewBody()
	makeApp(b)
	waitFor := func(s string) {
		t.Helper()
		require.Eventually(t, func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return strings.Contains(texts(b), s)
		}, 10*time.Second, 10*time.Millisecond, s)
	}
	core.RenderImage(b, core.Locked(b, func() {
		send := core.ButtonByText(b, "How do you call a function in Go?")
		require.NotNil(t, send)
		send.Send(events.Click)
		assert.Contains(t, texts(b), "How do you call a function in Go?", "the prompt")
	}), func() {
		waitFor("Call it with ().")
	}, core.Locked(b, func() {
		send := sendButton(b)
		send.Send(events.Click) // nothing to send
		assert.Contains(t, popupTexts(b), "Please enter a prompt")
		var tf *core.TextField
		for w := range b.AllWidgets() {
			if x, ok := w.(*core.TextField); ok {
				tf = x
			}
		}
		require.NotNil(t, tf)
		tf.SetText("Again")
		status = http.StatusInternalServerError
		send.Send(events.Click)
	}), func() {
		require.Eventually(t, func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return strings.Contains(popupTexts(b), "Error getting response (500")
		}, 10*time.Second, 10*time.Millisecond)
	})
}

// sendButton returns the button that sends the prompt.
func sendButton(b *core.Body) *core.Button {
	for w := range b.AllWidgets() {
		if bt, ok := w.(*core.Button); ok && bt.Icon == icons.Send {
			return bt
		}
	}
	return nil
}

// popupTexts returns the text of the text widgets in the popups (as the
// snackbars) of the given body.
func popupTexts(b *core.Body) string {
	var s strings.Builder
	for _, sc := range core.Popups(b) {
		for w := range sc.AllWidgets() {
			if tx, ok := w.(*core.Text); ok {
				s.WriteString(tx.Text + "\n")
			}
		}
	}
	return s.String()
}
