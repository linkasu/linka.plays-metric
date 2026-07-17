package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/linkasu/linka.plays-metric/internal/app"
	"github.com/linkasu/linka.plays-metric/internal/auth"
	"github.com/linkasu/linka.plays-metric/internal/collector"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("collector stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	installationSecret, err := app.Secret("INSTALLATION_HMAC_SECRET")
	if err != nil {
		return err
	}
	writerSecret, err := app.Secret("WRITER_HMAC_SECRET")
	if err != nil {
		return err
	}
	writerURL, err := app.Env("WRITER_URL")
	if err != nil {
		return err
	}
	tokens, err := auth.NewInstallationTokens(installationSecret)
	if err != nil {
		return err
	}
	writerClient, err := collector.NewHTTPWriter(writerURL, writerSecret, 10*time.Second)
	if err != nil {
		return err
	}
	address := os.Getenv("LISTEN_ADDR")
	if address == "" {
		address = ":8080"
	}
	return app.Serve(address, collector.NewServer(writerClient, tokens, logger), logger)
}
