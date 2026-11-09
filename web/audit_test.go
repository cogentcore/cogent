// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"golang.org/x/net/websocket"
)

// The audit of live pages finds the controls and images that do not work
// as in Chrome, all at once rather than as people notice them: the same
// script lists the links, buttons, form controls and images of a page in
// Chrome (run headless, over the DevTools protocol) and in the page of
// this package (with its scripts), with their boxes and the element that
// a click at the center of each goes to (document.elementFromPoint).
// The report lists, for each page:
//   - not shown: an element that Chrome shows, which is not shown here
//   - covered: an element that a click at its center goes to in Chrome,
//     which it does not go to here (as when an overlay takes the clicks)
//   - size: an element of a much different size
//   - image: an image that Chrome shows, which is not loaded here
//   - extra: an element shown here, which Chrome does not show
//   - typing: a text field that does not take typed text, or a form
//     that is not submitted by Enter in its first text field
//
// It is run with go test -run TestAudit -audit [-audit-run regexp], and
// writes the report to testdata/audit-report.md (not committed).

var (
	auditFlag = flag.Bool("audit", false, "audit the live pages of testdata/audit-urls.txt against Chrome")
	auditRun  = flag.String("audit-run", "", "regexp selecting the pages to audit")
)

const (
	auditWidth  = 1280 // the size of the viewport, in CSS px
	auditHeight = 1600
	auditSettle = 4 * time.Second // the time that the scripts of a page get after it loads
	auditMax    = 600             // the most elements listed per page
)

// auditScript lists the elements of the page as JSON (see auditElement).
// The images are marked with their keys (data-audit), so that their
// widgets can be found, and the elements are kept in __audit.
const auditScript = `(() => {
const sel = 'a[href],button,input:not([type=hidden]),textarea,select,summary,[role=button],[role=link],[role=tab],[role=checkbox],img';
const els = Array.from(document.querySelectorAll(sel)).slice(0, ` + "AUDITMAX" + `);
const key = e => {
	const parts = [];
	for (let n = e; n && n.nodeType === 1 && n !== document.documentElement; n = n.parentElement) {
		const t = n.tagName.toLowerCase();
		if (n.id) { parts.unshift(t + '#' + n.id); break; }
		let i = 1;
		for (let s = n.previousElementSibling; s; s = s.previousElementSibling) if (s.tagName === n.tagName) i++;
		parts.unshift(t + ':' + i);
	}
	return parts.join('>');
};
const vw = innerWidth, vh = innerHeight;
const out = [];
window.__audit = els;
els.forEach((e, i) => {
	const k = key(e);
	const r = e.getBoundingClientRect();
	const cs = getComputedStyle(e);
	const vis = r.width >= 2 && r.height >= 2 && cs.visibility !== 'hidden' && cs.display !== 'none' && !(parseFloat(cs.opacity) === 0);
	let inl = false; // here, an inline element in a text, which has no box of its own
	if (!(r.width > 0) && cs.display !== 'none') {
		for (let p = e.parentElement; p; p = p.parentElement) {
			const pr = p.getBoundingClientRect();
			if (pr.width > 0 && pr.height > 0) { inl = /^(a|span|label|abbr|code|em|strong|b|i|small|sup|sub|kbd|time|mark|u|s|q|cite)$/i.test(e.tagName); break; }
		}
	}
	const cx = r.x + r.width / 2, cy = r.y + r.height / 2;
	let hit = '', hk = '';
	if (vis && cx >= 0 && cy >= 0 && cx < vw && cy < vh) {
		const h = document.elementFromPoint(cx, cy);
		if (!h) hit = 'none';
		else if (h === e) hit = 'self';
		else if (e.contains(h)) hit = 'in';
		else if (h.contains(e)) hit = 'anc';
		else { hit = 'other'; hk = key(h); }
	}
	const tag = e.tagName.toLowerCase() + (e.type && e.tagName !== 'BUTTON' ? '[' + e.type + ']' : '');
	const rec = {i, k, tag, x: Math.round(r.x), y: Math.round(r.y), w: Math.round(r.width), h: Math.round(r.height), vis, inl, hit, hk,
		text: (e.getAttribute('aria-label') || e.textContent || e.getAttribute('alt') || e.getAttribute('placeholder') || '').trim().replace(/\s+/g, ' ').slice(0, 40)};
	if (e.tagName === 'IMG') {
		e.setAttribute('data-audit', k);
		rec.img = !!(e.complete && e.naturalWidth > 0);
	}
	out.push(rec);
});
return JSON.stringify({vw, vh, els: out});
})()`

