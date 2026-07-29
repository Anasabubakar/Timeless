package repository

import (
	"testing"

	"github.com/timeless/backend/internal/middleware"
)

// TestDefaultRolesMatchExpectedTiers guards the seed data every new
// organization gets. If this drifts from what RouteGuard/RBAC actually
// expect (wrong name, wrong permission slice), a brand-new org's Owner
// would register successfully and then be denied on every route —
// exactly the "created but locked out" failure mode Register() is
// supposed to prevent by seeding roles synchronously.
func TestDefaultRolesMatchExpectedTiers(t *testing.T) {
	roles := defaultRoles()

	wantNames := []string{"Owner", "Admin", "Manager", "Member", "Guest"}
	if len(roles) != len(wantNames) {
		t.Fatalf("defaultRoles() returned %d roles, want %d", len(roles), len(wantNames))
	}

	for i, want := range wantNames {
		if roles[i].Name != want {
			t.Errorf("defaultRoles()[%d].Name = %q, want %q", i, roles[i].Name, want)
		}
	}

	wantPerms := map[string][]string{
		"Owner":   middleware.OwnerPermissions,
		"Admin":   middleware.AdminPermissions,
		"Manager": middleware.ManagerPermissions,
		"Member":  middleware.MemberPermissions,
		"Guest":   middleware.GuestPermissions,
	}
	for _, r := range roles {
		want := wantPerms[r.Name]
		if len(r.Permissions) != len(want) {
			t.Errorf("defaultRoles() role %q has %d permissions, want %d", r.Name, len(r.Permissions), len(want))
			continue
		}
		for i := range want {
			if r.Permissions[i] != want[i] {
				t.Errorf("defaultRoles() role %q permission[%d] = %q, want %q", r.Name, i, r.Permissions[i], want[i])
			}
		}
	}

	// Owner must be unrestricted — everything else in RemoveMember/
	// UpdateMemberRole's "last owner" protection assumes this.
	if len(roles[0].Permissions) != 1 || roles[0].Permissions[0] != middleware.PermAll {
		t.Errorf("Owner role permissions = %v, want [%q]", roles[0].Permissions, middleware.PermAll)
	}
}
