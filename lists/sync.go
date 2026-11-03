// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/base/github"
	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/tree"
)

// The syncing of the lists with a GitHub repository, on all platforms,
// with the GitHub REST API, as configured in [SyncSettings]: after edits,
// at intervals, and on quitting, the changes in the repository since the
// last sync are merged with those here (see [mergeFiles]), and the merged
// files are committed, so that the repository has the history of all of
// the changes, from all of the devices.

// SyncSettings are the settings for syncing the lists with a GitHub
// repository.
type SyncSettings struct { //types:add

	// Repo is the GitHub repository for the lists, as owner/name, such as
	// a new private one, or none for no syncing. The files of the lists are
	// in its top directory, with the other files in it.
	Repo string

	// Branch is the branch of the repository for the lists.
	Branch string `default:"main"`

	// Token is a GitHub token that can read and write the repository: a
	// fine-grained personal access token, for only this repository, with
	// the Contents permission for Read and write, made at
	// github.com/settings/personal-access-tokens.
	Token Token

	// Interval is the interval for checking for changes in the
	// repository, in minutes.
	Interval float32 `default:"2" min:"0.5"`
}

// Token is a secret token, which is shown as a password.
type Token string

func (t *Token) Widget() core.Value {
	return core.NewTextField().SetTypePassword()
}

const (
	// syncFile is the file of the [syncState], in [Lists.Dir],
	// which is not synced.
	syncFile = "sync.json"

	// syncDelay is the delay of a sync after the last edit.
	syncDelay = 30 * time.Second

	// syncIdle is the time after the last sync after which a click or key
	// syncs, as when coming back to the app.
	syncIdle = time.Minute

	// syncTries is the number of times that a sync tries to commit its
	// changes, after merging the changes in the repository after each try.
	syncTries = 5
)

// githubAPI is the URL of the GitHub API, which is that of a test server
// in tests, or the default if empty.
var githubAPI = ""

// syncState is the state of the last sync, which is the base of the next
// three-way merge, saved in the [syncFile].
type syncState struct {

	// Repo and Branch are those of the [SyncSettings] for the state, which
	// starts over for another repository or branch.
	Repo, Branch string

	// Commit is the SHA of the last commit of the branch, as of the sync.
	Commit string

	// Files are the files of the lists, as of the sync, by path.
	Files map[string]string
}

// files returns the files of the state by path.
func (st *syncState) files() map[string][]byte {
	files := map[string][]byte{}
	for path, s := range st.Files {
		files[path] = []byte(s)
	}
	return files
}

// syncer has the state of the syncing of the lists.
type syncer struct {

	// running is whether a sync is running, and again whether another one
	// is requested meanwhile, which runs after it.
	running, again bool

	// auto is whether the lists sync automatically (see [Lists.StartSync]).
	auto bool

	// quitting is whether the app is quitting, when the sync does not
	// lock the window.
	quitting bool

	// timer is the timer for the sync after the last edit.
	timer *time.Timer

	// lastEdit is the time of the last edit, for the changes that win in
	// the merge, and lastSync that of the last sync.
	lastEdit, lastSync time.Time

	// err is the error of the last sync, if any.
	err error

	// button is the button of the status of the sync, in the toolbar.
	button *core.Button

	mu sync.Mutex
}

// syncing returns whether the lists sync with a repository: whether the
// [SyncSettings] have a repository and a token, which are needed for
// writing to it.
func syncing() bool {
	return Settings.Sync.Repo != "" && Settings.Sync.Token != ""
}

// Sync syncs the lists with the repository of the [SyncSettings] now,
// in the background, or after the sync that is running.
func (ls *Lists) Sync() { //types:add
	if !syncing() {
		return
	}
	sy := &ls.syncs
	sy.mu.Lock()
	if sy.running {
		sy.again = true
		sy.mu.Unlock()
		return
	}
	sy.running = true
	sy.mu.Unlock()
	go func() {
		ls.updateSyncButton()
		for {
			err := ls.syncOnce()
			if err != nil {
				slog.Error("Lists: error syncing", "repo", Settings.Sync.Repo, "err", err)
			}
			sy.mu.Lock()
			sy.err = err
			sy.lastSync = time.Now()
			again := sy.again
			sy.again = false
			sy.running = again
			sy.mu.Unlock()
			ls.updateSyncButton()
			if !again {
				return
			}
		}
	}()
}

