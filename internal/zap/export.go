package zap

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.followtheprocess.codes/zap/internal/format"
	"go.followtheprocess.codes/zap/internal/spec"
	"go.followtheprocess.codes/zap/internal/syntax"
)

const (
	formatJSON    = "json"
	formatCurl    = "curl"
	formatPostman = "postman"
)

// ExportOptions are the flags passed to the export subcommand.
type ExportOptions struct {
	// Format is the format of the export e.g. curl, postman etc.
	Format string

	// Debug controls debug logging.
	Debug bool
}

// Validate reports whether the ExportOptions is valid, returning a non-nil
// error if it's not.
func (e ExportOptions) Validate() error {
	switch f := e.Format; f {
	case formatJSON, formatCurl, formatPostman:
		return nil
	default:
		return fmt.Errorf("invalid option for --format %q, allowed values are 'json', 'curl', 'postman'", f)
	}
}

// TODO(@FollowTheProcess): This currently just exports a bunch of requests by themselves.
// What we should actually do is export the whole file. Maybe make an Exporter interface?
//
// Each implementation (curl, json, yaml, postman) etc. then simply chooses how to export a whole
// file. The curl one could simply write a bash script with global variables set for example, json
// and yaml are easy. Postman we might need an entirely different data structure to convert into
// as I'd like to create a collection per file (if there are multiple requests) and populate
// collection variables etc.

// Export handles the export subcommand.
func (z Zap) Export(ctx context.Context, file string, handler syntax.ErrorHandler, options ExportOptions) error {
	logger := z.logger.Prefixed("export")

	logger.Debug("Export configuration", slog.String("options", fmt.Sprintf("%+v", options)))

	start := time.Now()

	httpFile, err := z.parseFile(file, handler)
	if err != nil {
		return err
	}

	httpFile, err = z.evaluateGlobalPrompts(logger, httpFile)
	if err != nil {
		return fmt.Errorf("could not evaluate global prompts: %w", err)
	}

	logger.Debug("Parsed file successfully", slog.String("file", file), slog.Duration("took", time.Since(start)))

	evaluated, err := z.evaluateRequestPrompts(logger, httpFile.Requests, httpFile.Prompts)
	if err != nil {
		return fmt.Errorf("could not evaluate request prompts: %w", err)
	}

	// TODO(@FollowTheProcess): Should probably have an evaluateAllPrompts function that takes in
	// a file and returns the evaluated file
	httpFile.Requests = evaluated

	return z.exportFile(httpFile, options)
}

// exportFile performs the export operation on the given http file.
func (z Zap) exportFile(file spec.File, options ExportOptions) error {
	var exporter format.Exporter

	switch options.Format {
	case formatJSON:
		exporter = format.JSONExporter{}
	case formatCurl:
		exporter = format.CurlExporter{}
	default:
		fmt.Printf("TODO: Handle %s\n", options.Format)
		return nil
	}

	return exporter.Export(z.stdout, file)
}
