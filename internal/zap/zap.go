// Package zap implements the functionality of the program, the CLI in package cmd is simply the
// entrypoint to exported functions and methods in this package.
package zap

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"go.followtheprocess.codes/log"
	"go.followtheprocess.codes/zap/internal/spec"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/syntax/parser"
)

// Zap represents the zap program.
type Zap struct {
	stdin   io.Reader   // Program input (prompts) come from here
	stdout  io.Writer   // Normal program output is written here
	stderr  io.Writer   // Logs and errors are written here
	logger  *log.Logger // The logger for the application
	version string      // The app version
}

// New returns a new [Zap].
func New(debug bool, version string, stdin io.Reader, stdout, stderr io.Writer) Zap {
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
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		logger:  logger,
		version: version,
	}
}

// Hello is a placeholder method for wiring up the CLI.
func (z Zap) Hello(ctx context.Context) {
	fmt.Fprintln(z.stdout, "Hello from Zap!")
	z.logger.Debug("This is a debug log", slog.String("cheese", "brie"))
}

// parseFile reads a .http file, parses it and resolves it.
//
// Most operations begin by parsing the file so those steps are extracted here.
func (z Zap) parseFile(file string, handler syntax.ErrorHandler) (spec.File, error) {
	f, err := os.Open(file)
	if err != nil {
		return spec.File{}, fmt.Errorf("could not open file: %w", err)
	}
	defer f.Close()

	p, err := parser.New(file, f, handler)
	if err != nil {
		return spec.File{}, fmt.Errorf("could not initialise the parser: %w", err)
	}

	parsed, err := p.Parse()
	if err != nil {
		return spec.File{}, err
	}

	resolved, err := spec.ResolveFile(parsed)
	if err != nil {
		return spec.File{}, err
	}

	if resolved.ConnectionTimeout == 0 {
		resolved.ConnectionTimeout = DefaultConnectionTimeout
	}

	if resolved.Timeout == 0 {
		resolved.Timeout = DefaultTimeout
	}

	return resolved, nil
}

// PrettyConsoleHandler returns a [syntax.ErrorHandler] that formats the syntax error for
// display on the terminal to a user.
func PrettyConsoleHandler(w io.Writer) syntax.ErrorHandler {
	// TODO(@FollowTheProcess): This currently reads the whole file every time it's called
	// maybe we should gather up parse errors and then handle them "prettily" once at the end?
	return func(pos syntax.Position, msg string) {
		fmt.Fprintf(w, "%s: %s\n\n", pos, msg)

		contents, err := os.ReadFile(pos.Name)
		if err != nil {
			fmt.Fprintf(w, "unable to show src context: %v\n", err)
			return
		}

		lines := bytes.Split(contents, []byte("\n"))

		const contextLines = 3

		startLine := max(pos.Line-contextLines, 0)
		endLine := min(pos.Line+contextLines, len(lines))

		for i, line := range lines {
			i++ // Lines are 1 indexed
			if i >= startLine && i <= endLine {
				// Note: This is U+2502/"Box Drawings Light Vertical" NOT standard vertical pipe '|'
				margin := fmt.Sprintf("%d │ ", i)
				fmt.Fprintf(w, "%s%s\n", margin, line)
			}
		}
	}
}
