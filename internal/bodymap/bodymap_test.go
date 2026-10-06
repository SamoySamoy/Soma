package bodymap

import (
	"testing"

	"github.com/SamoySamoy/Soma/internal/apigen"
)

// The contract's enum lists must match the registry, or the map would send
// areas the client doesn't know. This test fails if either side drifts.
func TestRegistryMatchesContract(t *testing.T) {
	t.Parallel()

	keys := map[string]bool{}
	for _, v := range []apigen.BodyAreaKey{
		apigen.Mind, apigen.Self, apigen.Responsibilities,
		apigen.Heart, apigen.Body, apigen.Work,
		apigen.Money, apigen.Growth, apigen.Journeys,
		apigen.Home, apigen.Papers,
	} {
		keys[string(v)] = true
	}

	seen := map[Area]bool{}
	for _, def := range Registry {
		if seen[def.Area] {
			t.Errorf("area %q is listed twice", def.Area)
		}
		seen[def.Area] = true
		if !keys[string(def.Area)] {
			t.Errorf("area %q is not in the contract enum", def.Area)
		}
	}
	if len(seen) != len(keys) {
		t.Errorf("registry has %d areas, contract has %d", len(seen), len(keys))
	}
}
