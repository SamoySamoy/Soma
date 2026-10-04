package config

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	t.Parallel()

	const dbURL = "SOMA_DATABASE_URL=postgres://soma_app@localhost/soma"

	tests := []struct {
		name    string
		environ []string
		wantErr string
		check   func(t *testing.T, c Config)
	}{
		{
			name:    "applies defaults",
			environ: []string{dbURL},
			check: func(t *testing.T, c Config) {
				t.Helper()
				if c.Mode != ModeLocal {
					t.Errorf("Mode = %q, want %q", c.Mode, ModeLocal)
				}
				if c.HTTPAddr != "127.0.0.1:8080" {
					t.Errorf("HTTPAddr = %q, want 127.0.0.1:8080", c.HTTPAddr)
				}
				if !c.AutoMigrate {
					t.Error("AutoMigrate = false, want true")
				}
			},
		},
		{
			name:    "migrate URL falls back to database URL",
			environ: []string{dbURL},
			check: func(t *testing.T, c Config) {
				t.Helper()
				if c.MigrateDatabaseURL != c.DatabaseURL {
					t.Errorf("MigrateDatabaseURL = %q, want %q", c.MigrateDatabaseURL, c.DatabaseURL)
				}
			},
		},
		{
			name:    "requires database URL",
			environ: nil,
			wantErr: "SOMA_DATABASE_URL",
		},
		{
			name:    "rejects unknown mode",
			environ: []string{dbURL, "SOMA_MODE=cloud"},
			wantErr: "SOMA_MODE",
		},
		{
			name:    "rejects relative base URL",
			environ: []string{dbURL, "SOMA_BASE_URL=/soma"},
			wantErr: "SOMA_BASE_URL",
		},
		{
			name:    "requires https in public mode",
			environ: []string{dbURL, "SOMA_MODE=public", "SOMA_BASE_URL=http://soma.example.com"},
			wantErr: "https",
		},
		{
			name:    "rejects unknown log level",
			environ: []string{dbURL, "SOMA_LOG_LEVEL=verbose"},
			wantErr: "SOMA_LOG_LEVEL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := parse(tt.environ)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("parse() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse() error = %v", err)
			}
			tt.check(t, cfg)
		})
	}
}
