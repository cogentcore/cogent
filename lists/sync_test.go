// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/base/github/githubtest"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSync sets the settings for syncing with a new test server,
// which is closed after the test.
func testSync(t *testing.T) *githubtest.Server {
	s := githubtest.NewServer()
	s.Token = "token"
	t.Cleanup(s.Close)
	api := githubAPI
	githubAPI = s.URL
	t.Cleanup(func() {
		githubAPI = api
		Settings.Sync = SyncSettings{}
	})
	return s
}

// setValue sets the value of the given field of the given item of the
// given list, as JSON, as an edit.
func setValue(ls *Lists, l *List, it *Item, field, value string) {
	it.Values[field] = json.RawMessage(value)
	l.values = reflect.Value{} // made again from the items
	ls.saveChanges()
}

// TestSync tests syncing two devices with a repository: the first one
// starts the repository, the second one, which is new, gets its lists,
// and the edits of the same item on both are merged.
func TestSync(t *testing.T) {
	s := testSync(t)
	b := core.NewBody()
	a := testLists(t, b)
	Settings.Sync = SyncSettings{Repo: "me/lists", Branch: "main", Token: "token"}
	other := NewLists(b)
	require.NoError(t, other.Open(t.TempDir()))
	b.AssertRender(t, "sync", func() {
		require.NoError(t, a.syncOnce())
		files := s.Files("me/lists", "main")
		assert.Contains(t, files, "README.md")
		assert.Equal(t, a.files(), func() map[string][]byte {
			delete(files, "README.md")
			return files
		}())

		require.NoError(t, other.syncOnce())
		b.AsyncLock()
		assert.Equal(t, a.files(), other.files(), "the new device has the lists")
		chores, otherChores := findList(a, "Chores"), findList(other, "Chores")
		assert.Equal(t, chores.ID, otherChores.ID)
		setValue(a, chores, chores.Items[0], "Task", `"Do the laundry"`)
		setValue(other, otherChores, otherChores.Items[0], "Done", `false`)
		b.AsyncUnlock()

		require.NoError(t, a.syncOnce())
		require.NoError(t, other.syncOnce()) // merges the change of a
		require.NoError(t, a.syncOnce())     // gets the merge
		b.AsyncLock()
		defer b.AsyncUnlock()
		for _, ls := range []*Lists{a, other} {
			it := findList(ls, "Chores").Items[0]
			assert.Equal(t, `"Do the laundry"`, string(it.Values["Task"]))
			assert.Equal(t, `false`, string(it.Values["Done"]))
			assert.NotNil(t, ls.tree.FindSyncNode(findList(ls, "Chores")), "the tree has the reloaded lists")
		}
		assert.Equal(t, a.files(), other.files())
		// the second sync of the other device merges, and the last sync
		// gets the merge; the first commit has the lists, in the order of
		// their random IDs
		commits := s.Commits("me/lists", "main")
		require.Len(t, commits, 4)
		assert.Equal(t, []string{"Edit Chores", "Edit Chores"}, commits[:2])
		assert.Contains(t, commits[2], "Groceries")
		assert.True(t, strings.HasSuffix(commits[2], ", the tree of lists, the list types"), commits[2])
		assert.Equal(t, "Start the lists", commits[3])
	})
}

