package internal

import "github.com/CarveSystems/inzure/cmd/inzure/internal/autocomplete"

func SetupAutoCompletions() {
	autocomplete.AddCompletions(
		"search",
		map[string]autocomplete.CompleteFunc{
			"f": autocomplete.InzureJSONAutoComplete,
			"o": autocomplete.FileAutoComplete,
			autocomplete.Positional(1): autocomplete.IQSAutoComplete,
		},
	)

	autocomplete.AddCompletions(
		"diff",
		map[string]autocomplete.CompleteFunc{
			"new": autocomplete.InzureJSONAutoComplete,
			"old": autocomplete.InzureJSONAutoComplete,
			"o":   autocomplete.FileAutoComplete,
		},
	)

	autocomplete.AddCompletions(
		"gather",
		map[string]autocomplete.CompleteFunc{
			"cert": autocomplete.FileAutoComplete,
			"dir":  autocomplete.DirAutoComplete,
			"o":    autocomplete.FileAutoComplete,
		},
	)

	autocomplete.AddCompletions(
		"attacksurface",
		map[string]autocomplete.CompleteFunc{
			"f": autocomplete.InzureJSONAutoComplete,
			"o": autocomplete.FileAutoComplete,
		},
	)

	autocomplete.AddCompletions(
		"dump",
		map[string]autocomplete.CompleteFunc{
			"f": autocomplete.InzureJSONAutoComplete,
			"o": autocomplete.FileAutoComplete,
		},
	)

	autocomplete.AddCompletions(
		"whitelist",
		map[string]autocomplete.CompleteFunc{
			"f": autocomplete.InzureJSONAutoComplete,
			"o": autocomplete.FileAutoComplete,
		},
	)
}
