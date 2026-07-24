package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/timeless/backend/internal/ai/agent"
	"github.com/timeless/backend/internal/middleware"
)

type AgentLearningHandler struct {
	learning *agent.LearningService
}

func NewAgentLearningHandler(learning *agent.LearningService) *AgentLearningHandler {
	return &AgentLearningHandler{learning: learning}
}

type RecordOutcomeRequest struct {
	AgentType      string  `json:"agent_type" validate:"required"`
	ConversationID *string `json:"conversation_id"`
	Query          string  `json:"query" validate:"required"`
	Response       string  `json:"response"`
	Outcome        string  `json:"outcome" validate:"required"`
	Score          float64 `json:"score"`
	Feedback       *string `json:"feedback"`
}

func (h *AgentLearningHandler) RecordOutcome(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)

	var req RecordOutcomeRequest
	if err := c.Bind().JSON(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
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
	Feedback   string `json:"feedback" validate:"required"`
	IsPositive bool   `json:"is_positive"`
}

func (h *AgentLearningHandler) SubmitFeedback(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)

	var req FeedbackRequest
	if err := c.Bind().JSON(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
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
	Preference string  `json:"preference" validate:"required"`
	Confidence float64 `json:"confidence"`
}

func (h *AgentLearningHandler) StorePreference(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)

	var req StorePreferenceRequest
	if err := c.Bind().JSON(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
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
