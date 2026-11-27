// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mail implements a GUI email client.
package mail

//go:generate core generate

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"time"

	"cogentcore.org/cogent/mail/contacts"
	"cogentcore.org/cogent/mail/testserver"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/tree"
	"github.com/cogentcore/go-imap/v2"
	"github.com/cogentcore/go-imap/v2/imapclient"
	"golang.org/x/oauth2"
)

// App is an email client app.
type App struct {
	core.Frame

	// tokens are the sources of the OAuth tokens of the accounts that
	// are signed in, which keep the access tokens fresh, under authMu.
	tokens map[string]oauth2.TokenSource

	// authErrs are the errors of signing in to the accounts with their
	// saved tokens that could not be checked, as when offline, under
	// authMu; they are tried again in each sync.
	authErrs map[string]error

	// authMu is the mutex of tokens and authErrs, which the sync uses.
	authMu sync.Mutex

	// signInShown are the accounts with a dialog for signing in to them
	// open, which is not opened again for them (see [App.offerSignIn]),
	// under the lock of the window.
	signInShown map[string]bool

	// passwords are the passwords of the accounts that sign in with one,
	// from the keychain, under authMu (see [App.password]).
	passwords map[string]string

	// started is whether the app has signed in to the accounts at its
	// start, after which those added in the settings are signed in to
	// (see [App.accountsAdded]), and knownAccounts are the accounts then,
	// with those added since, under the lock of the window.
	started       bool
	knownAccounts map[string]bool

	// drafts are the messages being written and those in the outbox,
	// waiting to be sent, by their IDs, under draftsMu (see [draft]).
	drafts map[string]*draft

	// draftsMu is the mutex of drafts, and of sending them.
	draftsMu sync.Mutex

	// imapClient contains the imap clients for each account.
	imapClient map[string]*imapclient.Client

	// imapMu contains the imap client mutexes for each account.
	imapMu map[string]*sync.Mutex

	// composeMessage is the current message we are editing
	composeMessage *SendMessage

	// cache contains the cached message data, keyed by account and then MessageID.
	cache map[string]map[string]*CacheMessage

	// listCache is a sorted view of [App.cache] for the current email account
	// and labels, used for displaying a [core.List] of messages. It should not
	// be used for any other purpose.
	listCache []*CacheMessage

	// totalMessages is the total number of messages for each email account and label.
	totalMessages map[string]map[string]int

	// shownMessages is the number of messages with the current label
	// in the list of messages.
	shownMessages int

	// unreadMessages is the number of unread messages, or conversations
	// unless [App.flatList], for the current email account
	// and label, used for displaying a count.
	unreadMessages int

	// readMessage is the current message we are reading, or the start
	// of the conversation we are reading.
	readMessage *CacheMessage

	// searchText is the text of the current search, if any, and
	// searchResults are the keys of the messages it found (see
	// [searchKey]), or nil
	// if there is no search (see [App.SearchMessages]).
	searchText    string
	searchResults map[string]bool

	// searchField is the search field of the list of messages.
	searchField *core.TextField

	// selectedRows are the selected rows of the list of messages,
	// for the actions on all of them (see [App.readMessages]).
	selectedRows []*messageRow

	// readSingle is whether we are reading just [App.readMessage],
	// instead of its whole conversation.
	readSingle bool

	// listRows are the rows of the list of messages.
	listRows []*messageRow

	// messageRows and conversationRows are the rows of the list of
	// messages for each message and conversation start, which are
	// reused for the same ones.
	messageRows, conversationRows map[*CacheMessage]*messageRow

	// list is the tree table of the list of messages.
	list *core.TreeTable

	// splits are the splits of the panels.
	splits *core.Splits

	// keySeq1 is the first key of a key sequence being entered,
	// such as Control+X (see [App.mailKeys]).
	keySeq1 key.Chord

	// mailboxes is the tree of the accounts and their mailboxes.
	mailboxes *core.Tree

	// unread is the number of unread messages in each mailbox of each account.
	unread map[string]map[string]int

	// currentEmail is the current email account.
	currentEmail string

	// selectedMailbox is the currently selected mailbox for each email account in IMAP.
	selectedMailbox map[string]string

	// labels are all of the possible labels that messages can have in
	// each email account.
	labels map[string][]string

	// showLabel is the current label to show messages for.
	showLabel string

	// messageWindows are the windows of messages open (see
	// [App.openMessageWindow]), under the lock of the main window.
	messageWindows []*core.Body

	// allInboxes is whether the list of messages shows the inboxes of all
	// of the accounts together, with showLabel "INBOX", in which case
	// currentEmail is that of the message being read, which the actions on
	// it use (see [App.ShowAllInboxes]).
	allInboxes bool

	// servers are the server configurations of the accounts that are not
	// Gmail, by email address.
	servers map[string]Server

	// trash is the trash mailbox of each account.
	trash map[string]string

	// dataDirectory, if set, is the directory of the data of the app
	// instead of the standard one; see [App.dataDir].
	dataDirectory string

	// syncNow is sent to to sync the messages now instead of after
	// [App.syncInterval].
	syncNow chan struct{}

	// syncInbox is sent to to sync the inboxes now instead of after
	// [inboxSyncInterval] (see [App.quickSyncLoop]).
	syncInbox chan struct{}

	// syncInterval is the time between syncs of the messages.
	syncInterval time.Duration

	// syncMore is whether there are more messages to sync after the last
	// sync, which syncs a limited number of messages for each mailbox.
	syncMore bool

	// stores and searches are the store and search index of each account.
	stores   map[string]*messageStore
	searches map[string]*searchIndex

	// gmail and condStore are whether the server of each account has
	// the Gmail extension and CONDSTORE.
	gmail, condStore map[string]bool

	// specialUse are the special-use mailboxes of each account,
	// such as the Sent one, by their attributes.
	specialUse map[string]map[imap.MailboxAttr]string

	// downloading are the messages whose contents are being downloaded
	// to be read (see [App.downloadNow]).
	downloading map[*CacheMessage]bool

	// idleStop stops the connection that idles for each account
	// (see [App.idle]).
	idleStop map[string]context.CancelFunc

	// syncUpdatePending is whether an update of the app for the changes
	// from the sync is pending, under syncUpdateMu (see [App.syncUpdate]).
	syncUpdatePending bool
	syncUpdateMu      sync.Mutex

	// status is the status of the syncing, shown below the mailboxes
	// (see [App.setStatus]).
	status syncStatus

	// statusBar shows the status.
	statusBar *core.Frame

	// pendingLabels are the labels of the messages whose labels are being
	// changed on the server, as when they are filed or deleted, which
	// they are shown with until then, so that they leave the mailbox
	// right away (see [App.setLabelsOn]). It is used with the lock of the
	// window.
	pendingLabels map[*CacheMessage][]string

	// contacts is the store of the contacts, opened when first used
	// (see [App.contactsStore]).
	contacts *contacts.Store

	// mailCandidates are the candidates for completing addresses from the
	// mail, found at candidatesTime (see [App.addressCandidates]).
	mailCandidates []contacts.Candidate
	candidatesTime time.Time

	// senderNames are the names in the most messages sent from each account
	// (see [App.senderName]).
	senderNames map[string]string

	// contactsSynced is when the contacts of each account were last synced
	// (see [App.syncContacts]).
	contactsSynced map[string]time.Time

	// contactsServer, if set, is the CardDAV server of the contacts
	// instead of Google, in tests.
	contactsServer string

	// readPanel is the panel of the message being read, and toolbar is
	// the toolbar of the app, which are updated on their own when the
	// message being read changes (see [App.updateReading]).
	readPanel *core.Frame
	toolbar   *core.Toolbar

	// readShown is the message last shown in the read panel, for
	// scrolling it to the top when another one is shown.
	readShown *CacheMessage

	// sidebar is the panel of the mailboxes and the status, and listPanel
	// is that of the list of messages (see [App.updateMail]).
	sidebar, listPanel *core.Frame

	// indexChecked are the keys of the downloaded messages of each account
	// checked to be in its search index in this run (see [App.indexMissing]).
	indexChecked map[string]map[string]bool

	// testServer is the local simulated mail server in testing mode.
	testServer *testserver.Server
}

