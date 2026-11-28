// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePicker is a fake Picker API of Google Photos, which serves the
// given files as the photos picked, by their IDs.
func fakePicker(t *testing.T, files map[string]string) *httptest.Server {
	var srv *httptest.Server
	polls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/sessions", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"id": "s1", "pickerUri": "https://photos.google.com/picker/s1",
			"pollingConfig": map[string]string{"pollInterval": "0.01s", "timeoutIn": "60s"}})
	})
	mux.HandleFunc("GET /v1/sessions/s1", func(w http.ResponseWriter, r *http.Request) {
		polls++
		json.NewEncoder(w).Encode(map[string]any{"id": "s1", "mediaItemsSet": polls >= 2,
			"pollingConfig": map[string]string{"pollInterval": "0.01s", "timeoutIn": "60s"}})
	})
	mux.HandleFunc("DELETE /v1/sessions/s1", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("GET /v1/mediaItems", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "s1", r.URL.Query().Get("sessionId"))
		var items []map[string]any
		for id, fn := range files {
			typ, mime := "PHOTO", "image/jpeg"
			if strings.HasSuffix(fn, ".mov") {
				typ, mime = "VIDEO", "video/quicktime"
			}
			items = append(items, map[string]any{"id": id, "type": typ, "createTime": "2020-01-02T03:04:05Z",
				"mediaFile": map[string]any{"baseUrl": srv.URL + "/media/" + id, "mimeType": mime, "filename": filepath.Base(fn)}})
		}
		json.NewEncoder(w).Encode(map[string]any{"mediaItems": items})
	})
	mux.HandleFunc("GET /media/", func(w http.ResponseWriter, r *http.Request) {
		id, ok := strings.CutSuffix(strings.TrimPrefix(r.URL.Path, "/media/"), "=d")
		require.True(t, ok, "downloads the original: =d")
		http.ServeFile(w, r, files[id])
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	old := pickerAPI
	pickerAPI = srv.URL + "/v1"
	t.Cleanup(func() { pickerAPI = old })
	return srv
}

func TestAddGoogle(t *testing.T) {
	lib, dir := testLibrary(t)
	require.NoError(t, lib.Scan(nil, nil))
	n := lib.Len()
	home, _ := os.UserHomeDir()
	src := filepath.Join(home, "Pictures", "CogentPhotos", "All")
	other := filepath.Join(t.TempDir(), "PXL_new.jpg") // a photo not in the test library
	require.NoError(t, copyFile(other, filepath.Join(src, "img_2015_03_02_14_23_50_n0.jpg")))
	same := filepath.Join(dir, "All", lib.Photos(func(ph *Photo) bool { return ph.OrigName == "1000061149" })[0].Filename())
	video := filepath.Join(t.TempDir(), "clip.mov")
	os.WriteFile(video, []byte("not really"), 0o644)

	srv := fakePicker(t, map[string]string{"g1": other, "g2": same, "g3": video})
	pc := &pickerClient{http: srv.Client()}
	ctx := context.Background()
	ps, err := pc.createSession(ctx)
	require.NoError(t, err)
	picked, err := pc.waitPicked(ctx, ps)
	require.NoError(t, err)
	require.True(t, picked)
	items, err := pc.listItems(ctx, ps.ID)
	require.NoError(t, err)
	require.Len(t, items, 3)
	res, err := lib.addGoogle(ctx, pc, items, nil)
	require.NoError(t, err)
	t.Log(res)
	require.Len(t, res.Added, 1)
	assert.Equal(t, 1, res.Already)
	assert.Equal(t, 1, res.Videos)
	added := res.Added[0]
	assert.Equal(t, "g1", added.GoogleID)
	assert.Equal(t, "img_2015_03_02_14_23_50_n0", added.Name)
	assert.Equal(t, "PXL_new", added.OrigName)
	assert.FileExists(t, lib.PhotoFile(added))
	assert.Equal(t, n+1, lib.Len())
	assert.NoDirExists(t, filepath.Join(dir, incomingDir))
	sameph := lib.Photos(func(ph *Photo) bool { return ph.OrigName == "1000061149" })[0]
	assert.Equal(t, "g2", sameph.GoogleID, "the same photo gets its ID")

	// again: all known
	res, err = lib.addGoogle(ctx, pc, items, nil)
	require.NoError(t, err)
	assert.Len(t, res.Added, 0)
	assert.Equal(t, 2, res.Already)
	assert.Equal(t, n+1, lib.Len())
}
