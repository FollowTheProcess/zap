package cmd

import (
	"go.followtheprocess.codes/cli"
	"go.followtheprocess.codes/zap/internal/zap"
)

// export returns the zap export subcommand.
func export() (*cli.Command, error) {
	var options zap.ExportOptions

	// TODO(@FollowTheProcess): I think requests should just be a slice
	// so you can specify 1 or more requests to export

	// TODO(@FollowTheProcess): Support postman JSON, curl snippets etc.

	return cli.New(
		"export",
		cli.Short("Export http request files to various other formats"),
		cli.RequiredArg("file", "Path to the .http file"),
		cli.OptionalArg("request", "Name of a specific request", "all"),
		cli.Flag(&options.Debug, "debug", 'd', false, "Enable debug logging"),
		cli.Flag(&options.Format, "format", 'f', "", "Format to transform the http file into"),
		cli.Run(func(cmd *cli.Command, args []string) error {
			app := zap.New(options.Debug, cmd.Stdout(), cmd.Stderr())
			return app.Export(cmd.Arg("file"), cmd.Arg("request"), options)
		}),
	)
}
