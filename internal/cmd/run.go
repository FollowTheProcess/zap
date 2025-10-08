package cmd

import (
	"go.followtheprocess.codes/cli"
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

// run returns the zap do subcommand.
func run() (*cli.Command, error) {
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
		cli.Flag(&options.Output, "output", 'o', "", "Name of a file to save the response"),
		cli.Flag(&options.Debug, "debug", 'd', false, "Enable debug logging"),
		cli.Run(func(cmd *cli.Command, args []string) error {
			app := zap.New(options.Debug, cmd.Stdout(), cmd.Stderr())
			return app.Run(cmd.Arg("file"), args[1:], options)
		}),
	)
}
