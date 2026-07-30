package router

import (
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/timeout"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/ai/agent"
	"github.com/timeless/backend/internal/ai/memory"
	"github.com/timeless/backend/internal/ai/provider"
	"github.com/timeless/backend/internal/config"
	"github.com/timeless/backend/internal/email"
	"github.com/timeless/backend/internal/eventbus"
	"github.com/timeless/backend/internal/handler"
	"github.com/timeless/backend/internal/integration"
	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/realtime"
	"github.com/timeless/backend/internal/repository"
	"github.com/timeless/backend/internal/security"
	"github.com/timeless/backend/internal/service"
	"github.com/timeless/backend/internal/storage"
	"github.com/timeless/backend/internal/worker"
)

// aiRequestTimeout bounds how long a single AI-provider-calling request
// can run. Without it, a hung upstream provider call is only ever
// bounded by the connection-level WriteTimeout (10-30s depending on
// entrypoint) — fine for most routes, but generous enough that an AI
// call could tie up a handler goroutine for the practical duration of
// that timeout on every request, compounding under load.
const aiRequestTimeout = 45 * time.Second

func Setup(app *fiber.App, db *gorm.DB, rdb *redis.Client, cfg *config.Config, workerClient *worker.Client) {
	// Health/readiness/liveness. /health is kept as an alias of /health/live
	// for anything (existing uptime monitors, load balancer config) still
	// pointed at the old single endpoint.
	healthHandler := handler.NewHealthHandler(db, rdb)
	app.Get("/health", healthHandler.Live)
	app.Get("/health/live", healthHandler.Live)
	app.Get("/health/ready", healthHandler.Ready)

	// Middleware instances
	authMw := middleware.NewAuth(cfg)
	tenantMw := middleware.NewTenant(db)
	rl := middleware.NewRedisRateLimiter(rdb, db)
	idempotent := middleware.Idempotency(rdb, 10*time.Minute)

	// Email Provider Registry (built early: AuthService needs it for
	// verification/reset emails, well before the /emails endpoints below).
	emailRegistry := email.NewRegistry()
	if cfg.SMTPHost != "" {
		emailRegistry.Register(email.NewSMTP(email.SMTPConfig{
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			Username: cfg.SMTPUser,
			Password: cfg.SMTPPassword,
			FromAddr: cfg.SMTPFrom,
			FromName: cfg.SMTPFromName,
			UseTLS:   cfg.SMTPUseTLS,
		}))
	}
	if cfg.SendGridKey != "" {
		emailRegistry.Register(email.NewSendGrid(cfg.SendGridKey))
	}
	emailSender := email.NewSender(emailRegistry)

	// Repositories
	userRepo := repository.NewUserRepository(db)
	orgRepo := repository.NewOrganizationRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	emailVerifyRepo := repository.NewEmailVerificationRepository(db)
	passwordResetRepo := repository.NewPasswordResetRepository(db)
	roleRepo := repository.NewRoleRepository(db)

	// Services
	authSvc := service.NewAuthService(userRepo, orgRepo, sessionRepo, emailVerifyRepo, passwordResetRepo, roleRepo, cfg, rdb, emailSender, db)
	invitationRepo := repository.NewInvitationRepository(db)
	invSvc := service.NewInvitationService(invitationRepo, roleRepo, userRepo, orgRepo, authSvc, emailSender, db)

	// Handlers
	authHandler := handler.NewAuthHandler(authSvc)
	invitationHandler := handler.NewInvitationHandler(invSvc)

	// Public routes
	api := app.Group("/api/v1", middleware.WithAPIVersion)
	auth := api.Group("/auth", middleware.MaxBodySize(16*1024))
	auth.Post("/register", authHandler.Register, rl.Limit(middleware.RateLimitRegister()))
	auth.Get("/organizations/lookup", authHandler.LookupOrganization, rl.Limit(middleware.RateLimitOrgLookup()))
	auth.Post("/join", authHandler.Join, rl.Limit(middleware.RateLimitJoin()), rl.Limit(middleware.RateLimitJoinByOrg()))
	auth.Post("/login", authHandler.Login, rl.Limit(middleware.RateLimitLogin()), rl.Limit(middleware.RateLimitLoginByAccount()))
	auth.Post("/refresh", authHandler.RefreshToken, rl.Limit(middleware.RateLimitRefresh()))
	api.Post("/invitations/accept", invitationHandler.Accept, rl.Limit(middleware.RateLimitRegister()))
	auth.Post("/verify-email", authHandler.VerifyEmail, rl.Limit(middleware.RateLimitEmailVerification()))
	auth.Post("/resend-verification", authHandler.ResendVerification, rl.Limit(middleware.RateLimitEmailVerification()), rl.Limit(middleware.RateLimitEmailVerificationByAccount()))
	auth.Post("/forgot-password", authHandler.ForgotPassword, rl.Limit(middleware.RateLimitPasswordReset()), rl.Limit(middleware.RateLimitPasswordResetByAccount()))
	auth.Post("/reset-password", authHandler.ResetPassword, rl.Limit(middleware.RateLimitPasswordReset()))
	auth.Post("/mfa/verify-login", authHandler.VerifyMFALogin, rl.Limit(middleware.RateLimitMFAVerify()))

	// Integrations service (constructed here, ahead of the `protected`
	// group below, because the OAuth and webhook routes right after it
	// must themselves be registered before `protected` exists — see the
	// comment on that group for why).
	integrationRepo := repository.NewIntegrationRepository(db)
	syncRunRepo := repository.NewSyncRunRepository(db)
	credentialCipher := security.NewCredentialCipher(cfg.CredentialKey(), cfg.CredentialsEncryptionKeyPrevious...)
	registryCfg := integration.RegistryConfig{NotionClientID: cfg.NotionClientID, NotionClientSecret: cfg.NotionClientSecret}
	integrationSvc := service.NewIntegrationService(integrationRepo, syncRunRepo, credentialCipher, workerClient, registryCfg)

	// Event bus: entity services publish here on create/update/delete;
	// SetPublisher routes every Publish through asynq (same worker queue
	// as everything else) so an event survives a process restart instead
	// of only ever firing in-process. Subscribers (Notion push, future
	// Zapier/Apollo adapters) register with bus.Subscribe from their own
	// wiring point once those adapters exist.
	bus := eventbus.NewBus()
	if workerClient != nil {
		bus.SetPublisher(worker.NewEventPublisher(workerClient))
	}

	// OAuth (public: browser redirects can't carry auth headers). This
	// authenticates the start leg via a JWT query param instead, and the
	// callback leg via the state value minted at start time.
	oauthHandler := handler.NewOAuthHandler(cfg, rdb, integrationSvc, db)
	api.Get("/integrations/oauth/callback", oauthHandler.Callback, rl.Limit(middleware.RateLimitOAuth()))
	api.Get("/integrations/:provider/oauth/start", oauthHandler.Start, rl.Limit(middleware.RateLimitOAuth()))

	// Notion webhooks (public: Notion calls this directly, verified via
	// HMAC signature rather than our JWT auth)
	notionWebhookHandler := handler.NewNotionWebhookHandler(rdb, integrationSvc)
	api.Post("/integrations/notion/webhook", notionWebhookHandler.Receive, rl.Limit(middleware.RateLimitWebhookInbound()))

	// Protected routes
	//
	// IMPORTANT: this Group call uses an empty prefix ("") purely to attach
	// middleware to every route registered on `protected` from here on —
	// but Fiber implements that as app.Use("/api/v1", ...handlers), which
	// matches by path prefix against routes added AFTER this point in
	// registration order, regardless of which Group variable a later route
	// is added through. Any route meant to stay public (no auth/tenant/
	// RouteGuard middleware — see publicAPIRoutes below) MUST be registered
	// on `api` above this line, or it will silently start requiring a
	// bearer token. This bit Notion OAuth (both legs) and the Notion
	// webhook receiver in production: they were registered on `api` after
	// this group existed, so authMw rejected every request with "missing
	// authorization header" before the handler — which does its own JWT
	// parsing from a query param — ever ran.
	auditMw := middleware.AuditLog(middleware.AuditConfig{DB: db})
	rbacMw := middleware.NewRBAC(db)
	routeGuard := middleware.NewRouteGuard(rbacMw)
	protected := api.Group("", authMw.Handle, middleware.ValidateOrigin(cfg.CORSOrigins()), rl.Limit(middleware.RateLimitAPI()), tenantMw.Handle, auditMw, routeGuard.Handle)

	// Auth (protected)
	protected.Post("/auth/logout", authHandler.Logout)
	protected.Get("/auth/me", authHandler.Me)
	protected.Post("/auth/mfa/enroll", authHandler.EnrollMFA)
	protected.Post("/auth/mfa/confirm", authHandler.ConfirmMFA, rl.Limit(middleware.RateLimitMFAManage()))
	protected.Post("/auth/mfa/disable", authHandler.DisableMFA, rl.Limit(middleware.RateLimitMFAManage()))
	protected.Get("/auth/sessions", authHandler.ListSessions)
	protected.Delete("/auth/sessions/:id", authHandler.RevokeSession)
	protected.Post("/auth/sessions/revoke-all", authHandler.LogoutAllSessions)

	// Organizations
	orgHandler := handler.NewOrganizationHandler(service.NewOrganizationService(orgRepo, roleRepo, userRepo, cfg, db))
	protected.Get("/organizations/current", orgHandler.GetCurrent)
	protected.Patch("/organizations/current", orgHandler.Update, rl.Limit(middleware.RateLimitOrgSettings()))
	protected.Post("/organizations/current/transfer-ownership", orgHandler.TransferOwnership, rl.Limit(middleware.RateLimitOrgSettings()))

	// Profile
	profileHandler := handler.NewProfileHandler(userRepo, authSvc)
	protected.Patch("/profile", profileHandler.Update)
	protected.Post("/profile/password", profileHandler.ChangePassword)

	// Onboarding
	onboardingRepo := repository.NewOnboardingRepository(db)
	onboardingSvc := service.NewOnboardingService(onboardingRepo, userRepo)
	onboardingHandler := handler.NewOnboardingHandler(onboardingSvc)
	protected.Get("/onboarding/state", onboardingHandler.GetState)
	protected.Patch("/onboarding/state", onboardingHandler.SaveState)
	protected.Post("/onboarding/complete", onboardingHandler.Complete)

	// Companies
	companyRepo := repository.NewCompanyRepository(db)
	companySvc := service.NewCompanyService(companyRepo).SetBus(bus)
	companyHandler := handler.NewCompanyHandler(companySvc)
	companies := protected.Group("/companies", middleware.MaxBodySize(256*1024))
	companies.Get("/", companyHandler.List)
	companies.Post("/", companyHandler.Create)
	companies.Get("/:id", companyHandler.Get)
	companies.Patch("/:id", companyHandler.Update)
	companies.Delete("/:id", companyHandler.Delete)

	// Campaigns
	campaignRepo := repository.NewCampaignRepository(db)
	campaignSvc := service.NewCampaignService(campaignRepo)
	campaignHandler := handler.NewCampaignHandler(campaignSvc)
	campaigns := protected.Group("/campaigns", middleware.MaxBodySize(256*1024))
	campaigns.Get("/", campaignHandler.List)
	campaigns.Post("/", campaignHandler.Create)
	campaigns.Get("/:id", campaignHandler.Get)
	campaigns.Patch("/:id", campaignHandler.Update)
	campaigns.Delete("/:id", campaignHandler.Delete)

	// Sponsors
	sponsorRepo := repository.NewSponsorRepository(db)
	sponsorSvc := service.NewSponsorService(sponsorRepo).SetBus(bus)
	sponsorHandler := handler.NewSponsorHandler(sponsorSvc)
	sponsors := protected.Group("/sponsors", middleware.MaxBodySize(256*1024))
	sponsors.Get("/", sponsorHandler.List)
	sponsors.Post("/", sponsorHandler.Create)
	sponsors.Get("/:id", sponsorHandler.Get)
	sponsors.Patch("/:id", sponsorHandler.Update)
	sponsors.Delete("/:id", sponsorHandler.Delete)
	sponsors.Patch("/:id/stage", sponsorHandler.UpdateStage)

	// Contacts
	contactRepo := repository.NewContactRepository(db)
	contactSvc := service.NewContactService(contactRepo).SetBus(bus)
	contactHandler := handler.NewContactHandler(contactSvc)
	contacts := protected.Group("/contacts", middleware.MaxBodySize(256*1024))
	contacts.Get("/", contactHandler.List)
	contacts.Post("/", contactHandler.Create)
	contacts.Get("/:id", contactHandler.Get)
	contacts.Patch("/:id", contactHandler.Update)
	contacts.Delete("/:id", contactHandler.Delete)

	// Activities
	activityRepo := repository.NewActivityRepository(db)
	activitySvc := service.NewActivityService(activityRepo)
	activityHandler := handler.NewActivityHandler(activitySvc)
	activities := protected.Group("/activities")
	activities.Get("/", activityHandler.List)
	activities.Post("/", activityHandler.Create)

	// Outreach Sequences
	outreachRepo := repository.NewOutreachRepository(db)
	outreachSvc := service.NewOutreachService(outreachRepo)
	outreachHandler := handler.NewOutreachHandler(outreachSvc)
	sequences := protected.Group("/sequences")
	sequences.Get("/", outreachHandler.ListSequences)
	sequences.Post("/", outreachHandler.CreateSequence)
	sequences.Get("/:id", outreachHandler.GetSequence)
	sequences.Patch("/:id", outreachHandler.UpdateSequence)
	sequences.Delete("/:id", outreachHandler.DeleteSequence)
	sequences.Post("/:id/enroll", outreachHandler.Enroll)

	// AI Provider Registry (needed by proposals + AI endpoints).
	// Default chain, in priority order: Gemini -> Groq -> Nvidia -> OpenRouter.
	// Each hop is only included if its key is configured; if a hop errors at
	// request time (rate limit, outage, etc.) the chain automatically falls
	// through to the next one instead of failing the whole request.
	registry := provider.NewRegistry()
	var defaultChain []provider.Provider
	if cfg.GeminiKey != "" {
		p := provider.NewGemini(cfg.GeminiKey)
		registry.Register(p)
		defaultChain = append(defaultChain, p)
	}
	if cfg.GroqKey != "" {
		p := provider.NewGroq(cfg.GroqKey)
		registry.Register(p)
		defaultChain = append(defaultChain, p)
	}
	if cfg.NvidiaKey != "" {
		p := provider.NewNvidia(cfg.NvidiaKey, cfg.NvidiaBaseURL)
		registry.Register(p)
		defaultChain = append(defaultChain, p)
	}
	if cfg.OpenRouterKey != "" {
		p := provider.NewOpenRouter(cfg.OpenRouterKey)
		registry.Register(p)
		defaultChain = append(defaultChain, p)
	}
	if cfg.OpenAIKey != "" {
		registry.Register(provider.NewOpenAI(cfg.OpenAIKey))
	}
	if len(defaultChain) > 0 {
		registry.Register(provider.NewFallbackChain(defaultChain...))
		registry.SetDefault("auto")
	}

	// Proposals
	proposalRepo := repository.NewProposalRepository(db)
	var proposalAIProvider provider.Provider
	if p, err := registry.Get("openrouter"); err == nil {
		proposalAIProvider = p
	} else if p, err := registry.Get("openai"); err == nil {
		proposalAIProvider = p
	}
	proposalSvc := service.NewProposalService(proposalRepo, sponsorRepo, companyRepo, proposalAIProvider)
	proposalHandler := handler.NewProposalHandler(proposalSvc)
	proposals := protected.Group("/proposals", middleware.MaxBodySize(256*1024))
	proposals.Get("/", proposalHandler.List)
	proposals.Post("/", proposalHandler.Create)
	proposals.Post("/generate", timeout.New(proposalHandler.Generate, aiRequestTimeout), idempotent)
	proposals.Get("/:id", proposalHandler.Get)
	proposals.Patch("/:id", proposalHandler.Update)
	proposals.Delete("/:id", proposalHandler.Delete)

	// Integrations (service constructed earlier, alongside the public
	// OAuth/webhook routes above)
	integrationHandler := handler.NewIntegrationHandler(integrationSvc)
	integrations := protected.Group("/integrations")
	integrations.Get("/", integrationHandler.List)
	integrations.Get("/dashboard", integrationHandler.Dashboard)
	integrations.Get("/zapier/apps", integrationHandler.ZapierApps)
	integrations.Post("/", integrationHandler.Create)
	integrations.Get("/:id", integrationHandler.Get)
	integrations.Patch("/:id", integrationHandler.Update)
	integrations.Delete("/:id", integrationHandler.Delete)
	integrations.Post("/:id/revoke", integrationHandler.Revoke)
	integrations.Post("/:id/sync", integrationHandler.TriggerSync, idempotent)
	integrations.Post("/:provider/connect", integrationHandler.Connect)
	integrations.Patch("/notion/pages/:pageID", integrationHandler.PushNotionPage)
	integrations.Post("/rotate-credentials", integrationHandler.RotateCredentials)

	dedupeHandler := handler.NewDedupeHandler(db)
	protected.Post("/companies/dedupe", dedupeHandler.MergeCompanies)

	// Automations
	automationRepo := repository.NewAutomationRepository(db)
	automationSvc := service.NewAutomationService(automationRepo)
	automationHandler := handler.NewAutomationHandler(automationSvc)
	automations := protected.Group("/automations")
	automations.Get("/", automationHandler.List)
	automations.Post("/", automationHandler.Create)
	automations.Get("/:id", automationHandler.Get)
	automations.Patch("/:id", automationHandler.Update)
	automations.Delete("/:id", automationHandler.Delete)
	automations.Patch("/:id/toggle", automationHandler.Toggle)

	// Webhooks
	webhookRepo := repository.NewWebhookRepository(db)
	webhookSvc := service.NewWebhookService(webhookRepo, workerClient)
	webhookHandler := handler.NewWebhookHandler(webhookSvc)
	webhooks := protected.Group("/webhooks")
	webhooks.Get("/", webhookHandler.List)
	webhooks.Post("/", webhookHandler.Create)
	webhooks.Get("/:id", webhookHandler.Get)
	webhooks.Patch("/:id", webhookHandler.Update)
	webhooks.Delete("/:id", webhookHandler.Delete)
	webhooks.Post("/:id/rotate-secret", webhookHandler.RotateSecret)
	webhooks.Post("/:id/test", webhookHandler.Test)
	webhooks.Get("/:id/deliveries", webhookHandler.Deliveries)

	// Communications
	commRepo := repository.NewCommunicationRepository(db)
	commSvc := service.NewCommunicationService(commRepo)
	commHandler := handler.NewCommunicationHandler(commSvc)
	communications := protected.Group("/communications")
	communications.Get("/", commHandler.List)
	communications.Post("/", commHandler.Create)
	communications.Get("/stats", commHandler.Stats)
	communications.Get("/:id", commHandler.Get)
	communications.Patch("/:id", commHandler.Update)
	communications.Delete("/:id", commHandler.Delete)

	// Analytics
	analyticsHandler := handler.NewAnalyticsHandler(db)
	analytics := protected.Group("/analytics")
	analytics.Get("/dashboard", analyticsHandler.Dashboard)
	analytics.Get("/pipeline", analyticsHandler.Pipeline)
	analytics.Get("/activity", analyticsHandler.Activity)
	analytics.Get("/timeseries", analyticsHandler.TimeSeries)
	analytics.Get("/funnel", analyticsHandler.PipelineFunnel)
	analytics.Get("/velocity", analyticsHandler.DealVelocity)
	analytics.Get("/export/sponsors", analyticsHandler.ExportSponsors)
	analytics.Get("/export/campaigns", analyticsHandler.ExportCampaigns)

	// AI Agents
	learningService := agent.NewLearningService(db)
	orchestrator := agent.NewOrchestrator(registry, learningService)
	aiHandler := handler.NewAIHandler(orchestrator)
	ai := protected.Group("/ai", middleware.MaxBodySize(32*1024), rl.Limit(middleware.RateLimitAIPerUser()), rl.Limit(middleware.RateLimitAIPerOrg()))
	ai.Post("/query", timeout.New(aiHandler.Query, aiRequestTimeout), idempotent)
	ai.Get("/agents", aiHandler.ListAgents)

	// AI Learning & Feedback
	learningHandler := handler.NewAgentLearningHandler(learningService)
	ai.Post("/outcomes", learningHandler.RecordOutcome)
	ai.Get("/outcomes", learningHandler.GetOutcomes)
	ai.Post("/feedback", learningHandler.SubmitFeedback)
	ai.Post("/preferences", learningHandler.StorePreference)
	ai.Get("/preferences", learningHandler.GetPreferences)

	// Onboarding: AI workspace discovery, goal recommendation, automation planning
	projectRepo := repository.NewProjectRepository(db)
	discoverySvc := service.NewDiscoveryService(integrationRepo, projectRepo, orchestrator)
	goalSvc := service.NewGoalService(orchestrator)
	automationPlanSvc := service.NewAutomationPlanService(orchestrator, automationRepo)
	discoveryHandler := handler.NewDiscoveryHandler(discoverySvc, goalSvc, automationPlanSvc)
	aiOnboardingLimits := []fiber.Handler{rl.Limit(middleware.RateLimitAIPerUser()), rl.Limit(middleware.RateLimitAIPerOrg())}
	protected.Post("/onboarding/discovery/run", timeout.New(discoveryHandler.RunDiscovery, aiRequestTimeout), aiOnboardingLimits...)
	protected.Post("/onboarding/discovery/select", timeout.New(discoveryHandler.SelectProjects, aiRequestTimeout), aiOnboardingLimits...)
	protected.Post("/onboarding/goals/recommend", timeout.New(discoveryHandler.RecommendGoals, aiRequestTimeout), aiOnboardingLimits...)
	protected.Post("/onboarding/goals/plan", timeout.New(discoveryHandler.PlanAutomation, aiRequestTimeout), aiOnboardingLimits...)
	protected.Post("/onboarding/goals/approve", timeout.New(discoveryHandler.ApproveAutomation, aiRequestTimeout), aiOnboardingLimits...)

	// Knowledge Graph & Semantic Search
	var embedder provider.Embedder
	if p, err := registry.Get("openai"); err == nil {
		if e, ok := p.(provider.Embedder); ok {
			embedder = e
		}
	}
	memoryStore := memory.NewStore(db, embedder)
	knowledgeHandler := handler.NewKnowledgeHandler(memoryStore)
	knowledge := protected.Group("/knowledge")
	knowledge.Get("/search", knowledgeHandler.SemanticSearch)
	knowledge.Get("/memories", knowledgeHandler.SearchMemories)
	knowledge.Post("/memories", knowledgeHandler.StoreMemory)
	knowledge.Post("/nodes", knowledgeHandler.AddNode)
	knowledge.Post("/edges", knowledgeHandler.AddEdge)
	knowledge.Get("/nodes/:id/neighbors", knowledgeHandler.GetNeighbors)

	// File Uploads
	var store storage.Storage
	if cfg.S3Endpoint != "" {
		s, err := storage.NewMinIO(cfg)
		if err == nil {
			store = s
		}
	}
	uploadHandler := handler.NewUploadHandler(store, storage.NoopScanner{})
	files := protected.Group("/files", rl.Limit(middleware.RateLimitUploads()))
	files.Post("/upload", uploadHandler.Upload)
	files.Delete("/", uploadHandler.Delete)

	// Search
	searchHandler := handler.NewSearchHandler(db)
	protected.Get("/search", searchHandler.Search)

	// Import
	importHandler := handler.NewImportHandler(db)
	imports := protected.Group("/import", rl.Limit(middleware.RateLimitImports()))
	imports.Post("/companies", importHandler.ImportCompanies)
	imports.Post("/contacts", importHandler.ImportContacts)
	imports.Post("/sponsors", importHandler.ImportSponsors)

	// Batch Operations
	batchHandler := handler.NewBatchHandler(db)
	protected.Post("/sponsors/batch/update", batchHandler.BatchUpdate("sponsors"))
	protected.Post("/sponsors/batch/delete", batchHandler.BatchDelete("sponsors"))
	protected.Post("/companies/batch/update", batchHandler.BatchUpdate("companies"))
	protected.Post("/companies/batch/delete", batchHandler.BatchDelete("companies"))
	protected.Post("/contacts/batch/update", batchHandler.BatchUpdate("contacts"))
	protected.Post("/contacts/batch/delete", batchHandler.BatchDelete("contacts"))

	// Team Management
	teamHandler := handler.NewTeamHandler(db, roleRepo, rbacMw, invSvc)
	team := protected.Group("/team")
	team.Get("/members", teamHandler.ListMembers)
	team.Post("/members", teamHandler.InviteMember)
	team.Patch("/members/:id/roles", teamHandler.UpdateMemberRole)
	team.Delete("/members/:id", teamHandler.RemoveMember)
	team.Get("/roles", teamHandler.ListRoles)
	team.Get("/invitations", teamHandler.ListPendingInvitations)
	team.Delete("/invitations/:id", teamHandler.RevokeInvitation)

	// Email endpoints
	emailHandler := handler.NewEmailHandler(emailSender, workerClient)
	emails := protected.Group("/emails", rl.Limit(middleware.RateLimitEmailsSend()))
	emails.Post("/send", emailHandler.Send)
	emails.Post("/send-direct", emailHandler.SendDirect)
	emails.Post("/send-template", emailHandler.SendTemplate)

	// Realtime (WebSocket + SSE)
	hub := realtime.NewHub()
	go hub.Run()
	// Handlers are attached directly to this one route (not via a Group
	// with an empty prefix) so they only ever run for "/ws" itself. An
	// empty-prefix Group registers its middleware as a fiber Use at the
	// parent's own prefix — for app.Group("", ...) that's the app root,
	// matching every route registered afterward — which is exactly the
	// bug that broke Notion OAuth (see the comment on `protected` above)
	// and, here, silently made every "/api/v1/notifications/*" and
	// "/api/v1/events" route (registered after this one) require a `token`
	// query param instead of the normal Authorization header, since they
	// inherited authMw.HandleWS instead of the real auth middleware.
	// fiber assembles a route's Handlers as [middleware..., handler], so
	// authMw.HandleWS still runs first, before the WebSocket upgrade.
	//
	// WebSocket upgrades aren't subject to CORS/SOP the way a normal
	// fetch() is — a page on any origin can attempt to open one. This
	// app's token-in-query-param auth means an attacker's page can't
	// silently attach a victim's credentials the way a cookie-based app
	// would be at risk of (that's the actual CSWSH threat model), but
	// checking Origin on the upgrade is a cheap additional layer
	// consistent with the rest of the API.
	app.Get("/ws", hub.WebSocketHandler(), authMw.HandleWS, middleware.ValidateOriginAlways(cfg.CORSOrigins()), tenantMw.Handle)
	protected.Get("/events", hub.SSEHandler())

	// Notifications
	notifRepo := repository.NewNotificationRepository(db)
	notifSvc := service.NewNotificationService(notifRepo, hub)
	notifHandler := handler.NewNotificationHandler(notifSvc)
	notifications := protected.Group("/notifications")
	notifications.Get("/", notifHandler.List)
	notifications.Get("/count", notifHandler.UnreadCount)
	notifications.Post("/read-all", notifHandler.MarkAllRead)
	notifications.Patch("/:id/read", notifHandler.MarkRead)
	notifications.Delete("/:id", notifHandler.Delete)
	notifications.Get("/preferences", notifHandler.GetPreferences)
	notifications.Put("/preferences", notifHandler.UpdatePreference)

	verifyRouteGuardCoverage(app)
}

