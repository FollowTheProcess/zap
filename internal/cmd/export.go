package cmd

import (
	"context"

	"go.followtheprocess.codes/cli"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/zap"
)

// export returns the zap export subcommand.
func export() (*cli.Command, error) {
	var options zap.ExportOptions

	return cli.New(
		"export",
		cli.Short("Export a .http file to an alternative format"),
		cli.Arg(&options.File, "file", "Path to the .http file"),
		cli.Flag(
			&options.Format,
			"format",
			'f',
			"Export format, one of (json|curl|yaml|toml|postman)",
			cli.FlagDefault("json"),
		),
		cli.Flag(&options.Debug, "debug", 'd', "Enable debug logging"),
		cli.Run(func(ctx context.Context, cmd *cli.Command) error {
			app := zap.New(options.Debug, version, cmd.Stdin(), cmd.Stdout(), cmd.Stderr())
			return app.Export(ctx, syntax.PrettyConsoleHandler(cmd.Stderr()), options)
		}),
	)
}
