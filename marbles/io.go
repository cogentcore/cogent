package main

import (
	"fmt"
	"path/filepath"

	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/core"
)

// SaveLast saves to the last opened or saved file
func (gr *Graph) SaveLast() { //types:add
	if gr.State.File != "" {
		TheGraph.SaveJSON(gr.State.File)
	} else {
		HandleError(fmt.Errorf("Graph.SaveLast: no last file"))
	}
}

// OpenJSON opens a graph from a JSON file
func (gr *Graph) OpenJSON(filename core.Filename) error { //types:add
	err := jsonx.Open(gr, string(filename))
	if HandleError(err) {
		return err
	}
	gr.State.File = filename
	gr.graphAndUpdate()
	return nil
}

// OpenAutoSave opens the last graphed graph, stays between sessions of the app
func (gr *Graph) OpenAutoSave() error {
	err := jsonx.Open(gr, filepath.Join(core.TheApp.AppDataDir(), "autosave.json"))
	if HandleError(err) {
		return err
	}
	gr.graphAndUpdate()
	return nil
}

// SaveJSON saves a graph to a JSON file
func (gr *Graph) SaveJSON(filename core.Filename) error { //types:add
	var err error
	if TheSettings.PrettyJSON {
		err = jsonx.SaveIndent(gr, string(filename))
	} else {
		err = jsonx.Save(gr, string(filename))
	}
	if HandleError(err) {
		return err
	}
	gr.State.File = filename
	gr.graphAndUpdate()
	return nil
}

// AutoSave saves the graph to autosave.json, called automatically
func (gr *Graph) AutoSave() error {
	filename := filepath.Join(core.TheApp.AppDataDir(), "autosave.json")
	var err error
	if TheSettings.PrettyJSON {
		err = jsonx.SaveIndent(gr, filename)
	} else {
		err = jsonx.Save(gr, filename)
	}
	if HandleError(err) {
		return err
	}
	return nil
}