// publicAPIRoutes are the only /api/v1 routes intentionally registered
// outside the `protected` group (no auth/tenant/RouteGuard middleware) —
// login/registration and things browsers/external services must reach
// without a bearer token (OAuth redirects, the Notion webhook, which is
// verified by HMAC signature instead of a session).
var publicAPIRoutes = map[string]bool{
	"POST /api/v1/auth/register":                     true,
	"GET /api/v1/auth/organizations/lookup":          true,
	"POST /api/v1/auth/join":                         true,
	"POST /api/v1/invitations/accept":                true,
	"POST /api/v1/auth/login":                        true,
	"POST /api/v1/auth/refresh":                      true,
	"POST /api/v1/auth/verify-email":                 true,
	"POST /api/v1/auth/resend-verification":          true,
	"POST /api/v1/auth/forgot-password":              true,
	"POST /api/v1/auth/reset-password":               true,
	"POST /api/v1/auth/mfa/verify-login":             true,
	"GET /api/v1/integrations/oauth/callback":        true,
	"GET /api/v1/integrations/:provider/oauth/start": true,
	"POST /api/v1/integrations/notion/webhook":       true,
}

// verifyRouteGuardCoverage walks every registered route at boot and
// fails loudly if a /api/v1 route is neither in publicAPIRoutes nor in
// RouteGuard's permission table — i.e. it would 403 every request, which
// almost certainly means someone added a route and forgot to classify
// it. Catching that at startup beats catching it when a customer's
// first request to a brand-new endpoint mysteriously 403s.
func verifyRouteGuardCoverage(app *fiber.App) {
	for _, route := range app.GetRoutes() {
		if !strings.HasPrefix(route.Path, "/api/v1/") {
			continue
		}
		key := route.Method + " " + route.Path
		if publicAPIRoutes[key] {
			continue
		}
		if _, ok := middleware.RoutePermission(key); !ok {
			log.Printf("router: WARNING — %s has no RouteGuard permission entry and is not in publicAPIRoutes; it will 403 every request until classified", key)
		}
	}
}
