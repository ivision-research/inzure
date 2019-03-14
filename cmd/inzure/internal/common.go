package internal

import (
	"fmt"
	"io"
	"os"

	"github.com/CarveSystems/inzure"
	"github.com/codegangsta/cli"
)

var (
	OutputFile     = ""
	OutputFileFlag = cli.StringFlag{
		Name:        "o",
		Usage:       "Output file",
		Destination: &OutputFile,
	}
	InputFile     = ""
	InputFileFlag = cli.StringFlag{
		Name:        "f",
		EnvVar:      inzure.EnvSubscriptionJSON,
		Usage:       "Input inzure JSON file",
		Destination: &InputFile,
	}
)

func requiresInputFile() {
	if InputFile == "" {
		exitError(1, "Need to set -f option")
	}
}

func getOutputWriter(def string) io.Writer {
	var out io.Writer
	var err error
	if OutputFile != "" {
		out, err = os.OpenFile(OutputFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
		if err != nil {
			errorf("failed to open output file %s: %v", OutputFile, err)
			os.Exit(1)
		}
	} else if def != "" {
		out, err = os.OpenFile(def, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
		if err != nil {
			errorf("failed to open output file %s: %v", OutputFile, err)
			os.Exit(1)
		}
	} else {
		out = os.Stdout
	}
	return out
}

func getSubscription(c *cli.Context) *inzure.Subscription {
	sub, err := inzure.SubscriptionFromFilePassword(InputFile, getEncryptPassword(c))
	if err != nil {
		exitError(1, err.Error())
	}
	return sub
}

func errorf(fm string, params ...interface{}) {
	fmt.Fprintf(os.Stderr, fm+"\n", params...)
}

func exitError(code int, fm string, params ...interface{}) {
	errorf(fm, params...)
	os.Exit(code)
}

func getEncryptPassword(ctx *cli.Context) []byte {
	s := ctx.GlobalString("password")
	if s == "" {
		return nil
	}
	return []byte(s)
}
