// Copyright (c) 2021, The Grid Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grid

import (
	"fmt"
	"math"
	"strings"

	"github.com/goki/ki/sliceclone"
)

// DashMulWidth returns the dash array multiplied by the line width -- what is actually set
func DashMulWidth(lwidth float64, dary []float64) []float64 {
	mary := sliceclone.Float64(dary)
	for i := range mary {
		mary[i] *= lwidth
	}
	return mary
}

// DashString returns string of dash array values
func DashString(dary []float64) string {
	ds := ""
	for i := range dary {
		ds += fmt.Sprintf("%g,", dary[i])
	}
	ds = strings.TrimSuffix(ds, ",")
	return ds
}

// DashMatchArray returns the matching dash pattern for given array and line width.
// divides array and matches with wide tolerance.
// returns true if no match and thus new dash pattern was added, else false.
func DashMatchArray(lwidth float64, dary []float64) (bool, string) {
	if lwidth == 0 { // no div-by-0
		lwidth = 1
	}
	sz := len(dary)
	if sz == 0 {
		return false, "solid"
	}
	mary := sliceclone.Float64(dary)
	for i := range mary {
		mary[i] /= lwidth
	}
	for k, v := range AllDashesMap {
		if len(v) != sz {
			continue
		}
		match := true
		for i := range mary {
			if math.Abs(v[i]-mary[i]) > 0.5 {
				match = false
				break
			}
		}
		if match {
			return false, k
		}
	}
	// new beast -- add it
	return true, AddNewDash(mary)
}

// AddNewDash adds new dash pattern to available list, creating name based on pattern,
// which is returned.  the given array is copied before storing, just in case.
func AddNewDash(dary []float64) string {
	nm := "custom"
	for i := range dary {
		nm += fmt.Sprintf("-%g", dary[i])
	}
	AllDashNames = append(AllDashNames, nm)
	AllDashesMap[nm] = sliceclone.Float64(dary)
	return nm
}

// StdDashNames are standard dash patterns
var StdDashNames = []string{
	"solid",
	"dash-1-1",
	"dash-1-2",
	"dash-1-3",
	"dash-1-4",
	"dash-1-6",
	"dash-1-8",
	"dash-1-12",
	"dash-1-24",
	"dash-1-48",
	"dash-empty",
	"dash-2-1",
	"dash-3-1",
	"dash-4-1",
	"dash-6-1",
	"dash-8-1",
	"dash-12-1",
	"dash-24-1",
	"dash-2-2",
	"dash-3-3",
	"dash-4-4",
	"dash-6-6",
	"dash-8-8",
	"dash-12-12",
	"dash-24-24",
	"dash-2-4",
	"dash-4-2",
	"dash-2-6",
	"dash-6-2",
	"dash-4-8",
	"dash-8-4",
	"dash-2-1-012-1",
	"dash-4-2-1-2",
	"dash-8-2-1-2",
	"dash-012-012",
	"dash-014-014",
	"dash-0110-0110",
}

// StdDashesMap are standard dash patterns
var StdDashesMap = map[string][]float64{
	"solid":          {},
	"dash-1-1":       {1, 1},
	"dash-1-2":       {1, 2},
	"dash-1-3":       {1, 3},
	"dash-1-4":       {1, 4},
	"dash-1-6":       {1, 6},
	"dash-1-8":       {1, 8},
	"dash-1-12":      {1, 12},
	"dash-1-24":      {1, 24},
	"dash-1-48":      {1, 48},
	"dash-empty":     {0, 11},
	"dash-2-1":       {2, 1},
	"dash-3-1":       {3, 1},
	"dash-4-1":       {4, 1},
	"dash-6-1":       {6, 1},
	"dash-8-1":       {8, 1},
	"dash-12-1":      {12, 1},
	"dash-24-1":      {24, 1},
	"dash-2-2":       {2, 2},
	"dash-3-3":       {3, 3},
	"dash-4-4":       {4, 4},
	"dash-6-6":       {6, 6},
	"dash-8-8":       {8, 8},
	"dash-12-12":     {12, 12},
	"dash-24-24":     {24, 24},
	"dash-2-4":       {2, 4},
	"dash-4-2":       {4, 2},
	"dash-2-6":       {2, 6},
	"dash-6-2":       {6, 2},
	"dash-4-8":       {4, 8},
	"dash-8-4":       {8, 4},
	"dash-2-1-012-1": {2, 1, 0.5, 1},
	"dash-4-2-1-2":   {4, 2, 1, 2},
	"dash-8-2-1-2":   {8, 2, 1, 2},
	"dash-012-012":   {0.5, 0.5},
	"dash-014-014":   {0.25, 0.25},
	"dash-0110-0110": {0.1, 0.1},
}

// AllDashesMap contains all of the available Dashes.
// it is initialized from StdDashesMap
var AllDashesMap map[string][]float64

// AllDashNames contains all of the available dash names.
// it is initialized from StdDashNames.
var AllDashNames []string

func init() {
	AllDashesMap = make(map[string][]float64, len(StdDashesMap))
	AllDashNames = make([]string, len(StdDashNames))
	for k, v := range StdDashesMap {
		AllDashesMap[k] = v
	}
	for i, v := range StdDashNames {
		AllDashNames[i] = v
	}
}
