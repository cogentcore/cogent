// Copyright (c) 2021, The Vector Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vector

// ActionHelpMap contains a set of help strings for different actions
// which are the names given e.g., in the ActStart, SaveUndo etc.
var ActionHelpMap = map[string]string{
	"Move":    "<b>Alt</b> = move without snapping, <b>Ctrl</b> = constrain to axis with smallest delta",
	"Reshape": "<b>Alt</b> = rotate, <b>Ctrl</b> = constraint to axis with smallest delta",
}
