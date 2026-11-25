// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mail implements a GUI email client.
package mail

//go:generate core generate

import (
	"cmp"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"time"

	"cogentcore.org/cogent/mail/testserver"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-sasl"
	"golang.org/x/oauth2"
)

// App is an email client app.
type App struct {
	core.Frame

	// authToken contains the [oauth2.Token] for each account.
	authToken map[string]*oauth2.Token

	// authClient contains the [sasl.Client] authentication for sending messages for each account.
	authClient map[string]sasl.Client

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
	// unless [SettingsData.FlatList], for the current email account
	// and label, used for displaying a count.
	unreadMessages int

	// readMessage is the current message we are reading, or the start
	// of the conversation we are reading.
	readMessage *CacheMessage

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

	// syncInterval is the time between syncs of the messages.
	syncInterval time.Duration

	// syncMore is whether there are more messages to sync after the last
	// sync, which syncs a limited number of messages for each mailbox.
	syncMore bool

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
	a.authToken = map[string]*oauth2.Token{}
	a.authClient = map[string]sasl.Client{}
	a.imapClient = map[string]*imapclient.Client{}
	a.imapMu = map[string]*sync.Mutex{}
	a.cache = map[string]map[string]*CacheMessage{}
	a.totalMessages = map[string]map[string]int{}
	a.selectedMailbox = map[string]string{}
	a.labels = map[string][]string{}
	a.servers = map[string]Server{}
	a.trash = map[string]string{}
	a.syncNow = make(chan struct{}, 1)
	a.syncInterval = time.Minute
	a.showLabel = "INBOX"
	a.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})

	a.OnFirst(events.KeyChord, a.mailKeys)

	tree.AddChild(a, func(w *core.Splits) {
		a.splits = w
		w.SetSplits(0.15, 0.4, 0.45)
		tree.AddChild(w, a.initMailboxes)
		tree.AddChild(w, func(w *core.Frame) {
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
			})
			w.Updater(a.updateList)
			tree.AddChild(w, func(w *core.Text) {
				w.SetType(core.TextTitleMedium)
				w.Updater(func() {
					w.SetText(friendlyLabelName(a.showLabel))
				})
			})
			tree.AddChild(w, func(w *core.Text) {
				w.Updater(func() {
					w.Text = ""
					total := a.totalMessages[a.currentEmail][a.showLabel]
					if a.shownMessages < total { // still syncing
						w.Text += fmt.Sprintf("%d of ", a.shownMessages)
					}
					w.Text += fmt.Sprintf("%d messages", total)
					if !Settings.FlatList {
						w.Text += fmt.Sprintf(" in %d conversations", len(a.listCache))
					}
					if a.unreadMessages > 0 {
						w.Text += fmt.Sprintf(", %d unread", a.unreadMessages)
					}
				})
			})
			tree.AddChild(w, func(w *core.Separator) {})
			tree.AddChild(w, func(w *core.Switch) {
				w.SetText("Conversations").SetTooltip("Group the messages into conversations")
				w.Updater(func() {
					w.SetChecked(!Settings.FlatList)
				})
				w.OnChange(func(e events.Event) {
					Settings.FlatList = !w.IsChecked()
					errors.Log(core.SaveSettings(Settings))
					a.Update()
				})
			})
			tree.AddChild(w, a.initList)
		})
		tree.AddChild(w, func(w *core.Frame) {
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
			})
			w.Maker(func(p *tree.Plan) {
				if a.readMessage == nil {
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

// showMailbox shows the messages of the given mailbox of the given account.
func (a *App) showMailbox(email, mailbox string) {
	if email != a.currentEmail {
		a.currentEmail = email
		a.readMessage = nil
	}
	a.showLabel = mailbox
	a.Update()
}

func (a *App) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(a.Compose).SetIcon(icons.Send).SetKey(keymap.New)
	})
	if a.testServer != nil {
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.ReceiveTestMessage).SetIcon(icons.MoveToInbox)
		})
	}

	if a.readMessage != nil {
		tree.Add(p, func(w *core.Separator) {})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.Label).SetIcon(icons.DriveFileMove).SetKey(keymap.Save)
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.Delete).SetIcon(icons.Delete).SetKey(keymap.Delete)
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.Reply).SetIcon(icons.Reply).SetKey(keymap.Replace)
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.ReplyAll).SetIcon(icons.ReplyAll)
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.Forward).SetIcon(icons.Forward)
		})
		tree.Add(p, func(w *core.FuncButton) {
			w.SetFunc(a.MarkAsUnread).SetIcon(icons.MarkAsUnread)
		})
	}
}

// GetMail starts getting the mail of all of the accounts, signing in
// first if needed, and keeps it in sync with the servers.
func (a *App) GetMail() {
	go func() {
		if a.testServer == nil {
			err := a.auth()
			if err != nil {
				core.ErrorDialog(a, err, "Error authorizing")
				return
			}
		}
		// We keep caching messages forever to stay in sync.
		failed := false
		for {
			a.syncMore = false
			err := a.cacheMessages()
			if err != nil {
				slog.Error("caching messages", "err", err)
				if !failed { // only once until it works again
					core.ErrorDialog(a, err, "Error caching messages")
				}
				failed = true
			} else {
				failed = false
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

// cacheFilename returns the filename for the cached messages JSON file
// for the given email address.
func (a *App) cacheFilename(email string) string {
	return filepath.Join(a.dataDir(), "caching", filenameBase32(email), "cached-messages.json")
}

// saveCacheFile safely saves the given cache data for the
// given email account by going through a temporary file to
// avoid truncating it without writing it if we quit during the process.
func (a *App) saveCacheFile(cached map[string]*CacheMessage, email string) error {
	fname := a.cacheFilename(email)
	err := jsonx.Save(&cached, fname+".tmp")
	if err != nil {
		return fmt.Errorf("saving cache list: %w", err)
	}
	err = os.Rename(fname+".tmp", fname)
	if err != nil {
		return err
	}
	return nil
}

// conversationStart returns the first message in the conversation
// of the given message using [CacheMessage.InReplyTo] and the given
// cache map.
func (a *App) conversationStart(mp map[string]*CacheMessage, cm *CacheMessage) *CacheMessage {
	for {
		if len(cm.InReplyTo) == 0 {
			return cm
		}
		new, ok := mp[cm.InReplyTo[0]]
		if !ok {
			return cm
		}
		cm = new
	}
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
