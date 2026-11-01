// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package code

// formatGoal does nothing on the web, where Goal files are not formatted
// on save: goalfmt uses goimports, which does not build there.
func formatGoal(fname string) (bool, error) {
	return false, nil
}
