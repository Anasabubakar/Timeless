package middleware

import "fmt"

const (
	PermCampaignsRead   = "campaigns:read"
	PermCampaignsWrite  = "campaigns:write"
	PermCampaignsDelete = "campaigns:delete"

	PermSponsorsRead   = "sponsors:read"
	PermSponsorsWrite  = "sponsors:write"
	PermSponsorsDelete = "sponsors:delete"

	PermCompaniesRead   = "companies:read"
	PermCompaniesWrite  = "companies:write"
	PermCompaniesDelete = "companies:delete"

	PermContactsRead   = "contacts:read"
	PermContactsWrite  = "contacts:write"
	PermContactsDelete = "contacts:delete"

	PermProposalsRead     = "proposals:read"
	PermProposalsWrite    = "proposals:write"
	PermProposalsDelete   = "proposals:delete"
	PermProposalsGenerate = "proposals:generate"

	PermOutreachRead   = "outreach:read"
	PermOutreachWrite  = "outreach:write"
	PermOutreachDelete = "outreach:delete"

	PermAutomationsRead   = "automations:read"
	PermAutomationsWrite  = "automations:write"
	PermAutomationsDelete = "automations:delete"

	PermIntegrationsRead   = "integrations:read"
	PermIntegrationsWrite  = "integrations:write"
	PermIntegrationsDelete = "integrations:delete"

	PermWebhooksRead   = "webhooks:read"
	PermWebhooksWrite  = "webhooks:write"
	PermWebhooksDelete = "webhooks:delete"

	PermAnalyticsRead = "analytics:read"
	PermAIQuery       = "ai:query"

	PermSettingsRead  = "settings:read"
	PermSettingsWrite = "settings:write"

	PermUsersRead   = "users:read"
	PermUsersWrite  = "users:write"
	PermUsersDelete = "users:delete"

	PermFilesUpload = "files:upload"
	PermFilesRead   = "files:read"
	PermFilesDelete = "files:delete"

	PermActivitiesRead  = "activities:read"
	PermActivitiesWrite = "activities:write"

	PermCommunicationsRead   = "communications:read"
	PermCommunicationsWrite  = "communications:write"
	PermCommunicationsDelete = "communications:delete"

	PermKnowledgeRead  = "knowledge:read"
	PermKnowledgeWrite = "knowledge:write"

	PermNotificationsRead  = "notifications:read"
	PermNotificationsWrite = "notifications:write"

	PermTeamRead   = "team:read"
	PermTeamManage = "team:manage" // invite/remove members, change roles

	PermImportsWrite = "imports:write"

	PermEmailsSend = "emails:send"

	PermAll = "*"
)

// OwnerPermissions is granted to the user who created the organization.
// Functionally identical to AdminPermissions today; kept as a distinct
// name/role tier because ownership carries extra protections elsewhere
// (e.g. the last Owner can't be removed or demoted — see
// RoleRepository.CountUsersWithRole) that Admin does not.
var OwnerPermissions = []string{PermAll}

var AdminPermissions = []string{PermAll}

// ManagerPermissions sits between Admin and Member: full day-to-day CRM
// access plus team visibility and (unlike Member) the ability to manage
// team membership, but not org settings, integration deletion, or API
// key/webhook management.
var ManagerPermissions = []string{
	PermCampaignsRead, PermCampaignsWrite, PermCampaignsDelete,
	PermSponsorsRead, PermSponsorsWrite, PermSponsorsDelete,
	PermCompaniesRead, PermCompaniesWrite, PermCompaniesDelete,
	PermContactsRead, PermContactsWrite, PermContactsDelete,
	PermProposalsRead, PermProposalsWrite, PermProposalsDelete, PermProposalsGenerate,
	PermOutreachRead, PermOutreachWrite, PermOutreachDelete,
	PermAutomationsRead, PermAutomationsWrite,
	PermIntegrationsRead,
	PermWebhooksRead,
	PermAnalyticsRead,
	PermAIQuery,
	PermSettingsRead,
	PermActivitiesRead, PermActivitiesWrite,
	PermCommunicationsRead, PermCommunicationsWrite,
	PermKnowledgeRead, PermKnowledgeWrite,
	PermNotificationsRead, PermNotificationsWrite,
	PermTeamRead, PermTeamManage,
	PermFilesUpload, PermFilesRead,
	PermImportsWrite,
	PermEmailsSend,
}

