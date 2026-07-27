package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/timeless/backend/internal/ai/agent"
)

type RecommendedGoal struct {
	Title             string `json:"title"`
	Description       string `json:"description"`
	AutomationSummary string `json:"automation_summary"`
}

// GoalService recommends goals based on the projects/workspace context the
// user selected during discovery.
type GoalService struct {
	orchestrator *agent.Orchestrator
}

func NewGoalService(orchestrator *agent.Orchestrator) *GoalService {
	return &GoalService{orchestrator: orchestrator}
}

func (s *GoalService) Recommend(ctx context.Context, orgID, userID uuid.UUID, projectNames []string) ([]RecommendedGoal, error) {
	contextSummary := "General workspace, no specific projects selected yet."
	if len(projectNames) > 0 {
		contextSummary = "Selected projects/workspaces: " + strings.Join(projectNames, ", ")
	}

	output, err := s.orchestrator.Execute(ctx, agent.AgentGoalRecommendation, &agent.Input{
		Query:  contextSummary,
		OrgID:  orgID.String(),
		UserID: userID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("goal recommendation: %w", err)
	}

	var goals []RecommendedGoal
	if err := json.Unmarshal([]byte(output.Response), &goals); err != nil {
		return nil, fmt.Errorf("parse goal recommendation response: %w", err)
	}
	return goals, nil
}
