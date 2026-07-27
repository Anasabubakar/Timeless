package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/timeless/backend/internal/ai/agent"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/repository"
)

type PlannedAutomation struct {
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	TriggerType   string                 `json:"trigger_type"`
	TriggerConfig map[string]interface{} `json:"trigger_config"`
	ActionType    string                 `json:"action_type"`
}

// AutomationPlanService turns a chosen goal into a proposed automation plan,
// and — once approved — persists it as real, schedulable Automation records.
type AutomationPlanService struct {
	orchestrator   *agent.Orchestrator
	automationRepo *repository.AutomationRepository
}

func NewAutomationPlanService(orchestrator *agent.Orchestrator, automationRepo *repository.AutomationRepository) *AutomationPlanService {
	return &AutomationPlanService{orchestrator: orchestrator, automationRepo: automationRepo}
}

func (s *AutomationPlanService) Plan(ctx context.Context, orgID, userID uuid.UUID, goal string) ([]PlannedAutomation, error) {
	output, err := s.orchestrator.Execute(ctx, agent.AgentAutomationPlanning, &agent.Input{
		Query:  "Goal: " + goal,
		OrgID:  orgID.String(),
		UserID: userID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("automation planning: %w", err)
	}

	var steps []PlannedAutomation
	if err := json.Unmarshal([]byte(output.Response), &steps); err != nil {
		return nil, fmt.Errorf("parse automation plan response: %w", err)
	}
	return steps, nil
}

// Approve persists each planned step as a real Automation record, active
// immediately — nothing here is simulated.
func (s *AutomationPlanService) Approve(ctx context.Context, orgID, userID uuid.UUID, steps []PlannedAutomation) ([]models.Automation, error) {
	created := make([]models.Automation, 0, len(steps))
	for _, step := range steps {
		triggerConfig, err := json.Marshal(step.TriggerConfig)
		if err != nil {
			return nil, err
		}
		actions, err := json.Marshal([]map[string]interface{}{
			{"type": step.ActionType},
		})
		if err != nil {
			return nil, err
		}

		automation := models.Automation{
			OrganizationID: orgID,
			Name:           step.Title,
			Description:    &step.Description,
			TriggerType:    step.TriggerType,
			TriggerConfig:  datatypes.JSON(triggerConfig),
			Actions:        datatypes.JSON(actions),
			IsActive:       true,
			CreatedBy:      &userID,
		}
		if err := s.automationRepo.Create(ctx, &automation); err != nil {
			return nil, err
		}
		created = append(created, automation)
	}
	return created, nil
}
