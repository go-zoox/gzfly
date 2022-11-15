package main

import (
	"github.com/go-zoox/cli"
	"github.com/go-zoox/gzfly/command"
)

func main() {
	app := cli.NewMultipleProgram(&cli.MultipleProgramConfig{
		Name:    "multiple",
		Usage:   "multiple is a program that has multiple commands.",
		Version: Version,
	})

	command.RegisterClient(app)
	command.RegisterServer(app)

	app.Run()
}
