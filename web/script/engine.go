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
	"sync/atomic"
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
	//
	// It is called without the lock of the page (but while no script runs),
	// to read the document into new widgets, and returns the function that
	// shows them (as it replaces the widgets of the page), which is called
	// with the lock held, or nil.
	Render func() (commit func())

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

	// Submit, if set, opens the page of the given URL with a POST request
	// with the given body, of the given content type, as for the
	// submission of a form with the POST method.
	Submit func(url, contentType string, body []byte)

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

// maxCallStack is the maximum depth of the calls of scripts, about that
// of browsers.
const maxCallStack = 10000

// Engine runs the scripts of a page (see the package documentation).
type Engine struct {
	opts Options
	vm   *goja.Runtime
	doc  *html.Node

	// mu protects the fields below it, which are used by other goroutines.
	mu      sync.Mutex
	queue   []func()
	wake    chan struct{}
	closed  bool
	timers  map[int]*time.Timer
	timerID int
	blobs   map[string]blob // the object URLs of blobs (URL.createObjectURL)
	modules map[string]bool // the URLs of the modules defined (moduleGraph)

	lock     func() bool
	unlock   func()
	pending  int // tasks queued or running, for tests (Wait)
	stopped  chan struct{}
	inTask   bool
	nodes    nodeIDs
	dirty    bool
	loading  bool       // whether the scripts of the page are being loaded
	current  *html.Node // the script element that is running, if any
	jsDOM    *goja.Object
	dispatch goja.Callable // __dom.dispatch, set by the prelude
	fire     goja.Callable // __dom.fireTimer, set by the prelude
	callback goja.Callable // __dom.callback, set by the prelude
	session  *Storage      // the sessionStorage of the page
	// lastRender is when the page was last rendered, and renderTimer
	// whether a render is pending (see [Engine.renderSoon]).
	lastRender time.Time
	imports    *importMap    // the import map of the page, if any
	renderCost time.Duration // how long the last render took
	// changedNodes are the nodes that have changed since the page was
	// last rendered (see [Engine.changedAround]).
	changedNodes []*html.Node
	renderTimer  bool
	ran          map[*html.Node]bool // the script elements that have run

	// taskTimer interrupts the task that is running when it has run for
	// longer than TaskTimeout, not counting renders: taskLeft is the time
	// left since taskStart. They are used by the goroutine of the engine.
	taskTimer *time.Timer
	taskLeft  time.Duration
	taskStart time.Time
	timeouts  int // the number of tasks interrupted for running too long
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
	// a script that recurses without end gets a RangeError (as in browsers)
	// instead of running until it times out, using ever more memory
	e.vm.SetMaxCallStackSize(maxCallStack)
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
	e.pending++
	e.queue = append(e.queue, task)
	select {
	case e.wake <- struct{}{}:
	default:
	}
}

// Closed returns whether the engine has been closed, as when the scripts
// of the page were stopped for running too long: then it runs no tasks.
func (e *Engine) Closed() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.closed
}

// Wait waits until the engine has no tasks queued or running (as for
// tests; timers that have not fired do not count).
func (e *Engine) Wait() {
	for {
		e.mu.Lock()
		n := e.pending
		e.mu.Unlock()
		if n == 0 {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// done records that a task has run.
func (e *Engine) done() {
	e.mu.Lock()
	e.pending--
	e.mu.Unlock()
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
	e.pending -= len(e.queue)
	e.queue = nil
	e.mu.Unlock()
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
				e.done()
				go e.Close() // the page is gone
				continue
			}
			e.runTask(task, unlock)
			e.renderSoon(lock, unlock)
			e.done()
		}
	}
}

// runTask runs the given task (with the lock of the page held, which
// unlock unlocks if not nil, at the end), within the time limit of a task.
func (e *Engine) runTask(task func(), unlock func()) {
	if unlock != nil {
		defer unlock()
	}
	e.taskLeft = e.opts.TaskTimeout
	e.taskStart = time.Now()
	var timedOut atomic.Bool
	timer := time.AfterFunc(e.opts.TaskTimeout, func() {
		timedOut.Store(true)
		e.vm.Interrupt(fmt.Errorf("a script ran for more than %v", e.opts.TaskTimeout))
	})
	e.taskTimer = timer
	defer func() { e.taskTimer = nil }()
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
	if timedOut.Load() {
		e.timeouts++
		if e.timeouts >= maxTimeouts {
			e.logf(slog.LevelError, "the scripts of the page were stopped, as they ran for too long %d times", e.timeouts)
			go e.Close() // the page stays as it is
		}
	}
}

