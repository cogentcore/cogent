// Code generated by "core generate"; DO NOT EDIT.

package code

import (
	"cogentcore.org/core/enums"
)

var _ArgVarTypesValues = []ArgVarTypes{0, 1, 2, 3, 4, 5}

// ArgVarTypesN is the highest valid value for type ArgVarTypes, plus one.
const ArgVarTypesN ArgVarTypes = 6

var _ArgVarTypesValueMap = map[string]ArgVarTypes{`File`: 0, `Dir`: 1, `Ext`: 2, `Pos`: 3, `Text`: 4, `Prompt`: 5}

var _ArgVarTypesDescMap = map[ArgVarTypes]string{0: `ArgVarFile is a file name, not a directory`, 1: `ArgVarDir is a directory name, not a file`, 2: `ArgVarExt is a file extension`, 3: `ArgVarPos is a text position`, 4: `ArgVarText is text from a buffer`, 5: `ArgVarPrompt is a user-prompted variable`}

var _ArgVarTypesMap = map[ArgVarTypes]string{0: `File`, 1: `Dir`, 2: `Ext`, 3: `Pos`, 4: `Text`, 5: `Prompt`}

// String returns the string representation of this ArgVarTypes value.
func (i ArgVarTypes) String() string { return enums.String(i, _ArgVarTypesMap) }

// SetString sets the ArgVarTypes value from its string representation,
// and returns an error if the string is invalid.
func (i *ArgVarTypes) SetString(s string) error {
	return enums.SetString(i, s, _ArgVarTypesValueMap, "ArgVarTypes")
}

// Int64 returns the ArgVarTypes value as an int64.
func (i ArgVarTypes) Int64() int64 { return int64(i) }

// SetInt64 sets the ArgVarTypes value from an int64.
func (i *ArgVarTypes) SetInt64(in int64) { *i = ArgVarTypes(in) }

// Desc returns the description of the ArgVarTypes value.
func (i ArgVarTypes) Desc() string { return enums.Desc(i, _ArgVarTypesDescMap) }

// ArgVarTypesValues returns all possible values for the type ArgVarTypes.
func ArgVarTypesValues() []ArgVarTypes { return _ArgVarTypesValues }

// Values returns all possible values for the type ArgVarTypes.
func (i ArgVarTypes) Values() []enums.Enum { return enums.Values(_ArgVarTypesValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i ArgVarTypes) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *ArgVarTypes) UnmarshalText(text []byte) error {
	return enums.UnmarshalText(i, text, "ArgVarTypes")
}

var _FindLocValues = []FindLoc{0, 1, 2, 3, 4}

// FindLocN is the highest valid value for type FindLoc, plus one.
const FindLocN FindLoc = 5

var _FindLocValueMap = map[string]FindLoc{`Open`: 0, `All`: 1, `File`: 2, `Dir`: 3, `NotTop`: 4}

var _FindLocDescMap = map[FindLoc]string{0: `FindOpen finds in all open folders in the left file browser`, 1: `FindLocAll finds in all directories under the root path. can be slow for large file trees`, 2: `FindLocFile only finds in the current active file`, 3: `FindLocDir only finds in the directory of the current active file`, 4: `FindLocNotTop finds in all open folders *except* the top-level folder`}

var _FindLocMap = map[FindLoc]string{0: `Open`, 1: `All`, 2: `File`, 3: `Dir`, 4: `NotTop`}

// String returns the string representation of this FindLoc value.
func (i FindLoc) String() string { return enums.String(i, _FindLocMap) }

// SetString sets the FindLoc value from its string representation,
// and returns an error if the string is invalid.
func (i *FindLoc) SetString(s string) error {
	return enums.SetString(i, s, _FindLocValueMap, "FindLoc")
}

// Int64 returns the FindLoc value as an int64.
func (i FindLoc) Int64() int64 { return int64(i) }

// SetInt64 sets the FindLoc value from an int64.
func (i *FindLoc) SetInt64(in int64) { *i = FindLoc(in) }

// Desc returns the description of the FindLoc value.
func (i FindLoc) Desc() string { return enums.Desc(i, _FindLocDescMap) }

// FindLocValues returns all possible values for the type FindLoc.
func FindLocValues() []FindLoc { return _FindLocValues }

// Values returns all possible values for the type FindLoc.
func (i FindLoc) Values() []enums.Enum { return enums.Values(_FindLocValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i FindLoc) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *FindLoc) UnmarshalText(text []byte) error { return enums.UnmarshalText(i, text, "FindLoc") }

var _KeyFunctionsValues = []KeyFunctions{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21}

// KeyFunctionsN is the highest valid value for type KeyFunctions, plus one.
const KeyFunctionsN KeyFunctions = 22

