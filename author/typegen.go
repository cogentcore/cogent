// Code generated by "core generate -add-types -add-funcs"; DO NOT EDIT.

package author

import (
	"cogentcore.org/core/types"
)

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/author.Formats", IDName: "formats"})

var _ = types.AddType(&types.Type{Name: "cogentcore.org/cogent/author.Config", IDName: "config", Doc: "Config is the configuration information for the author cli.", Fields: []types.Field{{Name: "Output", Doc: "Output is the base name of the file or path to generate.\nThe appropriate extension will be added based on the output format."}, {Name: "Formats", Doc: "Formats are the list of formats for the generated output."}}})

var _ = types.AddFunc(&types.Func{Name: "cogentcore.org/cogent/author.Setup", Doc: "Setup runs commands to install the necessary pandoc files using\nplatform specific install commands.", Args: []string{"c"}, Returns: []string{"error"}})