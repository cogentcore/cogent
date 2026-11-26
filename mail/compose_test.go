// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/textcore"
	gomail "github.com/emersion/go-message/mail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComposeTabAndSender tests the compose window: Tab goes from To to the
// Subject, skipping Cc, Bcc, Attachments, and From, which is a chooser of
// the accounts, last.
func TestComposeTabAndSender(t *testing.T) {
	b, a := newTestApp(t, nil)
	Settings.Accounts = []string{testEmail, "other@example.com"}
	var cb *core.Body
	core.RenderImage(b, core.Locked(b, func() {
		a.composeMessage = &SendMessage{To: []*gomail.Address{{}}}
		cb = a.compose("Compose")
	}), core.Locked(b, func() {
		core.Locked(cb, func() {
			var fields []string
			var to, subject core.Widget
			var from *core.Chooser
			for w := range cb.AllWidgets() {
				name := w.AsTree().Name
				if strings.HasPrefix(name, "value-") {
					fields = append(fields, strings.Split(name, "-")[1])
				}
				switch {
				case strings.HasPrefix(name, "value-To-"):
					for cw := range w.AsWidget().AllWidgets() {
						if _, ok := cw.(*AddressTextField); ok && to == nil {
							to = cw
						}
					}
				case strings.HasPrefix(name, "value-Subject-"):
					subject = w
				case strings.HasPrefix(name, "value-From-"):
					from, _ = w.(*core.Chooser)
				}
			}
			assert.Equal(t, "From", fields[len(fields)-1], "last")
			require.NotNil(t, from, "a chooser")
			assert.Len(t, from.Items, 2, "of the accounts")
			require.NotNil(t, to)
			require.NotNil(t, subject)
			to.AsWidget().SetFocus()
			cb.Events().FocusNextFrom(to)
			assert.True(t, subject.AsWidget().StateIs(states.Focused), "from To to the Subject")
			cb.Scene.Close()
		})()
	}))
}

// TestSwitchSender tests changing the sender of a message being written,
// which changes its signature and its automatic Bcc.
func TestSwitchSender(t *testing.T) {
	old := *Settings
	t.Cleanup(func() { *Settings = old })
	Settings.Signatures = []Signature{{Name: "Work", Text: "Dr. Me\n"}, {Name: "Home", Text: "- Me\n"}}
	Settings.DefaultSignatures = map[string]SignatureName{"work@x.test": "Work", "home@x.test": "Home"}
	Settings.AutoBCC = true
	b := core.NewBody()
	ed := textcore.NewEditor(b)
	sm := &SendMessage{From: "work@x.test", Bcc: []*gomail.Address{{Address: "work@x.test"}}}
	sm.body = "Hello\n\n" + signatureText("work@x.test") + "\n> quoted\n"
	ed.Lines.SetString(sm.body)
	switchSender(sm, ed, "work@x.test", "home@x.test")
	assert.Contains(t, sm.body, "- Me")
	assert.NotContains(t, sm.body, "Dr. Me", "the old signature replaced")
	assert.Contains(t, sm.body, "> quoted")
	assert.Equal(t, sm.body, ed.Lines.String(), "in the editor")
	assert.Equal(t, "home@x.test", sm.Bcc[0].Address, "the Bcc to the sender")
}
