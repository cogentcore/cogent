// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The DOM and web APIs of a page, on top of the Go functions of __dom
// (dom.go, net.go). Nodes are referred to by ids, which are wrapped in one
// object each (wrap). It runs before the scripts of the page.

(function (g, D) {
	"use strict";

	// Promise is the native one, which the prelude schedules with (as for
	// queueMicrotask), even if a page replaces it (core-js on BBC does,
	// with a Promise that schedules with queueMicrotask).
	const Promise = g.Promise;
	const promiseThen = Promise.prototype.then; // which core-js also replaces
	const resolvedPromise = Promise.resolve();
	// The native then makes its promise with the species of the constructor
	// of the promise, which core-js sets to its polyfill, whose microtasks
	// are queueMicrotask, which made another of its promises: a loop that
	// stopped the scripts of bbc.com. This promise keeps the native one.
	{
		const NativePromise = Promise;
		Object.defineProperty(resolvedPromise, "constructor", { value: { [Symbol.species]: NativePromise } });
	}

	// WeakRef and FinalizationRegistry, which the engine does not have (the
	// scripts of google.com use WeakRef): the references are strong, and
	// the finalizers are never called.
	if (typeof g.WeakRef === "undefined") {
		g.WeakRef = class WeakRef {
			constructor(target) { this._target = target; }
			deref() { return this._target; }
		};
	}
	if (typeof g.FinalizationRegistry === "undefined") {
		g.FinalizationRegistry = class FinalizationRegistry {
			register() {}
			unregister() { return false; }
		};
	}

	// ---- wrapping nodes ----

	const wrappers = new Map(); // id -> object
	const ID = Symbol("id");

	const tagClasses = {}; // tag name -> class of its elements

	// the namespaces of elements, by their URI, and their URIs
	const namespaceURIs = { "": "http://www.w3.org/1999/xhtml", svg: "http://www.w3.org/2000/svg", math: "http://www.w3.org/1998/Math/MathML" };
	const namespaces = { "http://www.w3.org/2000/svg": "svg", "http://www.w3.org/1998/Math/MathML": "math" };

	function wrap(id) {
		if (!id) return null;
		let w = wrappers.get(id);
		if (w) return w;
		let cls;
		switch (D.type(id)) {
			case 1: cls = D.ns(id) === "svg" ? (D.name(id) === "svg" ? SVGSVGElement : SVGElement) : (tagClasses[D.name(id)] || HTMLElement); break;
			case 3: cls = Text; break;
			case 8: cls = Comment; break;
			case 9: cls = HTMLDocument; break; // (documents are parsed as HTML)
			case 10: cls = DocumentType; break;
			case 11: cls = DocumentFragment; break;
			default: cls = Node;
		}
		w = Object.create(cls.prototype);
		Object.defineProperty(w, ID, { value: id });
		wrappers.set(id, w);
		if (registry.size && registry.has(D.name(id))) upgrade(w);
		return w;
	}

	// ---- custom elements ----

	const registry = new Map(); // name -> { cls, observed }
	const definedNames = new Map(); // class -> name
	const connectedSet = new WeakSet(); // the custom elements connected
	let upgrading = null; // the element that HTMLElement's constructor returns

	// upgrade runs the constructor of the class of the given custom element
	// on it (once), and its connectedCallback if it is in the document.
	function upgrade(w) {
		const def = registry.get(D.name(w[ID]));
		if (!def || w instanceof def.cls) return;
		upgrading = w;
		try {
			new def.cls();
		} catch (e) {
			reportError(e);
			return;
		} finally {
			upgrading = null;
		}
		if (typeof w.attributeChangedCallback === "function") {
			for (const n of def.observed) {
				const v = D.attr(w[ID], n);
				if (v !== null && v !== undefined) callReaction(w, "attributeChangedCallback", n, null, v);
			}
		}
		if (connected(w[ID])) connect(w[ID]);
	}

	function callReaction(w, name, ...args) {
		if (typeof w[name] !== "function") return;
		try { w[name](...args); } catch (e) { reportError(e); }
	}

	const shadowHosts = new Map(); // the id of a shadow root -> that of its host
	const shadows = new WeakMap(); // element -> { root, mode }

	// connected returns whether the node of the given id is in the document
	// (through the hosts of shadow roots).
	function connected(id) {
		const doc = D.doc();
		for (let n = id; n; n = D.parent(n) || shadowHosts.get(n)) if (n === doc) return true;
		return false;
	}

	// customIn returns the ids of the defined custom elements in the tree of
	// the node of the given id, including it, in tree order.
	function customIn(id) {
		if (!registry.size || !id) return [];
		const sel = [...registry.keys()].join(",");
		const ids = D.type(id) === 1 && D.matches(id, sel) ? [id] : [];
		return ids.concat(D.queryAll(id, sel));
	}

	function connect(id) {
		const w = wrap(id); // which upgrades it, and connects it then
		if (!(w instanceof HTMLElement) || connectedSet.has(w) || !registry.has(D.name(id)) || !(w instanceof registry.get(D.name(id)).cls)) return;
		connectedSet.add(w);
		callReaction(w, "connectedCallback");
	}

	function disconnect(id) {
		const w = wrappers.get(id);
		if (!w || !connectedSet.has(w)) return;
		connectedSet.delete(w);
		callReaction(w, "disconnectedCallback");
	}

	// the reactions of custom elements to the changes of the DOM
	{
		const insert = D.insert, remove = D.remove, setInner = D.setInner, write = D.write, setText = D.setText;
		const setAttr = D.setAttr, removeAttr = D.removeAttr;
		const removing = id => connected(id) ? customIn(id) : [];
		const childrenOf = id => D.type(id) === 11 ? D.children(id) : [id];
		D.insert = function (p, c, ref) {
			if (!registry.size) return insert(p, c, ref);
			const roots = childrenOf(c);
			const moved = [].concat(...roots.map(removing));
			insert(p, c, ref);
			moved.forEach(disconnect);
			if (connected(p)) for (const r of roots) customIn(r).forEach(connect);
		};
		D.remove = function (id) {
			const ids = registry.size ? removing(id) : [];
			remove(id);
			ids.forEach(disconnect);
		};
		const replacing = (f) => function (id, v) {
			if (!registry.size) return f(id, v);
			const old = connected(id) ? [].concat(...D.children(id).map(customIn)) : [];
			f(id, v);
			old.forEach(disconnect);
			if (connected(id)) for (const c of D.children(id)) customIn(c).forEach(connect);
		};
		D.setInner = replacing(setInner);
		D.setText = replacing(setText);
		D.write = function (html) {
			write(html);
			customIn(D.doc()).forEach(connect);
		};
		const attrChanged = (id, n, f) => {
			const w = wrappers.get(id);
			const def = w && registry.get(D.name(id));
			if (!def || !def.observed.includes(n) || !(w instanceof def.cls)) return f();
			const old = D.attr(id, n);
			f();
			callReaction(w, "attributeChangedCallback", n, old === undefined ? null : old, D.attr(id, n));
		};
		D.setAttr = (id, n, v) => attrChanged(id, n, () => setAttr(id, n, v));
		D.removeAttr = (id, n) => attrChanged(id, n, () => removeAttr(id, n));
	}

	// the records of the changes of the DOM for the mutation observers (see
	// MutationObserver), made as the DOM is changed while any observe
	const mutationObservers = new Set();
	{
		const insert = D.insert, remove = D.remove, setInner = D.setInner, setText = D.setText;
		const setAttr = D.setAttr, removeAttr = D.removeAttr, setData = D.setData;
		const record = (type, target, init) => {
			for (const mo of mutationObservers) mo._record(type, target, init);
		};
		const nodes = ids => ids.map(wrap);
		const childList = (p, added, removed, prev, next) => record("childList", p, {
			addedNodes: nodes(added), removedNodes: nodes(removed),
			previousSibling: wrap(prev), nextSibling: wrap(next),
		});
		D.insert = function (p, c, ref) {
			if (!mutationObservers.size) return insert(p, c, ref);
			const added = D.type(c) === 11 ? D.children(c) : [c];
			const old = D.type(c) === 11 ? 0 : D.parent(c);
			if (old) childList(old, [], [c], D.prev(c), D.next(c));
			const prev = ref ? D.prev(ref) : D.last(p);
			insert(p, c, ref);
			if (added.length) childList(p, added, [], prev, ref || 0);
		};
		D.remove = function (id) {
			const p = mutationObservers.size ? D.parent(id) : 0;
			if (!p) return remove(id);
			const prev = D.prev(id), next = D.next(id);
			remove(id);
			childList(p, [], [id], prev, next);
		};
		const replacing = f => function (id, v) {
			if (!mutationObservers.size) return f(id, v);
			if (D.type(id) === 3 || D.type(id) === 8) {
				const old = D.data(id);
				f(id, v);
				record("characterData", id, { oldValue: old });
				return;
			}
			const removed = D.children(id);
			f(id, v);
			const added = D.children(id);
			if (removed.length || added.length) childList(id, added, removed, 0, 0);
		};
		D.setInner = replacing(setInner);
		D.setText = replacing(setText);
		D.setData = function (id, v) {
			if (!mutationObservers.size) return setData(id, v);
			const old = D.data(id);
			setData(id, v);
			record("characterData", id, { oldValue: old });
		};
		const attr = (f, removing) => function (id, n, v) {
			if (!mutationObservers.size) return f(id, n, v);
			const old = D.attr(id, n);
			if (removing && old === undefined) return f(id, n, v);
			f(id, n, v);
			record("attributes", id, { attributeName: String(n).toLowerCase(), oldValue: old === undefined ? null : old });
		};
		D.setAttr = attr(setAttr, false);
		D.removeAttr = attr(removeAttr, true);
	}

	function idOf(n) {
		if (n === null || n === undefined || n[ID] === undefined) {
			throw new TypeError("the value is not a node");
		}
		return n[ID];
	}

	// NodeList and HTMLCollection are arrays (so that scripts can use the
	// methods of arrays on them, as many do), whose map and filter return
	// plain arrays.
	class NodeList extends Array {
		static get [Symbol.species]() { return Array; }
		item(i) { return this[i] === undefined ? null : this[i]; }
	}
	class HTMLCollection extends NodeList {
		namedItem(n) { return this.find(el => el.id === n || el.getAttribute("name") === n) || null; }
	}

	function list(ids, cls) {
		const a = new (cls || NodeList)();
		for (let i = 0; i < ids.length; i++) a.push(wrap(ids[i]));
		return a;
	}

	// DOMException is the error of the DOM, whose name is its kind (as
	// NotSupportedError), as scripts check it (as the custom elements of
	// GitHub, which ignore a NotSupportedError DOMException of a name that
	// is already defined, and got an Error).
	const domExceptionCodes = { IndexSizeError: 1, HierarchyRequestError: 3, WrongDocumentError: 4, InvalidCharacterError: 5, NoModificationAllowedError: 7, NotFoundError: 8, NotSupportedError: 9, InvalidStateError: 11, SyntaxError: 12, InvalidModificationError: 13, NamespaceError: 14, InvalidAccessError: 15, TypeMismatchError: 17, SecurityError: 18, NetworkError: 19, AbortError: 20, URLMismatchError: 21, QuotaExceededError: 22, TimeoutError: 23, InvalidNodeTypeError: 24, DataCloneError: 25 };
	class DOMException extends Error {
		constructor(msg, name) { super(msg === undefined ? "" : String(msg)); this.name = name === undefined ? "Error" : String(name); }
		get code() { return domExceptionCodes[this.name] || 0; }
	}
	for (const [k, v] of Object.entries(domExceptionCodes)) {
		DOMException[k.replace(/([a-z])([A-Z])/g, "$1_$2").toUpperCase().replace(/_ERROR$/, "_ERR")] = v;
	}
	function domException(name, msg) {
		return new DOMException(msg, name);
	}

	// ---- events ----

	const listeners = new WeakMap(); // target -> {type: [listener]}

	class Event {
		constructor(type, init) {
			init = init || {};
			this.type = String(type);
			this.bubbles = !!init.bubbles;
			this.cancelable = !!init.cancelable;
			this.composed = !!init.composed;
			this.defaultPrevented = false;
			this._target = null;
			this._currentTarget = null;
			this._phase = 0;
			this.timeStamp = performance.now();
			this.isTrusted = false;
			this._stop = false;
			this._stopNow = false;
		}
		preventDefault() { if (this.cancelable) this.defaultPrevented = true; }
		stopPropagation() { this._stop = true; }
		stopImmediatePropagation() { this._stop = true; this._stopNow = true; }
		composedPath() { return this._path ? this._path.slice() : []; }
		// (accessors of the prototype, as in browsers, whose getters
		// libraries use, as the delegated events of GitHub, which define
		// the currentTarget of an event with that of Event.prototype)
		get target() { return this._target; }
		get currentTarget() { return this._currentTarget; }
		get eventPhase() { return this._phase; }
		get srcElement() { return this.target; }
		get returnValue() { return !this.defaultPrevented; }
		set returnValue(v) { if (!v) this.preventDefault(); }
		get cancelBubble() { return this._stop; }
		set cancelBubble(v) { if (v) this._stop = true; }
	}
	Event.NONE = 0; Event.CAPTURING_PHASE = 1; Event.AT_TARGET = 2; Event.BUBBLING_PHASE = 3;

	class CustomEvent extends Event {
		constructor(type, init) { super(type, init); this.detail = init && init.detail !== undefined ? init.detail : null; }
	}
	class UIEvent extends Event {
		constructor(type, init) { super(type, init); this.detail = (init && init.detail) || 0; this.view = (init && init.view) || null; }
	}
	class MouseEvent extends UIEvent {
		constructor(type, init) {
			super(type, init);
			init = init || {};
			for (const k of ["clientX", "clientY", "screenX", "screenY", "pageX", "pageY", "offsetX", "offsetY", "button", "buttons"]) this[k] = init[k] || 0;
			for (const k of ["ctrlKey", "shiftKey", "altKey", "metaKey"]) this[k] = !!init[k];
			this.relatedTarget = init.relatedTarget || null;
		}
		getModifierState(k) { return !!this[k.toLowerCase() + "Key"]; }
	}
	class PointerEvent extends MouseEvent {
		constructor(type, init) { super(type, init); this.pointerType = (init && init.pointerType) || "mouse"; this.pointerId = 1; this.isPrimary = true; }
	}
	class KeyboardEvent extends UIEvent {
		constructor(type, init) {
			super(type, init);
			init = init || {};
			this.key = init.key || ""; this.code = init.code || "";
			this.keyCode = init.keyCode || 0; this.which = this.keyCode; this.charCode = init.charCode || 0;
			this.repeat = !!init.repeat; this.isComposing = false;
			for (const k of ["ctrlKey", "shiftKey", "altKey", "metaKey"]) this[k] = !!init[k];
		}
		getModifierState(k) { return !!this[k.toLowerCase() + "Key"]; }
	}
	class FocusEvent extends UIEvent { constructor(type, init) { super(type, init); this.relatedTarget = (init && init.relatedTarget) || null; } }
	class InputEvent extends UIEvent { constructor(type, init) { super(type, init); this.data = (init && init.data) || null; this.inputType = (init && init.inputType) || ""; } }
	class WheelEvent extends MouseEvent { constructor(type, init) { super(type, init); this.deltaX = (init && init.deltaX) || 0; this.deltaY = (init && init.deltaY) || 0; this.deltaZ = 0; this.deltaMode = 0; } }
	class ErrorEvent extends Event { constructor(type, init) { super(type, init); init = init || {}; this.message = init.message || ""; this.error = init.error; this.filename = init.filename || ""; } }
	class MessageEvent extends Event { constructor(type, init) { super(type, init); this.data = init && init.data; this.origin = (init && init.origin) || ""; this.source = (init && init.source) || null; this.ports = (init && init.ports) || []; this.lastEventId = ""; } }
	class ProgressEvent extends Event { constructor(type, init) { super(type, init); init = init || {}; this.loaded = init.loaded || 0; this.total = init.total || 0; this.lengthComputable = !!init.lengthComputable; } }
	const popStates = new WeakMap();
	class PopStateEvent extends Event {
		constructor(type, init) { super(type, init); popStates.set(this, init && init.state !== undefined ? init.state : null); }
		get state() { return popStates.get(this); }
	}
	class HashChangeEvent extends Event { constructor(type, init) { super(type, init); this.oldURL = (init && init.oldURL) || ""; this.newURL = (init && init.newURL) || ""; } }
	class SubmitEvent extends Event { constructor(type, init) { super(type, init); this.submitter = (init && init.submitter) || null; } }
	class AnimationEvent extends Event {
		constructor(type, init) { super(type, init); init = init || {}; this.animationName = init.animationName || ""; this.elapsedTime = init.elapsedTime || 0; this.pseudoElement = init.pseudoElement || ""; }
	}
	class TransitionEvent extends Event {
		constructor(type, init) { super(type, init); init = init || {}; this.propertyName = init.propertyName || ""; this.elapsedTime = init.elapsedTime || 0; this.pseudoElement = init.pseudoElement || ""; }
	}
	class TouchEvent extends UIEvent {}

	class EventTarget {
		addEventListener(type, fn, opts) {
			if (this === undefined || this === null) return g.addEventListener(type, fn, opts); // as a function of the window
			if (!fn) return;
			const capture = typeof opts === "boolean" ? opts : !!(opts && opts.capture);
			const once = !!(opts && typeof opts === "object" && opts.once);
			let m = listeners.get(this);
			if (!m) { m = {}; listeners.set(this, m); }
			const l = m[type] || (m[type] = []);
			for (const x of l) if (x.fn === fn && x.capture === capture) return;
			l.push({ fn, capture, once });
			listenerAdded(this, type);
			if (opts && typeof opts === "object" && opts.signal) {
				opts.signal.addEventListener("abort", () => this.removeEventListener(type, fn, opts));
			}
		}
		removeEventListener(type, fn, opts) {
			if (this === undefined || this === null) return g.removeEventListener(type, fn, opts);
			const capture = typeof opts === "boolean" ? opts : !!(opts && opts.capture);
			const m = listeners.get(this);
			if (!m || !m[type]) return;
			m[type] = m[type].filter(x => !(x.fn === fn && x.capture === capture));
		}
		dispatchEvent(ev) {
			if (this === undefined || this === null) return g.dispatchEvent(ev);
			ev._target = this;
			const path = [];
			for (let t = this; t; t = parentTarget(t, ev.type)) path.push(t);
			ev._path = path;
			const invoke = (t, phase) => {
				ev._currentTarget = t;
				ev._phase = phase;
				const m = listeners.get(t);
				const ls = m && m[ev.type] ? m[ev.type].slice() : [];
				for (const x of ls) {
					if (phase === 1 && !x.capture) continue;
					if (phase === 3 && x.capture) continue;
					if (x.once) t.removeEventListener(ev.type, x.fn, x.capture);
					callListener(x.fn, t, ev);
					if (ev._stopNow) return;
				}
				if (phase !== 1) {
					const h = handlerOf(t, ev.type);
					if (h) {
						const r = callListener(h, t, ev);
						if (r === false) ev.preventDefault();
					}
				}
			};
			for (let i = path.length - 1; i > 0 && !ev._stop; i--) invoke(path[i], 1);
			if (!ev._stop) invoke(this, 2);
			if (ev.bubbles) for (let i = 1; i < path.length && !ev._stop; i++) invoke(path[i], 3);
			ev._currentTarget = null;
			ev._phase = 0;
			return !ev.defaultPrevented;
		}
	}

	// wiredTypes are the types of events for which the widgets of elements
	// are set up when the page is rendered (see hasListener), which it is
	// rendered again for when a listener for them is added.
	const wiredTypes = { click: true, submit: true };
	function listenerAdded(t, type) {
		if (!wiredTypes[type]) return;
		if (t instanceof Node) D.listenerAdded(t[ID]);
		else if (t === g) D.listenerAdded(D.doc());
	}

	function callListener(fn, t, ev) {
		try {
			if (typeof fn === "function") return fn.call(t, ev);
			if (fn && typeof fn.handleEvent === "function") return fn.handleEvent(ev);
		} catch (e) {
			reportError(e);
		}
	}

	function reportError(e) {
		let msg = e && e.stack ? e.stack : String(e);
		if (e && e.message && msg.indexOf(e.message) < 0) msg = e.message + "\n" + msg;
		D.log("error", "Uncaught " + msg);
		const h = g.onerror;
		if (typeof h === "function") {
			try { h(String(e && e.message || e), "", 0, 0, e); } catch (e2) {}
		}
	}

	// parentTarget returns the next target of an event after the given one:
	// its parent, or the slot that it is assigned to when its parent has a
	// shadow root, or the host of a shadow root, as events propagate in the
	// composed tree in browsers (the menus of MDN open on clicks on their
	// buttons, which their shadow roots listen to on their slots).
	function parentTarget(t, type) {
		if (t === g) return null;
		// a load event of an element (as a script or an image) does not
		// reach the window, as in browsers: whenReady of web-vitals (on
		// nytimes.com) listens for the load of the window with a capturing
		// listener, which adds another for each load until the page is
		// complete, so that they doubled with each script that loaded
		if (type === "load" && t instanceof Document) return null;
		if (t instanceof Node) {
			const p = t.parentNode;
			if (p) {
				const sh = shadows.get(p);
				if (sh) {
					const slot = assignedSlot(sh.root, t);
					if (slot) return slot;
				}
				return p;
			}
			if (t instanceof Document) return g;
			const host = t[ID] !== undefined && shadowHosts.get(t[ID]);
			if (host) return wrap(host);
			return null;
		}
		return null;
	}

	// assignedSlot returns the slot of the given shadow root that the given
	// child of its host is assigned to, if any.
	function assignedSlot(root, child) {
		const name = child.nodeType === 1 ? (child.getAttribute("slot") || "") : "";
		for (const s of root.querySelectorAll("slot")) {
			if ((s.getAttribute("name") || "") === name) return s;
		}
		return null;
	}

	// handlerOf returns the on<type> handler of the given target: its
	// property (as onclick = f), or that of its attribute, compiled.
	const attrHandlers = new WeakMap(); // element -> {type: [source, fn]}
	function handlerOf(t, type) {
		const p = t["_on" + type];
		if (typeof p === "function") return p;
		if (!(t instanceof Element)) {
			if (t === g || t instanceof Document) {
				const b = document.body;
				if (t === g && b && (type === "load" || type === "unload" || type === "beforeunload")) return handlerOf(b, type);
			}
			return null;
		}
		const src = D.attr(t[ID], "on" + type);
		if (src === null) return null;
		let m = attrHandlers.get(t);
		if (!m) { m = {}; attrHandlers.set(t, m); }
		const c = m[type];
		if (c && c[0] === src) return c[1];
		let fn = null;
		try {
			fn = new Function("event", "with (this.ownerDocument || document) { with (this) { " + src + "\n} }");
		} catch (e) {
			try { fn = new Function("event", src); } catch (e2) { reportError(e2); }
		}
		m[type] = [src, fn];
		return fn;
	}

	const eventTypes = ["click", "dblclick", "mousedown", "mouseup", "mouseover", "mouseout", "mousemove", "mouseenter",
		"mouseleave", "keydown", "keyup", "keypress", "focus", "blur", "change", "input", "submit", "reset", "load",
		"error", "resize", "scroll", "contextmenu", "wheel", "pointerdown", "pointerup", "pointermove", "touchstart",
		"touchend", "animationend", "transitionend", "toggle", "beforeunload", "unload", "hashchange", "popstate",
		"message", "select", "focusin", "focusout", "DOMContentLoaded", "readystatechange", "abort", "pageshow", "pagehide", "storage", "invalid"];
	function defineHandlers(proto) {
		for (const type of eventTypes) {
			Object.defineProperty(proto, "on" + type.toLowerCase(), {
				get() { return this["_on" + type] || null; },
				set(v) { this["_on" + type] = typeof v === "function" ? v : null; listenerAdded(this, type); },
				configurable: true,
			});
		}
	}

	// ---- nodes ----

	class Node extends EventTarget {
		get nodeType() { return D.type(this[ID]); }
		get nodeName() {
			switch (this.nodeType) {
				case 1: return this.tagName;
				case 3: return "#text";
				case 8: return "#comment";
				case 9: return "#document";
				case 10: return D.name(this[ID]);
				case 11: return "#document-fragment";
			}
			return "";
		}
		get nodeValue() { const t = this.nodeType; return t === 3 || t === 8 ? D.data(this[ID]) : null; }
		set nodeValue(v) { const t = this.nodeType; if (t === 3 || t === 8) D.setData(this[ID], String(v)); }
		get parentNode() { return wrap(D.parent(this[ID])); }
		get parentElement() { const p = this.parentNode; return p && p.nodeType === 1 ? p : null; }
		get childNodes() { return list(D.children(this[ID])); }
		get firstChild() { return wrap(D.first(this[ID])); }
		get lastChild() { return wrap(D.last(this[ID])); }
		get nextSibling() { return wrap(D.next(this[ID])); }
		get previousSibling() { return wrap(D.prev(this[ID])); }
		get ownerDocument() { return this.nodeType === 9 ? null : document; }
		get isConnected() { return connected(this[ID]); }
		get textContent() { const t = this.nodeType; return t === 9 || t === 10 ? null : D.text(this[ID]); }
		set textContent(v) { D.setText(this[ID], v === null || v === undefined ? "" : String(v)); }
		get baseURI() { return D.baseURL(this[ID]); }
		hasChildNodes() { return D.first(this[ID]) !== 0; }
		getRootNode() { let n = this; while (n.parentNode) n = n.parentNode; return n; }
		contains(o) { for (let n = o; n; n = n.parentNode) if (n === this) return true; return false; }
		appendChild(c) { D.insert(this[ID], idOf(c), 0); return c; }
		insertBefore(c, ref) { D.insert(this[ID], idOf(c), ref ? idOf(ref) : 0); return c; }
		removeChild(c) {
			if (c.parentNode !== this) throw domException("NotFoundError", "the node is not a child of this node");
			D.remove(idOf(c)); return c;
		}
		replaceChild(n, old) { this.insertBefore(n, old); return this.removeChild(old); }
		cloneNode(deep) { return wrap(D.clone(this[ID], !!deep)); }
		isEqualNode(o) { return !!o && D.outer(this[ID]) === D.outer(idOf(o)); }
		isSameNode(o) { return o === this; }
		compareDocumentPosition(o) {
			if (o === this) return 0;
			if (this.contains(o)) return 20; // contained by, following
			if (o.contains(this)) return 10; // contains, preceding
			const all = document.documentElement ? [document.documentElement].concat(document.documentElement.querySelectorAll("*")) : [];
			const a = all.indexOf(this), b = all.indexOf(o);
			return a < b ? 4 : 2;
		}
		normalize() {}
	}
	Node.ELEMENT_NODE = 1; Node.TEXT_NODE = 3; Node.COMMENT_NODE = 8; Node.DOCUMENT_NODE = 9;
	Node.DOCUMENT_TYPE_NODE = 10; Node.DOCUMENT_FRAGMENT_NODE = 11;
	Node.DOCUMENT_POSITION_DISCONNECTED = 1; Node.DOCUMENT_POSITION_PRECEDING = 2; Node.DOCUMENT_POSITION_FOLLOWING = 4;
	Node.DOCUMENT_POSITION_CONTAINS = 8; Node.DOCUMENT_POSITION_CONTAINED_BY = 16;

	// methods of nodes that have children (Element, Document, DocumentFragment)
	const parentNodeMethods = {
		get children() { return list(D.children(this[ID]).filter(id => D.type(id) === 1), HTMLCollection); },
		get firstElementChild() { const c = this.children; return c.length ? c[0] : null; },
		get lastElementChild() { const c = this.children; return c.length ? c[c.length - 1] : null; },
		get childElementCount() { return this.children.length; },
		querySelector(sel) { return wrap(D.query(this[ID], String(sel))); },
		querySelectorAll(sel) { return list(D.queryAll(this[ID], String(sel))); },
		getElementsByTagName(tag) { return list(D.queryAll(this[ID], tag === "*" ? "*" : String(tag).toLowerCase().replace(/:/g, "\\:")), HTMLCollection); },
		getElementsByClassName(cls) {
			const names = String(cls).trim().split(/\s+/).filter(Boolean);
			if (!names.length) return list([], HTMLCollection);
			return list(D.queryAll(this[ID], names.map(c => "." + cssEscape(c)).join("")), HTMLCollection);
		},
		append(...nodes) { for (const n of nodes) this.appendChild(toNode(n)); },
		prepend(...nodes) { const f = this.firstChild; for (const n of nodes) this.insertBefore(toNode(n), f); },
		replaceChildren(...nodes) { D.setText(this[ID], ""); this.append(...nodes); },
	};
	// methods of nodes that are children
	const childNodeMethods = {
		remove() { D.remove(this[ID]); },
		before(...nodes) { const p = this.parentNode; if (p) for (const n of nodes) p.insertBefore(toNode(n), this); },
		after(...nodes) { const p = this.parentNode; if (!p) return; let r = this.nextSibling; for (const n of nodes) p.insertBefore(toNode(n), r); },
		replaceWith(...nodes) { const p = this.parentNode; if (!p) return; const r = this.nextSibling; D.remove(this[ID]); for (const n of nodes) p.insertBefore(toNode(n), r); },
		get nextElementSibling() { for (let n = this.nextSibling; n; n = n.nextSibling) if (n.nodeType === 1) return n; return null; },
		get previousElementSibling() { for (let n = this.previousSibling; n; n = n.previousSibling) if (n.nodeType === 1) return n; return null; },
	};
	function mixin(cls, m) {
		for (const k of Object.getOwnPropertyNames(m)) Object.defineProperty(cls.prototype, k, Object.getOwnPropertyDescriptor(m, k));
	}
	function toNode(n) { return n instanceof Node ? n : document.createTextNode(String(n)); }

	function cssEscape(s) {
		return String(s).replace(/([^\w-])/g, "\\$1").replace(/^(\d)/, "\\3$1 ");
	}

	class CharacterData extends Node {
		get data() { return D.data(this[ID]); }
		set data(v) { D.setData(this[ID], String(v)); }
		get length() { return this.data.length; }
		appendData(s) { this.data += s; }
		substringData(o, n) { return this.data.substr(o, n); }
		replaceData(o, n, s) { const d = this.data; this.data = d.slice(0, o) + s + d.slice(o + n); }
		insertData(o, s) { this.replaceData(o, 0, s); }
		deleteData(o, n) { this.replaceData(o, n, ""); }
	}
	class Text extends CharacterData {
		get wholeText() { return this.data; }
		splitText(o) {
			const d = this.data;
			const t = document.createTextNode(d.slice(o));
			this.data = d.slice(0, o);
			if (this.parentNode) this.parentNode.insertBefore(t, this.nextSibling);
			return t;
		}
	}
	class Comment extends CharacterData {}
	// (never made from HTML, but there for scripts that patch the
	// prototypes of all node types, as the shady DOM polyfill of
	// youtube.com, which stopped at them)
	class CDATASection extends Text {}
	class ProcessingInstruction extends CharacterData {
		get target() { return ""; }
	}
	class DocumentType extends Node {
		get name() { return D.name(this[ID]); }
		get publicId() { return ""; }
		get systemId() { return ""; }
	}
	mixin(CharacterData, childNodeMethods);
	mixin(DocumentType, childNodeMethods);

	// ---- elements ----

	class DOMTokenList {
		constructor(el, attr) { this._el = el; this._attr = attr; }
		_get() { const v = this._el.getAttribute(this._attr); return v ? v.split(/\s+/).filter(Boolean) : []; }
		_set(a) { this._el.setAttribute(this._attr, a.join(" ")); }
		get length() { return this._get().length; }
		get value() { return this._el.getAttribute(this._attr) || ""; }
		set value(v) { this._el.setAttribute(this._attr, v); }
		item(i) { const a = this._get(); return i < a.length ? a[i] : null; }
		contains(t) { return this._get().indexOf(t) >= 0; }
		add(...ts) { const a = this._get(); for (const t of ts) if (a.indexOf(t) < 0) a.push(t); this._set(a); }
		remove(...ts) { const a = this._get().filter(x => ts.indexOf(x) < 0); this._set(a); }
		toggle(t, force) {
			const has = this.contains(t);
			const add = force === undefined ? !has : !!force;
			if (add && !has) this.add(t); else if (!add && has) this.remove(t);
			return add;
		}
		replace(a, b) { const l = this._get(); const i = l.indexOf(a); if (i < 0) return false; l[i] = b; this._set(l); return true; }
		supports() { return true; }
		forEach(f, t) { this._get().forEach((v, i) => f.call(t, v, i, this)); }
		toString() { return this.value; }
		[Symbol.iterator]() { return this._get()[Symbol.iterator](); }
		entries() { return this._get().entries(); }
		keys() { return this._get().keys(); }
		values() { return this._get().values(); }
	}

	// the style attribute of an element as a CSSStyleDeclaration
	function camelToKebab(p) {
		if (p.startsWith("--")) return p;
		return p.replace(/[A-Z]/g, m => "-" + m.toLowerCase()).replace(/^(webkit|moz|ms)-/, "-$1-");
	}
	function parseStyle(s) {
		const out = new Map();
		if (!s) return out;
		for (const decl of s.split(";")) {
			const i = decl.indexOf(":");
			if (i < 0) continue;
			const k = decl.slice(0, i).trim().toLowerCase();
			const v = decl.slice(i + 1).trim();
			if (k) out.set(k, v);
		}
		return out;
	}
	function serializeStyle(m) {
		const parts = [];
		for (const [k, v] of m) parts.push(k + ": " + v + ";");
		return parts.join(" ");
	}
	class CSSStyleDeclaration {
		constructor(el) { this._el = el; }
		getPropertyValue(p) { const m = parseStyle(this._el.getAttribute("style")); return m.get(camelToKebab(p)) || ""; }
		getPropertyPriority() { return ""; }
		setProperty(p, v, prio) {
			p = camelToKebab(String(p));
			const m = parseStyle(this._el.getAttribute("style"));
			if (v === null || v === undefined || v === "") m.delete(p);
			else m.set(p, String(v) + (prio === "important" ? " !important" : ""));
			const s = serializeStyle(m);
			if (s) this._el.setAttribute("style", s); else this._el.removeAttribute("style");
		}
		removeProperty(p) { const v = this.getPropertyValue(p); this.setProperty(p, ""); return v; }
		get cssText() { return this._el.getAttribute("style") || ""; }
		set cssText(v) { if (v) this._el.setAttribute("style", v); else this._el.removeAttribute("style"); }
		get length() { return parseStyle(this._el.getAttribute("style")).size; }
		item(i) { return Array.from(parseStyle(this._el.getAttribute("style")).keys())[i] || ""; }
	}
	// isStyleProperty returns whether the given name is that of a CSS
	// property of a style (as for "opacity" in el.style, by which libraries
	// such as GSAP find the properties that are supported): a camelCase or
	// kebab-case name that is not vendor prefixed, or a custom property.
	function isStyleProperty(p) {
		return typeof p === "string" && (p.startsWith("--") || /^[a-z][a-zA-Z]*$/.test(p) && !/^(webkit|moz|ms|o)[A-Z]/.test(p) || /^[a-z]+(-[a-z]+)+$/.test(p));
	}
	const styleHandler = {
		has(t, p) { return p in t || isStyleProperty(p); },
		get(t, p, r) {
			if (typeof p !== "string" || p in t || p.startsWith("_")) return Reflect.get(t, p, t);
			return t.getPropertyValue(p);
		},
		set(t, p, v, r) {
			if (typeof p !== "string" || p in t || p.startsWith("_")) return Reflect.set(t, p, v, t);
			t.setProperty(p, v);
			return true;
		},
	};
	const styles = new WeakMap();
	function styleOf(el) {
		let s = styles.get(el);
		if (!s) { s = new Proxy(new CSSStyleDeclaration(el), styleHandler); styles.set(el, s); }
		return s;
	}

	function datasetOf(el) {
		const key = p => "data-" + p.replace(/[A-Z]/g, m => "-" + m.toLowerCase());
		return new Proxy({}, {
			get(_, p) { if (typeof p !== "string") return undefined; const v = el.getAttribute(key(p)); return v === null ? undefined : v; },
			set(_, p, v) { el.setAttribute(key(p), String(v)); return true; },
			has(_, p) { return el.hasAttribute(key(p)); },
			deleteProperty(_, p) { el.removeAttribute(key(p)); return true; },
			ownKeys() { return el.getAttributeNames().filter(n => n.startsWith("data-")).map(n => n.slice(5).replace(/-([a-z])/g, (m, c) => c.toUpperCase())); },
			getOwnPropertyDescriptor(_, p) { const v = el.getAttribute(key(p)); return v === null ? undefined : { value: v, enumerable: true, configurable: true, writable: true }; },
		});
	}

	// Attr is an attribute of an element as a node, whose value is that
	// of the attribute of its element (or its own when it has none).
	class Attr {
		constructor(name, owner) { this._name = String(name); this._owner = owner || null; this._value = ""; }
		get name() { return this._name; }
		get localName() { return this._name; }
		get nodeName() { return this._name; }
		get nodeType() { return 2; }
		get namespaceURI() { return null; }
		get prefix() { return null; }
		get specified() { return true; }
		get ownerElement() { return this._owner; }
		get value() { return this._owner ? (this._owner.getAttribute(this._name) ?? "") : this._value; }
		set value(v) { if (this._owner) this._owner.setAttribute(this._name, v); else this._value = String(v); }
		get nodeValue() { return this.value; }
		set nodeValue(v) { this.value = v; }
		get textContent() { return this.value; }
		set textContent(v) { this.value = v; }
	}

	class Element extends Node {
		get tagName() { const n = D.name(this[ID]); return D.ns(this[ID]) ? n : n.toUpperCase(); }
		get localName() { return D.name(this[ID]); }
		get namespaceURI() { return namespaceURIs[D.ns(this[ID])] || namespaceURIs[""]; }
		get prefix() { return null; }
		get id() { return this.getAttribute("id") || ""; }
		set id(v) { this.setAttribute("id", v); }
		get className() { return this.getAttribute("class") || ""; }
		set className(v) { this.setAttribute("class", v); }
		get classList() { return new DOMTokenList(this, "class"); }
		get slot() { return this.getAttribute("slot") || ""; }
		set slot(v) { this.setAttribute("slot", v); }
		getAttribute(n) { return D.attr(this[ID], String(n)); }
		getAttributeNS(ns, n) { return this.getAttribute(n); }
		setAttribute(n, v) { D.setAttr(this[ID], String(n), String(v)); }
		setAttributeNS(ns, n, v) { this.setAttribute(n, v); }
		removeAttribute(n) { D.removeAttr(this[ID], String(n)); }
		removeAttributeNS(ns, n) { this.removeAttribute(n); }
		hasAttribute(n) { return D.attr(this[ID], String(n)) !== null; }
		hasAttributeNS(ns, n) { return this.hasAttribute(n); }
		hasAttributes() { return D.attrs(this[ID]).length > 0; }
		toggleAttribute(n, force) {
			const has = this.hasAttribute(n);
			const add = force === undefined ? !has : !!force;
			if (add && !has) this.setAttribute(n, ""); else if (!add && has) this.removeAttribute(n);
			return add;
		}
		getAttributeNames() { const kv = D.attrs(this[ID]); const a = []; for (let i = 0; i < kv.length; i += 2) a.push(kv[i]); return a; }
		get attributes() {
			const kv = D.attrs(this[ID]);
			const a = [];
			for (let i = 0; i < kv.length; i += 2) a.push(new Attr(kv[i], this));
			a.getNamedItem = n => a.find(x => x.name === String(n).toLowerCase()) || null;
			a.item = i => a[i] || null;
			a.setNamedItem = at => this.setAttributeNode(at);
			a.removeNamedItem = n => { const at = this.getAttributeNode(n); if (!at) throw domException("NotFoundError", "no such attribute"); this.removeAttribute(n); return at; };
			return a;
		}
		// the attributes as nodes (as the key handler of the sign in of
		// Google, which gets the tabindex with getAttributeNode, which
		// was not defined, so that it stopped at its email)
		getAttributeNode(n) { n = String(n).toLowerCase(); return this.hasAttribute(n) ? new Attr(n, this) : null; }
		setAttributeNode(at) {
			const old = this.getAttributeNode(at.name);
			this.setAttribute(at.name, at.value);
			at._owner = this;
			return old;
		}
		removeAttributeNode(at) {
			if (!this.hasAttribute(at.name)) throw domException("NotFoundError", "no such attribute");
			const v = this.getAttribute(at.name);
			this.removeAttribute(at.name);
			const r = new Attr(at.name, null);
			r._value = v;
			return r;
		}
		get style() { return styleOf(this); }
		set style(v) { this.setAttribute("style", v); }
		get dataset() { return datasetOf(this); }
		get innerHTML() { return D.inner(this[ID]); }
		set innerHTML(v) { D.setInner(this[ID], v === null ? "" : String(v)); }
		get outerHTML() { return D.outer(this[ID]); }
		set outerHTML(v) {
			const p = this.parentNode;
			if (!p) return;
			const f = wrap(D.parse(String(v), p[ID]));
			p.insertBefore(f, this);
			D.remove(this[ID]);
		}
		get innerText() { return D.text(this[ID]); }
		set innerText(v) { D.setText(this[ID], String(v)); }
		get outerText() { return this.innerText; }
		insertAdjacentHTML(pos, html) {
			const ctx = pos === "beforebegin" || pos === "afterend" ? this.parentNode : this;
			const f = wrap(D.parse(String(html), ctx ? ctx[ID] : 0));
			this._insertAdjacent(pos, f);
		}
		insertAdjacentElement(pos, el) { this._insertAdjacent(pos, el); return el; }
		insertAdjacentText(pos, t) { this._insertAdjacent(pos, document.createTextNode(t)); }
		_insertAdjacent(pos, n) {
			switch (String(pos).toLowerCase()) {
				case "beforebegin": if (this.parentNode) this.parentNode.insertBefore(n, this); break;
				case "afterbegin": this.insertBefore(n, this.firstChild); break;
				case "beforeend": this.appendChild(n); break;
				case "afterend": if (this.parentNode) this.parentNode.insertBefore(n, this.nextSibling); break;
				default: throw domException("SyntaxError", "invalid position " + pos);
			}
		}
		matches(sel) { return D.matches(this[ID], String(sel)); }
		webkitMatchesSelector(sel) { return this.matches(sel); }
		closest(sel) { for (let n = this; n && n.nodeType === 1; n = n.parentNode) if (n.matches(sel)) return n; return null; }
		getBoundingClientRect() {
			const r = D.rect(this[ID]);
			if (!r) return new DOMRect(0, 0, 0, 0);
			return new DOMRect(r[0] - scrollState.x, r[1] - scrollState.y, r[2], r[3]);
		}
		getClientRects() { const r = D.rect(this[ID]); return r ? [this.getBoundingClientRect()] : []; }
		get offsetWidth() { const r = D.rect(this[ID]); return r ? Math.round(r[2]) : 0; }
		get offsetHeight() { const r = D.rect(this[ID]); return r ? Math.round(r[3]) : 0; }
		get offsetLeft() { const r = D.rect(this[ID]); return r ? Math.round(r[0]) : 0; }
		get offsetTop() { const r = D.rect(this[ID]); return r ? Math.round(r[1]) : 0; }
		// the body, as offsetLeft and offsetTop are of the page, but none for
		// the body and the root element, as in browsers (GitHub walks up
		// the offset parents to null, which never ended at the body)
		get offsetParent() {
			if (this === document.body || this === document.documentElement) return null;
			return D.rect(this[ID]) ? document.body : null;
		}
		// the client size of the root element is that of the viewport, as
		// pages read for their layout (craigslist shows its mobile layout
		// for a width of 0)
		get clientWidth() { return this === document.documentElement ? Math.round(D.viewport()[0]) : this.offsetWidth; }
		get clientHeight() { return this === document.documentElement ? Math.round(D.viewport()[1]) : this.offsetHeight; }
		get clientLeft() { return 0; }
		get clientTop() { return 0; }
		get scrollWidth() { return this.offsetWidth; }
		get scrollHeight() { return this.offsetHeight; }
		get scrollTop() { return 0; }
		set scrollTop(v) {}
		get scrollLeft() { return 0; }
		set scrollLeft(v) {}
		scrollIntoView() {}
		scrollTo() {}
		scrollBy() {}
		scroll() {}
		focus() { focused = this; D.focus(this[ID]); }
		blur() { if (focused === this) focused = null; }
		click() { this.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, button: 0 })); }
		attachShadow(init) {
			if (shadows.has(this)) throw domException("NotSupportedError", "the element already has a shadow root");
			const id = D.createFragment();
			const sr = Object.create(ShadowRoot.prototype);
			Object.defineProperty(sr, ID, { value: id });
			wrappers.set(id, sr);
			shadowHosts.set(id, this[ID]);
			D.setShadow(this[ID], id);
			shadows.set(this, { root: sr, mode: init && init.mode === "closed" ? "closed" : "open" });
			return sr;
		}
		get shadowRoot() { const s = shadows.get(this); return s && s.mode === "open" ? s.root : null; }
		animate() { return { finished: Promise.resolve(), cancel() {}, play() {}, pause() {}, onfinish: null, addEventListener() {} }; }
		getAnimations() { return []; }
		requestFullscreen() { return Promise.reject(domException("NotSupportedError", "fullscreen is not supported")); }
		setPointerCapture() {}
		releasePointerCapture() {}
		hasPointerCapture() { return false; }
	}
	mixin(Element, parentNodeMethods);
	mixin(Element, childNodeMethods);
	defineHandlers(Element.prototype);

	// reflect defines properties of the given class that reflect attributes
	function reflect(cls, props, bools) {
		for (const p of props || []) {
			const a = p.toLowerCase();
			Object.defineProperty(cls.prototype, p, {
				get() { return this.getAttribute(a) || ""; },
				set(v) { this.setAttribute(a, v); },
				configurable: true,
			});
		}
		for (const p of bools || []) {
			const a = p.toLowerCase();
			Object.defineProperty(cls.prototype, p, {
				get() { return this.hasAttribute(a); },
				set(v) { this.toggleAttribute(a, !!v); },
				configurable: true,
			});
		}
	}
	function reflectURL(cls, props) {
		for (const p of props) {
			Object.defineProperty(cls.prototype, p, {
				get() { const v = this.getAttribute(p); if (v === null) return ""; const u = D.parseURL(v, D.baseURL()); return u ? u[0] : v; },
				set(v) { this.setAttribute(p, v); },
				configurable: true,
			});
		}
	}

	// HTMLElement's constructor is called by those of custom elements: it
	// returns the element being upgraded, or a new one (new MyElement()).
	class HTMLElement extends Element {
		constructor() {
			if (upgrading) {
				const w = upgrading;
				upgrading = null;
				Object.setPrototypeOf(w, new.target.prototype);
				return w;
			}
			const name = definedNames.get(new.target);
			if (!name) throw new TypeError("Illegal constructor");
			const id = D.create(name);
			const w = Object.create(new.target.prototype);
			Object.defineProperty(w, ID, { value: id });
			wrappers.set(id, w);
			return w;
		}
	}
	reflect(HTMLElement, ["title", "lang", "dir", "accessKey"], ["hidden", "inert", "draggable", "spellcheck"]);
	Object.defineProperty(HTMLElement.prototype, "tabIndex", {
		get() { const v = parseInt(this.getAttribute("tabindex"), 10); return isNaN(v) ? -1 : v; },
		set(v) { this.setAttribute("tabindex", String(v)); },
	});
	Object.defineProperty(HTMLElement.prototype, "contentEditable", { get() { return this.getAttribute("contenteditable") || "inherit"; }, set(v) { this.setAttribute("contenteditable", v); } });
	Object.defineProperty(HTMLElement.prototype, "isContentEditable", { get() { return this.getAttribute("contenteditable") === "true"; } });

	function elementClass(tags, base, def) {
		const cls = class extends (base || HTMLElement) {};
		if (def) def(cls);
		for (const t of tags) tagClasses[t] = cls;
		return cls;
	}

	// the values of form controls, which are not their attributes once changed
	const values = new WeakMap();
	const checks = new WeakMap();

	const HTMLAnchorElement = elementClass(["a"], HTMLElement, cls => {
		reflectURL(cls, ["href"]);
		reflect(cls, ["target", "rel", "download", "hreflang", "type", "referrerPolicy"]);
		urlParts(cls, function () { return this.href; }, function (v) { this.href = v; });
	});
	const HTMLAreaElement = elementClass(["area"], HTMLElement, cls => { reflectURL(cls, ["href"]); reflect(cls, ["alt", "coords", "shape", "target"]); });
	const HTMLImageElement = elementClass(["img"], HTMLElement, cls => {
		reflectURL(cls, ["src"]);
		reflect(cls, ["alt", "srcset", "sizes", "crossOrigin", "useMap", "loading", "decoding"]);
		Object.defineProperty(cls.prototype, "width", { get() { const r = D.rect(this[ID]); return r ? Math.round(r[2]) : (parseInt(this.getAttribute("width"), 10) || 0); }, set(v) { this.setAttribute("width", String(v)); } });
		Object.defineProperty(cls.prototype, "height", { get() { const r = D.rect(this[ID]); return r ? Math.round(r[3]) : (parseInt(this.getAttribute("height"), 10) || 0); }, set(v) { this.setAttribute("height", String(v)); } });
		Object.defineProperty(cls.prototype, "naturalWidth", { get() { return this.width; } });
		Object.defineProperty(cls.prototype, "naturalHeight", { get() { return this.height; } });
		Object.defineProperty(cls.prototype, "complete", { get() { return true; } });
		Object.defineProperty(cls.prototype, "currentSrc", { get() { return this.src; } });
		cls.prototype.decode = function () { return Promise.resolve(); };
		// Images fire load (once for each source) for the scripts that wait
		// for it, as the page loads them (as the photos of bbc.com, whose
		// placeholders were removed only on it, and covered them).
		const loadedSrc = new WeakMap(); // image -> the source it fired load for
		const sourceOf = img => img.getAttribute("src") || img.getAttribute("srcset") || "";
		const fireLoad = img => {
			const src = sourceOf(img);
			if (!src || loadedSrc.get(img) === src) return;
			loadedSrc.set(img, src);
			setTimeout(() => {
				if (sourceOf(img) !== src) return; // another source since
				const e = new Event("load");
				e.isTrusted = true;
				img.dispatchEvent(e);
			}, 0);
		};
		const addListener = cls.prototype.addEventListener;
		cls.prototype.addEventListener = function (type, ...rest) {
			const r = addListener.call(this, type, ...rest);
			if (type === "load") fireLoad(this);
			return r;
		};
		const srcDesc = Object.getOwnPropertyDescriptor(cls.prototype, "src");
		Object.defineProperty(cls.prototype, "src", {
			get: srcDesc.get,
			set(v) { srcDesc.set.call(this, v); fireLoad(this); }, // (as for img.onload = f; img.src = u)
			configurable: true,
		});
	});
	const HTMLScriptElement = elementClass(["script"], HTMLElement, cls => {
		reflectURL(cls, ["src"]);
		reflect(cls, ["type", "charset", "crossOrigin", "integrity", "referrerPolicy", "nonce"], ["async", "defer", "noModule"]);
		Object.defineProperty(cls.prototype, "text", { get() { return this.textContent; }, set(v) { this.textContent = v; } });
	});
	const HTMLLinkElement = elementClass(["link"], HTMLElement, cls => { reflectURL(cls, ["href"]); reflect(cls, ["rel", "type", "media", "as", "crossOrigin", "integrity"]); Object.defineProperty(cls.prototype, "relList", { get() { return new DOMTokenList(this, "rel"); } }); });
	// CSSStyleSheet has the rules of a style sheet as text. That of a style
	// element changes its text (as CSS-in-JS inserts rules), so that the
	// page is styled by them; constructed ones that a document or shadow
	// root adopts (adoptedStyleSheets, as of Lit, and the dialog of
	// Cookiebot, which was not styled) are applied by a style element of
	// it that has their rules (see syncAdopted).
	function cssBlocks(text) {
		const out = [];
		let depth = 0, start = 0, quote = "";
		for (let i = 0; i < text.length; i++) {
			const c = text[i];
			if (quote) { if (c === "\\") i++; else if (c === quote) quote = ""; continue; }
			if (c === '"' || c === "'") quote = c;
			else if (c === "/" && text[i + 1] === "*") { const e = text.indexOf("*/", i + 2); i = e < 0 ? text.length : e + 1; }
			else if (c === "{") depth++;
			else if (c === "}" && --depth <= 0) { depth = 0; const r = text.slice(start, i + 1).trim(); if (r) out.push(r); start = i + 1; }
			else if (c === ";" && depth === 0) { const r = text.slice(start, i + 1).trim(); if (r) out.push(r); start = i + 1; }
		}
		return out;
	}
	// CSSRule and its subclasses are the rules of style sheets (cssRules),
	// made from the text of each rule, as libraries (and the page scripts of
	// Google) read and extend their prototypes. The style of a style rule
	// sets the text of the rule in its sheet.
	class CSSRule {
		constructor(sheet, index, text) { this._sheet = sheet; this._index = index; this._text = text; }
		get cssText() { return this._text; }
		get parentStyleSheet() { return this._sheet || null; }
		get parentRule() { return null; }
		get type() { return 0; }
	}
	Object.assign(CSSRule, { STYLE_RULE: 1, CHARSET_RULE: 2, IMPORT_RULE: 3, MEDIA_RULE: 4, FONT_FACE_RULE: 5, PAGE_RULE: 6, KEYFRAMES_RULE: 7, KEYFRAME_RULE: 8, NAMESPACE_RULE: 10, SUPPORTS_RULE: 12 });
	// ruleBody returns the text of the block of the given rule, without its braces.
	function ruleBody(text) { const i = text.indexOf("{"); return i < 0 ? "" : text.slice(i + 1, text.lastIndexOf("}")).trim(); }
	// rulePrelude returns the text of the given rule before its block.
	function rulePrelude(text) { const i = text.indexOf("{"); return (i < 0 ? text.replace(/;$/, "") : text.slice(0, i)).trim(); }
	class CSSStyleRule extends CSSRule {
		get type() { return 1; }
		get selectorText() { return rulePrelude(this._text); }
		set selectorText(v) { this._update(String(v) + " { " + ruleBody(this._text) + " }"); }
		get style() {
			if (!this._style) {
				const rule = this;
				const decl = {
					getAttribute() { return ruleBody(rule._text) || null; },
					setAttribute(_, v) { rule._update(rule.selectorText + " { " + v + " }"); },
					removeAttribute() { rule._update(rule.selectorText + " { }"); },
				};
				this._style = new Proxy(new CSSStyleDeclaration(decl), styleHandler);
			}
			return this._style;
		}
		_update(text) {
			this._text = text;
			const sh = this._sheet;
			if (!sh) return;
			const rules = sh._current().slice();
			if (this._index < rules.length) { rules[this._index] = text; sh._set(rules); }
		}
	}
	class CSSGroupingRule extends CSSRule {
		get cssRules() { return cssBlocks(ruleBody(this._text)).map((t, i) => makeRule(null, i, t)); }
		insertRule() { throw domException("NotSupportedError", "rules cannot be inserted into grouping rules"); }
		deleteRule() { throw domException("NotSupportedError", "rules cannot be deleted from grouping rules"); }
	}
	class CSSConditionRule extends CSSGroupingRule {
		get conditionText() { return rulePrelude(this._text).replace(/^@[\w-]+\s*/, ""); }
	}
	class CSSMediaRule extends CSSConditionRule {
		get type() { return 4; }
		get media() { return { mediaText: this.conditionText, length: 1 }; }
	}
	class CSSSupportsRule extends CSSConditionRule { get type() { return 12; } }
	class CSSContainerRule extends CSSConditionRule { get containerName() { return ""; } }
	class CSSLayerBlockRule extends CSSGroupingRule { get name() { return rulePrelude(this._text).replace(/^@layer\s*/, ""); } }
	class CSSImportRule extends CSSRule {
		get type() { return 3; }
		get href() { const m = /url\(\s*["']?([^"')]*)|["']([^"']*)/.exec(rulePrelude(this._text).slice(7)); return m ? (m[1] || m[2] || "") : ""; }
		get media() { return { mediaText: "" }; }
		get styleSheet() { return null; }
	}
	class CSSFontFaceRule extends CSSRule { get type() { return 5; } }
	class CSSPageRule extends CSSRule { get type() { return 6; } }
	class CSSKeyframesRule extends CSSRule {
		get type() { return 7; }
		get name() { return rulePrelude(this._text).replace(/^@[\w-]*keyframes\s*/, ""); }
		get cssRules() { return cssBlocks(ruleBody(this._text)).map((t, i) => new CSSKeyframeRule(null, i, t)); }
	}
	class CSSKeyframeRule extends CSSRule {
		get type() { return 8; }
		get keyText() { return rulePrelude(this._text); }
	}
	class CSSNamespaceRule extends CSSRule { get type() { return 10; } }
	class CSSCharsetRule extends CSSRule { get type() { return 2; } }
	// makeRule returns the rule of the given text, at the given index of the given sheet.
	function makeRule(sheet, index, text) {
		const at = /^@(-[a-z]+-)?([a-z-]+)/i.exec(text);
		const kind = at ? at[2].toLowerCase() : "";
		const cls = !at ? CSSStyleRule : {
			media: CSSMediaRule, supports: CSSSupportsRule, container: CSSContainerRule, layer: text.includes("{") ? CSSLayerBlockRule : CSSRule,
			import: CSSImportRule, "font-face": CSSFontFaceRule, page: CSSPageRule, keyframes: CSSKeyframesRule,
			namespace: CSSNamespaceRule, charset: CSSCharsetRule,
		}[kind] || CSSRule;
		return new cls(sheet, index, text);
	}
	const sheetOwners = new WeakMap(); // style sheet -> style element
	// CSSRuleList is the list of the rules of a style sheet (cssRules),
	// whose rules are made as they are gotten, by index (and its length is
	// that of the rules of the sheet, without making them).
	class CSSRuleList {
		constructor(sheet) { this._sheet = sheet; }
		get length() { return this._sheet._current().length; }
		item(i) { return this._sheet._rule(i >>> 0); }
		*[Symbol.iterator]() { const n = this.length; for (let i = 0; i < n; i++) yield this.item(i); }
	}
	const ruleListHandler = {
		get(t, p, r) {
			if (typeof p === "string" && /^\d+$/.test(p)) return t.item(Number(p)) || undefined;
			return Reflect.get(t, p, t);
		},
		has(t, p) { return typeof p === "string" && /^\d+$/.test(p) ? Number(p) < t.length : p in t; },
	};
	class CSSStyleSheet {
		constructor(opts) { this._rules = []; this._objs = []; this.disabled = false; this.media = { mediaText: (opts && opts.media) || "" }; }
		// _current returns the texts of the rules of the sheet, parsed from
		// the text of its style element again only when the document has
		// changed (nytimes.com read them in loops of its scripts, which ran
		// too long, as they were parsed on each read).
		_current() {
			const o = sheetOwners.get(this);
			if (o && this._at !== D.domChanges()) {
				const text = o.textContent;
				if (text !== this._text) { this._text = text; this._rules = cssBlocks(text); this._objs = []; }
				this._at = D.domChanges();
			}
			return this._rules;
		}
		// _rule returns the rule at the given index, made once.
		_rule(i) {
			const rules = this._current();
			if (i >= rules.length) return null;
			return this._objs[i] || (this._objs[i] = makeRule(this, i, rules[i]));
		}
		get cssRules() { return this._list || (this._list = new Proxy(new CSSRuleList(this), ruleListHandler)); }
		get rules() { return this.cssRules; }
		get ownerNode() { return sheetOwners.get(this) || null; }
		_set(rules) {
			this._rules = rules;
			this._objs = [];
			const o = sheetOwners.get(this);
			if (o) {
				o.textContent = rules.join("\n");
				this._text = undefined; // (the rules are current)
				this._at = D.domChanges();
			}
			if (this._adopters) for (const owner of this._adopters) syncAdopted(owner);
		}
		insertRule(rule, index) {
			const rules = this._current();
			index = index === undefined ? 0 : index >>> 0;
			if (index > rules.length) throw domException("IndexSizeError", "the index is beyond the rules");
			rule = String(rule);
			const o = sheetOwners.get(this);
			if (o && (index === rules.length || index === 0)) {
				// a rule at the end (or the start) is a text of its own in the
				// element, not all of its text again (as the rules that
				// emotion inserts one by one, which took too long as each
				// wrote the text of all of them)
				if (index === rules.length) {
					o.appendChild(document.createTextNode("\n" + rule));
					rules.push(rule);
				} else {
					o.insertBefore(document.createTextNode(rule + "\n"), o.firstChild);
					rules.unshift(rule);
				}
				this._objs = [];
				this._text = undefined;
				this._at = D.domChanges();
				return index;
			}
			const all = rules.slice();
			all.splice(index, 0, rule);
			this._set(all);
			return index;
		}
		deleteRule(index) { const rules = this._current().slice(); rules.splice(index >>> 0, 1); this._set(rules); }
		addRule(sel, style, index) { return this.insertRule(sel + "{" + style + "}", index === undefined ? this._current().length : index); }
		removeRule(index) { this.deleteRule(index || 0); }
		replaceSync(text) { this._set(cssBlocks(String(text))); }
		replace(text) { this.replaceSync(text); return Promise.resolve(this); }
	}
	// setAdopted sets the style sheets that the given document or shadow
	// root adopts (adoptedStyleSheets), which are applied (see syncAdopted).
	function setAdopted(owner, v) {
		for (const sh of owner._adopted || []) if (sh._adopters) sh._adopters.delete(owner);
		owner._adopted = Array.from(v);
		for (const sh of owner._adopted) (sh._adopters || (sh._adopters = new Set())).add(owner);
		syncAdopted(owner);
	}
	// syncAdopted applies the style sheets that the given document or shadow
	// root has adopted, by the text of a style element of it that has their
	// rules (after the others of a document, in its head).
	function syncAdopted(owner) {
		const text = (owner._adopted || []).map(sh => sh._rules.join("\n")).join("\n");
		let el = owner._adoptedStyle;
		if (!el) {
			if (!text) return;
			el = document.createElement("style");
			el.setAttribute("data-adopted-style-sheets", "");
			owner._adoptedStyle = el;
		}
		const parent = owner === document ? (document.head || document.documentElement) : owner;
		if (el.parentNode !== parent && parent) parent.appendChild(el);
		if (el.textContent !== text) el.textContent = text;
	}
	const styleSheets = new WeakMap(); // style element -> its sheet
	const HTMLStyleElement = elementClass(["style"], HTMLElement, cls => {
		reflect(cls, ["media", "type"]);
		Object.defineProperty(cls.prototype, "sheet", {
			get() {
				if (!this.isConnected) return null;
				let sh = styleSheets.get(this);
				if (!sh) { sh = new CSSStyleSheet(); sheetOwners.set(sh, this); styleSheets.set(this, sh); }
				return sh;
			},
		});
	});
	const HTMLMetaElement = elementClass(["meta"], HTMLElement, cls => reflect(cls, ["name", "content", "httpEquiv", "charset"]));
	const HTMLBaseElement = elementClass(["base"], HTMLElement, cls => { reflectURL(cls, ["href"]); reflect(cls, ["target"]); });
	const HTMLIFrameElement = elementClass(["iframe"], HTMLElement, cls => {
		reflectURL(cls, ["src"]); reflect(cls, ["name", "width", "height", "allow", "loading", "srcdoc", "referrerPolicy"]);
		// the restrictions of its page, as a token list (Google adds them)
		Object.defineProperty(cls.prototype, "sandbox", { get() { return new DOMTokenList(this, "sandbox"); }, set(v) { this.setAttribute("sandbox", String(v)); } });
		// the window of the page in its frame, a proxy that can only post
		// messages to it (see frames.go), as for one of another origin
		Object.defineProperty(cls.prototype, "contentWindow", { get() { return windowProxy(D.frameWindow(this[ID])); } });
		Object.defineProperty(cls.prototype, "contentDocument", { get() { return null; } });
	});

	// changedState marks the element as changed, so that the page is
	// rendered again (as for the value of a control, which is not an
	// attribute).
	function changedState(el) {
		D.setAttr(el[ID], "data-cc-state", ""); D.removeAttr(el[ID], "data-cc-state");
	}

	function controlValue(cls, def) {
		Object.defineProperty(cls.prototype, "value", {
			get() { return values.has(this) ? values.get(this) : def.call(this); },
			set(v) { values.set(this, v === null || v === undefined ? "" : String(v)); changedState(this); },
			configurable: true,
		});
		Object.defineProperty(cls.prototype, "form", { get() { return this.closest("form"); } });
		reflect(cls, ["name", "autocomplete"], ["disabled", "required", "autofocus", "readOnly"]);
		Object.defineProperty(cls.prototype, "validity", { get() { return { valid: true, valueMissing: false, typeMismatch: false }; } });
		cls.prototype.checkValidity = function () { return true; };
		cls.prototype.reportValidity = function () { return true; };
		cls.prototype.setCustomValidity = function () {};
		Object.defineProperty(cls.prototype, "willValidate", { get() { return true; } });
		Object.defineProperty(cls.prototype, "labels", { get() { const id = this.id; return id ? document.querySelectorAll("label[for='" + id + "']") : list([]); } });
	}

	const HTMLInputElement = elementClass(["input"], HTMLElement, cls => {
		controlValue(cls, function () { return this.getAttribute("value") || (this.type === "checkbox" || this.type === "radio" ? "on" : ""); });
		reflect(cls, ["placeholder", "pattern", "min", "max", "step", "accept", "list", "inputMode", "alt"], ["multiple"]);
		reflectURL(cls, ["src"]);
		Object.defineProperty(cls.prototype, "type", {
			get() { return (this.getAttribute("type") || "text").toLowerCase(); },
			set(v) { this.setAttribute("type", v); },
		});
		Object.defineProperty(cls.prototype, "checked", {
			get() { return checks.has(this) ? checks.get(this) : this.hasAttribute("checked"); },
			set(v) {
				v = !!v;
				checks.set(this, v);
				if (v && this.type === "radio" && this.name) {
					const scope = this.form || document;
					for (const r of scope.querySelectorAll("input[type=radio]")) if (r !== this && r.name === this.name) checks.set(r, false);
				}
				changedState(this);
			},
		});
		Object.defineProperty(cls.prototype, "defaultValue", { get() { return this.getAttribute("value") || ""; }, set(v) { this.setAttribute("value", v); } });
		Object.defineProperty(cls.prototype, "defaultChecked", { get() { return this.hasAttribute("checked"); }, set(v) { this.toggleAttribute("checked", !!v); } });
		Object.defineProperty(cls.prototype, "valueAsNumber", { get() { return parseFloat(this.value); }, set(v) { this.value = String(v); } });
		Object.defineProperty(cls.prototype, "maxLength", { get() { const v = parseInt(this.getAttribute("maxlength"), 10); return isNaN(v) ? -1 : v; }, set(v) { this.setAttribute("maxlength", String(v)); } });
		Object.defineProperty(cls.prototype, "size", { get() { return parseInt(this.getAttribute("size"), 10) || 20; }, set(v) { this.setAttribute("size", String(v)); } });
		cls.prototype.select = function () {};
		cls.prototype.setSelectionRange = function () {};
		Object.defineProperty(cls.prototype, "selectionStart", { get() { return this.value.length; }, set(v) {} });
		Object.defineProperty(cls.prototype, "selectionEnd", { get() { return this.value.length; }, set(v) {} });
		cls.prototype.click = function () {
			const t = this.type;
			if (t === "checkbox") this.checked = !this.checked;
			else if (t === "radio") this.checked = true;
			const ok = this.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true }));
			if (t === "checkbox" || t === "radio") {
				this.dispatchEvent(new Event("input", { bubbles: true }));
				this.dispatchEvent(new Event("change", { bubbles: true }));
			} else if (ok && (t === "submit" || t === "image") && this.form) this.form.requestSubmit(this);
			else if (ok && t === "reset" && this.form) this.form.reset();
		};
	});
	const HTMLTextAreaElement = elementClass(["textarea"], HTMLElement, cls => {
		controlValue(cls, function () { return this.textContent; });
		reflect(cls, ["placeholder", "wrap"]);
		Object.defineProperty(cls.prototype, "type", { get() { return "textarea"; } });
		Object.defineProperty(cls.prototype, "defaultValue", { get() { return this.textContent; }, set(v) { this.textContent = v; } });
		Object.defineProperty(cls.prototype, "rows", { get() { return parseInt(this.getAttribute("rows"), 10) || 2; }, set(v) { this.setAttribute("rows", String(v)); } });
		Object.defineProperty(cls.prototype, "cols", { get() { return parseInt(this.getAttribute("cols"), 10) || 20; }, set(v) { this.setAttribute("cols", String(v)); } });
		cls.prototype.select = function () {};
		cls.prototype.setSelectionRange = function () {};
	});
	// the classes of the other elements, which scripts use (as the select
	// panel of the pull requests of GitHub, which checks for
	// HTMLOptGroupElement), with their attributes
	const HTMLOptGroupElement = elementClass(["optgroup"], HTMLElement, cls => { reflect(cls, ["label"], ["disabled"]); });
	const HTMLLegendElement = elementClass(["legend"], HTMLElement);
	const HTMLProgressElement = elementClass(["progress"], HTMLElement, cls => {
		Object.defineProperty(cls.prototype, "value", { get() { return parseFloat(this.getAttribute("value")) || 0; }, set(v) { this.setAttribute("value", String(v)); } });
		Object.defineProperty(cls.prototype, "max", { get() { return parseFloat(this.getAttribute("max")) || 1; }, set(v) { this.setAttribute("max", String(v)); } });
	});
	const HTMLMeterElement = elementClass(["meter"], HTMLElement, cls => {
		for (const k of ["value", "min", "max", "low", "high", "optimum"]) {
			Object.defineProperty(cls.prototype, k, { get() { return parseFloat(this.getAttribute(k)) || 0; }, set(v) { this.setAttribute(k, String(v)); } });
		}
	});
	const HTMLOutputElement = elementClass(["output"], HTMLElement, cls => {
		reflect(cls, ["name"]);
		Object.defineProperty(cls.prototype, "value", { get() { return this.textContent; }, set(v) { this.textContent = v; } });
	});
	const HTMLSourceElement = elementClass(["source"], HTMLElement, cls => { reflectURL(cls, ["src"]); reflect(cls, ["srcset", "sizes", "media", "type"]); });
	const HTMLPictureElement = elementClass(["picture"], HTMLElement);
	const HTMLTrackElement = elementClass(["track"], HTMLElement, cls => { reflectURL(cls, ["src"]); reflect(cls, ["kind", "srclang", "label"], ["default"]); });
	const HTMLTimeElement = elementClass(["time"], HTMLElement, cls => { reflect(cls, ["dateTime"]); });
	const HTMLDataElement = elementClass(["data"], HTMLElement, cls => { reflect(cls, ["value"]); });
	const HTMLDataListElement = elementClass(["datalist"], HTMLElement, cls => {
		Object.defineProperty(cls.prototype, "options", { get() { return this.querySelectorAll("option"); } });
	});
	const HTMLMapElement = elementClass(["map"], HTMLElement, cls => { reflect(cls, ["name"]); });
	const HTMLQuoteElement = elementClass(["q", "blockquote"], HTMLElement, cls => { reflectURL(cls, ["cite"]); });
	const HTMLModElement = elementClass(["ins", "del"], HTMLElement, cls => { reflectURL(cls, ["cite"]); reflect(cls, ["dateTime"]); });
	const HTMLTableSectionElement = elementClass(["thead", "tbody", "tfoot"], HTMLElement, cls => {
		Object.defineProperty(cls.prototype, "rows", { get() { return Array.from(this.children).filter(c => c.localName === "tr"); } });
	});
	const HTMLTableCaptionElement = elementClass(["caption"], HTMLElement);
	const HTMLTableColElement = elementClass(["col", "colgroup"], HTMLElement, cls => {
		Object.defineProperty(cls.prototype, "span", { get() { return parseInt(this.getAttribute("span"), 10) || 1; }, set(v) { this.setAttribute("span", String(v)); } });
	});
	const HTMLTitleElement = elementClass(["title"], HTMLElement, cls => {
		Object.defineProperty(cls.prototype, "text", { get() { return this.textContent; }, set(v) { this.textContent = v; } });
	});
	const HTMLMenuElement = elementClass(["menu"], HTMLElement);
	const HTMLDListElement = elementClass(["dl"], HTMLElement);
	const HTMLEmbedElement = elementClass(["embed"], HTMLElement, cls => { reflectURL(cls, ["src"]); reflect(cls, ["type", "width", "height"]); });
	const HTMLObjectElement = elementClass(["object"], HTMLElement, cls => { reflectURL(cls, ["data"]); reflect(cls, ["type", "name", "width", "height"]); });
	const HTMLParamElement = elementClass(["param"], HTMLElement, cls => { reflect(cls, ["name", "value"]); });
	const HTMLSlotElement = elementClass(["slot"], HTMLElement, cls => { reflect(cls, ["name"]); });
	const HTMLFontElement = elementClass(["font"], HTMLElement, cls => { reflect(cls, ["color", "face", "size"]); });
	const HTMLOptionElement = elementClass(["option"], HTMLElement, cls => {
		Object.defineProperty(cls.prototype, "value", { get() { const v = this.getAttribute("value"); return v === null ? this.textContent.trim() : v; }, set(v) { this.setAttribute("value", v); } });
		Object.defineProperty(cls.prototype, "text", { get() { return this.textContent.trim(); }, set(v) { this.textContent = v; } });
		Object.defineProperty(cls.prototype, "label", { get() { return this.getAttribute("label") || this.text; }, set(v) { this.setAttribute("label", v); } });
		Object.defineProperty(cls.prototype, "selected", {
			get() { const s = this.closest("select"); return s ? s.selectedIndex === this.index : this.hasAttribute("selected"); },
			set(v) { const s = this.closest("select"); if (s && v) s.value = this.value; else this.toggleAttribute("selected", !!v); },
		});
		Object.defineProperty(cls.prototype, "index", { get() { const s = this.closest("select"); return s ? Array.prototype.indexOf.call(s.options, this) : 0; } });
		reflect(cls, [], ["disabled", "defaultSelected"]);
	});
	const HTMLSelectElement = elementClass(["select"], HTMLElement, cls => {
		controlValue(cls, function () { const o = this.options[this.selectedIndex]; return o ? o.value : ""; });
		reflect(cls, [], ["multiple"]);
		Object.defineProperty(cls.prototype, "options", { get() { const a = this.querySelectorAll("option"); a.namedItem = n => a.find(o => o.id === n || o.getAttribute("name") === n) || null; a.add = (o, b) => this.add(o, b); a.remove = i => this.remove(i); return a; } });
		Object.defineProperty(cls.prototype, "type", { get() { return this.multiple ? "select-multiple" : "select-one"; } });
		Object.defineProperty(cls.prototype, "length", { get() { return this.options.length; } });
		Object.defineProperty(cls.prototype, "selectedIndex", {
			get() {
				const opts = this.options;
				if (values.has(this)) { const v = values.get(this); for (let i = 0; i < opts.length; i++) if (opts[i].value === v) return i; return -1; }
				for (let i = 0; i < opts.length; i++) if (opts[i].hasAttribute("selected")) return i;
				return opts.length ? 0 : -1;
			},
			set(i) { const o = this.options[i]; this.value = o ? o.value : ""; },
		});
		Object.defineProperty(cls.prototype, "selectedOptions", { get() { const i = this.selectedIndex; return i >= 0 ? list([this.options[i][ID]]) : list([]); } });
		cls.prototype.add = function (o, before) { this.insertBefore(o, typeof before === "number" ? this.options[before] || null : before || null); };
		cls.prototype.remove = function (i) { if (i === undefined) { D.remove(this[ID]); return; } const o = this.options[i]; if (o) o.remove(); };
		cls.prototype.item = function (i) { return this.options[i] || null; };
	});
	const HTMLButtonElement = elementClass(["button"], HTMLElement, cls => {
		reflect(cls, ["name", "value"], ["disabled", "autofocus"]);
		Object.defineProperty(cls.prototype, "type", { get() { return (this.getAttribute("type") || "submit").toLowerCase(); }, set(v) { this.setAttribute("type", v); } });
		Object.defineProperty(cls.prototype, "form", { get() { return this.closest("form"); } });
		cls.prototype.click = function () {
			const ok = this.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true }));
			if (ok && this.type === "submit" && this.form) this.form.requestSubmit(this);
			else if (ok && this.type === "reset" && this.form) this.form.reset();
		};
	});
	const HTMLFormElement = elementClass(["form"], HTMLElement, cls => {
		reflect(cls, ["name", "method", "enctype", "target", "acceptCharset", "autocomplete"], ["noValidate"]);
		reflectURL(cls, ["action"]);
		Object.defineProperty(cls.prototype, "elements", {
			get() {
				const a = this.querySelectorAll("input, select, textarea, button, fieldset, output");
				a.namedItem = n => a.find(el => el.id === n || el.getAttribute("name") === n) || null;
				return a;
			},
		});
		Object.defineProperty(cls.prototype, "length", { get() { return this.elements.length; } });
		cls.prototype.submit = function () { submitForm(this, null); };
		cls.prototype.requestSubmit = function (submitter) {
			if (this.dispatchEvent(new SubmitEvent("submit", { bubbles: true, cancelable: true, submitter: submitter || null }))) submitForm(this, submitter);
		};
		cls.prototype.reset = function () {
			if (!this.dispatchEvent(new Event("reset", { bubbles: true, cancelable: true }))) return;
			for (const el of this.elements) { values.delete(el); checks.delete(el); }
			changedState(this);
		};
		cls.prototype.checkValidity = function () { return true; };
		cls.prototype.reportValidity = function () { return true; };
	});
	const HTMLLabelElement = elementClass(["label"], HTMLElement, cls => {
		Object.defineProperty(cls.prototype, "htmlFor", { get() { return this.getAttribute("for") || ""; }, set(v) { this.setAttribute("for", v); } });
		Object.defineProperty(cls.prototype, "control", { get() { const f = this.htmlFor; return f ? document.getElementById(f) : this.querySelector("input, select, textarea, button"); } });
	});
	// The content of a template is one fragment (its children, moved into
	// it when it is first used), as in browsers: Lit prepares the content
	// (removing the markers of its bindings) and then clones it, and with a
	// new fragment each time, the clones kept the markers and the bindings
	// were lost (the menus of MDN were all open).
	const templateContents = new WeakMap();
	const HTMLTemplateElement = elementClass(["template"], HTMLElement, cls => {
		Object.defineProperty(cls.prototype, "content", {
			get() {
				let f = templateContents.get(this);
				if (!f) {
					f = document.createDocumentFragment();
					while (this.firstChild) f.appendChild(this.firstChild);
					templateContents.set(this, f);
				}
				return f;
			},
		});
		Object.defineProperty(cls.prototype, "innerHTML", {
			get() { return D.inner(this.content[ID]); },
			set(v) { D.setInner(this.content[ID], v === null ? "" : String(v)); },
			configurable: true,
		});
	});
	// The 2D context of a canvas draws nothing (the canvas is not rendered),
	// but scripts that draw on one work (as the QR code of the postings of
	// craigslist.org, whose script stopped with no context); other contexts
	// (as webgl) are not supported, which scripts check for.
	class CanvasRenderingContext2D {
		constructor(canvas) {
			this.canvas = canvas;
			Object.assign(this, {
				fillStyle: "#000000", strokeStyle: "#000000", lineWidth: 1, lineCap: "butt", lineJoin: "miter",
				miterLimit: 10, font: "10px sans-serif", textAlign: "start", textBaseline: "alphabetic",
				globalAlpha: 1, globalCompositeOperation: "source-over", shadowBlur: 0, shadowColor: "rgba(0, 0, 0, 0)",
				shadowOffsetX: 0, shadowOffsetY: 0, imageSmoothingEnabled: true, direction: "ltr", filter: "none",
			});
		}
		measureText(text) {
			const size = parseFloat(/(\d+(\.\d+)?)px/.exec(this.font)?.[1] || "10");
			const width = String(text).length * size * 0.5;
			return { width, actualBoundingBoxLeft: 0, actualBoundingBoxRight: width, actualBoundingBoxAscent: size * 0.8, actualBoundingBoxDescent: size * 0.2, fontBoundingBoxAscent: size * 0.8, fontBoundingBoxDescent: size * 0.2 };
		}
		createImageData(w, h) {
			if (typeof w === "object") { h = w.height; w = w.width; }
			return { width: w, height: h, data: new Uint8ClampedArray(Math.max(0, w * h * 4)) };
		}
		getImageData(x, y, w, h) { return this.createImageData(w, h); }
		createLinearGradient() { return { addColorStop() {} }; }
		createRadialGradient() { return { addColorStop() {} }; }
		createConicGradient() { return { addColorStop() {} }; }
		createPattern() { return {}; }
		getTransform() { return new DOMMatrixStub(); }
		getLineDash() { return []; }
		isPointInPath() { return false; }
		isPointInStroke() { return false; }
		getContextAttributes() { return { alpha: true }; }
	}
	for (const m of ["fillRect", "strokeRect", "clearRect", "fillText", "strokeText", "beginPath", "closePath", "moveTo", "lineTo",
		"bezierCurveTo", "quadraticCurveTo", "arc", "arcTo", "ellipse", "rect", "roundRect", "fill", "stroke", "clip", "save", "restore",
		"scale", "rotate", "translate", "transform", "setTransform", "resetTransform", "drawImage", "putImageData", "setLineDash", "drawFocusIfNeeded", "reset"]) {
		CanvasRenderingContext2D.prototype[m] = function () {};
	}
	g.CanvasRenderingContext2D = CanvasRenderingContext2D;
	const HTMLCanvasElement = elementClass(["canvas"], HTMLElement, cls => {
		const contexts = new WeakMap();
		cls.prototype.getContext = function (type) {
			if (String(type).toLowerCase() !== "2d") return null;
			if (!contexts.has(this)) contexts.set(this, new CanvasRenderingContext2D(this));
			return contexts.get(this);
		};
		cls.prototype.toDataURL = function () { return "data:,"; };
		reflect(cls, ["width", "height"]);
	});
	// OffscreenCanvas is a canvas that is not in a document, with a 2D
	// context as that of canvas elements (which draws nothing).
	class OffscreenCanvas extends EventTarget {
		constructor(width, height) { super(); this.width = width >>> 0; this.height = height >>> 0; }
		getContext(type) {
			if (String(type).toLowerCase() !== "2d") return null;
			if (!this._ctx) this._ctx = new CanvasRenderingContext2D(this);
			return this._ctx;
		}
		convertToBlob(opts) { return Promise.resolve(new Blob([], { type: (opts && opts.type) || "image/png" })); }
		transferToImageBitmap() { return { width: this.width, height: this.height, close() {} }; }
	}
	// BroadcastChannel sends messages to the other channels of its name in
	// the page (the only one of its origin here), asynchronously.
	const broadcastChannels = new Set();
	class BroadcastChannel extends EventTarget {
		constructor(name) { super(); this.name = String(name); this._closed = false; this.onmessage = null; this.onmessageerror = null; broadcastChannels.add(this); }
		postMessage(data) {
			if (this._closed) throw domException("InvalidStateError", "the channel is closed");
			const msg = cloneJSON(data);
			for (const c of broadcastChannels) {
				if (c === this || c.name !== this.name) continue;
				setTimeout(() => {
					if (c._closed) return;
					const ev = new MessageEvent("message", { data: cloneDecode(JSON.parse(msg)), origin: location.origin });
					if (typeof c.onmessage === "function") c.onmessage(ev);
					c.dispatchEvent(ev);
				}, 0);
			}
		}
		close() { this._closed = true; broadcastChannels.delete(this); }
	}
	// Media elements (video and audio): the video elements are played by
	// the player of the app (D.hasMediaPlayer, see MediaPlayer in Go), which
	// sends the events of their playing (D.mediaEvents) with its state. The
	// others (and all without a player) are not played, but the metadata
	// (duration and size) of their source is loaded (D.mediaMetadata) when
	// a script first uses their state or listens to their events, and when
	// their source changes, as browsers do for preload="metadata": pages
	// wait for it (as for the chapters of the video of github.com). Their
	// play() is rejected as when autoplay is not allowed.
	const mediaPlayer = D.hasMediaPlayer();
	const mediaEvents = new Set(["loadstart", "durationchange", "loadedmetadata", "loadeddata", "canplay",
		"canplaythrough", "error", "emptied", "progress", "suspend", "abort"]);
	const mediaStates = { HAVE_NOTHING: 0, HAVE_METADATA: 1, HAVE_CURRENT_DATA: 2, HAVE_FUTURE_DATA: 3, HAVE_ENOUGH_DATA: 4,
		NETWORK_EMPTY: 0, NETWORK_IDLE: 1, NETWORK_LOADING: 2, NETWORK_NO_SOURCE: 3 };
	// TrackList is a list of the tracks of a media element (TextTrackList,
	// AudioTrackList, VideoTrackList).
	class TrackList extends EventTarget {
		constructor() { super(); this._list = []; }
		get length() { return this._list.length; }
		getTrackById(id) { return this._list.find(t => t.id === id) || null; }
		_add(t) {
			this[this._list.length] = t;
			this._list.push(t);
			const ev = new Event("addtrack");
			ev.track = t;
			this.dispatchEvent(ev);
		}
		[Symbol.iterator]() { return this._list[Symbol.iterator](); }
	}
	defineSimpleHandlers(TrackList, ["addtrack", "removetrack", "change"]);
	class TextTrack extends EventTarget {
		constructor(kind, label, language) {
			super();
			Object.assign(this, { kind: String(kind || "subtitles"), label: String(label || ""), language: String(language || ""), id: "", mode: "hidden", inBandMetadataTrackDispatchType: "" });
			this.cues = [];
			this.activeCues = [];
		}
		addCue(c) { this.cues.push(c); }
		removeCue(c) { this.cues = this.cues.filter(x => x !== c); }
	}
	defineSimpleHandlers(TextTrack, ["cuechange"]);

	const HTMLMediaElement = elementClass(["video", "audio"], HTMLElement, cls => {
		reflectURL(cls, ["src"]);
		reflect(cls, ["preload", "crossOrigin"], ["autoplay", "controls", "loop", "playsInline"]);
		// played by the player of the app
		cls.prototype._played = function () { return mediaPlayer && this.localName === "video"; };
		Object.assign(cls, mediaStates);
		Object.assign(cls.prototype, mediaStates);
		const srcDesc = Object.getOwnPropertyDescriptor(cls.prototype, "src");
		Object.defineProperty(cls.prototype, "src", {
			get: srcDesc.get,
			set(v) { srcDesc.set.call(this, v); this._mediaLoad(); },
			configurable: true,
		});
		// the media state of the element
		cls.prototype._media = function () {
			if (!this.__media) {
				this.__media = { src: null, readyState: 0, networkState: 0, duration: NaN, width: 0, height: 0, error: null, currentTime: 0, volume: 1, rate: 1,
					paused: true, ended: false, muted: this.hasAttribute("muted") };
			}
			return this.__media;
		};
		// the URL of the source of the element: its src, or that of its first source element
		cls.prototype._mediaSrc = function () {
			if (this.hasAttribute("src")) return this.src;
			const s = this.querySelector("source[src]");
			return s ? new URL(s.getAttribute("src"), document.baseURI).href : "";
		};
		// _mediaLoad loads the metadata of the source of the element if it
		// has changed (or if force).
		cls.prototype._mediaLoad = function (force) {
			const m = this._media();
			const src = this._mediaSrc();
			if (src === m.src && !force) return;
			const had = m.src;
			Object.assign(m, { src, readyState: 0, duration: NaN, width: 0, height: 0, error: null, currentTime: 0 });
			if (had) this.dispatchEvent(new Event("emptied"));
			if (!src) {
				m.networkState = had === null ? 0 : 3;
				return;
			}
			m.networkState = 2;
			Promise.resolve().then(() => { if (m.src === src) this.dispatchEvent(new Event("loadstart")); });
			if (this._played()) { // (its events are sent by the player)
				const ms = mediaSourcesByURL.get(src);
				if (ms) ms._attach(this);
				D.mediaLoad(this[ID]);
				return;
			}
			D.mediaMetadata(newCallback((duration, width, height, err) => {
				if (m.src !== src) return; // another source since
				if (err) {
					m.networkState = 3;
					m.error = new MediaError(MediaError.MEDIA_ERR_SRC_NOT_SUPPORTED, String(err));
					this.dispatchEvent(new Event("error"));
					return;
				}
				Object.assign(m, { duration, width, height, readyState: 1, networkState: 1 });
				this.dispatchEvent(new Event("durationchange"));
				this.dispatchEvent(new Event("loadedmetadata"));
				this.dispatchEvent(new Event("suspend"));
			}), src);
		};
		const state = (name, f) => Object.defineProperty(cls.prototype, name, {
			get() { this._mediaLoad(); return f(this._media()); },
			configurable: true,
		});
		state("readyState", m => m.readyState);
		state("networkState", m => m.networkState);
		state("duration", m => m.duration);
		state("videoWidth", m => m.width);
		state("videoHeight", m => m.height);
		state("error", m => m.error);
		state("currentSrc", m => m.src || "");
		const noRange = () => { throw domException("IndexSizeError", "no time range"); };
		const ranges = { length: 0, start: noRange, end: noRange };
		for (const p of ["played", "seekable"]) Object.defineProperty(cls.prototype, p, { get() { return ranges; } });
		// the buffered ranges of a MediaSource
		Object.defineProperty(cls.prototype, "buffered", {
			get() {
				const ms = this.__mediaSource;
				return ms && ms._url === this._media().src ? timeRanges(ms._buffered()) : ranges;
			},
		});
		Object.defineProperty(cls.prototype, "currentTime", {
			get() {
				// (the time of the player as it plays, between timeupdates)
				const m = this._media();
				if (this._played() && !m.paused) {
					const t = D.mediaTime(this[ID]);
					if (t >= 0) m.currentTime = t;
				}
				return m.currentTime;
			},
			set(v) {
				const m = this._media();
				m.currentTime = Math.max(0, Number(v) || 0);
				if (Number.isFinite(m.duration)) m.currentTime = Math.min(m.currentTime, m.duration);
				if (this._played()) {
					D.mediaSeek(this[ID], m.currentTime);
					m.ended = false;
					this.dispatchEvent(new Event("seeking"));
					Promise.resolve().then(() => this.dispatchEvent(new Event("seeked")));
				}
				this.dispatchEvent(new Event("timeupdate"));
			},
		});
		const setVolume = (el, volume, muted) => {
			const m = el._media();
			if (m.volume === volume && m.muted === muted) return;
			Object.assign(m, { volume, muted });
			if (el._played()) D.mediaVolume(el[ID], volume, muted);
			el.dispatchEvent(new Event("volumechange"));
		};
		Object.defineProperty(cls.prototype, "volume", {
			get() { return this._media().volume; },
			set(v) {
				v = Number(v);
				if (!(v >= 0 && v <= 1)) throw domException("IndexSizeError", "the volume is not between 0 and 1");
				setVolume(this, v, this._media().muted);
			},
		});
		Object.defineProperty(cls.prototype, "muted", { get() { return this._media().muted; }, set(v) { setVolume(this, this._media().volume, !!v); } });
		Object.defineProperty(cls.prototype, "defaultMuted", {
			get() { return this.hasAttribute("muted"); },
			set(v) { if (v) this.setAttribute("muted", ""); else this.removeAttribute("muted"); },
		});
		Object.defineProperty(cls.prototype, "playbackRate", { get() { return this._media().rate; }, set(v) { this._media().rate = Number(v); this.dispatchEvent(new Event("ratechange")); } });
		Object.defineProperty(cls.prototype, "defaultPlaybackRate", { get() { return 1; }, set() {} });
		Object.defineProperty(cls.prototype, "paused", { get() { return this._media().paused; } });
		Object.defineProperty(cls.prototype, "ended", { get() { return this._media().ended; } });
		Object.defineProperty(cls.prototype, "seeking", { get() { return false; } });
		cls.prototype.play = function () {
			this._mediaLoad();
			if (!this._played()) return Promise.reject(domException("NotAllowedError", "media playback is not supported"));
			const err = D.mediaPlay(this[ID]);
			if (err) return Promise.reject(domException("NotSupportedError", err));
			const m = this._media();
			if (m.paused) { // (the player sends play and playing)
				m.paused = false;
				m.ended = false;
			}
			return Promise.resolve();
		};
		cls.prototype.pause = function () {
			if (!this._played()) return;
			const m = this._media();
			D.mediaPause(this[ID]);
			m.paused = true; // (the player sends pause)
		};
		cls.prototype.load = function () { this._mediaLoad(true); };
		// the types of media that can be played: with the player (FFmpeg),
		// also WebM, QuickTime and Ogg, and HLS (as in Safari), which players
		// of web pages (as video.js) then set as the source (instead of MSE,
		// MediaSource, which is not supported)
		cls.prototype.canPlayType = function (t) {
			t = String(t).toLowerCase();
			if (/^(video|audio)\/mp4\b/.test(t)) return "maybe";
			if (!this._played()) return "";
			return /^(video\/(webm|quicktime|ogg)|audio\/(webm|ogg|mpeg|mp3|aac|wav)|application\/(vnd\.apple\.mpegurl|x-mpegurl)|audio\/(x-)?mpegurl)\b/.test(t) ? "maybe" : "";
		};
		cls.prototype.fastSeek = function (t) { this.currentTime = t; };
		// the lists of tracks of the media (text tracks, as of captions, are
		// not shown; the audio and video tracks are those of the player)
		for (const p of ["textTracks", "audioTracks", "videoTracks"]) {
			Object.defineProperty(cls.prototype, p, {
				get() {
					const k = "__" + p;
					if (!this[k]) this[k] = new TrackList();
					return this[k];
				},
			});
		}
		cls.prototype.addTextTrack = function (kind, label, language) {
			const t = new TextTrack(kind, label, language);
			this.textTracks._add(t);
			return t;
		};
		const addListener = cls.prototype.addEventListener;
		cls.prototype.addEventListener = function (type, ...rest) {
			const r = addListener.call(this, type, ...rest);
			if (mediaEvents.has(String(type))) this._mediaLoad();
			return r;
		};
	});
	// ---- Media Source Extensions (see mse.go) ----

	// timeRanges returns TimeRanges of the given flat array of the starts
	// and ends of ranges.
	const timeRanges = flat => {
		const n = flat.length / 2;
		const check = i => { if (!(i >= 0 && i < n)) throw domException("IndexSizeError", "no time range " + i); };
		return { length: n, start(i) { check(i); return flat[2 * i]; }, end(i) { check(i); return flat[2 * i + 1]; } };
	};
	// intersectRanges returns the intersection of the given flat arrays of
	// ranges (the buffered ranges of a media element, of its buffers).
	const intersectRanges = (a, b) => {
		const r = [];
		for (let i = 0; i < a.length; i += 2) {
			for (let j = 0; j < b.length; j += 2) {
				const s = Math.max(a[i], b[j]), e = Math.min(a[i + 1], b[j + 1]);
				if (s < e) r.push(s, e);
			}
		}
		return r;
	};
	const mediaSourcesByURL = new Map(); // object URL -> MediaSource

	class SourceBuffer extends EventTarget {
		constructor(ms, id, type) {
			super();
			Object.assign(this, { _ms: ms, _id: id, _type: type, updating: false, mode: "segments", _offset: 0, appendWindowStart: 0, appendWindowEnd: Infinity });
		}
		// the timestampOffset, which is added to the times of the media
		// appended after it is set
		get timestampOffset() { return this._offset; }
		set timestampOffset(v) {
			if (this.updating) throw domException("InvalidStateError", "the SourceBuffer is updating");
			this._offset = Number(v) || 0;
			D.mseSetOffset(this._id, this._offset);
		}
		get buffered() { return timeRanges(D.mseBuffered(this._id)); }
		_ranges() { return D.mseBuffered(this._id); }
		// _op does the given operation of the buffer at once (as an append,
		// whose data is copied), with its events after (as in browsers).
		_op(f) {
			const ms = this._ms;
			if (ms.readyState === "closed") throw domException("InvalidStateError", "the MediaSource is closed");
			if (this.updating) throw domException("InvalidStateError", "the SourceBuffer is updating");
			if (ms.readyState === "ended") {
				ms.readyState = "open";
				ms.dispatchEvent(new Event("sourceopen"));
			}
			const err = f();
			this.updating = true;
			this.dispatchEvent(new Event("updatestart"));
			setTimeout(() => {
				if (!this.updating) return; // (aborted)
				this.updating = false;
				this.dispatchEvent(new Event(err ? "error" : "update"));
				this.dispatchEvent(new Event("updateend"));
			}, 0);
		}
		appendBuffer(data) { this._op(() => D.mseAppend(this._id, data)); }
		remove(start, end) { this._op(() => { D.mseRemove(this._id, Number(start), Number(end)); return ""; }); }
		abort() {
			D.mseAbort(this._id);
			if (this.updating) {
				this.updating = false;
				this.dispatchEvent(new Event("abort"));
				this.dispatchEvent(new Event("updateend"));
			}
		}
		changeType(type) { this._type = String(type); }
	}
	defineSimpleHandlers(SourceBuffer, ["updatestart", "update", "updateend", "error", "abort"]);

	class SourceBufferList extends EventTarget {
		constructor() { super(); this._list = []; }
		get length() { return this._list.length; }
		_set(list) {
			for (let i = 0; i < this._list.length; i++) delete this[i];
			this._list = list;
			list.forEach((b, i) => { this[i] = b; });
		}
		[Symbol.iterator]() { return this._list[Symbol.iterator](); }
	}
	defineSimpleHandlers(SourceBufferList, ["addsourcebuffer", "removesourcebuffer"]);

	class MediaSource extends EventTarget {
		constructor() {
			super();
			const [id, url] = D.mseCreate();
			Object.assign(this, { _id: id, _url: url, readyState: "closed", _duration: NaN, _el: null });
			this.sourceBuffers = new SourceBufferList();
			this.activeSourceBuffers = this.sourceBuffers;
			mediaSourcesByURL.set(url, this);
		}
		static isTypeSupported(t) { return D.mseTypeSupported(String(t)); }
		static get canConstructInDedicatedWorker() { return false; }
		get duration() { return this._duration; }
		set duration(v) {
			if (this.readyState !== "open") throw domException("InvalidStateError", "the MediaSource is not open");
			this._duration = Number(v);
			D.mseSetDuration(this._id, this._duration);
			const el = this._el;
			if (el) {
				el._media().duration = this._duration;
				el.dispatchEvent(new Event("durationchange"));
			}
		}
		addSourceBuffer(type) {
			type = String(type);
			if (!MediaSource.isTypeSupported(type)) throw domException("NotSupportedError", "the type is not supported: " + type);
			if (this.readyState !== "open") throw domException("InvalidStateError", "the MediaSource is not open");
			const sb = new SourceBuffer(this, D.mseAddBuffer(this._id, type), type);
			this.sourceBuffers._set([...this.sourceBuffers._list, sb]);
			this.sourceBuffers.dispatchEvent(new Event("addsourcebuffer"));
			return sb;
		}
		removeSourceBuffer(sb) {
			D.mseRemoveBuffer(sb._id);
			this.sourceBuffers._set(this.sourceBuffers._list.filter(b => b !== sb));
			this.sourceBuffers.dispatchEvent(new Event("removesourcebuffer"));
		}
		endOfStream() {
			if (this.readyState !== "open") throw domException("InvalidStateError", "the MediaSource is not open");
			this.readyState = "ended";
			D.mseEnd(this._id);
			this.dispatchEvent(new Event("sourceended"));
		}
		setLiveSeekableRange() {}
		clearLiveSeekableRange() {}
		// _attach attaches the media source to the given media element,
		// whose source is its object URL, which opens it.
		_attach(el) {
			this._el = el;
			el.__mediaSource = this;
			if (this.readyState !== "closed") return;
			this.readyState = "open";
			Promise.resolve().then(() => this.dispatchEvent(new Event("sourceopen")));
		}
		// _buffered returns the buffered ranges of its media (those of all
		// of its buffers).
		_buffered() {
			const list = this.sourceBuffers._list;
			if (!list.length) return [];
			return list.slice(1).reduce((r, b) => intersectRanges(r, b._ranges()), list[0]._ranges());
		}
	}
	defineSimpleHandlers(MediaSource, ["sourceopen", "sourceended", "sourceclose"]);

	const HTMLDetailsElement = elementClass(["details"], HTMLElement, cls => reflect(cls, [], ["open"]));
	const HTMLDialogElement = elementClass(["dialog"], HTMLElement, cls => {
		reflect(cls, [], ["open"]);
		// a modal dialog is marked for the :modal selector of the style
		// sheets (htmlcore.ModalAttr), which is in the top layer
		cls.prototype.show = function () { this.open = true; };
		cls.prototype.showModal = function () { this.setAttribute("data-htmlcore-modal", ""); this.open = true; };
		cls.prototype.close = function (v) {
			if (!this.open) return;
			this.removeAttribute("data-htmlcore-modal");
			this.open = false;
			this.returnValue = v === undefined ? this.returnValue || "" : String(v);
			this.dispatchEvent(new Event("close"));
		};
	});
	const HTMLDivElement = elementClass(["div"]);
	const HTMLSpanElement = elementClass(["span"]);
	const HTMLParagraphElement = elementClass(["p"]);
	const HTMLHeadingElement = elementClass(["h1", "h2", "h3", "h4", "h5", "h6"]);
	const HTMLUListElement = elementClass(["ul"]);
	const HTMLOListElement = elementClass(["ol"]);
	const HTMLLIElement = elementClass(["li"]);
	const HTMLTableElement = elementClass(["table"], HTMLElement, cls => {
		Object.defineProperty(cls.prototype, "rows", { get() { return this.querySelectorAll("tr"); } });
		Object.defineProperty(cls.prototype, "tBodies", { get() { return this.querySelectorAll("tbody"); } });
	});
	const HTMLTableRowElement = elementClass(["tr"], HTMLElement, cls => {
		Object.defineProperty(cls.prototype, "cells", { get() { return list(D.children(this[ID]).filter(id => D.name(id) === "td" || D.name(id) === "th")); } });
	});
	const HTMLTableCellElement = elementClass(["td", "th"]);
	const HTMLHtmlElement = elementClass(["html"]);
	const HTMLHeadElement = elementClass(["head"]);
	const HTMLBodyElement = elementClass(["body"]);
	const HTMLPreElement = elementClass(["pre"]);
	const HTMLBRElement = elementClass(["br"]);
	const HTMLHRElement = elementClass(["hr"]);
	const HTMLFieldSetElement = elementClass(["fieldset"]);
	const HTMLUnknownElement = class extends HTMLElement {};
	const SVGElement = class extends Element {};
	// SVGSVGElement has the methods of svg elements that scripts check for
	// (as the support check of the VisualEditor of Wikipedia), which make
	// geometry objects that are not connected to the rendering.
	const SVGSVGElement = class extends SVGElement {
		createSVGRect() { return { x: 0, y: 0, width: 0, height: 0 }; }
		createSVGPoint() { return { x: 0, y: 0, matrixTransform() { return { x: this.x, y: this.y }; } }; }
		createSVGMatrix() { return new DOMMatrixStub(); }
		createSVGTransform() { return { type: 1, matrix: new DOMMatrixStub(), setMatrix() {}, setTranslate() {}, setScale() {}, setRotate() {} }; }
		getScreenCTM() { return new DOMMatrixStub(); }
		getCTM() { return new DOMMatrixStub(); }
		getBBox() { const r = this.getBoundingClientRect(); return { x: 0, y: 0, width: r.width, height: r.height }; }
	};
	// DOMMatrixStub is an identity matrix, for the geometry of SVG.
	class DOMMatrixStub {
		constructor() { Object.assign(this, { a: 1, b: 0, c: 0, d: 1, e: 0, f: 0 }); }
		multiply() { return new DOMMatrixStub(); }
		inverse() { return new DOMMatrixStub(); }
		translate() { return new DOMMatrixStub(); }
		scale() { return new DOMMatrixStub(); }
		rotate() { return new DOMMatrixStub(); }
	}

	// submitForm submits the given form, by navigating (as for a GET).
	function submitForm(form, submitter) {
		const method = (form.getAttribute("method") || "get").toLowerCase();
		const action = form.action || location.href;
		const fd = new FormData(form);
		if (submitter && submitter.name) fd.append(submitter.name, submitter.value);
		const q = new URLSearchParams();
		for (const [k, v] of fd) q.append(k, typeof v === "string" ? v : v.name || "");
		// the iframe that the form targets by its name, if any (as Google
		// opens its account menu in a frame), in which the page is opened
		const target = (submitter && submitter.getAttribute && submitter.getAttribute("formtarget")) || form.getAttribute("target") || "";
		let frame = null;
		if (target && !/^_(self|top|parent|blank)$/i.test(target)) {
			for (const f of document.getElementsByTagName("iframe")) {
				if (f.getAttribute("name") === target) { frame = f; break; }
			}
		}
		if (frame) {
			if (method === "get") {
				const u = new URL(action);
				u.search = "?" + q.toString();
				D.submit(u.href, "", "", frame[ID], false);
			} else {
				D.submit(action, "application/x-www-form-urlencoded", q.toString(), frame[ID], true);
			}
			return;
		}
		if (method === "get") {
			const u = new URL(action);
			u.search = "?" + q.toString();
			location.href = u.href;
		} else if ((form.getAttribute("enctype") || "").toLowerCase() === "multipart/form-data") {
			// the files of the form are not sent, only their names
			const boundary = "----CogentFormBoundary" + Math.random().toString(36).slice(2);
			let body = "";
			for (const [k, v] of fd) body += "--" + boundary + '\r\nContent-Disposition: form-data; name="' + k.replace(/"/g, "%22") + '"\r\n\r\n' + (typeof v === "string" ? v : v.name || "") + "\r\n";
			D.submit(action, "multipart/form-data; boundary=" + boundary, body + "--" + boundary + "--\r\n");
		} else {
			D.submit(action, "application/x-www-form-urlencoded", q.toString());
		}
	}

	// ---- document ----

	let focused = null;
	let readyState = "loading";

	class Document extends Node {
		get documentElement() { return wrap(D.query(this[ID], "html")); }
		get head() { return wrap(D.query(this[ID], "head")); }
		get body() { return wrap(D.query(this[ID], "body")); }
		set body(v) { const b = this.body; if (b) this.documentElement.replaceChild(v, b); }
		get title() { const t = this.querySelector("title"); return t ? t.textContent.trim().replace(/\s+/g, " ") : ""; }
		set title(v) {
			let t = this.querySelector("title");
			if (!t && this.head) { t = this.createElement("title"); this.head.appendChild(t); }
			if (t) t.textContent = v;
		}
		get doctype() { return wrap(D.children(this[ID]).find(id => D.type(id) === 10) || 0); }
		get URL() { return location.href; }
		get documentURI() { return location.href; }
		get location() { return this === document ? location : null; }
		set location(v) { location.href = v; }
		get domain() { return location.hostname; }
		get referrer() { return ""; }
		get cookie() { return D.cookie(); }
		set cookie(v) { D.setCookie(String(v)); }
		get readyState() { return readyState; }
		get defaultView() { return this === document ? g : null; }
		get characterSet() { return "UTF-8"; }
		get charset() { return "UTF-8"; }
		get inputEncoding() { return "UTF-8"; }
		get contentType() { return "text/html"; }
		get compatMode() { return "CSS1Compat"; }
		get visibilityState() { return "visible"; }
		get hidden() { return false; }
		get activeElement() { return focused && focused.isConnected ? focused : this.body; }
		get currentScript() { return wrap(D.currentScript()); }
		get scrollingElement() { return this.documentElement; }
		get forms() { return this.querySelectorAll("form"); }
		get images() { return this.querySelectorAll("img"); }
		get links() { return this.querySelectorAll("a[href], area[href]"); }
		get scripts() { return this.querySelectorAll("script"); }
		get styleSheets() { return this.querySelectorAll("style").map(st => st.sheet); }
		get adoptedStyleSheets() { return this._adopted || (this._adopted = []); }
		set adoptedStyleSheets(v) { setAdopted(this, v); }
		get fonts() { return { ready: Promise.resolve(), check() { return true; }, load() { return Promise.resolve([]); }, addEventListener() {}, add() {}, forEach() {} }; }
		get implementation() { return { createHTMLDocument(title) { return wrap(D.parseDocument("<!DOCTYPE html><title>" + (title || "") + "</title>")); }, hasFeature() { return true; } }; }
		createElement(tag) { return wrap(D.create(String(tag))); }
		createAttribute(name) { return new Attr(String(name).toLowerCase(), null); }
		createElementNS(ns, tag) {
			let n = String(tag);
			if (n.indexOf(":") >= 0) n = n.split(":")[1];
			return wrap(D.create(n, namespaces[ns] || ""));
		}
		createTextNode(s) { return wrap(D.createText(String(s))); }
		createComment(s) { return wrap(D.createComment(String(s))); }
		createDocumentFragment() { return wrap(D.createFragment()); }
		createEvent(type) {
			const e = new Event("");
			e.initEvent = function (t, b, c) { this.type = t; this.bubbles = !!b; this.cancelable = !!c; };
			e.initCustomEvent = function (t, b, c, d) { this.initEvent(t, b, c); this.detail = d; };
			e.initMouseEvent = e.initEvent;
			return e;
		}
		createRange() { return new Range(); }
		createTreeWalker(root, what, filter) { return treeWalker(root, what, filter); }
		createNodeIterator(root, what, filter) { const w = treeWalker(root, what, filter); return { nextNode: () => w.nextNode() }; }
		importNode(n, deep) { return n.cloneNode(deep); }
		adoptNode(n) { if (n.parentNode) n.remove(); return n; }
		getElementById(id) { return wrap(D.byId(this[ID], String(id))); }
		getElementsByName(n) { return this.querySelectorAll("[name='" + String(n).replace(/'/g, "\\'") + "']"); }
		hasFocus() { return true; }
		write(...s) { D.write(s.join("")); }
		writeln(...s) { D.write(s.join("") + "\n"); }
		open() { return this; }
		close() {}
		execCommand() { return false; }
		queryCommandSupported() { return false; }
		getSelection() { return g.getSelection(); }
		elementFromPoint(x, y) { return wrap(D.elementAt(+x + scrollState.x, +y + scrollState.y)); }
		elementsFromPoint(x, y) {
			const out = [];
			for (let n = this.elementFromPoint(x, y); n && n.nodeType === 1; n = n.parentNode) out.push(n);
			return out;
		}
		exitFullscreen() { return Promise.resolve(); }
		get fullscreenElement() { return null; }
		// (the prefixed names of Chrome and Safari, and the access of
		// frames to their storage, which they have)
		webkitExitFullscreen() { return Promise.resolve(); }
		get webkitFullscreenElement() { return null; }
		get webkitIsFullScreen() { return false; }
		get prerendering() { return false; }
		hasStorageAccess() { return Promise.resolve(true); }
		requestStorageAccess() { return Promise.resolve(); }
	}
	mixin(Document, parentNodeMethods);
	defineHandlers(Document.prototype);
	class HTMLDocument extends Document {}
	class DocumentFragment extends Node {
		getElementById(id) { return wrap(D.byId(this[ID], String(id))); }
		get innerHTML() { return D.inner(this[ID]); }
		set innerHTML(v) { D.setInner(this[ID], String(v)); }
	}
	mixin(DocumentFragment, parentNodeMethods);

	// ShadowRoot is the shadow root of an element, a fragment whose content
	// is not rendered (only the children of the element are), but which
	// scripts use (as web components render into them).
	// Range is a range of the DOM between two boundary points (nodes and
	// offsets in them: of characters in character data, of children in the
	// others). Its contents can be deleted, extracted and cloned when its
	// points are in the same node (as the frames of Turbo on GitHub, which
	// replace the children of an element with those of another).
	class Range {
		constructor() {
			this.startContainer = this.endContainer = document;
			this.startOffset = this.endOffset = 0;
		}
		get collapsed() { return this.startContainer === this.endContainer && this.startOffset === this.endOffset; }
		get commonAncestorContainer() {
			for (let a = this.startContainer; a; a = a.parentNode) {
				if (a === this.endContainer || a.contains(this.endContainer)) return a;
			}
			return this.startContainer;
		}
		setStart(n, o) { this.startContainer = n; this.startOffset = o | 0; if (this.collapsedAfterEnd()) this.collapse(true); }
		setEnd(n, o) { this.endContainer = n; this.endOffset = o | 0; if (this.collapsedAfterEnd()) this.collapse(false); }
		// collapsedAfterEnd returns whether the start is after the end in
		// the same node (then the range collapses to the point set).
		collapsedAfterEnd() { return this.startContainer === this.endContainer && this.startOffset > this.endOffset; }
		setStartBefore(n) { this.setStart(n.parentNode, rangeIndex(n)); }
		setStartAfter(n) { this.setStart(n.parentNode, rangeIndex(n) + 1); }
		setEndBefore(n) { this.setEnd(n.parentNode, rangeIndex(n)); }
		setEndAfter(n) { this.setEnd(n.parentNode, rangeIndex(n) + 1); }
		selectNode(n) { const p = n.parentNode, i = rangeIndex(n); this.startContainer = this.endContainer = p; this.startOffset = i; this.endOffset = i + 1; }
		selectNodeContents(n) { this.startContainer = this.endContainer = n; this.startOffset = 0; this.endOffset = rangeLength(n); }
		collapse(toStart) {
			if (toStart) { this.endContainer = this.startContainer; this.endOffset = this.startOffset; }
			else { this.startContainer = this.endContainer; this.startOffset = this.endOffset; }
		}
		cloneRange() { const r = new Range(); Object.assign(r, { startContainer: this.startContainer, startOffset: this.startOffset, endContainer: this.endContainer, endOffset: this.endOffset }); return r; }
		detach() {}
		// rangeNodes returns the container of the range and the children of
		// it in the range, or null if its points are in different nodes.
		rangeNodes() {
			const c = this.startContainer;
			if (c !== this.endContainer) { D.log("debug", "missing: contents of a Range in different nodes"); return null; }
			return Array.from(c.childNodes).slice(this.startOffset, this.endOffset);
		}
		isText() { const t = this.startContainer.nodeType; return t === 3 || t === 4 || t === 8; }
		deleteContents() {
			if (this.isText() && this.startContainer === this.endContainer) {
				const c = this.startContainer, d = c.data;
				c.data = d.slice(0, this.startOffset) + d.slice(this.endOffset);
			} else {
				const ns = this.rangeNodes();
				if (ns) for (const n of ns) n.remove();
			}
			this.collapse(true);
		}
		cloneContents() {
			const f = document.createDocumentFragment();
			if (this.isText() && this.startContainer === this.endContainer) {
				f.appendChild(document.createTextNode(this.startContainer.data.slice(this.startOffset, this.endOffset)));
			} else {
				const ns = this.rangeNodes();
				if (ns) for (const n of ns) f.appendChild(n.cloneNode(true));
			}
			return f;
		}
		extractContents() {
			if (this.isText() && this.startContainer === this.endContainer) {
				const f = this.cloneContents();
				this.deleteContents();
				return f;
			}
			const f = document.createDocumentFragment();
			const ns = this.rangeNodes();
			if (ns) for (const n of ns) f.appendChild(n);
			this.collapse(true);
			return f;
		}
		insertNode(n) {
			const c = this.startContainer;
			if (this.isText()) {
				const after = c.splitText(this.startOffset);
				c.parentNode.insertBefore(n, after);
			} else {
				c.insertBefore(n, c.childNodes[this.startOffset] || null);
			}
			if (this.startContainer === this.endContainer && !this.isText()) this.endOffset += n.nodeType === 11 ? 0 : 1;
		}
		surroundContents(n) { n.appendChild(this.extractContents()); this.insertNode(n); this.selectNode(n); }
		toString() {
			if (this.isText() && this.startContainer === this.endContainer) return this.startContainer.data.slice(this.startOffset, this.endOffset);
			const ns = this.startContainer === this.endContainer ? Array.from(this.startContainer.childNodes || []).slice(this.startOffset, this.endOffset) : [];
			return ns.map(n => n.textContent || "").join("");
		}
		compareBoundaryPoints() { return 0; }
		isPointInRange(n, o) { return this.startContainer === n ? o >= this.startOffset && o <= this.endOffset : this.commonAncestorContainer.contains(n); }
		intersectsNode(n) { return this.commonAncestorContainer.contains(n); }
		getBoundingClientRect() { return new DOMRect(); }
		getClientRects() { return []; }
		createContextualFragment(html) { return wrap(D.parse(String(html), 0)); }
	}
	Object.assign(Range, { START_TO_START: 0, START_TO_END: 1, END_TO_END: 2, END_TO_START: 3 });
	// rangeIndex returns the index of the given node in its parent.
	function rangeIndex(n) { return n.parentNode ? Array.prototype.indexOf.call(n.parentNode.childNodes, n) : 0; }
	// rangeLength returns the length of a node as a boundary point: its
	// characters for character data, else its children.
	function rangeLength(n) { const t = n.nodeType; return t === 3 || t === 4 || t === 8 ? n.data.length : n.childNodes.length; }

	class ShadowRoot extends DocumentFragment {
		get host() { return wrap(shadowHosts.get(this[ID])); }
		get mode() { const s = shadows.get(this.host); return s ? s.mode : "open"; }
		get activeElement() { return null; }
		get adoptedStyleSheets() { return this._adopted || (this._adopted = []); }
		set adoptedStyleSheets(v) { setAdopted(this, v); }
	}

	// treeWalker returns a TreeWalker, which walks the live tree from its
	// currentNode, as in the DOM spec: Lit sets the currentNode to the
	// content of a template (outside of the root, the document) and walks
	// it while it changes it (with the nodes of the root listed when it
	// was made, Lit walked the document, and its templates kept the
	// markers of their bindings: the menus of MDN were all open).
	function treeWalker(root, what, filter) {
		what = what === undefined ? 0xFFFFFFFF : what;
		const ACCEPT = 1, REJECT = 2;
		const check = n => { // 1 accept, 2 reject (with its descendants), 3 skip
			if (!(what & (1 << (n.nodeType - 1)))) return 3;
			if (!filter) return ACCEPT;
			const f = typeof filter === "function" ? filter : filter.acceptNode;
			return f.call(filter, n);
		};
		return {
			root, currentNode: root, whatToShow: what, filter: filter || null,
			nextNode() {
				let n = this.currentNode, r = ACCEPT;
				for (;;) {
					while (r !== REJECT && n.firstChild) {
						n = n.firstChild;
						if ((r = check(n)) === ACCEPT) return (this.currentNode = n);
					}
					let s = null;
					for (let t = n; t && t !== root; t = t.parentNode) {
						if (t.nextSibling) { s = t.nextSibling; break; }
					}
					if (!s) return null;
					n = s;
					if ((r = check(n)) === ACCEPT) return (this.currentNode = n);
				}
			},
			previousNode() {
				let n = this.currentNode;
				while (n !== root) {
					let s = n.previousSibling;
					while (s) {
						n = s;
						let r = check(n);
						while (r !== REJECT && n.lastChild) { n = n.lastChild; r = check(n); }
						if (r === ACCEPT) return (this.currentNode = n);
						s = n.previousSibling;
					}
					if (!n.parentNode || n === root) return null;
					n = n.parentNode;
					if (check(n) === ACCEPT) return (this.currentNode = n);
				}
				return null;
			},
			parentNode() {
				for (let n = this.currentNode; n && n !== root;) {
					n = n.parentNode;
					if (n && check(n) === ACCEPT) return (this.currentNode = n);
				}
				return null;
			},
			firstChild() {
				for (let c = this.currentNode.firstChild; c; c = c.nextSibling) if (check(c) === ACCEPT) return (this.currentNode = c);
				return null;
			},
			lastChild() {
				for (let c = this.currentNode.lastChild; c; c = c.previousSibling) if (check(c) === ACCEPT) return (this.currentNode = c);
				return null;
			},
			nextSibling() {
				for (let s = this.currentNode.nextSibling; s; s = s.nextSibling) if (check(s) === ACCEPT) return (this.currentNode = s);
				return null;
			},
			previousSibling() {
				for (let s = this.currentNode.previousSibling; s; s = s.previousSibling) if (check(s) === ACCEPT) return (this.currentNode = s);
				return null;
			},
		};
	}

	class DOMRect {
		constructor(x, y, w, h) { this.x = x || 0; this.y = y || 0; this.width = w || 0; this.height = h || 0; }
		get left() { return this.x; }
		get top() { return this.y; }
		get right() { return this.x + this.width; }
		get bottom() { return this.y + this.height; }
		toJSON() { return { x: this.x, y: this.y, width: this.width, height: this.height, left: this.left, top: this.top, right: this.right, bottom: this.bottom }; }
	}

	// ---- the window ----

	const document = wrap(D.doc());
	const scrollState = { x: 0, y: 0 };

	function urlParts(cls, getHref, setHref) {
		const parts = ["href", "protocol", "username", "password", "host", "hostname", "port", "pathname", "search", "hash", "origin"];
		parts.forEach((p, i) => {
			if (p === "href") return;
			Object.defineProperty(cls.prototype, p, {
				get() { const u = D.parseURL(getHref.call(this)); return u ? u[i] : ""; },
				set(v) {
					if (p === "origin") return;
					const u = new URL(getHref.call(this));
					u[p] = v;
					setHref.call(this, u.href);
				},
				configurable: true,
			});
		});
	}

	class Location {
		get href() { return D.pageURL(); }
		set href(v) { navigate(String(v), false); }
		assign(v) { navigate(String(v), false); }
		replace(v) { navigate(String(v), true); }
		reload() { D.navigate(D.pageURL(), true); }
		toString() { return this.href; }
		get ancestorOrigins() { return []; }
	}
	urlParts(Location, function () { return this.href; }, function (v) { this.href = v; });

	function navigate(u, replace) {
		if (/^\s*javascript:/i.test(u)) {
			try { (0, eval)(decodeURIComponent(u.slice(u.indexOf(":") + 1))); } catch (e) { reportError(e); }
			return;
		}
		const cur = D.parseURL(D.pageURL());
		const to = D.parseURL(u, D.baseURL());
		if (!to) return;
		if (cur && to[0].split("#")[0] === cur[0].split("#")[0] && to[9] !== cur[9]) {
			// only the fragment changes: the page is not opened again
			D.pushState(to[0], replace);
			g.dispatchEvent(new HashChangeEvent("hashchange", { oldURL: cur[0], newURL: to[0] }));
			return;
		}
		D.navigate(to[0], replace);
	}
	const location = new Location();

	// History has its properties on its prototype, as in browsers, which
	// polyfills (as of the Navigation API, on Reddit) patch.
	let historyState = null;
	class History {
		constructor() { this.scrollRestoration = "auto"; }
		get length() { return 1; }
		get state() { return historyState; }
		pushState(state, title, url) { historyState = state; if (url !== undefined && url !== null) D.pushState(String(url), false); }
		replaceState(state, title, url) { historyState = state; if (url !== undefined && url !== null) D.pushState(String(url), true); }
		back() {}
		forward() {}
		go() {}
	}
	const history = new History();

	// Navigation is a minimal Navigation API (window.navigation), so that
	// sites do not load polyfills of it (Reddit's recurses through history
	// until a RangeError, which goja cannot catch). It never fires navigate
	// events: links and navigate() open pages as location does.
	let navEntry = null;
	class NavigationHistoryEntry extends EventTarget {
		constructor(state) { super(); this.key = this.id = String(++callbackID); this._state = state; }
		get url() { return location.href; }
		get index() { return 0; }
		get sameDocument() { return true; }
		getState() { return this._state; }
	}
	class Navigation extends EventTarget {
		get currentEntry() { if (!navEntry) navEntry = new NavigationHistoryEntry(undefined); return navEntry; }
		entries() { return [this.currentEntry]; }
		get transition() { return null; }
		get activation() { return null; }
		get canGoBack() { return false; }
		get canGoForward() { return false; }
		navigate(url, opts) {
			navigate(String(url), !!opts && opts.history === "replace");
			const entry = this.currentEntry;
			return { committed: Promise.resolve(entry), finished: Promise.resolve(entry) };
		}
		reload() { return this.navigate(location.href, { history: "replace" }); }
		updateCurrentEntry(opts) { navEntry = new NavigationHistoryEntry(opts ? opts.state : undefined); }
		traverseTo() { const p = Promise.reject(domException("InvalidStateError", "no such entry")); p.catch(() => {}); return { committed: p, finished: p }; }
		back() { return this.traverseTo(); }
		forward() { return this.traverseTo(); }
	}
	const navigation = new Navigation();

	const navigator = {
		// the user agent is that of the requests of the page (sites check
		// that they match), as are the languages (Accept-Language)
		userAgent: D.userAgent, appVersion: D.userAgent.replace(/^Mozilla\//, ""),
		appName: "Netscape", appCodeName: "Mozilla", product: "Gecko", vendor: "Cogent Core",
		platform: D.platform, language: D.languages[0], languages: D.languages, onLine: true, cookieEnabled: true,
		hardwareConcurrency: 4, deviceMemory: 8, maxTouchPoints: 0, doNotTrack: null, webdriver: false, pdfViewerEnabled: false,
		plugins: [], mimeTypes: [],
		sendBeacon() { return true; },
		javaEnabled() { return false; },
		clipboard: { writeText() { return Promise.resolve(); }, readText() { return Promise.resolve(""); } },
		permissions: { query() { return Promise.resolve({ state: "denied", addEventListener() {} }); } },
		storage: { estimate() { return Promise.resolve({ quota: 0, usage: 0 }); }, persist() { return Promise.resolve(false); } },
		userActivation: { hasBeenActive: false, isActive: false },
		// the locks of the page (Web Locks), which it alone takes here: a
		// request gets its lock at once
		locks: {
			request(name, opts, cb) {
				if (typeof opts === "function") cb = opts;
				return Promise.resolve().then(() => cb({ name: String(name), mode: (opts && opts.mode) || "exclusive" }));
			},
			query() { return Promise.resolve({ held: [], pending: [] }); },
		},
		// no cameras or microphones (Google account security pages list
		// them), and no credentials of the browser (passkeys or saved
		// passwords): sites fall back to their other ways
		mediaDevices: Object.assign(new EventTarget(), {
			enumerateDevices() { return Promise.resolve([]); },
			getSupportedConstraints() { return {}; },
			getUserMedia() { return Promise.reject(domException("NotAllowedError", "Permission denied")); },
			getDisplayMedia() { return Promise.reject(domException("NotAllowedError", "Permission denied")); },
			ondevicechange: null,
		}),
		credentials: {
			get() { return Promise.reject(domException("NotAllowedError", "The operation is not allowed")); },
			create() { return Promise.reject(domException("NotAllowedError", "The operation is not allowed")); },
			store(c) { return Promise.resolve(c); },
			preventSilentAccess() { return Promise.resolve(); },
		},
	};

	// storageKinds are the kinds (local or session) of the Storage objects,
	// not in properties of them: the keys of their proxies are only their
	// items, and a proxy cannot hide a non-configurable property of its
	// target (Object.keys(localStorage) threw on GitHub).
	const storageKinds = new WeakMap();
	class Storage {
		constructor(kind) { storageKinds.set(this, kind); }
		getItem(k) { return D.storage("get", storageKinds.get(this), String(k)); }
		setItem(k, v) { D.storage("set", storageKinds.get(this), String(k), String(v)); }
		removeItem(k) { D.storage("remove", storageKinds.get(this), String(k)); }
		clear() { D.storage("clear", storageKinds.get(this)); }
		key(i) { const ks = D.storage("keys", storageKinds.get(this)); return i < ks.length ? ks[i] : null; }
		get length() { return D.storage("keys", storageKinds.get(this)).length; }
	}
	function storageProxy(kind) {
		return new Proxy(new Storage(kind), {
			get(t, p) { if (typeof p !== "string" || p in t) { const v = Reflect.get(t, p, t); return typeof v === "function" ? v.bind(t) : v; } const v = t.getItem(p); return v === null ? undefined : v; },
			set(t, p, v) { if (typeof p !== "string" || p in t) return Reflect.set(t, p, v, t); t.setItem(p, v); return true; },
			deleteProperty(t, p) { t.removeItem(p); return true; },
			has(t, p) { return p in t || t.getItem(p) !== null; },
			ownKeys(t) { return D.storage("keys", storageKinds.get(t)); },
			getOwnPropertyDescriptor(t, p) { const v = t.getItem(p); return v === null ? undefined : { value: v, enumerable: true, configurable: true, writable: true }; },
		});
	}

	class MediaQueryList extends EventTarget {
		constructor(q) { super(); this.media = String(q); }
		get matches() { return D.mediaMatches(this.media); }
		addListener(f) { this.addEventListener("change", f); }
		removeListener(f) { this.removeEventListener("change", f); }
	}
	defineHandlers(MediaQueryList.prototype);

	// getComputedStyle returns an approximation of the computed style:
	// the inline style, or the value of the style sheets of the page (as
	// last read), with the display of an element that has no box.
	const inlineTags = "span, a, b, i, em, strong, code, img, label, small, sup, sub, abbr, q, cite, time, mark, u, s, kbd, var, dfn, bdi, bdo, data, br, wbr";
	function getComputedStyle(el) {
		const inline = el && el.style ? el.style : null;
		const get = p => {
			p = camelToKebab(String(p));
			const v = inline ? inline.getPropertyValue(p) : "";
			if (v) return v.replace(/\s*!important$/, "");
			const cv = el && el[ID] ? D.computed(el[ID], p) : "";
			if (cv && p !== "display") return cv;
			const box = el ? D.rect(el[ID]) : null;
			if (cv && box) return cv; // display
			switch (p) {
				case "display": {
					// an element with no box of its own is not displayed only
					// if it or an element that it is in is not: the inline
					// elements in a text (as a link in a paragraph) are in the
					// widget of the text, and are inline
					if (cv === "none") return "none";
					const byTag = el.matches(inlineTags) ? "inline" : "block";
					if (box) return byTag;
					// the html element has no box of its own (nor can the
					// body), and is displayed unless it is styled not to be,
					// as the elements in it (scripts check whether they are
					// shown up to it, and it was none)
					const root = document.documentElement;
					if (el === root) return cv || "block";
					for (let a = el.parentElement; a && a[ID] != null; a = a.parentElement) {
						if (D.computed(a[ID], "display") === "none") return "none";
						if (D.rect(a[ID]) || a === root) return cv || byTag;
					}
					return "none";
				}
				case "visibility": return "visible";
				case "position": return "static";
				case "opacity": return "1";
				case "width": return box ? box[2] + "px" : "auto";
				case "height": return box ? box[3] + "px" : "auto";
				case "direction": return "ltr";
				case "font-size": return "16px";
				case "line-height": return "normal";
				case "overflow": case "overflow-x": case "overflow-y": return "visible";
				case "transform": return "none";
				case "z-index": return "auto";
				case "box-sizing": return "content-box";
			}
			if (/^(margin|padding)(-|$)|border-.*width/.test(p)) return "0px";
			return "";
		};
		return new Proxy({ getPropertyValue: get, getPropertyPriority: () => "", item: () => "", length: 0, cssText: "" }, {
			get(t, p) { if (typeof p !== "string") return undefined; if (p in t) return t[p]; return get(p); },
			has(t, p) { return p in t || isStyleProperty(p); },
		});
	}

	// ---- timers ----

	const timers = new Map(); // id -> [fn, args, repeat]
	function setTimer(fn, ms, args, repeat) {
		if (typeof fn !== "function") { const src = String(fn); fn = () => (0, eval)(src); }
		const id = D.setTimer(Number(ms) || 0, repeat);
		timers.set(id, [fn, args, repeat]);
		return id;
	}
	function clearTimer(id) { if (timers.delete(id)) D.clearTimer(id); }
	D.fireTimer = function (id) {
		const t = timers.get(id);
		if (!t) return;
		if (!t[2]) timers.delete(id);
		try { t[0].apply(g, t[1]); } catch (e) { reportError(e); }
	};

	// ---- callbacks of Go (network) ----

	const callbacks = new Map();
	let callbackID = 0;
	function newCallback(f) { const id = ++callbackID; callbacks.set(id, f); return id; }
	D.callback = function (id, ...args) {
		const f = callbacks.get(id);
		if (!f) return;
		callbacks.delete(id);
		try { f(...args); } catch (e) { reportError(e); }
	};

	// __dynamicImport is what the dynamic imports of scripts (import(...))
	// are rewritten as: it gets and runs the module, bundled.
	function dynamicImport(base, spec) {
		return new Promise((resolve, reject) => {
			D.importModule(newCallback((u, err) => {
				if (err) reject(new TypeError("Failed to import " + spec + ": " + err));
				else {
					try { resolve(requireModule(u)); } catch (e) { reject(e); }
				}
			}), String(spec), base);
		});
	}

	// The registry of modules (see modules.go): each is evaluated when it
	// is first required, once.
	const modules = new Map(); // url -> { deps, factory, module, error }
	function defineModule(url, deps, factory) {
		if (!modules.has(url)) modules.set(url, { deps, factory, module: null, error: null });
	}
	function requireModule(url) {
		const m = modules.get(url);
		if (!m) throw new TypeError("the module is not loaded: " + url);
		if (m.error) throw m.error;
		if (m.module) return m.module.exports; // done, or being evaluated (a cycle)
		m.module = { exports: {} };
		const req = spec => {
			const u = m.deps[spec];
			if (u === undefined) throw new TypeError("Failed to resolve module specifier \"" + spec + "\" from " + url);
			return requireModule(u);
		};
		try {
			m.factory.call(undefined, m.module.exports, req, m.module);
		} catch (e) {
			m.error = e;
			throw e;
		}
		return m.module.exports;
	}

	// ---- URL ----

	class URLSearchParams {
		constructor(init) {
			this._list = [];
			if (init === undefined || init === null) return;
			const decode = s => { try { return decodeURIComponent(s.replace(/\+/g, " ")); } catch (e) { return s; } };
			if (typeof init === "string") {
				const s = init.charAt(0) === "?" ? init.slice(1) : init;
				for (const part of s.split("&")) {
					if (!part) continue;
					const i = part.indexOf("=");
					const k = i < 0 ? part : part.slice(0, i);
					const v = i < 0 ? "" : part.slice(i + 1);
					this._list.push([decode(k), decode(v)]);
				}
			} else if (typeof init[Symbol.iterator] === "function") {
				for (const [k, v] of init) this._list.push([String(k), String(v)]);
			} else {
				for (const k of Object.keys(init)) this._list.push([k, String(init[k])]);
			}
		}
		_update() { if (this._url) { const s = this.toString(); this._url._set("search", s ? "?" + s : ""); } }
		append(k, v) { this._list.push([String(k), String(v)]); this._update(); }
		delete(k) { this._list = this._list.filter(x => x[0] !== k); this._update(); }
		get(k) { const x = this._list.find(x => x[0] === k); return x ? x[1] : null; }
		getAll(k) { return this._list.filter(x => x[0] === k).map(x => x[1]); }
		has(k) { return this._list.some(x => x[0] === k); }
		set(k, v) {
			const i = this._list.findIndex(x => x[0] === k);
			if (i < 0) this._list.push([String(k), String(v)]);
			else { this._list[i][1] = String(v); this._list = this._list.filter((x, j) => j <= i || x[0] !== k); }
			this._update();
		}
		sort() { this._list.sort((a, b) => a[0] < b[0] ? -1 : a[0] > b[0] ? 1 : 0); this._update(); }
		forEach(f, t) { for (const [k, v] of this._list) f.call(t, v, k, this); }
		keys() { return this._list.map(x => x[0])[Symbol.iterator](); }
		values() { return this._list.map(x => x[1])[Symbol.iterator](); }
		entries() { return this._list.map(x => [x[0], x[1]])[Symbol.iterator](); }
		[Symbol.iterator]() { return this.entries(); }
		get size() { return this._list.length; }
		toString() {
			const enc = s => encodeURIComponent(s).replace(/%20/g, "+").replace(/[!'()~]/g, c => "%" + c.charCodeAt(0).toString(16).toUpperCase());
			return this._list.map(x => enc(x[0]) + "=" + enc(x[1])).join("&");
		}
	}

	const urlFields = ["href", "protocol", "username", "password", "host", "hostname", "port", "pathname", "search", "hash", "origin"];
	class URL {
		constructor(u, base) {
			const p = D.parseURL(String(u), base === undefined ? undefined : String(base));
			if (!p) throw new TypeError("Invalid URL: " + u);
			this._parts = p;
			this._search = null;
		}
		static canParse(u, base) { return !!D.parseURL(String(u), base === undefined ? undefined : String(base)); }
		static createObjectURL(b) {
			if (b instanceof MediaSource) return b._url;
			return b instanceof Blob ? D.createObjectURL(b._data, b.type || "") : "blob:" + D.pageURL() + "#" + (++callbackID);
		}
		// (the URL of a MediaSource stays usable by the player of the page,
		// as players revoke it once the source is open)
		static revokeObjectURL(u) { if (!mediaSourcesByURL.has(String(u))) D.revokeObjectURL(String(u)); }
		_get(i) { return this._parts[i]; }
		_set(field, v) {
			const p = this._parts;
			let [, protocol, user, pass, host, , , path, search, hash] = p;
			v = String(v);
			switch (field) {
				case "href": { const n = D.parseURL(v); if (!n) throw new TypeError("Invalid URL"); this._parts = n; return; }
				case "protocol": protocol = v.replace(/:?$/, ":"); break;
				case "username": user = v; break;
				case "password": pass = v; break;
				case "host": host = v; break;
				case "hostname": host = v + (p[6] ? ":" + p[6] : ""); break;
				case "port": host = p[5] + (v ? ":" + v : ""); break;
				case "pathname": path = v.charAt(0) === "/" ? v : "/" + v; break;
				case "search": search = v && v.charAt(0) !== "?" ? "?" + v : (v === "?" ? "" : v); break;
				case "hash": hash = v && v.charAt(0) !== "#" ? "#" + v : (v === "#" ? "" : v); break;
			}
			const auth = user ? user + (pass ? ":" + pass : "") + "@" : "";
			const n = D.parseURL(protocol + "//" + auth + host + path + search + hash);
			if (n) this._parts = n;
		}
		get searchParams() {
			if (!this._search) { this._search = new URLSearchParams(this.search); this._search._url = this; }
			return this._search;
		}
		toString() { return this.href; }
		toJSON() { return this.href; }
	}
	urlFields.forEach((f, i) => {
		Object.defineProperty(URL.prototype, f, {
			get() { return this._get(i); },
			set(v) { this._set(f, v); if (f === "search" || f === "href") this._search = null; },
		});
	});

	// ---- fetch and XMLHttpRequest ----

	class Headers {
		constructor(init) {
			this._m = new Map();
			if (!init) return;
			if (init instanceof Headers) init.forEach((v, k) => this.append(k, v));
			else if (typeof init[Symbol.iterator] === "function") for (const [k, v] of init) this.append(k, v);
			else for (const k of Object.keys(init)) this.append(k, init[k]);
		}
		append(k, v) { k = String(k).toLowerCase(); const c = this._m.get(k); this._m.set(k, c === undefined ? String(v) : c + ", " + v); }
		set(k, v) { this._m.set(String(k).toLowerCase(), String(v)); }
		get(k) { const v = this._m.get(String(k).toLowerCase()); return v === undefined ? null : v; }
		has(k) { return this._m.has(String(k).toLowerCase()); }
		delete(k) { this._m.delete(String(k).toLowerCase()); }
		forEach(f, t) { for (const [k, v] of this._m) f.call(t, v, k, this); }
		keys() { return this._m.keys(); }
		values() { return this._m.values(); }
		entries() { return this._m.entries(); }
		[Symbol.iterator]() { return this._m.entries(); }
		_flat() { const a = []; for (const [k, v] of this._m) a.push(k, v); return a; }
	}

	function bodyBytes(body) {
		if (body === undefined || body === null) return undefined;
		if (typeof body === "string") return body;
		if (body instanceof URLSearchParams) return body.toString();
		if (body instanceof FormData) { const q = new URLSearchParams(); for (const [k, v] of body) q.append(k, typeof v === "string" ? v : ""); return q.toString(); }
		if (body instanceof Blob) return body._data;
		if (body instanceof ArrayBuffer) return body;
		if (ArrayBuffer.isView(body)) return body.buffer.slice(body.byteOffset, body.byteOffset + body.byteLength);
		return String(body);
	}

	class Body {
		_consume() {
			if (this.bodyUsed) return Promise.reject(new TypeError("the body has been used"));
			this.bodyUsed = true;
			return Promise.resolve(this._body === undefined ? "" : this._body);
		}
		text() { return this._consume().then(b => typeof b === "string" ? b : D.decodeUTF8(b)); }
		json() { return this.text().then(t => JSON.parse(t)); }
		arrayBuffer() { return this._consume().then(b => typeof b === "string" ? D.encodeUTF8(b) : b); }
		blob() { return this._consume().then(b => new Blob([b], { type: this.headers.get("content-type") || "" })); }
		formData() { return this.text().then(t => { const f = new FormData(); for (const [k, v] of new URLSearchParams(t)) f.append(k, v); return f; }); }
		get body() { return null; }
	}
	// the state of each Request, which fetch uses, not its properties,
	// which scripts can redefine (as a check of youtube.com for blockers of
	// its ads, which redefines the url and method of a Request of a data:
	// URL, expecting its data from fetch, and which posted to the API of its
	// player instead)
	const requestState = new WeakMap();
	class Request extends Body {
		constructor(input, init) {
			super();
			init = init || {};
			const from = input instanceof Request ? requestState.get(input) : null;
			const st = from ? { url: from.url, method: from.method, headers: new Headers(from.headers), body: from.body, signal: from.signal, credentials: from.credentials, mode: from.mode, cache: from.cache }
				: { url: new URL(String(input), D.baseURL()).href, method: "GET", headers: new Headers(), body: undefined, signal: null, credentials: "same-origin", mode: "cors", cache: "default" };
			if (init.method) st.method = String(init.method).toUpperCase();
			if (init.headers) st.headers = new Headers(init.headers);
			if (init.body !== undefined) st.body = bodyBytes(init.body);
			if (init.signal !== undefined) st.signal = init.signal || null;
			if (init.credentials) st.credentials = init.credentials;
			if (init.mode) st.mode = init.mode;
			if (init.cache) st.cache = init.cache;
			requestState.set(this, st);
			this._body = st.body;
			this.bodyUsed = false;
		}
		get url() { return requestState.get(this).url; }
		get method() { return requestState.get(this).method; }
		get headers() { return requestState.get(this).headers; }
		get signal() { return requestState.get(this).signal; }
		get credentials() { return requestState.get(this).credentials; }
		get mode() { return requestState.get(this).mode; }
		get cache() { return requestState.get(this).cache; }
		clone() { return new Request(this); }
	}
	class Response extends Body {
		constructor(body, init) {
			super();
			init = init || {};
			this._body = bodyBytes(body);
			this.status = init.status === undefined ? 200 : init.status;
			this.statusText = init.statusText || "";
			this.headers = new Headers(init.headers);
			this.ok = this.status >= 200 && this.status < 300;
			this.url = "";
			this.redirected = false;
			this.type = "basic";
			this.bodyUsed = false;
		}
		clone() { const r = new Response(this._body, { status: this.status, statusText: this.statusText, headers: this.headers }); r.url = this.url; return r; }
		static json(data, init) { const h = new Headers(init && init.headers); h.set("content-type", "application/json"); return new Response(JSON.stringify(data), Object.assign({}, init, { headers: h })); }
		static error() { const r = new Response(null, { status: 0 }); r.type = "error"; return r; }
		static redirect(u, s) { return new Response(null, { status: s || 302, headers: { location: u } }); }
	}

	// goFetch makes the given request, with the cookies of other origins if
	// credentials (as fetch with credentials: "include", and XMLHttpRequest
	// withCredentials, as the APIs of Google on clients6.google.com).
	function goFetch(url, method, headers, body, cb, credentials) {
		D.fetch(newCallback(cb), url, method, headers, body, !!credentials);
	}

	// noteEmptyBody logs a request of a method with a body (as POST) whose
	// body is empty, with the type of the body that the script gave and
	// where, as such a request of youtube.com (to its API of the player)
	// was rejected, with no body (400 Bad Request).
	function noteEmptyBody(method, url, given, body) {
		if (method === "GET" || method === "HEAD") return;
		if (body !== undefined && (typeof body === "string" ? body.length : body.byteLength) > 0) return;
		let type = given === undefined ? "undefined" : given === null ? "null" : typeof given;
		if (given && typeof given === "object") type = (given.constructor && given.constructor.name) || "object";
		const stack = String(new Error().stack || "").split("\n").slice(2, 8).join("\n");
		D.log("debug", method + " " + url + " with an empty body (given: " + type + ")\n" + stack);
	}

	function fetch(input, init) {
		let req;
		try { req = requestState.get(new Request(input, init)); } catch (e) { return Promise.reject(e); }
		return new Promise((resolve, reject) => {
			if (req.signal && req.signal.aborted) { reject(domException("AbortError", "the request was aborted")); return; }
			let done = false;
			if (req.signal) req.signal.addEventListener("abort", () => { if (!done) { done = true; reject(domException("AbortError", "the request was aborted")); } });
			noteEmptyBody(req.method, req.url, init && init.body !== undefined ? init.body : input instanceof Request ? requestState.get(input).body : undefined, req.body);
			goFetch(req.url, req.method, req.headers._flat(), req.body, (status, statusText, headers, body, final, err) => {
				if (done) return;
				done = true;
				if (err) { reject(new TypeError("Failed to fetch: " + err)); return; }
				const h = new Headers();
				for (let i = 0; i + 1 < headers.length; i += 2) h.append(headers[i], headers[i + 1]);
				const r = new Response(body, { status, statusText, headers: h });
				r.url = final;
				r.redirected = final !== req.url;
				resolve(r);
			}, req.credentials === "include");
		});
	}

	class XMLHttpRequest extends EventTarget {
		constructor() {
			super();
			this.readyState = 0; this.status = 0; this.statusText = ""; this.responseText = ""; this.response = null;
			this.responseType = ""; this.responseURL = ""; this.timeout = 0; this.withCredentials = false;
			this._headers = new Headers(); this._respHeaders = new Headers(); this._aborted = false;
			this.upload = new EventTarget();
		}
		open(method, url) {
			this._method = String(method).toUpperCase();
			this._url = new URL(String(url), D.baseURL()).href;
			this._aborted = false;
			this._setState(1);
		}
		setRequestHeader(k, v) { this._headers.append(k, v); }
		getResponseHeader(k) { return this._respHeaders.get(k); }
		getAllResponseHeaders() { let s = ""; this._respHeaders.forEach((v, k) => { s += k + ": " + v + "\r\n"; }); return s; }
		overrideMimeType() {}
		_setState(s) { this.readyState = s; this.dispatchEvent(new Event("readystatechange")); }
		send(body) {
			this.dispatchEvent(new ProgressEvent("loadstart"));
			const bytes = bodyBytes(body);
			noteEmptyBody(this._method, this._url, body, bytes);
			goFetch(this._url, this._method, this._headers._flat(), bytes, (status, statusText, headers, data, final, err) => {
				if (this._aborted) return;
				if (err) {
					this.readyState = 4; this.status = 0;
					this.dispatchEvent(new Event("readystatechange"));
					this.dispatchEvent(new ProgressEvent("error"));
					this.dispatchEvent(new ProgressEvent("loadend"));
					return;
				}
				for (let i = 0; i + 1 < headers.length; i += 2) this._respHeaders.append(headers[i], headers[i + 1]);
				this.status = status; this.statusText = statusText; this.responseURL = final;
				this._setState(2);
				this._setState(3);
				const text = D.decodeUTF8(data);
				this.responseText = text;
				switch (this.responseType) {
					case "json": try { this.response = JSON.parse(text); } catch (e) { this.response = null; } break;
					case "arraybuffer": this.response = data; break;
					case "blob": this.response = new Blob([data]); break;
					case "document": this.response = new DOMParser().parseFromString(text, "text/html"); break;
					default: this.response = text;
				}
				this._setState(4);
				this.dispatchEvent(new ProgressEvent("load", { loaded: text.length, total: text.length, lengthComputable: true }));
				this.dispatchEvent(new ProgressEvent("loadend"));
			}, this.withCredentials);
		}
		abort() { this._aborted = true; this.readyState = 0; this.dispatchEvent(new ProgressEvent("abort")); }
	}
	XMLHttpRequest.UNSENT = 0; XMLHttpRequest.OPENED = 1; XMLHttpRequest.HEADERS_RECEIVED = 2; XMLHttpRequest.LOADING = 3; XMLHttpRequest.DONE = 4;
	function defineSimpleHandlers(cls, types) {
		for (const t of types) {
			Object.defineProperty(cls.prototype, "on" + t, {
				get() { return this["_on" + t] || null; },
				set(v) { this["_on" + t] = v; },
			});
		}
	}
	defineSimpleHandlers(XMLHttpRequest, ["readystatechange", "load", "error", "abort", "loadend", "loadstart", "progress", "timeout"]);

	class Blob {
		constructor(parts, opts) {
			let s = "";
			for (const p of parts || []) s += typeof p === "string" ? p : p instanceof Blob ? p._data : (p instanceof ArrayBuffer || ArrayBuffer.isView(p)) ? D.decodeUTF8(p) : String(p);
			this._data = s;
			this.type = (opts && opts.type) || "";
		}
		get size() { return this._data.length; }
		text() { return Promise.resolve(this._data); }
		arrayBuffer() { return Promise.resolve(D.encodeUTF8(this._data)); }
		slice(a, b, t) { return new Blob([this._data.slice(a, b)], { type: t }); }
	}
	class File extends Blob { constructor(parts, name, opts) { super(parts, opts); this.name = name; this.lastModified = Date.now(); } }
	class FileReader extends EventTarget {
		_read(b, f) { Promise.resolve().then(() => { this.result = f(b); this.readyState = 2; this.dispatchEvent(new ProgressEvent("load")); this.dispatchEvent(new ProgressEvent("loadend")); }); }
		readAsText(b) { this._read(b, x => x._data); }
		readAsDataURL(b) { this._read(b, x => "data:" + (x.type || "application/octet-stream") + ";base64," + D.btoa(unescape(encodeURIComponent(x._data)))); }
		readAsArrayBuffer(b) { this._read(b, x => D.encodeUTF8(x._data)); }
	}
	defineSimpleHandlers(FileReader, ["load", "error", "loadend", "abort"]);

	class FormData {
		constructor(form) {
			this._list = [];
			if (!form) return;
			for (const el of form.elements) {
				const name = el.getAttribute("name");
				if (!name || el.disabled) continue;
				const t = el.type;
				if ((t === "checkbox" || t === "radio") && !el.checked) continue;
				if (t === "submit" || t === "button" || t === "reset" || t === "image" || t === "file" || el.tagName === "BUTTON" || el.tagName === "FIELDSET") continue;
				this._list.push([name, el.value]);
			}
		}
		append(k, v) { this._list.push([String(k), v instanceof Blob ? v : String(v)]); }
		set(k, v) { this.delete(k); this.append(k, v); }
		get(k) { const x = this._list.find(x => x[0] === k); return x ? x[1] : null; }
		getAll(k) { return this._list.filter(x => x[0] === k).map(x => x[1]); }
		has(k) { return this._list.some(x => x[0] === k); }
		delete(k) { this._list = this._list.filter(x => x[0] !== k); }
		forEach(f, t) { for (const [k, v] of this._list) f.call(t, v, k, this); }
		keys() { return this._list.map(x => x[0])[Symbol.iterator](); }
		values() { return this._list.map(x => x[1])[Symbol.iterator](); }
		entries() { return this._list.map(x => [x[0], x[1]])[Symbol.iterator](); }
		[Symbol.iterator]() { return this.entries(); }
	}

	class AbortSignal extends EventTarget {
		constructor() { super(); this.aborted = false; this.reason = undefined; }
		throwIfAborted() { if (this.aborted) throw this.reason; }
		static abort(r) { const c = new AbortController(); c.abort(r); return c.signal; }
		static timeout(ms) { const c = new AbortController(); setTimer(() => c.abort(domException("TimeoutError", "timed out")), ms, [], false); return c.signal; }
	}
	defineHandlers(AbortSignal.prototype);
	class AbortController {
		constructor() { this.signal = new AbortSignal(); }
		abort(r) {
			if (this.signal.aborted) return;
			this.signal.aborted = true;
			this.signal.reason = r === undefined ? domException("AbortError", "aborted") : r;
			this.signal.dispatchEvent(new Event("abort"));
		}
	}

	class TextEncoder {
		get encoding() { return "utf-8"; }
		encode(s) { return new Uint8Array(D.encodeUTF8(String(s === undefined ? "" : s))); }
		encodeInto(s, dst) { const b = this.encode(s); const n = Math.min(b.length, dst.length); dst.set(b.subarray(0, n)); return { read: s.length, written: n }; }
	}
	class TextDecoder {
		constructor(enc) { this.encoding = (enc || "utf-8").toLowerCase(); }
		decode(b) { if (b === undefined) return ""; return D.decodeUTF8(b); }
	}

	class DOMParser {
		parseFromString(s) { return wrap(D.parseDocument(String(s))); }
	}
	class XMLSerializer {
		serializeToString(n) { return n.nodeType === 9 ? D.inner(n[ID]) : D.outer(n[ID]); }
	}

	// observers: mutations are not reported (MutationObserver), and every
	// observed element is taken to be visible, of its laid out size
	// MutationObserver reports the changes of the DOM in the nodes that it
	// observes (and in their subtrees, with subtree), in records that are
	// delivered together after the task that made them (as the selector
	// observer of GitHub, which schedules its work with the change of an
	// attribute of a detached element, and polls for the sign in with
	// GitHub Mobile when its element is added: it never did).
	class MutationRecord {
		constructor(type, target, init) {
			Object.assign(this, { type, target, addedNodes: [], removedNodes: [], previousSibling: null, nextSibling: null, attributeName: null, attributeNamespace: null, oldValue: null }, init);
		}
	}
	class MutationObserver {
		constructor(cb) { this._cb = cb; this._targets = new Map(); this._records = []; }
		observe(target, opts) {
			opts = Object.assign({}, opts);
			if (opts.attributeOldValue || opts.attributeFilter) opts.attributes = opts.attributes !== false;
			if (opts.characterDataOldValue) opts.characterData = opts.characterData !== false;
			if (!opts.childList && !opts.attributes && !opts.characterData) throw new TypeError("the options observe nothing");
			this._targets.set(idOf(target), opts);
			mutationObservers.add(this);
		}
		disconnect() { this._targets.clear(); this._records = []; mutationObservers.delete(this); }
		takeRecords() { const r = this._records; this._records = []; return r; }
		// _record records the change of the given type of the node of the
		// given id, if it observes it.
		_record(type, id, init) {
			for (let n = id, own = true; n; n = D.parent(n) || shadowHosts.get(n), own = false) {
				const o = this._targets.get(n);
				if (!o || (!own && !o.subtree)) continue;
				if (type === "attributes" && (!o.attributes || (o.attributeFilter && !o.attributeFilter.includes(init.attributeName)))) continue;
				if (type === "childList" && !o.childList) continue;
				if (type === "characterData" && !o.characterData) continue;
				const rec = new MutationRecord(type, wrap(id), init);
				if (!(type === "attributes" ? o.attributeOldValue : type === "characterData" ? o.characterDataOldValue : false)) rec.oldValue = null;
				this._records.push(rec);
				if (!this._pending) {
					this._pending = true;
					queueMicrotask(() => {
						this._pending = false;
						const records = this.takeRecords();
						if (records.length) this._cb(records, this);
					});
				}
				return;
			}
		}
	}
	class IntersectionObserver {
		constructor(cb, opts) { this._cb = cb; this.root = (opts && opts.root) || null; this.rootMargin = (opts && opts.rootMargin) || "0px"; this.thresholds = [0]; this._targets = []; }
		observe(t) {
			this._targets.push(t);
			setTimer(() => {
				if (this._targets.indexOf(t) < 0) return;
				const r = t.getBoundingClientRect();
				this._cb([{ target: t, isIntersecting: true, intersectionRatio: 1, boundingClientRect: r, intersectionRect: r, rootBounds: null, time: performance.now() }], this);
			}, 0, [], false);
		}
		unobserve(t) { this._targets = this._targets.filter(x => x !== t); }
		disconnect() { this._targets = []; }
		takeRecords() { return []; }
	}
	// ResizeObserver reports the size of each target when it is observed,
	// and then when it changes, which a timer shared by all of them checks
	// for while any are observed (as the header of edition.cnn.com, which
	// sizes itself by its ad slot as that is laid out).
	const resizeObservers = new Set();
	let resizeTimer = 0;
	const checkResizes = () => {
		resizeTimer = 0;
		for (const ro of resizeObservers) ro._check(false);
		if (resizeObservers.size > 0) resizeTimer = setTimer(checkResizes, 250, [], false);
	};
	class ResizeObserver {
		constructor(cb) { this._cb = cb; this._targets = []; this._sizes = new Map(); }
		observe(t) {
			if (this._targets.indexOf(t) < 0) this._targets.push(t);
			this._sizes.delete(t);
			resizeObservers.add(this);
			setTimer(() => this._check(true), 0, [], false);
			if (!resizeTimer) resizeTimer = setTimer(checkResizes, 250, [], false);
		}
		// _check reports the targets whose size changed since they were
		// last reported (all of those not yet reported, if first).
		_check(first) {
			const entries = [];
			for (const t of this._targets) {
				const r = t.getBoundingClientRect();
				const last = this._sizes.get(t);
				if (last && last[0] === r.width && last[1] === r.height) continue;
				if (!last && !first) continue;
				this._sizes.set(t, [r.width, r.height]);
				const size = [{ inlineSize: r.width, blockSize: r.height }];
				entries.push({ target: t, contentRect: r, borderBoxSize: size, contentBoxSize: size, devicePixelContentBoxSize: size });
			}
			if (entries.length > 0) this._cb(entries, this);
		}
		unobserve(t) { this._targets = this._targets.filter(x => x !== t); this._sizes.delete(t); if (this._targets.length === 0) resizeObservers.delete(this); }
		disconnect() { this._targets = []; this._sizes.clear(); resizeObservers.delete(this); }
	}
	class PerformanceObserver { observe() {} disconnect() {} static get supportedEntryTypes() { return []; } }

	// ---- Intl ----

	// Intl is a minimal implementation of the internationalization API
	// (which goja lacks) for English (en-US) only, in the local time zone.
	const Intl = (() => {
		const locale = "en-US";
		const supportedLocalesOf = () => [locale];
		const months = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"];
		const days = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];
		const group = s => s.replace(/\B(?=(\d{3})+(?!\d))/g, ",");
		const symbols = { USD: "$", EUR: "€", GBP: "£", JPY: "¥", INR: "₹", CNY: "CN¥", CAD: "CA$", AUD: "A$" };

		class NumberFormat {
			constructor(loc, opts) {
				this._o = Object.assign({ style: "decimal", useGrouping: true, notation: "standard" }, opts || {});
				const o = this._o;
				const cur = o.style === "currency";
				if (o.minimumFractionDigits === undefined) o.minimumFractionDigits = cur ? 2 : 0;
				if (o.maximumFractionDigits === undefined) o.maximumFractionDigits = Math.max(o.minimumFractionDigits, cur ? 2 : o.style === "percent" ? 0 : 3);
			}
			format(n) { return this.formatToParts(n).map(p => p.value).join(""); }
			formatToParts(n) {
				const o = this._o;
				n = Number(n);
				if (o.style === "percent") n *= 100;
				let suffix = "";
				if (o.notation === "compact" && Math.abs(n) >= 1000) {
					const units = o.compactDisplay === "long" ? [" thousand", " million", " billion", " trillion"] : ["K", "M", "B", "T"];
					let i = -1;
					while (Math.abs(n) >= 1000 && i < 3) { n /= 1000; i++; }
					suffix = units[i];
					// two significant digits below 100, as 1.2K and 12K
					if (o.maximumSignificantDigits === undefined) n = Math.abs(n) < 100 ? Number(n.toPrecision(2)) : Math.round(n);
				}
				const parts = [];
				if (n < 0) { parts.push({ type: "minusSign", value: "-" }); n = -n; }
				if (o.style === "currency") parts.push({ type: "currency", value: o.currencyDisplay === "code" ? o.currency + " " : symbols[o.currency] || o.currency + " " });
				if (!isFinite(n)) parts.push({ type: isNaN(n) ? "nan" : "infinity", value: isNaN(n) ? "NaN" : "∞" });
				else {
					let s;
					if (o.maximumSignificantDigits !== undefined) s = String(Number(n.toPrecision(o.maximumSignificantDigits)));
					else s = n.toFixed(o.maximumFractionDigits);
					let [int, frac = ""] = s.split(".");
					if (o.maximumSignificantDigits === undefined) {
						frac = frac.replace(/0+$/, "");
						while (frac.length < o.minimumFractionDigits) frac += "0";
					}
					parts.push({ type: "integer", value: o.useGrouping === false ? int : group(int) });
					if (frac) parts.push({ type: "decimal", value: "." }, { type: "fraction", value: frac });
				}
				if (suffix) parts.push({ type: "compact", value: suffix });
				if (o.style === "percent") parts.push({ type: "percentSign", value: "%" });
				if (o.style === "unit" && o.unit) parts.push({ type: "unit", value: " " + o.unit });
				return parts;
			}
			resolvedOptions() { return Object.assign({ locale, numberingSystem: "latn" }, this._o); }
			static supportedLocalesOf() { return supportedLocalesOf(); }
		}

		const pad = (n, w) => String(n).padStart(w, "0");
		class DateTimeFormat {
			constructor(loc, opts) {
				const o = Object.assign({}, opts || {});
				const fields = ["weekday", "year", "month", "day", "hour", "minute", "second"];
				if (o.dateStyle || o.timeStyle) {
					const ds = { full: { weekday: "long", year: "numeric", month: "long", day: "numeric" }, long: { year: "numeric", month: "long", day: "numeric" }, medium: { year: "numeric", month: "short", day: "numeric" }, short: { year: "2-digit", month: "numeric", day: "numeric" } };
					const ts = { full: { hour: "numeric", minute: "2-digit", second: "2-digit" }, long: { hour: "numeric", minute: "2-digit", second: "2-digit" }, medium: { hour: "numeric", minute: "2-digit", second: "2-digit" }, short: { hour: "numeric", minute: "2-digit" } };
					Object.assign(o, ds[o.dateStyle] || {}, ts[o.timeStyle] || {});
				} else if (!fields.some(f => o[f] !== undefined)) Object.assign(o, { year: "numeric", month: "numeric", day: "numeric" });
				this._o = o;
			}
			formatToParts(d) {
				d = d === undefined ? new Date() : new Date(typeof d === "number" ? d : d.valueOf());
				const o = this._o, utc = o.timeZone === "UTC";
				const get = f => utc ? d["getUTC" + f]() : d["get" + f]();
				const parts = [];
				const lit = v => parts.push({ type: "literal", value: v });
				const textMonth = o.month === "long" || o.month === "short" || o.month === "narrow";
				const m = get("Month"), day = get("Date"), y = get("FullYear");
				if (o.weekday) { const w = days[get("Day")]; parts.push({ type: "weekday", value: o.weekday === "long" ? w : w.slice(0, 3) }); if (o.year || o.month || o.day) lit(", "); }
				if (textMonth) {
					const name = o.month === "long" ? months[m] : months[m].slice(0, 3);
					parts.push({ type: "month", value: name });
					if (o.day) { lit(" "); parts.push({ type: "day", value: o.day === "2-digit" ? pad(day, 2) : String(day) }); }
					if (o.year) { lit(o.day ? ", " : " "); parts.push({ type: "year", value: o.year === "2-digit" ? pad(y % 100, 2) : String(y) }); }
				} else {
					const ds = [];
					if (o.month) ds.push({ type: "month", value: o.month === "2-digit" ? pad(m + 1, 2) : String(m + 1) });
					if (o.day) ds.push({ type: "day", value: o.day === "2-digit" ? pad(day, 2) : String(day) });
					if (o.year) ds.push({ type: "year", value: o.year === "2-digit" ? pad(y % 100, 2) : String(y) });
					ds.forEach((p, i) => { if (i) lit("/"); parts.push(p); });
				}
				if (o.hour || o.minute || o.second) {
					if (parts.length) lit(", ");
					const h24 = get("Hours"), h12 = o.hour12 === false || o.hourCycle === "h23" ? null : (h24 % 12 || 12);
					const ts = [];
					if (o.hour) ts.push({ type: "hour", value: h12 === null ? pad(h24, 2) : o.hour === "2-digit" ? pad(h12, 2) : String(h12) });
					if (o.minute) ts.push({ type: "minute", value: pad(get("Minutes"), 2) });
					if (o.second) ts.push({ type: "second", value: pad(get("Seconds"), 2) });
					ts.forEach((p, i) => { if (i) lit(":"); parts.push(p); });
					if (o.hour && h12 !== null) { lit(" "); parts.push({ type: "dayPeriod", value: h24 < 12 ? "AM" : "PM" }); }
					if (o.timeZoneName) { lit(" "); parts.push({ type: "timeZoneName", value: utc ? "UTC" : "local" }); }
				}
				return parts;
			}
			format(d) { return this.formatToParts(d).map(p => p.value).join(""); }
			formatRange(a, b) { return this.format(a) + " – " + this.format(b); }
			resolvedOptions() { return Object.assign({ locale, calendar: "gregory", numberingSystem: "latn", timeZone: this._o.timeZone || "UTC" }, this._o); }
			static supportedLocalesOf() { return supportedLocalesOf(); }
		}

		class Collator {
			constructor(loc, opts) { this._o = Object.assign({ sensitivity: "variant", numeric: false }, opts || {}); this.compare = this.compare.bind(this); }
			compare(a, b) {
				a = String(a); b = String(b);
				if (this._o.sensitivity === "base" || this._o.sensitivity === "accent") { a = a.toLowerCase(); b = b.toLowerCase(); }
				if (this._o.numeric) {
					const na = a.match(/\d+|\D+/g) || [], nb = b.match(/\d+|\D+/g) || [];
					for (let i = 0; i < Math.min(na.length, nb.length); i++) {
						const x = na[i], y = nb[i];
						if (/^\d/.test(x) && /^\d/.test(y)) { if (+x !== +y) return +x < +y ? -1 : 1; }
						else if (x !== y) return x.localeCompare(y);
					}
					return na.length - nb.length;
				}
				return a < b ? -1 : a > b ? 1 : 0;
			}
			resolvedOptions() { return Object.assign({ locale, usage: "sort" }, this._o); }
			static supportedLocalesOf() { return supportedLocalesOf(); }
		}

		class PluralRules {
			constructor(loc, opts) { this._o = Object.assign({ type: "cardinal" }, opts || {}); }
			select(n) {
				n = Number(n);
				if (this._o.type === "ordinal") {
					const t = n % 10, h = n % 100;
					return t === 1 && h !== 11 ? "one" : t === 2 && h !== 12 ? "two" : t === 3 && h !== 13 ? "few" : "other";
				}
				return n === 1 ? "one" : "other";
			}
			resolvedOptions() { return Object.assign({ locale, pluralCategories: ["one", "other"] }, this._o); }
			static supportedLocalesOf() { return supportedLocalesOf(); }
		}

		class RelativeTimeFormat {
			constructor(loc, opts) { this._o = Object.assign({ numeric: "always", style: "long" }, opts || {}); }
			format(v, unit) {
				v = Number(v);
				unit = String(unit).replace(/s$/, "");
				if (this._o.numeric === "auto") {
					const words = { day: ["yesterday", "today", "tomorrow"], year: ["last year", "this year", "next year"], month: ["last month", "this month", "next month"], week: ["last week", "this week", "next week"] };
					if (words[unit] && Math.abs(v) <= 1) return words[unit][v + 1];
					if (unit === "second" && v === 0) return "now";
				}
				const n = Math.abs(v);
				const s = new NumberFormat().format(n) + " " + unit + (n === 1 ? "" : "s");
				return v < 0 || Object.is(v, -0) ? s + " ago" : "in " + s;
			}
			formatToParts(v, unit) { return [{ type: "literal", value: this.format(v, unit) }]; }
			resolvedOptions() { return Object.assign({ locale, numberingSystem: "latn" }, this._o); }
			static supportedLocalesOf() { return supportedLocalesOf(); }
		}

		class ListFormat {
			constructor(loc, opts) { this._o = Object.assign({ type: "conjunction", style: "long" }, opts || {}); }
			format(list) {
				const a = Array.from(list, String);
				const word = this._o.type === "disjunction" ? "or" : "and";
				if (a.length < 2) return a.join("");
				if (a.length === 2) return a[0] + " " + word + " " + a[1];
				return a.slice(0, -1).join(", ") + ", " + word + " " + a[a.length - 1];
			}
			static supportedLocalesOf() { return supportedLocalesOf(); }
		}

		class Segmenter {
			constructor(loc, opts) { this._o = Object.assign({ granularity: "grapheme" }, opts || {}); }
			segment(str) {
				str = String(str);
				const g = this._o.granularity;
				const re = g === "word" ? /\w+|\s+|[^\w\s]/gu : g === "sentence" ? /[^.!?]+[.!?]*\s*/g : /./gsu;
				const segs = [];
				let m;
				while ((m = re.exec(str)) !== null) segs.push({ segment: m[0], index: m.index, input: str, isWordLike: g === "word" ? /^\w/.test(m[0]) : undefined });
				return { [Symbol.iterator]() { return segs[Symbol.iterator](); }, containing(i) { return segs.find(s => i >= s.index && i < s.index + s.segment.length); } };
			}
			static supportedLocalesOf() { return supportedLocalesOf(); }
		}

		class Locale {
			constructor(tag) { this.baseName = String(tag || locale); [this.language, this.region] = this.baseName.split("-"); }
			toString() { return this.baseName; }
			maximize() { return this; }
			minimize() { return this; }
		}

		class DisplayNames {
			constructor(loc, opts) { this._o = opts || {}; }
			of(code) { return String(code); }
		}

		return {
			NumberFormat, DateTimeFormat, Collator, PluralRules, RelativeTimeFormat, ListFormat, Segmenter, Locale, DisplayNames,
			getCanonicalLocales: l => (l === undefined ? [] : Array.isArray(l) ? l : [l]).map(String),
			supportedValuesOf: () => [],
		};
	})();
	// the locale strings of numbers are of Intl.NumberFormat, as in
	// browsers: goja took the locale as a radix ((1234).toLocaleString("en-US")
	// threw a RangeError, which broke the pull request list of GitHub)
	Object.defineProperty(Number.prototype, "toLocaleString", {
		value: function toLocaleString(locales, options) { return new Intl.NumberFormat(locales, options).format(Number(this)); },
		writable: true, configurable: true,
	});

	// ---- MessageChannel ----

	// MessagePort sends messages to the other port of its channel, in
	// tasks (as React's scheduler yields with them).
	// A port can be transferred to another engine (see ports.go): its
	// partner then posts to the end of the port there (_remote).
	class MessagePort extends EventTarget {
		constructor() { super(); this._other = null; this._remote = 0; this._closed = false; this.onmessage = null; }
		// postMessage posts the given message to the entangled port, with
		// the ports of the given transfer list (an array, or the transfer
		// of options), which move with it (as Google replies to the
		// messages of its account menu through the port of each message)
		postMessage(data, transfer) {
			if (this._closed) return;
			const ports = (Array.isArray(transfer) ? transfer : (transfer && transfer.transfer) || []).filter(p => p instanceof MessagePort);
			if (this._remote) {
				D.portPost(this._remote, cloneJSON(data), transferIDs(ports));
				return;
			}
			const to = this._other;
			if (!to) return;
			setTimer(() => to._deliver(data, ports), 0, [], false);
		}
		_deliver(data, ports) {
			if (this._closed) {
				// a message that was on its way to a port that has moved to
				// another engine follows it there, as in browsers
				const via = this._movedVia;
				if (via && via._remote) D.portPost(via._remote, cloneJSON(data), transferIDs(ports));
				return;
			}
			const ev = new MessageEvent("message", { data, ports: ports || [] });
			if (typeof this.onmessage === "function") {
				try { this.onmessage(ev); } catch (e) { reportError(e); }
			}
			this.dispatchEvent(ev);
		}
		start() {}
		close() { this._closed = true; }
	}
	const localPorts = new Map(); // the ports that are linked to other engines, by their numbers
	let portCount = 0;
	const portID = p => { if (!p._id) { p._id = ++portCount; localPorts.set(p._id, p); } return p._id; };
	// transferIDs returns the numbers of the partners of the ports in the
	// given transfer list of a message to another engine (see ports.go),
	// which are moved there, and are closed here.
	function transferIDs(transfer) {
		const ids = [];
		for (const p of transfer || []) {
			if (!(p instanceof MessagePort) || !p._other) continue;
			const q = p._other;
			p._other = null;
			p._closed = true;
			p._movedVia = q;
			q._other = null;
			ids.push(portID(q));
		}
		return ids;
	}
	// receivedPorts returns the ports of the given pairs of ends of a
	// message from another engine, which are made here.
	const receivedPorts = pairs => (pairs || []).map(([partner, end]) => {
		const p = new MessagePort();
		p._remote = partner;
		D.portBind(end, portID(p));
		return p;
	});
	D.linkPort = function (id, end) { const q = localPorts.get(id); if (q) q._remote = end; };
	D.portMessage = function (id, data, pairs) {
		const p = localPorts.get(id);
		if (!p) return;
		let v = null;
		try { v = cloneDecode(JSON.parse(data)); } catch (e) { }
		p._deliver(v, receivedPorts(pairs));
	};
	class MessageChannel {
		constructor() {
			this.port1 = new MessagePort();
			this.port2 = new MessagePort();
			this.port1._other = this.port2;
			this.port2._other = this.port1;
		}
	}
	class MediaError {
		constructor(code, message) { this.code = code; this.message = message || ""; }
	}
	Object.assign(MediaError, { MEDIA_ERR_ABORTED: 1, MEDIA_ERR_NETWORK: 2, MEDIA_ERR_DECODE: 3, MEDIA_ERR_SRC_NOT_SUPPORTED: 4 });

	const waiting = new Map(); // name -> resolve functions of whenDefined
	const customElements = {
		define(name, cls) {
			name = String(name).toLowerCase();
			if (registry.has(name)) throw domException("NotSupportedError", "the name has already been used: " + name);
			const observed = Array.isArray(cls.observedAttributes) ? cls.observedAttributes.map(String) : [];
			registry.set(name, { cls, observed });
			definedNames.set(cls, name);
			for (const id of customIn(D.doc())) if (D.name(id) === name) upgrade(wrap(id));
			for (const r of waiting.get(name) || []) r(cls);
			waiting.delete(name);
		},
		get(name) { const d = registry.get(String(name).toLowerCase()); return d ? d.cls : undefined; },
		getName(cls) { return definedNames.get(cls) || null; },
		whenDefined(name) {
			name = String(name).toLowerCase();
			if (registry.has(name)) return Promise.resolve(registry.get(name).cls);
			return new Promise(r => { if (!waiting.has(name)) waiting.set(name, []); waiting.get(name).push(r); });
		},
		upgrade(root) { if (root && root[ID]) for (const id of customIn(root[ID])) upgrade(wrap(id)); },
	};

	const t0 = Date.now();
	const performance = {
		now() { return Date.now() - t0; },
		timeOrigin: t0,
		mark() {}, measure() {}, clearMarks() {}, clearMeasures() {}, clearResourceTimings() {}, setResourceTimingBufferSize() {},
		getEntries() { return []; }, getEntriesByType() { return []; }, getEntriesByName() { return []; },
		timing: { navigationStart: t0 },
		navigation: { type: 0 },
	};

	// subtle has the digests of data (as Google signs the requests of its
	// APIs with), but no keys, encryption or signatures
	const subtle = {
		digest(alg, data) {
			const name = typeof alg === "string" ? alg : alg && alg.name;
			const d = D.digest(String(name || ""), data);
			return d ? Promise.resolve(d) : Promise.reject(domException("NotSupportedError", "Unrecognized algorithm name"));
		},
	};
	for (const m of ["encrypt", "decrypt", "sign", "verify", "generateKey", "deriveKey", "deriveBits", "importKey", "exportKey", "wrapKey", "unwrapKey"]) {
		subtle[m] = () => Promise.reject(domException("NotSupportedError", "The operation is not supported"));
	}
	const crypto = {
		subtle,
		getRandomValues(a) { for (let i = 0; i < a.length; i++) a[i] = Math.floor(Math.random() * 256); return a; },
		randomUUID() {
			const h = "0123456789abcdef";
			let s = "";
			for (let i = 0; i < 36; i++) s += i === 8 || i === 13 || i === 18 || i === 23 ? "-" : i === 14 ? "4" : h[i === 19 ? 8 + Math.floor(Math.random() * 4) : Math.floor(Math.random() * 16)];
			return s;
		},
	};

	// Function compiles code that goja does not support (as async
	// generators, which pages test for) rewritten by esbuild.
	const OrigFunction = Function;
	const FunctionProxy = function Function(...args) { return makeFunction(args); };
	FunctionProxy.prototype = OrigFunction.prototype;
	function makeFunction(args) {
		try {
			return Reflect.construct(OrigFunction, args);
		} catch (e) {
			if (!(e instanceof SyntaxError)) throw e;
			const params = args.slice(0, -1).map(String).join(",");
			const body = args.length ? String(args[args.length - 1]) : "";
			const code = D.lower("(function anonymous(" + params + "\n) {\n" + body + "\n})");
			if (code === null) throw e;
			return (0, eval)(code);
		}
	}

	// ---- instanceof with proxies of functions ----

	// The instanceof of goja throws for a proxy of a function (as of URL
	// or Request, which scripts wrap to watch them), where browsers use
	// Function.prototype[Symbol.hasInstance] of its target (as the
	// getRequestURL of the bot checks of DataDome does for every fetch,
	// which failed). The proxies of functions get it from a get trap that
	// gives it for the target, which is all that instanceof gets from them
	// (Function.prototype[Symbol.hasInstance] cannot be replaced).
	{
		const NativeProxy = Proxy;
		const nativeRevocable = Proxy.revocable; // (defined on the wrapper below, which is the native one's)
		const fnHasInstance = Function.prototype[Symbol.hasInstance];
		// handlerFor returns the handler of a proxy of the given target
		// with the given handler.
		const handlerFor = (target, handler) => {
			if (typeof target !== "function" || handler === null || typeof handler !== "object") return handler;
			const h = Object.create(handler); // (its other traps are the handler's)
			h.get = function (t, k, r) {
				const v = typeof handler.get === "function" ? handler.get.call(handler, t, k, r) : Reflect.get(t, k, r);
				if (k === Symbol.hasInstance && v === fnHasInstance) return function (o) { return fnHasInstance.call(t, o); };
				return v;
			};
			return h;
		};
		const WrappedProxy = new NativeProxy(NativeProxy, {
			construct(T, args, nt) { return Reflect.construct(T, [args[0], handlerFor(args[0], args[1])], nt === WrappedProxy ? T : nt); },
		});
		Object.defineProperty(WrappedProxy, "revocable", {
			value: function revocable(target, handler) { return nativeRevocable(target, handlerFor(target, handler)); },
			writable: true, configurable: true,
		});
		Object.defineProperty(g, "Proxy", { value: WrappedProxy, writable: true, configurable: true, enumerable: false });
	}

	// ---- the windows of frames (see frames.go) ----

	// cloneJSON returns the given message as JSON, as it is copied to the
	// window or worker that it is posted to (see cloneDecode), as for the
	// structured clone of browsers: with its array buffers, typed arrays,
	// dates, regular expressions, maps, sets, undefined values, and
	// numbers that JSON has not.
	const cloneError = () => new DOMException("could not be cloned", "DataCloneError");
	function cloneEncode(v) {
		switch (typeof v) {
			case "undefined": return { $c: "u" };
			case "bigint": return { $c: "b", v: String(v) };
			case "number": return isFinite(v) && !Object.is(v, -0) ? v : { $c: "n", v: String(v) };
			case "function": case "symbol": throw cloneError();
			case "object": break;
			default: return v;
		}
		if (v === null) return v;
		if (v instanceof ArrayBuffer) return { $c: "ab", v: Array.from(new Uint8Array(v)) };
		if (ArrayBuffer.isView(v)) return { $c: "ta", t: v.constructor.name, v: Array.from(new Uint8Array(v.buffer, v.byteOffset, v.byteLength)) };
		if (v instanceof Date) return { $c: "d", v: v.getTime() };
		if (v instanceof RegExp) return { $c: "r", s: v.source, f: v.flags };
		if (v instanceof Map) return { $c: "m", v: Array.from(v, ([k, x]) => [cloneEncode(k), cloneEncode(x)]) };
		if (v instanceof Set) return { $c: "s", v: Array.from(v, cloneEncode) };
		if (v instanceof Error) return { $c: "e", n: v.name, m: v.message };
		if (Array.isArray(v)) return v.map(cloneEncode);
		if (v instanceof Node) throw cloneError();
		const o = {};
		for (const k of Object.keys(v)) o[k] = cloneEncode(v[k]);
		return "$c" in o ? { $c: "o", v: o } : o;
	}
	function cloneDecode(v) {
		if (v === null || typeof v !== "object") return v;
		if (Array.isArray(v)) return v.map(cloneDecode);
		const decodeObject = o => { const r = {}; for (const k of Object.keys(o)) r[k] = cloneDecode(o[k]); return r; };
		switch (v.$c) {
			case undefined: return decodeObject(v);
			case "u": return undefined;
			case "b": return BigInt(v.v);
			case "n": return Number(v.v);
			case "ab": return new Uint8Array(v.v).buffer;
			case "ta": {
				const buf = new Uint8Array(v.v).buffer;
				if (v.t === "DataView") return new DataView(buf);
				const C = g[v.t] || Uint8Array;
				return new C(buf, 0, buf.byteLength / (C.BYTES_PER_ELEMENT || 1));
			}
			case "d": return new Date(v.v);
			case "r": return new RegExp(v.s, v.f);
			case "m": return new Map(v.v.map(([k, x]) => [cloneDecode(k), cloneDecode(x)]));
			case "s": return new Set(v.v.map(cloneDecode));
			case "e": { const C = g[v.n] || Error; const e = new C(v.m); return e; }
			case "o": return decodeObject(v.v);
		}
		return decodeObject(v);
	}
	const cloneJSON = v => JSON.stringify(cloneEncode(v));
	// targetOrigin returns the target origin of the options of postMessage
	// (a string, or an object with targetOrigin), "/" by default.
	const targetOrigin = o => { const t = o !== null && typeof o === "object" ? o.targetOrigin : o; return t === undefined ? "/" : String(t); };
	// transferOf returns the transfer list of postMessage of a window: of
	// its options (its second argument, an object, of which an array is
	// one, as in browsers: reCAPTCHA posts its worker setup with
	// postMessage(data, [port]) in windows too, which is posted without the
	// port, and it took its own port as that of its frame), or its third
	// argument. Workers and ports take a transfer list as their second.
	const transferOf = (o, transfer) => o !== null && typeof o === "object" ? o.transfer : transfer;
	const transferList = t => Array.isArray(t) ? t : t !== null && typeof t === "object" ? t.transfer : undefined;
	const proxies = new Map(); // the window proxies, by their number
	// windowProxy returns the window of the given number (see
	// Engine.windowID): null for none, this window for 0, and otherwise a
	// proxy of another window, which can post messages to it.
	// Window is the class of windows: the global object and the proxies of
	// the windows of other frames are its instances (GitHub checks for
	// instanceof Window, which was not defined).
	class Window extends EventTarget {
		constructor() { super(); throw new TypeError("Illegal constructor"); }
		static [Symbol.hasInstance](o) { return o === g || windowProxies.has(o); }
	}
	const windowProxies = new WeakSet();
	function windowProxy(id) {
		if (id < 0) return null;
		if (id === 0) return g;
		let p = proxies.get(id);
		if (p) return p;
		const base = {
			postMessage(data, opts, transfer) { D.postTo(id, cloneJSON(data), targetOrigin(opts), transferIDs(transferOf(opts, transfer))); },
			get closed() { return D.windowClosed(id); },
			get parent() { return windowProxy(D.windowRel(id, "parent")); },
			get top() { return windowProxy(D.windowRel(id, "top")); },
			get window() { return p; }, get self() { return p; }, get frames() { return p; },
			get length() { return D.windowLength(id); },
			focus() {}, blur() {}, close() {},
		};
		// the windows of its frames by their names and indexes (as
		// reCAPTCHA gets the window of its checkbox by parent.frames[name])
		p = new Proxy(base, {
			get(t, k, r) {
				if (typeof k !== "string" || k in t) return Reflect.get(t, k, r);
				const w = windowProxy(D.windowFrame(id, k));
				if (!w && traceGlobal) D.log("debug", "missing: frame " + JSON.stringify(k) + " of another window (frames: " + D.windowFrameNames(id) + ")");
				return w || undefined;
			},
		});
		windowProxies.add(p);
		proxies.set(id, p);
		return p;
	}

	// ---- workers (see worker.go) ----

	const workers = new Map(); // the Worker objects, by their numbers
	let workerCount = 0;
	class Worker extends EventTarget {
		constructor(url, opts) {
			super();
			const id = ++workerCount;
			Object.defineProperty(this, "_id", { value: id });
			workers.set(id, this);
			D.newWorker(id, String(url));
		}
		postMessage(data, transfer) { D.postToWorker(this._id, cloneJSON(data), transferIDs(transferList(transfer))); }
		terminate() { D.terminateWorker(this._id); workers.delete(this._id); }
	}
	defineSimpleHandlers(Worker, ["message", "error", "messageerror"]);

	// ---- the global object (window) ----

	const props = {
		window: g, self: g, frames: g, opener: null, frameElement: null, closed: false,
		NodeList, HTMLCollection, ShadowRoot, CSSStyleSheet, Intl,
		CSSRule, CSSRuleList, CSSStyleRule, CSSGroupingRule, CSSConditionRule, CSSMediaRule, CSSSupportsRule, CSSContainerRule, CSSLayerBlockRule,
		CSSImportRule, CSSFontFaceRule, CSSPageRule, CSSKeyframesRule, CSSKeyframeRule, CSSNamespaceRule, CSSCharsetRule, MessageChannel, MessagePort, MediaError, Worker,
		MediaSource, SourceBuffer, SourceBufferList,
		TextTrack, TextTrackList: TrackList, AudioTrackList: TrackList, VideoTrackList: TrackList,
		document, location, history, navigation, navigator, Navigation, NavigationHistoryEntry,
		screenX: 0, screenY: 0, screenLeft: 0, screenTop: 0,
		screen: { width: 1920, height: 1080, availWidth: 1920, availHeight: 1080, availTop: 0, availLeft: 0, colorDepth: 24, pixelDepth: 24, orientation: { type: "landscape-primary", angle: 0, addEventListener() {} } },
		localStorage: storageProxy("local"), sessionStorage: storageProxy("session"),
		performance, crypto, customElements,
		devicePixelRatio: 1, status: "", isSecureContext: true, OffscreenCanvas, BroadcastChannel,
		Node, Element, HTMLElement, Text, Comment, CDATASection, ProcessingInstruction, CharacterData, Document, HTMLDocument, DocumentFragment, DocumentType, Range, Window,
		HTMLAnchorElement, HTMLAreaElement, HTMLImageElement, HTMLScriptElement, HTMLLinkElement, HTMLStyleElement,
		HTMLMetaElement, HTMLBaseElement, HTMLIFrameElement, HTMLInputElement, HTMLTextAreaElement, HTMLSelectElement,
		HTMLOptionElement, HTMLButtonElement, HTMLFormElement, HTMLTemplateElement, HTMLCanvasElement, HTMLMediaElement,
		HTMLVideoElement: HTMLMediaElement, HTMLAudioElement: HTMLMediaElement, HTMLDetailsElement, HTMLDialogElement,
		HTMLDivElement, HTMLSpanElement, HTMLParagraphElement, HTMLHeadingElement, HTMLUListElement, HTMLOListElement,
		HTMLLIElement, HTMLTableElement, HTMLTableRowElement, HTMLTableCellElement, HTMLHtmlElement, HTMLHeadElement,
		HTMLBodyElement, HTMLPreElement, HTMLBRElement, HTMLHRElement, HTMLFieldSetElement, HTMLUnknownElement,
		Attr, HTMLOptGroupElement, HTMLLabelElement, HTMLLegendElement, HTMLProgressElement, HTMLMeterElement, HTMLOutputElement,
		HTMLSourceElement, HTMLPictureElement, HTMLTrackElement, HTMLTimeElement, HTMLDataElement, HTMLDataListElement,
		HTMLMapElement, HTMLQuoteElement, HTMLModElement, HTMLTableSectionElement, HTMLTableCaptionElement, HTMLTableColElement,
		HTMLTitleElement, HTMLMenuElement, HTMLDListElement, HTMLEmbedElement, HTMLObjectElement, HTMLParamElement,
		HTMLSlotElement, HTMLFontElement, SVGElement, SVGSVGElement,
		EventTarget, Event, CustomEvent, UIEvent, MouseEvent, PointerEvent, KeyboardEvent, FocusEvent, InputEvent,
		WheelEvent, ErrorEvent, MessageEvent, ProgressEvent, PopStateEvent, HashChangeEvent, SubmitEvent, History,
		AnimationEvent, TransitionEvent, TouchEvent,
		DOMTokenList, CSSStyleDeclaration, DOMRect, DOMRectReadOnly: DOMRect, Location, MediaQueryList,
		URL, URLSearchParams, Headers, Request, Response, XMLHttpRequest, Blob, File, FileReader, FormData,
		AbortController, AbortSignal, TextEncoder, TextDecoder, DOMParser, XMLSerializer,
		MutationObserver, MutationRecord, IntersectionObserver, ResizeObserver, PerformanceObserver,
		fetch,
		Function: FunctionProxy,
		__dynamicImport: dynamicImport,
		__defineModule: defineModule,
		__requireModule: requireModule,
		NodeFilter: { SHOW_ALL: 0xFFFFFFFF, SHOW_ELEMENT: 1, SHOW_TEXT: 4, SHOW_COMMENT: 128, FILTER_ACCEPT: 1, FILTER_REJECT: 2, FILTER_SKIP: 3 },
		DOMException,
		CSS: { supports() { return false; }, escape: cssEscape },
		Image: function Image(w, h) { const i = document.createElement("img"); if (w !== undefined) i.width = w; if (h !== undefined) i.height = h; return i; },
		Option: function Option(text, value, def, sel) { const o = document.createElement("option"); if (text !== undefined) o.text = text; if (value !== undefined) o.value = value; if (sel) o.selected = true; return o; },
		Audio: function Audio() { return document.createElement("audio"); },
		setTimeout(fn, ms, ...args) { return setTimer(fn, ms, args, false); },
		setInterval(fn, ms, ...args) { return setTimer(fn, ms, args, true); },
		clearTimeout: clearTimer,
		clearInterval: clearTimer,
		requestAnimationFrame(fn) { return setTimer(() => fn(performance.now()), 16, [], false); },
		cancelAnimationFrame: clearTimer,
		requestIdleCallback(fn) { return setTimer(() => fn({ didTimeout: false, timeRemaining: () => 10 }), 1, [], false); },
		cancelIdleCallback: clearTimer,
		queueMicrotask(fn) { promiseThen.call(resolvedPromise, () => { try { fn(); } catch (e) { reportError(e); } }); },
		structuredClone(v) { return v === undefined ? undefined : JSON.parse(JSON.stringify(v)); },
		reportError,
		atob(s) { return D.atob(String(s)); },
		btoa(s) { return D.btoa(String(s)); },
		matchMedia(q) { return new MediaQueryList(q); },
		getComputedStyle,
		getSelection() { return { rangeCount: 0, isCollapsed: true, toString() { return ""; }, removeAllRanges() {}, addRange() {}, getRangeAt() { return document.createRange(); }, collapse() {} }; },
		alert(m) { D.log("info", "alert: " + m); },
		confirm(m) { D.log("info", "confirm: " + m); return false; },
		prompt(m) { D.log("info", "prompt: " + m); return null; },
		print() {},
		open(u) { if (u) navigate(String(u), false); return null; },
		close() {},
		focus() {},
		blur() {},
		stop() {},
		postMessage(data, opts, transfer) { D.postTo(0, cloneJSON(data), targetOrigin(opts), transferIDs(transferOf(opts, transfer))); },
		scrollTo() {}, scrollBy() {}, scroll() {},
		addEventListener: EventTarget.prototype.addEventListener,
		removeEventListener: EventTarget.prototype.removeEventListener,
		dispatchEvent: EventTarget.prototype.dispatchEvent,
	};
	for (const k of Object.keys(props)) {
		Object.defineProperty(g, k, { value: props[k], writable: true, configurable: true, enumerable: false });
	}
	for (const [k, f] of [["innerWidth", 0], ["outerWidth", 0], ["innerHeight", 1], ["outerHeight", 1]]) {
		Object.defineProperty(g, k, { get() { return Math.round(D.viewport()[f]); }, configurable: true });
	}
	for (const [k, f] of [["scrollX", "x"], ["pageXOffset", "x"], ["scrollY", "y"], ["pageYOffset", "y"]]) {
		Object.defineProperty(g, k, { get() { return scrollState[f]; }, configurable: true });
	}
	Object.defineProperty(g, "origin", { get() { return location.origin; }, configurable: true });
	// the window of the page that this one is in a frame of, and the top
	// one (this window if it is not in a frame)
	Object.defineProperty(g, "parent", { get() { return windowProxy(D.windowRel(0, "parent")) || g; }, configurable: true });
	Object.defineProperty(g, "top", { get() { return windowProxy(D.windowRel(0, "top")) || g; }, configurable: true });
	defineHandlers(g);

	const console = {};
	const fmt = args => args.map(a => {
		if (typeof a === "string") return a;
		if (a instanceof Error) return a.stack || String(a);
		if (a instanceof Node) return a.nodeType === 1 ? "<" + a.localName + ">" : a.nodeName;
		try { const s = JSON.stringify(a); return s === undefined ? String(a) : s; } catch (e) { return String(a); }
	}).join(" ");
	for (const [k, level] of [["log", "info"], ["info", "info"], ["debug", "debug"], ["trace", "debug"], ["warn", "warn"], ["error", "error"], ["dir", "info"], ["table", "info"]]) {
		console[k] = (...args) => D.log(level, fmt(args));
	}
	for (const k of ["group", "groupCollapsed", "groupEnd", "time", "timeEnd", "timeLog", "count", "countReset", "clear", "profile", "profileEnd", "timeStamp"]) console[k] = () => {};
	console.assert = (c, ...args) => { if (!c) D.log("error", "Assertion failed: " + fmt(args)); };
	Object.defineProperty(g, "console", { value: console, writable: true, configurable: true });

	// ---- the functions that Go calls ----

	// dispatch dispatches a trusted event from the app (as a click) to the
	// node of the given id, returning whether its default was prevented.
	D.dispatch = function (id, type, init) {
		const t = wrap(id);
		if (!t) return false;
		init = init || {};
		let ev;
		const opts = Object.assign({ bubbles: true, cancelable: true, view: g }, init);
		if (/^pointer/.test(type)) ev = new PointerEvent(type, opts);
		else if (/^(click|dblclick|mouse|contextmenu)/.test(type)) ev = new MouseEvent(type, opts);
		else if (/^key/.test(type)) ev = new KeyboardEvent(type, opts);
		else if (/^(focus|blur)/.test(type)) ev = new FocusEvent(type, Object.assign(opts, { bubbles: type === "focusin" || type === "focusout" }));
		else if (type === "input") ev = new InputEvent(type, opts);
		else if (/^transition/.test(type)) ev = new TransitionEvent(type, Object.assign(opts, { cancelable: false }));
		else if (/^animation/.test(type)) ev = new AnimationEvent(type, Object.assign(opts, { cancelable: false }));
		else ev = new Event(type, opts);
		ev.isTrusted = true;
		if (type === "focus" || type === "focusin") focused = t;
		if (init.value !== undefined && "value" in t) values.set(t, String(init.value));
		if (init.checked !== undefined) {
			// a radio button that is checked unchecks the others of its group
			if (init.checked && t.type === "radio") t.checked = true;
			else checks.set(t, !!init.checked);
		}
		t.dispatchEvent(ev);
		if (type === "click" && !ev.defaultPrevented) {
			summaryActivation(t);
			labelActivation(t);
		}
		return ev.defaultPrevented;
	};

	// summaryActivation does the default action of a click on the given
	// target in a summary (not in a link or control in it): it toggles the
	// details of the summary, as in browsers, once, unless a listener
	// prevented the default of the click (as the sidebar of MDN).
	function summaryActivation(t) {
		for (let n = t; n && n.nodeType === 1; n = n.parentNode) {
			const tag = n.localName;
			if (tag === "a" || tag === "button" || tag === "input" || tag === "select" || tag === "textarea" || tag === "label") return;
			if (tag === "summary") {
				const d = n.parentNode;
				if (d && d.localName === "details" && d.querySelector(":scope > summary") === n) {
					d.open = !d.open;
					Promise.resolve().then(() => d.dispatchEvent(new Event("toggle")));
				}
				return;
			}
			if (tag === "details") return;
		}
	}

	// labelActivation does the default action of a click on the given target
	// in a label (not in a link or control in it): it clicks the control of
	// the label, as in browsers (as the hidden checkboxes that the [-]
	// labels of the comments of lobste.rs check, which fold the threads).
	function labelActivation(t) {
		for (let n = t; n && n.nodeType === 1; n = n.parentNode) {
			const tag = n.localName;
			if (tag === "a" || tag === "button" || tag === "input" || tag === "select" || tag === "textarea" || tag === "summary") return;
			if (tag === "label") {
				const c = n.control;
				if (c && c !== t && !c.contains(t) && typeof c.click === "function") c.click();
				return;
			}
		}
	}

	// activate does the default action of the element of the given id
	// after a click on it that was not prevented (as submitting its form
	// for a submit button), or submits its form for a text field (as for
	// Enter, its implicit submission).
	D.activate = function (id) {
		const el = wrap(id);
		if (!el) return;
		const form = el.closest && el.closest("form");
		if (!form) return;
		const t = (el.getAttribute("type") || "").toLowerCase();
		if (el.tagName === "BUTTON" ? (t === "" || t === "submit") : (t === "submit" || t === "image")) form.requestSubmit(el);
		else if (el.tagName === "BUTTON" && t === "reset" || el.tagName === "INPUT" && t === "reset") form.reset();
		else if (el.tagName === "INPUT") form.requestSubmit();
	};

	// state returns the state of the form control of the given id that
	// is not in its attributes: its value and checked state, or null
	// for each that the user or scripts have not changed.
	// hover dispatches the events of the mouse moving from the element of
	// the first id to that of the second (0 for none), as in browsers:
	// pointerout and mouseout on the old one, pointerleave and mouseleave
	// on it and its ancestors that are not around the new one, pointerover
	// and mouseover on the new one, and pointerenter and mouseenter on it
	// and its ancestors that are not around the old one, from the outermost
	// (React makes its onPointerEnter of these, as for the repository grid
	// of github.com).
	// press dispatches the events of a mouse button pressed (down) or
	// released on the given element (see Engine.Press).
	let pressCanceled = false;
	D.press = function (id, down, init) {
		const t = wrap(id);
		if (!t) return;
		const opts = Object.assign({ bubbles: true, cancelable: true, view: g, pointerType: "mouse" }, init || {}, { buttons: down ? 1 : 0 });
		const pe = new PointerEvent(down ? "pointerdown" : "pointerup", opts);
		pe.isTrusted = true;
		t.dispatchEvent(pe);
		if (down) pressCanceled = pe.defaultPrevented;
		if (!pressCanceled) {
			const me = new MouseEvent(down ? "mousedown" : "mouseup", opts);
			me.isTrusted = true;
			t.dispatchEvent(me);
		}
	};

	D.hover = function (oldId, newId, x, y) {
		const o = oldId ? wrap(oldId) : null, n = newId ? wrap(newId) : null;
		const ev = (C, type, t, rel, bubbles) => {
			const e = new C(type, { bubbles, cancelable: bubbles, view: g, relatedTarget: rel, pointerType: "mouse", clientX: x, clientY: y });
			e.isTrusted = true;
			t.dispatchEvent(e);
		};
		const chain = (from, around) => {
			const c = [];
			for (let a = from; a && a.nodeType === 1 && !(around && a.contains(around)); a = a.parentNode) c.push(a);
			return c;
		};
		if (o) {
			ev(PointerEvent, "pointerout", o, n, true);
			ev(MouseEvent, "mouseout", o, n, true);
			for (const a of chain(o, n)) {
				ev(PointerEvent, "pointerleave", a, n, false);
				ev(MouseEvent, "mouseleave", a, n, false);
			}
		}
		if (n) {
			ev(PointerEvent, "pointerover", n, o, true);
			ev(MouseEvent, "mouseover", n, o, true);
			for (const a of chain(n, o).reverse()) {
				ev(PointerEvent, "pointerenter", a, o, false);
				ev(MouseEvent, "mouseenter", a, o, false);
			}
		}
	};

	// toggleDetails toggles the details element of the given id open or
	// closed, with its toggle event (see Engine.ToggleDetails).
	D.toggleDetails = function (id) {
		const d = wrap(id);
		if (!(d instanceof HTMLDetailsElement)) return;
		d.open = !d.open;
		Promise.resolve().then(() => d.dispatchEvent(new Event("toggle")));
	};

	D.state = function (id) {
		const el = wrappers.get(id); // no state without a wrapper (see hasListener)
		if (!el) return [null, null];
		return [values.has(el) ? values.get(el) : null, checks.has(el) ? checks.get(el) : null];
	};

	// fireEvent fires a simple event of the given type at the node of
	// the given id (as the load event of a script).
	D.fireEvent = function (id, type) {
		const t = wrap(id);
		if (t) t.dispatchEvent(new Event(type));
	};

	// hasListener returns whether the node of the given id or one that it
	// is in (or the document or window) has a listener of the given type.
	// It is called while the page is read (rendered): it runs no scripts of
	// the page (as wrap could, which constructs custom elements).
	D.hasListener = function (id, type) {
		const has = t => {
			const m = listeners.get(t);
			return !!(m && m[type] && m[type].length) || typeof t["_on" + type] === "function";
		};
		for (let n = id; n; n = D.parent(n)) {
			const w = wrappers.get(n);
			if (w && has(w)) return true;
			if (D.type(n) === 1 && D.attr(n, "on" + type) !== null) return true;
		}
		return has(g);
	};

	// resized fires the resize event of the window, when its viewport has
	// another size.
	D.resized = function () {
		g.dispatchEvent(new Event("resize"));
	};

	// message dispatches a message posted to this window (as JSON) from
	// the given origin and window (see frames.go).
	D.message = function (data, origin, source, pairs) {
		let v = null;
		try { v = cloneDecode(JSON.parse(data)); } catch (e) { }
		g.dispatchEvent(new MessageEvent("message", { data: v, origin, source: windowProxy(source), ports: receivedPorts(pairs) }));
	};

	// workerMessage and workerError dispatch a message (as JSON, see
	// cloneJSON) and an error from the worker of the given number to its
	// Worker object (see worker.go).
	D.workerMessage = function (id, data, pairs) {
		const w = workers.get(id);
		if (!w) return;
		let v = null;
		try { v = cloneDecode(JSON.parse(data)); } catch (e) { }
		w.dispatchEvent(new MessageEvent("message", { data: v, ports: receivedPorts(pairs) }));
	};
	D.workerError = function (id, msg) {
		const w = workers.get(id);
		if (!w) return;
		const e = new Event("error", { cancelable: true });
		e.message = msg;
		w.dispatchEvent(e);
	};

	// mediaEvents sets the state of the playing of the given video element
	// (from its player, see MediaPlayer in Go) and dispatches the given
	// events to it.
	D.mediaEvents = function (id, st, types) {
		const el = wrappers.get(id) || wrap(id);
		if (!(el instanceof HTMLMediaElement)) return;
		const m = el._media();
		Object.assign(m, {
			paused: st.paused, ended: st.ended, currentTime: st.currentTime, volume: st.volume, muted: st.muted,
			duration: st.duration > 0 ? st.duration : m.duration, width: st.width || m.width, height: st.height || m.height,
		});
		for (const t of types) {
			switch (t) {
				case "loadedmetadata": m.readyState = Math.max(m.readyState, 1); m.networkState = 1; break;
				case "loadeddata": case "canplay": case "canplaythrough": case "playing": m.readyState = 4; break;
				case "error":
					m.networkState = 3;
					m.error = new MediaError(MediaError.MEDIA_ERR_SRC_NOT_SUPPORTED, st.error);
					break;
			}
			el.dispatchEvent(new Event(t));
		}
	};

	// frameLoaded fires the load event of the given iframe element, after
	// the page in its frame has loaded.
	D.frameLoaded = function (id) {
		const el = wrap(id);
		if (el) el.dispatchEvent(new Event("load"));
	};

	// loaded sets the ready state of the document, and fires its events.
	D.loaded = function (state) {
		readyState = state;
		document.dispatchEvent(new Event("readystatechange"));
		if (state === "interactive") {
			document.dispatchEvent(new Event("DOMContentLoaded", { bubbles: true }));
		} else if (state === "complete") {
			g.dispatchEvent(new Event("load"));
			g.dispatchEvent(new Event("pageshow"));
		}
	};

	// the global scope of a worker (see worker.go): no window or document
	// (which scripts check for whether they run in a worker), and
	// importScripts, and postMessage and close for its owner
	if (D.isWorker()) {
		for (const k of ["window", "document", "frames", "parent", "top", "opener", "frameElement", "localStorage", "sessionStorage", "alert", "confirm", "prompt"]) delete g[k];
		const wprops = {
			postMessage(data, transfer) { D.postToOwner(cloneJSON(data), transferIDs(transferList(transfer))); },
			close() { D.closeWorker(); },
			importScripts(...urls) { for (const u of urls) D.importScripts(String(u)); },
		};
		for (const k of Object.keys(wprops)) {
			Object.defineProperty(g, k, { value: wprops[k], writable: true, configurable: true, enumerable: false });
		}
		// (self instanceof WorkerGlobalScope, as scripts check whether
		// they run in a worker)
		for (const name of ["WorkerGlobalScope", "DedicatedWorkerGlobalScope"]) {
			const scope = function () { throw new TypeError("Illegal constructor"); };
			Object.defineProperty(scope, "name", { value: name });
			Object.defineProperty(scope, Symbol.hasInstance, { value: o => o === g });
			Object.defineProperty(g, name, { value: scope, writable: true, configurable: true });
		}
	}

	// the properties that scripts get that the objects of the browser do
	// not have, in the console (once each), when the engine traces (see
	// Options.Trace): the root of the prototypes of the global object, of
	// the DOM (EventTarget), and of the other objects of the browser, is a
	// proxy of Object.prototype that reports them (as for the functions
	// of browsers that a captcha uses, whose errors it catches)
	let traceGlobal = null;
	if (D.tracing()) {
		const seen = new Set();
		let reporting = false;
		const ignored = new Set(["then", "toJSON", "constructor", "prototype", "length", "name", "inspect", "nodeType", "$$typeof", "asymmetricMatch"]);
		const objectName = (o) => {
			if (o === g) return "window";
			for (const k of ["navigator", "screen", "location", "history", "performance", "crypto", "document"]) {
				try { if (o === g[k]) return k; } catch (e) { }
			}
			if (o && o.nodeType === 1) return "<" + o.localName + ">";
			if (o && o.nodeType === 3) return "#text";
			const c = o && o.constructor;
			return (c && c.name) || "Object";
		};
		const report = (o, k, kind) => {
			if (typeof k !== "string" || k.startsWith("_") || ignored.has(k) || reporting) return;
			reporting = true;
			try {
				const key = kind + objectName(o) + "." + k;
				if (!seen.has(key)) {
					seen.add(key);
					D.log("debug", key);
				}
			} catch (e) { } finally { reporting = false; }
		};
		const proxy = new Proxy(Object.prototype, {
			get(t, k, r) {
				if (!(k in t)) report(r, k, "missing: ");
				return Reflect.get(t, k, r);
			},
		});
		traceGlobal = (k) => report(g, k, "missing: ");
		const rootOf = (o) => {
			for (let p = o; p; p = Object.getPrototypeOf(p)) {
				if (Object.getPrototypeOf(p) === Object.prototype) return p;
			}
			return null;
		};
		const roots = new Set();
		for (const o of [g.EventTarget && g.EventTarget.prototype, g.navigator, g.screen, g.location, g.history, g.performance, g.crypto]) {
			const r = o && rootOf(o);
			if (r && r !== g) roots.add(r);
		}
		for (const r of roots) {
			try { Object.setPrototypeOf(r, proxy); } catch (e) { }
		}
	}

	// the windows of the frames of this window by their names and indexes
	// (as window[name] and frames[name]), from the root of the prototypes
	// of the global object, a proxy of Object.prototype, which also reports
	// the global names that are not defined when the engine traces
	// the name of the window: that of its iframe, for a frame, until it is
	// set (as reCAPTCHA gets the name of the frame of its checkbox, by
	// parent.frames[name], from that of its challenge)
	let windowName = null;
	Object.defineProperty(g, "name", {
		get() { return windowName !== null ? windowName : D.isWorker() ? "" : D.frameName(); },
		set(v) { windowName = String(v); },
		configurable: true, enumerable: true,
	});
	try {
		Object.defineProperty(g, "length", { get() { return D.windowLength(0); }, configurable: true });
		if (Object.getPrototypeOf(g) === Object.prototype) {
			// the window is a Window, and an EventTarget, of whose prototype
			// it has its methods (as the shady DOM polyfill of youtube.com
			// patches them there, for the window too)
			Object.setPrototypeOf(g, new Proxy(Window.prototype, {
				get(t, k, r) {
					if (typeof k !== "string" || k in t) return Reflect.get(t, k, r);
					const f = D.windowFrame(0, k);
					if (f >= 0) return windowProxy(f);
					if (traceGlobal) traceGlobal(k);
					return undefined;
				},
				has(t, k) {
					const has = k in t;
					if (!has && traceGlobal) traceGlobal(k);
					return has;
				},
			}));
			for (const k of ["addEventListener", "removeEventListener", "dispatchEvent"]) delete g[k];
		}
	} catch (e) { }
})(globalThis, __dom);
