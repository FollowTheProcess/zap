package cmd

import (
	"go.followtheprocess.codes/cli"
	"go.followtheprocess.codes/zap/internal/zap"
)

// buildImport returns the zap import subcommand.
func buildImport() (*cli.Command, error) {
	var options zap.ImportOptions

	// TODO(@FollowTheProcess): I think requests should just be a slice
	// so you can specify 1 or more requests to import

	return cli.New(
		"import",
		cli.Short("Import http requests in other formats to .http"),
		cli.RequiredArg("file", "Path to a file containing the import data"),
		cli.Flag(&options.Debug, "debug", 'd', false, "Enable debug logging"),
		cli.Flag(&options.Format, "format", 'f', "", "Format of the data to import"),
		cli.Run(func(cmd *cli.Command, args []string) error {
			app := zap.New(options.Debug, cmd.Stdout(), cmd.Stderr())
			return app.Import(cmd.Arg("file"), options)
		}),
	)
}
