package self

import (
	"testing"

	"github.com/SamoySamoy/Soma/internal/platform/apperr"
)

func TestProfileFields(t *testing.T) {
	t.Parallel()

	filled, total := Profile{PreferredName: "Lan", CoreValues: "honesty"}.Fields()
	if filled != 2 || total != 4 {
		t.Errorf("Fields() = %d of %d, want 2 of 4", filled, total)
	}
}

func TestInputValidate(t *testing.T) {
	t.Parallel()

	if err := (Input{PreferredName: "Lan", BirthDate: "1990-04-23"}).normalized().validate(); err != nil {
		t.Fatalf("valid profile refused: %v", err)
	}
	e, ok := apperr.As((Input{BirthDate: "23 April 1990"}).normalized().validate())
	if !ok || e.Kind != apperr.KindInvalid {
		t.Fatalf("bad birth date accepted")
	}
	if _, has := e.Fields["birth_date"]; !has {
		t.Errorf("fields = %v, want birth_date", e.Fields)
	}
}

func TestPatchApply(t *testing.T) {
	t.Parallel()

	cur := Input{PreferredName: "Lan", Bio: "old"}
	got, err := Patch{"bio": nil, "core_values": "kindness"}.apply(cur)
	if err != nil {
		t.Fatalf("apply() error = %v", err)
	}
	want := Input{PreferredName: "Lan", CoreValues: "kindness"}
	if got != want {
		t.Errorf("apply() = %+v, want %+v", got, want)
	}
	if _, err := (Patch{"version": 9.0}).apply(cur); err == nil {
		t.Error("apply(version) succeeded; the version is not writable")
	}
}