func auditJS() string { return strings.Replace(auditScript, "AUDITMAX", fmt.Sprint(auditMax), 1) }

// auditElement is an element listed by auditScript.
type auditElement struct {
	I          int
	K          string // the key: the path of tags (and indexes) from the nearest id
	Tag        string
	X, Y, W, H float32
	Vis        bool   // shown
	Inl        bool   // an inline element in a text, with no box of its own
	Hit        string // what a click at its center goes to: self, in (an element in it), anc (one that it is in), other, none, or "" if not tested
	Hk         string // the key of the other element
	Text       string
	Img        *bool // for an img: loaded (in Chrome)
}

type auditList struct {
	VW, VH float32
	Els    []*auditElement
}

// auditFinding is a difference of the page here from Chrome.
type auditFinding struct {
	Kind   string
	El     *auditElement
	Detail string
}

func TestAudit(t *testing.T) {
	if !*auditFlag {
		t.Skip("run with -audit (live pages and Chrome)")
	}
	f, err := os.Open("testdata/audit-urls.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var re *regexp.Regexp
	if *auditRun != "" {
		re = regexp.MustCompile(*auditRun)
	}
	var report strings.Builder
	report.WriteString("# Page audit\n\nControls and images that differ from Chrome (see audit_test.go).\n")
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, u, _ := strings.Cut(line, " ")
		u = strings.TrimSpace(u)
		if re != nil && !re.MatchString(name) {
			continue
		}
		t.Run(name, func(t *testing.T) {
			findings, notes, ok := auditPage(t, u)
			fmt.Fprintf(&report, "\n## %s\n\n%s\n\n", name, u)
			for _, n := range notes {
				fmt.Fprintf(&report, "- %s\n", n)
			}
			if !ok {
				return
			}
			if len(findings) == 0 {
				report.WriteString("\nNo differences.\n")
				return
			}
			counts := map[string]int{}
			for _, fd := range findings {
				counts[fd.Kind]++
			}
			var cs []string
			for _, k := range []string{"not shown", "covered", "size", "image", "extra", "typing"} {
				if counts[k] > 0 {
					cs = append(cs, fmt.Sprintf("%s: %d", k, counts[k]))
				}
			}
			fmt.Fprintf(&report, "\n%s\n\n| kind | element | text | detail |\n|---|---|---|---|\n", strings.Join(cs, ", "))
			for _, fd := range findings {
				fmt.Fprintf(&report, "| %s | `%s` %s | %s | %s |\n", fd.Kind, fd.El.Tag, fd.El.K, strings.ReplaceAll(fd.El.Text, "|", "/"), fd.Detail)
			}
			t.Logf("%s: %s", name, strings.Join(cs, ", "))
		})
	}
	out := filepath.Join("testdata", "audit-report.md")
	if err := os.WriteFile(out, []byte(report.String()), 0666); err != nil {
		t.Fatal(err)
	}
	t.Log("report:", out)
}

