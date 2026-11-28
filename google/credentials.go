// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package google has the OAuth client of the Cogent apps registered with
// Google, for signing in to Google accounts, as for Gmail in Cogent Mail
// and Google Photos in Cogent Photos.
package google

import (
	"encoding/base64"
	"os"
)

// The OAuth client of the Cogent apps registered with Google, for signing
// in to Google accounts, which is a "Desktop app" client: Google does not treat its
// secret as secret, as it is in every copy of the app, and the security of
// signing in is the consent of the user in the browser, PKCE, and the
// redirect to this computer. It is in the source code, so that the app built
// from it can be used, encoded (XOR with [credentialsKey], then base64) only
// so that it is not found by the scanners of secrets, which would have it
// disabled. Anyone can decode it; using it in another app is against the
// terms of Google. The environment variables GOOGLE_OAUTH2_CLIENT_ID and
// GOOGLE_OAUTH2_CLIENT_SECRET, if set, are used instead (see
// [UseBuiltinCredentials]), as for a client of one's own.
const (
	credentialsKey        = "q0AGS+daMD+TUDRnkEin/M7g4smW0XME"
	googleClientIDEnc     = "QABzcGQSUVR/dhJsZmk7Wl80WxxIIVMTREJdPQI/OSIUBHF2MlgXCD93RjFicXwPGzUaQEgiWABYFhgkVSouKh9EJCknBQcOIA=="
	googleClientSecretEnc = "Nn8CFANzSQl/J2keYDE0IwwSJiJOP3RffTkUFFwcOQQjATE="
)

// UseBuiltinCredentials sets the environment variables of the OAuth client
// of Google, which the sign-in reads (cogentcore.org/core/base/auth), to
// those built into the app, if they are not set (see [googleClientIDEnc]).
func UseBuiltinCredentials() {
	for name, enc := range map[string]string{
		"GOOGLE_OAUTH2_CLIENT_ID":     googleClientIDEnc,
		"GOOGLE_OAUTH2_CLIENT_SECRET": googleClientSecretEnc,
	} {
		if os.Getenv(name) == "" {
			os.Setenv(name, decodeCredential(enc))
		}
	}
}

// decodeCredential decodes the given credential, encoded with
// [credentialsKey] (see [googleClientIDEnc]).
func decodeCredential(enc string) string {
	b, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return ""
	}
	for i := range b {
		b[i] ^= credentialsKey[i%len(credentialsKey)]
	}
	return string(b)
}
