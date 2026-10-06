package authz

import "testing"

func TestAllows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		role Role
		perm Permission
		want bool
	}{
		{"owner reads people", RoleOwner, PeopleRead, true},
		{"owner writes people", RoleOwner, PeopleWrite, true},
		{"editor reads people", RoleEditor, PeopleRead, true},
		{"editor writes people", RoleEditor, PeopleWrite, true},
		{"viewer reads people", RoleViewer, PeopleRead, true},
		{"viewer cannot write people", RoleViewer, PeopleWrite, false},
		{"unknown role grants nothing", Role("guest"), PeopleRead, false},
		{"unknown permission is denied", RoleOwner, Permission("guests:read"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Allows(tt.role, tt.perm); got != tt.want {
				t.Errorf("Allows(%q, %q) = %v, want %v", tt.role, tt.perm, got, tt.want)
			}
		})
	}
}