// auditPage audits the page at the given URL, returning its findings and
// notes about it (as the elements that could not be compared).
func auditPage(t *testing.T, u string) ([]auditFinding, []string, bool) {
	var notes []string
	chrome, err := chromeList(t, u)
	if err != nil {
		t.Errorf("chrome: %v", err)
		return nil, []string{"Chrome failed: " + err.Error()}, false
	}
	ours, typing, err := ourList(t, u)
	if err != nil {
		t.Errorf("ours: %v", err)
		return nil, []string{"the page failed: " + err.Error()}, false
	}
	findings := compareAudit(chrome, ours)
	findings = append(findings, typing...)
	missing, inline := 0, 0
	ok := map[string]*auditElement{}
	for _, e := range ours.Els {
		ok[e.K] = e
		if e.Inl {
			inline++
		}
	}
	for _, e := range chrome.Els {
		if e.Vis && ok[e.K] == nil {
			missing++
		}
	}
	if len(chrome.Els) == 0 && len(ours.Els) > 0 {
		notes = append(notes, "Chrome listed no elements: the site may block headless Chrome (as with a bot check), so only typing is audited")
	}
	notes = append(notes, fmt.Sprintf("elements: %d in Chrome, %d here; %d shown in Chrome are not in the document here (as made differently by scripts); %d here are inline in texts (no boxes of their own, not compared)", len(chrome.Els), len(ours.Els), missing, inline))
	return findings, notes, true
}

// compareAudit returns the differences of the elements here from Chrome.
func compareAudit(chrome, ours *auditList) []auditFinding {
	byKey := map[string]*auditElement{}
	for _, e := range ours.Els {
		byKey[e.K] = e
	}
	var fs []auditFinding
	add := func(kind string, e *auditElement, f string, args ...any) {
		fs = append(fs, auditFinding{Kind: kind, El: e, Detail: fmt.Sprintf(f, args...)})
	}
	for _, c := range chrome.Els {
		o := byKey[c.K]
		if o == nil {
			continue // not in the document here (see the notes)
		}
		if o.Inl {
			continue
		}
		inView := c.Y < chrome.VH && c.Y+c.H > 0 && c.X < chrome.VW && c.X+c.W > 0 // (not as an off-canvas menu)
		switch {
		case c.Vis && !o.Vis && inView:
			add("not shown", c, "%gx%g at %g,%g in Chrome; %gx%g here", c.W, c.H, c.X, c.Y, o.W, o.H)
			continue
		case !c.Vis && o.Vis && c.W == 0 && o.Y < ours.VH && o.X < ours.VW && o.X+o.W > 0:
			add("extra", o, "%gx%g at %g,%g here; not shown in Chrome", o.W, o.H, o.X, o.Y)
			continue
		case !c.Vis || !o.Vis:
			continue
		}
		if dw, dh := math.Abs(float64(c.W-o.W)), math.Abs(float64(c.H-o.H)); inView && (dw > max(24, 0.4*float64(c.W)) || dh > max(16, 0.5*float64(c.H))) {
			add("size", c, "%gx%g in Chrome; %gx%g here", c.W, c.H, o.W, o.H)
		}
		if (c.Hit == "self" || c.Hit == "in") && o.Hit != "" && o.Hit != "self" && o.Hit != "in" {
			d := o.Hit
			if o.Hit == "other" {
				d = "by `" + o.Hk + "`"
			} else if o.Hit == "anc" {
				d = "goes to an element that it is in"
			}
			add("covered", c, "a click at its center %s", d)
		}
		if c.Img != nil && *c.Img && inView && (o.Img == nil || !*o.Img) {
			add("image", c, "loaded in Chrome, %gx%g; not here", c.W, c.H)
		}
	}
	return fs
}

