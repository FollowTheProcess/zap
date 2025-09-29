// Package zap implements the functionality of the program, the CLI in package cmd is simply the
// entrypoint to exported functions and methods in this package.
package zap

import (
	"fmt"
	"io"
	"time"

	"go.followtheprocess.codes/log"
)

// HTTP config.
const (
	// DefaultTimeout is the default amount of time allowed for the entire request cycle.
	DefaultTimeout = 30 * time.Second

	// DefaultConnectionTimeout is the default amount of time allowed for the HTTP connection/TLS handshake.
	DefaultConnectionTimeout = 10 * time.Second
)

// Zap represents the zap program.
type Zap struct {
	stdout io.Writer   // Normal program output is written here
	stderr io.Writer   // Logs and errors are written here
	logger *log.Logger // The logger for the application
}

// New returns a new [Zap].
func New(debug bool, stdout, stderr io.Writer) Zap {
	level := log.LevelInfo
	if debug {
		level = log.LevelDebug
	}

	logger := log.New(
		stderr,
		log.WithLevel(level),
		log.Prefix("zap"),
	)

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

// DoOptions are the options passed to the do subcommand.
type DoOptions struct {
	// Output is the name of a file in which to save the response, if empty,
	// the response is printed to stdout.
	Output string

	// Timeout is the overall per-request timeout.
	Timeout time.Duration

	// ConnectionTimeout is the per-request connection timeout.
	ConnectionTimeout time.Duration

	// NoRedirect, if true, disables following http redirects.
	NoRedirect bool

	// Verbose enables debug logging.
	Verbose bool
}

// Do implements the do subcommand.
func (z Zap) Do(file, request string, options DoOptions) error {
	if request == "all" {
		fmt.Fprintf(z.stdout, "Executing all requests in file: %s\n", file)
	} else {
		fmt.Fprintf(z.stdout, "Executing specific request %q in file: %s\n", request, file)
	}

	fmt.Fprintf(z.stdout, "Options: %+v\n", options)

	return nil
}