// needed for interface import
var _ tree.Node = (*App)(nil)

// theApp is the current app instance.
// TODO: ideally we could remove this.
var theApp *App

func (a *App) Init() {
	theApp = a
	a.Frame.Init()
	a.tokens = map[string]oauth2.TokenSource{}
	a.authErrs = map[string]error{}
	a.signInShown = map[string]bool{}
	a.passwords = map[string]string{}
	a.drafts = map[string]*draft{}
	a.imapClient = map[string]*imapclient.Client{}
	a.imapMu = map[string]*sync.Mutex{}
	a.cache = map[string]map[string]*CacheMessage{}
	a.totalMessages = map[string]map[string]int{}
	a.selectedMailbox = map[string]string{}
	a.labels = map[string][]string{}
	a.servers = map[string]Server{}
	a.stores = map[string]*messageStore{}
	a.searches = map[string]*searchIndex{}
	a.gmail = map[string]bool{}
	a.condStore = map[string]bool{}
	a.specialUse = map[string]map[imap.MailboxAttr]string{}
	a.downloading = map[*CacheMessage]bool{}
	a.indexChecked = map[string]map[string]bool{}
	a.idleStop = map[string]context.CancelFunc{}
	a.contactsSynced = map[string]time.Time{}
	a.status.progress = -1
	a.trash = map[string]string{}
	a.syncNow = make(chan struct{}, 1)
	a.syncInbox = make(chan struct{}, 1)
	a.syncInterval = time.Minute
	a.showLabel = "INBOX"
	if Settings.hasAccount(Settings.ShownAccount) && Settings.ShownMailbox != "" {
		a.currentEmail, a.showLabel = Settings.ShownAccount, Settings.ShownMailbox // as last shown
	} else if Settings.ShownAccount == allInboxesShown && len(Settings.Accounts) > 0 {
		a.allInboxes = true
	}
	a.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})

	a.OnFirst(events.KeyChord, a.mailKeys)

	tree.AddChild(a, func(w *core.Splits) {
		a.splits = w
		w.SetSplits(0.15, 0.4, 0.45)
		if len(Settings.Splits) == 3 {
			w.SetSplits(Settings.Splits...)
		}
		w.OnChange(func(e events.Event) {
			Settings.Splits = w.Splits()
			a.saveSettings()
		})
		tree.AddChild(w, func(w *core.Frame) { // the sidebar
			a.sidebar = w
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
			})
			tree.AddChild(w, func(w *core.Frame) {
				w.Styler(func(s *styles.Style) {
					s.Direction = styles.Column
					s.Grow.Set(1, 1)
					s.Overflow.Set(styles.OverflowAuto)
				})
				tree.AddChild(w, a.initMailboxes)
			})
			tree.AddChild(w, func(w *core.Separator) {}) // above the status
			tree.AddChild(w, a.initStatusBar)
		})
		tree.AddChild(w, func(w *core.Frame) {
			a.listPanel = w
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
			})
			w.Updater(a.updateList)
			w.Maker(a.makeListPanel)
		})
		tree.AddChild(w, func(w *core.Frame) {
			a.readPanel = w
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
				// it takes the focus, to scroll the message with the keys
				s.SetAbilities(true, abilities.Focusable)
			})
			w.Updater(func() {
				if a.readMessage != a.readShown { // a new message: from its top
					a.readShown = a.readMessage
					w.Defer(func() { w.ScrollDimToContentStart(math32.Y) })
				}
			})
			w.Maker(func(p *tree.Plan) {
				if a.readMessage == nil {
					return
				}
				if n := len(a.selectedRows); n > 1 {
					tree.Add(p, func(w *core.Text) {
						w.SetType(core.TextHeadlineSmall)
						w.Updater(func() {
							w.SetText(fmt.Sprintf("%d selected", len(a.selectedRows)))
						})
					})
					tree.Add(p, func(w *core.Text) {
						w.SetType(core.TextSupporting).SetText("File, tag, delete, and mark them all at once.")
					})
					return
				}
				add := func(cm *CacheMessage) {
					tree.AddAt(p, cm.Filename, func(w *DisplayMessageFrame) {
						w.Updater(func() {
							w.SetMessage(cm)
						})
					})
				}
				if a.readSingle {
					add(a.readMessage)
					return
				}
				slices.SortFunc(a.readMessage.replies, func(a, b *CacheMessage) int {
					return cmp.Compare(b.Date.UnixNano(), a.Date.UnixNano())
				})
				for i, reply := range a.readMessage.replies {
					add(reply)
					tree.AddAt(p, "separator"+strconv.Itoa(i), func(w *core.Separator) {})
				}
				add(a.readMessage)
			})
		})
	})
}

