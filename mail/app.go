// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mail implements a GUI email client.
package mail

//go:generate core generate

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
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

	// unreadMessages is the number of unread messages for the current email account
	// and labels, used for displaying a count.
	unreadMessages int

	// readMessage is the current message we are reading.
	readMessage *CacheMessage

	// currentEmail is the current email account.
	currentEmail string

	// selectedMailbox is the currently selected mailbox for each email account in IMAP.
	selectedMailbox map[string]string

	// labels are all of the possible labels that messages can have in
	// each email account.
	labels map[string][]string

	// showLabel is the current label to show messages for.
	showLabel string
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
	a.showLabel = "INBOX"
	a.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})

	tree.AddChild(a, func(w *core.Splits) {
		w.SetSplits(0.1, 0.2, 0.7)
		tree.AddChild(w, func(w *core.Tree) {
			w.SetText("Accounts")
			w.Maker(func(p *tree.Plan) {
				for _, email := range Settings.Accounts {
					tree.AddAt(p, email, func(w *core.Tree) {
						a.makeLabelTree(w, email, "")
					})
				}
			})
		})
		tree.AddChild(w, func(w *core.Frame) {
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
			})
			w.Updater(func() {
				a.listCache = nil
				a.unreadMessages = 0
				mp := a.cache[a.currentEmail]
				for _, cm := range mp {
					if start := a.conversationStart(mp, cm); start != cm {
						if !slices.Contains(start.replies, cm) {
							start.replies = append(start.replies, cm)
						}
						continue
					}
					for _, label := range cm.Labels {
						if label.Name != a.showLabel {
							continue
						}
						a.listCache = append(a.listCache, cm)
						if !slices.Contains(cm.Flags, imap.FlagSeen) {
							a.unreadMessages++
						}
						break
					}
				}
				slices.SortFunc(a.listCache, func(a, b *CacheMessage) int {
					return cmp.Compare(b.latestDate().UnixNano(), a.latestDate().UnixNano())
				})
			})
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
					if len(a.listCache) < total {
						w.Text += fmt.Sprintf("%d of ", len(a.listCache))
					}
					w.Text += fmt.Sprintf("%d messages", total)
					if a.unreadMessages > 0 {
						w.Text += fmt.Sprintf(", %d unread", a.unreadMessages)
					}
				})
			})
			tree.AddChild(w, func(w *core.Separator) {})
			tree.AddChild(w, func(w *core.List) {
				w.SetSlice(&a.listCache)
				w.SetReadOnly(true)
			})
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

// makeLabelTree recursively adds a Maker to the given tree to form a nested tree of labels.
func (a *App) makeLabelTree(w *core.Tree, email, parentLabel string) {
	w.Maker(func(p *tree.Plan) {
		friendlyParentLabel := friendlyLabelName(parentLabel)
		for _, label := range a.labels[email] {
			if skipLabels[label] {
				continue
			}
			friendlyLabel := friendlyLabelName(label)
			// Skip labels that are not directly nested under the parent label.
			if parentLabel == "" && strings.Contains(friendlyLabel, "/") {
				continue
			} else if parentLabel != "" {
				if !strings.HasPrefix(friendlyLabel, friendlyParentLabel+"/") ||
					strings.Count(friendlyLabel, "/") > strings.Count(friendlyParentLabel, "/")+1 {
					continue
				}
			}
			tree.AddAt(p, label, func(w *core.Tree) {
				a.makeLabelTree(w, email, label)
				w.Updater(func() {
					// Recompute the friendly labels in case they have changed.
					w.SetText(strings.TrimPrefix(friendlyLabelName(label), friendlyLabelName(parentLabel)+"/"))
					if ic, ok := labelIcons[w.Text]; ok {
						w.SetIcon(ic)
					} else {
						w.SetIcon(icons.Label)
					}
				})
				w.OnSelect(func(e events.Event) {
					a.showLabel = label
					a.Update()
				})
			})
		}
	})
}

func (a *App) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(a.Compose).SetIcon(icons.Send).SetKey(keymap.New)
	})

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

func (a *App) GetMail() {
	go func() {
		err := a.auth()
		if err != nil {
			core.ErrorDialog(a, err, "Error authorizing")
			return
		}
		// We keep caching messages forever to stay in sync.
		for {
			err = a.cacheMessages()
			if err != nil {
				core.ErrorDialog(a, err, "Error caching messages")
			}
		}
	}()
}

// selectMailbox selects the given mailbox for the given email for the given client.
// It does nothing if the given mailbox is already selected.
func (a *App) selectMailbox(c *imapclient.Client, email string, mailbox string) error {
	if a.selectedMailbox[email] == mailbox {
		return nil // already selected
	}
	sd, err := c.Select(mailbox, nil).Wait()
	if err != nil {
		return fmt.Errorf("selecting mailbox: %w", err)
	}
	a.selectedMailbox[email] = mailbox
	a.totalMessages[email][mailbox] = int(sd.NumMessages)
	return nil
}

// cacheFilename returns the filename for the cached messages JSON file
// for the given email address.
func (a *App) cacheFilename(email string) string {
	return filepath.Join(core.TheApp.AppDataDir(), "caching", filenameBase32(email), "cached-messages.json")
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
