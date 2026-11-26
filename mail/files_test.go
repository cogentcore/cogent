// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/core"
	"github.com/cogentcore/go-imap/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// TestMigrateFiles tests moving the files of an account from the names
// of before, in base32, to its email address, with its messages moved
// into subdirectories, and its saved sign-in moved to the keychain.
func TestMigrateFiles(t *testing.T) {
	b, a := newTestApp(t, nil)
	dir := a.dataDirectory
	old := filenameBase32(testEmail)

	// the old files
	mdir := filepath.Join(dir, "mail", old)
	require.NoError(t, os.MkdirAll(mdir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(mdir, "12345"), []byte("Subject: Hi\r\n\r\nHello\r\n"), 0o600))
	st, err := openStore(filepath.Join(dir, "store", old+".db"))
	require.NoError(t, err)
	require.NoError(t, st.put(map[string]*CacheMessage{
		"hi":   {Envelope: imap.Envelope{Subject: "Hi", Date: time.Now()}, Filename: "12345", Labels: []Label{{Name: "INBOX", UID: 1}}},
		"gone": {Envelope: imap.Envelope{Subject: "Gone"}, Filename: "67890", Labels: []Label{{Name: "INBOX", UID: 2}}},
	}))
	require.NoError(t, st.Close())
	si, err := openSearch(filepath.Join(dir, "search", old+".bleve"))
	require.NoError(t, err)
	require.NoError(t, si.Close())
	oldToken := filepath.Join(dir, "auth", old, "google-token.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(oldToken), 0o700))
	require.NoError(t, jsonx.Save(&oauth2.Token{AccessToken: "a", RefreshToken: "r"}, oldToken))

	core.RenderImage(b, func() {
		require.NoError(t, a.openLocal(testEmail))
	}, core.Locked(b, func() {
		assert.Equal(t, filepath.Join(dir, "mail", testEmail), a.messagesDir(testEmail), "named by the address")
		assert.NoDirExists(t, mdir)
		assert.NoFileExists(t, filepath.Join(dir, "store", old+".db"))
		assert.NoDirExists(t, filepath.Join(dir, "search", old+".bleve"))
		assert.FileExists(t, a.storeFile(testEmail))
		assert.DirExists(t, a.searchDir(testEmail))

		hi := a.cache[testEmail]["hi"]
		require.NotNil(t, hi)
		assert.Equal(t, shardName("12345"), hi.Filename, "in a subdirectory")
		assert.True(t, strings.HasSuffix(hi.Filename, ".eml"))
		assert.FileExists(t, filepath.Join(a.messagesDir(testEmail), hi.Filename))
		assert.Empty(t, a.cache[testEmail]["gone"].Filename, "its file is missing: downloaded again")

		assert.NoDirExists(t, filepath.Dir(oldToken), "the sign-in moved")
		assert.NoFileExists(t, a.tokenFile(testEmail), "to the keychain")
	}))
	// saved, as the next start shows
	a.closeStores()
	st, err = openStore(a.storeFile(testEmail))
	require.NoError(t, err)
	defer st.Close()
	msgs, err := st.load()
	require.NoError(t, err)
	assert.Equal(t, shardName("12345"), msgs["hi"].Filename)
}

// TestAccountName tests the names of the files of the accounts.
func TestAccountName(t *testing.T) {
	assert.Equal(t, "rcoreilly5@gmail.com", accountName("rcoreilly5@gmail.com"))
	assert.Equal(t, "a_b_c@x.test", accountName("a/b\\c@x.test"))
	assert.NotEqual(t, filepath.Dir(shardName("1")), filepath.Dir(shardName("2")))
}
