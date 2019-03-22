package main

import (
	"encoding/json"
	"os"
)

type Query struct {
	QS           string `json:"q"`
	TemplateFile string `json:"template_file"`
}

type Config struct {
	Queries  []Query `json:"queries"`
	SlackURL string  `json:"slack_url"`
}

func loadConfig(fname string) (c Config, err error) {
	var f *os.File
	f, err = os.Open(fname)
	if err != nil {
		return
	}
	err = json.NewDecoder(f).Decode(&c)
	return
}
