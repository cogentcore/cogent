// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAccountName tests the names of the files of the accounts.
func TestAccountName(t *testing.T) {
	assert.Equal(t, "rcoreilly5@gmail.com", accountName("rcoreilly5@gmail.com"))
	assert.Equal(t, "a_b_c@x.test", accountName("a/b\\c@x.test"))
	assert.NotEqual(t, filepath.Dir(shardName("1")), filepath.Dir(shardName("2")))
}
