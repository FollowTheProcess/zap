package cmd

import (
	"context"

	"go.followtheprocess.codes/cli"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/zap"
)

const runLong = `
The request headers, body and other settings will be taken from the
file but may be overridden by the use of command line flags like
'--timeout' etc.

The '--connection-timeout' and '--timeout' flags apply to individual requests,
if you're executing multiple requests and want an overall timeout for
the entire collection, pass '--overall--timeout'.

Responses can be saved to a file with the '--output' flag. This may
also be specified in the file with '> ./response.json'. If both are
used, the command line flag takes precedence.
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

		// TODO(@FollowTheProcess): A --verbose flag that shows the request headers and body in a similar way to
		// how the response is shown. Maybe don't show the request headers by default?

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
			cli.Flag(&options.Output, "output", 'o', "", "Name of a file to save the response"),
			cli.Flag(&options.Debug, "debug", 'd', false, "Enable debug logging"),
			cli.Run(func(cmd *cli.Command, args []string) error {
				app := zap.New(options.Debug, version, cmd.Stdout(), cmd.Stderr())
				return app.Run(ctx, cmd.Arg("file"), args[1:], syntax.PrettyConsoleHandler(cmd.Stderr()), options)
			}),
		)
	}
}
