// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"sync"

	"github.com/dop251/goja"
)

// Message ports that are transferred to other engines: a port of a
// MessageChannel that is in the transfer list of a message posted to
// another window or worker (as postMessage(data, origin, [port])) moves to
// the engine that gets the message, in its ports, and the port that it was
// entangled with sends its messages there, and gets those of the moved
// port (as reCAPTCHA talks to its frames and its worker, which it timed
// out waiting for without them). Each end of such a pair of ports has a
// number in portEnds, of the engine and the number of the port in it,
// which is 0 until the engine that gets the moved port has made it, while
// the messages to it wait.

// portEnd is an end of a pair of entangled ports in different engines.
type portEnd struct {
	e       *Engine
	id      int      // the number of the port in e (0 until it is made)
	pending []string // messages (as JSON) that wait for id
}

// portEnds are the ends of the ports of other engines, by their numbers.
var portEnds = struct {
	sync.Mutex
	ends map[int64]*portEnd
	last int64
}{ends: map[int64]*portEnd{}}

// newPortEnd returns the number of a new end of a port in the given engine
// with the given number there (0 for one that is not made yet).
func newPortEnd(e *Engine, id int) int64 {
	portEnds.Lock()
	defer portEnds.Unlock()
	portEnds.last++
	portEnds.ends[portEnds.last] = &portEnd{e: e, id: id}
	return portEnds.last
}

// postToPortEnd posts the given message (as JSON) to the port of the end
// of the given number, or keeps it until the port is made.
func postToPortEnd(end int64, data string) {
	portEnds.Lock()
	pe := portEnds.ends[end]
	if pe == nil {
		portEnds.Unlock()
		return
	}
	if pe.id == 0 {
		pe.pending = append(pe.pending, data)
		portEnds.Unlock()
		return
	}
	e, id := pe.e, pe.id
	portEnds.Unlock()
	e.Post(func() { e.callPrelude("portMessage", id, data) })
}

// bindPortEnd sets the number of the port of the given end, which has been
// made, and posts the messages that waited for it.
func bindPortEnd(end int64, id int) {
	portEnds.Lock()
	pe := portEnds.ends[end]
	if pe == nil {
		portEnds.Unlock()
		return
	}
	pe.id = id
	pending := pe.pending
	pe.pending = nil
	portEnds.Unlock()
	for _, data := range pending {
		postToPortEnd(end, data)
	}
}

// transferPorts moves the ports of the given numbers in this engine (the
// partners of the ports in the transfer list of a message, which stay
// here) to the given engine, which gets the message: each is linked to a
// new end there, and it returns the pairs of the numbers of the ends (of
// the partner here, and of the new port there) for the message. It is
// called in a task, from the scripts.
func (e *Engine) transferPorts(to *Engine, partners []int) [][2]int64 {
	var pairs [][2]int64
	for _, id := range partners {
		here := newPortEnd(e, id)
		there := newPortEnd(to, 0)
		e.callPrelude("linkPort", id, there)
		pairs = append(pairs, [2]int64{here, there})
	}
	return pairs
}

// portIDs returns the numbers of the ports of the given argument of a
// function of the prelude (an array of numbers), if any.
func portIDs(v goja.Value) []int {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil
	}
	var ids []int
	if o, ok := v.(*goja.Object); ok {
		for _, k := range o.Keys() {
			ids = append(ids, int(o.Get(k).ToInteger()))
		}
	}
	return ids
}

// definePorts defines the functions of transferred ports for the prelude.
func (e *Engine) definePorts() {
	d := e.jsDOM
	d.Set("portPost", func(call goja.FunctionCall) goja.Value {
		postToPortEnd(call.Argument(0).ToInteger(), call.Argument(1).String())
		return goja.Undefined()
	})
	d.Set("portBind", func(call goja.FunctionCall) goja.Value {
		bindPortEnd(call.Argument(0).ToInteger(), int(call.Argument(1).ToInteger()))
		return goja.Undefined()
	})
}
