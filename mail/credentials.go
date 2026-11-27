// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import "cogentcore.org/cogent/google"

// useBuiltinCredentials sets the environment variables of the OAuth client
// of Google to those built into the Cogent apps, if they are not set
// (see [google.UseBuiltinCredentials]).
func useBuiltinCredentials() {
	google.UseBuiltinCredentials()
}
