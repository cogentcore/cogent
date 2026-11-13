// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// sharedFile is a JSON file that is shared by processes, such as different
// apps: it is read again when another process has changed it, and it is
// written with a lock file held, after reading the changes of the other
// processes (see [sharedFile.update]), so that they are not lost. This is
// the logic of base/dirsort and of the window geometry of core, which
// could be a base package.
type sharedFile struct {

	// path is the path of the file, which is not read or written if empty.
	path string

	// modTime and size are those of the file when it was last read or
	// written, to read it again when another process changes it.
	modTime time.Time
	size    int64
}

// changed returns whether the file exists and has changed since it was
// last read or written.
func (sf *sharedFile) changed() bool {
	if sf.path == "" {
		return false
	}
	info, err := os.Stat(sf.path)
	if err != nil {
		return false
	}
	return !info.ModTime().Equal(sf.modTime) || info.Size() != sf.size
}

// stat records the state of the file after it is read or written.
func (sf *sharedFile) stat() {
	if info, err := os.Stat(sf.path); err == nil {
		sf.modTime, sf.size = info.ModTime(), info.Size()
	}
}

// read reads the file into the given value.
func (sf *sharedFile) read(v any) error {
	b, err := os.ReadFile(sf.path)
	if err != nil {
		return err
	}
	sf.stat()
	return json.Unmarshal(b, v)
}

// update changes the data of the file with the given change function, and
// writes the given value of the data, with the file locked, after reading
// the changes of other processes with the given sync function (which reads
// the file if it has [sharedFile.changed]). The file is written whole and
// then renamed, so that other processes never read a partial file.
func (sf *sharedFile) update(sync, change func(), v any) error {
	if sf.path == "" {
		change()
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(sf.path), 0o755); err != nil {
		change()
		return err
	}
	unlock, err := lockFile(sf.path + ".lck")
	if err != nil {
		change() // (in memory, at least)
		return err
	}
	defer unlock()
	sync()
	change()
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return err
	}
	tmp := sf.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, sf.path); err != nil {
		return err
	}
	sf.stat()
	return nil
}

// lockStale is how long a lock file can be held before it is taken as
// left by a process that ended while holding it.
const lockStale = 2 * time.Second

// lockFile locks the given lock file, by creating it (which only one
// process can do), waiting for another process to remove it, or taking it
// over when it is stale. It returns the function to unlock it.
func lockFile(fn string) (unlock func(), err error) {
	for range 100 {
		f, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			f.WriteString(strconv.FormatInt(time.Now().UnixNano(), 10))
			f.Close()
			return func() { os.Remove(fn) }, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, err
		}
		if info, err := os.Stat(fn); err == nil && time.Since(info.ModTime()) > lockStale {
			os.Remove(fn)
			continue
		}
		time.Sleep(20 * time.Millisecond)
	}
	return nil, errors.New("commands: could not lock " + fn)
}
