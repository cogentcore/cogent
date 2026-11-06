// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package script runs the scripts of the web pages of the web browser,
// with the goja JavaScript engine.
//
// The DOM of a page is its parsed HTML (x/net/html nodes), which htmlcore
// reads into widgets. Scripts change the nodes through the DOM API, which
// is written in JavaScript (prelude.js) on top of a small set of Go
// functions (dom.go), and the page is read into widgets again after each
// task of the scripts that changes it (see [Options.Render]).
//
// As in browsers, the scripts of a page run in one thread: an [Engine] has
// a goroutine that runs its tasks (scripts, timers, events, the results of
// network requests) one at a time, each while holding the lock of the
// widgets of the page (see [Options.Lock]), so that the scripts never run
// while the page is being styled or rendered. Promise callbacks
// (microtasks) run at the end of each task.
package script

import (
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/dop251/goja"
	"golang.org/x/net/html"
)

//go:embed prelude.js
var preludeSource string

// prelude is the compiled prelude, which is shared by the engines.
var prelude = sync.OnceValues(func() (*goja.Program, error) {
	return goja.Compile("prelude.js", preludeSource, true)
})

// Options are the options of an [Engine]: the functions with which the
// scripts of a page interact with the app that shows it.
type Options struct {

	// PageURL is the URL of the page, which the URLs of the
	// page are relative to (and its location).
	PageURL string

	// GetURL gets the resources of the page (scripts and the requests of
	// its scripts), as [htmlcore.Context.GetURL].
	GetURL func(url string) (*http.Response, error)

	// Lock and Unlock, if set, lock and unlock the widgets of the page
	// for running a task (see [Engine.SetLock]). Lock returns false if
	// the page is gone (as when its window is closed), which closes the
	// engine.
	Lock   func() bool
	Unlock func()

	// Render, if set, reads the document into the widgets of the page
	// again after a task has changed it. It is called with the lock held.
	Render func()

	// Layout, if set, returns the position and size of the box of the
	// given node in CSS px, relative to the top left of the page, after
	// the page is laid out (rendering it first if it has changed), and
	// false if it has none (as when it is not displayed). It is called
	// with the lock held.
	Layout func(n *html.Node) (x, y, w, h float32, ok bool)

	// Viewport, if set, returns the width and height of the viewport
	// in CSS px (as for innerWidth).
	Viewport func() (w, h float32)

	// MediaMatches, if set, returns whether a media query list matches
	// the viewport (as for matchMedia).
	MediaMatches func(query string) bool

	// Navigate, if set, opens the given URL, as for location.href, which
	// replaces the current page in the history if replace is true.
	Navigate func(url string, replace bool)

	// PushState, if set, is called when a script changes the URL of the
	// page without opening it (history.pushState and replaceState).
	PushState func(url string, replace bool)

	// Storage, if set, has the localStorage of the pages, by origin.
	Storage *Storage

	// Cookies, if set, has the cookies of the pages (as for document.cookie).
	Cookies http.CookieJar

	// Log, if set, receives the messages of the console of the
	// page (and the errors of its scripts), with their level.
	Log func(level slog.Level, msg string)

	// TaskTimeout is the longest time that a task can run before it is
	// interrupted (as a script that does not end); 0 is 10 seconds.
	TaskTimeout time.Duration
}

// Engine runs the scripts of a page (see the package documentation).
type Engine struct {
	opts Options
	vm   *goja.Runtime
	doc  *html.Node

	// mu protects the fields below it, which are used by other goroutines.
	mu       sync.Mutex
	queue    []func()
	wake     chan struct{}
	closed   bool
	timers   map[int]*time.Timer
	timerID  int
	lock     func() bool
	unlock   func()
	pending  sync.WaitGroup // tasks queued or running, for tests (Wait)
	stopped  chan struct{}
	inTask   bool
	nodes    nodeIDs
	dirty    bool
	loading  bool       // whether the scripts of the page are being loaded
	current  *html.Node // the script element that is running, if any
	jsDOM    *goja.Object
	dispatch goja.Callable       // __dom.dispatch, set by the prelude
	fire     goja.Callable       // __dom.fireTimer, set by the prelude
	callback goja.Callable       // __dom.callback, set by the prelude
	session  *Storage            // the sessionStorage of the page
	ran      map[*html.Node]bool // the script elements that have run
}

