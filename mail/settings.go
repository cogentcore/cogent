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
	core.AddAppSettings(Settings)
}

// Settings is the currently active global Cogent Mail settings instance.
var Settings = &SettingsData{
	Name: "Mail",
	File: filepath.Join(core.TheApp.DataDir(), "Cogent Mail", "settings.toml"),
}

// SettingsData is the data type for the global Cogent Mail settings.
type SettingsData struct { //types:add
	core.SettingsBase

	// Accounts are the email accounts, added with Add account in the
	// menu of an account.
	// Those of earlier versions are email addresses, which are read as
	// Gmail accounts (see [Account.UnmarshalText]).
	Accounts []Account `display:"no-inline"`

	// ListViews are the options of the view of the list of messages,
	// also set in its view menu.
	ListViews ListViews `grow:"1"`

	// FilingAccount is the account whose folders are offered first for
	// filing messages (Control+O), from any account, as for filing all of
	// them into the folders of one account. By default it is the account
	// of the message.
	FilingAccount AccountName

	// ContactsAccount is the account whose Google Contacts the contacts
	// made here are added to, of those whose contacts are synced (in the
	// menu of the account). By default it is the first of them.
	ContactsAccount ContactsAccountName

	// AutoBCC sends a blind copy (Bcc) of every message to its sender,
	// whose address is put in the Bcc field of each new message, where
	// it can be removed for that message.
	AutoBCC bool

	// Signatures are the signatures that can be added to the end of the
	// messages, in markdown.
	Signatures []Signature

	// DefaultSignatures are the names of the signatures added to new
	// messages from each account, by its email address.
	DefaultSignatures map[string]SignatureName

	// ColumnWidths are the widths of the columns of the list of messages
	// that are set, by the names of the fields (see [core.Table.ColumnWidths]).
	ColumnWidths map[string]float32 `display:"-"`

	// Sort is the column the list of messages is sorted by, with its
	// direction (see [core.Table.SortFieldName]).
	Sort string `display:"-"`

	// Splits are the proportions of the panels.
	Splits []float32 `display:"-"`

	// OpenMailboxes records whether each account and mailbox with
	// subfolders is open in the tree of mailboxes, by [mailboxKey], for
	// those that have been opened or closed.
	OpenMailboxes map[string]bool `display:"-"`

	// SyncContacts are the Gmail accounts whose contacts are synced with
	// Google Contacts, by email address (see [App.SetSyncContacts]).
	SyncContacts map[string]bool `display:"-"`

	// Keys are the keyboard shortcuts of the actions in the main window,
	// by their chords, with Command for Meta on the Mac and Control
	// elsewhere.
	Keys KeyMap

	// KeysKnown are the actions whose default keyboard shortcuts have been
	// added to [SettingsData.Keys], so that those of actions added in later
	// versions are added, but not again those that were removed.
	KeysKnown []KeyFunctions `display:"-"`

	// ShownAccount and ShownMailbox are the account and mailbox shown
	// last, which are shown when the app starts again.
	ShownAccount, ShownMailbox string `display:"-"`
}

// Defaults sets the default settings: the list of messages groups them
// into conversations. The keyboard shortcuts are [DefaultKeys], set in
// [SettingsData.Apply] if there are none, as the saved ones would be
// merged into them, so that removed ones would come back.
func (sd *SettingsData) Defaults() {
	sd.ListViews.SetFlag(true, ViewConversations)
	sd.ListViews.SetFlag(true, ViewStars)
}

// Apply makes sure that each account has an entry in the
// [SettingsData.DefaultSignatures], so that it can be set, without those
// of blank accounts (as one just added in the settings, before its
// address is typed) or unset ones of accounts that are gone.
func (sd *SettingsData) Apply() {
	if theApp != nil && theApp.contacts != nil { // as the account of new contacts changes
		theApp.contacts.NewAccount = newContactsAccount()
	}
	sd.addDefaultKeys()
	if sd.DefaultSignatures == nil {
		sd.DefaultSignatures = map[string]SignatureName{}
	}
	emails := sd.Emails()
	for account, sig := range sd.DefaultSignatures {
		if strings.TrimSpace(account) == "" || (sig == "" && !slices.Contains(emails, account)) {
			delete(sd.DefaultSignatures, account)
		}
	}
	for _, account := range emails {
		if _, ok := sd.DefaultSignatures[account]; !ok {
			sd.DefaultSignatures[account] = ""
		}
	}
	if theApp != nil && sd == Settings { // an account added here is signed in to
		accounts := emails
		go func() {
			theApp.AsyncLock()
			defer theApp.AsyncUnlock()
			theApp.accountsAdded(accounts)
		}()
	}
}

