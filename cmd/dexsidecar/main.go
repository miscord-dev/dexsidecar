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
		select {
		case <-timer.C:
			if err := issuer.Rotate(ctx); err != nil {
				slog.Error("failed to rotate token", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
