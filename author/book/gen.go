// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package book

import "embed"

//go:generate cosh build
//go:generate core generate

//go:embed pandoc-inputs/*
var PandocInputs embed.FS
