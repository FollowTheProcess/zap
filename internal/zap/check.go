package zap

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"go.followtheprocess.codes/msg"
	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/syntax/parser"
	"golang.org/x/sync/errgroup"
)

// CheckOptions are the options passed to the check subcommand.
type CheckOptions struct {
	// Debug enables debug logging.
	Debug bool
}

// Check implements the check subcommand.
func (z Zap) Check(ctx context.Context, path string, handler syntax.ErrorHandler, options CheckOptions) error {
	z.logger.Debug("Checking path", slog.String("path", path))

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("could not get path info: %w", err)
	}

	var paths []string

	if info.IsDir() {
		z.logger.Debug("Path is a directory", slog.String("path", path))

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
		z.logger.Debug("Path is a file", slog.String("path", path))

		paths = []string{path}
	}

	z.logger.Debug("Checking http files", slog.Int("number", len(paths)))

	group := errgroup.Group{}

	for _, path := range paths {
		group.Go(func() error {
			return z.checkFile(path, handler)
		})
	}

	if err := group.Wait(); err != nil {
		return err
	}

	for _, path := range paths {
		msg.Fsuccess(z.stdout, "%s is valid", path)
	}

	return nil
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

	return nil
}
