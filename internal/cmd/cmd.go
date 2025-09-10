// Package cmd implements zap's CLI.
package cmd

import (
	"os"

	"go.followtheprocess.codes/cli"
	"go.followtheprocess.codes/zap/internal/zap"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

// Build builds and returns the zap CLI.
func Build() (*cli.Command, error) {
	var debug bool

	return cli.New(
		"zap",
		cli.Short("A command line .http file toolkit"),
		cli.Version(version),
		cli.Commit(commit),
		cli.BuildDate(date),
		cli.Example("Pick .http files and requests interactively", "zap"),
		cli.Example("Execute all the requests in a specific file", "zap do ./demo.http"),
		cli.Example(
			"Execute a single request from a file, setting a bunch of options",
			"zap do ./demo.http --request MyRequest --timeout 10s --no-redirect",
		),
		cli.Example("Check for syntax errors in a file", "zap check ./demo.http"),
		cli.Example("Check for syntax errors in multiple files (recursively)", "zap check ./examples"),
		cli.Allow(cli.NoArgs()),
		cli.Flag(&debug, "debug", 'd', false, "Enable debug logs"),
		cli.Run(func(cmd *cli.Command, args []string) error {
			app := zap.New(debug, os.Stdout, os.Stderr)
			app.Hello()

			return nil
		}),
	)
}