var _KeyFunctionsValueMap = map[string]KeyFunctions{`None`: 0, `Needs2`: 1, `NextPanel`: 2, `PrevPanel`: 3, `FileOpen`: 4, `BufSelect`: 5, `BufClone`: 6, `BufSave`: 7, `BufSaveAs`: 8, `BufClose`: 9, `ExecCmd`: 10, `RectCopy`: 11, `RectCut`: 12, `RectPaste`: 13, `RegCopy`: 14, `RegPaste`: 15, `CommentOut`: 16, `Indent`: 17, `Jump`: 18, `SetSplit`: 19, `BuildProject`: 20, `RunProject`: 21}

var _KeyFunctionsDescMap = map[KeyFunctions]string{0: ``, 1: `special internal signal returned by KeyFunction indicating need for second key`, 2: `move to next panel to the right`, 3: `move to prev panel to the left`, 4: `open a new file in active texteditor`, 5: `select an open buffer to edit in active texteditor`, 6: `open active file in other view`, 7: `save active texteditor buffer to its file`, 8: `save as active texteditor buffer to its file`, 9: `close active texteditor buffer`, 10: `execute a command on active texteditor buffer`, 11: `copy rectangle`, 12: `cut rectangle`, 13: `paste rectangle`, 14: `copy selection to named register`, 15: `paste selection from named register`, 16: `comment out region`, 17: `indent region`, 18: `jump to line (same as keyfun.Jump)`, 19: `set named splitter config`, 20: `build overall project`, 21: `run overall project`}

var _KeyFunctionsMap = map[KeyFunctions]string{0: `None`, 1: `Needs2`, 2: `NextPanel`, 3: `PrevPanel`, 4: `FileOpen`, 5: `BufSelect`, 6: `BufClone`, 7: `BufSave`, 8: `BufSaveAs`, 9: `BufClose`, 10: `ExecCmd`, 11: `RectCopy`, 12: `RectCut`, 13: `RectPaste`, 14: `RegCopy`, 15: `RegPaste`, 16: `CommentOut`, 17: `Indent`, 18: `Jump`, 19: `SetSplit`, 20: `BuildProject`, 21: `RunProject`}

// String returns the string representation of this KeyFunctions value.
func (i KeyFunctions) String() string { return enums.String(i, _KeyFunctionsMap) }

// SetString sets the KeyFunctions value from its string representation,
// and returns an error if the string is invalid.
func (i *KeyFunctions) SetString(s string) error {
	return enums.SetString(i, s, _KeyFunctionsValueMap, "KeyFunctions")
}

// Int64 returns the KeyFunctions value as an int64.
func (i KeyFunctions) Int64() int64 { return int64(i) }

// SetInt64 sets the KeyFunctions value from an int64.
func (i *KeyFunctions) SetInt64(in int64) { *i = KeyFunctions(in) }

// Desc returns the description of the KeyFunctions value.
func (i KeyFunctions) Desc() string { return enums.Desc(i, _KeyFunctionsDescMap) }

// KeyFunctionsValues returns all possible values for the type KeyFunctions.
func KeyFunctionsValues() []KeyFunctions { return _KeyFunctionsValues }

// Values returns all possible values for the type KeyFunctions.
func (i KeyFunctions) Values() []enums.Enum { return enums.Values(_KeyFunctionsValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i KeyFunctions) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *KeyFunctions) UnmarshalText(text []byte) error {
	return enums.UnmarshalText(i, text, "KeyFunctions")
}

var _SymScopesValues = []SymScopes{0, 1}

// SymScopesN is the highest valid value for type SymScopes, plus one.
const SymScopesN SymScopes = 2

var _SymScopesValueMap = map[string]SymScopes{`Package`: 0, `File`: 1}

var _SymScopesDescMap = map[SymScopes]string{0: `SymScopePackage scopes list of symbols to the package of the active file`, 1: `SymScopeFile restricts the list of symbols to the active file`}

var _SymScopesMap = map[SymScopes]string{0: `Package`, 1: `File`}

// String returns the string representation of this SymScopes value.
func (i SymScopes) String() string { return enums.String(i, _SymScopesMap) }

// SetString sets the SymScopes value from its string representation,
// and returns an error if the string is invalid.
func (i *SymScopes) SetString(s string) error {
	return enums.SetString(i, s, _SymScopesValueMap, "SymScopes")
}

// Int64 returns the SymScopes value as an int64.
func (i SymScopes) Int64() int64 { return int64(i) }

// SetInt64 sets the SymScopes value from an int64.
func (i *SymScopes) SetInt64(in int64) { *i = SymScopes(in) }

// Desc returns the description of the SymScopes value.
func (i SymScopes) Desc() string { return enums.Desc(i, _SymScopesDescMap) }

// SymScopesValues returns all possible values for the type SymScopes.
func SymScopesValues() []SymScopes { return _SymScopesValues }

// Values returns all possible values for the type SymScopes.
func (i SymScopes) Values() []enums.Enum { return enums.Values(_SymScopesValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i SymScopes) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *SymScopes) UnmarshalText(text []byte) error {
	return enums.UnmarshalText(i, text, "SymScopes")
}