// ListViews are the options of the view of the list of messages,
// which are shown as switches in its view menu.
type ListViews int64 //enums:bitflag -trim-prefix View

const (
	// ViewConversations groups the messages into conversations,
	// instead of showing each one on its own.
	ViewConversations ListViews = iota

	// ViewCompact shows each message in two lines, in one column,
	// as always on mobile.
	ViewCompact

	// ViewLabels shows the other labels of the messages, in a column,
	// or after the subject in the compact view.
	ViewLabels

	// ViewAttachments shows the number of attachments of the messages, in
	// a column, or after the subject in the compact view, empty for none.
	ViewAttachments

	// ViewStars shows the stars of the starred messages, in a column,
	// or before the sender in the compact view.
	ViewStars
)

// AccountName is the email address of one of the [SettingsData.Accounts],
// chosen from them, or "" for none.
type AccountName string

func (an AccountName) Widget() core.Value {
	ch := core.NewChooser()
	ch.Items = append(ch.Items, core.ChooserItem{Value: AccountName(""), Text: "(the account of the message)"})
	for _, account := range Settings.Emails() {
		ch.Items = append(ch.Items, core.ChooserItem{Value: AccountName(account), Text: account})
	}
	return ch
}

// ContactsAccountName is the email address of one of the accounts whose
// contacts are synced, chosen from them, or "" for the first of them.
type ContactsAccountName string

func (an ContactsAccountName) Widget() core.Value {
	ch := core.NewChooser()
	ch.Items = append(ch.Items, core.ChooserItem{Value: ContactsAccountName(""), Text: "(the first that syncs contacts)"})
	for _, account := range Settings.Emails() {
		if Settings.SyncContacts[account] {
			ch.Items = append(ch.Items, core.ChooserItem{Value: ContactsAccountName(account), Text: account})
		}
	}
	return ch
}

// SignatureName is the name of one of the [SettingsData.Signatures],
// chosen from them, or "" for none.
type SignatureName string

func (sn SignatureName) Widget() core.Value {
	ch := core.NewChooser()
	ch.Items = append(ch.Items, core.ChooserItem{Value: SignatureName(""), Text: "(none)"})
	for _, sig := range Settings.Signatures {
		ch.Items = append(ch.Items, core.ChooserItem{Value: SignatureName(sig.Name), Text: sig.Name})
	}
	return ch
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
// account, with a line break at the end of each line, or "" if it has none.
// No "-- " separator line is added: the signature can include one.
func signatureText(email string) string {
	return signatureNamed(Settings.DefaultSignatures[email])
}

// signatureNamed returns the text of the signature with the given name, as
// [signatureText], or "" for none.
func signatureNamed(sn SignatureName) string {
	name := string(sn)
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
		return strings.Join(lines, "  \n") + "\n"
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
	// the names of other servers, as iCloud
	"Sent Messages":    "Sent",
	"Sent Items":       "Sent",
	"Deleted Messages": "Trash",
	"Deleted Items":    "Trash",
}

// skipLabels are a temporary set of labels that should not be cached or displayed.
// TODO: figure out a better approach to this.
var skipLabels = map[string]bool{
	"[Gmail]":           true,
	"[Gmail]/All Mail":  true,
	"[Gmail]/Important": true,
}

// labelIcons are the icons for each friendly label name.
var labelIcons = map[string]icons.Icon{
	"Inbox":   icons.Inbox,
	"Drafts":  icons.Draft,
	"Outbox":  icons.Outbox,
	"Sent":    icons.Send,
	"Starred": icons.Star,
	"Spam":    icons.Report,
	"Junk":    icons.Report,
	"Trash":   icons.Delete,
}