// ourList opens the page here and lists its elements, and returns the
// findings of typing in its text fields.
func ourList(t *testing.T, u string) (*auditList, []auditFinding, error) {
	b := core.NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(auditWidth), units.Dp(auditHeight))
		s.Max = s.Min
	})
	pg := NewPage(b)
	pg.Scripts = true
	pg.OpenURL(u, nil)
	var list *auditList
	var typing []auditFinding
	var lerr error
	b.RenderImage(func() {
		pg.wait()
		settle(pg)
		time.Sleep(auditSettle)
		settle(pg)
	}, func() {
		if pg.scripts == nil {
			lerr = fmt.Errorf("no scripts")
			return
		}
		// a page that a script replaces (as duckduckgo.com goes to the page
		// of its results) is audited when the new one has loaded
		v, err := pg.scripts.engine.RunString(auditJS())
		for try := 0; err != nil && strings.Contains(err.Error(), "closed") && try < 3; try++ {
			time.Sleep(auditSettle)
			settle(pg)
			v, err = pg.scripts.engine.RunString(auditJS())
		}
		if err != nil {
			lerr = err
			return
		}
		list = &auditList{}
		if lerr = json.Unmarshal([]byte(fmt.Sprint(v)), list); lerr != nil {
			return
		}
		// images are loaded if their widgets have them, and are shown
		b.AsyncLock()
		loaded := map[string]bool{}
		pg.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			n := htmlcore.NodeOf(cw)
			if n == nil {
				return true
			}
			k := htmlcore.GetAttr(n, "data-audit")
			if k == "" {
				return true
			}
			switch w := cw.(type) {
			case *core.Image:
				loaded[k] = w.Image != nil && !cwb.Geom.TotalBBox.Empty()
			case *core.SVG:
				loaded[k] = !cwb.Geom.TotalBBox.Empty()
			}
			return true
		})
		b.AsyncUnlock()
		for _, e := range list.Els {
			if e.Tag == "img" {
				ok := loaded[e.K]
				e.Img = &ok
			}
		}
		typing = auditTyping(t, b, pg, list)
	})
	return list, typing, lerr
}

// auditTyping clicks in the text fields of the page that are shown in its
// viewport and types in them, returning those that do not take the text,
// and presses Enter in the first one in a form, which should submit it.
func auditTyping(t *testing.T, b *core.Body, pg *Page, list *auditList) []auditFinding {
	var fs []auditFinding
	ev := b.Scene.RenderWindow().SystemWindow.Events()
	run := func(js string) string {
		v, err := pg.scripts.engine.RunString(js)
		if err != nil {
			return "error: " + err.Error()
		}
		return fmt.Sprint(v)
	}
	isText := func(tag string) bool {
		return tag == "textarea" || slices.Contains([]string{"input[text]", "input[search]", "input[email]", "input[url]", "input[tel]", "input[password]", "input[number]"}, tag)
	}
	n := 0
	for _, e := range list.Els {
		if !isText(e.Tag) || !e.Vis || e.Y < 0 || e.Y+e.H/2 >= list.VH || n >= 4 {
			continue
		}
		n++
		var scx, scy float32 // (scripts run outside of the lock, which they take)
		fmt.Sscan(run("scrollX"), &scx)
		fmt.Sscan(run("scrollY"), &scy)
		b.AsyncLock()
		dp := pg.Styles.UnitContext.Dots(units.UnitDp)
		pos := pg.Geom.Pos.Content.Add(math32.Vec2(e.X+e.W/2+scx, e.Y+e.H/2+scy).MulScalar(dp))
		b.AsyncUnlock()
		pt := image.Pt(int(pos.X), int(pos.Y))
		ev.MouseMove(pt)
		ev.MouseButton(events.MouseDown, events.Left, pt, 0)
		ev.MouseButton(events.MouseUp, events.Left, pt, 0)
		time.Sleep(300 * time.Millisecond)
		for _, r := range "zq" {
			ev.KeyChord(r, 0, 0)
			time.Sleep(50 * time.Millisecond)
		}
		time.Sleep(500 * time.Millisecond)
		settle(pg)
		val := run(fmt.Sprintf(`String(__audit[%d].value)`, e.I))
		if !strings.Contains(val, "zq") {
			fs = append(fs, auditFinding{Kind: "typing", El: e, Detail: fmt.Sprintf("typed zq; its value is %q", val)})
			continue
		}
		if n == 1 && run(fmt.Sprintf(`!!__audit[%d].form`, e.I)) == "true" {
			run(`window.__auditSubmitted = false; document.addEventListener("submit", e => { window.__auditSubmitted = true; e.preventDefault(); }, true)`)
			before := pg.History[pg.Current]
			ev.KeyChord('\r', key.CodeReturnEnter, 0)
			time.Sleep(time.Second)
			settle(pg)
			submitted := run("window.__auditSubmitted") == "true" || pg.History[pg.Current] != before
			if !submitted {
				fs = append(fs, auditFinding{Kind: "typing", El: e, Detail: "Enter does not submit its form"})
			}
			return fs // the page may have changed
		}
	}
	return fs
}

