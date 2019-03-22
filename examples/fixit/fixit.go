package main

import (
	"errors"
	"fmt"
	"plugin"
	"reflect"

	"github.com/CarveSystems/inzure"
)

type cmdArgs struct {
	inzureFile string
	configFile string
}

func fixit(cli *CLI, args *cmdArgs) int {
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
		fmt.Fprintf(cli.errStream, "bad inzure JSON file: %s\n", args.inzureFile)
		return ExitCodeError
	}

	for _, q := range conf.Queries {
		val, err := sub.ReflectFromQueryString(q.QS)
		if err != nil {
			fmt.Fprintf(cli.errStream, "failed to execute query %s: %v\n", q.QS, err)
			return ExitCodeError
		}
		if !val.IsValid() {
			continue
		}
		for val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		err = runFixitPlugin(val.Interface(), q.PluginFile)
		if err != nil {
			fmt.Fprintf(
				cli.errStream,
				"failed to fix issue using plugin %s: %v\n",
				q.PluginFile, err,
			)
			return ExitCodeError
		}
	}
	return ExitCodeOK
}

func runFixitPlugin(v interface{}, pluginFile string) error {
	p, err := plugin.Open(pluginFile)
	if err != nil {
		return err
	}
	sym, err := p.Lookup("FixitFunc")
	if err != nil {
		return err
	}
	f, ok := sym.(func(interface{}) error)
	if !ok {
		return errors.New("FixitFunc had the wrong signature")
	}
	return f(v)
}
