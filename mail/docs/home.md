+++
URL = ""
Title = "Cogent Mail"
+++

<img src="icon.svg" alt="Cogent Mail icon" width="128">

**Cogent Mail** is an email app for macOS, Windows, and Linux, built with Cogent Core. It reads and sends the mail of several accounts at once: Gmail, Microsoft 365 and Outlook.com, iCloud, and any other IMAP account. Your mail is kept on your computer as well as on the servers, so you can read, search, and write messages while you are offline; what you do offline is done on the servers once you are connected again. Messages are written in [markdown](https://www.markdownguide.org/basic-syntax/), and are sent as both plain text and formatted HTML.

## Getting started

The first time you start Cogent Mail, it asks you to add an account. You can add more accounts later with **Add account** in the context menu of **Accounts**, in the mailboxes on the left. The kind of the account determines how you sign in:

- **Gmail**: you sign in to Google in your browser.
- **Microsoft 365**: a work or school account, or a personal Outlook.com, Hotmail, or Live account. You sign in in your browser, as with the single sign-on of your organization. Your organization must allow IMAP, and it may need to approve the app.
- **iCloud**: you sign in with an *app-specific password*, which you make at [account.apple.com](https://account.apple.com), under Sign-In and Security.
- **IMAP**: any other account, with the addresses of its IMAP and SMTP servers (as `imap.example.com:993` and `smtp.example.com:587`), and its password.

Cogent Mail signs in to Gmail with its own client registered with Google, which is built into the app, also when you build it from the source code. To use a client of your own instead, set the environment variables `GOOGLE_OAUTH2_CLIENT_ID` and `GOOGLE_OAUTH2_CLIENT_SECRET`. Microsoft accounts need the client ID of an app registered with Microsoft, in `MICROSOFT_OAUTH2_CLIENT_ID`.

Passwords are kept in the keychain of your system, not in the settings. If a password changes, use **Change password** in the context menu of the account; if signing in expires, use **Sign in again**. **Remove account** removes the account and its mail from Cogent Mail (not from the server).

## Layout

The window has three panels, side by side, which you can resize by dragging the dividers between them:

- **Mailboxes** (left): **Inboxes** at the top, with the inbox of each account, and then each account with its mailboxes (folders). The numbers are the unread messages. Below them, the status shows what is being synced.
- **Messages** (middle): the messages of the mailbox, with its name and the numbers of its messages above them, and a search field.
- **Message** (right): the message or conversation you are reading.

Each Gmail account also has **Untagged**: the messages that no other mailbox shows, without any label (other than Important), not sent by you, and not starred, as those archived in Gmail, and those whose only label was deleted, which would otherwise not be shown, as Gmail keeps them in All Mail. File a message out of it to give it a label.

Click **Inboxes** to see the inboxes of all of your accounts together, the newest first. Replying to, filing, or deleting a message there acts in its own account.

When you show another mailbox, its first message is selected and shown (it is marked as read only when you click it).

### The list of messages

By default, the messages are grouped into *conversations*: a message and its replies, which you can open to see each of them. The Sent, Drafts, Outbox, and Trash mailboxes always show each message on its own. Click the column headers to sort the list.

The view button (the tune icon) above the list sets what it shows:

- **Conversations**: groups the messages into conversations.
- **Compact**: one column, with the sender, subject, and date of each message on two lines, as on a narrow screen.
- **Labels**: the other mailboxes (labels) that each message is in.
- **Attachments**: the number of attachments of each message.
- **Stars**: the stars of the starred messages.

The filter button, before the refresh button, shows only the unread messages (or conversations with any), and shows all of them again when clicked again. The message you read stays in the list until you read another one. The refresh button checks for new messages in the mailbox now. Inboxes are also checked every 15 seconds, and right away when the server reports new mail; the other mailboxes are synced every minute.

You can select several messages with `Shift` or `Command` and a click, and then file, tag, star, delete, or mark them all at once. The first of them is shown in the message panel. Double-click a message to open it in a window of its own, so that you can see several messages at the same time; its toolbar, at the bottom, has **Reply**, **Reply all**, and **Forward**. Double-clicking a draft opens it for writing it more.

### Reading messages

Messages are shown as they are formatted, with their images, including those attached to the message. Click a link to open it in your browser. Click an address in the header, or a `mailto:` link, to write to it.

Attachments are shown as buttons in the header: click one to open it, or right-click it to save it. Drag an attachment, or an image in a message, to a message you are writing to attach it there, or to the Finder, the desktop, or another app to save it there.

The **Date** line of the header also has the **Size** of the message, with its attachments. The labels in the header of a message are the other mailboxes it is in; click one to go to that mailbox.

## Actions on messages

The context menu of a message (and the toolbar) has these actions, which apply to all of the selected messages:

- **Reply** (`Command+R`), **Reply all** (`Command+W` or `Command+Shift+R`), and **Forward** (`Command+F` or `Command+Shift+F`). A forward includes the attachments.
- **File** (`Control+O`): moves the messages to another mailbox, out of the one shown. You can file into a mailbox of another account, which copies the messages there. The **FilingAccount** setting puts the mailboxes of one account first, as for filing all of your mail into the folders of one account. The dialog starts on the mailbox you filed into last, so that you can file more there with its **File** button (or `Return`).
- **Tag** (`Control+T`): adds another label (mailbox) to the messages, which stay in the mailbox shown. On Gmail, mailboxes are labels, and a message can be in several of them.
- **Star** (`Command+Shift+L`): stars the messages, or unstars them if they are all starred. The **Starred** mailbox has all of the starred messages.
- **Delete** (`Delete`, `Backspace`, or `Control+D`): moves the messages to the trash. The next message is selected.
- **Mark as read** and **Mark as unread** (`Command+Shift+U`).
- **Remove attachments** (`Command+X`): removes the attachments of the messages, to save space, both here and on the server, replacing each with a note of its name and size. The attachments cannot be recovered, so it asks first.

The context menu of a mailbox has **Mark all as read**, and for your own mailboxes (folders and labels) **New folder**, **Rename folder**, and **Delete folder**. The standard mailboxes of the server (Inbox, Drafts, Sent, Spam, Trash, and Starred) cannot be renamed or deleted, but other mailboxes with the same names can, such as the "Sent Messages" and "Deleted Messages" labels that other mail apps make on Gmail, which are shown with their own names. The context menu of an account has **Sync now**.

## Search

Type in the search field above the list (`Command+Alt+F`) to search all of the messages of the account, in all of its mailboxes, or of all of your accounts when Inboxes is shown. The messages found are listed on their own, the newest first. A search finds the messages with all of its words. You can also use:

- `"a phrase"`: the words together, in that order.
- `from:`, `to:`, `cc:`, `subject:`, `body:`, and `attachment:`: a word or phrase in that part of the message, as `from:alice` or `subject:"project plan"`.
- `-word`: messages without the word.

Press `Escape`, or click the clear button, to show the mailbox again.

## Writing messages

Click **Compose** in the toolbar (`Command+N`) to write a new message. The window has the recipients (**To**, **Cc**, and **Bcc**, with **+** buttons for more), the **Subject**, the **Attachments**, and the **From** line, with the account the message is sent from and its **Signature**.

- **Addresses** are completed as you type, from your contacts first and then from the people in your mail. You can paste `mailto:` links and lists of addresses.
- **Formatting**: the message is markdown, so `*italic*`, `**bold**`, lists, and links are formatted when it is sent. Lines starting with `>` are quoted. The buttons at the end of the From line quote the selected lines one more level (`Command+'`) or one less (`Command+Alt+'`).
- **Replies** quote the message replied to, below the signature. In a reply to a `mailto:` link, **Reply all** adds everyone in the message. The **Attachments** switch includes the attachments of the message replied to.
- **Attachments**: drop files on the window, from the Finder or from other messages, to attach them.
- **Drafts**: the message is saved in Drafts as you write. **Save draft** closes the window and keeps it there; **Discard** deletes it. To keep writing a draft, double-click it in Drafts, or select it and choose **Edit draft**. This also works for drafts written elsewhere, as in Gmail on the web or on your phone: once you save, send, or discard it here, the draft on the server is moved to the trash, as the one here replaces it. Closing a new message asks whether to save it.
- **Send** (`Command+Return`): the message goes to the **Outbox** first, and is sent from there. If it cannot be sent, as when you are offline, it waits there and is sent once you are connected; if the server refuses it, it stays there with the reason, and you can **Edit draft** or **Send again**.

Changing the From account changes the signature to its default one, and the automatic Bcc.

## Contacts

Click **Contacts** in the toolbar to look up and edit your contacts. Their names and addresses come first when completing addresses and when showing who sent a message. You can search them, add new ones, and edit their names, organizations, email addresses, phone numbers, postal addresses, birthdays, and notes.

Right-click your own contact and choose **Make this my card**. Your name in it is then the name in the mail you send from any of your addresses, and all of its addresses count as yours (as when replying to all, which leaves you out).

To sync the contacts of a Gmail account with Google Contacts, choose **Sync contacts** in the context menu of the account. The contacts you add in Cogent Mail are added to the Google Contacts of the **ContactsAccount** setting (by default, the first account that syncs its contacts). The contacts are kept as vCards, and are also used by the Cogent Contacts app.

## Keyboard shortcuts

These are the default shortcuts of the actions of the main window:

- `Command+N`: compose a new message.
- `Command+R`: reply. `Command+W` or `Command+Shift+R`: reply all. `Command+F` or `Command+Shift+F`: forward.
- `Control+O`: file. `Control+T`: tag. `Command+Shift+L`: star.
- `Delete`, `Backspace`, or `Control+D`: delete.
- `Command+Shift+U`: mark as unread. `Command+X`: remove attachments.
- `Command+Shift+N`: check for new messages now.
- `Command+Alt+F`: search.
- `Control+Tab` and `Control+Shift+Tab` (or `Control+X o` and `Control+X p` with the Emacs key map): move the keyboard focus to the next or previous panel: the mailboxes, the list of messages, and the message being read. In the message, the arrow keys, `PageUp`, `PageDown`, `Home`, and `End` scroll it.

In the list of messages, the arrow keys move the selection. While you type in the search field, the keys without `Command` edit the text.

To change the shortcuts, open the settings and click **Keyboard shortcuts**: click a shortcut to record another key, choose the action it does, and use the context menu of a row to add or delete one. **Defaults** resets all of them.

`Command` is the Command key on macOS and the Control key on other platforms.

## Settings

To open the settings, click the settings button (the gear) in the toolbar. The Mail settings are:

- **Accounts**: the accounts, with their kinds and, for IMAP accounts, their servers.
- **ListViews**: what the list of messages shows (as with its view button).
- **FilingAccount**: the account whose mailboxes are offered first for filing.
- **ContactsAccount**: the account whose Google Contacts the new contacts are added to.
- **AutoBCC**: sends a blind copy of every message to its sender, whose address is put in the Bcc of each new message, where you can remove it for that message.
- **Signatures**: your signatures, each with a name and its text, in markdown.
- **DefaultSignatures**: the signature of the new messages of each account.
- **Keys**: the keyboard shortcuts.

The settings are saved in the `settings.toml` file in the Cogent Mail folder of your app data folder (`~/Library/Cogent Mail` on macOS), with your mail, its search index, and the logs. If the app stops unexpectedly, the details are saved in its `crash-logs` folder.
