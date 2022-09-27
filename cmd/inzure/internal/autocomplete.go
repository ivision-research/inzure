package internal

import "github.com/CarveSystems/inzure/cmd/inzure/internal/autocomplete"

func SetupAutoCompletions() {
	autocomplete.AddCompletions(
		"search",
		map[string]autocomplete.CompleteFunc{
			"f":                        autocomplete.InzureJSONAutoComplete("-f"),
			"o":                        autocomplete.FileAutoComplete,
			autocomplete.Positional(1): autocomplete.IQSAutoComplete,
		},
	)

	autocomplete.AddCompletions(
		"diff",
		map[string]autocomplete.CompleteFunc{
			"new": autocomplete.InzureJSONAutoComplete("-new"),
			"old": autocomplete.InzureJSONAutoComplete("-old"),
			"o":   autocomplete.FileAutoComplete,
		},
	)

	autocomplete.AddCompletions(
		"gather",
		map[string]autocomplete.CompleteFunc{
			"cert":     autocomplete.FileAutoComplete,
			"dir":      autocomplete.DirAutoComplete,
			"o":        autocomplete.FileAutoComplete,
			"sub-file": autocomplete.FileAutoComplete,
			"targets":  autocomplete.TargetsAutoComplete,
			"exclude":  autocomplete.TargetsAutoComplete,
		},
	)

	autocomplete.AddCompletions(
		"attacksurface",
		map[string]autocomplete.CompleteFunc{
			"f": autocomplete.InzureJSONAutoComplete("-f"),
			"o": autocomplete.FileAutoComplete,
		},
	)

	autocomplete.AddCompletions(
		"dump",
		map[string]autocomplete.CompleteFunc{
			"f": autocomplete.InzureJSONAutoComplete("-f"),
			"o": autocomplete.FileAutoComplete,
		},
	)

	autocomplete.AddCompletions(
		"whitelist",
		map[string]autocomplete.CompleteFunc{
			"f": autocomplete.InzureJSONAutoComplete("-f"),
			"o": autocomplete.FileAutoComplete,
		},
	)
}
