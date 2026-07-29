package handler

import (
	"github.com/timeless/backend/internal/logging"

	"github.com/gofiber/fiber/v3"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/service"
)

type DiscoveryHandler struct {
	discovery *service.DiscoveryService
	goals     *service.GoalService
	plans     *service.AutomationPlanService
}

func NewDiscoveryHandler(discovery *service.DiscoveryService, goals *service.GoalService, plans *service.AutomationPlanService) *DiscoveryHandler {
	return &DiscoveryHandler{discovery: discovery, goals: goals, plans: plans}
}

func (h *DiscoveryHandler) RunDiscovery(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	projects, err := h.discovery.Run(c.Context(), orgID, userID)
	if err != nil {
		logging.Printf("discovery: run failed for org %s: %v", orgID, err)
		return fiber.NewError(fiber.StatusInternalServerError, "discovery run failed")
	}
	return c.JSON(fiber.Map{"data": projects})
}

func (h *DiscoveryHandler) SelectProjects(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	var input service.SelectProjectsInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	projects, err := h.discovery.Select(c.Context(), orgID, userID, input)
	if err != nil {
		logging.Printf("discovery: select failed for org %s: %v", orgID, err)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to save project selection")
	}
	return c.JSON(fiber.Map{"data": projects})
}

type recommendGoalsRequest struct {
	ProjectNames []string `json:"project_names"`
}

func (h *DiscoveryHandler) RecommendGoals(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	var req recommendGoalsRequest
	if err := c.Bind().JSON(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	goals, err := h.goals.Recommend(c.Context(), orgID, userID, req.ProjectNames)
	if err != nil {
		logging.Printf("discovery: goal recommendation failed for org %s: %v", orgID, err)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate goal recommendations")
	}
	return c.JSON(fiber.Map{"data": goals})
}

type planAutomationRequest struct {
	Goal string `json:"goal"`
}

func (h *DiscoveryHandler) PlanAutomation(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	var req planAutomationRequest
	if err := c.Bind().JSON(&req); err != nil || req.Goal == "" {
		return fiber.NewError(fiber.StatusBadRequest, "goal is required")
	}

	steps, err := h.plans.Plan(c.Context(), orgID, userID, req.Goal)
	if err != nil {
		logging.Printf("discovery: automation planning failed for org %s: %v", orgID, err)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to plan automation")
	}
	return c.JSON(fiber.Map{"data": steps})
}

type approveAutomationRequest struct {
	Steps []service.PlannedAutomation `json:"steps"`
}

func (h *DiscoveryHandler) ApproveAutomation(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	userID := middleware.GetUserID(c)

	var req approveAutomationRequest
	if err := c.Bind().JSON(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	automations, err := h.plans.Approve(c.Context(), orgID, userID, req.Steps)
	if err != nil {
		logging.Printf("discovery: automation approval failed for org %s: %v", orgID, err)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to approve automation")
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": automations})
}
