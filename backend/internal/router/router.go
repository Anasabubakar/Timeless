package router

import (
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/ai/agent"
	"github.com/timeless/backend/internal/ai/memory"
	"github.com/timeless/backend/internal/ai/provider"
	"github.com/timeless/backend/internal/config"
	"github.com/timeless/backend/internal/email"
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

func Setup(app *fiber.App, db *gorm.DB, rdb *redis.Client, cfg *config.Config, workerClient *worker.Client) {
	// Health check
	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "timeless-api"})
	})

	// Middleware instances
	authMw := middleware.NewAuth(cfg)
	tenantMw := middleware.NewTenant(db)

	// Repositories
	userRepo := repository.NewUserRepository(db)
	orgRepo := repository.NewOrganizationRepository(db)

	// Services
	authSvc := service.NewAuthService(userRepo, orgRepo, cfg, rdb)

	// Handlers
	authHandler := handler.NewAuthHandler(authSvc)

	// Public routes
	api := app.Group("/api/v1")
	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/refresh", authHandler.RefreshToken)

	// Protected routes
	auditMw := middleware.AuditLog(middleware.AuditConfig{DB: db})
	protected := api.Group("", authMw.Handle, tenantMw.Handle, auditMw)

	// Auth (protected)
	protected.Post("/auth/logout", authHandler.Logout)
	protected.Get("/auth/me", authHandler.Me)

	// Organizations
	orgHandler := handler.NewOrganizationHandler(service.NewOrganizationService(orgRepo))
	protected.Get("/organizations/current", orgHandler.GetCurrent)
	protected.Patch("/organizations/current", orgHandler.Update)

	// Profile
	profileHandler := handler.NewProfileHandler(userRepo)
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
	companySvc := service.NewCompanyService(companyRepo)
	companyHandler := handler.NewCompanyHandler(companySvc)
	companies := protected.Group("/companies")
	companies.Get("/", companyHandler.List)
	companies.Post("/", companyHandler.Create)
	companies.Get("/:id", companyHandler.Get)
	companies.Patch("/:id", companyHandler.Update)
	companies.Delete("/:id", companyHandler.Delete)

	// Campaigns
	campaignRepo := repository.NewCampaignRepository(db)
	campaignSvc := service.NewCampaignService(campaignRepo)
	campaignHandler := handler.NewCampaignHandler(campaignSvc)
	campaigns := protected.Group("/campaigns")
	campaigns.Get("/", campaignHandler.List)
	campaigns.Post("/", campaignHandler.Create)
	campaigns.Get("/:id", campaignHandler.Get)
	campaigns.Patch("/:id", campaignHandler.Update)
	campaigns.Delete("/:id", campaignHandler.Delete)

	// Sponsors
	sponsorRepo := repository.NewSponsorRepository(db)
	sponsorSvc := service.NewSponsorService(sponsorRepo)
	sponsorHandler := handler.NewSponsorHandler(sponsorSvc)
	sponsors := protected.Group("/sponsors")
	sponsors.Get("/", sponsorHandler.List)
	sponsors.Post("/", sponsorHandler.Create)
	sponsors.Get("/:id", sponsorHandler.Get)
	sponsors.Patch("/:id", sponsorHandler.Update)
	sponsors.Delete("/:id", sponsorHandler.Delete)
	sponsors.Patch("/:id/stage", sponsorHandler.UpdateStage)

	// Contacts
	contactRepo := repository.NewContactRepository(db)
	contactSvc := service.NewContactService(contactRepo)
	contactHandler := handler.NewContactHandler(contactSvc)
	contacts := protected.Group("/contacts")
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
	proposals := protected.Group("/proposals")
	proposals.Get("/", proposalHandler.List)
	proposals.Post("/", proposalHandler.Create)
	proposals.Post("/generate", proposalHandler.Generate)
	proposals.Get("/:id", proposalHandler.Get)
	proposals.Patch("/:id", proposalHandler.Update)
	proposals.Delete("/:id", proposalHandler.Delete)

	// Integrations
	integrationRepo := repository.NewIntegrationRepository(db)
	syncRunRepo := repository.NewSyncRunRepository(db)
	credentialCipher := security.NewCredentialCipher(cfg.JWTSecret)
	registryCfg := integration.RegistryConfig{NotionClientID: cfg.NotionClientID, NotionClientSecret: cfg.NotionClientSecret}
	integrationSvc := service.NewIntegrationService(integrationRepo, syncRunRepo, credentialCipher, workerClient, registryCfg)
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
	integrations.Post("/:provider/connect", integrationHandler.Connect)
	integrations.Patch("/notion/pages/:pageID", integrationHandler.PushNotionPage)

	dedupeHandler := handler.NewDedupeHandler(db)
	protected.Post("/companies/dedupe", dedupeHandler.MergeCompanies)

	// OAuth (public: browser redirects can't carry auth headers)
	oauthHandler := handler.NewOAuthHandler(cfg, rdb, integrationSvc)
	api.Get("/integrations/oauth/callback", oauthHandler.Callback)
	api.Get("/integrations/:provider/oauth/start", oauthHandler.Start)

	// Notion webhooks (public: Notion calls this directly, verified via
	// HMAC signature rather than our JWT auth)
	notionWebhookHandler := handler.NewNotionWebhookHandler(rdb, integrationSvc)
	api.Post("/integrations/notion/webhook", notionWebhookHandler.Receive)

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
	ai := protected.Group("/ai")
	ai.Post("/query", aiHandler.Query)
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
	protected.Post("/onboarding/discovery/run", discoveryHandler.RunDiscovery)
	protected.Post("/onboarding/discovery/select", discoveryHandler.SelectProjects)
	protected.Post("/onboarding/goals/recommend", discoveryHandler.RecommendGoals)
	protected.Post("/onboarding/goals/plan", discoveryHandler.PlanAutomation)
	protected.Post("/onboarding/goals/approve", discoveryHandler.ApproveAutomation)

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
	uploadHandler := handler.NewUploadHandler(store)
	files := protected.Group("/files")
	files.Post("/upload", uploadHandler.Upload)
	files.Delete("/", uploadHandler.Delete)

	// Search
	searchHandler := handler.NewSearchHandler(db)
	protected.Get("/search", searchHandler.Search)

	// Import
	importHandler := handler.NewImportHandler(db)
	imports := protected.Group("/import")
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
	teamHandler := handler.NewTeamHandler(db)
	team := protected.Group("/team")
	team.Get("/members", teamHandler.ListMembers)
	team.Post("/members", teamHandler.InviteMember)
	team.Patch("/members/:id/roles", teamHandler.UpdateMemberRole)
	team.Delete("/members/:id", teamHandler.RemoveMember)
	team.Get("/roles", teamHandler.ListRoles)

	// Email Provider Registry
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

	// Email endpoints
	emailHandler := handler.NewEmailHandler(emailSender, workerClient)
	emails := protected.Group("/emails")
	emails.Post("/send", emailHandler.Send)
	emails.Post("/send-direct", emailHandler.SendDirect)
	emails.Post("/send-template", emailHandler.SendTemplate)

	// Realtime (WebSocket + SSE)
	hub := realtime.NewHub()
	go hub.Run()
	app.Get("/ws", authMw.Handle, tenantMw.Handle, hub.WebSocketHandler())
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
}
