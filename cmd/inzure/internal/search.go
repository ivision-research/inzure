package internal

import (
	"encoding/json"
	"io"
	"os"

	"github.com/CarveSystems/inzure"
	"github.com/urfave/cli"
)

var (
	SearchIQS string
)

var CmdSearchFlags = []cli.Flag{
	InputFileFlag,
	OutputFileFlag,
}

func CmdSearch(c *cli.Context) {
	args := c.Args()
	if len(args) == 0 {
		exitError(1, "need to pass a search string")
	}
	SearchIQS = args[0]
	if InputFile == "" {
		exitError(1, "need to specify an input inzure JSON with -f")
	}
	sub, err := inzure.SubscriptionFromFile(InputFile)
	if err != nil {
		exitError(1, err.Error())
	}

	v, err := sub.ReflectFromQueryString(SearchIQS)
	if err != nil {
		le, is := err.(inzure.LexError)
		if is {
			exitError(1, le.ErrorWithHint())
		}
		exitError(1, err.Error())
	}
	var out io.Writer
	if OutputFile != "" {
		out, err = os.OpenFile(OutputFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
		if err != nil {
			exitError(1, err.Error())
		}
	} else {
		out = os.Stdout
	}
	err = json.NewEncoder(out).Encode(v.Interface())
	if err != nil {
		exitError(1, err.Error())
	}
}
