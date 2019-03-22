package main

import (
	"encoding/json"
	"os"
)

type Query struct {
	QS         string `json:"q"`
	PluginFile string `json:"plugin_file"`
}

type Config struct {
	Queries []Query `json:"queries"`
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
