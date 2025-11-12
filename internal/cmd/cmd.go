// Package cmd implements zap's CLI.
package cmd

import (
	"context"
	"os"

	"go.followtheprocess.codes/cli"
	"go.followtheprocess.codes/zap/internal/zap"
)

//nolint:gochecknoglobals // These have to be here
var (
	version = "dev"
	commit  = ""
	date    = ""
)

// TODO(@FollowTheProcess): Also, the opposite! An import subcommand that can take curl snippets, postman
// collections etc. and convert them to .http files

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
		cli.Example("Execute all the requests in a specific file", "zap run ./demo.http"),
		cli.Example(
			"Execute a single request from a file, setting a bunch of options",
			"zap run ./demo.http --request MyRequest --timeout 10s --no-redirect",
		),
		cli.Example("Check for syntax errors in a file", "zap check ./demo.http"),
		cli.Example("Check for syntax errors in multiple files (recursively)", "zap check ./examples"),
		cli.Flag(&debug, "debug", 'd', "Enable debug logs"),
		cli.SubCommands(
			run,
			check,
			export,
			test,
		),
		cli.Run(func(ctx context.Context, cmd *cli.Command) error {
			app := zap.New(debug, version, os.Stdin, os.Stdout, os.Stderr)
			app.Hello(ctx)

			return nil
		}),
	)
}
