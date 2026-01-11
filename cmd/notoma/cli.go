package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/lmittmann/tint"
)

// Shared flag variables for all commands.
var (
	configPath string
	verbose    bool
)

// setupLogger creates and sets the default logger.
// If output is nil, logs go to stderr.
func setupLogger(output io.Writer, verbose bool) *slog.Logger {
	if output == nil {
		output = os.Stderr
	}

	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	logger := slog.New(tint.NewHandler(output, &tint.Options{
		Level: level,
	}))
	slog.SetDefault(logger)

	return logger
}

// setupSignalHandler creates a context that cancels on SIGINT/SIGTERM.
// The returned cancel function should be deferred.
func setupSignalHandler(logger *slog.Logger) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		logger.Info("received shutdown signal, canceling...")
		cancel()
	}()

	return ctx, cancel
}
