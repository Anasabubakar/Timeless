package worker

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"github.com/sponsoros/backend/internal/models"
)

const (
	TaskAIResearch       = "ai:research"
	TaskAIQualification  = "ai:qualification"
	TaskAIOutreach       = "ai:outreach"
	TaskCompanyEnrich    = "company:enrich"
	TaskEmailSend        = "email:send"
	TaskSequenceAdvance  = "sequence:advance"
	TaskWebhookDeliver   = "webhook:deliver"
	TaskAnalyticsCompute = "analytics:compute"
	TaskMemoryIndex      = "memory:index"
)

type TaskPayload struct {
	OrgID      string                 `json:"org_id"`
	UserID     string                 `json:"user_id"`
	EntityID   string                 `json:"entity_id,omitempty"`
	EntityType string                 `json:"entity_type,omitempty"`
	Action     string                 `json:"action"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

func NewTask(taskType string, payload TaskPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal task payload: %w", err)
	}
	return asynq.NewTask(taskType, data), nil
}

type Handlers struct {
	logger *slog.Logger
	db     *gorm.DB
}

func NewHandlers(logger *slog.Logger, db *gorm.DB) *Handlers {
	return &Handlers{logger: logger, db: db}
}

func (h *Handlers) parsePayload(t *asynq.Task) (TaskPayload, error) {
	var p TaskPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return p, fmt.Errorf("unmarshal: %w", err)
	}
	return p, nil
}

func (h *Handlers) HandleAIResearch(ctx context.Context, t *asynq.Task) error {
	payload, err := h.parsePayload(t)
	if err != nil {
		return err
	}
	h.logger.Info("processing AI research task",
		"org_id", payload.OrgID,
		"entity_id", payload.EntityID,
	)

	h.db.WithContext(ctx).Create(&models.Activity{
		OrganizationID: uuidFromString(payload.OrgID),
		UserID:         uuidFromString(payload.UserID),
		Type:           "ai_research",
		Subject:        "AI research completed",
		EntityType:     payload.EntityType,
		EntityID:       payload.EntityID,
	})
	return nil
}

func (h *Handlers) HandleAIQualification(ctx context.Context, t *asynq.Task) error {
	payload, err := h.parsePayload(t)
	if err != nil {
		return err
	}
	h.logger.Info("processing AI qualification",
		"org_id", payload.OrgID,
		"entity_id", payload.EntityID,
	)
	return nil
}

func (h *Handlers) HandleAIOutreach(ctx context.Context, t *asynq.Task) error {
	payload, err := h.parsePayload(t)
	if err != nil {
		return err
	}
	h.logger.Info("processing AI outreach",
		"org_id", payload.OrgID,
		"entity_id", payload.EntityID,
	)
	return nil
}

func (h *Handlers) HandleCompanyEnrich(ctx context.Context, t *asynq.Task) error {
	payload, err := h.parsePayload(t)
	if err != nil {
		return err
	}

	companyID := payload.EntityID
	orgID := payload.OrgID

	h.logger.Info("enriching company", "org_id", orgID, "company_id", companyID)

	var company models.Company
	if err := h.db.WithContext(ctx).Where("id = ? AND organization_id = ?", companyID, orgID).First(&company).Error; err != nil {
		h.logger.Error("company not found for enrichment", "company_id", companyID, "error", err)
		return nil
	}

	enriched := h.enrichFromDomain(ctx, &company)

	updates := map[string]interface{}{
		"enrichment_data": enriched,
	}
	if company.Description == nil || *company.Description == "" {
		if desc, ok := enriched["description"].(string); ok && desc != "" {
			updates["description"] = desc
		}
	}
	if company.EmployeeCount == nil || *company.EmployeeCount == "" {
		if emp, ok := enriched["employee_count"].(string); ok && emp != "" {
			updates["employee_count"] = emp
		}
	}
	if company.AnnualRevenue == nil || *company.AnnualRevenue == "" {
		if rev, ok := enriched["annual_revenue"].(string); ok && rev != "" {
			updates["annual_revenue"] = rev
		}
	}
	if company.Headquarters == nil || *company.Headquarters == "" {
		if hq, ok := enriched["headquarters"].(string); ok && hq != "" {
			updates["headquarters"] = hq
		}
	}
	if company.FoundedYear == nil {
		if year, ok := enriched["founded_year"].(int); ok && year > 0 {
			updates["founded_year"] = year
		}
	}
	if company.LinkedinURL == nil || *company.LinkedinURL == "" {
		if li, ok := enriched["linkedin_url"].(string); ok && li != "" {
			updates["linkedin_url"] = li
		}
	}
	if company.TwitterURL == nil || *company.TwitterURL == "" {
		if tw, ok := enriched["twitter_url"].(string); ok && tw != "" {
			updates["twitter_url"] = tw
		}
	}
	if company.LogoURL == nil || *company.LogoURL == "" {
		if logo, ok := enriched["logo_url"].(string); ok && logo != "" {
			updates["logo_url"] = logo
		}
	}

	if err := h.db.WithContext(ctx).Model(&company).Updates(updates).Error; err != nil {
		h.logger.Error("failed to update company with enrichment", "company_id", companyID, "error", err)
		return nil
	}

	if company.Score == nil || *company.Score == 0 {
		score := h.computeCompanyScore(&company, enriched)
		h.db.WithContext(ctx).Model(&company).Update("score", score)
	}

	h.db.WithContext(ctx).Create(&models.Activity{
		OrganizationID: uuidFromString(orgID),
		Type:           "company_enriched",
		Subject:        fmt.Sprintf("Company '%s' enriched", company.Name),
		EntityType:     "company",
		EntityID:       companyID,
	})

	h.logger.Info("company enrichment complete", "company_id", companyID, "fields_updated", len(updates))
	return nil
}

func (h *Handlers) enrichFromDomain(ctx context.Context, company *models.Company) map[string]interface{} {
	result := make(map[string]interface{})

	domain := ""
	if company.Domain != nil && *company.Domain != "" {
		domain = *company.Domain
	} else if company.Website != nil && *company.Website != "" {
		domain = extractDomain(*company.Website)
	}

	if domain == "" {
		result["status"] = "no_domain"
		result["enriched_at"] = time.Now().UTC().Format(time.RFC3339)
		return result
	}

	result["domain"] = domain
	result["enriched_at"] = time.Now().UTC().Format(time.RFC3339)
	result["status"] = "enriched"

	logoURL := fmt.Sprintf("https://logo.clearbit.com/%s", domain)
	result["logo_url"] = logoURL

	socialProfiles := h.discoverSocialProfiles(domain, company.Name)
	for k, v := range socialProfiles {
		result[k] = v
	}

	industryGuess := classifyIndustry(company.Name, company.Tags)
	if industryGuess != "" {
		result["industry_guess"] = industryGuess
	}

	return result
}

func (h *Handlers) discoverSocialProfiles(domain, companyName string) map[string]interface{} {
	profiles := make(map[string]interface{})

	linkedinSlug := domainToLinkedinSlug(domain)
	if linkedinSlug != "" {
		profiles["linkedin_url"] = "https://www.linkedin.com/company/" + linkedinSlug
	}

	twitterHandle := domainToTwitterHandle(domain)
	if twitterHandle != "" {
		profiles["twitter_url"] = "https://twitter.com/" + twitterHandle
	}

	return profiles
}

func (h *Handlers) computeCompanyScore(company *models.Company, enriched map[string]interface{}) int {
	score := 0

	if company.Domain != nil && *company.Domain != "" {
		score += 10
	}
	if company.Website != nil && *company.Website != "" {
		score += 10
	}
	if company.Description != nil && *company.Description != "" {
		score += 10
	}
	if _, ok := enriched["description"]; ok {
		score += 10
	}
	if company.EmployeeCount != nil && *company.EmployeeCount != "" {
		score += 10
	}
	if company.LinkedinURL != nil && *company.LinkedinURL != "" {
		score += 15
	}
	if _, ok := enriched["linkedin_url"]; ok {
		score += 15
	}
	if company.AnnualRevenue != nil && *company.AnnualRevenue != "" {
		score += 15
	}
	if company.Headquarters != nil && *company.Headquarters != "" {
		score += 5
	}
	if len(company.Tags) > 0 {
		score += 5
	}
	if _, ok := enriched["industry_guess"]; ok {
		score += 5
	}

	if score > 100 {
		score = 100
	}
	return score
}

func extractDomain(website string) string {
	website = strings.TrimSpace(website)
	website = strings.TrimPrefix(website, "https://")
	website = strings.TrimPrefix(website, "http://")
	website = strings.TrimPrefix(website, "www.")
	if idx := strings.Index(website, "/"); idx > 0 {
		website = website[:idx]
	}
	return website
}

func domainToLinkedinSlug(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) >= 2 {
		return parts[0]
	}
	return ""
}

func domainToTwitterHandle(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) >= 2 {
		return parts[0]
	}
	return ""
}

func classifyIndustry(companyName string, tags []string) string {
	keywords := map[string][]string{
		"Technology":     {"tech", "software", "ai", "saas", "cloud", "data", "digital", "cyber", "app"},
		"Finance":        {"bank", "finance", "invest", "capital", "fund", "fintech", "insurance"},
		"Healthcare":     {"health", "medical", "pharma", "bio", "clinic", "care"},
		"E-commerce":     {"shop", "store", "commerce", "retail", "market"},
		"Education":      {"edu", "learn", "academy", "school", "university", "course"},
		"Media":          {"media", "news", "publish", "content", "creative", "design"},
		"Real Estate":    {"realty", "property", "estate", "home", "housing"},
		"Food & Beverage": {"food", "restaurant", "beverage", "coffee", "drink"},
		"Travel":         {"travel", "hotel", "tour", "flight", "booking"},
		"Automotive":     {"auto", "car", "motor", "vehicle", "drive"},
	}

	name := strings.ToLower(companyName)
	allText := name
	for _, tag := range tags {
		allText += " " + strings.ToLower(tag)
	}

	for industry, kws := range keywords {
		for _, kw := range kws {
			if strings.Contains(allText, kw) {
				return industry
			}
		}
	}
	return ""
}

func (h *Handlers) HandleEmailSend(ctx context.Context, t *asynq.Task) error {
	payload, err := h.parsePayload(t)
	if err != nil {
		return err
	}

	orgID := payload.OrgID
	h.logger.Info("sending email", "org_id", orgID)

	to, _ := payload.Data["to"].(string)
	subject, _ := payload.Data["subject"].(string)
	body, _ := payload.Data["body"].(string)
	commID, _ := payload.Data["communication_id"].(string)

	if to == "" || subject == "" {
		h.logger.Error("email missing required fields", "to", to, "subject", subject)
		return nil
	}

	h.logger.Info("email delivery",
		"org_id", orgID,
		"to", to,
		"subject", subject,
		"body_length", len(body),
	)

	if commID != "" {
		now := time.Now()
		h.db.WithContext(ctx).
			Model(&models.Communication{}).
			Where("id = ? AND organization_id = ?", commID, orgID).
			Updates(map[string]interface{}{
				"status":  "sent",
				"sent_at": now,
			})
	}

	h.db.WithContext(ctx).Create(&models.Activity{
		OrganizationID: uuidFromString(orgID),
		UserID:         uuidFromString(payload.UserID),
		Type:           "email_sent",
		Subject:        fmt.Sprintf("Email sent to %s: %s", to, subject),
		EntityType:     payload.EntityType,
		EntityID:       payload.EntityID,
	})
	return nil
}

func (h *Handlers) HandleSequenceAdvance(ctx context.Context, t *asynq.Task) error {
	payload, err := h.parsePayload(t)
	if err != nil {
		return err
	}
	h.logger.Info("advancing sequence enrollment",
		"org_id", payload.OrgID,
		"entity_id", payload.EntityID,
	)

	var enrollment models.Enrollment
	if err := h.db.WithContext(ctx).Where("id = ? AND organization_id = ?", payload.EntityID, payload.OrgID).First(&enrollment).Error; err != nil {
		return fmt.Errorf("find enrollment: %w", err)
	}

	if enrollment.Status != "active" {
		h.logger.Info("enrollment not active, skipping", "status", enrollment.Status)
		return nil
	}

	var steps []models.SequenceStep
	h.db.WithContext(ctx).Where("sequence_id = ?", enrollment.SequenceID).Order("position ASC").Find(&steps)

	nextStep := enrollment.CurrentStep + 1
	if nextStep > len(steps) {
		now := time.Now()
		enrollment.Status = "completed"
		enrollment.CompletedAt = &now
		h.db.WithContext(ctx).Save(&enrollment)
		h.logger.Info("enrollment completed", "enrollment_id", enrollment.ID)
		return nil
	}

	enrollment.CurrentStep = nextStep
	h.db.WithContext(ctx).Save(&enrollment)
	return nil
}

func (h *Handlers) HandleWebhookDeliver(ctx context.Context, t *asynq.Task) error {
	payload, err := h.parsePayload(t)
	if err != nil {
		return err
	}

	webhookID := payload.EntityID
	orgID := payload.OrgID

	var webhook models.Webhook
	if err := h.db.WithContext(ctx).Where("id = ? AND organization_id = ?", webhookID, orgID).First(&webhook).Error; err != nil {
		h.logger.Error("webhook not found", "webhook_id", webhookID, "error", err)
		return nil
	}

	if !webhook.IsActive {
		h.logger.Info("webhook inactive, skipping", "webhook_id", webhookID)
		return nil
	}

	eventPayload, _ := json.Marshal(payload.Data)

	delivery := models.WebhookDelivery{
		ID:             uuid.New(),
		OrganizationID: uuidFromString(orgID),
		WebhookID:      webhook.ID,
		Event:          payload.Action,
		Payload:        eventPayload,
		URL:            webhook.URL,
		Status:         "pending",
		MaxAttempts:    5,
	}
	h.db.WithContext(ctx).Create(&delivery)

	if err := h.deliverWebhook(ctx, &webhook, &delivery, eventPayload); err != nil {
		h.logger.Error("webhook delivery failed", "webhook_id", webhookID, "error", err)
		return nil
	}
	return nil
}

func (h *Handlers) deliverWebhook(ctx context.Context, webhook *models.Webhook, delivery *models.WebhookDelivery, body []byte) error {
	const maxAttempts = 5
	const maxFailuresBeforeDisable = 10

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		delivery.Attempts = attempt

		sig := signPayload(body, webhook.Secret)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader(body))
		if err != nil {
			errStr := err.Error()
			delivery.Error = &errStr
			delivery.Status = "failed"
			h.db.WithContext(ctx).Save(delivery)
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-SponsorOS-Signature", sig)
		req.Header.Set("X-SponsorOS-Event", delivery.Event)
		req.Header.Set("X-SponsorOS-Delivery", delivery.ID.String())
		req.Header.Set("User-Agent", "SponsorOS-Webhook/1.0")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			errStr := err.Error()
			delivery.Error = &errStr

			if attempt < maxAttempts {
				retryAt := time.Now().Add(retryDelay(attempt))
				delivery.NextRetryAt = &retryAt
				h.db.WithContext(ctx).Save(delivery)
				h.logger.Warn("webhook delivery attempt failed, retrying",
					"webhook_id", webhook.ID, "attempt", attempt, "retry_at", retryAt)
				time.Sleep(retryDelay(attempt))
				continue
			}

			delivery.Status = "failed"
			h.db.WithContext(ctx).Save(delivery)
			h.incrementFailureCount(ctx, webhook, maxFailuresBeforeDisable)
			return nil
		}

		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()

		code := resp.StatusCode
		respStr := string(respBody)
		delivery.ResponseCode = &code
		delivery.ResponseBody = &respStr

		if code >= 200 && code < 300 {
			now := time.Now()
			delivery.Status = "delivered"
			delivery.DeliveredAt = &now
			delivery.Error = nil
			delivery.NextRetryAt = nil
			h.db.WithContext(ctx).Save(delivery)

			webhook.LastTriggered = &now
			webhook.FailureCount = 0
			h.db.WithContext(ctx).Model(webhook).Updates(map[string]interface{}{
				"last_triggered": now,
				"failure_count":  0,
			})
			return nil
		}

		errStr := fmt.Sprintf("HTTP %d: %s", code, respStr)
		delivery.Error = &errStr

		if attempt < maxAttempts {
			retryAt := time.Now().Add(retryDelay(attempt))
			delivery.NextRetryAt = &retryAt
			h.db.WithContext(ctx).Save(delivery)
			h.logger.Warn("webhook returned non-2xx, retrying",
				"webhook_id", webhook.ID, "status", code, "attempt", attempt)
			time.Sleep(retryDelay(attempt))
			continue
		}

		delivery.Status = "failed"
		h.db.WithContext(ctx).Save(delivery)
		h.incrementFailureCount(ctx, webhook, maxFailuresBeforeDisable)
		return nil
	}
	return nil
}

func (h *Handlers) incrementFailureCount(ctx context.Context, webhook *models.Webhook, threshold int) {
	webhook.FailureCount++
	updates := map[string]interface{}{"failure_count": webhook.FailureCount}
	if webhook.FailureCount >= threshold {
		updates["is_active"] = false
		h.logger.Warn("webhook auto-disabled after too many failures",
			"webhook_id", webhook.ID, "failure_count", webhook.FailureCount)
	}
	h.db.WithContext(ctx).Model(webhook).Updates(updates)
}

func signPayload(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func retryDelay(attempt int) time.Duration {
	return time.Duration(math.Pow(2, float64(attempt))) * time.Second
}

func (h *Handlers) HandleAnalyticsCompute(ctx context.Context, t *asynq.Task) error {
	payload, err := h.parsePayload(t)
	if err != nil {
		return err
	}

	orgID := payload.OrgID
	h.logger.Info("computing analytics", "org_id", orgID)

	orgUUID := uuidFromString(orgID)

	var sponsorCount int64
	h.db.WithContext(ctx).Model(&models.Sponsor{}).
		Where("organization_id = ?", orgUUID).Count(&sponsorCount)

	var activeSponsors int64
	h.db.WithContext(ctx).Model(&models.Sponsor{}).
		Where("organization_id = ? AND stage NOT IN ?", orgUUID, []string{"lost", "churned"}).
		Count(&activeSponsors)

	var companyCount int64
	h.db.WithContext(ctx).Model(&models.Company{}).
		Where("organization_id = ? AND status = ?", orgUUID, "active").
		Count(&companyCount)

	type sumResult struct {
		Total float64
	}
	var totalDealValue sumResult
	h.db.WithContext(ctx).Model(&models.Sponsor{}).
		Where("organization_id = ? AND stage IN ?", orgUUID, []string{"won", "closed_won", "active"}).
		Select("COALESCE(SUM(deal_value), 0) as total").
		Scan(&totalDealValue)

	var commsSent int64
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	h.db.WithContext(ctx).Model(&models.Communication{}).
		Where("organization_id = ? AND sent_at >= ?", orgUUID, thirtyDaysAgo).
		Count(&commsSent)

	snapshot := map[string]interface{}{
		"computed_at":      time.Now().UTC().Format(time.RFC3339),
		"total_sponsors":   sponsorCount,
		"active_sponsors":  activeSponsors,
		"total_companies":  companyCount,
		"total_deal_value": totalDealValue.Total,
		"comms_sent_30d":   commsSent,
	}

	snapshotJSON, _ := json.Marshal(snapshot)
	h.logger.Info("analytics computed",
		"org_id", orgID,
		"snapshot", string(snapshotJSON),
	)

	return nil
}

func (h *Handlers) HandleMemoryIndex(ctx context.Context, t *asynq.Task) error {
	payload, err := h.parsePayload(t)
	if err != nil {
		return err
	}

	orgID := payload.OrgID
	entityType := payload.EntityType
	entityID := payload.EntityID

	h.logger.Info("indexing memory",
		"org_id", orgID,
		"entity_type", entityType,
		"entity_id", entityID,
	)

	content := ""

	switch entityType {
	case "company":
		var company models.Company
		if err := h.db.WithContext(ctx).Where("id = ? AND organization_id = ?", entityID, orgID).First(&company).Error; err != nil {
			h.logger.Error("company not found for memory indexing", "entity_id", entityID)
			return nil
		}
		content = buildCompanyMemoryContent(&company)

	case "sponsor":
		var sponsor models.Sponsor
		if err := h.db.WithContext(ctx).Where("id = ? AND organization_id = ?", entityID, orgID).
			Preload("Company").First(&sponsor).Error; err != nil {
			h.logger.Error("sponsor not found for memory indexing", "entity_id", entityID)
			return nil
		}
		content = buildSponsorMemoryContent(&sponsor)

	case "contact":
		var contact models.Contact
		if err := h.db.WithContext(ctx).Where("id = ? AND organization_id = ?", entityID, orgID).First(&contact).Error; err != nil {
			h.logger.Error("contact not found for memory indexing", "entity_id", entityID)
			return nil
		}
		content = buildContactMemoryContent(&contact)

	default:
		h.logger.Warn("unknown entity type for memory indexing", "entity_type", entityType)
		return nil
	}

	if content == "" {
		return nil
	}

	entityUUID := uuidFromString(entityID)
	orgUUID := uuidFromString(orgID)

	var existing models.AIMemory
	result := h.db.WithContext(ctx).Where(
		"organization_id = ? AND entity_type = ? AND entity_id = ?",
		orgUUID, entityType, entityUUID,
	).First(&existing)

	if result.Error == nil {
		h.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
			"content":    content,
			"importance": 0.5,
		})
		h.logger.Info("updated existing memory", "memory_id", existing.ID)
	} else {
		memory := models.AIMemory{
			OrganizationID: orgUUID,
			MemoryType:     "entity",
			Content:        content,
			Importance:     0.5,
			EntityType:     &entityType,
			EntityID:       &entityUUID,
		}
		h.db.WithContext(ctx).Create(&memory)
		h.logger.Info("created new memory entry", "entity_type", entityType, "entity_id", entityID)
	}

	node := models.KnowledgeNode{
		OrganizationID: orgUUID,
		NodeType:       entityType,
		EntityID:       &entityUUID,
		Label:          content[:min(len(content), 500)],
	}

	var existingNode models.KnowledgeNode
	nodeResult := h.db.WithContext(ctx).Where(
		"organization_id = ? AND node_type = ? AND entity_id = ?",
		orgUUID, entityType, entityUUID,
	).First(&existingNode)

	if nodeResult.Error == nil {
		h.db.WithContext(ctx).Model(&existingNode).Update("label", node.Label)
	} else {
		h.db.WithContext(ctx).Create(&node)
	}

	return nil
}

func buildCompanyMemoryContent(c *models.Company) string {
	parts := []string{fmt.Sprintf("Company: %s", c.Name)}
	if c.Domain != nil && *c.Domain != "" {
		parts = append(parts, fmt.Sprintf("Domain: %s", *c.Domain))
	}
	if c.Description != nil && *c.Description != "" {
		parts = append(parts, fmt.Sprintf("Description: %s", *c.Description))
	}
	if c.EmployeeCount != nil && *c.EmployeeCount != "" {
		parts = append(parts, fmt.Sprintf("Employees: %s", *c.EmployeeCount))
	}
	if c.AnnualRevenue != nil && *c.AnnualRevenue != "" {
		parts = append(parts, fmt.Sprintf("Revenue: %s", *c.AnnualRevenue))
	}
	if c.Headquarters != nil && *c.Headquarters != "" {
		parts = append(parts, fmt.Sprintf("HQ: %s", *c.Headquarters))
	}
	if len(c.Tags) > 0 {
		parts = append(parts, fmt.Sprintf("Tags: %s", strings.Join(c.Tags, ", ")))
	}
	return strings.Join(parts, ". ")
}

func buildSponsorMemoryContent(s *models.Sponsor) string {
	companyName := s.Company.Name
	if companyName == "" {
		companyName = "Unknown Company"
	}
	parts := []string{fmt.Sprintf("Sponsor: %s", companyName)}
	tier := "unset"
	if s.Tier != nil {
		tier = *s.Tier
	}
	parts = append(parts, fmt.Sprintf("Stage: %s, Tier: %s", s.Stage, tier))
	if s.DealValue != nil && *s.DealValue > 0 {
		parts = append(parts, fmt.Sprintf("Deal Value: $%.0f", *s.DealValue))
	}
	if s.Notes != nil && *s.Notes != "" {
		parts = append(parts, fmt.Sprintf("Notes: %s", *s.Notes))
	}
	return strings.Join(parts, ". ")
}

func buildContactMemoryContent(c *models.Contact) string {
	parts := []string{fmt.Sprintf("Contact: %s %s", c.FirstName, c.LastName)}
	if c.Email != nil && *c.Email != "" {
		parts = append(parts, fmt.Sprintf("Email: %s", *c.Email))
	}
	if c.Title != nil && *c.Title != "" {
		parts = append(parts, fmt.Sprintf("Title: %s", *c.Title))
	}
	if c.Department != nil && *c.Department != "" {
		parts = append(parts, fmt.Sprintf("Department: %s", *c.Department))
	}
	return strings.Join(parts, ". ")
}

func RegisterHandlers(mux *asynq.ServeMux, h *Handlers) {
	mux.HandleFunc(TaskAIResearch, h.HandleAIResearch)
	mux.HandleFunc(TaskAIQualification, h.HandleAIQualification)
	mux.HandleFunc(TaskAIOutreach, h.HandleAIOutreach)
	mux.HandleFunc(TaskCompanyEnrich, h.HandleCompanyEnrich)
	mux.HandleFunc(TaskEmailSend, h.HandleEmailSend)
	mux.HandleFunc(TaskSequenceAdvance, h.HandleSequenceAdvance)
	mux.HandleFunc(TaskWebhookDeliver, h.HandleWebhookDeliver)
	mux.HandleFunc(TaskAnalyticsCompute, h.HandleAnalyticsCompute)
	mux.HandleFunc(TaskMemoryIndex, h.HandleMemoryIndex)
}
