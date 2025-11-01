package cmd

import (
	"context"

	"go.followtheprocess.codes/cli"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/zap"
)

const checkLong = `
The path argument may be a directory or a file.

If it is the name of a .http file, then this file alone is checked
for validity.

If it is a directory, this directory is scanned recursively for all
files with the '.http' extension and any matching files will be validated.
`

// check returns the check subcommand.
func check(ctx context.Context) func() (*cli.Command, error) {
	return func() (*cli.Command, error) {
		var options zap.CheckOptions

		return cli.New(
			"check",
			cli.Short("Check http files for syntax errors"),
			cli.Long(checkLong),
			cli.OptionalArg("path", "Path to check, may be directory or file", "."),
			cli.Flag(&options.Debug, "debug", 'd', false, "Enable debug logging"),
			cli.Run(func(cmd *cli.Command, args []string) error {
				app := zap.New(options.Debug, version, cmd.Stdin(), cmd.Stdout(), cmd.Stderr())
				return app.Check(ctx, cmd.Arg("path"), syntax.PrettyConsoleHandler(cmd.Stderr()), options)
			}),
		)
	}
}
