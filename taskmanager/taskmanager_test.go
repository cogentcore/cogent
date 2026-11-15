// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"os/exec"
	"slices"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tasks returns the tasks of the given table.
func tasks(tv *core.Table) []*Task {
	return *tv.Slice.(*[]*Task)
}

// TestTaskManager tests the list of tasks, which updates, pausing it,
// the info of a task, and ending the selected task, which stays selected
// as the list is updated and sorted again.
func TestTaskManager(t *testing.T) {
	cmd := exec.Command("sleep", "60")
	require.NoError(t, cmd.Start())
	defer cmd.Process.Kill()
	t.Cleanup(core.TrackWindows())

	b := core.NewBody()
	tv := makeApp(b, 50*time.Millisecond)
	index := func(pid int) int {
		return slices.IndexFunc(tasks(tv), func(t *Task) bool { return t.PID == int32(pid) })
	}
	require.GreaterOrEqual(t, index(os.Getpid()), 0, "this process")
	updated := func() {
		t.Helper()
		prev := tasks(tv)
		require.Eventually(t, func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return &tasks(tv)[0] != &prev[0]
		}, 10*time.Second, 10*time.Millisecond, "updated")
	}
	core.RenderImage(b, updated, core.Locked(b, func() {
		pause := core.ButtonByText(b.Scene, "Pause")
		require.NotNil(t, pause)
		pause.Send(events.Click)
		assert.Equal(t, "Resume", pause.Text)
		pause.Send(events.Click)
		assert.Equal(t, "Pause", pause.Text)

		i := index(cmd.Process.Pid)
		require.GreaterOrEqual(t, i, 0, "the sleep task")
		assert.Equal(t, "sleep", tasks(tv)[i].Name)
		tv.SelectIndexEvent(i, events.SelectOne)
	}), updated, updated, core.Locked(b, func() {
		require.GreaterOrEqual(t, tv.SelectedIndex, 0)
		require.Equal(t, int32(cmd.Process.Pid), tasks(tv)[tv.SelectedIndex].PID, "still selected after the updates")
		tv.Send(events.DoubleClick)
	}), core.Locked(b, func() {
		assert.NotNil(t, core.ButtonByText(core.TopScene(b), "OK"), "the info dialog")
		// (only the sleep task, which is selected, as checked above)
		require.Equal(t, int32(cmd.Process.Pid), tasks(tv)[tv.SelectedIndex].PID)
		core.ButtonByText(b.Scene, "End task").Send(events.Click)
	}), func() {
		done := make(chan error)
		go func() { done <- cmd.Wait() }()
		select {
		case err := <-done:
			assert.Error(t, err, "killed")
		case <-time.After(10 * time.Second):
			t.Error("the task did not end")
		}
	})
}
