package internal

import (
	"encoding/json"

	"github.com/urfave/cli"
)

var (
	CmdAttackSurfaceFlags = []cli.Flag{
		InputFileFlag,
		OutputFileFlag,
	}
)

func CmdAttackSurface(c *cli.Context) {
	requiresInputFile()
	to := getOutputWriter("")
	sub := getSubscription(c)
	as := sub.GetAttackSurface()
	err := json.NewEncoder(to).Encode(&as)
	if err != nil {
		exitError(1, err.Error())
	}
}
