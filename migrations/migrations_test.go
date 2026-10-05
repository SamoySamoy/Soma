package migrations

import (
	"io/fs"
	"regexp"
	"testing"
)

var namePattern = regexp.MustCompile(`^\d{14}_[a-z0-9_]+\.sql$`)

func TestFileNamesFollowConvention(t *testing.T) {
	t.Parallel()

	entries, err := fs.Glob(FS(), "*.sql")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no migrations embedded")
	}
	for _, name := range entries {
		if !namePattern.MatchString(name) {
			t.Errorf("migration %q does not match YYYYMMDDHHMMSS_snake_description.sql", name)
		}
	}
}

func TestLatest(t *testing.T) {
	t.Parallel()

	v, err := Latest()
	if err != nil {
		t.Fatalf("Latest() error = %v", err)
	}
	if v < 20261005000000 {
		t.Errorf("Latest() = %d, want at least the baseline 20261005000000", v)
	}
}
