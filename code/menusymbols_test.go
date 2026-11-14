// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

import (
	"sync/atomic"
	"testing"
	"time"

	"cogentcore.org/cogent/code/lang"
	"github.com/stretchr/testify/assert"
)

// TestSymbolCache checks that the menu search does not wait for slow
// symbols for longer than the given time, and that it then uses the
// cached symbols immediately, while they are fetched again.
func TestSymbolCache(t *testing.T) {
	var sc symbolCache
	var calls atomic.Int32
	release := make(chan struct{})
	fetch := func() []lang.Symbol {
		n := calls.Add(1)
		if n == 1 {
			<-release
		}
		return []lang.Symbol{{Name: "S" + string(rune('0'+n))}}
	}

	st := time.Now()
	assert.Nil(t, sc.get("pkg", 50*time.Millisecond, fetch), "slow: not there yet")
	assert.Less(t, time.Since(st), time.Second)
	assert.Nil(t, sc.get("pkg", 10*time.Millisecond, fetch), "one fetch at a time")
	assert.Equal(t, int32(1), calls.Load())

	close(release)
	assert.Eventually(t, func() bool { return sc.get("pkg", 0, fetch) != nil }, time.Second, time.Millisecond)
	assert.Eventually(t, func() bool {
		syms := sc.get("pkg", 0, fetch)
		return len(syms) == 1 && syms[0].Name != "S1"
	}, time.Second, time.Millisecond, "fetched again in the background")

	fast := func() []lang.Symbol { return []lang.Symbol{{Name: "F"}} }
	assert.Equal(t, "F", sc.get("other", time.Second, fast)[0].Name, "fast: waited for")
}
