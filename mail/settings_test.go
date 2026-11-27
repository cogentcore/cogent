// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"testing"

	"cogentcore.org/core/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApplyDefaultSignatures tests that each account has an entry in the
// default signatures, but not a blank one, as one just added in the
// settings before its address is typed, and not an unset one of an
// account that is gone, while a set one is kept.
func TestApplyDefaultSignatures(t *testing.T) {
	sd := &SettingsData{Accounts: gmailAccounts("a@x.test", "")}
	sd.Apply()
	assert.Equal(t, map[string]SignatureName{"a@x.test": ""}, sd.DefaultSignatures, "no blank one")

	sd.Accounts[1].Email = "b@x.test" // typed
	sd.DefaultSignatures["old@x.test"] = ""
	sd.DefaultSignatures["kept@x.test"] = "Work"
	sd.Apply()
	assert.Equal(t, map[string]SignatureName{"a@x.test": "", "b@x.test": "", "kept@x.test": "Work"}, sd.DefaultSignatures)
}

// TestSignInOffered tests that the dialog for signing in to an account is
// not opened again while it is open, and that the accounts added in the
// settings after the start are recorded once, as they are offered to be
// signed in to.
func TestSignInOffered(t *testing.T) {
	b, a := newTestingApp(t)
	core.RenderImage(b, func() {}, core.Locked(b, func() {
		a.signInDialogFor([]string{"new@x.test"}, "Sign in")
		d := core.TopScene(b)
		require.NotEqual(t, b.Scene, d, "the dialog")
		assert.True(t, a.signInShown["new@x.test"])
		a.signInDialogFor([]string{"new@x.test"}, "Sign in")
		assert.Equal(t, d, core.TopScene(b), "not again")
		d.Close()
	}), core.Locked(b, func() {
		assert.False(t, a.signInShown["new@x.test"], "closed")

		a.started = true
		a.accountsAdded([]string{"a@x.test"}) // at the start
		a.accountsAdded([]string{"a@x.test", " b@x.test ", ""})
		assert.Equal(t, map[string]bool{"a@x.test": true, "b@x.test": true}, a.knownAccounts)
	}))
}
