// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
)

func init() {
	core.AllSettings = slices.Insert(core.AllSettings, 1, core.Settings(Settings))
}

// Settings is the currently active global Cogent Mail settings instance.
var Settings = &SettingsData{
	Name: "Mail",
	File: filepath.Join(core.TheApp.DataDir(), "Cogent Mail", "settings.toml"),
}

// SettingsData is the data type for the global Cogent Mail settings.
type SettingsData struct { //types:add
	core.SettingsBase

	// Accounts are the email accounts the user is signed into.
	Accounts []string

	// FlatList shows each message on its own in the list of messages,
	// instead of grouped into conversations.
	FlatList bool

	// Signatures are the signatures that can be added to the end of the
	// messages, in markdown.
	Signatures []Signature

	// DefaultSignatures are the names of the signatures added to new
	// messages from each account, by its email address.
	DefaultSignatures map[string]string
}

// Signature is a signature that can be added to the end of messages.
type Signature struct {

	// Name is the name of the signature, by which the accounts
	// choose it (see [SettingsData.DefaultSignatures]).
	Name string

	// Text is the text of the signature, in markdown, where each line
	// is on its own line.
	Text SignatureText
}

// SignatureText is the markdown text of a [Signature], which is edited
// in a multi-line editor (see [SignatureButton]).
type SignatureText string

// signatureText returns the text of the default signature of the given
// account, after the standard "-- " separator line, with a line break at
// the end of each line, or "" if it has none.
func signatureText(email string) string {
	name := Settings.DefaultSignatures[email]
	if name == "" {
		return ""
	}
	for _, sig := range Settings.Signatures {
		text := string(sig.Text)
		if sig.Name != name || strings.TrimSpace(text) == "" {
			continue
		}
		lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
		// two spaces at the end of a line are a line break in markdown
		return "--  \n" + strings.Join(lines, "  \n") + "\n"
	}
	return ""
}

// friendlyLabelName converts the given label name to a user-friendly version.
func friendlyLabelName(name string) string {
	name = strings.TrimPrefix(name, "[Gmail]/")
	if f, ok := friendlyLabelNames[name]; ok {
		return f
	}
	return name
}

var friendlyLabelNames = map[string]string{
	"INBOX":     "Inbox",
	"Sent Mail": "Sent",
}

// skipLabels are a temporary set of labels that should not be cached or displayed.
// TODO: figure out a better approach to this.
var skipLabels = map[string]bool{
	"[Gmail]":           true,
	"[Gmail]/All Mail":  true,
	"[Gmail]/Important": true,
	"[Gmail]/Starred":   true,
}

// labelIcons are the icons for each friendly label name.
var labelIcons = map[string]icons.Icon{
	"Inbox":  icons.Inbox,
	"Drafts": icons.Draft,
	"Sent":   icons.Send,
	"Spam":   icons.Report,
	"Junk":   icons.Report,
	"Trash":  icons.Delete,
}
