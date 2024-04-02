package log

import (
	"log/slog"
	"os"
)

func NewLogger(verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	return slog.New(
		slog.NewTextHandler(
			os.Stderr, &slog.HandlerOptions{
				Level: level,
			},
		),
	)
}
