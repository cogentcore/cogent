// Code generated by "core generate"; DO NOT EDIT.

package mail

import (
	"net/mail"

	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/mail.App", IDName: "app", Doc: "App is an email client app.", Methods: []types.Method{{Name: "Move", Doc: "Move moves the current message to the given mailbox.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Args: []string{"mailbox"}}, {Name: "Reply", Doc: "Reply opens a dialog to reply to the current message.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "ReplyAll", Doc: "ReplyAll opens a dialog to reply to all people involved in the current message.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "Forward", Doc: "Forward opens a dialog to forward the current message to others.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "MarkAsRead", Doc: "MarkAsRead marks the current message as read.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "MarkAsUnread", Doc: "MarkAsUnread marks the current message as unread.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "Compose", Doc: "Compose opens a dialog to send a new message.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}}, {Name: "Send", Doc: "Send sends the current message", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Returns: []string{"error"}}}, Embeds: []types.Field{{Name: "Frame"}}, Fields: []types.Field{{Name: "authToken", Doc: "authToken contains the [oauth2.Token] for each account."}, {Name: "authClient", Doc: "authClient contains the [sasl.Client] authentication for sending messages for each account."}, {Name: "imapClient", Doc: "imapClient contains the imap clients for each account."}, {Name: "imapMu", Doc: "imapMu contains the imap client mutexes for each account."}, {Name: "composeMessage", Doc: "composeMessage is the current message we are editing"}, {Name: "cache", Doc: "cache contains the cache data, keyed by account and then mailbox."}, {Name: "currentCache", Doc: "currentCache is [App.cache] for the current email account and mailbox."}, {Name: "readMessage", Doc: "readMessage is the current message we are reading"}, {Name: "readMessageReferences", Doc: "readMessageReferences is the References header of the current readMessage."}, {Name: "readMessagePlain", Doc: "readMessagePlain is the plain text body of the current readMessage."}, {Name: "currentEmail", Doc: "The current email account"}, {Name: "currentMailbox", Doc: "The current mailbox"}}})

// NewApp returns a new [App] with the given optional parent:
// App is an email client app.
func NewApp(parent ...tree.Node) *App { return tree.New[App](parent...) }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/mail.SettingsData", IDName: "settings-data", Doc: "SettingsData is the data type for the global Cogent Mail settings.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Embeds: []types.Field{{Name: "SettingsBase"}}, Fields: []types.Field{{Name: "Accounts", Doc: "Accounts are the email accounts the user is signed into."}}})

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/mail.MessageListItem", IDName: "message-list-item", Doc: "MessageListItem represents a [CacheData] with a [core.Frame] for the message list.", Embeds: []types.Field{{Name: "Frame"}}, Fields: []types.Field{{Name: "Data"}}})

// NewMessageListItem returns a new [MessageListItem] with the given optional parent:
// MessageListItem represents a [CacheData] with a [core.Frame] for the message list.
func NewMessageListItem(parent ...tree.Node) *MessageListItem {
	return tree.New[MessageListItem](parent...)
}

// SetData sets the [MessageListItem.Data]
func (t *MessageListItem) SetData(v *CacheData) *MessageListItem { t.Data = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/mail.AddressTextField", IDName: "address-text-field", Doc: "AddressTextField represents a [mail.Address] with a [core.TextField].", Embeds: []types.Field{{Name: "TextField"}}, Fields: []types.Field{{Name: "Address"}}})

// NewAddressTextField returns a new [AddressTextField] with the given optional parent:
// AddressTextField represents a [mail.Address] with a [core.TextField].
func NewAddressTextField(parent ...tree.Node) *AddressTextField {
	return tree.New[AddressTextField](parent...)
}

// SetAddress sets the [AddressTextField.Address]
func (t *AddressTextField) SetAddress(v mail.Address) *AddressTextField { t.Address = v; return t }