// StartSync starts syncing the lists automatically, now, after edits, at
// the interval of the [SyncSettings], on a click or key after a while
// (see [syncIdle]), and on quitting.
func (ls *Lists) StartSync() {
	ls.syncs.auto = true
	ls.Sync()
	go func() {
		for {
			interval := time.Duration(max(Settings.Sync.Interval, 0.5) * float32(time.Minute))
			time.Sleep(interval)
			ls.Sync()
		}
	}()
	core.TheApp.AddQuitCleanFunc(func() {
		sy := &ls.syncs
		sy.mu.Lock()
		edited := sy.lastEdit.After(sy.lastSync)
		sy.quitting = true
		sy.mu.Unlock()
		if syncing() && edited {
			ls.syncOnce()
		}
	})
}

// edited schedules a sync after an edit, if the lists sync automatically.
func (ls *Lists) edited() {
	sy := &ls.syncs
	sy.mu.Lock()
	defer sy.mu.Unlock()
	sy.lastEdit = time.Now()
	if !sy.auto || !syncing() {
		return
	}
	if sy.timer != nil {
		sy.timer.Stop()
	}
	sy.timer = time.AfterFunc(syncDelay, ls.Sync)
}

// syncIfIdle syncs the lists if they last synced a while ago, as after
// coming back to the app.
func (ls *Lists) syncIfIdle() {
	sy := &ls.syncs
	sy.mu.Lock()
	idle := sy.auto && !sy.running && time.Since(sy.lastSync) > syncIdle
	sy.mu.Unlock()
	if idle {
		ls.Sync()
	}
}

// errEditing is the error of a merge that waits for the text that is
// being edited.
var errEditing = errors.New("waiting for the edits of a text field")

// syncOnce syncs the lists with the repository: it merges the changes in
// the repository since the last sync with those here, and commits the
// merged files, if they are changed, trying again if the repository
// changes meanwhile.
func (ls *Lists) syncOnce() error {
	cfg := Settings.Sync
	if cfg.Branch == "" {
		cfg.Branch = "main"
	}
	c := &github.Client{API: githubAPI, Token: string(cfg.Token), HTTP: syncHTTP}
	st := ls.readSyncState(cfg)
	for range syncTries {
		head, err := c.Head(cfg.Repo, cfg.Branch)
		switch github.StatusOf(err) {
		case http.StatusConflict: // a repository without commits, which the API needs
			err := c.PutFile(cfg.Repo, cfg.Branch, "README.md", "Start the lists", []byte("# Lists\n\nThe lists of [Cogent Lists](https://cogentcore.org/cogent/lists).\n"))
			if err != nil {
				return fmt.Errorf("making the first commit of %s: %w", cfg.Repo, syncError(err))
			}
			continue
		case http.StatusNotFound:
			return fmt.Errorf("the repository %s, with the branch %s, is not found, or the token cannot access it: the token must be for this repository, with the Contents permission for Read and write: %w", cfg.Repo, cfg.Branch, err)
		}
		if err != nil {
			return syncError(err)
		}
		base := st.files()
		remote, date, err := ls.remoteFiles(c, cfg.Repo, head, st, base)
		if err != nil {
			return syncError(err)
		}

		// the merge of the changes, here
		var merged map[string][]byte
		err = ls.syncLock(func() error {
			if ls.editingText() {
				return errEditing
			}
			local := ls.files()
			merged = local
			if head == st.Commit {
				return nil
			}
			ls.syncs.mu.Lock()
			// the repository wins on the first sync of a device, with no base
			oursWins := st.Commit != "" && ls.syncs.lastEdit.After(date)
			ls.syncs.mu.Unlock()
			merged = mergeFiles(base, local, remote, oursWins)
			if st.Commit == "" && remote[treeFile] != nil && ls.isDefault() {
				merged = remote // the first sync of a new device
			}
			if maps.EqualFunc(merged, local, bytes.Equal) {
				return nil
			}
			if err := writeFiles(ls.Dir, merged); err != nil {
				return err
			}
			ls.reload(local, merged)
			return nil
		})
		if errors.Is(err, errEditing) {
			time.AfterFunc(5*time.Second, ls.Sync)
			return nil
		}
		if err != nil {
			return err
		}

		// the commit of the merged changes
		changes := changedFiles(remote, merged)
		if len(changes) > 0 {
			head, err = c.CommitFiles(cfg.Repo, cfg.Branch, head, commitMessage(remote, merged, changes), changes)
			if errors.Is(err, github.ErrNotFastForward) { // the repository changed meanwhile
				continue
			}
			if err != nil {
				return syncError(err)
			}
		}
		st.Commit = head
		st.Files = map[string]string{}
		for path, b := range merged {
			st.Files[path] = string(b)
		}
		return ls.writeSyncState(st)
	}
	return fmt.Errorf("the repository %s kept changing while syncing", cfg.Repo)
}

