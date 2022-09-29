package internal

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"reflect"

	"github.com/CarveSystems/inzure/pkg/inzure"
	"github.com/urfave/cli"
)

var (
	SearchIQS         string
	SubscriptionFiles cli.StringSlice
	NoLegacy          = false
)

var CmdSearchFlags = []cli.Flag{
	cli.StringSliceFlag{
		Name:   "f",
		EnvVar: inzure.EnvSubscriptionJSON,
		Usage:  "Input inzure JSON files",
		Value:  &SubscriptionFiles,
	},
	cli.BoolFlag{
		Name:        "no-legacy",
		Usage:       "Do not fall back to legacy output when only one file is provided. This standardizes the command output",
		Destination: &NoLegacy,
	},
	OutputFileFlag,
	BatchFlag,
}

func CmdSearch(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) == 0 {
		exitError(1, "need to pass a search string")
	}

	if Batch {
		files, err := inzure.BatchFilesFromEnv()
		if err != nil {
			exitError(1, "failed to get batch files: %v", err)
		}
		SubscriptionFiles = append(SubscriptionFiles, files...)
	}

	if SubscriptionFiles == nil || len(SubscriptionFiles) == 0 {
		if Batch {
			exitError(1, "batch file %s doesn't contain any subscription files", inzure.EnvSubscriptionBatchFiles)
		}
		exitError(1, "need to specify an input inzure JSON with -f")
	}

	SearchIQS = args[0]
	subscriptionFiles := SubscriptionFiles

	if !NoLegacy && len(subscriptionFiles) == 1 {
		cmdSearchSingleFile(ctx, subscriptionFiles[0])
		return
	}
	results := make(map[string]*json.RawMessage)
	var buf bytes.Buffer
	for _, subFile := range subscriptionFiles {
		buf.Reset()
		subID := doSearch(ctx, subFile, &buf)
		into := new(json.RawMessage)
		err := json.Unmarshal(buf.Bytes(), into)
		if err != nil {
			exitError(1, err.Error())
		}
		results[subID] = into
	}

	out := getOutputFile()
	if f, is := out.(*os.File); is {
		defer f.Close()
	}
	err := json.NewEncoder(out).Encode(results)
	if err != nil {
		exitError(1, err.Error())
	}
}

func doSearch(ctx *cli.Context, inputFile string, out io.Writer) string {
	if out == nil {
		out = os.Stdout
	}
	sub := getSubscriptionForFile(ctx, inputFile, nil)

	v, err := sub.ReflectFromQueryString(SearchIQS)
	if err != nil {
		le, is := err.(inzure.LexError)
		if is {
			exitError(1, le.ErrorWithHint())
		}
		exitError(1, err.Error())
	}

	if v.Kind() == reflect.Ptr && (v.IsNil() || v.Elem().IsNil()) {
		_, err = out.Write([]byte("[]\n"))
	} else {
		err = json.NewEncoder(out).Encode(v.Interface())

	}
	if err != nil {
		exitError(1, err.Error())
	}
	return sub.ID
}

func getOutputFile() io.Writer {
	var err error
	var out io.Writer
	if OutputFile != "" {
		out, err = os.OpenFile(OutputFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
		if err != nil {
			exitError(1, err.Error())
		}
	} else {
		out = os.Stdout
	}
	return out
}

func cmdSearchSingleFile(ctx *cli.Context, inputFile string) {
	out := getOutputFile()
	if f, is := out.(*os.File); is {
		defer f.Close()
	}
	doSearch(ctx, inputFile, out)
}