var MemberPermissions = []string{
	PermCampaignsRead, PermCampaignsWrite,
	PermSponsorsRead, PermSponsorsWrite,
	PermCompaniesRead, PermCompaniesWrite,
	PermContactsRead, PermContactsWrite,
	PermProposalsRead, PermProposalsWrite, PermProposalsGenerate,
	PermOutreachRead, PermOutreachWrite,
	PermAutomationsRead,
	PermIntegrationsRead,
	PermWebhooksRead,
	PermAnalyticsRead,
	PermAIQuery,
	PermSettingsRead,
	PermActivitiesRead, PermActivitiesWrite,
	PermCommunicationsRead, PermCommunicationsWrite,
	PermKnowledgeRead, PermKnowledgeWrite,
	PermNotificationsRead, PermNotificationsWrite,
	PermTeamRead,
	PermFilesUpload, PermFilesRead,
	PermImportsWrite,
	PermEmailsSend,
}

var ViewerPermissions = []string{
	PermCampaignsRead,
	PermSponsorsRead,
	PermCompaniesRead,
	PermContactsRead,
	PermProposalsRead,
	PermAnalyticsRead,
	PermSettingsRead,
	PermActivitiesRead,
	PermCommunicationsRead,
	PermKnowledgeRead,
	PermNotificationsRead,
	PermFilesRead,
}

// GuestPermissions is the most restricted tier: read-only access to core
// CRM records, nothing else (no analytics, settings, integrations, or
// team visibility).
var GuestPermissions = []string{
	PermCampaignsRead,
	PermSponsorsRead,
	PermCompaniesRead,
	PermContactsRead,
	PermNotificationsRead,
}

// permissionDescriptions turns a permission key like "settings:write"
// into the plain-English action it gates, so a 403 can tell a user what
// they were trying to do instead of exposing the internal permission
// name.
var permissionDescriptions = map[string]string{
	PermCampaignsRead:        "view campaigns",
	PermCampaignsWrite:       "create or edit campaigns",
	PermCampaignsDelete:      "delete campaigns",
	PermSponsorsRead:         "view sponsors",
	PermSponsorsWrite:        "create or edit sponsors",
	PermSponsorsDelete:       "delete sponsors",
	PermCompaniesRead:        "view companies",
	PermCompaniesWrite:       "create or edit companies",
	PermCompaniesDelete:      "delete companies",
	PermContactsRead:         "view contacts",
	PermContactsWrite:        "create or edit contacts",
	PermContactsDelete:       "delete contacts",
	PermProposalsRead:        "view proposals",
	PermProposalsWrite:       "create or edit proposals",
	PermProposalsDelete:      "delete proposals",
	PermProposalsGenerate:    "generate proposals",
	PermOutreachRead:         "view outreach sequences",
	PermOutreachWrite:        "create or edit outreach sequences",
	PermOutreachDelete:       "delete outreach sequences",
	PermAutomationsRead:      "view automations",
	PermAutomationsWrite:     "create or edit automations",
	PermAutomationsDelete:    "delete automations",
	PermIntegrationsRead:     "view integrations",
	PermIntegrationsWrite:    "connect or edit integrations",
	PermIntegrationsDelete:   "remove integrations",
	PermWebhooksRead:         "view webhooks",
	PermWebhooksWrite:        "create or edit webhooks",
	PermWebhooksDelete:       "delete webhooks",
	PermAnalyticsRead:        "view analytics",
	PermAIQuery:              "use the AI assistant",
	PermSettingsRead:         "view organization settings",
	PermSettingsWrite:        "change organization settings",
	PermUsersRead:            "view users",
	PermUsersWrite:           "create or edit users",
	PermUsersDelete:          "delete users",
	PermFilesUpload:          "upload files",
	PermFilesRead:            "view files",
	PermFilesDelete:          "delete files",
	PermActivitiesRead:       "view activity history",
	PermActivitiesWrite:      "log activity",
	PermCommunicationsRead:   "view communications",
	PermCommunicationsWrite:  "send or edit communications",
	PermCommunicationsDelete: "delete communications",
	PermKnowledgeRead:        "view the knowledge base",
	PermKnowledgeWrite:       "edit the knowledge base",
	PermNotificationsRead:    "view notifications",
	PermNotificationsWrite:   "manage notifications",
	PermTeamRead:             "view team members",
	PermTeamManage:           "manage team members and roles",
	PermImportsWrite:         "import data",
	PermEmailsSend:           "send emails",
}

// permissionDeniedMessage renders a 403 body for a missing permission in
// plain English, e.g. "You don't have permission to change organization
// settings. Ask an Owner or Admin in your organization to grant you
// access." Falls back to a generic phrasing for any permission key not
// in permissionDescriptions (new permissions added later without an
// entry here still get a sensible message instead of an error).
func permissionDeniedMessage(missing string) string {
	action, ok := permissionDescriptions[missing]
	if !ok {
		action = "do that"
	}
	return fmt.Sprintf("You don't have permission to %s. Ask an Owner or Admin in your organization to grant you access.", action)
}
