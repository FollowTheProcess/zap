package zap

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"go.followtheprocess.codes/msg"
	"golang.org/x/sync/errgroup"
)

// CheckOptions are the options passed to the check subcommand.
type CheckOptions struct {
	// Path is the path (file or directory) to check.
	Path string

	// Debug enables debug logging.
	Debug bool
}

// Check implements the check subcommand.
func (z Zap) Check(ctx context.Context, options CheckOptions) error {
	logger := z.logger.Prefixed("check").With(slog.String("path", options.Path))
	logger.Debug("Checking path")

	info, err := os.Stat(options.Path)
	if err != nil {
		return fmt.Errorf("could not get path info: %w", err)
	}

	var paths []string

	if info.IsDir() {
		logger.Debug("Path is a directory")

		err = filepath.WalkDir(options.Path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if filepath.Ext(path) == ".http" {
				paths = append(paths, path)
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("could not walk %s: %w", options.Path, err)
		}
	} else {
		logger.Debug("Path is a file")

		paths = []string{options.Path}
	}

	logger.Debug("Checking http files given by path", slog.Int("number", len(paths)))

	group := errgroup.Group{}

	for _, path := range paths {
		group.Go(func() error {
			_, err := z.parseFile(path)
			return err
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
