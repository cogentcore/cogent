// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
)

// RPCError is an error returned by the other side of an RPC call,
// per the JSON-RPC 2.0 error object spec.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (re *RPCError) Error() string {
	// gopls and others use code 0 for plain user-facing failure
	// messages, where the code number is noise:
	if re.Code == 0 {
		return "lsp: " + re.Message
	}
	return fmt.Sprintf("lsp: rpc error %d: %s", re.Code, re.Message)
}

// ErrServerClosed indicates that the connection to the server was
// closed while a request was pending.
var ErrServerClosed = errors.New("lsp: server connection closed")

// rpcMsg is the generic JSON-RPC 2.0 message envelope, covering
// requests, responses, and notifications.
type rpcMsg struct {
	Version string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// Handler handles incoming requests and notifications from the server.
// For requests, it returns the result to send back (or an error).
// Notifications are handled and discarded.
type Handler interface {

	// Request handles a server-initiated request.
	Request(method string, params json.RawMessage) (result any, err *RPCError)

	// Notify handles a server-initiated notification.
	Notify(method string, params json.RawMessage)
}

// Conn is a JSON-RPC 2.0 connection over a pair of streams,
// using the LSP base protocol framing:
// Content-Length: N\r\n\r\n followed by N bytes of JSON.
//
// The framing is based on the transport used by the sample gopls
// client: golang.org/x/tools/gopls/internal/cmd (BSD license).
type Conn struct {
	rd *bufio.Reader
	wr io.Writer
	wm sync.Mutex

	idSeq int64
	pend  map[int64]chan *rpcMsg
	pmu   sync.Mutex

	handler Handler
	serve   sync.Once
	closed  chan struct{}
	once    sync.Once
}

// NewConn returns a new JSON-RPC connection for reading messages
// from rd and writing them to wr. Call [Serve] in a separate goroutine
// to process incoming messages from the server.
func NewConn(rd io.Reader, wr io.Writer) *Conn {
	return &Conn{
		rd:      bufio.NewReader(rd),
		wr:      wr,
		pend:    make(map[int64]chan *rpcMsg),
		closed:  make(chan struct{}),
		handler: nopHandler{},
	}
}

// SetHandler sets the handler for incoming requests and notifications.
// It must be called before [Serve].
func (c *Conn) SetHandler(h Handler) {
	if h != nil {
		c.handler = h
	}
}

// Closed returns a channel that is closed when the connection
// to the server terminates.
func (c *Conn) Closed() <-chan struct{} { return c.closed }

// close marks the connection closed, releasing all pending calls.
func (c *Conn) close() {
	c.once.Do(func() {
		close(c.closed)
		c.pmu.Lock()
		for _, ch := range c.pend {
			close(ch)
		}
		c.pend = make(map[int64]chan *rpcMsg)
		c.pmu.Unlock()
	})
}

// writeMsg sends one JSON-RPC message with the standard headers.
func (c *Conn) writeMsg(msg *rpcMsg) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	c.wm.Lock()
	defer c.wm.Unlock()
	fmt.Fprintf(c.wr, "Content-Length: %d\r\n\r\n", len(data))
	if _, err := c.wr.Write(data); err != nil {
		return err
	}
	return nil
}

// readMsg reads one JSON-RPC message, returning io.EOF at end of stream.
func (c *Conn) readMsg() (*rpcMsg, error) {
	var contentLen int
	for {
		ln, err := c.rd.ReadString('\n')
		if err != nil {
			return nil, err
		}
		ln = strings.TrimRight(ln, "\r\n")
		if ln == "" {
			break // end of headers
		}
		key, val, found := strings.Cut(ln, ":")
		if !found {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(key), "Content-Length") {
			contentLen, err = strconv.Atoi(strings.TrimSpace(val))
			if err != nil {
				return nil, fmt.Errorf("lsp: bad Content-Length: %q", val)
			}
		}
		// other headers (Content-Type) are ignored, per the spec
	}
	if contentLen <= 0 {
		return nil, fmt.Errorf("lsp: missing or invalid Content-Length")
	}
	body := make([]byte, contentLen)
	if _, err := io.ReadFull(c.rd, body); err != nil {
		return nil, err
	}
	msg := &rpcMsg{}
	if err := json.Unmarshal(body, msg); err != nil {
		return nil, fmt.Errorf("lsp: bad message: %w", err)
	}
	return msg, nil
}

