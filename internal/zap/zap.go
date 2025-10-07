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
	// DefaultOverallTimeout is the default amount of time allowed for the entire
	// execution. Typically only used when executing multiple requests as a collection.
	DefaultOverallTimeout = 1 * time.Minute
	// DefaultTimeout is the default amount of time allowed for the entire request/response
	// cycle for a single request.
	DefaultTimeout = 30 * time.Second

	// DefaultConnectionTimeout is the default amount of time allowed for the HTTP connection/TLS handshake
	// for a single request.
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

	// OverallTimeout is the overall timeout, used when running multiple requests.
	OverallTimeout time.Duration

	// NoRedirect, if true, disables following http redirects.
	NoRedirect bool

	// Debug enables debug logging.
	Debug bool
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

// CheckOptions are the options passed to the check subcommand.
type CheckOptions struct {
	// Debug enables debug logging.
	Debug bool
}

// Check implements the check subcommand.
func (z Zap) Check(path string, options CheckOptions) error {
	fmt.Fprintf(z.stdout, "Checking %s for syntax errors\n", path)
	fmt.Fprintf(z.stdout, "Options: %+v\n", options)

	return nil
}

// ExportOptions are the options passed to the export subcommand.
type ExportOptions struct {
	// Format specifies the format for the export
	Format string

	// Debug enables debug logging.
	Debug bool
}

// Export implements the export subcommand.
func (z Zap) Export(file, request string, options ExportOptions) error {
	fmt.Fprintf(z.stdout, "Exporting request %q from file %s\n", request, file)
	fmt.Fprintf(z.stdout, "Options: %+v\n", options)

	return nil
}

// ImportOptions are the options passed to the import subcommand.
type ImportOptions struct {
	// Format specifies the format for the import
	Format string

	// Debug enables debug logging.
	Debug bool
}

// Import implements the import subcommand.
func (z Zap) Import(file string, options ImportOptions) error {
	fmt.Fprintf(z.stdout, "Importing HTTP data from %s\n", file)
	fmt.Fprintf(z.stdout, "Options: %+v\n", options)

	return nil
}
