// zap is a command line http file toolkit.
package main

import (
	"context"
	"os"
	"os/signal"

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
	ctx := context.Background()

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer cancel()

	cli, err := cmd.Build(ctx)
	if err != nil {
		return err
	}

	return cli.Execute()
}
