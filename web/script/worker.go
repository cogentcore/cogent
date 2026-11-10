// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/dop251/goja"
	"golang.org/x/net/html"
)

// Web workers: a worker (new Worker(url)) is an engine of its own with an
// empty document, whose global scope is that of a worker (without window
// and document, and with importScripts, and postMessage and close for its
// owner), which runs the script of the worker. It and the scripts that
// made it post messages to each other, copied as with the structured clone
// of browsers (see cloneEncode in the prelude). Workers are used as for
// the work of the captchas of bot checks (as reCAPTCHA, whose checkbox did
// nothing without them).

// newWorker makes the worker of the given number (of the Worker object of
// the scripts) for the script at the given URL, which is gotten by this
// engine (as it can be a blob: URL of it), and run in the worker. It is
// called in a task.
func (e *Engine) newWorker(id int, src string) {
	u, err := resolveURL(src, e.baseURL())
	if err != nil {
		e.callPrelude("workerError", id, err.Error())
		return
	}
	doc, _ := html.Parse(strings.NewReader("<html><head></head><body></body></html>"))
	opts := Options{
		PageURL:     u,
		GetURL:      e.opts.GetURL,
		Storage:     e.opts.Storage,
		Cookies:     e.opts.Cookies,
		TaskTimeout: e.opts.TaskTimeout,
		Trace:       e.opts.Trace,
		Log: func(level slog.Level, msg string) {
			e.logf(level, "[worker %s] %s", workerName(u), msg)
		},
		worker: true,
	}
	w, err := New(doc, opts)
	if err != nil {
		e.callPrelude("workerError", id, err.Error())
		return
	}
	w.owner, w.ownerID = e, id
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		w.Close()
		return
	}
	if e.workers == nil {
		e.workers = map[int]*Engine{}
	}
	e.workers[id] = w
	e.mu.Unlock()
	e.logf(slog.LevelDebug, "worker %s started", workerName(u))
	go func() {
		code, err := e.getScript(u) // (outside of a task, as it can take time)
		if err != nil {
			e.logf(slog.LevelWarn, "worker %s: %v", u, err)
			e.Post(func() { e.callPrelude("workerError", id, err.Error()) })
			return
		}
		w.Post(func() {
			w.run(u, code)
			w.mu.Lock()
			w.workerReady = true
			pending := w.workerPending
			w.workerPending = nil
			w.mu.Unlock()
			for _, m := range pending {
				w.receive(m)
			}
		})
	}()
}

// receive dispatches the given message (as JSON) from the owner of the
// worker to it, once its script has run.
func (w *Engine) receive(m workerMessage) {
	w.mu.Lock()
	if !w.workerReady {
		w.workerPending = append(w.workerPending, m)
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()
	w.Post(func() { w.callPrelude("message", m.data, "", -1, m.ports) })
}

// workerName returns the name of a worker of the given URL for the
// console: its URL without its query, or "blob" for a blob: URL.
func workerName(u string) string {
	if strings.HasPrefix(u, "blob:") {
		return "blob"
	}
	u, _, _ = strings.Cut(u, "?")
	return u
}

// worker returns the worker of the given number, or nil.
func (e *Engine) worker(id int) *Engine {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.workers[id]
}

// closeWorkers closes the workers of the engine (as it is closed).
func (e *Engine) closeWorkers() {
	e.mu.Lock()
	ws := e.workers
	e.workers = nil
	e.mu.Unlock()
	for _, w := range ws {
		w.Close()
	}
}

// defineWorkers defines the functions of workers for the prelude.
func (e *Engine) defineWorkers() {
	vm := e.vm
	d := e.jsDOM
	d.Set("isWorker", func(call goja.FunctionCall) goja.Value { return vm.ToValue(e.opts.worker) })
	d.Set("newWorker", func(call goja.FunctionCall) goja.Value {
		e.newWorker(int(call.Argument(0).ToInteger()), call.Argument(1).String())
		return goja.Undefined()
	})
	d.Set("postToWorker", func(call goja.FunctionCall) goja.Value {
		if w := e.worker(int(call.Argument(0).ToInteger())); w != nil {
			w.receive(workerMessage{call.Argument(1).String(), e.transferPorts(w, portIDs(call.Argument(2)))})
		}
		return goja.Undefined()
	})
	d.Set("terminateWorker", func(call goja.FunctionCall) goja.Value {
		id := int(call.Argument(0).ToInteger())
		if w := e.worker(id); w != nil {
			e.mu.Lock()
			delete(e.workers, id)
			e.mu.Unlock()
			w.Close()
		}
		return goja.Undefined()
	})
	d.Set("postToOwner", func(call goja.FunctionCall) goja.Value {
		if o := e.owner; o != nil {
			data, id := call.Argument(0).String(), e.ownerID
			pairs := e.transferPorts(o, portIDs(call.Argument(1)))
			o.Post(func() { o.callPrelude("workerMessage", id, data, pairs) })
		}
		return goja.Undefined()
	})
	d.Set("closeWorker", func(call goja.FunctionCall) goja.Value {
		go e.Close() // (not in its own task, which Close interrupts)
		return goja.Undefined()
	})
	d.Set("importScripts", func(call goja.FunctionCall) goja.Value {
		for _, a := range call.Arguments {
			u, err := resolveURL(a.String(), e.baseURL())
			if err != nil {
				panic(vm.NewTypeError(err.Error()))
			}
			code, err := e.getScript(u)
			if err != nil {
				panic(vm.NewGoError(&domException{name: "NetworkError", msg: fmt.Sprintf("importScripts %s: %v", u, err)}))
			}
			prog, err := e.compile(u, code)
			if err != nil {
				panic(vm.NewGoError(&domException{name: "SyntaxError", msg: err.Error()}))
			}
			if _, err := vm.RunProgram(prog); err != nil {
				panic(err)
			}
		}
		return goja.Undefined()
	})
}
