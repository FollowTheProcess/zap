package zap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.followtheprocess.codes/zap/internal/syntax"
)

// TestOptions are the options passed to the test subcommand.
type TestOptions struct {
	// Requests are the names of specific requests to be run.
	//
	// Empty or nil means run all requests in the file.
	// Mutually exclusive with Filter and Pattern.
	Requests []string

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

	// Verbose shows additional details about the request, by default
	// only the status and the body are shown.
	Verbose bool
}

// Validate reports whether the TestOptions is valid, returning an error
// if it's not.
//
// nil means the options are valid.
func (t TestOptions) Validate() error {
	switch {
	case t.Timeout == 0:
		return errors.New("timeout cannot be 0")
	case t.ConnectionTimeout == 0:
		return errors.New("connection-timeout cannot be 0")
	case t.OverallTimeout == 0:
		return errors.New("overall-timeout cannot be 0")
	case t.ConnectionTimeout >= t.OverallTimeout:
		return fmt.Errorf(
			"connection-timeout (%s) cannot be larger than overall-timeout (%s)",
			t.ConnectionTimeout,
			t.OverallTimeout,
		)
	case t.ConnectionTimeout >= t.Timeout:
		return fmt.Errorf("connection-timeout (%s) cannot be larger than timeout (%s)", t.ConnectionTimeout, t.Timeout)
	case t.Timeout >= t.OverallTimeout:
		return fmt.Errorf("timeout (%s) cannot be larger than overall-timeout (%s)", t.Timeout, t.OverallTimeout)
	default:
		return nil
	}
}

// Test implements the test subcommand.
func (z Zap) Test(ctx context.Context, path string, handler syntax.ErrorHandler, options TestOptions) error {
	logger := z.logger.Prefixed("test").With(slog.String("path", path))
	logger.Debug("Collecting tests in path")

	return nil
}
