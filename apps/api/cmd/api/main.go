package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/devjaime/control-propiedades/apps/api/internal/config"
	"github.com/devjaime/control-propiedades/apps/api/internal/platform/httpserver"
	"github.com/devjaime/control-propiedades/apps/api/internal/platform/objectstorage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	poolConfig, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		logger.Error("invalid database URL", "error", err)
		os.Exit(1)
	}
	poolConfig.MaxConns = cfg.Database.MaxConns

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		logger.Error("create database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	storage, err := objectstorage.New(
		cfg.Storage.Endpoint,
		cfg.Storage.Region,
		cfg.Storage.AccessKey,
		cfg.Storage.SecretKey,
		cfg.Storage.Bucket,
		cfg.Storage.UseSSL,
	)
	if err != nil {
		logger.Error("create object storage client", "error", err)
		os.Exit(1)
	}
	if err := storage.EnsureBucket(ctx); err != nil {
		logger.Error("ensure object storage bucket", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr: cfg.Server.Address,
		Handler: httpserver.NewApplication(logger, pool, storage, httpserver.Options{
			Environment:    cfg.Environment,
			WebOrigin:      cfg.Auth.WebOrigin,
			PublicWebURL:   cfg.Auth.PublicWebURL,
			SessionTTL:     cfg.Auth.SessionTTL,
			CookieName:     cfg.Auth.CookieName,
			CookieSecure:   cfg.Auth.CookieSecure,
			MaxUploadBytes: cfg.Storage.MaxUploadBytes,
		}),
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("api listening", "address", cfg.Server.Address, "environment", cfg.Environment)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown requested")
	case err = <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
}
