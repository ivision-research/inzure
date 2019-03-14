package internal

import (
	"encoding/json"

	"github.com/CarveSystems/inzure"
	"github.com/codegangsta/cli"
)

var (
	DiffNewFile string
	DiffOldFile string
)

type diffReport struct {
	Added    []string
	Modified []string
	Removed  []string
}

var CmdDiffFlags = []cli.Flag{
	cli.StringFlag{
		Name:        "new",
		Usage:       "newer inzure JSON file",
		Destination: &DiffNewFile,
	},
	cli.StringFlag{
		Name:        "old",
		Usage:       "old inzure JSON file",
		Destination: &DiffOldFile,
	},
	OutputFileFlag,
}

var (
	diffRGFields []string
)

func CmdDiff(c *cli.Context) {
	subNew, err := inzure.SubscriptionFromFilePassword(DiffNewFile, getEncryptPassword(c))
	if err != nil {
		exitError(1, err.Error())
	}
	subOld, err := inzure.SubscriptionFromFilePassword(DiffOldFile, getEncryptPassword(c))
	if err != nil {
		exitError(1, err.Error())
	}
	if subNew.ID != subOld.ID {
		exitError(1, "subscriptions IDs don't match: %s != %s", subNew.ID, subOld.ID)
	}
	report, err := subNew.Diff(subOld)
	if err != nil {
		exitError(1, err.Error())
	}
	ow := getOutputWriter("")
	json.NewEncoder(ow).Encode(&report)
}