// chromeList opens the page in Chrome (headless, with a new profile) and
// lists its elements, over the DevTools protocol.
func chromeList(t *testing.T, u string) (*auditList, error) {
	dir := t.TempDir()
	cmd := exec.Command(chromePath(), "--headless=new", "--disable-gpu", "--no-first-run", "--no-default-browser-check",
		"--remote-debugging-port=0", "--remote-allow-origins=*", "--user-data-dir="+dir, "--force-device-scale-factor=1",
		fmt.Sprintf("--window-size=%d,%d", auditWidth, auditHeight),
		"--user-agent="+auditChromeUA, "about:blank")
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()
	var port string
	for range 100 {
		if b, err := os.ReadFile(filepath.Join(dir, "DevToolsActivePort")); err == nil {
			if p, _, ok := strings.Cut(string(b), "\n"); ok {
				port = p
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	if port == "" {
		return nil, fmt.Errorf("no DevTools port")
	}
	resp, err := http.Get("http://127.0.0.1:" + port + "/json/list")
	if err != nil {
		return nil, err
	}
	var targets []struct {
		Type                 string
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	err = json.NewDecoder(resp.Body).Decode(&targets)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	wsURL := ""
	for _, tg := range targets {
		if tg.Type == "page" {
			wsURL = tg.WebSocketDebuggerURL
			break
		}
	}
	if wsURL == "" {
		return nil, fmt.Errorf("no page target")
	}
	ws, err := websocket.Dial(wsURL, "", "http://127.0.0.1/")
	if err != nil {
		return nil, err
	}
	defer ws.Close()
	ws.SetDeadline(time.Now().Add(90 * time.Second))
	id := 0
	call := func(method string, params any) (json.RawMessage, error) {
		id++
		if err := websocket.JSON.Send(ws, map[string]any{"id": id, "method": method, "params": params}); err != nil {
			return nil, err
		}
		for {
			var msg struct {
				ID     int
				Result json.RawMessage
				Error  *struct{ Message string }
			}
			if err := websocket.JSON.Receive(ws, &msg); err != nil {
				return nil, err
			}
			if msg.ID != id {
				continue // an event
			}
			if msg.Error != nil {
				return nil, fmt.Errorf("%s: %s", method, msg.Error.Message)
			}
			return msg.Result, nil
		}
	}
	eval := func(js string) (string, error) {
		res, err := call("Runtime.evaluate", map[string]any{"expression": js, "returnByValue": true, "awaitPromise": true})
		if err != nil {
			return "", err
		}
		var r struct {
			Result struct {
				Value any
			}
			ExceptionDetails *struct{ Text string }
		}
		if err := json.Unmarshal(res, &r); err != nil {
			return "", err
		}
		if r.ExceptionDetails != nil {
			return "", fmt.Errorf("exception: %s", r.ExceptionDetails.Text)
		}
		return fmt.Sprint(r.Result.Value), nil
	}
	if _, err := call("Page.navigate", map[string]any{"url": u}); err != nil {
		return nil, err
	}
	for range 300 {
		if s, err := eval("document.readyState"); err == nil && s == "complete" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	time.Sleep(auditSettle)
	out, err := eval(auditJS())
	if err != nil {
		return nil, err
	}
	list := &auditList{}
	if err := json.Unmarshal([]byte(out), list); err != nil {
		return nil, err
	}
	return list, nil
}

// auditChromeUA is the user agent of Chrome in the audit: that of the
// desktop browser (that of headless Chrome is blocked by many sites).
const auditChromeUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"

// chromePath returns the path of Chrome, which is $CHROME if it is set.
func chromePath() string {
	if p := os.Getenv("CHROME"); p != "" {
		return p
	}
	return "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
}
