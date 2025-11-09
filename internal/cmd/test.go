package cmd

import (
	"context"

	"go.followtheprocess.codes/cli"
	"go.followtheprocess.codes/cli/flag"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/zap"
)

// TODO(@FollowTheProcess): Work this out a bit more, would be good to get some sort of assertion
// like functionality, checking status code, inspecting body elements with JSONPath etc.
//
// Idea: The response ref is a yaml file containing two "files", the top one with metadata about the test,
// this is where assertions could go. The bottom one containing the JSON of
// the response, since all JSON is valid yaml, this should work?

const testLong = `
The test command executes a collection of http requests/files as tests.

HTTP requests may define a response reference in the form '<> response.json', in these
cases, the request will be run as a test with the response ref file serving as the
golden file. If the fetched response does not match the reference, the test will fail.

Path is a .http file or a directory containing .http files, in the latter case, the directory
is recursed and all .http files collected for testing.

In test mode, the responses are typically hidden (unless the test fails) in favour of
a compact summary. This can be enhanced with the '--verbose' flag.
`

// test returns the zap test subcommand.
func test(ctx context.Context) func() (*cli.Command, error) {
	return func() (*cli.Command, error) {
		var options zap.TestOptions

		return cli.New(
			"test",
			cli.Short("Run http requests as tests"),
			cli.Long(testLong),
			cli.Arg(&options.Path, "path", "Path to test, may be directory or file", cli.ArgDefault(".")),
			cli.Flag(&options.Timeout, "timeout", flag.NoShortHand, zap.DefaultTimeout, "Timeout for the request"),
			cli.Flag(
				&options.ConnectionTimeout,
				"connection-timeout",
				flag.NoShortHand,
				zap.DefaultConnectionTimeout,
				"Connection timeout for the request",
			),
			cli.Flag(
				&options.OverallTimeout,
				"overall-timeout",
				flag.NoShortHand,
				zap.DefaultOverallTimeout,
				"Overall timeout for the execution",
			),
			cli.Flag(&options.NoRedirect, "no-redirect", flag.NoShortHand, false, "Disable following redirects"),
			cli.Flag(&options.Requests, "request", 'r', nil, "Name(s) of requests to test"),
			cli.Flag(&options.Verbose, "verbose", 'v', false, "Show additional test information"),
			cli.Flag(&options.Debug, "debug", 'd', false, "Enable debug logging"),
			cli.Run(func(cmd *cli.Command) error {
				app := zap.New(options.Debug, version, cmd.Stdin(), cmd.Stdout(), cmd.Stderr())
				return app.Test(ctx, syntax.PrettyConsoleHandler(cmd.Stderr()), options)
			}),
		)
	}
}
