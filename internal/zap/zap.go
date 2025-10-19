// Package zap implements the functionality of the program, the CLI in package cmd is simply the
// entrypoint to exported functions and methods in this package.
package zap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"go.followtheprocess.codes/log"
	"go.followtheprocess.codes/msg"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/syntax/parser"
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
func (z Zap) Hello(ctx context.Context) {
	fmt.Fprintln(z.stdout, "Hello from Zap!")
	z.logger.Debug("This is a debug log", "cheese", "brie")
}

// RunOptions are the options passed to the run subcommand.
type RunOptions struct {
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

// Run implements the run subcommand.
func (z Zap) Run(ctx context.Context, file string, requests []string, options RunOptions) error {
	if len(requests) == 0 {
		fmt.Fprintf(z.stdout, "Executing all requests in file: %s\n", file)
	} else {
		fmt.Fprintf(z.stdout, "Executing specific request(s) %v in file: %s\n", requests, file)
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
func (z Zap) Check(ctx context.Context, path string, handler syntax.ErrorHandler, options CheckOptions) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("could not get path info: %w", err)
	}

	var paths []string

	if info.IsDir() {
		err = filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if filepath.Ext(path) == ".http" {
				paths = append(paths, path)
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("could not walk %s: %w", path, err)
		}
	} else {
		paths = []string{path}
	}

	var errs []error

	// TODO(@FollowTheProcess): Do this concurrently, each file is self contained
	// this is embarrassingly parallel.
	for _, path := range paths {
		if err := z.checkFile(path, handler); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// checkFile runs a parse check on a single file.
func (z Zap) checkFile(path string, handler syntax.ErrorHandler) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("could not open file: %w", err)
	}
	defer file.Close()

	p, err := parser.New(path, file, handler)
	if err != nil {
		return fmt.Errorf("could not initialise the parser: %w", err)
	}

	// We don't actually care about the result, just that it parses
	_, err = p.Parse()
	if err != nil {
		return err
	}

	msg.Fsuccess(z.stdout, "%s is valid", path)

	return nil
}
