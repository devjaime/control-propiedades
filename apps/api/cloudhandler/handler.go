package cloudhandler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/devjaime/control-propiedades/apps/api/internal/config"
	"github.com/devjaime/control-propiedades/apps/api/internal/platform/httpserver"
	"github.com/devjaime/control-propiedades/apps/api/internal/platform/objectstorage"
)

var (
	initializeOnce sync.Once
	cloudHandler   http.Handler
	initializeErr  error
)

// Handler serves the complete API from a serverless Go entrypoint.
func Handler(w http.ResponseWriter, r *http.Request) {
	initializeOnce.Do(initialize)
	if initializeErr != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    "api_initialization_failed",
				"message": "La API no pudo iniciar. Revisa la configuración del despliegue.",
			},
		})
		return
	}

	// vercel.json preserves the public route in this internal query parameter.
	if route := strings.TrimPrefix(r.URL.Query().Get("__route"), "/"); route != "" {
		clonedRequest := r.Clone(r.Context())
		clonedURL := *r.URL
		query := clonedURL.Query()
		query.Del("__route")
		clonedURL.RawQuery = query.Encode()
		clonedURL.Path = "/" + route
		clonedRequest.URL = &clonedURL
		r = clonedRequest
	}
	cloudHandler.ServeHTTP(w, r)
}

func initialize() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		initializeErr = err
		logger.Error("invalid cloud configuration", "error", err)
		return
	}
	poolConfig, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		initializeErr = err
		logger.Error("invalid database URL", "error", err)
		return
	}
	poolConfig.MaxConns = cfg.Database.MaxConns
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		initializeErr = err
		logger.Error("create database pool", "error", err)
		return
	}
	storage, err := objectstorage.New(
		cfg.Storage.Endpoint,
		cfg.Storage.Region,
		cfg.Storage.AccessKey,
		cfg.Storage.SecretKey,
		cfg.Storage.Bucket,
		cfg.Storage.UseSSL,
	)
	if err != nil {
		pool.Close()
		initializeErr = err
		logger.Error("create object storage client", "error", err)
		return
	}
	cloudHandler = httpserver.NewApplication(logger, pool, storage, httpserver.Options{
		Environment:    cfg.Environment,
		WebOrigin:      cfg.Auth.WebOrigin,
		PublicWebURL:   cfg.Auth.PublicWebURL,
		SessionTTL:     cfg.Auth.SessionTTL,
		CookieName:     cfg.Auth.CookieName,
		CookieSecure:   cfg.Auth.CookieSecure,
		MaxUploadBytes: cfg.Storage.MaxUploadBytes,
	})
}
