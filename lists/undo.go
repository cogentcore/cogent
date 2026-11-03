// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lists

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/undo"
)

// undoState is the state of a list for undo: its type and items,
// without the widths of its columns, which are not edits of it.
type undoState struct {
	Type  string
	Items []*Item
}

// undoLines returns the state of the given list for undo,
// as the lines of its JSON.
func undoLines(l *List) []string {
	b, err := json.MarshalIndent(&undoState{Type: l.TypeName, Items: l.Items}, "", "  ")
	errors.Log(err)
	return strings.Split(string(b), "\n")
}

// undoStack returns the undo stack of the given list, making it if needed.
func (l *List) undoStack() *undo.Stack {
	if l.undos == nil {
		l.undos = &undo.Stack{}
	}
	return l.undos
}

// initUndo starts the undo of the given list with its current state,
// if it is not started.
func initUndo(l *List) {
	if l.undoState == nil {
		l.undoState = undoLines(l)
	}
}

// saveUndo saves the state of the given list before the given action,
// after the action, if the action changed it.
func saveUndo(l *List, action string) {
	state := undoLines(l)
	if l.undoState == nil {
		l.undoState = state
		return
	}
	if slices.Equal(state, l.undoState) {
		return
	}
	l.undoStack().Save(action, "", l.undoState)
	l.undoState = state
}

// Undo undoes the last edit of the current list, including a change of
// its type, which also undoes the edits after it.
func (ls *Lists) Undo() { //types:add
	l := ls.Current
	if l == nil || l.IsGroup() || len(l.undoStack().Records) == 0 || l.undoStack().Index < 0 {
		core.MessageSnackbar(ls, "Nothing to undo")
		return
	}
	if l.undoStack().MustSaveUndoStart() { // so that the undo can be redone
		l.undoStack().SaveUndoStart(undoLines(l))
	}
	act, _, state := l.undoStack().Undo()
	ls.restoreUndo(l, state, "Undid "+act)
}

// Redo redoes the last undone edit of the current list.
func (ls *Lists) Redo() { //types:add
	l := ls.Current
	if l == nil || l.IsGroup() || !l.undoStack().HasRedoAvail() {
		core.MessageSnackbar(ls, "Nothing to redo")
		return
	}
	act, _, state := l.undoStack().Redo()
	ls.restoreUndo(l, state, "Redid "+act)
}

// restoreUndo restores the given state of the given list for undo or
// redo, with the given message.
func (ls *Lists) restoreUndo(l *List, state []string, msg string) {
	if state == nil {
		return
	}
	var st undoState
	if errors.Log(json.Unmarshal([]byte(strings.Join(state, "\n")), &st)) != nil {
		return
	}
	l.TypeName = st.Type
	l.Items = st.Items
	l.values = reflect.Value{}
	l.undoState = state
	ls.keepUsedTypes() // if its type was deleted since
	ls.saveChanges()
	ls.Update()
	ls.updateItems()
	core.MessageSnackbar(ls, msg)
}
