// zap is a command line http file toolkit.
package main

import (
	"context"
	"os"

	"go.followtheprocess.codes/msg"
	"go.followtheprocess.codes/zap/internal/cmd"
)

func main() {
	if err := run(); err != nil {
		msg.Err(err)
		os.Exit(1)
	}
}

func run() error {
	cli, err := cmd.Build(context.Background())
	if err != nil {
		return err
	}

	return cli.Execute()
}
