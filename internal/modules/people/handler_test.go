package people

import (
	"testing"

	"github.com/SamoySamoy/Soma/internal/platform/apperr"
)

func TestParseIfMatch(t *testing.T) {
	t.Parallel()

	str := func(s string) *string { return &s }
	tests := []struct {
		name    string
		header  *string
		want    int32
		wantErr bool
	}{
		{"quoted version", str(`"3"`), 3, false},
		{"unquoted version", str("12"), 12, false},
		{"surrounding spaces", str(`  "7" `), 7, false},
		{"missing header", nil, 0, true},
		{"not a number", str(`"abc"`), 0, true},
		{"zero is not a version", str(`"0"`), 0, true},
		{"weak validator is not accepted", str(`W/"3"`), 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseIfMatch(tt.header)
			if tt.wantErr {
				e, ok := apperr.As(err)
				if !ok || e.Kind != apperr.KindPreconditionFailed {
					t.Fatalf("parseIfMatch() error = %v, want a PreconditionFailed error", err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("parseIfMatch() = %d, %v; want %d, nil", got, err, tt.want)
			}
		})
	}
}