// New returns a new [Engine] for the given parsed document with the given
// options, which runs the prelude (the DOM and web APIs). The scripts of
// the document are run with [Engine.Load].
func New(doc *html.Node, opts Options) (*Engine, error) {
	prog, err := prelude()
	if err != nil {
		return nil, fmt.Errorf("compiling prelude: %w", err)
	}
	if opts.TaskTimeout == 0 {
		opts.TaskTimeout = 10 * time.Second
	}
	e := &Engine{opts: opts, doc: doc, wake: make(chan struct{}, 1), stopped: make(chan struct{}),
		timers: map[int]*time.Timer{}, lock: opts.Lock, unlock: opts.Unlock}
	e.nodes.init()
	e.vm = goja.New()
	e.vm.SetFieldNameMapper(goja.UncapFieldNameMapper())
	e.jsDOM = e.vm.NewObject()
	e.defineDOM()
	e.vm.Set("__dom", e.jsDOM)
	if _, err := e.vm.RunProgram(prog); err != nil {
		return nil, fmt.Errorf("running prelude: %w", err)
	}
	for name, dst := range map[string]*goja.Callable{"dispatch": &e.dispatch, "fireTimer": &e.fire, "callback": &e.callback} {
		f, ok := goja.AssertFunction(e.jsDOM.Get(name))
		if !ok {
			return nil, fmt.Errorf("prelude: no __dom.%s", name)
		}
		*dst = f
	}
	go e.loop()
	return e, nil
}

// SetLock sets the functions that lock and unlock the widgets of the page
// for each task (as once the page is shown).
func (e *Engine) SetLock(lock func() bool, unlock func()) {
	e.mu.Lock()
	e.lock, e.unlock = lock, unlock
	e.mu.Unlock()
}

// PageURL returns the URL of the page, which its scripts can change
// (with history.pushState). It must be called in a task.
func (e *Engine) PageURL() string {
	return e.opts.PageURL
}

// Document returns the document of the engine.
func (e *Engine) Document() *html.Node {
	return e.doc
}

// Post adds the given task to the queue of the engine, which runs it in
// its goroutine (with the lock of the page held). It does nothing once
// the engine is closed.
func (e *Engine) Post(task func()) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return
	}
	e.pending.Add(1)
	e.queue = append(e.queue, task)
	select {
	case e.wake <- struct{}{}:
	default:
	}
}

// Wait waits until the engine has no tasks queued or running (as for
// tests; timers that have not fired do not count).
func (e *Engine) Wait() {
	e.pending.Wait()
}

// Close stops the engine: its timers and the tasks that are queued.
func (e *Engine) Close() {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return
	}
	e.closed = true
	for _, t := range e.timers {
		t.Stop()
	}
	e.timers = nil
	n := len(e.queue)
	e.queue = nil
	e.mu.Unlock()
	for range n {
		e.pending.Done()
	}
	e.vm.Interrupt(errClosed)
	close(e.wake)
}

var errClosed = errors.New("the page is closed")

// loop runs the tasks of the engine, until it is closed.
func (e *Engine) loop() {
	defer close(e.stopped)
	for range e.wake {
		for {
			e.mu.Lock()
			if e.closed || len(e.queue) == 0 {
				e.mu.Unlock()
				break
			}
			task := e.queue[0]
			e.queue = e.queue[1:]
			lock, unlock := e.lock, e.unlock
			e.mu.Unlock()
			if lock != nil && !lock() {
				e.pending.Done()
				go e.Close() // the page is gone
				continue
			}
			e.runTask(task, unlock)
			e.pending.Done()
		}
	}
}

// runTask runs the given task (with the lock of the page held, which
// unlock unlocks if not nil), within the time limit of a task, and renders
// the page afterward if the task changed it.
func (e *Engine) runTask(task func(), unlock func()) {
	if unlock != nil {
		defer unlock()
	}
	timer := time.AfterFunc(e.opts.TaskTimeout, func() {
		e.vm.Interrupt(fmt.Errorf("a script ran for more than %v", e.opts.TaskTimeout))
	})
	e.inTask = true
	func() {
		defer func() {
			if r := recover(); r != nil {
				e.logf(slog.LevelError, "script panic: %v", r)
			}
		}()
		task()
	}()
	e.inTask = false
	timer.Stop()
	e.vm.ClearInterrupt()
	e.renderIfDirty()
}

// renderIfDirty renders the page if the document has changed.
func (e *Engine) renderIfDirty() {
	if e.dirty && e.opts.Render != nil && !e.loading {
		e.dirty = false
		e.opts.Render()
	}
}

// logf logs the given message to the console of the page.
func (e *Engine) logf(level slog.Level, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if e.opts.Log != nil {
		e.opts.Log(level, msg)
		return
	}
	slog.Log(nil, level, "script: "+msg)
}

