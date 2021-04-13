package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"text/template"

	"github.com/CarveSystems/inzure/pkg/inzure"
)

type cmdArgs struct {
	inzureFile string
	configFile string
}

func slackposter(cli *CLI, args *cmdArgs) int {
	conf, err := loadConfig(args.configFile)
	if err != nil {
		fmt.Fprintf(
			cli.errStream,
			"failed to load config from %s: %v\n",
			args.configFile, err,
		)
		return ExitCodeError
	}
	sub, err := inzure.SubscriptionFromFile(args.inzureFile)
	if err != nil {
		fmt.Fprintf(cli.errStream, "bad inzure JSON file %s: %v\n", args.inzureFile, err)
		return ExitCodeError
	}

	var buf bytes.Buffer
	sendReport := false

	_, err = buf.WriteString(
		fmt.Sprintf(
			"inzure has found the following problems in subscription %s:\n\n",
			sub.ID,
		),
	)
	if err != nil {
		fmt.Fprintf(cli.errStream, "failed to write to string: %v\n", err)
		return ExitCodeError
	}

	for _, q := range conf.Queries {
		val, err := sub.ReflectFromQueryString(q.QS)
		if err != nil {
			fmt.Fprintf(cli.errStream, "failed to execute query %s: %v\n", q.QS, err)
			return ExitCodeError
		}
		if val.IsNil() || !val.IsValid() {
			continue
		}
		sendReport = true
		t, err := template.ParseFiles(q.TemplateFile)
		if err != nil {
			fmt.Fprintf(cli.errStream, "bad template %s: %v\n", q.TemplateFile, err)
			return ExitCodeError
		}
		err = t.Execute(&buf, val.Interface())
		if err != nil {
			fmt.Fprintf(
				cli.errStream, "failed to execute template %s: %v\n",
				q.TemplateFile, err,
			)
			return ExitCodeError
		}
	}
	if sendReport {
		err = postMessage(conf.SlackURL, buf.String())
		if err != nil {
			fmt.Fprintf(cli.errStream, "failed to post message: %v\n", err)
			return ExitCodeError
		}
	}
	return ExitCodeOK
}

func postMessage(url string, s string) error {
	m := make(map[string]string)
	m["text"] = s
	val, err := json.Marshal(m)
	if err != nil {
		return err
	}
	_, err = http.Post(url, "application/json", bytes.NewReader(val))
	return err
}
