// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/text/parse/languages/bibtex"
)

// BibTexCited extracts markdown citations in the format [@Ref] from .md files and
// looks those up in a specified srcBib .bib file, writing the refs to output .bib
// file which can then be used by pandoc to efficiently process references.
func BibTexCited(srcDir, srcBib, outBib string, verbose bool) error {
	if srcBib == "" {
		return fmt.Errorf("BibTexCited: source .bib file must be specified")
	}
	exp := regexp.MustCompile(`\[(@([[:alnum:]]+-?)+(;[[:blank:]]+)?)+\]`)
	mds := fsx.Filenames(srcDir, ".md")
	if len(mds) == 0 {
		return fmt.Errorf("BibTexCited: No .md files found in: %s", srcDir)
	}

	bf, err := os.Open(srcBib)
	if err != nil {
		return err
	}

	parsed, err := bibtex.Parse(bf)
	if err != nil {
		bf.Close()
		return fmt.Errorf("BibTexCited: %s not loaded due to error(s), %s", srcBib, err.Error())
	}
	bf.Close()

	refs := map[string]int{}

	for _, md := range mds {
		fn := filepath.Join(srcDir, md)
		if verbose {
			fmt.Printf("processing: %v\n", fn)
		}
		f, err := os.Open(fn)
		if err != nil {
			fmt.Println(err)
			continue
		}
		scan := bufio.NewScanner(f)
		for scan.Scan() {
			cs := exp.FindAllString(string(scan.Bytes()), -1)
			for _, c := range cs {
				tc := c[1 : len(c)-1]
				sp := strings.Split(tc, "@")
				for _, ac := range sp {
					a := strings.TrimSpace(ac)
					a = strings.TrimSuffix(a, ";")
					if a == "" {
						continue
					}
					cc, _ := refs[a]
					cc++
					refs[a] = cc
				}
			}
		}
		f.Close()
	}
	if verbose {
		fmt.Printf("cites:\n%v\n", refs)
	}

	ob := bibtex.NewBibTex()
	ob.Preambles = parsed.Preambles
	ob.StringVar = parsed.StringVar

	for r, _ := range refs {
		be, has := parsed.Lookup(r)
		if has {
			ob.Entries = append(ob.Entries, be)
		} else {
			fmt.Printf("Error: Reference key: %v not found in %s\n", r, srcBib)
		}
	}

	ob.SortEntries()
	out := ob.PrettyString()

	of, err := os.Create(outBib)
	if err != nil {
		return err
	}
	of.WriteString(out)
	of.Close()
	return nil
}
