package internal

import (
	"encoding/json"
	"github.com/CarveSystems/inzure/pkg/inzure"

	"github.com/urfave/cli"
)

var (
	CmdAttackSurfaceFlags = []cli.Flag{
		InputFileFlag,
		OutputFileFlag,
		BatchFlag,
	}
)

func CmdAttackSurface(c *cli.Context) {
	if Batch {
		subs, err := getBatchSubscriptions(c)
		if err != nil {
			exitError(1, err.Error())
		}
		for _, s := range subs {
			var defaultFile string
			if s.Alias != "" {
				defaultFile = s.Alias + "-attacksurface.json"
			} else {
				defaultFile = s.ID + "-attacksurface.json"
			}
			generateAttackSurface(s, defaultFile)
		}
	} else {
		requiresInputFile()
		sub := getSubscription(c)
		generateAttackSurface(sub, "")
	}

}

func generateAttackSurface(sub *inzure.Subscription, defaultFile string) {
	to := getOutputWriter(defaultFile)
	as := sub.GetAttackSurface()
	err := json.NewEncoder(to).Encode(&as)
	if err != nil {
		exitError(1, err.Error())
	}
}
