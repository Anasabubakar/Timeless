package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/timeless/backend/internal/ai/agent"
	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/pkg/reqbind"
)

type AgentLearningHandler struct {
	learning *agent.LearningService
}

func NewAgentLearningHandler(learning *agent.LearningService) *AgentLearningHandler {
	return &AgentLearningHandler{learning: learning}
}

// Every free-text field below (Query, Response, Feedback, Preference)
// eventually feeds into buildSystemPrompt's learned_context block for
// future AI calls — the max=4000 validation tags keep a single
// submission from being disproportionately large relative to
// everything else in the prompt, on top of the delimiting fix in
// agents.go. Not a substitute for that fix, just one more knob that
// makes an injection attempt smaller and more conspicuous.

type RecordOutcomeRequest struct {
	AgentType      string  `json:"agent_type" validate:"required"`
	ConversationID *string `json:"conversation_id,omitempty" validate:"omitempty,uuid"`
	Query          string  `json:"query" validate:"required,max=4000"`
	Response       string  `json:"response" validate:"max=4000"`
	Outcome        string  `json:"outcome" validate:"required,oneof=success failure neutral positive_feedback negative_feedback"`
	Score          float64 `json:"score" validate:"gte=0,lte=1"`
	Feedback       *string `json:"feedback,omitempty" validate:"omitempty,max=4000"`
}

func (h *AgentLearningHandler) RecordOutcome(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)

	var req RecordOutcomeRequest
	if verr := reqbind.JSON(c, &req); verr != nil {
		return verr
	}

	outcome := &agent.AgentOutcome{
		OrganizationID: orgID,
		AgentType:      req.AgentType,
		Query:          req.Query,
		Response:       req.Response,
		Outcome:        agent.OutcomeType(req.Outcome),
		Score:          req.Score,
		Feedback:       req.Feedback,
	}

	if req.ConversationID != nil {
		id, err := uuid.Parse(*req.ConversationID)
		if err == nil {
			outcome.ConversationID = &id
		}
	}

	if err := h.learning.RecordOutcome(c.Context(), outcome); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to record outcome")
	}

	return c.Status(fiber.StatusCreated).JSON(outcome)
}

type FeedbackRequest struct {
	AgentType  string `json:"agent_type" validate:"required"`
	Feedback   string `json:"feedback" validate:"required,max=4000"`
	IsPositive bool   `json:"is_positive"`
}

func (h *AgentLearningHandler) SubmitFeedback(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)

	var req FeedbackRequest
	if verr := reqbind.JSON(c, &req); verr != nil {
		return verr
	}

	if err := h.learning.LearnFromFeedback(
		c.Context(), orgID, agent.AgentType(req.AgentType), req.Feedback, req.IsPositive,
	); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to record feedback")
	}

	return c.JSON(fiber.Map{"message": "Feedback recorded"})
}

type StorePreferenceRequest struct {
	AgentType  string  `json:"agent_type" validate:"required"`
	Category   string  `json:"category" validate:"required"`
	Preference string  `json:"preference" validate:"required,max=4000"`
	Confidence float64 `json:"confidence" validate:"gte=0,lte=1"`
}

func (h *AgentLearningHandler) StorePreference(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)

	var req StorePreferenceRequest
	if verr := reqbind.JSON(c, &req); verr != nil {
		return verr
	}

	confidence := req.Confidence
	if confidence == 0 {
		confidence = 0.5
	}

	pref := &agent.LearnedPreference{
		OrganizationID: orgID,
		AgentType:      req.AgentType,
		Category:       req.Category,
		Preference:     req.Preference,
		Confidence:     confidence,
	}

	if err := h.learning.StorePreference(c.Context(), pref); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to store preference")
	}

	return c.Status(fiber.StatusCreated).JSON(pref)
}

func (h *AgentLearningHandler) GetPreferences(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	agentType := c.Query("agent_type")

	if agentType == "" {
		return fiber.NewError(fiber.StatusBadRequest, "agent_type query parameter required")
	}

	prefs, err := h.learning.GetPreferences(c.Context(), orgID, agent.AgentType(agentType))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get preferences")
	}

	return c.JSON(fiber.Map{"data": prefs})
}

func (h *AgentLearningHandler) GetOutcomes(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	agentType := c.Query("agent_type")

	if agentType == "" {
		return fiber.NewError(fiber.StatusBadRequest, "agent_type query parameter required")
	}

	outcomes, err := h.learning.GetRecentOutcomes(c.Context(), orgID, agent.AgentType(agentType), 20)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get outcomes")
	}

	return c.JSON(fiber.Map{"data": outcomes})
}
