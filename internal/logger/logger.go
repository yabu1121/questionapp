package logger

import (
	"log/slog"
	"os"
)

func New() *slog.Logger {
	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: false,
			Level:     slog.LevelInfo,
		}),
	)
	return logger
}
