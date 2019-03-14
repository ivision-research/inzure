package main

import (
	"fmt"
	"os"

	internal "github.com/CarveSystems/inzure/cmd/inzure/internal"
	"github.com/codegangsta/cli"
)

var GlobalFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "p",
		Usage: "Set if you would like to enter an encryption password for inzure JSON files",
	},
	cli.StringFlag{
		Name:   "password",
		Hidden: true,
	},
}

var Commands = []cli.Command{
	{
		Name:   "gather",
		Usage:  "Gather all of the data from an Azure subscription",
		Action: internal.CmdGather,
		Flags:  internal.CmdGatherFlags,
	},
	/*
		{
			Name:   "view",
			Usage:  "Start a server to view inzure reports in a browser",
			Action: internal.CmdView,
			Flags:  []cli.Flag{},
		},
	*/
	{
		Name:   "search",
		Usage:  "Use inzure query strings to quickly search for data in an inzure JSON",
		Action: internal.CmdSearch,
		Flags:  internal.CmdSearchFlags,
	},
	{
		Name:   "diff",
		Usage:  "View a diff of two inzure JSON files",
		Action: internal.CmdDiff,
		Flags:  internal.CmdDiffFlags,
	},
	{
		Name:   "dump",
		Usage:  "Dump an encrypted inzure JSON",
		Action: internal.CmdDump,
		Flags:  internal.CmdDumpFlags,
	},
	{
		Name:   "whitelist",
		Usage:  "Takes a current inzure firewall setup and formats it as a whitelist for ingestion by the testing framework",
		Action: internal.CmdWhitelist,
		Flags:  internal.CmdWhitelistFlags,
	},
	{
		Name:   "attacksurface",
		Usage:  "Returns all potentially reachable IPs and hosts found in the subscription",
		Action: internal.CmdAttackSurface,
		Flags:  internal.CmdAttackSurfaceFlags,
	},
	{
		Name:   "pipeqs",
		Usage:  "Reads standard input for RawIDs and coverts them to query strings",
		Action: internal.CmdPipeQS,
		Flags:  internal.CmdPipeQSFlags,
	},
}

func CommandNotFound(c *cli.Context, command string) {
	fmt.Fprintf(os.Stderr, "%s: '%s' is not a %s internal. See '%s --help'.", c.App.Name, command, c.App.Name, c.App.Name)
	os.Exit(2)
}
