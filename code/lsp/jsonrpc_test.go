// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCallCancel checks that a request that times out is canceled
// with the server.
func TestCallCancel(t *testing.T) {
	cr, sw := io.Pipe() // server -> client
	sr, cw := io.Pipe() // client -> server
	defer sw.Close()
	defer cw.Close()
	c := NewConn(cr, cw)
	go c.Serve(context.Background())
	srv := NewConn(sr, sw) // only used to read raw messages
	msgs := make(chan *rpcMsg, 4)
	go func() {
		for {
			msg, err := srv.readMsg()
			if err != nil {
				return
			}
			msgs <- msg
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := c.Call(ctx, "slow", nil, nil)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	req := <-msgs
	assert.Equal(t, "slow", req.Method)
	select {
	case msg := <-msgs:
		assert.Equal(t, "$/cancelRequest", msg.Method)
		var p struct {
			ID json.RawMessage `json:"id"`
		}
		assert.NoError(t, json.Unmarshal(msg.Params, &p))
		assert.Equal(t, string(req.ID), string(p.ID))
	case <-time.After(time.Second):
		t.Fatal("no $/cancelRequest sent")
	}
}

// TestProgress checks the handling of $/progress notifications,
// which are reported as the message of the most recent operation.
func TestProgress(t *testing.T) {
	b := NewBackend(nil)
	defer b.Stop()
	var msgs []string
	b.Progress = func(msg string) { msgs = append(msgs, msg) }
	srv := NewServer(b.Reg, &ServerConfig{ID: "gopls"}, t.TempDir())
	srv.Progress = func(token, title, msg string, done bool) {
		b.handleProgress("gopls", token, title, msg, done)
	}
	notify := func(token, value string) {
		srv.Notify("$/progress", json.RawMessage(`{"token":`+token+`,"value":`+value+`}`))
	}
	notify(`"a"`, `{"kind":"begin","title":"Loading packages"}`)
	notify(`1`, `{"kind":"begin","title":"Indexing","percentage":10}`)
	notify(`"a"`, `{"kind":"report","message":"3/10"}`)
	notify(`"a"`, `{"kind":"end"}`)
	notify(`1`, `{"kind":"end"}`)
	assert.Equal(t, []string{
		"gopls: Loading packages",
		"gopls: Indexing: 10%",
		"gopls: Loading packages: 3/10",
		"gopls: Indexing: 10%",
		"",
	}, msgs)
}
