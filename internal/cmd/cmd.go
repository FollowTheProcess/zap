// Package cmd implements zap's CLI.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"go.followtheprocess.codes/zap/internal/zap"
)

const example = `
# Pick HTTP files and requests interactively
zap

# Execute all the requests in a specific file
zap do ./demo.http

# Execute a single request from a file, setting a bunch of options
zap do ./demo.http --request MyRequest --timeout 10s --no-redirect

# Check for syntax errors in a file
zap check ./demo.http

# Check for syntax errors in multiple files (recursively)
zap check ./examples`

// Build builds and returns the zap CLI.
func Build() *cobra.Command {
	var debug bool

	cmd := &cobra.Command{
		Use:     "zap [command] args... [flags]",
		Short:   "A command line .http file toolkit",
		Example: example,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := zap.New(debug, os.Stdout, os.Stderr)
			app.Hello()
			return nil
		},
	}

	cmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logs")

	return cmd
}
