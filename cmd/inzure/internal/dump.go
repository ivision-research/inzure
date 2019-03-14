package internal

import (
	"encoding/json"

	"github.com/codegangsta/cli"
)

var CmdDumpFlags = []cli.Flag{
	OutputFileFlag,
	InputFileFlag,
}

func CmdDump(c *cli.Context) {
	requiresInputFile()
	to := getOutputWriter("")
	sub := getSubscription(c)
	err := json.NewEncoder(to).Encode(sub)
	if err != nil {
		exitError(1, "error dumping %s: %v", InputFile, err)
	}
}
