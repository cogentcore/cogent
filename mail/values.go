// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mail

import (
	"fmt"
	"net/mail"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/views"
)

func init() {
	views.AddValue(mail.Address{}, func() views.Value { return &AddressValue{} })
}

// AddressValue represents a [mail.Address] with a [core.TextField].
type AddressValue struct {
	views.ValueBase[*core.TextField]
}

func (v *AddressValue) Update() {
	address := reflectx.NonPointerValue(v.Value).Interface().(mail.Address)
	if v.IsReadOnly() && address.Name != "" && address.Name != address.Address {
		v.Widget.SetText(fmt.Sprintf("%s (%s)", address.Name, address.Address)).Update()
		return
	}
	v.Widget.SetText(address.Address).Update()
}
