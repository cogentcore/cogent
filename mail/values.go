// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"cogentcore.org/core/core"
)

func init() {
	// TODO: unclear if we need a distinct type here?
	// core.AddValue(mail.Address{}, func() core.Value { return &AddressValue{} })
}

// AddressTextField represents a [mail.Address] with a [core.TextField].
type AddressTextField struct {
	core.TextField
}

// AddressTextField registers [core.Chooser] as the [core.Value] widget
// for [SplitName]
func (av AddressTextField) Value() core.Value {
	return core.NewTextField()
}

// func (v *AddressValue) Config() {
// 	v.Widget.OnChange(func(e events.Event) {
// 		reflectx.OnePointerValue(v.Value).Interface().(*mail.Address).Address = v.Widget.Text()
// 	})
// }
//
// func (v *AddressValue) Update() {
// 	address := reflectx.NonPointerValue(v.Value).Interface().(mail.Address)
// 	if v.IsReadOnly() && address.Name != "" && address.Name != address.Address {
// 		v.Widget.SetText(fmt.Sprintf("%s (%s)", address.Name, address.Address)).Update()
// 		return
// 	}
// 	v.Widget.SetText(address.Address).Update()
// }
