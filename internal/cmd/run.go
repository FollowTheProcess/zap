package cmd

import (
	"context"

	"go.followtheprocess.codes/cli"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/zap"
)

const runLong = `
The run command executes one or more http requests from a file, by default all
the requests in the target file will be run in order of their definition.

The '--request' flag may be used to filter the list of requests to execute.

Configuration such as timeouts, redirects etc. are set in the .http file, or assume their
default values if not specified. However, they can be overridden by flags with flags
taking precedence over values defined in the file.

The '--connection-timeout' and '--timeout' flags apply to individual requests,
if you're executing multiple requests and want an overall timeout for
the entire collection, pass '--overall--timeout'.

Responses can be displayed in different formats with the '--output' flag. By default
responses are printed in a user-friendly format to stdout, but may also be serialized as
json by passing '--output json'.
`

// run returns the zap run subcommand.
func run(ctx context.Context) func() (*cli.Command, error) {
	return func() (*cli.Command, error) {
		var options zap.RunOptions

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
			cli.Flag(&options.Output, "output", 'o', "stdout", "Output format, one of (stdout|json|yaml)"),
			cli.Flag(&options.Requests, "request", 'r', nil, "Name(s) of requests to execute"),
			cli.Flag(&options.Verbose, "verbose", 'v', false, "Show additional response data"),
			cli.Flag(&options.Debug, "debug", 'd', false, "Enable debug logging"),
			cli.Run(func(cmd *cli.Command, args []string) error {
				app := zap.New(options.Debug, version, cmd.Stdin(), cmd.Stdout(), cmd.Stderr())
				return app.Run(ctx, cmd.Arg("file"), syntax.PrettyConsoleHandler(cmd.Stderr()), options)
			}),
		)
	}
}
