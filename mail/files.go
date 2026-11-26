// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"fmt"
	"hash/fnv"
	"path/filepath"
	"strings"
)

// The local files of each account are in the directory of the data of
// the app (see [App.dataDir]), named by its email address (see
// [accountName]): its messages in mail/<account>, each in its own file in
// the standard format (as .eml files), in 256 subdirectories (see
// [shardName]), and its drafts in mail/<account>/local; the store of its
// messages in store/<account>.db; and its search index in
// search/<account>.bleve. The tokens of signing in are in the system
// keychain (see [auth.SaveToken]), named by auth/<account>/google-token.json.

// accountName returns the name of the files of the given account:
// its email address, with any characters that are not allowed in the
// names of files on some systems replaced.
func accountName(email string) string {
	return strings.Map(func(r rune) rune {
		if r < ' ' || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, email)
}

// storeFile and searchDir return the file of the store and the directory
// of the search index of the given account.
func (a *App) storeFile(email string) string {
	return filepath.Join(a.dataDir(), "store", accountName(email)+".db")
}

func (a *App) searchDir(email string) string {
	return filepath.Join(a.dataDir(), "search", accountName(email)+".bleve")
}

// shardName returns the name of the file of a message with the given name,
// relative to the directory of the messages of its account: in one of 256
// subdirectories, by a hash of the name, so that there are not too many
// files in one directory, with the .eml extension of its format.
func shardName(name string) string {
	h := fnv.New32a()
	h.Write([]byte(name))
	return filepath.Join(fmt.Sprintf("%02x", h.Sum32()&0xff), name+".eml")
}
