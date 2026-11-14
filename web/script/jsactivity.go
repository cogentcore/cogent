// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"bytes"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/google/pprof/profile"
)

// The time of the scripts, by script, every few seconds, in the verbose
// consoles of the pages (see [Options.Trace]), from the sampling profiler
// of goja, which samples all of the engines (with those of frames): to
// tell what the scripts do in the long times in which their pages do not
// change visibly (as nytimes.com did, for 20s, as it loaded, in which its
// requests and those of its ads were answered, which saved pages can not
// show).

// activityWindow is the time of each summary of the time of the scripts.
const activityWindow = 5 * time.Second

// activity are the engines of pages that have the summaries of the time of
// the scripts (see [Engine.watchActivity]).
var activity struct {
	sync.Mutex
	engines map[*Engine]bool
	running bool
}

// watchActivity adds the summaries of the time of the scripts to the
// verbose console of the engine, if it is that of a page (not of a frame),
// while it traces, until it is closed (see [Engine.unwatchActivity]).
func (e *Engine) watchActivity() {
	if e.opts.Trace == nil || e.opts.Parent != nil && e.opts.Parent() != nil {
		return
	}
	activity.Lock()
	defer activity.Unlock()
	if activity.engines == nil {
		activity.engines = map[*Engine]bool{}
	}
	activity.engines[e] = true
	if !activity.running {
		activity.running = true
		go watchActivity()
	}
}

// unwatchActivity stops the summaries for the engine.
func (e *Engine) unwatchActivity() {
	activity.Lock()
	delete(activity.engines, e)
	activity.Unlock()
}

// tracingEngines returns the engines that have the summaries now, or nil
// and false if there are none, when the summaries stop.
func tracingEngines() ([]*Engine, bool) {
	activity.Lock()
	defer activity.Unlock()
	if len(activity.engines) == 0 {
		activity.running = false
		return nil, false
	}
	var es []*Engine
	for e := range activity.engines {
		if e.tracing() {
			es = append(es, e)
		}
	}
	return es, true
}

// watchActivity profiles the scripts in windows of [activityWindow] while
// an engine traces, and logs their summaries to those that do, until there
// are no engines.
func watchActivity() {
	for {
		es, ok := tracingEngines()
		if !ok {
			return
		}
		if len(es) == 0 {
			time.Sleep(activityWindow) // (none trace: no profiling)
			continue
		}
		var buf bytes.Buffer
		if goja.StartProfile(&buf) != nil {
			// (as a test profiles the scripts itself)
			time.Sleep(activityWindow)
			continue
		}
		time.Sleep(activityWindow)
		goja.StopProfile()
		summary := activitySummary(buf.Bytes())
		if summary == "" {
			continue
		}
		es, _ = tracingEngines()
		for _, e := range es {
			e.tracef("%s", summary)
		}
	}
}

// activitySummary returns the summary of the given profile of the scripts:
// how long they ran (and waited in the functions that they called), and
// the scripts (files) of the functions that they were in, with the
// longest, or "" if they ran for less than 100ms.
func activitySummary(prof []byte) string {
	p, err := profile.Parse(bytes.NewReader(prof))
	if err != nil || len(p.SampleType) == 0 {
		return ""
	}
	// each sample is of a period (in nanoseconds), counted in its first value
	period := p.Period
	if period <= 0 {
		period = int64(10 * time.Millisecond)
	}
	byFile := map[string]int64{}
	var total int64
	for _, s := range p.Sample {
		if len(s.Location) == 0 || len(s.Value) == 0 {
			continue
		}
		v := s.Value[0] * period
		total += v
		file := "?"
		if ls := s.Location[0].Line; len(ls) > 0 && ls[0].Function != nil {
			file = shortScript(ls[0].Function.Filename)
		}
		byFile[file] += v
	}
	unit := time.Duration(1)
	if time.Duration(total) < 100*time.Millisecond {
		return ""
	}
	files := make([]string, 0, len(byFile))
	for f := range byFile {
		files = append(files, f)
	}
	sort.Slice(files, func(i, j int) bool { return byFile[files[i]] > byFile[files[j]] })
	var sb strings.Builder
	fmt.Fprintf(&sb, "scripts ran %v of the last %v:", (time.Duration(total) * unit).Round(10*time.Millisecond), activityWindow)
	for i, f := range files {
		if i == 6 {
			break
		}
		fmt.Fprintf(&sb, " %s %v,", f, (time.Duration(byFile[f]) * unit).Round(10*time.Millisecond))
	}
	return strings.TrimSuffix(sb.String(), ",")
}

// shortScript returns the short name of the script of the given URL: its
// host and the last part of its path.
func shortScript(u string) string {
	pu, err := url.Parse(u)
	if err != nil || pu.Host == "" {
		if i := strings.LastIndexByte(u, '/'); i >= 0 {
			return u[i+1:]
		}
		return u
	}
	path := pu.Path
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		path = path[i+1:]
	}
	return pu.Host + "/" + path
}