// isDefault returns whether the lists are those of a new device,
// which has only the standard empty list.
func (ls *Lists) isDefault() bool {
	if ls.Root.NumChildren() != 1 {
		return false
	}
	l := ls.Root.Child(0).(*List)
	return !l.IsGroup() && l.Name == "To do" && len(l.Items) == 0
}

// syncError returns the given error of a request of the GitHub API, with
// an explanation for an error of the token.
func syncError(err error) error {
	switch github.StatusOf(err) {
	case http.StatusUnauthorized:
		return fmt.Errorf("the token is not valid, or has expired: %w", err)
	case http.StatusForbidden:
		return fmt.Errorf("the token cannot write the repository: it needs the Contents permission for Read and write: %w", err)
	}
	return err
}

// remoteFiles returns the files of the lists of the given commit of the
// given repository by path, getting those that are not in the given base
// files of the given last sync, with the time of the commit.
func (ls *Lists) remoteFiles(c *github.Client, repo, head string, st *syncState, base map[string][]byte) (map[string][]byte, time.Time, error) {
	if head == st.Commit {
		return base, time.Time{}, nil
	}
	cm, err := c.Commit(repo, head)
	if err != nil {
		return nil, time.Time{}, err
	}
	entries, err := c.Files(repo, cm.Tree)
	if err != nil {
		return nil, time.Time{}, err
	}
	remote := map[string][]byte{}
	for path, e := range entries {
		if !isListsFile(path) {
			continue
		}
		if b, ok := base[path]; ok && github.BlobSHA(b) == e.SHA {
			remote[path] = b
			continue
		}
		b, err := c.Blob(repo, e.SHA)
		if err != nil {
			return nil, time.Time{}, err
		}
		remote[path] = b
	}
	return remote, cm.Date, nil
}

// changedFiles returns the files of the given new files by path that
// are changed from the given old ones, with nil for those deleted.
func changedFiles(old, new map[string][]byte) map[string][]byte {
	changes := map[string][]byte{}
	for path, b := range new {
		if !bytes.Equal(old[path], b) {
			changes[path] = b
		}
	}
	for path := range old {
		if _, ok := new[path]; !ok {
			changes[path] = nil
		}
	}
	return changes
}

// commitMessage returns the message of the commit of the given changed
// files, from the given old files to the given new ones, with the names
// of the lists that are changed.
func commitMessage(old, new map[string][]byte, changes map[string][]byte) string {
	names := map[string]string{} // the names of the lists by path
	for _, files := range []map[string][]byte{old, new} {
		var root treeNode
		json.Unmarshal(files[treeFile], &root)
		var add func(tns []*treeNode)
		add = func(tns []*treeNode) {
			for _, tn := range tns {
				names[listPath(tn.ID)] = tn.Name
				add(tn.Lists)
			}
		}
		add(root.Lists)
	}
	var lists, others []string
	for _, path := range slices.Sorted(maps.Keys(changes)) {
		switch {
		case path == treeFile:
			others = append(others, "the tree of lists")
		case path == typesFile:
			others = append(others, "the list types")
		case changes[path] == nil:
			lists = append(lists, "delete "+names[path])
		default:
			lists = append(lists, names[path])
		}
	}
	if len(lists) > 3 {
		lists = []string{fmt.Sprintf("%d lists", len(lists))}
	}
	return "Edit " + strings.Join(append(lists, others...), ", ")
}

