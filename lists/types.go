// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/strcase"
)

// FieldTypes are the types of the values of the fields of a list type.
type FieldTypes int32 //enums:enum

const (
	// Text is text.
	Text FieldTypes = iota

	// Check is a check box: yes or no.
	Check

	// Integer is a whole number.
	Integer

	// Number is a number with decimals.
	Number

	// Date is a date.
	Date

	// DateTime is a date and a time.
	DateTime

	// Duration is a length of time, such as 1h30m.
	Duration
)

// goType returns the Go type of the values of the field type,
// and its struct tags.
func (ft FieldTypes) goType() (reflect.Type, string) {
	switch ft {
	case Check:
		return reflect.TypeFor[bool](), ""
	case Integer:
		return reflect.TypeFor[int](), ""
	case Number:
		return reflect.TypeFor[float64](), ""
	case Date:
		return reflect.TypeFor[time.Time](), `display:"date"`
	case DateTime:
		return reflect.TypeFor[time.Time](), ""
	case Duration:
		return reflect.TypeFor[time.Duration](), ""
	}
	return reflect.TypeFor[string](), ""
}

// Field is a field of a list type, a column of its lists.
type Field struct {

	// Name is the name of the field, at the top of its column.
	Name string

	// Type is the type of its values.
	Type FieldTypes

	// Default is the value for a new item, such as 1 for a number.
	Default string

	// Tags are more struct field tags for how the values are shown and
	// edited, such as min:"0" max:"10" step:"0.5" for a number,
	// format:"%.2f" for its format, and edit:"-" for no editing.
	Tags string

	// SetBy is, for a date field, the name of a check field, such as
	// Done, whose checking sets this date to today, if it is not set.
	SetBy string
}

// ListType is a type of list, with the fields of its items.
// The first field is the tree of the items, with the categories
// of the items as their parents, such as the aisles of a store
// in a shopping list.
type ListType struct {

	// Name is the name of the list type.
	Name string

	// Fields are the fields of the items, the columns of the lists.
	Fields []*Field
}

func (lt *ListType) Label() string { return lt.Name }

// The fields of the struct type of the items for the tree structure,
// which are not in [ListType.Fields], and so have names that a field
// name never becomes (see [goName]).
const (
	closedField   = "Closed_"
	childrenField = "Items_"
)

// StructType returns the struct type of the items of the list type, made
// with [reflect.StructOf], which returns the same type for the same fields.
// Field i of the struct is Fields[i], followed by the closed and children
// fields for the [core.TreeTable].
func (lt *ListType) StructType() reflect.Type {
	fields := make([]reflect.StructField, 0, len(lt.Fields)+2)
	used := map[string]bool{}
	for _, f := range lt.Fields {
		name := goName(f.Name, used)
		typ, tags := f.Type.goType()
		if strcase.ToSentence(name) != f.Name && f.Name != "" {
			tags += fmt.Sprintf(" label:%q", f.Name)
		}
		if f.Default != "" {
			tags += fmt.Sprintf(" default:%q", f.Default)
		}
		tags += " " + f.Tags
		fields = append(fields, reflect.StructField{Name: name, Type: typ, Tag: reflect.StructTag(strings.TrimSpace(tags))})
	}
	fields = append(fields,
		reflect.StructField{Name: closedField, Type: reflect.TypeFor[bool](), Tag: `treetable:"closed" display:"-"`},
		reflect.StructField{Name: childrenField, Type: reflect.TypeFor[[]any](), Tag: `treetable:"children" display:"-"`})
	return reflect.StructOf(fields)
}

// goName returns the name of a struct field for the field with the given
// name: an exported Go identifier, not in used, which it is added to.
func goName(name string, used map[string]bool) string {
	gn := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, strcase.ToCamel(name))
	if gn == "" || !unicode.IsLetter([]rune(gn)[0]) || !unicode.IsUpper([]rune(gn)[0]) {
		gn = "F" + gn
	}
	base := gn
	for i := 2; used[gn]; i++ {
		gn = fmt.Sprintf("%s%d", base, i)
	}
	used[gn] = true
	return gn
}

// Item is the saved form of an item of a list, with its values by field
// name, so that they are kept when the fields of its type change.
type Item struct {

	// Values are the values of the item by field name, in JSON.
	Values map[string]json.RawMessage

	// Closed is whether the item is closed, with its items not shown.
	Closed bool `json:",omitempty"`

	// Items are the items in the item, as in a category.
	Items []*Item `json:",omitempty"`
}

// newItems returns a pointer to a new slice of pointers to items of
// the [ListType.StructType], with the values of the given saved items.
func (lt *ListType) newItems(saved []*Item) reflect.Value {
	st := lt.StructType()
	items := reflect.New(reflect.SliceOf(reflect.PointerTo(st)))
	for _, it := range saved {
		items.Elem().Set(reflect.Append(items.Elem(), lt.newItem(st, it)))
	}
	return items
}

// newItem returns a new item of the given struct type, with the values
// of the given saved item, and the default values for the others.
func (lt *ListType) newItem(st reflect.Type, saved *Item) reflect.Value {
	item := reflect.New(st)
	errors.Log(reflectx.SetFromDefaultTags(item.Interface()))
	v := item.Elem()
	for i, f := range lt.Fields {
		if raw, ok := saved.Values[f.Name]; ok {
			// a value of another type, after the type of the field
			// changes, is dropped
			json.Unmarshal(raw, v.Field(i).Addr().Interface())
		}
	}
	v.Field(len(lt.Fields)).SetBool(saved.Closed)
	children := v.Field(len(lt.Fields) + 1)
	for _, c := range saved.Items {
		children.Set(reflect.Append(children, lt.newItem(st, c)))
	}
	return item
}

