package command

import (
	"gopkg.in/urfave/cli.v1"
)

var commands []cli.Command

type Commander interface {
	Execute(c *cli.Context)
}

func RegisterCommand(command cli.Command) {
	commands = append(commands, command)
}

func GetCommands() []cli.Command {
	return commands
}
