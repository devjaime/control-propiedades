package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment     string
	ShutdownTimeout time.Duration
	Server          Server
	Database        Database
	Auth            Auth
	Storage         Storage
}

type Server struct {
	Address           string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

type Database struct {
	URL      string
	MaxConns int32
}

type Auth struct {
	WebOrigin    string
	PublicWebURL string
	SessionTTL   time.Duration
	CookieName   string
	CookieSecure bool
}

type Storage struct {
	Endpoint       string
	Region         string
	AccessKey      string
	SecretKey      string
	Bucket         string
	UseSSL         bool
	MaxUploadBytes int64
}

func Load() (Config, error) {
	shutdownTimeout, err := durationEnv("SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	maxConns, err := int32Env("DATABASE_MAX_CONNS", 10)
	if err != nil {
		return Config{}, err
	}
	sessionTTL, err := durationEnv("SESSION_TTL", 7*24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	maxUploadBytes, err := int32Env("MAX_UPLOAD_BYTES", 50*1024*1024)
	if err != nil {
		return Config{}, err
	}
	storageUseSSL, err := boolEnv("STORAGE_USE_SSL", false)
	if err != nil {
		return Config{}, err
	}

	webOrigin := stringEnv("WEB_ORIGIN", "http://localhost:3000")
	cfg := Config{
		Environment:     stringEnv("APP_ENV", "local"),
		ShutdownTimeout: shutdownTimeout,
		Server: Server{
			Address:           stringEnv("API_ADDR", ":8080"),
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
		Database: Database{
			URL:      strings.TrimSpace(os.Getenv("DATABASE_URL")),
			MaxConns: maxConns,
		},
		Auth: Auth{
			WebOrigin:    webOrigin,
			PublicWebURL: stringEnv("PUBLIC_WEB_URL", webOrigin),
			SessionTTL:   sessionTTL,
			CookieName:   stringEnv("SESSION_COOKIE_NAME", "cp_session"),
			CookieSecure: stringEnv("APP_ENV", "local") == "production",
		},
		Storage: Storage{
			Endpoint:       stringEnv("STORAGE_ENDPOINT", "localhost:9000"),
			Region:         stringEnv("STORAGE_REGION", "us-east-1"),
			AccessKey:      strings.TrimSpace(os.Getenv("STORAGE_ACCESS_KEY")),
			SecretKey:      strings.TrimSpace(os.Getenv("STORAGE_SECRET_KEY")),
			Bucket:         stringEnv("STORAGE_BUCKET", "control-propiedades"),
			UseSSL:         storageUseSSL,
			MaxUploadBytes: int64(maxUploadBytes),
		},
	}

	if cfg.Database.URL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if cfg.Storage.AccessKey == "" || cfg.Storage.SecretKey == "" {
		return Config{}, errors.New("STORAGE_ACCESS_KEY and STORAGE_SECRET_KEY are required")
	}
	return cfg, nil
}

func boolEnv(name string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean", name)
	}
	return parsed, nil
}

func stringEnv(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func durationEnv(name string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", name)
	}
	return parsed, nil
}

func int32Env(name string, fallback int32) (int32, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return int32(parsed), nil
}
