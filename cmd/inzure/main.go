package main

import (
	"os"

	"github.com/CarveSystems/inzure/cmd/inzure/internal"
	"github.com/CarveSystems/inzure/cmd/inzure/internal/autocomplete"
	"github.com/urfave/cli"
)

func setPassword(ctx *cli.Context) error {
	if ctx.GlobalBool("p") {
		return internal.DoSetPassword(ctx)
	}
	return nil
}

func main() {

	if autocomplete.IsCompletion() {
		internal.SetupAutoCompletions()
		autocomplete.DoAutoComplete(Commands, GlobalFlags)
		return
	}

	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = Name
	app.Version = Version
	app.Usage = ""
	app.Author = "Danny Rosseau"
	app.Email = "danny.rosseau@carvesystems.com"

	app.Flags = GlobalFlags
	app.Commands = Commands
	app.CommandNotFound = CommandNotFound

	app.Before = setPassword

	app.Run(os.Args)
}
