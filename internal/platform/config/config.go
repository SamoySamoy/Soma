// Package config loads Soma's runtime configuration from environment variables.
// It is the only package allowed to read the process environment.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
)

// Mode is the deployment mode. It changes defaults such as registration and
// TLS expectations, never the code path that handles a request.
type Mode string

// Deployment modes, see business spec section 2.
const (
	ModeLocal   Mode = "local"
	ModePrivate Mode = "private"
	ModePublic  Mode = "public"
)

// Config is the full runtime configuration.
type Config struct {
	Mode    Mode   `env:"SOMA_MODE"     envDefault:"local"`
	BaseURL string `env:"SOMA_BASE_URL" envDefault:"http://localhost:8080"`

	HTTPAddr string `env:"SOMA_HTTP_ADDR" envDefault:"127.0.0.1:8080"`

	// DatabaseURL connects as the application role (soma_app), which is
	// subject to row-level security.
	DatabaseURL string `env:"SOMA_DATABASE_URL,required"`
	// MigrateDatabaseURL connects as the schema owner and is used only to run
	// migrations. Defaults to DatabaseURL when empty.
	MigrateDatabaseURL string `env:"SOMA_MIGRATE_DATABASE_URL"`
	// AutoMigrate applies pending migrations at startup.
	AutoMigrate bool `env:"SOMA_AUTO_MIGRATE" envDefault:"true"`

	LogLevel  string `env:"SOMA_LOG_LEVEL"  envDefault:"info"`
	LogFormat string `env:"SOMA_LOG_FORMAT" envDefault:"json"`
}

// Load reads configuration from the process environment.
func Load() (Config, error) {
	return parse(os.Environ())
}

// parse builds a Config from KEY=VALUE pairs. Split out so tests don't touch
// the real environment.
func parse(environ []string) (Config, error) {
	vars := make(map[string]string, len(environ))
	for _, kv := range environ {
		k, v, ok := strings.Cut(kv, "=")
		if ok {
			vars[k] = v
		}
	}

	var cfg Config
	if err := env.ParseWithOptions(&cfg, env.Options{Environment: vars}); err != nil {
		return Config{}, fmt.Errorf("parse environment: %w", err)
	}
	if cfg.MigrateDatabaseURL == "" {
		cfg.MigrateDatabaseURL = cfg.DatabaseURL
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	var errs []error

	switch c.Mode {
	case ModeLocal, ModePrivate, ModePublic:
	default:
		errs = append(errs, fmt.Errorf("SOMA_MODE must be local, private or public, got %q", c.Mode))
	}

	if u, err := url.Parse(c.BaseURL); err != nil || u.Scheme == "" || u.Host == "" {
		errs = append(errs, fmt.Errorf("SOMA_BASE_URL must be an absolute URL, got %q", c.BaseURL))
	} else if c.Mode == ModePublic && u.Scheme != "https" {
		errs = append(errs, errors.New("SOMA_BASE_URL must use https in public mode"))
	}

	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		errs = append(errs, fmt.Errorf("SOMA_LOG_LEVEL must be debug, info, warn or error, got %q", c.LogLevel))
	}

	switch c.LogFormat {
	case "json", "text":
	default:
		errs = append(errs, fmt.Errorf("SOMA_LOG_FORMAT must be json or text, got %q", c.LogFormat))
	}

	return errors.Join(errs...)
}
