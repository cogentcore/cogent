// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
)

// A tag is a path: the groups it is in, from the top, and its name, with
// a / between them, as 2017/Kyoto, People/Family/Alice, or just Home, at
// the top. The groups are shown as the branches of the tree of tags.
// A tag that starts with a year, as 2026 Groningen or 2026_Groningen, is
// in the group of that year (2026/Groningen), when it is in no other.
// Groups are there for the tags in them, and also when they are made
// explicitly, empty, which are recorded in the store (see [Library.Groups]).

// tagSep is the separator of the groups and the name of a tag.
const tagSep = "/"

// cleanTag returns the given tag cleaned up: without spaces around its
// parts, or empty parts, and in the group of its year if it starts with
// a year and is in no other group, as 2026 Groningen.
func cleanTag(tag string) string {
	var parts []string
	for _, p := range strings.Split(tag, tagSep) {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 1 {
		if year, name, ok := yearPrefix(parts[0]); ok {
			parts = []string{year, name}
		}
	}
	return strings.Join(parts, tagSep)
}

// yearPrefix returns the year and the rest of the given name, if it starts
// with a year and an underscore or a space, as 2017_Kyoto or 2017 Kyoto.
func yearPrefix(name string) (year, rest string, ok bool) {
	if len(name) < 6 || !isYear(name[:4]) || (name[4] != '_' && name[4] != ' ') {
		return "", name, false
	}
	rest = strings.TrimSpace(name[5:])
	return name[:4], rest, rest != ""
}

// isYear returns whether the given name of a group is a year, as 2017,
// which are sorted before other groups, the newest first.
func isYear(name string) bool {
	if len(name) != 4 {
		return false
	}
	for _, c := range name {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// joinTag returns the tag of the given name in the given group, "" for the
// top, cleaned up (see [cleanTag]).
func joinTag(group, name string) string {
	return cleanTag(group + tagSep + name)
}

// splitTag returns the group of the given tag (or group), "" for the top,
// and its name.
func splitTag(tag string) (group, name string) {
	i := strings.LastIndex(tag, tagSep)
	if i < 0 {
		return "", tag
	}
	return tag[:i], tag[i+1:]
}

// tagGroups returns the groups that the given tag (or group) is in, from
// the top.
func tagGroups(tag string) []string {
	var res []string
	for i, c := range tag {
		if c == '/' {
			res = append(res, tag[:i])
		}
	}
	return res
}

// inGroup returns whether the given tag (or group) is in the given group,
// at any depth.
func inGroup(tag, group string) bool {
	return strings.HasPrefix(tag, group+tagSep)
}

// tagLabel returns the given tag as shown, as 2017 / Kyoto.
func tagLabel(tag string) string {
	return strings.ReplaceAll(tag, tagSep, " / ")
}

// groupsKey is the key of the groups made explicitly in the meta of the store.
const groupsKey = "groups"

// madeGroups returns the groups made explicitly, which may be empty.
func (lib *Library) madeGroups() []string {
	var res []string
	if s := lib.store.meta(groupsKey); s != "" {
		json.Unmarshal([]byte(s), &res)
	}
	return res
}

// setMadeGroups sets the groups made explicitly.
func (lib *Library) setMadeGroups(groups []string) error {
	slices.Sort(groups)
	groups = slices.Compact(groups)
	b, err := json.Marshal(groups)
	if err != nil {
		return err
	}
	return lib.store.setMeta(groupsKey, string(b))
}

// Groups returns all of the groups: those made explicitly, and those of
// the tags of the photos (not in the trash), with those they are in.
func (lib *Library) Groups() []string {
	set := map[string]bool{}
	add := func(g string) {
		set[g] = true
		for _, a := range tagGroups(g) {
			set[a] = true
		}
	}
	for _, g := range lib.madeGroups() {
		add(g)
	}
	for t := range lib.Tags() {
		if g, _ := splitTag(t); g != "" {
			add(g)
		}
	}
	res := make([]string, 0, len(set))
	for g := range set {
		res = append(res, g)
	}
	slices.Sort(res)
	return res
}

// AddGroup makes the given group, which is there even when no tags are
// in it, until it is deleted.
func (lib *Library) AddGroup(group string) error {
	group = cleanTag(group)
	if group == "" {
		return nil
	}
	return lib.setMadeGroups(append(lib.madeGroups(), group))
}

// MoveTag moves the given tag into the given group, "" for the top,
// returning its new tag.
func (lib *Library) MoveTag(tag, group string) (string, error) {
	_, name := splitTag(tag)
	nt := joinTag(group, name)
	return nt, lib.RenameTag(tag, nt)
}

// RenameGroup renames the given group as the given one, with all of the
// tags and groups in it, as for moving it into another group. When the
// new group is "", the tags and groups in it are moved to the top, and
// the group is gone, as it is when it is renamed as its own group, which
// moves them up into that group.
func (lib *Library) RenameGroup(old, group string) error {
	group = cleanTag(group)
	if group == old {
		return nil
	}
	if inGroup(group, old) {
		return errors.New("a group cannot be moved into itself")
	}
	move := func(t string) string {
		return joinTag(group, strings.TrimPrefix(t, old+tagSep))
	}
	var made []string
	for _, g := range lib.madeGroups() {
		switch {
		case g == old:
			if g, _ := splitTag(old); g != group { // not deleted
				made = append(made, group)
			}
		case inGroup(g, old):
			made = append(made, move(g))
		default:
			made = append(made, g)
		}
	}
	var errs []error
	errs = append(errs, lib.setMadeGroups(slices.DeleteFunc(made, func(g string) bool { return g == "" })))
	phs := lib.Photos(func(ph *Photo) bool {
		return slices.ContainsFunc(ph.Tags, func(t string) bool { return inGroup(t, old) })
	})
	lib.mu.Lock()
	for _, ph := range phs {
		tags := slices.Clone(ph.Tags)
		for _, t := range tags {
			if inGroup(t, old) {
				ph.RemoveTag(t)
				ph.AddTag(move(t))
			}
		}
	}
	lib.mu.Unlock()
	errs = append(errs, lib.Save(phs...))
	return errors.Join(errs...)
}

// DeleteGroup deletes the given group, moving the tags and groups in it
// up into the group it is in.
func (lib *Library) DeleteGroup(group string) error {
	parent, _ := splitTag(group)
	return lib.RenameGroup(group, parent)
}

// tagFormatKey is the key of the format of the tags in the meta of the
// store: tags were years and names with an underscore, as 2017_Kyoto,
// before they were paths of groups.
const tagFormatKey = "tagFormat"

// migrateTags changes the tags of years with an underscore or a space, as
// 2017_Kyoto, into tags in the groups of their years, as 2017/Kyoto, once.
func (lib *Library) migrateTags() error {
	if lib.store.meta(tagFormatKey) == "groups" {
		return nil
	}
	var changed []*Photo
	lib.mu.Lock()
	for _, ph := range lib.photos {
		ch := false
		for i, t := range ph.Tags {
			if strings.Contains(t, tagSep) {
				continue
			}
			if year, name, ok := yearPrefix(t); ok {
				ph.Tags[i] = year + tagSep + name
				ch = true
			}
		}
		if ch {
			changed = append(changed, ph)
		}
	}
	lib.mu.Unlock()
	if err := lib.Save(changed...); err != nil {
		return err
	}
	return lib.store.setMeta(tagFormatKey, "groups")
}