// savedItems returns the saved items for the given pointer
// to a slice of pointers to items, from [ListType.newItems].
func (lt *ListType) savedItems(items reflect.Value) []*Item {
	sl := items.Elem()
	saved := make([]*Item, 0, sl.Len())
	for i := range sl.Len() {
		saved = append(saved, lt.savedItem(reflect.ValueOf(sl.Index(i).Interface())))
	}
	return saved
}

// savedItem returns the saved item for the given pointer to an item.
func (lt *ListType) savedItem(item reflect.Value) *Item {
	v := item.Elem()
	saved := &Item{Values: map[string]json.RawMessage{}}
	for i, f := range lt.Fields {
		b, err := json.Marshal(v.Field(i).Interface())
		if errors.Log(err) == nil {
			saved.Values[f.Name] = b
		}
	}
	saved.Closed = v.Field(len(lt.Fields)).Bool()
	children := v.Field(len(lt.Fields) + 1)
	for i := range children.Len() {
		saved.Items = append(saved.Items, lt.savedItem(children.Index(i).Elem()))
	}
	return saved
}

// checkKey is a check field of an item, for [ListType.checkStates].
type checkKey struct {
	item  any // the pointer to the item
	field int
}

// setByFields returns the indexes of the date fields that are set by
// check fields (see [Field.SetBy]), and of those check fields.
func (lt *ListType) setByFields() (dates, checks []int) {
	for i, f := range lt.Fields {
		if (f.Type != Date && f.Type != DateTime) || f.SetBy == "" {
			continue
		}
		for j, c := range lt.Fields {
			if c.Type == Check && c.Name == f.SetBy {
				dates, checks = append(dates, i), append(checks, j)
				break
			}
		}
	}
	return
}

// walkItems calls the given function on each of the given items, from
// [ListType.newItems], and the items in them, as struct values.
func (lt *ListType) walkItems(items reflect.Value, fun func(item reflect.Value)) {
	var walk func(sl reflect.Value)
	walk = func(sl reflect.Value) {
		for i := range sl.Len() {
			item := sl.Index(i)
			if item.Kind() == reflect.Interface {
				item = item.Elem()
			}
			fun(item)
			walk(item.Elem().Field(len(lt.Fields) + 1))
		}
	}
	walk(items.Elem())
}

// checkStates returns the states of the check fields of the given items,
// from [ListType.newItems], that set date fields.
func (lt *ListType) checkStates(items reflect.Value) map[checkKey]bool {
	_, checks := lt.setByFields()
	states := map[checkKey]bool{}
	if len(checks) == 0 {
		return states
	}
	lt.walkItems(items, func(item reflect.Value) {
		for _, c := range checks {
			states[checkKey{item.Interface(), c}] = item.Elem().Field(c).Bool()
		}
	})
	return states
}

// setDates sets the date fields of the given items, from
// [ListType.newItems], whose check fields that set them were checked
// since the given states, to now, if they are not set, returning
// whether any were.
func (lt *ListType) setDates(items reflect.Value, prev map[checkKey]bool) bool {
	dates, checks := lt.setByFields()
	set := false
	now := time.Now()
	lt.walkItems(items, func(item reflect.Value) {
		for k, c := range checks {
			v := item.Elem()
			if !v.Field(c).Bool() || prev[checkKey{item.Interface(), c}] {
				continue
			}
			date := v.Field(dates[k])
			if !date.Interface().(time.Time).IsZero() {
				continue
			}
			t := now.Truncate(time.Minute)
			if lt.Fields[dates[k]].Type == Date {
				y, m, d := now.Date()
				t = time.Date(y, m, d, 0, 0, 0, 0, time.Local)
			}
			date.Set(reflect.ValueOf(t))
			set = true
		}
	})
	return set
}

// renameField renames the values of the field with the old name to
// the new name in the given saved items.
func renameField(items []*Item, old, new string) {
	for _, it := range items {
		if v, ok := it.Values[old]; ok {
			delete(it.Values, old)
			it.Values[new] = v
		}
		renameField(it.Items, old, new)
	}
}

// standardTypes returns the list types of a new library.
func standardTypes() []*ListType {
	return []*ListType{
		{Name: "To do", Fields: []*Field{
			{Name: "Task"},
			{Name: "Done", Type: Check},
			{Name: "Due", Type: Date},
			{Name: "Priority", Type: Integer, Default: "3", Tags: `min:"1" max:"5"`},
			{Name: "Notes"},
			{Name: "Done on", Type: Date, SetBy: "Done"},
		}},
		{Name: "Shopping", Fields: []*Field{
			{Name: "Item"},
			{Name: "Got", Type: Check},
			{Name: "Quantity", Type: Integer, Default: "1", Tags: `min:"0"`},
			{Name: "Store"},
			{Name: "Notes"},
		}},
		{Name: "Project", Fields: []*Field{
			{Name: "Task"},
			{Name: "Done", Type: Check},
			{Name: "Owner"},
			{Name: "Due", Type: Date},
			{Name: "Hours", Type: Number, Tags: `min:"0" step:"0.5"`},
			{Name: "Done on", Type: Date, SetBy: "Done"},
		}},
		{Name: "Reading", Fields: []*Field{
			{Name: "Title"},
			{Name: "Author"},
			{Name: "Read", Type: Check},
			{Name: "Rating", Type: Integer, Tags: `min:"0" max:"5"`},
		}},
		{Name: "Simple", Fields: []*Field{
			{Name: "Item"},
			{Name: "Done", Type: Check},
		}},
	}
}
