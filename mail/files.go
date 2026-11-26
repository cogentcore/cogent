// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"errors"
	"fmt"
	"hash/fnv"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/core/base/auth"
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

// migrateFiles moves the local files of the given account from the names
// of before they were named by its email address, with base32, if any.
func (a *App) migrateFiles(email string) {
	old := filenameBase32(email)
	if old == accountName(email) {
		return
	}
	dir := a.dataDir()
	for _, m := range [][2]string{
		{filepath.Join(dir, "mail", old), a.messagesDir(email)},
		{filepath.Join(dir, "store", old+".db"), a.storeFile(email)},
		{filepath.Join(dir, "search", old+".bleve"), a.searchDir(email)},
	} {
		if _, err := os.Stat(m[0]); err != nil {
			continue // nothing to move
		}
		os.Remove(m[1]) // if it is an empty directory
		if _, err := os.Stat(m[1]); err == nil {
			slog.Warn("mail: not moving files to where there are others", "from", m[0], "to", m[1])
			continue
		}
		if err := os.Rename(m[0], m[1]); err != nil {
			slog.Error("mail: moving files", "from", m[0], "to", m[1], "err", err)
			continue
		}
		slog.Info("mail: moved files", "from", m[0], "to", m[1])
	}
	oldAuth := filepath.Join(dir, "auth", old)
	if err := auth.MoveToken(filepath.Join(oldAuth, "google-token.json"), a.tokenFile(email)); err != nil {
		slog.Error("mail: moving the saved sign-in", "account", email, "err", err)
		return
	}
	if err := os.Remove(oldAuth); err != nil && !errors.Is(err, fs.ErrNotExist) {
		slog.Error("mail: removing the old directory of the sign-in", "dir", oldAuth, "err", err)
	}
}

// shardMessages moves the files of the given messages of the given account
// that are not in subdirectories (see [shardName]) into them, returning the
// messages that changed, whose files are missing if their names are "".
func (a *App) shardMessages(email string, msgs map[string]*CacheMessage) map[string]*CacheMessage {
	dir := a.messagesDir(email)
	changed := map[string]*CacheMessage{}
	made := map[string]bool{}
	for k, cm := range msgs {
		if cm.Filename == "" || strings.ContainsRune(cm.Filename, '/') || strings.ContainsRune(cm.Filename, filepath.Separator) {
			continue
		}
		name := shardName(cm.Filename)
		sub := filepath.Join(dir, filepath.Dir(name))
		if !made[sub] {
			if err := os.MkdirAll(sub, 0o700); err != nil {
				slog.Error("mail: making the directory of messages", "dir", sub, "err", err)
				return changed
			}
			made[sub] = true
		}
		if err := os.Rename(filepath.Join(dir, cm.Filename), filepath.Join(dir, name)); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				slog.Error("mail: moving a message", "file", cm.Filename, "err", err)
				continue
			}
			name = "" // downloaded again
		}
		cm.Filename = name
		changed[k] = cm
	}
	if len(changed) > 0 {
		slog.Info("mail: moved messages into subdirectories", "account", email, "n", len(changed))
	}
	return changed
}
