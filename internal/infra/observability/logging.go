package observability

import (
	"log/slog"
	"os"
)

func NewLogger(appEnv string) *slog.Logger {
	// JSON logs are friendly for aggregation (stdout -> docker logs -> ELK/Loki).
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if appEnv == "development" {
		opts.Level = slog.LevelDebug
	}

	// If you want plain text in local dev, change this handler accordingly.
	h := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(h)
}

