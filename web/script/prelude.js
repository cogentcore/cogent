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

	// ---- wrapping nodes ----

	const wrappers = new Map(); // id -> object
	const ID = Symbol("id");

	const tagClasses = {}; // tag name -> class of its elements

	function wrap(id) {
		if (!id) return null;
		let w = wrappers.get(id);
		if (w) return w;
		let cls;
		switch (D.type(id)) {
			case 1: cls = tagClasses[D.name(id)] || HTMLElement; break;
			case 3: cls = Text; break;
			case 8: cls = Comment; break;
			case 9: cls = Document; break;
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

	function domException(name, msg) {
		const e = new Error(msg);
		e.name = name;
		return e;
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
			this.target = null;
			this.currentTarget = null;
			this.eventPhase = 0;
			this.timeStamp = performance.now();
			this.isTrusted = false;
			this._stop = false;
			this._stopNow = false;
		}
		preventDefault() { if (this.cancelable) this.defaultPrevented = true; }
		stopPropagation() { this._stop = true; }
		stopImmediatePropagation() { this._stop = true; this._stopNow = true; }
		composedPath() { return this._path ? this._path.slice() : []; }
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
	class MessageEvent extends Event { constructor(type, init) { super(type, init); this.data = init && init.data; this.origin = (init && init.origin) || ""; } }
	class ProgressEvent extends Event { constructor(type, init) { super(type, init); init = init || {}; this.loaded = init.loaded || 0; this.total = init.total || 0; this.lengthComputable = !!init.lengthComputable; } }
	const popStates = new WeakMap();
	class PopStateEvent extends Event {
		constructor(type, init) { super(type, init); popStates.set(this, init && init.state !== undefined ? init.state : null); }
		get state() { return popStates.get(this); }
	}
	class HashChangeEvent extends Event { constructor(type, init) { super(type, init); this.oldURL = (init && init.oldURL) || ""; this.newURL = (init && init.newURL) || ""; } }
	class SubmitEvent extends Event { constructor(type, init) { super(type, init); this.submitter = (init && init.submitter) || null; } }
	class AnimationEvent extends Event {}
	class TransitionEvent extends Event {}
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
			ev.target = this;
			const path = [];
			for (let t = this; t; t = parentTarget(t)) path.push(t);
			ev._path = path;
			const invoke = (t, phase) => {
				ev.currentTarget = t;
				ev.eventPhase = phase;
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
			ev.currentTarget = null;
			ev.eventPhase = 0;
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

	function parentTarget(t) {
		if (t === g) return null;
		if (t instanceof Node) {
			const p = t.parentNode;
			if (p) return p;
			if (t instanceof Document) return g;
			return null;
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
		get baseURI() { return D.baseURL(); }
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
	const styleHandler = {
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

	class Element extends Node {
		get tagName() { return D.name(this[ID]).toUpperCase(); }
		get localName() { return D.name(this[ID]); }
		get namespaceURI() { return "http://www.w3.org/1999/xhtml"; }
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
			for (let i = 0; i < kv.length; i += 2) a.push({ name: kv[i], localName: kv[i], value: kv[i + 1], nodeValue: kv[i + 1], specified: true });
			a.getNamedItem = n => a.find(x => x.name === String(n).toLowerCase()) || null;
			a.item = i => a[i] || null;
			return a;
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
		get offsetParent() { return D.rect(this[ID]) ? document.body : null; }
		get clientWidth() { return this.offsetWidth; }
		get clientHeight() { return this.offsetHeight; }
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
		focus() { focused = this; }
		blur() { if (focused === this) focused = null; }
		click() { this.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, button: 0 })); }
		attachShadow(init) {
			if (shadows.has(this)) throw domException("NotSupportedError", "the element already has a shadow root");
			const id = D.createFragment();
			const sr = Object.create(ShadowRoot.prototype);
			Object.defineProperty(sr, ID, { value: id });
			wrappers.set(id, sr);
			shadowHosts.set(id, this[ID]);
			shadows.set(this, { root: sr, mode: init && init.mode === "closed" ? "closed" : "open" });
			sr.adoptedStyleSheets = [];
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
	});
	const HTMLScriptElement = elementClass(["script"], HTMLElement, cls => {
		reflectURL(cls, ["src"]);
		reflect(cls, ["type", "charset", "crossOrigin", "integrity", "referrerPolicy", "nonce"], ["async", "defer", "noModule"]);
		Object.defineProperty(cls.prototype, "text", { get() { return this.textContent; }, set(v) { this.textContent = v; } });
	});
	const HTMLLinkElement = elementClass(["link"], HTMLElement, cls => { reflectURL(cls, ["href"]); reflect(cls, ["rel", "type", "media", "as", "crossOrigin", "integrity"]); Object.defineProperty(cls.prototype, "relList", { get() { return new DOMTokenList(this, "rel"); } }); });
	// CSSStyleSheet has the rules of a style sheet as text. That of a style
	// element changes its text (as CSS-in-JS inserts rules), so that the
	// page is styled by them; constructed ones (adoptedStyleSheets, as of
	// Lit) are not applied.
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
	const sheetOwners = new WeakMap(); // style sheet -> style element
	class CSSStyleSheet {
		constructor(opts) { this._rules = []; this.disabled = false; this.media = { mediaText: (opts && opts.media) || "" }; }
		get cssRules() { const o = sheetOwners.get(this); if (o) this._rules = cssBlocks(o.textContent); return this._rules.map(t => ({ cssText: t, type: 1 })); }
		get rules() { return this.cssRules; }
		get ownerNode() { return sheetOwners.get(this) || null; }
		_set(rules) { this._rules = rules; const o = sheetOwners.get(this); if (o) o.textContent = rules.join("\n"); }
		insertRule(rule, index) {
			const rules = this.cssRules.map(r => r.cssText);
			index = index === undefined ? 0 : index >>> 0;
			if (index > rules.length) throw domException("IndexSizeError", "the index is beyond the rules");
			rules.splice(index, 0, String(rule));
			this._set(rules);
			return index;
		}
		deleteRule(index) { const rules = this.cssRules.map(r => r.cssText); rules.splice(index >>> 0, 1); this._set(rules); }
		addRule(sel, style, index) { return this.insertRule(sel + "{" + style + "}", index === undefined ? this.cssRules.length : index); }
		removeRule(index) { this.deleteRule(index || 0); }
		replaceSync(text) { this._set(cssBlocks(String(text))); }
		replace(text) { this.replaceSync(text); return Promise.resolve(this); }
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
		reflectURL(cls, ["src"]); reflect(cls, ["name", "width", "height", "allow", "loading"]);
		Object.defineProperty(cls.prototype, "contentWindow", { get() { return null; } });
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
	elementClass(["label"], HTMLElement, cls => {
		Object.defineProperty(cls.prototype, "htmlFor", { get() { return this.getAttribute("for") || ""; }, set(v) { this.setAttribute("for", v); } });
		Object.defineProperty(cls.prototype, "control", { get() { const f = this.htmlFor; return f ? document.getElementById(f) : this.querySelector("input, select, textarea, button"); } });
	});
	const HTMLTemplateElement = elementClass(["template"], HTMLElement, cls => {
		Object.defineProperty(cls.prototype, "content", {
			get() { const f = document.createDocumentFragment(); f.innerHTML = this.innerHTML; return f; },
		});
	});
	const HTMLCanvasElement = elementClass(["canvas"], HTMLElement, cls => {
		cls.prototype.getContext = function () { return null; };
		cls.prototype.toDataURL = function () { return "data:,"; };
		reflect(cls, ["width", "height"]);
	});
	const HTMLMediaElement = elementClass(["video", "audio"], HTMLElement, cls => {
		reflectURL(cls, ["src"]);
		reflect(cls, [], ["autoplay", "controls", "loop", "muted"]);
		cls.prototype.play = function () { return Promise.reject(domException("NotSupportedError", "media is not supported")); };
		cls.prototype.pause = function () {};
		cls.prototype.load = function () {};
		cls.prototype.canPlayType = function () { return ""; };
		Object.defineProperty(cls.prototype, "paused", { get() { return true; } });
	});
	const HTMLDetailsElement = elementClass(["details"], HTMLElement, cls => reflect(cls, [], ["open"]));
	const HTMLDialogElement = elementClass(["dialog"], HTMLElement, cls => {
		reflect(cls, [], ["open"]);
		cls.prototype.show = function () { this.open = true; };
		cls.prototype.showModal = function () { this.open = true; };
		cls.prototype.close = function (v) { this.open = false; this.returnValue = v || ""; this.dispatchEvent(new Event("close")); };
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

	// submitForm submits the given form, by navigating (as for a GET).
	function submitForm(form, submitter) {
		const method = (form.getAttribute("method") || "get").toLowerCase();
		const action = form.action || location.href;
		const fd = new FormData(form);
		if (submitter && submitter.name) fd.append(submitter.name, submitter.value);
		const q = new URLSearchParams();
		for (const [k, v] of fd) q.append(k, typeof v === "string" ? v : v.name || "");
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
		set adoptedStyleSheets(v) { this._adopted = Array.from(v); }
		get fonts() { return { ready: Promise.resolve(), check() { return true; }, load() { return Promise.resolve([]); }, addEventListener() {}, add() {}, forEach() {} }; }
		get implementation() { return { createHTMLDocument(title) { return wrap(D.parseDocument("<!DOCTYPE html><title>" + (title || "") + "</title>")); }, hasFeature() { return true; } }; }
		createElement(tag) { return wrap(D.create(String(tag))); }
		createElementNS(ns, tag) { const n = String(tag); return wrap(D.create(n.indexOf(":") >= 0 ? n.split(":")[1] : n)); }
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
		createRange() { return { setStart() {}, setEnd() {}, selectNodeContents() {}, collapse() {}, getBoundingClientRect() { return new DOMRect(); }, getClientRects() { return []; }, createContextualFragment: html => wrap(D.parse(String(html), 0)) }; }
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
		elementFromPoint() { return null; }
		elementsFromPoint() { return []; }
		exitFullscreen() { return Promise.resolve(); }
		get fullscreenElement() { return null; }
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
	class ShadowRoot extends DocumentFragment {
		get host() { return wrap(shadowHosts.get(this[ID])); }
		get mode() { const s = shadows.get(this.host); return s ? s.mode : "open"; }
		get activeElement() { return null; }
	}

	function treeWalker(root, what, filter) {
		what = what === undefined ? 0xFFFFFFFF : what;
		const accept = n => {
			if (!(what & (1 << (n.nodeType - 1)))) return false;
			if (!filter) return true;
			const f = typeof filter === "function" ? filter : filter.acceptNode;
			return f.call(filter, n) === 1;
		};
		const order = [];
		const visit = n => { for (let c = n.firstChild; c; c = c.nextSibling) { order.push(c); visit(c); } };
		visit(root);
		let i = -1;
		return {
			root, currentNode: root,
			nextNode() { while (++i < order.length) if (accept(order[i])) return (this.currentNode = order[i]); return null; },
			previousNode() { while (--i >= 0) if (accept(order[i])) return (this.currentNode = order[i]); return null; },
			firstChild() { const c = this.currentNode.firstChild; return c ? (this.currentNode = c) : null; },
			parentNode() { const p = this.currentNode.parentNode; return p && p !== root.parentNode ? (this.currentNode = p) : null; },
			nextSibling() { const s = this.currentNode.nextSibling; return s ? (this.currentNode = s) : null; },
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
		userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) CogentCore/1.0 Safari/537.36",
		appName: "Netscape", appVersion: "5.0", appCodeName: "Mozilla", product: "Gecko", vendor: "Cogent Core",
		platform: "MacIntel", language: "en-US", languages: ["en-US", "en"], onLine: true, cookieEnabled: true,
		hardwareConcurrency: 4, maxTouchPoints: 0, doNotTrack: null, webdriver: false, pdfViewerEnabled: false,
		plugins: [], mimeTypes: [],
		sendBeacon() { return true; },
		javaEnabled() { return false; },
		clipboard: { writeText() { return Promise.resolve(); }, readText() { return Promise.resolve(""); } },
		permissions: { query() { return Promise.resolve({ state: "denied", addEventListener() {} }); } },
		storage: { estimate() { return Promise.resolve({ quota: 0, usage: 0 }); }, persist() { return Promise.resolve(false); } },
		userActivation: { hasBeenActive: false, isActive: false },
	};

	class Storage {
		constructor(kind) { Object.defineProperty(this, "_kind", { value: kind }); }
		getItem(k) { return D.storage("get", this._kind, String(k)); }
		setItem(k, v) { D.storage("set", this._kind, String(k), String(v)); }
		removeItem(k) { D.storage("remove", this._kind, String(k)); }
		clear() { D.storage("clear", this._kind); }
		key(i) { const ks = D.storage("keys", this._kind); return i < ks.length ? ks[i] : null; }
		get length() { return D.storage("keys", this._kind).length; }
	}
	function storageProxy(kind) {
		return new Proxy(new Storage(kind), {
			get(t, p) { if (typeof p !== "string" || p in t) { const v = Reflect.get(t, p, t); return typeof v === "function" ? v.bind(t) : v; } const v = t.getItem(p); return v === null ? undefined : v; },
			set(t, p, v) { if (typeof p !== "string" || p in t) return Reflect.set(t, p, v, t); t.setItem(p, v); return true; },
			deleteProperty(t, p) { t.removeItem(p); return true; },
			has(t, p) { return p in t || t.getItem(p) !== null; },
			ownKeys(t) { return D.storage("keys", t._kind); },
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
	// the inline style, with the display of an element that has no box.
	function getComputedStyle(el) {
		const inline = el && el.style ? el.style : null;
		const get = p => {
			p = camelToKebab(String(p));
			const v = inline ? inline.getPropertyValue(p) : "";
			if (v) return v.replace(/\s*!important$/, "");
			const box = el ? D.rect(el[ID]) : null;
			switch (p) {
				case "display": return box ? (el.matches("span, a, b, i, em, strong, code, img, label, small, sup, sub, abbr") ? "inline" : "block") : "none";
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
		static createObjectURL(b) { return b instanceof Blob ? D.createObjectURL(b._data, b.type || "") : "blob:" + D.pageURL() + "#" + (++callbackID); }
		static revokeObjectURL(u) { D.revokeObjectURL(String(u)); }
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
	class Request extends Body {
		constructor(input, init) {
			super();
			init = init || {};
			if (input instanceof Request) { this.url = input.url; this.method = input.method; this.headers = new Headers(input.headers); this._body = input._body; }
			else { this.url = new URL(String(input), D.baseURL()).href; this.method = "GET"; this.headers = new Headers(); }
			if (init.method) this.method = String(init.method).toUpperCase();
			if (init.headers) this.headers = new Headers(init.headers);
			if (init.body !== undefined) this._body = bodyBytes(init.body);
			this.signal = init.signal || null;
			this.credentials = init.credentials || "same-origin";
			this.mode = init.mode || "cors";
			this.bodyUsed = false;
		}
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

	function goFetch(url, method, headers, body, cb) {
		D.fetch(newCallback(cb), url, method, headers, body);
	}

	function fetch(input, init) {
		let req;
		try { req = new Request(input, init); } catch (e) { return Promise.reject(e); }
		return new Promise((resolve, reject) => {
			if (req.signal && req.signal.aborted) { reject(domException("AbortError", "the request was aborted")); return; }
			let done = false;
			if (req.signal) req.signal.addEventListener("abort", () => { if (!done) { done = true; reject(domException("AbortError", "the request was aborted")); } });
			goFetch(req.url, req.method, req.headers._flat(), req._body, (status, statusText, headers, body, final, err) => {
				if (done) return;
				done = true;
				if (err) { reject(new TypeError("Failed to fetch: " + err)); return; }
				const h = new Headers();
				for (let i = 0; i + 1 < headers.length; i += 2) h.append(headers[i], headers[i + 1]);
				const r = new Response(body, { status, statusText, headers: h });
				r.url = final;
				r.redirected = final !== req.url;
				resolve(r);
			});
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
			goFetch(this._url, this._method, this._headers._flat(), bodyBytes(body), (status, statusText, headers, data, final, err) => {
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
			});
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
	class MutationObserver {
		constructor(cb) { this._cb = cb; }
		observe() {}
		disconnect() {}
		takeRecords() { return []; }
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
	class ResizeObserver {
		constructor(cb) { this._cb = cb; this._targets = []; }
		observe(t) {
			this._targets.push(t);
			setTimer(() => {
				if (this._targets.indexOf(t) < 0) return;
				const r = t.getBoundingClientRect();
				const size = [{ inlineSize: r.width, blockSize: r.height }];
				this._cb([{ target: t, contentRect: r, borderBoxSize: size, contentBoxSize: size, devicePixelContentBoxSize: size }], this);
			}, 0, [], false);
		}
		unobserve(t) { this._targets = this._targets.filter(x => x !== t); }
		disconnect() { this._targets = []; }
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

	// ---- MessageChannel ----

	// MessagePort sends messages to the other port of its channel, in
	// tasks (as React's scheduler yields with them).
	class MessagePort extends EventTarget {
		constructor() { super(); this._other = null; this._closed = false; this.onmessage = null; }
		postMessage(data) {
			const to = this._other;
			if (!to || this._closed) return;
			setTimer(() => {
				if (to._closed) return;
				const ev = new MessageEvent("message", { data });
				if (typeof to.onmessage === "function") {
					try { to.onmessage(ev); } catch (e) { reportError(e); }
				}
				to.dispatchEvent(ev);
			}, 0, [], false);
		}
		start() {}
		close() { this._closed = true; }
	}
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
		mark() {}, measure() {}, clearMarks() {}, clearMeasures() {},
		getEntries() { return []; }, getEntriesByType() { return []; }, getEntriesByName() { return []; },
		timing: { navigationStart: t0 },
		navigation: { type: 0 },
	};

	const crypto = {
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

	// ---- the global object (window) ----

	const props = {
		window: g, self: g, top: g, parent: g, frames: g, opener: null, frameElement: null, closed: false,
		NodeList, HTMLCollection, ShadowRoot, CSSStyleSheet, Intl, MessageChannel, MessagePort, MediaError,
		document, location, history, navigation, navigator, Navigation, NavigationHistoryEntry,
		screen: { width: 1920, height: 1080, availWidth: 1920, availHeight: 1080, colorDepth: 24, pixelDepth: 24, orientation: { type: "landscape-primary", angle: 0, addEventListener() {} } },
		localStorage: storageProxy("local"), sessionStorage: storageProxy("session"),
		performance, crypto, customElements,
		devicePixelRatio: 1, name: "", status: "", length: 0, isSecureContext: true,
		Node, Element, HTMLElement, Text, Comment, CharacterData, Document, HTMLDocument, DocumentFragment, DocumentType,
		HTMLAnchorElement, HTMLAreaElement, HTMLImageElement, HTMLScriptElement, HTMLLinkElement, HTMLStyleElement,
		HTMLMetaElement, HTMLBaseElement, HTMLIFrameElement, HTMLInputElement, HTMLTextAreaElement, HTMLSelectElement,
		HTMLOptionElement, HTMLButtonElement, HTMLFormElement, HTMLTemplateElement, HTMLCanvasElement, HTMLMediaElement,
		HTMLVideoElement: HTMLMediaElement, HTMLAudioElement: HTMLMediaElement, HTMLDetailsElement, HTMLDialogElement,
		HTMLDivElement, HTMLSpanElement, HTMLParagraphElement, HTMLHeadingElement, HTMLUListElement, HTMLOListElement,
		HTMLLIElement, HTMLTableElement, HTMLTableRowElement, HTMLTableCellElement, HTMLHtmlElement, HTMLHeadElement,
		HTMLBodyElement, HTMLPreElement, HTMLBRElement, HTMLHRElement, HTMLFieldSetElement, HTMLUnknownElement, SVGElement,
		EventTarget, Event, CustomEvent, UIEvent, MouseEvent, PointerEvent, KeyboardEvent, FocusEvent, InputEvent,
		WheelEvent, ErrorEvent, MessageEvent, ProgressEvent, PopStateEvent, HashChangeEvent, SubmitEvent, History,
		AnimationEvent, TransitionEvent, TouchEvent,
		DOMTokenList, CSSStyleDeclaration, DOMRect, DOMRectReadOnly: DOMRect, Location, MediaQueryList,
		URL, URLSearchParams, Headers, Request, Response, XMLHttpRequest, Blob, File, FileReader, FormData,
		AbortController, AbortSignal, TextEncoder, TextDecoder, DOMParser, XMLSerializer,
		MutationObserver, IntersectionObserver, ResizeObserver, PerformanceObserver,
		fetch,
		Function: FunctionProxy,
		__dynamicImport: dynamicImport,
		__defineModule: defineModule,
		__requireModule: requireModule,
		NodeFilter: { SHOW_ALL: 0xFFFFFFFF, SHOW_ELEMENT: 1, SHOW_TEXT: 4, SHOW_COMMENT: 128, FILTER_ACCEPT: 1, FILTER_REJECT: 2, FILTER_SKIP: 3 },
		DOMException: class DOMException extends Error { constructor(msg, name) { super(msg); this.name = name || "Error"; } },
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
		postMessage(data) { setTimer(() => g.dispatchEvent(new MessageEvent("message", { data, origin: location.origin })), 0, [], false); },
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
		if (/^(click|dblclick|mouse|contextmenu)/.test(type)) ev = new MouseEvent(type, opts);
		else if (/^key/.test(type)) ev = new KeyboardEvent(type, opts);
		else if (/^(focus|blur)/.test(type)) ev = new FocusEvent(type, Object.assign(opts, { bubbles: type === "focusin" || type === "focusout" }));
		else if (type === "input") ev = new InputEvent(type, opts);
		else ev = new Event(type, opts);
		ev.isTrusted = true;
		if (type === "focus" || type === "focusin") focused = t;
		if (init.value !== undefined && "value" in t) values.set(t, String(init.value));
		if (init.checked !== undefined) checks.set(t, !!init.checked);
		t.dispatchEvent(ev);
		return ev.defaultPrevented;
	};

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
})(globalThis, __dom);
