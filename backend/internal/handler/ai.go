package handler

import (
	"github.com/gofiber/fiber/v3"

	"github.com/timeless/backend/internal/ai/agent"
	"github.com/timeless/backend/internal/logging"
	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/pkg/reqbind"
)

type AIHandler struct {
	orchestrator *agent.Orchestrator
}

func NewAIHandler(orch *agent.Orchestrator) *AIHandler {
	return &AIHandler{orchestrator: orch}
}

// AIQueryRequest.Query is capped at 8000 characters — generous for any
// legitimate research/qualification question, but bounded so a single
// request can't balloon token cost or use the query field itself as an
// oversized injection surface. CampaignID/SponsorID/CompanyID are
// validated as UUIDs since they're used to scope which org records the
// agent can reference.
type AIQueryRequest struct {
	Query      string `json:"query" validate:"required,max=8000"`
	AgentType  string `json:"agent_type,omitempty"`
	CampaignID string `json:"campaign_id,omitempty" validate:"omitempty,uuid"`
	SponsorID  string `json:"sponsor_id,omitempty" validate:"omitempty,uuid"`
	CompanyID  string `json:"company_id,omitempty" validate:"omitempty,uuid"`
}

func (h *AIHandler) Query(c fiber.Ctx) error {
	var req AIQueryRequest
	if verr := reqbind.JSON(c, &req); verr != nil {
		return verr
	}

	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	input := &agent.Input{
		Query:      req.Query,
		OrgID:      orgID.String(),
		UserID:     userID.String(),
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
			logging.Printf("ai: routing failed for org %s: %v", orgID, err)
			return fiber.NewError(fiber.StatusInternalServerError, "routing failed")
		}
		agentType = routed
	}

	output, err := h.orchestrator.Execute(c.Context(), agentType, input)
	if err != nil {
		logging.Printf("ai: query failed for org %s agent %s: %v", orgID, agentType, err)
		return fiber.NewError(fiber.StatusInternalServerError, "AI query failed")
	}

	return c.JSON(fiber.Map{
		"agent":       string(agentType),
		"response":    output.Response,
		"data":        output.Data,
		"actions":     output.Actions,
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
