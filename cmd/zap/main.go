package main

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"go.followtheprocess.codes/zap/internal/cmd"
)

func main() {
	if err := fang.Execute(
		context.Background(),
		cmd.Build(),
		fang.WithNotifySignal(os.Interrupt, os.Kill),
		fang.WithColorSchemeFunc(fang.AnsiColorScheme),
	); err != nil {
		os.Exit(1)
	}
}
