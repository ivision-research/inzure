package main

import (
	"bytes"
	"errors"
	"os"

	"github.com/CarveSystems/inzure/cmd/inzure/internal"
	"github.com/CarveSystems/inzure/cmd/inzure/internal/autocomplete"
	"github.com/chzyer/readline"
	"github.com/codegangsta/cli"
)

func setPassword(ctx *cli.Context) error {
	if ctx.GlobalBool("p") {
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
		ctx.GlobalSet("password", string(pass1))
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

	app.Before = cli.BeforeFunc(setPassword)

	app.Run(os.Args)
}
