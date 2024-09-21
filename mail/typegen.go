// Code generated by "core generate"; DO NOT EDIT.

package mail

import (
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
	"github.com/emersion/go-message/mail"
)

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/mail.App", IDName: "app", Doc: "App is an email client app.", Methods: []types.Method{{Name: "Label", Doc: "Label opens a dialog for changing the labels (mailboxes) of the current message.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "Reply", Doc: "Reply opens a dialog to reply to the current message.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "ReplyAll", Doc: "ReplyAll opens a dialog to reply to all people involved in the current message.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "Forward", Doc: "Forward opens a dialog to forward the current message to others.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "MarkAsRead", Doc: "MarkAsRead marks the current message as read.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "MarkAsUnread", Doc: "MarkAsUnread marks the current message as unread.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "Compose", Doc: "Compose opens a dialog to send a new message.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "Send", Doc: "Send sends the current message", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Returns: []string{"error"}}}, Embeds: []types.Field{{Name: "Frame"}}, Fields: []types.Field{{Name: "authToken", Doc: "authToken contains the [oauth2.Token] for each account."}, {Name: "authClient", Doc: "authClient contains the [sasl.Client] authentication for sending messages for each account."}, {Name: "imapClient", Doc: "imapClient contains the imap clients for each account."}, {Name: "imapMu", Doc: "imapMu contains the imap client mutexes for each account."}, {Name: "composeMessage", Doc: "composeMessage is the current message we are editing"}, {Name: "cache", Doc: "cache contains the cached message data, keyed by account and then MessageID."}, {Name: "listCache", Doc: "listCache is a sorted view of [App.cache] for the current email account\nand labels, used for displaying a [core.List] of messages. It should not\nbe used for any other purpose."}, {Name: "readMessage", Doc: "readMessage is the current message we are reading"}, {Name: "readMessageReferences", Doc: "readMessageReferences is the References header of the current readMessage."}, {Name: "readMessagePlain", Doc: "readMessagePlain is the plain text body of the current readMessage."}, {Name: "currentEmail", Doc: "currentEmail is the current email account."}, {Name: "selectedMailbox", Doc: "selectedMailbox is the currently selected mailbox for each email account in IMAP."}, {Name: "labels", Doc: "labels are all of the possible labels that messages have.\nThe first key is the account for which the labels are stored,\nand the second key is for each label name."}, {Name: "showLabel", Doc: "showLabel is the current label to show messages for."}}})

// NewApp returns a new [App] with the given optional parent:
// App is an email client app.
func NewApp(parent ...tree.Node) *App { return tree.New[App](parent...) }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/mail.SettingsData", IDName: "settings-data", Doc: "SettingsData is the data type for the global Cogent Mail settings.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Embeds: []types.Field{{Name: "SettingsBase"}}, Fields: []types.Field{{Name: "Accounts", Doc: "Accounts are the email accounts the user is signed into."}}})

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/mail.MessageListItem", IDName: "message-list-item", Doc: "MessageListItem represents a [CacheMessage] with a [core.Frame] for the message list.", Embeds: []types.Field{{Name: "Frame"}}, Fields: []types.Field{{Name: "Data"}}})

// NewMessageListItem returns a new [MessageListItem] with the given optional parent:
// MessageListItem represents a [CacheMessage] with a [core.Frame] for the message list.
func NewMessageListItem(parent ...tree.Node) *MessageListItem {
	return tree.New[MessageListItem](parent...)
}

// SetData sets the [MessageListItem.Data]
func (t *MessageListItem) SetData(v *CacheMessage) *MessageListItem { t.Message = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/mail.AddressTextField", IDName: "address-text-field", Doc: "AddressTextField represents a [mail.Address] with a [core.TextField].", Embeds: []types.Field{{Name: "TextField"}}, Fields: []types.Field{{Name: "Address"}}})

// NewAddressTextField returns a new [AddressTextField] with the given optional parent:
// AddressTextField represents a [mail.Address] with a [core.TextField].
func NewAddressTextField(parent ...tree.Node) *AddressTextField {
	return tree.New[AddressTextField](parent...)
}

// SetAddress sets the [AddressTextField.Address]
func (t *AddressTextField) SetAddress(v mail.Address) *AddressTextField { t.Address = v; return t }
