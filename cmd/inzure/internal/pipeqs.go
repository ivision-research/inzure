package internal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/CarveSystems/inzure"
	"github.com/urfave/cli"
)

type pqsMeta struct {
	Meta inzure.ResourceID
}

func (m *pqsMeta) display() {
	qs, err := m.Meta.QueryString()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		fmt.Println(qs)
	}
}

var (
	PipeQSJSON = false
)

var CmdPipeQSFlags = []cli.Flag{
	cli.BoolFlag{
		Name:        "json",
		Usage:       "Read a JSON instead of newline separated RawIDs",
		Destination: &PipeQSJSON,
	},
}

func CmdPipeQS(c *cli.Context) {
	if PipeQSJSON {
		cmdPipeQSJSON(c)
	} else {
		cmdPipeQSNewlines(c)
	}
}

func cmdPipeQSJSON(c *cli.Context) {
	var tmp json.RawMessage
	err := json.NewDecoder(os.Stdin).Decode(&tmp)
	if err != nil {
		exitError(1, err.Error())
	}
	switch tmp[0] {
	case '[':
		into := make([]pqsMeta, 0, 5)
		if err := json.Unmarshal(tmp, &into); err != nil {
			exitError(1, err.Error())
		}
		for _, meta := range into {
			meta.display()
		}
	case '{':
		var into pqsMeta
		if err := json.Unmarshal(tmp, &into); err != nil {
			exitError(1, err.Error())
		}
		into.display()
	}
}

func cmdPipeQSNewlines(c *cli.Context) {
	scan := bufio.NewScanner(os.Stdin)
	for scan.Scan() {
		var r inzure.ResourceID
		t := strings.Trim(scan.Text(), `"'`)
		r.FromID(t)
		qs, err := r.QueryString()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			fmt.Fprintln(os.Stdout, qs)
		}
	}
}
