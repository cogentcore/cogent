// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

// bundleModule returns the given module script (with the given name, its
// URL if external) as a classic script that goja can run.
func (e *Engine) bundleModule(code, name string) (string, error) {
	return code, nil // TODO: bundle the imports with esbuild
}