// maxTimeouts is the number of times that the tasks of a page can be
// interrupted for running too long (TaskTimeout) before its scripts are
// stopped, as browsers stop pages that do not respond, rather than use a
// processor for as long as the page is open.
const maxTimeouts = 3

// renderInterval is the shortest time between the renders of a page at
// the end of tasks, which renders that are needed sooner wait for (as for
// a script that changes the page in each frame of an animation, or loads
// many modules that each change it); it is longer for pages that take
// longer to render (twice the time of the last render), so that rendering
// takes at most about a third of the time.
const renderInterval = 100 * time.Millisecond

// renderIfDirty renders the page if the document has changed (now, as
// for the layout of an element that a script needs). The widgets are
// shown with the given lock held, if not nil (otherwise it is held).
func (e *Engine) renderIfDirty(lock func() bool, unlock func()) {
	if !e.dirty || e.opts.Render == nil || e.loading {
		return
	}
	e.dirty = false
	e.changedNodes = e.changedNodes[:0]
	start := time.Now()
	// the time of rendering is not that of the scripts of the task
	if e.taskTimer != nil && e.taskTimer.Stop() {
		e.taskLeft -= start.Sub(e.taskStart)
		defer func() {
			e.taskStart = time.Now()
			e.taskTimer.Reset(max(e.taskLeft, 0))
		}()
	}
	commit := e.opts.Render()
	if commit != nil {
		if lock == nil {
			commit()
		} else if lock() {
			commit()
			unlock()
		}
	}
	e.lastRender = time.Now()
	e.renderCost = e.lastRender.Sub(start)
}

// renderSoon renders the page if the document has changed, after a task
// (without the lock of the page, which it takes to show it): now, or after
// [renderInterval] since the last render.
func (e *Engine) renderSoon(lock func() bool, unlock func()) {
	if !e.dirty || e.opts.Render == nil || e.loading {
		return
	}
	wait := max(renderInterval, 2*e.renderCost) - time.Since(e.lastRender)
	if wait <= 0 {
		e.renderIfDirty(lock, unlock)
		return
	}
	if e.renderTimer {
		return
	}
	e.renderTimer = true
	time.AfterFunc(wait, func() {
		e.Post(func() { e.renderTimer = false }) // renders at its end
	})
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
	var so *goja.StackOverflowError
	if errors.As(err, &so) {
		e.logf(slog.LevelError, "RangeError: Maximum call stack size exceeded%s", so.String())
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
		e.logf(slog.LevelError, "%s", ie.String()) // with the stack
		return
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
	prog, err := goja.Compile(name, rewriteImports(src, name), false)
	if err == nil {
		return prog, nil
	}
	if lowered, lerr := lower(src, name); lerr == nil {
		if p, perr := goja.Compile(name, rewriteImports(lowered, name), false); perr == nil {
			return p, nil
		}
	}
	if moduleSyntax.MatchString(src) {
		if bundled, berr := e.bundleModule(src, name); berr == nil {
			if p, perr := goja.Compile(name, rewriteImports(bundled, name), false); perr == nil {
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

// Resized fires the resize event of the window in a task, as when the
// viewport has another size.
func (e *Engine) Resized() {
	e.Post(func() { e.callPrelude("resized") })
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
	task := func() {
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
	}
	if d == 0 && !repeat {
		// queued now, so that such timers run in the order that they are
		// set (as React's scheduler and MessageChannel need), which the
		// goroutines of timers do not keep; a stopped timer for clearTimer
		t := time.AfterFunc(time.Hour, func() {})
		t.Stop()
		e.timers[id] = t
		e.pending++
		e.queue = append(e.queue, task)
		select {
		case e.wake <- struct{}{}:
		default:
		}
		return id
	}
	var fire func()
	fire = func() {
		e.Post(task)
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
