// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databrowser

import "cogentcore.org/core/tensor/table"

// Result has info for one loaded result, in form of an etable.Table
type Result struct {

	// job id for results
	JobID string

	// description of job
	Message string

	// path to data
	Path string

	// result data
	Table *table.Table
}
