// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
)

// The time zones of Intl.DateTimeFormat (prelude.js), from the time zone
// database of Go: the time of a page was formatted in the local time zone
// whatever its timeZone option, so the times of nytimes.com, which are
// formatted in that of New York, differed from those of its server
// ("9:52 a.m. ET"), and React rendered the whole page again (error 418).

// zones are the time zones that have been loaded, by name (nil if there is
// no such zone).
var zones sync.Map

// zone returns the time zone of the given name, or nil if there is none.
func zone(name string) *time.Location {
	if z, ok := zones.Load(name); ok {
		return z.(*time.Location)
	}
	z, err := time.LoadLocation(name)
	if err != nil || name == "" || name == "Local" {
		z = nil
	}
	zones.Store(name, z)
	return z
}

// localZone returns the name of the local time zone, as "Europe/Amsterdam":
// that of $TZ, or the link of /etc/localtime into the time zone database,
// or "UTC".
var localZone = sync.OnceValue(func() string {
	if tz := os.Getenv("TZ"); tz != "" && !strings.HasPrefix(tz, ":") {
		return tz
	}
	if p, err := filepath.EvalSymlinks("/etc/localtime"); err == nil {
		if _, name, ok := strings.Cut(p, "zoneinfo/"); ok && zone(name) != nil {
			return name
		}
	}
	return "UTC"
})

// defineIntl defines the Go functions of Intl on __dom (e.jsDOM): they do
// not use the page, so they are not natives that lock it.
func (e *Engine) defineIntl() {
	vm := e.vm
	d := e.jsDOM
	// tzOffset(name, ms) is the offset in minutes from UTC of the time zone
	// at the time, or NaN if there is no such zone.
	d.Set("tzOffset", func(call goja.FunctionCall) goja.Value {
		z := zone(call.Argument(0).String())
		if z == nil {
			return vm.ToValue(math.NaN())
		}
		_, off := time.UnixMilli(call.Argument(1).ToInteger()).In(z).Zone()
		return vm.ToValue(off / 60)
	})
	// tzAbbrev(name, ms) is the abbreviation of the time zone at the time,
	// as "EST" ("" if there is no such zone).
	d.Set("tzAbbrev", func(call goja.FunctionCall) goja.Value {
		z := zone(call.Argument(0).String())
		if z == nil {
			return vm.ToValue("")
		}
		name, _ := time.UnixMilli(call.Argument(1).ToInteger()).In(z).Zone()
		return vm.ToValue(name)
	})
	// tzCanonical(name) is the name of the time zone as Intl has it (the
	// case of the given name may differ), or "" if there is none.
	d.Set("tzCanonical", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		switch strings.ToUpper(name) {
		case "UTC", "ETC/UTC", "GMT", "ETC/GMT", "Z":
			return vm.ToValue("UTC")
		}
		if z := zone(name); z != nil {
			return vm.ToValue(z.String())
		}
		return vm.ToValue("")
	})
	d.Set("localZone", func(call goja.FunctionCall) goja.Value { return vm.ToValue(localZone()) })
}
