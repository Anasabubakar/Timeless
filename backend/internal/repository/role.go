package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/models"
)

type RoleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

// defaultRole is a system role seeded into every new organization. Names
// match the tiers requested for RBAC: Owner/Admin/Manager/Member/Guest.
type defaultRole struct {
	Name        string
	Permissions []string
}

func defaultRoles() []defaultRole {
	return []defaultRole{
		{Name: "Owner", Permissions: middleware.OwnerPermissions},
		{Name: "Admin", Permissions: middleware.AdminPermissions},
		{Name: "Manager", Permissions: middleware.ManagerPermissions},
		{Name: "Member", Permissions: middleware.MemberPermissions},
		{Name: "Guest", Permissions: middleware.GuestPermissions},
	}
}

// SeedDefaultRoles creates the standard Owner/Admin/Manager/Member/Guest
// system roles for a newly created organization and returns the Owner
// role's ID so the caller can assign it to the org's creator. Without
// this, RBACMiddleware.Require has nothing to look up and every route it
// guards fails closed for every user in the org — this is the seed data
// that makes RBAC enforcement actually usable rather than a lockout.
func (r *RoleRepository) SeedDefaultRoles(ctx context.Context, orgID uuid.UUID) (ownerRoleID uuid.UUID, err error) {
	for _, dr := range defaultRoles() {
		permsJSON, marshalErr := json.Marshal(dr.Permissions)
		if marshalErr != nil {
			return uuid.Nil, marshalErr
		}
		role := &models.Role{
			OrganizationID: orgID,
			Name:           dr.Name,
			Permissions:    permsJSON,
			IsSystem:       true,
		}
		if createErr := r.db.WithContext(ctx).Create(role).Error; createErr != nil {
			return uuid.Nil, createErr
		}
		if dr.Name == "Owner" {
			ownerRoleID = role.ID
		}
	}
	return ownerRoleID, nil
}

// AssignRole links a user to a role via the user_roles join table.
func (r *RoleRepository) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		"INSERT INTO user_roles (user_id, role_id) VALUES (?, ?) ON CONFLICT DO NOTHING", userID, roleID,
	).Error
}

// FindByName looks up a system role by its org-scoped name (e.g. for
// invite/role-change flows that reference roles by name).
func (r *RoleRepository) FindByName(ctx context.Context, orgID uuid.UUID, name string) (*models.Role, error) {
	var role models.Role
	err := r.db.WithContext(ctx).First(&role, "organization_id = ? AND name = ?", orgID, name).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// CountUsersWithRole is used to guard against removing the last user
// holding a given role in an org (e.g. the last Owner).
func (r *RoleRepository) CountUsersWithRole(ctx context.Context, orgID uuid.UUID, roleName string, excludingUserID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("user_roles").
		Joins("JOIN roles ON roles.id = user_roles.role_id").
		Where("roles.organization_id = ? AND roles.name = ? AND user_roles.user_id != ?", orgID, roleName, excludingUserID).
		Count(&count).Error
	return count, err
}
