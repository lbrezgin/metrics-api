// Package logger provides initialization helpers for the standard library slog logger.
//
// It supports configuration of:
//   - log level (debug/info/warn/error)
//   - handler type (text/json)
//   - output destination (stdout/file/both)
//
// Init sets the created logger as the global default via [slog.SetDefault], so packages
// can use [slog.Info]/[slog.Debug]/etc without passing a logger instance around.
//
// Note: Init may open OS resources (e.g. a log file). If Init returns a non-nil [io.Closer],
// the caller is responsible for closing it when the application shuts down.
package logger

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/lbrezgin/telemetry/internal/config"
)

const (
	logTypeJSON = "json"
	logTypeText = "text"

	logLevelDebug = "debug"
	logLevelInfo  = "info"
	logLevelWarn  = "warn"
	logLevelError = "error"

	logOutputStdout = "stdout"
	logOutputFile   = "file"
	logOutputBoth   = "both"
)

var (
	levelMap = map[string]slog.Level{
		logLevelDebug: slog.LevelDebug,
		logLevelInfo:  slog.LevelInfo,
		logLevelWarn:  slog.LevelWarn,
		logLevelError: slog.LevelError,
	}
)

var (
	ErrUnsupportedLogType   = errors.New("not supported log type")
	ErrUnsupportedLogLevel  = errors.New("not supported log level")
	ErrUnsupportedLogOutput = errors.New("not supported log output")
)

const (
	logFileName = "logs.log"
)

// Init initializes slog based on the provided configuration and sets it as the default logger.
//
// Side effects:
//   - Calls [slog.SetDefault] to install the created logger globally.
//   - May open a log file when output is "file" or "both".
//
// Resource management:
//   - If output is "file" or "both", Init returns a non-nil [io.Closer] which must be closed
//     by the caller during shutdown.
func Init(cfg *config.LogConfig) (io.Closer, error) {
	opts := &slog.HandlerOptions{}

	// Validate and apply configured log level.
	if err := setLevel(cfg.Level, opts); err != nil {
		return nil, err
	}

	// Choose the output destination. When a file is used, chooseWriter returns an io.Closer
	// (the opened file) that must be closed by the caller.
	writer, closer, err := chooseWriter(cfg.Output)
	if err != nil {
		return nil, err
	}

	// Choose the handler implementation (text/json) and bind it to the selected writer.
	handler, err := chooseHandler(cfg.Type, writer, opts)
	if err != nil {
		return nil, err
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("logger initialized")
	// closer may be nil for stdout-only output.
	return closer, nil
}

// setLevel validates a string log level and writes the resolved value into opts.
//
// It returns ErrUnsupportedLogLevel if the level is unknown.
func setLevel(logLevel string, opts *slog.HandlerOptions) error {
	level, ok := levelMap[logLevel]
	if !ok {
		return fmt.Errorf(
			"%q: %w",
			logLevel,
			ErrUnsupportedLogLevel,
		)
	}
	opts.Level = level
	return nil
}

// chooseWriter selects the output destination based on logOutput.
//
// Returns:
//   - io.Writer used by slog handlers for writing log records
//   - io.Closer that must be closed by the caller (non-nil only when a file is opened)
//   - error if the output is unsupported or a file cannot be opened
//
// For output "both", the returned writer duplicates writes to stdout and to the opened file.
func chooseWriter(logOutput string) (io.Writer, io.Closer, error) {
	if logOutput == logOutputStdout {
		return os.Stdout, nil, nil
	}

	if logOutput != logOutputFile && logOutput != logOutputBoth {
		return nil, nil, fmt.Errorf(
			"%q: %w",
			logOutput,
			ErrUnsupportedLogOutput,
		)
	}

	// Open the log file in append mode. The returned file must be closed by the caller.
	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, err
	}

	if logOutput == logOutputFile {
		return file, file, nil
	}

	// Duplicate logs to both stdout and file. We still return the file as the closer.
	mw := io.MultiWriter(os.Stdout, file)
	return mw, file, nil
}

// chooseHandler creates a slog handler of the requested type and binds it to writer.
//
// Returns ErrUnsupportedLogType if logType is unknown.
func chooseHandler(
	logType string,
	writer io.Writer,
	opts *slog.HandlerOptions) (slog.Handler, error) {

	switch logType {
	case logTypeJSON:
		return slog.NewJSONHandler(writer, opts), nil
	case logTypeText:
		return slog.NewTextHandler(writer, opts), nil
	}

	return nil, fmt.Errorf(
		"%q: %w",
		logType,
		ErrUnsupportedLogType,
	)
}

// LoggingMiddleware logs HTTP request and response details,
// including method, path, duration, status code, and response size.
func LoggingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		URL := r.URL.Path
		method := r.Method

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		h.ServeHTTP(ww, r)

		duration := time.Since(start)
		slog.Info(
			"got request",
			"url", URL,
			"method", method,
			"duration", duration,
		)

		slog.Info(
			"sent response",
			"status", ww.Status(),
			"size", ww.BytesWritten(),
		)
	})
}
