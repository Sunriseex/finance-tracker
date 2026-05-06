package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sunriseex/finance-manager/internal/config"
	"github.com/sunriseex/finance-manager/internal/http/handlers"
	"github.com/sunriseex/finance-manager/internal/postgres"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := config.Init(); err != nil {
		return fmt.Errorf("config init: %w", err)
	}

	addr := flag.String("addr", ":8080", "HTTP listen address")
	databaseURL := flag.String("database-url", config.AppConfig.DatabaseURL, "PostgreSQL connection URL")
	flag.Parse()

	pool, err := postgres.OpenPool(ctx, *databaseURL)
	if err != nil {
		return fmt.Errorf("open postgres pool: %w", err)
	}
	defer pool.Close()

	store := postgres.NewStore(pool)
	if err := store.Ping(ctx); err != nil {
		return fmt.Errorf("postgres ping: %w", err)
	}

	server := &http.Server{
		Addr:              *addr,
		Handler:           handlers.NewRouter(store),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErr := make(chan error, 1)

	go func() {
		slog.Info("server listening", "addr", *addr)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("listen and serve: %w", err)
			return
		}

		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		return err

	case <-ctx.Done():
		slog.Info("shutdown signal received")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("shutdown server: %w", err)
		}

		if err := <-serverErr; err != nil {
			return err
		}

		return nil
	}
}
