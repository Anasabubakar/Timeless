package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/timeless/backend/internal/ai/agent"
)

type AIHandler struct {
	orchestrator *agent.Orchestrator
}

func NewAIHandler(orch *agent.Orchestrator) *AIHandler {
	return &AIHandler{orchestrator: orch}
}

type AIQueryRequest struct {
	Query      string `json:"query" validate:"required"`
	AgentType  string `json:"agent_type,omitempty"`
	CampaignID string `json:"campaign_id,omitempty"`
	SponsorID  string `json:"sponsor_id,omitempty"`
	CompanyID  string `json:"company_id,omitempty"`
}

func (h *AIHandler) Query(c fiber.Ctx) error {
	var req AIQueryRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	orgID := c.Locals("org_id").(string)
	userID := c.Locals("user_id").(string)

	input := &agent.Input{
		Query:      req.Query,
		OrgID:      orgID,
		UserID:     userID,
		CampaignID: req.CampaignID,
		SponsorID:  req.SponsorID,
		CompanyID:  req.CompanyID,
	}

	var agentType agent.AgentType
	if req.AgentType != "" {
		agentType = agent.AgentType(req.AgentType)
	} else {
		routed, err := h.orchestrator.Route(c.Context(), input)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "routing failed"})
		}
		agentType = routed
	}

	output, err := h.orchestrator.Execute(c.Context(), agentType, input)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"agent":      string(agentType),
		"response":   output.Response,
		"data":       output.Data,
		"actions":    output.Actions,
		"tokens_used": output.TokensUsed,
	})
}

func (h *AIHandler) ListAgents(c fiber.Ctx) error {
	agents := []fiber.Map{
		{"type": "research", "name": "Research Agent", "description": "Finds potential sponsors and market opportunities"},
		{"type": "qualification", "name": "Qualification Agent", "description": "Scores and evaluates sponsor fit"},
		{"type": "company_intel", "name": "Company Intel Agent", "description": "Enriches company data and signals"},
		{"type": "decision_maker", "name": "Decision Maker Agent", "description": "Identifies key contacts and stakeholders"},
		{"type": "duplicate", "name": "Duplicate Agent", "description": "Detects and merges duplicate records"},
		{"type": "crm", "name": "CRM Agent", "description": "Manages pipeline and relationship data"},
		{"type": "outreach", "name": "Outreach Agent", "description": "Drafts personalized communications"},
		{"type": "meeting", "name": "Meeting Agent", "description": "Prepares briefs and summarizes meetings"},
		{"type": "analytics", "name": "Analytics Agent", "description": "Generates insights and forecasts"},
		{"type": "memory", "name": "Memory Agent", "description": "Maintains organizational knowledge"},
		{"type": "strategy", "name": "Strategy Agent", "description": "Advises on pricing and negotiation"},
	}
	return c.JSON(fiber.Map{"agents": agents})
}
