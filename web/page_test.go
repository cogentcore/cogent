// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
)

// TestPageFile shows the page in the file of the WEB_PAGE environment
// variable, served as the URL in WEB_URL (with its other resources from
// the web), printing its first texts, and saving its image to WEB_OUT.
func TestPageFile(t *testing.T) {
	file := os.Getenv("WEB_PAGE")
	if file == "" {
		t.Skip("set WEB_PAGE to a saved HTML page")
	}
	b := core.NewBody()
	pg := NewPage(b)
	pg.Context.GetURL = func(u string) (*http.Response, error) {
		pu, _ := url.Parse(u)
		if pu.Path == "/" || pu.Path == "" || pu.Path == os.Getenv("WEB_PATH") {
			f, err := os.Open(file)
			if err != nil {
				return nil, err
			}
			return &http.Response{StatusCode: 200, Status: "200 OK", Body: f, Request: &http.Request{URL: pu}}, nil
		}
		return http.Get(u)
	}
	pg.OpenURL(os.Getenv("WEB_URL"), nil)
	img := b.RenderImage(func() {
		n := 0
		pg.WalkDown(func(nd tree.Node) bool {
			if tx, ok := nd.(*core.Text); ok && n < 5 {
				fmt.Println(tx.Text, tx.Geom.Pos.Total, tx.Geom.TotalBBox, tx.Styles.Color, tx.IsVisible())
				n++
			}
			return true
		})
	})
	if out := os.Getenv("WEB_OUT"); out != "" {
		imagex.Save(img, out)
	}
}
