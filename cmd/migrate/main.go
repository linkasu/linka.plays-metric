package main

import (
	"context"
	"crypto/tls"
	"log/slog"
	"os"
	"strings"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/linkasu/linka.plays-metric/internal/app"
	metricmigrations "github.com/linkasu/linka.plays-metric/migrations"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}
	logger.Info("migrations applied")
}

func run() error {
	password, err := app.Env("CLICKHOUSE_ADMIN_PASSWORD")
	if err != nil {
		return err
	}
	addresses := envDefault("CLICKHOUSE_ADDRS", "clickhouse:9000")
	username := envDefault("CLICKHOUSE_ADMIN_USER", "admin")
	database := envDefault("CLICKHOUSE_DATABASE", "linka_metric")
	options := &ch.Options{
		Addr: strings.Split(addresses, ","), Auth: ch.Auth{Database: "default", Username: username, Password: password},
		DialTimeout: 5 * time.Second,
	}
	if os.Getenv("CLICKHOUSE_SECURE") == "true" {
		options.TLS = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	connection, err := ch.Open(options)
	if err != nil {
		return err
	}
	defer connection.Close()
	backend, err := metricmigrations.NewClickHouseBackend(connection, database)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	return metricmigrations.Run(ctx, backend)
}

func envDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
