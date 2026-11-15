// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDocumentText tests fetching a local web page that a server asks
// to show, as text.
func TestDocumentText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/page":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("<html><head><title>Doc</title><style>x{}</style></head><body><h1>Head</h1><p>some   text</p><script>no()</script></body></html>"))
		case "/plain":
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("plain text"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	assert.True(t, IsLocalURL(srv.URL))
	assert.False(t, IsLocalURL("https://example.com/x"))
	assert.False(t, IsLocalURL("file:///x"))

	title, text, err := DocumentText(srv.URL + "/page")
	require.NoError(t, err)
	assert.Equal(t, "Doc", title)
	assert.Contains(t, text, "Head\n")
	assert.Contains(t, text, "some text")
	assert.NotContains(t, text, "no()")
	_, text, err = DocumentText(srv.URL + "/plain")
	require.NoError(t, err)
	assert.Equal(t, "plain text", text)
	_, _, err = DocumentText(srv.URL + "/none")
	assert.Error(t, err)
	_, _, err = DocumentText("http://127.0.0.1:1/none")
	assert.Error(t, err)
}

// TestServerStatuses tests the statuses of the servers of the registry,
// their install instructions, and the root directories of projects.
func TestServerStatuses(t *testing.T) {
	rg := NewRegistry(nil)
	sts := rg.Statuses("darwin", true)
	require.Len(t, sts, len(rg.Servers))
	for _, st := range sts {
		assert.NotEmpty(t, st.ID)
		if st.ID == "gopls" && st.Installed {
			assert.NotEmpty(t, st.Path)
			assert.NotEmpty(t, st.Version, "probed")
		}
	}
	assert.Equal(t, "", probeVersion(filepath.Join(t.TempDir(), "none"), nil))

	scs, err := rg.Select([]string{"gopls"}, nil)
	require.NoError(t, err)
	sc := scs[0]
	ins := sc.Install.Instructions("darwin")
	assert.Contains(t, ins, "go install golang.org/x/tools/gopls@latest")
	assert.Contains(t, sc.Install.Instructions("linux")[len(sc.Install.Instructions("linux"))-1], "docs:")
	var nilSpec *InstallSpec
	assert.Nil(t, nilSpec.Instructions(""))
	all := (&InstallSpec{Brew: "b", GoModule: "g", Npm: []string{"n"}, Pip: "p", Apt: "a", Docs: "d"}).Instructions("linux")
	assert.Len(t, all, 6)

	_, err = Install(&ServerConfig{ID: "none"}, "darwin")
	assert.Error(t, err, "no install methods")

	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "a", "b"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), nil, 0o644))
	fpath := filepath.Join(tmp, "a", "b", "c.go")
	assert.Equal(t, tmp, (&ServerConfig{RootMarkers: []string{"go.mod"}}).RootDir(fpath))
	assert.Equal(t, filepath.Dir(fpath), (&ServerConfig{}).RootDir(fpath), "no markers")
	assert.Equal(t, filepath.Dir(fpath), (&ServerConfig{RootMarkers: []string{"none.mod"}}).RootDir(fpath), "not found")
}

// TestServerHandshake tests starting gopls in a sample project and
// listing its capabilities, as the lsp-server test command does.
func TestServerHandshake(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found in PATH:", err)
	}
	t.Setenv("GOWORK", "off")
	rg := NewRegistry(nil)
	scs, err := rg.Select([]string{"gopls"}, nil)
	require.NoError(t, err)
	tr := TestServer(scs[0], 60*time.Second)
	require.True(t, tr.OK, tr.Err)
	assert.NotEmpty(t, tr.Version)
	assert.Contains(t, tr.Caps, "definition")
	assert.True(t, tr.Symbols)
	tr = TestServer(&ServerConfig{ID: "none", Exec: "no-such-server"}, time.Second)
	assert.False(t, tr.OK)
	assert.NotEmpty(t, tr.Err)
}

// TestMarkupFlatten tests getting the text of the forms of hover and
// documentation contents.
func TestMarkupFlatten(t *testing.T) {
	k, v := markupFlatten(rawJSON(`{"kind":"markdown","value":"**x**"}`))
	assert.Equal(t, "markdown", k)
	assert.Equal(t, "**x**", v)
	k, v = markupFlatten(rawJSON(`"plain"`))
	assert.Equal(t, "plaintext", k)
	assert.Equal(t, "plain", v)
	k, v = markupFlatten(rawJSON(`["a", {"language":"go","value":"func f()"}]`))
	assert.Equal(t, "markdown", k)
	assert.Equal(t, "a\n```go\nfunc f()\n```\n", v)
	k, v = markupFlatten(nil)
	assert.Equal(t, "", k+v)
	k, v = markupFlatten(rawJSON(`3`))
	assert.Equal(t, "", k+v)

	assert.True(t, jsonObjectBool(rawJSON(`{"resolveProvider":true}`), "resolveProvider"))
	assert.False(t, jsonObjectBool(rawJSON(`true`), "resolveProvider"))
	assert.Equal(t, "file:///a", DocumentURI("file:///a").String())

	var sh *SignatureHelp
	assert.Equal(t, 0, sh.ActiveParam())
	ap := 2
	assert.Equal(t, 2, (&SignatureHelp{ActiveParameter: &ap}).ActiveParam())
}
