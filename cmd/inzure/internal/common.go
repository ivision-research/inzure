package internal

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/CarveSystems/inzure/pkg/inzure"
	"github.com/chzyer/readline"
	"github.com/urfave/cli"
	"io"
	"os"
	"strings"
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

	Batch     = false
	BatchFlag = cli.BoolFlag{
		Name:        "b",
		Usage:       fmt.Sprintf("Read input files from %s env var", inzure.EnvSubscriptionBatchFiles),
		Destination: &Batch,
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

func getSubscriptionForFile(c *cli.Context, fileName string, password []byte) *inzure.Subscription {
	var err error
	if password == nil && strings.HasSuffix(fileName, inzure.EncryptedFileExtension) {
		password = getEncryptPassword(c)
	}
	sub, err := inzure.SubscriptionFromFilePassword(fileName, password)
	if err != nil {
		exitError(1, err.Error())
	}
	return sub
}

func getSubscription(c *cli.Context) *inzure.Subscription {
	return getSubscriptionForFile(c, InputFile, nil)
}

func getBatchSubscriptions(c *cli.Context) ([]*inzure.Subscription, error) {
	fileNames, err := inzure.BatchFilesFromEnv()
	if err != nil {
		return nil, err
	}
	if len(fileNames) == 0 {
		return nil, fmt.Errorf("no subscriptions in batch file %s", inzure.EnvSubscriptionBatchFiles)
	}
	var subs = make([]*inzure.Subscription, 0, len(fileNames))
	var password []byte = nil
	for _, fname := range fileNames {
		if password == nil && strings.HasSuffix(fname, inzure.EncryptedFileExtension) {
			password = getEncryptPassword(c)
		}
		subs = append(subs, getSubscriptionForFile(c, fname, password))
	}
	return subs, nil
}

func errorf(fm string, params ...interface{}) {
	fmt.Fprintf(os.Stderr, fm+"\n", params...)
}

func exitError(code int, fm string, params ...interface{}) {
	errorf(fm, params...)
	os.Exit(code)
}

func getEncryptPassword(ctx *cli.Context) []byte {

	cmdLineValue := ctx.GlobalString("password")
	if cmdLineValue != "" {
		return []byte(cmdLineValue)
	}

	fromEnv := os.Getenv(inzure.KeyEnvironmentalVariableName)
	if fromEnv != "" {
		return []byte(fromEnv)
	}

	if err := DoSetPassword(ctx); err != nil {
		exitError(1, err.Error())
	}

	return []byte(ctx.GlobalString("password"))
}

func DoSetPassword(ctx *cli.Context) error {
	rl, err := readline.New("")
	if err != nil {
		return err
	}
	pass1, err := rl.ReadPassword("Encryption password: ")
	if err != nil {
		return err
	}
	pass2, err := rl.ReadPassword("Confirm: ")
	if err != nil {
		return err
	}
	if bytes.Compare(pass1, pass2) != 0 {
		return errors.New("passwords don't match")
	}
	return ctx.GlobalSet("password", string(pass1))
}
