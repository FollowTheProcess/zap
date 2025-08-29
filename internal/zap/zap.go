// Package zap implements the functionality of the program, the CLI in package cmd is simply the
// entrypoint to exported functions and methods in this package.
package zap

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/log/v2"
)

// Zap represents the zap program.
type Zap struct {
	stdout io.Writer   // Normal program output is written here
	stderr io.Writer   // Logs and errors are written here
	logger *log.Logger // The logger for the application
}

// New returns a new [Zap].
func New(debug bool, stdout, stderr io.Writer) Zap {
	level := log.InfoLevel
	if debug {
		level = log.DebugLevel
	}

	logger := log.NewWithOptions(stderr, log.Options{
		TimeFormat:      time.RFC3339Nano,
		Level:           level,
		Prefix:          "zap",
		ReportTimestamp: true,
	})

	logger.SetStyles(defaultLogStyles())

	return Zap{
		stdout: stdout,
		stderr: stderr,
		logger: logger,
	}
}

// Hello is a placeholder method for wiring up the CLI.
func (z Zap) Hello() {
	fmt.Fprintln(z.stdout, "Hello from Zap!")
	z.logger.Debug("This is a debug log", "cheese", "brie")
}
