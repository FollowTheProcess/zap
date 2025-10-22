package cmd

import (
	"context"

	"go.followtheprocess.codes/cli"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/zap"
)

const runLong = `
The request headers, body and other settings will be taken from the
file but some things may be overridden by the use of command line flags like
'--timeout' etc.

The '--connection-timeout' and '--timeout' flags apply to individual requests,
if you're executing multiple requests and want an overall timeout for
the entire collection, pass '--overall--timeout'.

Responses can be displayed in different formats with the '--output' flag. By default
responses a printed to stdout, but may also be serialized as json or yaml by passing
'--output json'.
`

// run returns the zap run subcommand.
func run(ctx context.Context) func() (*cli.Command, error) {
	return func() (*cli.Command, error) {
		var options zap.RunOptions

		// TODO(@FollowTheProcess): A way of filtering/selecting requests within the file, options are:
		// 1) A --request flag to specify a single request, can be repeated
		// 2) A --filter flag that takes like a glob pattern e.g. --filter 'get*' would do every request who's name starts with 'get'
		// 3) A --pattern (or --filter) flag that takes a full regex pattern, only matches get run
		//
		// Maybe we do all?

		// TODO(@FollowTheProcess): Can we syntax highlight the body based on Content-Type?

		return cli.New(
			"run",
			cli.Short("Execute one or more http requests from a file"),
			cli.Long(runLong),
			cli.RequiredArg("file", "Path to the .http file"),
			cli.Flag(&options.Timeout, "timeout", cli.NoShortHand, zap.DefaultTimeout, "Timeout for the request"),
			cli.Flag(
				&options.ConnectionTimeout,
				"connection-timeout",
				cli.NoShortHand,
				zap.DefaultConnectionTimeout,
				"Connection timeout for the request",
			),
			cli.Flag(
				&options.OverallTimeout,
				"overall-timeout",
				cli.NoShortHand,
				zap.DefaultOverallTimeout,
				"Overall timeout for the execution",
			),
			cli.Flag(&options.NoRedirect, "no-redirect", cli.NoShortHand, false, "Disable following redirects"),
			cli.Flag(&options.Output, "output", 'o', "stdout", "Output format, one of 'stdout', 'json' or 'yaml'"),
			cli.Flag(&options.Requests, "request", 'r', nil, "Name(s) of requests to execute"),
			cli.Flag(&options.Verbose, "verbose", 'v', false, "Show additional response data"),
			cli.Flag(&options.Debug, "debug", 'd', false, "Enable debug logging"),
			cli.Run(func(cmd *cli.Command, args []string) error {
				app := zap.New(options.Debug, version, cmd.Stdout(), cmd.Stderr())
				return app.Run(ctx, cmd.Arg("file"), args[1:], syntax.PrettyConsoleHandler(cmd.Stderr()), options)
			}),
		)
	}
}
