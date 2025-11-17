package cmd

import (
	"context"

	"go.followtheprocess.codes/cli"
	"go.followtheprocess.codes/cli/flag"
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
func run() (*cli.Command, error) {
	var options zap.RunOptions

	return cli.New(
		"run",
		cli.Short("Execute one or more http requests from a file"),
		cli.Long(runLong),
		cli.Arg(&options.File, "file", "Path to the .http file"),
		cli.Flag(
			&options.Timeout,
			"timeout",
			flag.NoShortHand,
			"Timeout for the request",
			cli.FlagDefault(zap.DefaultTimeout),
		),
		cli.Flag(
			&options.ConnectionTimeout,
			"connection-timeout",
			flag.NoShortHand,
			"Connection timeout for the request",
			cli.FlagDefault(zap.DefaultConnectionTimeout),
		),
		cli.Flag(
			&options.OverallTimeout,
			"overall-timeout",
			flag.NoShortHand,
			"Overall timeout for the execution",
			cli.FlagDefault(zap.DefaultOverallTimeout),
		),
		cli.Flag(&options.NoRedirect, "no-redirect", flag.NoShortHand, "Disable following redirects"),
		cli.Flag(&options.Output, "output", 'o', "Output format, one of (stdout|json|yaml)", cli.FlagDefault("stdout")),
		cli.Flag(&options.Requests, "request", 'r', "Name(s) of requests to execute"),
		cli.Flag(&options.Verbose, "verbose", 'v', "Show additional response data"),
		cli.Flag(&options.Debug, "debug", 'd', "Enable debug logging"),
		cli.Run(func(ctx context.Context, cmd *cli.Command) error {
			app := zap.New(options.Debug, version, cmd.Stdin(), cmd.Stdout(), cmd.Stderr())
			return app.Run(ctx, zap.PrettyConsoleHandler(cmd.Stderr()), options)
		}),
	)
}
