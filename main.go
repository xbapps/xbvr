package main

//go:generate fileb0x .assets.toml

import (
	"fmt"
	"github.com/cld9x/xbvr/command"
	"github.com/cld9x/xbvr/xbase"
	"gopkg.in/urfave/cli.v1"
	"os"
)

var version = "CURRENT"
var commit = "HEAD"
var branch = "master"
var date = "moment ago"

func main() {
	app := cli.NewApp()
	app.Name = "xbvr"
	app.UsageText = "xbvr command [command options]"
	app.Version = fmt.Sprintf("%s (%s)", version, branch)
	app.Commands = command.GetCommands()

	app.Before = func(c *cli.Context) error {
		return nil
	}

	app.Action = func(c *cli.Context) error {
		xbase.StartServer()
		return nil
	}

	app.Run(os.Args)
}
