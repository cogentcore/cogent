package script

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// SRCURL=url POS=line:col MODE=module|lower|bundleurl
func TestZZLower(t *testing.T) {
	u := os.Getenv("SRCURL")
	if u == "" {
		t.Skip()
	}
	get := func(u string) (*http.Response, error) {
		req, _ := http.NewRequest("GET", u, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; CogentCore; +https://cogentcore.org)")
		return http.DefaultClient.Do(req)
	}
	var doc *html.Node
	if p := os.Getenv("PAGE"); p != "" {
		resp, err := get(p)
		if err == nil {
			doc, _ = html.Parse(resp.Body)
			resp.Body.Close()
		}
	}
	e, _ := New(doc, Options{PageURL: os.Getenv("PAGE"), GetURL: get})
	e.readImportMaps()
	var out string
	var err error
	switch os.Getenv("MODE") {
	case "lower":
		src, _ := e.getScript(u)
		out, err = lower(src, u)
	default:
		src, _ := e.getScript(u)
		out, _, err = e.transformModule(src, u, "")
	}
	if err != nil {
		t.Fatal(err)
	}
	pos := strings.Split(os.Getenv("POS"), ":")
	ln, _ := strconv.Atoi(pos[0])
	col, _ := strconv.Atoi(pos[1])
	ls := strings.Split(out, "\n")
	for i := max(0, ln-8); i < ln+2 && i < len(ls); i++ {
		l := ls[i]
		if i == ln-1 && len(l) > 300 {
			l = l[max(0, col-200):min(len(l), col+200)]
		} else if len(l) > 300 {
			l = l[:300]
		}
		fmt.Println(i+1, l)
	}
}
