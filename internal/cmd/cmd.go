// Package cmd implements zap's CLI.
package cmd

import (
	"context"
	"os"

	"go.followtheprocess.codes/cli"
	"go.followtheprocess.codes/zap/internal/zap"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

// TODO(@FollowTheProcess): A transform subcommand that can take a valid file and turn each request
// (or a specific one) into a curl snippet, postman request etc.

// TODO(@FollowTheProcess): Also, the opposite! An import subcommand that can take curl snippets, postman
// collections etc. and convert them to .http files

// Build builds and returns the zap CLI.
func Build(ctx context.Context) (*cli.Command, error) {
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
		cli.Allow(cli.NoArgs()),
		cli.Flag(&debug, "debug", 'd', false, "Enable debug logs"),
		cli.SubCommands(run(ctx), check(ctx)),
		cli.Run(func(cmd *cli.Command, args []string) error {
			app := zap.New(debug, os.Stdout, os.Stderr)
			app.Hello(ctx)

			return nil
		}),
	)
}