// reportError logs the given error of a script.
func (e *Engine) reportError(err error) {
	if err == nil {
		return
	}
	var ex *goja.Exception
	if errors.As(err, &ex) {
		e.logf(slog.LevelError, "%s", ex.String())
		return
	}
	var ie *goja.InterruptedError
	if errors.As(err, &ie) {
		if v, ok := ie.Value().(error); ok && errors.Is(v, errClosed) {
			return
		}
	}
	e.logf(slog.LevelError, "%v", err)
}

// run runs the given source as a script with the given name, reporting
// its errors.
// If it does not compile, it is rewritten without the features that goja
// does not support (see [lower]), or bundled as a module if it has the
// syntax of modules.
func (e *Engine) run(name, src string) {
	prog, err := e.compile(name, src)
	if err != nil {
		e.reportError(err)
		return
	}
	_, err = e.vm.RunProgram(prog)
	e.reportError(err)
}

// compile compiles the given script, rewriting it if needed (see [Engine.run]).
func (e *Engine) compile(name, src string) (*goja.Program, error) {
	prog, err := goja.Compile(name, rewriteImports(src), false)
	if err == nil {
		return prog, nil
	}
	if lowered, lerr := lower(src, name); lerr == nil {
		if p, perr := goja.Compile(name, rewriteImports(lowered), false); perr == nil {
			return p, nil
		}
	}
	if moduleSyntax.MatchString(src) {
		if bundled, berr := e.bundleModule(src, name); berr == nil {
			if p, perr := goja.Compile(name, rewriteImports(bundled), false); perr == nil {
				return p, nil
			}
		}
	}
	return nil, err
}

// RunString runs the given JavaScript source in a task and returns its
// result (exported to Go) and error, waiting for it (as for tests).
func (e *Engine) RunString(src string) (any, error) {
	type result struct {
		v   any
		err error
	}
	ch := make(chan result, 1)
	e.Post(func() {
		v, err := e.vm.RunString(src)
		if err != nil {
			ch <- result{nil, err}
			return
		}
		ch <- result{v.Export(), nil}
	})
	select {
	case r := <-ch:
		return r.v, r.err
	case <-e.stopped:
		return nil, errClosed
	}
}

// Rerender renders the page again in a task, as when it is read again
// for another width of its viewport.
func (e *Engine) Rerender() {
	e.Post(func() { e.dirty = true })
}

// Eval runs the given JavaScript in a task (as for a javascript: URL).
func (e *Engine) Eval(src string) {
	e.Post(func() { e.run("javascript: URL", src) })
}

// Changed records that the document has changed (in a task), so that the
// page is rendered again at the end of the task, as for the value of a
// form control that is changed by the user.
func (e *Engine) Changed() {
	e.dirty = true
}

// setTimer starts a timer that fires the timer of the given id in the
// prelude after the given duration, once or repeatedly.
func (e *Engine) setTimer(ms float64, repeat bool) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return 0
	}
	e.timerID++
	id := e.timerID
	d := time.Duration(max(ms, 0) * float64(time.Millisecond))
	if repeat {
		d = max(d, 4*time.Millisecond) // as browsers clamp nested timers
	}
	var fire func()
	fire = func() {
		e.Post(func() {
			e.mu.Lock()
			_, ok := e.timers[id]
			if ok && !repeat {
				delete(e.timers, id)
			}
			e.mu.Unlock()
			if !ok {
				return // cleared
			}
			_, err := e.fire(goja.Undefined(), e.vm.ToValue(id))
			e.reportError(err)
		})
		if repeat {
			e.mu.Lock()
			if t, ok := e.timers[id]; ok {
				t.Reset(d)
			}
			e.mu.Unlock()
		}
	}
	e.timers[id] = time.AfterFunc(d, fire)
	return id
}

// clearTimer stops the timer of the given id.
func (e *Engine) clearTimer(id int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if t, ok := e.timers[id]; ok {
		t.Stop()
		delete(e.timers, id)
	}
}

// callBack calls the callback of the given id in the prelude with the
// given arguments, in a task (as for the result of a network request).
func (e *Engine) callBack(id int, args ...any) {
	e.Post(func() {
		vals := []goja.Value{e.vm.ToValue(id)}
		for _, a := range args {
			vals = append(vals, e.vm.ToValue(a))
		}
		_, err := e.callback(goja.Undefined(), vals...)
		e.reportError(err)
	})
}
