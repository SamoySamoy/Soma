// Package migrations embeds Soma's goose SQL migrations into the binary.
package migrations

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"
)

//go:embed *.sql
var files embed.FS

// FS returns the embedded migration files.
func FS() fs.FS { return files }

// Latest returns the version of the newest embedded migration: the numeric
// prefix of its file name.
func Latest() (int64, error) {
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return 0, fmt.Errorf("read migrations: %w", err)
	}
	var latest int64
	for _, e := range entries {
		if path.Ext(e.Name()) != ".sql" {
			continue
		}
		prefix, _, ok := strings.Cut(e.Name(), "_")
		if !ok {
			return 0, fmt.Errorf("migration %q has no version prefix", e.Name())
		}
		v, err := strconv.ParseInt(prefix, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("migration %q: parse version: %w", e.Name(), err)
		}
		latest = max(latest, v)
	}
	return latest, nil
}
