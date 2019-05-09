package command

import (
	"github.com/cld9x/xbvr/xbase"
	"gopkg.in/urfave/cli.v1"
)

func ActionRun(c *cli.Context) {
	xbase.StartServer()
}

func init() {
	RegisterCommand(cli.Command{
		Name:   "run",
		Usage:  "run xbvr",
		Action: ActionRun,
	})
}
