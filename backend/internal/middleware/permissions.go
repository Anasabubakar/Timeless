package middleware

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