// saveSettings saves the [Settings], showing any error.
func (a *App) saveSettings() {
	errors.Log(os.MkdirAll(filepath.Dir(Settings.Filename()), 0o700))
	core.ErrorSnackbar(a, core.SaveSettings(Settings), "Error saving settings")
}

// showMailbox shows the messages of the given mailbox of the given account.
func (a *App) showMailbox(email, mailbox string) {
	if a.allInboxes || email != a.currentEmail || mailbox != a.showLabel {
		a.selectedRows = nil
		a.searchResults = nil // back to the mailbox
		if a.searchField != nil {
			a.searchField.SetText("")
		}
	}
	a.offerSignIn(email) // when clicked, if it is not signed in
	if email != a.currentEmail {
		a.currentEmail = email
		a.readMessage = nil
	}
	a.allInboxes = false
	a.showLabel = mailbox
	if Settings.ShownAccount != email || Settings.ShownMailbox != mailbox {
		Settings.ShownAccount, Settings.ShownMailbox = email, mailbox
		a.saveSettings() // shown when the app starts again
	}
	a.Update()
}

func (a *App) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.FuncButton) {
		a.toolbar, _ = w.Parent.(*core.Toolbar)
		w.SetFunc(core.SettingsWindow).SetText("").SetIcon(icons.Settings)
		w.SetTooltip("Settings")
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(a.Contacts).SetIcon(icons.Contacts)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(a.Compose).SetIcon(icons.Send).SetShortcut(shortcutOf("Compose"))
	})
	if a.testServer != nil {
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.ReceiveTestMessage).SetIcon(icons.MoveToInbox)
		})
	}

	if a.readMessage != nil {
		tree.Add(p, func(w *core.Separator) {})
		if a.readMessage.isLocal() && len(a.selectedRows) <= 1 {
			a.draftButtons(p)
		} else {
			a.messageButtons(p, true)
		}
	}
}

