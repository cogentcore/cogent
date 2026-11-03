// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTypesRoundTrip tests that the list types are the same after saving
// and loading them, with all of the settings of their fields, also in
// JavaScript (see TestGoScript), where they are synced on the web.
func TestTypesRoundTrip(t *testing.T) {
	types := standardTypes()
	b := marshal(types)
	t.Log(string(b[:min(len(b), 600)]))
	var back []*ListType
	require.NoError(t, json.Unmarshal(b, &back))
	require.Len(t, back, len(types))
	for i, lt := range types {
		require.Len(t, back[i].Fields, len(lt.Fields), lt.Name)
		for j, f := range lt.Fields {
			assert.Equal(t, *f, *back[i].Fields[j], lt.Name)
		}
	}
	assert.Equal(t, string(b), string(marshal(back)), "the same file")
}