// syncLock runs the given function with the window locked, as for a
// change of the lists from another goroutine, unless the app is quitting.
func (ls *Lists) syncLock(f func() error) error {
	ls.syncs.mu.Lock()
	quitting := ls.syncs.quitting
	ls.syncs.mu.Unlock()
	if quitting || ls.Scene == nil || ls.Scene.RenderWindow() == nil {
		return f()
	}
	ls.AsyncLock()
	defer ls.AsyncUnlock()
	return f()
}

// editingText returns whether a text field of the lists has edits that
// are not applied yet, which a merge of the changes in the repository
// would revert, as while typing.
func (ls *Lists) editingText() bool {
	editing := false
	ls.WidgetWalkDown(func(w core.Widget, wb *core.WidgetBase) bool {
		if tf, ok := w.(*core.TextField); ok && tf.Edited() {
			editing = true
			return tree.Break
		}
		return tree.Continue
	})
	return editing
}

// reload reloads the lists from the given files by path, after a merge
// of the changes in the repository with the given local ones, keeping
// the current list, and the undo of the lists that are not changed.
func (ls *Lists) reload(local, files map[string][]byte) {
	old := map[string]*List{}
	ls.Root.WalkDown(func(n tree.Node) bool {
		old[n.(*List).ID] = n.(*List)
		return tree.Continue
	})
	var current string
	if ls.Current != nil {
		current = ls.Current.ID
	}
	ls.load(files)
	ls.Current = nil
	ls.Root.WalkDown(func(n tree.Node) bool {
		l := n.(*List)
		path := listPath(l.ID)
		if o := old[l.ID]; o != nil && bytes.Equal(local[path], files[path]) {
			l.undos, l.undoState = o.undos, o.undoState
		}
		if l.ID == current {
			ls.Current = l
		}
		return tree.Continue
	})
	ls.tableList = nil // for the column widths and sort of the current list
	ls.Update()
	if ls.Current != nil {
		if tn := ls.tree.FindSyncNode(ls.Current); tn != nil {
			tn.Select()
		}
	}
}

// readSyncState returns the state of the last sync, for the given
// settings, which is empty for another repository or branch.
func (ls *Lists) readSyncState(cfg SyncSettings) *syncState {
	st := &syncState{}
	b, err := os.ReadFile(filepath.Join(ls.Dir, syncFile))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		core.ErrorSnackbar(ls, err, "Error reading the state of the sync")
	}
	json.Unmarshal(b, st)
	if st.Repo != cfg.Repo || st.Branch != cfg.Branch {
		st = &syncState{}
	}
	st.Repo, st.Branch = cfg.Repo, cfg.Branch
	return st
}

// writeSyncState writes the given state of the last sync.
func (ls *Lists) writeSyncState(st *syncState) error {
	return os.WriteFile(filepath.Join(ls.Dir, syncFile), marshal(st), 0o644)
}

// updateSyncButton updates the button of the status of the sync,
// from another goroutine.
func (ls *Lists) updateSyncButton() {
	ls.syncs.mu.Lock()
	quitting := ls.syncs.quitting
	ls.syncs.mu.Unlock()
	if bt := ls.syncs.button; bt != nil && bt.This != nil && !quitting && bt.Scene != nil && bt.Scene.RenderWindow() != nil {
		bt.AsyncLock()
		bt.Update()
		bt.AsyncUnlock()
	}
}

// syncStatus returns the icon and text of the status of the sync.
func (ls *Lists) syncStatus() (icons.Icon, string) {
	sy := &ls.syncs
	sy.mu.Lock()
	defer sy.mu.Unlock()
	switch {
	case sy.running:
		return icons.CloudSync, "Syncing with " + Settings.Sync.Repo
	case sy.err != nil:
		return icons.CloudOff, "Error syncing with " + Settings.Sync.Repo + ": " + sy.err.Error() + "; click to sync now"
	case sy.lastSync.IsZero():
		return icons.Cloud, "Not synced yet with " + Settings.Sync.Repo + "; click to sync now"
	}
	return icons.CloudDone, "Synced with " + Settings.Sync.Repo + " at " + sy.lastSync.Format("15:04") + "; click to sync now"
}