// GetMail starts getting the mail of all of the accounts, signing in
// first if needed, and keeps it in sync with the servers.
func (a *App) GetMail() {
	core.TheApp.AddQuitCleanFunc(a.closeStores)
	go func() {
		a.openAllLocal() // shown before connecting, as when offline
		if a.testServer == nil {
			need := a.signInSaved()
			a.AsyncLock()
			if len(Settings.Accounts) == 0 {
				a.AddAccount()
			} else if len(need) > 0 {
				a.signInDialog(need)
			}
			a.started = true // accounts added in the settings are signed in to now
			a.accountsAdded(Settings.Emails())
			a.AsyncUnlock()
		}
		go a.quickSyncLoop()
		// We keep caching messages forever to stay in sync.
		for {
			a.syncMore = false
			err := a.cacheMessages()
			if err != nil {
				slog.Error("caching messages", "err", err)
			}
			a.syncDone(err) // the error is shown in the status
			for _, email := range Settings.Emails() {
				if a.imapClient[email] != nil { // connected, so it can idle
					a.startIdle(email)
				}
			}
			if a.syncMore && err == nil {
				continue
			}
			select {
			case <-a.syncNow:
			case <-time.After(a.syncInterval):
			}
		}
	}()
}

// SyncNow syncs the messages now instead of at the next regular sync.
func (a *App) SyncNow() {
	select {
	case a.syncNow <- struct{}{}:
	default: // already pending
	}
}

// selectMailbox selects the given mailbox for the given email for the given client.
// It does nothing if the given mailbox is already selected. It must be called
// with the IMAP mutex of the account held, and not the lock of the window.
func (a *App) selectMailbox(c *imapclient.Client, email string, mailbox string) error {
	if a.selectedMailbox[email] == mailbox {
		return nil // already selected
	}
	sd, err := c.Select(mailbox, nil).Wait()
	if err != nil {
		return fmt.Errorf("selecting mailbox: %w", err)
	}
	a.selectedMailbox[email] = mailbox
	// with the lock of the window, which shows the number of messages
	a.AsyncLock()
	a.totalMessages[email][mailbox] = int(sd.NumMessages)
	a.AsyncUnlock()
	return nil
}

// conversationStart returns the first message in the conversation
// of the given message using [CacheMessage.InReplyTo] and the given
// messages by their Message-IDs.
func conversationStart(byID map[string]*CacheMessage, cm *CacheMessage) *CacheMessage {
	seen := map[*CacheMessage]bool{}
	for !seen[cm] {
		seen[cm] = true
		if len(cm.InReplyTo) == 0 {
			return cm
		}
		new, ok := byID[cm.InReplyTo[0]]
		if !ok {
			return cm
		}
		cm = new
	}
	return cm // a loop of replies
}

// latestDate returns the latest date/time of the message and all of its replies.
func (cm *CacheMessage) latestDate() time.Time {
	res := cm.Date
	for _, reply := range cm.replies {
		if reply.Date.After(res) {
			res = reply.Date
		}
	}
	return res
}

// isRead returns whether the message and all of its replies are marked as read.
func (cm *CacheMessage) isRead() bool {
	if !slices.Contains(cm.Flags, imap.FlagSeen) {
		return false
	}
	for _, reply := range cm.replies {
		if !slices.Contains(reply.Flags, imap.FlagSeen) {
			return false
		}
	}
	return true
}
