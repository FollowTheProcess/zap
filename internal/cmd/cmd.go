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

// TODO(@FollowTheProcess): A transform subcommand that can take a valid file and turn each request
// (or a specific one) into a curl snippet, postman request etc.

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
		cli.Example("Execute all the requests in a specific file", "zap do ./demo.http"),
		cli.Example(
			"Execute a single request from a file, setting a bunch of options",
			"zap do ./demo.http --request MyRequest --timeout 10s --no-redirect",
		),
		cli.Example("Check for syntax errors in a file", "zap check ./demo.http"),
		cli.Example("Check for syntax errors in multiple files (recursively)", "zap check ./examples"),
		cli.Allow(cli.NoArgs()),
		cli.Flag(&debug, "debug", 'd', false, "Enable debug logs"),
		cli.SubCommands(do),
		cli.Run(func(cmd *cli.Command, args []string) error {
			app := zap.New(debug, os.Stdout, os.Stderr)
			app.Hello()

			return nil
		}),
	)
}

const doLong = `
The request headers, body and other settings will be taken from the
file but may be overridden by the use of command line flags like
'--timeout' etc.

Responses can be saved to a file with the '--output' flag.
`

// do returns the zap do subcommand.
func do() (*cli.Command, error) {
	var options zap.DoOptions

	return cli.New(
		"do",
		cli.Short("Execute a http request from a file"),
		cli.Long(doLong),
		cli.RequiredArg("file", "Path to the .http file"),
		cli.OptionalArg("request", "Name of a specific request", "all"),
		cli.Flag(&options.Timeout, "timeout", cli.NoShortHand, zap.DefaultTimeout, "Timeout for the request"),
		cli.Flag(
			&options.ConnectionTimeout,
			"connection-timeout",
			cli.NoShortHand,
			zap.DefaultConnectionTimeout,
			"Connection timeout for the request",
		),
		cli.Flag(&options.NoRedirect, "no-redirect", cli.NoShortHand, false, "Disable following redirects"),
		cli.Flag(&options.Output, "output", 'o', "", "Name of a file to save the response"),
		cli.Flag(&options.Verbose, "verbose", 'v', false, "Enable debug logging"),
		cli.Run(func(cmd *cli.Command, args []string) error {
			app := zap.New(options.Verbose, cmd.Stdout(), cmd.Stderr())
			return app.Do(cmd.Arg("file"), cmd.Arg("request"), options)
		}),
	)
}
