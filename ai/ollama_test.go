// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/aandrew-me/tgpt/v2/structs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleToken tests getting the tokens of the lines of a streamed
// chat completion.
func TestHandleToken(t *testing.T) {
	tok, err := HandleToken(`data: {"id":"1","choices":[{"delta":{"content":"Hi"}}]}`)
	require.NoError(t, err)
	assert.Equal(t, "Hi", tok)
	tok, err = HandleToken(`{"choices":[{"delta":{"content":"there"}}]}`)
	require.NoError(t, err)
	assert.Equal(t, "there", tok)
	for _, s := range []string{"data: [DONE]", "", `data: {"choices":[]}`, `data: {"id":"1"}`} {
		tok, err = HandleToken(s)
		assert.NoError(t, err, s)
		assert.Empty(t, tok, s)
	}
	_, err = HandleToken("data: {")
	assert.Error(t, err)
}

// TestNewRequest tests the request to the local server, with the
// parameters and the previous messages.
func TestNewRequest(t *testing.T) {
	var got map[string]any
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		assert.NoError(t, json.Unmarshal(b, &got), string(b))
		auth = r.Header.Get("Authorization")
		w.Write([]byte("data: [DONE]\n"))
	}))
	defer srv.Close()
	old := ollamaURL
	ollamaURL = srv.URL
	defer func() { ollamaURL = old }()
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("http_proxy", "")
	t.Chdir(t.TempDir()) // no proxy.txt

	resp, err := NewRequest(`say "hi"`, structs.Params{ApiKey: "key"}, `{"content": "before", "role": "user"},`)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, "Bearer key", auth)
	assert.Equal(t, "mistral", got["model"])
	assert.Equal(t, 0.5, got["temperature"])
	msgs := got["messages"].([]any)
	require.Len(t, msgs, 2)
	assert.Equal(t, `say "hi"`, msgs[1].(map[string]any)["content"])

	resp, err = NewRequest("x", structs.Params{ApiModel: "gemma", Temperature: "0.1", Top_p: "0.9"}, "")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, "gemma", got["model"])
	assert.Equal(t, 0.9, got["top_p"])

	// proxies
	require.NoError(t, os.WriteFile(filepath.Join(".", "proxy.txt"), []byte("http://127.0.0.1:1\n"), 0o644))
	_, err = NewClient()
	assert.NoError(t, err)
	t.Setenv("HTTP_PROXY", "socks5://127.0.0.1:1")
	_, err = NewClient()
	assert.NoError(t, err)
}