// TestSyncButton tests a sync in the background, with its status
// in the button in the toolbar.
func TestSyncButton(t *testing.T) {
	s := testSync(t)
	b := core.NewBody()
	ls := testLists(t, b)
	Settings.Sync = SyncSettings{Repo: "me/lists", Branch: "main", Token: "token"}
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(ls.MakeToolbar)
	})
	b.AssertRender(t, "sync-button", func() {
		require.NotNil(t, ls.syncs.button)
		b.AsyncLock()
		icon, status := ls.syncStatus()
		b.AsyncUnlock()
		assert.Equal(t, icons.Cloud, icon)
		assert.Contains(t, status, "Not synced yet")
		ls.Sync()
		for range 100 { // until it is done
			ls.syncs.mu.Lock()
			running := ls.syncs.running
			ls.syncs.mu.Unlock()
			if !running {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.NoError(t, ls.syncs.err)
		assert.Equal(t, icons.CloudDone, ls.syncs.button.Icon)
		assert.Contains(t, ls.syncs.button.Tooltip, "Synced with me/lists")
		assert.Len(t, s.Commits("me/lists", "main"), 2)
	})
}

// TestSyncWaitsForEdits tests that a sync does not merge the changes in
// the repository while a text field has edits that are not applied yet,
// which would revert them.
func TestSyncWaitsForEdits(t *testing.T) {
	s := testSync(t)
	b := core.NewBody()
	ls := testLists(t, b)
	Settings.Sync = SyncSettings{Repo: "me/lists", Branch: "main", Token: "token"}
	b.AssertRender(t, "sync-waits", func() {
		require.NoError(t, ls.syncOnce())
		// a change in the repository, by another device
		c := s.Client()
		head, err := c.Head("me/lists", "main")
		require.NoError(t, err)
		b.AsyncLock()
		chores := findList(ls, "Chores")
		ls.SetCurrent(chores)
		path := listPath(chores.ID)
		lf := chores.listFile()
		b.AsyncUnlock()
		edited := *lf.Items[1]
		edited.Values = map[string]json.RawMessage{"Task": json.RawMessage(`"Fix the faucet"`)}
		lf.Items = []*Item{lf.Items[0], &edited}
		_, err = c.CommitFiles("me/lists", "main", head, "Edit Chores", map[string][]byte{path: marshal(lf)})
		require.NoError(t, err)

		// typing in the first text field of the table, after it is shown
		time.Sleep(200 * time.Millisecond)
		var tf *core.TextField
		b.AsyncLock()
		ls.table.WidgetWalkDown(func(w core.Widget, wb *core.WidgetBase) bool {
			if f, ok := w.(*core.TextField); ok && tf == nil {
				tf = f
			}
			return tf == nil
		})
		require.NotNil(t, tf)
		tf.SetFocus()
		b.AsyncUnlock()
		src := b.Scene.RenderWindow().SystemWindow.Events()
		src.KeyChord('x', key.CodeX, 0) // as text, which a key is not
		time.Sleep(200 * time.Millisecond)
		b.AsyncLock()
		assert.True(t, tf.Edited())
		b.AsyncUnlock()

		require.NoError(t, ls.syncOnce())
		b.AsyncLock()
		assert.Equal(t, `"Fix faucet"`, string(findList(ls, "Chores").Items[1].Values["Task"]), "not merged yet")
		assert.True(t, tf.Edited(), "still edited")
		tf.Text() // applies the edits
		b.AsyncUnlock()

		require.NoError(t, ls.syncOnce())
		b.AsyncLock()
		defer b.AsyncUnlock()
		chores = findList(ls, "Chores")
		assert.Equal(t, `"Fix the faucet"`, string(chores.Items[1].Values["Task"]), "merged")
		assert.Contains(t, string(chores.Items[0].Values["Task"]), "x", "with the edit")
	})
}

// TestSyncErrors tests the errors of syncing with a token that is not
// valid, and with a branch that is not in the repository.
func TestSyncErrors(t *testing.T) {
	s := testSync(t)
	ls := testLists(t, core.NewBody())
	assert.False(t, syncing())
	Settings.Sync = SyncSettings{Repo: "me/lists", Branch: "main"}
	assert.False(t, syncing(), "without a token")
	Settings.Sync.Token = "wrong"
	err := ls.syncOnce()
	assert.ErrorContains(t, err, "the token is not valid")

	Settings.Sync.Token = "token"
	require.NoError(t, ls.syncOnce())
	assert.Len(t, s.Commits("me/lists", "main"), 2)
	Settings.Sync.Branch = "other"
	err = ls.syncOnce()
	assert.ErrorContains(t, err, "is not found, or the token cannot access it")
	assert.Len(t, s.Commits("me/lists", "other"), 0, "no commit, of a README")
}