// Serve runs the read loop, dispatching incoming messages until the
// stream ends or the context is canceled. It must be called exactly once,
// in a separate goroutine, after [SetHandler].
func (c *Conn) Serve(ctx context.Context) {
	c.serve.Do(func() {
		defer c.close()
		go func() {
			select {
			case <-ctx.Done():
			case <-c.closed:
			}
			c.close()
		}()
		for {
			msg, err := c.readMsg()
			if err != nil {
				if !errors.Is(err, io.EOF) && ctx.Err() == nil {
					log.Println("lsp:", err)
				}
				return
			}
			c.dispatch(msg)
		}
	})
}

// dispatch routes one incoming message.
func (c *Conn) dispatch(msg *rpcMsg) {
	switch {
	case len(msg.ID) > 0 && msg.Method == "":
		// response to one of our requests
		var id int64
		if err := json.Unmarshal(msg.ID, &id); err != nil {
			return
		}
		c.pmu.Lock()
		ch, has := c.pend[id]
		delete(c.pend, id)
		c.pmu.Unlock()
		if has {
			ch <- msg
			close(ch)
		}

	case len(msg.ID) > 0 && msg.Method != "":
		// server-initiated request: handle and reply
		result, rpcerr := c.handler.Request(msg.Method, msg.Params)
		resp := &rpcMsg{Version: "2.0", ID: msg.ID}
		switch {
		case rpcerr != nil:
			resp.Error = rpcerr
		default:
			data, err := json.Marshal(result)
			if err != nil {
				data = []byte("null")
			}
			resp.Result = data
		}
		if err := c.writeMsg(resp); err != nil {
			log.Println("lsp: reply err:", err)
		}

	default:
		// notification
		c.handler.Notify(msg.Method, msg.Params)
	}
}

// Call makes a request and waits for the response, unmarshaling it
// into result (which may be nil to ignore). It returns an error on
// timeout, context cancellation, connection close, or server error.
func (c *Conn) Call(ctx context.Context, method string, params, result any) error {
	c.pmu.Lock()
	c.idSeq++
	id := c.idSeq
	ch := make(chan *rpcMsg, 1)
	c.pend[id] = ch
	c.pmu.Unlock()

	var rawParams json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return err
		}
		rawParams = data
	}
	msg := &rpcMsg{Version: "2.0", ID: json.RawMessage(strconv.FormatInt(id, 10)), Method: method, Params: rawParams}
	if err := c.writeMsg(msg); err != nil {
		c.pmu.Lock()
		delete(c.pend, id)
		c.pmu.Unlock()
		return err
	}

	select {
	case resp, ok := <-ch:
		if !ok || resp == nil {
			return ErrServerClosed
		}
		if resp.Error != nil {
			return resp.Error
		}
		if result != nil && len(resp.Result) > 0 {
			return json.Unmarshal(resp.Result, result)
		}
		return nil
	case <-c.closed:
		return ErrServerClosed
	case <-ctx.Done():
		c.pmu.Lock()
		delete(c.pend, id)
		c.pmu.Unlock()
		return ctx.Err()
	}
}

// Notify sends a notification (no response expected).
func (c *Conn) Notify(method string, params any) error {
	var rawParams json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return err
		}
		rawParams = data
	}
	return c.writeMsg(&rpcMsg{Version: "2.0", Method: method, Params: rawParams})
}

// nopHandler is the default handler that rejects requests
// and ignores notifications, until a real handler is set.
type nopHandler struct{}

func (nh nopHandler) Request(method string, params json.RawMessage) (any, *RPCError) {
	return nil, &RPCError{Code: -32601, Message: "method not handled: " + method}
}

func (nh nopHandler) Notify(method string, params json.RawMessage) {}
