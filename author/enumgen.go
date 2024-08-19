// Code generated by "core generate -add-types -add-funcs"; DO NOT EDIT.

package author

import (
	"cogentcore.org/core/enums"
)

var _FormatsValues = []Formats{0, 1, 2, 3, 4}

// FormatsN is the highest valid value for type Formats, plus one.
const FormatsN Formats = 5

var _FormatsValueMap = map[string]Formats{`html`: 0, `pdf`: 1, `docx`: 2, `epub`: 3, `latex`: 4}

var _FormatsDescMap = map[Formats]string{0: `HTML is a single standalone .html file.`, 1: `PDF, via LaTeX, with full math support.`, 2: `DOCX is a Microsoft Word compatible .docx file.`, 3: `EPUB is a standard eBook .epub file.`, 4: `LaTeX is a latex file, which can be further customized.`}

var _FormatsMap = map[Formats]string{0: `html`, 1: `pdf`, 2: `docx`, 3: `epub`, 4: `latex`}

// String returns the string representation of this Formats value.
func (i Formats) String() string { return enums.String(i, _FormatsMap) }

// SetString sets the Formats value from its string representation,
// and returns an error if the string is invalid.
func (i *Formats) SetString(s string) error {
	return enums.SetString(i, s, _FormatsValueMap, "Formats")
}

// Int64 returns the Formats value as an int64.
func (i Formats) Int64() int64 { return int64(i) }

// SetInt64 sets the Formats value from an int64.
func (i *Formats) SetInt64(in int64) { *i = Formats(in) }

// Desc returns the description of the Formats value.
func (i Formats) Desc() string { return enums.Desc(i, _FormatsDescMap) }

// FormatsValues returns all possible values for the type Formats.
func FormatsValues() []Formats { return _FormatsValues }

// Values returns all possible values for the type Formats.
func (i Formats) Values() []enums.Enum { return enums.Values(_FormatsValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i Formats) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *Formats) UnmarshalText(text []byte) error { return enums.UnmarshalText(i, text, "Formats") }