package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/ivision-research/inzure/pkg/inzure"
)

// Exit codes are int values that represent an exit code for a particular error.
const (
	ExitCodeOK    int = 0
	ExitCodeError int = 1 + iota
)

// CLI is the command line object
type CLI struct {
	// outStream and errStream are the stdout and stderr
	// to write message from the CLI.
	outStream, errStream io.Writer
}

// Run invokes the CLI with the given arguments.
func (cli *CLI) Run(args []string) int {
	var (
		cArgs cmdArgs

		version bool
	)

	// Define option flag parse
	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.errStream)

	flags.StringVar(&cArgs.inzureFile, "f", "", "inzure output JSON file")
	flags.StringVar(&cArgs.configFile, "c", "", "config file")

	if cArgs.inzureFile == "" {
		cArgs.inzureFile = os.Getenv(inzure.EnvSubscriptionJSON)
	}

	flags.BoolVar(&version, "version", false, "Print version information and quit.")

	// Parse commandline flag
	if err := flags.Parse(args[1:]); err != nil {
		return ExitCodeError
	}

	// Show version
	if version {
		fmt.Fprintf(cli.errStream, "%s version %s\n", Name, Version)
		return ExitCodeOK
	}

	return slackposter(cli, &cArgs)
}
