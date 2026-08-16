// Package config loads and validates runtime configuration from the environment.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr             string
	AdminUser        string
	AdminPassHash    string
	SessionTTL       time.Duration
	CookieSecure     bool
	TrustProxyHeader bool
	LogLevel         slog.Level
}

// Load reads configuration from the environment, applying defaults for
// everything except the admin credentials, which have no safe default.
func Load() (Config, error) {
	cfg := Config{
		Addr:          envString("DW_ADDR", ":8111"),
		AdminUser:     os.Getenv("DW_ADMIN_USER"),
		AdminPassHash: os.Getenv("DW_ADMIN_PASS_HASH"),
	}

	var err error
	if cfg.SessionTTL, err = envDuration("DW_SESSION_TTL", 12*time.Hour); err != nil {
		return Config{}, err
	}
	if cfg.CookieSecure, err = envBool("DW_COOKIE_SECURE", true); err != nil {
		return Config{}, err
	}
	if cfg.TrustProxyHeader, err = envBool("DW_TRUST_PROXY_HEADER", false); err != nil {
		return Config{}, err
	}
	if cfg.LogLevel, err = envLogLevel("DW_LOG_LEVEL", slog.LevelInfo); err != nil {
		return Config{}, err
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	if c.AdminUser == "" {
		return errors.New("DW_ADMIN_USER is required")
	}
	if c.AdminPassHash == "" {
		return errors.New("DW_ADMIN_PASS_HASH is required")
	}
	if c.SessionTTL <= 0 {
		return errors.New("DW_SESSION_TTL must be positive")
	}
	return nil
}

func envString(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) (bool, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s: %w", key, err)
	}
	return parsed, nil
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return parsed, nil
}

func envLogLevel(key string, fallback slog.Level) (slog.Level, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(v)); err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return level, nil
}
