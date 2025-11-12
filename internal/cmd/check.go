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
func check() (*cli.Command, error) {
	var options zap.CheckOptions

	return cli.New(
		"check",
		cli.Short("Check http files for syntax errors"),
		cli.Long(checkLong),
		cli.Arg(&options.Path, "path", "The path to check", cli.ArgDefault(".")),
		cli.Flag(&options.Debug, "debug", 'd', "Enable debug logging"),
		cli.Run(func(ctx context.Context, cmd *cli.Command) error {
			app := zap.New(options.Debug, version, cmd.Stdin(), cmd.Stdout(), cmd.Stderr())
			return app.Check(ctx, syntax.PrettyConsoleHandler(cmd.Stderr()), options)
		}),
	)
}
