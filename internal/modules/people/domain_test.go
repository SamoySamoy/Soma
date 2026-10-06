package people

import (
	"strings"
	"testing"

	"github.com/SamoySamoy/Soma/internal/platform/apperr"
)

func TestInputValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		in         Input
		wantFields []string // empty means valid
	}{
		{"name only is valid", Input{DisplayName: "Lan"}, nil},
		{"full contact is valid", Input{DisplayName: "Lan", Birthday: "1990-04-23", Email: "lan@example.com"}, nil},
		{"name is required", Input{DisplayName: ""}, []string{"display_name"}},
		{"name over 200 characters", Input{DisplayName: strings.Repeat("a", 201)}, []string{"display_name"}},
		{"nickname over 100 characters", Input{DisplayName: "Lan", Nickname: strings.Repeat("n", 101)}, []string{"nickname"}},
		{"birthday must be a date", Input{DisplayName: "Lan", Birthday: "23/04/1990"}, []string{"birthday"}},
		{"birthday must be a real date", Input{DisplayName: "Lan", Birthday: "1990-02-30"}, []string{"birthday"}},
		{"several problems are all reported", Input{DisplayName: "", Notes: strings.Repeat("x", 10001), Birthday: "nope"},
			[]string{"birthday", "display_name", "notes"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.in.normalized().validate()
			if len(tt.wantFields) == 0 {
				if err != nil {
					t.Fatalf("validate() error = %v, want nil", err)
				}
				return
			}
			e, ok := apperr.As(err)
			if !ok || e.Kind != apperr.KindInvalid {
				t.Fatalf("validate() error = %v, want an Invalid error", err)
			}
			for _, f := range tt.wantFields {
				if _, has := e.Fields[f]; !has {
					t.Errorf("fields = %v, want a message for %q", e.Fields, f)
				}
			}
			if len(e.Fields) != len(tt.wantFields) {
				t.Errorf("fields = %v, want exactly %v", e.Fields, tt.wantFields)
			}
		})
	}
}

func TestInputNormalizedTrimsWhitespace(t *testing.T) {
	t.Parallel()

	got := Input{DisplayName: "  Lan  ", Phone: "   "}.normalized()
	if got.DisplayName != "Lan" || got.Phone != "" {
		t.Errorf("normalized() = %+v, want trimmed name and empty phone", got)
	}
}

func TestPatchApply(t *testing.T) {
	t.Parallel()

	ptr := func(s string) *string { return &s }
	cur := Input{DisplayName: "Lan", Nickname: "L", Phone: "123", Email: "lan@example.com"}

	tests := []struct {
		name    string
		patch   Patch
		want    Input
		wantErr bool
	}{
		{
			name:  "missing keys keep their values",
			patch: Patch{"phone": ptr("456")},
			want:  Input{DisplayName: "Lan", Nickname: "L", Phone: "456", Email: "lan@example.com"},
		},
		{
			name:  "null clears a field",
			patch: Patch{"nickname": nil},
			want:  Input{DisplayName: "Lan", Phone: "123", Email: "lan@example.com"},
		},
		{
			name:    "unknown field is rejected",
			patch:   Patch{"nickname_": ptr("x")},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.patch.apply(cur)
			if tt.wantErr {
				if e, ok := apperr.As(err); !ok || e.Kind != apperr.KindInvalid {
					t.Fatalf("apply() error = %v, want an Invalid error", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("apply() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("apply() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestBirthdayRoundTrip(t *testing.T) {
	t.Parallel()

	ptr, err := birthdayOf("1990-04-23")
	if err != nil {
		t.Fatalf("birthdayOf() error = %v", err)
	}
	if got := dateOf(ptr); got != "1990-04-23" {
		t.Errorf("dateOf(birthdayOf(x)) = %q, want 1990-04-23", got)
	}
	if p, err := birthdayOf(""); err != nil || p != nil {
		t.Errorf("birthdayOf(\"\") = %v, %v; want nil, nil", p, err)
	}
	if _, err := birthdayOf("bad"); err == nil {
		t.Errorf("birthdayOf(bad) error = %v, want an error", err)
	}
}
