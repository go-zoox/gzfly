package main

import (
	"github.com/go-zoox/cli"
	"github.com/go-zoox/gzfly/command"
)

func main() {
	app := cli.NewMultipleProgram(&cli.MultipleProgramConfig{
		Name:    "gzfly",
		Usage:   "gzfly is a program for proxy.",
		Version: Version,
	})

	command.RegisterClient(app)
	command.RegisterServer(app)

	app.Run()
}
