package cmd

import (
	"context"

	"go.followtheprocess.codes/cli"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/zap"
)

// export returns the zap export subcommand.
func export(ctx context.Context) func() (*cli.Command, error) {
	return func() (*cli.Command, error) {
		var options zap.ExportOptions

		return cli.New(
			"export",
			cli.Short("Export a .http file to an alternative format"),
			cli.RequiredArg("file", "Path to the .http file"),
			cli.Flag(
				&options.Format,
				"format",
				'f',
				"json",
				"Export format, one of (json|curl|yaml|toml|postman)",
			),
			cli.Flag(&options.Debug, "debug", 'd', false, "Enable debug logging"),
			cli.Run(func(cmd *cli.Command, args []string) error {
				app := zap.New(options.Debug, version, cmd.Stdin(), cmd.Stdout(), cmd.Stderr())
				return app.Export(ctx, cmd.Arg("file"), syntax.PrettyConsoleHandler(cmd.Stderr()), options)
			}),
		)
	}
}
