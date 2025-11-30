package zap

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"go.followtheprocess.codes/zap/internal/format"
	"go.followtheprocess.codes/zap/internal/spec"
)

const (
	formatJSON    = "json"
	formatYAML    = "yaml"
	formatTOML    = "toml"
	formatCurl    = "curl"
	formatPostman = "postman"
)

// ExportOptions are the flags passed to the export subcommand.
type ExportOptions struct {
	// File is the file name of the http file to export.
	File string

	// Format is the format of the export e.g. curl, postman etc.
	Format string

	// Debug controls debug logging.
	Debug bool
}

// Validate reports whether the ExportOptions is valid, returning a non-nil
// error if it's not.
func (e ExportOptions) Validate() error {
	allowed := []string{formatJSON, formatYAML, formatTOML, formatCurl, formatPostman}
	if !slices.Contains(allowed, e.Format) {
		return fmt.Errorf("invalid option for --format, expected one of (%s)", strings.Join(allowed, ", "))
	}

	return nil
}

// Export handles the export subcommand.
func (z Zap) Export(ctx context.Context, options ExportOptions) error {
	logger := z.logger.Prefixed("export")

	logger.Debug("Export configuration", slog.String("options", fmt.Sprintf("%+v", options)))

	if err := options.Validate(); err != nil {
		return err
	}

	start := time.Now()

	httpFile, err := z.parseFile(options.File)
	if err != nil {
		return err
	}

	logger.Debug(
		"Parsed file successfully",
		slog.String("file", options.File),
		slog.Duration("took", time.Since(start)),
	)

	httpFile, err = z.evaluateAllPrompts(logger, httpFile)
	if err != nil {
		return err
	}

	return z.exportFile(httpFile, options)
}

// exportFile performs the export operation on the given http file.
func (z Zap) exportFile(file spec.File, options ExportOptions) error {
	// Default to JSON
	var exporter format.Exporter

	switch options.Format {
	case formatJSON:
		exporter = format.JSONExporter{}
	case formatYAML:
		exporter = format.YAMLExporter{}
	case formatTOML:
		exporter = format.TOMLExporter{}
	case formatCurl:
		exporter = format.CurlExporter{}
	default:
		fmt.Printf("TODO: Handle %s\n", options.Format)
		return nil
	}

	return exporter.Export(z.stdout, file)
}
