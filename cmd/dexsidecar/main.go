package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/miscord-dev/dexsidecar/pkg/issuer"
)

func main() {
	var err error
	var refreshInterval time.Duration
	if d := os.Getenv("dex_refresh_interval"); d != "" {
		refreshInterval, err = time.ParseDuration(d)
		if err != nil {
			slog.Error("failed to parse dex_refresh_interval", "error", err)

			os.Exit(1)
		}
	} else {
		refreshInterval = time.Minute
	}

	issuer := issuer.NewIssuer(issuer.ConfigFromEnvs)

	timer := time.NewTicker(refreshInterval)
	defer timer.Stop()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	for {
		slog.Debug("rotating token")

		if err := issuer.Rotate(ctx); err != nil {
			slog.Error("failed to rotate token", "error", err)
		}

		select {
		case <-timer.C:
		case <-ctx.Done():
			return
		}
	}
}
