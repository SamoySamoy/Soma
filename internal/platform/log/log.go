// Package log builds Soma's structured logger and carries it through contexts.
package log

import (
	"context"
	"io"
	"log/slog"
	"strings"
)

// Redacted replaces the value of any attribute whose key looks sensitive.
const Redacted = "[redacted]"

// sensitiveKeys are matched case-insensitively against attribute keys, as
// substrings. Logging these by accident must not leak them.
var sensitiveKeys = []string{
	"password", "passphrase", "secret", "token", "authorization",
	"cookie", "session", "email", "api_key", "apikey", "master_key", "totp",
}

// New returns a logger writing to w. level is debug, info, warn or error;
// format is json or text.
func New(w io.Writer, level, format string) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:       parseLevel(level),
		ReplaceAttr: redact,
	}
	var h slog.Handler
	if format == "text" {
		h = slog.NewTextHandler(w, opts)
	} else {
		h = slog.NewJSONHandler(w, opts)
	}
	return slog.New(h)
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func redact(_ []string, a slog.Attr) slog.Attr {
	key := strings.ToLower(a.Key)
	for _, s := range sensitiveKeys {
		if strings.Contains(key, s) {
			return slog.String(a.Key, Redacted)
		}
	}
	return a
}

type ctxKey struct{}

// WithLogger returns a copy of ctx carrying l.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext returns the logger in ctx, or slog.Default if there is none.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
