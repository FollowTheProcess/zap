package zap

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"time"

	"go.followtheprocess.codes/zap/internal/syntax"
	"go.followtheprocess.codes/zap/internal/syntax/parser"
)

// ExportOptions are the flags passed to the export subcommand.
type ExportOptions struct {
	// Format is the format of the export e.g. curl, postman etc.
	Format string

	// Requests is the list of request names to export, empty or nil means
	// export all requests from the file.
	Requests []string

	// Debug controls debug logging.
	Debug bool
}

// Validate reports whether the ExportOptions is valid, returning a non-nil
// error if it's not.
func (e ExportOptions) Validate() error {
	switch format := e.Format; format {
	case "json", "curl", "postman":
		return nil
	default:
		return fmt.Errorf("invalid option for --format %q, allowed values are 'json', 'curl', 'postman'", format)
	}
}

// Export handles the export subcommand.
func (z Zap) Export(ctx context.Context, file string, handler syntax.ErrorHandler, options ExportOptions) error {
	logger := z.logger.Prefixed("export")

	logger.Debug("Export configuration", slog.String("options", fmt.Sprintf("%+v", options)))

	start := time.Now()

	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("could not open file: %w", err)
	}
	defer f.Close()

	p, err := parser.New(file, f, handler)
	if err != nil {
		return fmt.Errorf("could not initialise the parser: %w", err)
	}

	parsed, err := p.Parse()
	if err != nil {
		return err
	}

	parsed, err = z.evaluateGlobalPrompts(logger, parsed)
	if err != nil {
		return fmt.Errorf("could not evaluate global prompts: %w", err)
	}

	logger.Debug("Parsed file successfully", slog.String("file", file), slog.Duration("took", time.Since(start)))

	var toExport []syntax.Request

	if len(options.Requests) == 0 {
		// No filter, so execute all the requests
		toExport = parsed.Requests
	} else {
		// Only execute the ones asked for (if they exist)
		for _, actualRequest := range parsed.Requests {
			if slices.Contains(options.Requests, actualRequest.Name) {
				toExport = append(toExport, actualRequest)
			}
		}
	}

	if len(toExport) == 0 {
		return fmt.Errorf("no matching requests for names %v in %s", options.Requests, file)
	}

	logger.Debug("Filtered requests to execute", slog.Int("count", len(toExport)))

	toExport, err = z.evaluateRequestPrompts(logger, toExport, parsed.Prompts)
	if err != nil {
		return fmt.Errorf("could not evaluate request prompts: %w", err)
	}

	for _, request := range toExport {
		logger.Debug(
			"Exporting request",
			slog.String("request", request.Name),
			slog.String("format", options.Format),
		)

		exported, err := z.exportRequest(request, options)
		if err != nil {
			return fmt.Errorf("could not export request %s: %w", request.Name, err)
		}

		fmt.Fprintln(z.stdout, exported)
	}

	return nil
}

// exportRequest performs the export operation on a single request.
func (z Zap) exportRequest(request syntax.Request, options ExportOptions) (string, error) {
	switch options.Format {
	case "json":
		out, err := json.MarshalIndent(request, "", "  ")
		if err != nil {
			return "", err
		}

		return string(out), nil
	default:
		fmt.Printf("TODO: Handle %s\n", options.Format)
		return "", nil
	}
}
