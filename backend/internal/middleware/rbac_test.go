package middleware

import "testing"

func TestSatisfiesAll(t *testing.T) {
	cases := []struct {
		name      string
		userPerms []string
		required  []string
		wantOK    bool
		wantMiss  string
	}{
		{"wildcard grants everything", []string{PermAll}, []string{PermCompaniesWrite, PermSponsorsDelete}, true, ""},
		{"exact match single", []string{PermCompaniesRead}, []string{PermCompaniesRead}, true, ""},
		{"has all required", []string{PermCompaniesRead, PermCompaniesWrite}, []string{PermCompaniesRead, PermCompaniesWrite}, true, ""},
		{"missing one of several", []string{PermCompaniesRead}, []string{PermCompaniesRead, PermCompaniesWrite}, false, PermCompaniesWrite},
		{"missing entirely", []string{PermSponsorsRead}, []string{PermCompaniesRead}, false, PermCompaniesRead},
		{"no permissions at all", nil, []string{PermCompaniesRead}, false, PermCompaniesRead},
		{"empty required is vacuously satisfied", []string{PermCompaniesRead}, nil, true, ""},
		{"empty user perms, empty required", nil, nil, true, ""},
		{"similar but distinct resource does not satisfy", []string{PermCompaniesRead}, []string{PermContactsRead}, false, PermContactsRead},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			missing, ok := satisfiesAll(tc.userPerms, tc.required)
			if ok != tc.wantOK {
				t.Fatalf("satisfiesAll(%v, %v) ok = %v, want %v", tc.userPerms, tc.required, ok, tc.wantOK)
			}
			if !ok && missing != tc.wantMiss {
				t.Fatalf("satisfiesAll(%v, %v) missing = %q, want %q", tc.userPerms, tc.required, missing, tc.wantMiss)
			}
		})
	}
}

func TestSatisfiesAny(t *testing.T) {
	cases := []struct {
		name      string
		userPerms []string
		required  []string
		want      bool
	}{
		{"wildcard grants everything", []string{PermAll}, []string{PermCompaniesWrite}, true},
		{"has one of several", []string{PermCompaniesRead}, []string{PermCompaniesRead, PermSponsorsRead}, true},
		{"has none of several", []string{PermCompaniesRead}, []string{PermSponsorsRead, PermContactsRead}, false},
		{"empty required is never satisfied", []string{PermCompaniesRead}, nil, false},
		{"no user perms", nil, []string{PermCompaniesRead}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := satisfiesAny(tc.userPerms, tc.required)
			if got != tc.want {
				t.Fatalf("satisfiesAny(%v, %v) = %v, want %v", tc.userPerms, tc.required, got, tc.want)
			}
		})
	}
}

// TestRolePermissionSetsAreWellFormed guards against typos in the
// hand-written permission slices (e.g. a copy-pasted constant from the
// wrong resource) by asserting some basic invariants about the role
// tiers requested for RBAC: Owner and Admin are unrestricted, tiers
// nest roughly Owner >= Admin >= Manager >= Member >= Guest for the
// core CRM read permissions, and Guest never gets a delete permission.
func TestRolePermissionSetsAreWellFormed(t *testing.T) {
	contains := func(perms []string, p string) bool {
		for _, x := range perms {
			if x == p {
				return true
			}
		}
		return false
	}

	if !contains(OwnerPermissions, PermAll) {
		t.Error("OwnerPermissions must be unrestricted (*)")
	}
	if !contains(AdminPermissions, PermAll) {
		t.Error("AdminPermissions must be unrestricted (*)")
	}

	for _, readPerm := range []string{PermCompaniesRead, PermSponsorsRead, PermContactsRead, PermCampaignsRead} {
		if !contains(ManagerPermissions, readPerm) {
			t.Errorf("ManagerPermissions missing %s", readPerm)
		}
		if !contains(MemberPermissions, readPerm) {
			t.Errorf("MemberPermissions missing %s", readPerm)
		}
		if !contains(GuestPermissions, readPerm) {
			t.Errorf("GuestPermissions missing %s", readPerm)
		}
	}

	for _, deletePerm := range []string{PermCompaniesDelete, PermSponsorsDelete, PermContactsDelete, PermCampaignsDelete} {
		if contains(GuestPermissions, deletePerm) {
			t.Errorf("GuestPermissions must not include %s — guests are read-only", deletePerm)
		}
		if contains(MemberPermissions, deletePerm) {
			t.Errorf("MemberPermissions must not include %s — deletion is a Manager+ action", deletePerm)
		}
	}

	if !contains(ManagerPermissions, PermTeamManage) {
		t.Error("ManagerPermissions should include team:manage")
	}
	if contains(MemberPermissions, PermTeamManage) {
		t.Error("MemberPermissions must not include team:manage")
	}
	if contains(GuestPermissions, PermTeamManage) || contains(GuestPermissions, PermTeamRead) {
		t.Error("GuestPermissions must not include any team permission")
	}
}
