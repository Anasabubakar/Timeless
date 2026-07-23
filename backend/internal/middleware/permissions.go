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
	PermFilesDelete = "files:delete"

	PermAll = "*"
)

var AdminPermissions = []string{PermAll}

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
	PermFilesUpload,
}

var ViewerPermissions = []string{
	PermCampaignsRead,
	PermSponsorsRead,
	PermCompaniesRead,
	PermContactsRead,
	PermProposalsRead,
	PermAnalyticsRead,
	PermSettingsRead,
}
