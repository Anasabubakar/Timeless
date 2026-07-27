package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/ai/agent"
	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/repository"
)

type DiscoveredProject struct {
	Name           string   `json:"name"`
	Confidence     int      `json:"confidence"`
	Explanation    string   `json:"explanation"`
	Sources        []string `json:"sources"`
	DocumentCount  int      `json:"document_count"`
	RecentActivity string   `json:"recent_activity"`
}

// DiscoveryService infers what the user is likely working on from what's
// connected so far, and turns their selection into real Project records.
type DiscoveryService struct {
	integrationRepo *repository.IntegrationRepository
	projectRepo     *repository.ProjectRepository
	orchestrator    *agent.Orchestrator
}

func NewDiscoveryService(integrationRepo *repository.IntegrationRepository, projectRepo *repository.ProjectRepository, orchestrator *agent.Orchestrator) *DiscoveryService {
	return &DiscoveryService{integrationRepo: integrationRepo, projectRepo: projectRepo, orchestrator: orchestrator}
}

// Run summarizes what's connected and asks the WorkspaceDiscoveryAgent to
// infer likely projects. Returns an empty list (not an error) if nothing is
// connected yet — there's simply no signal to work from.
func (s *DiscoveryService) Run(ctx context.Context, orgID, userID uuid.UUID) ([]DiscoveredProject, error) {
	integrations, err := s.integrationRepo.List(ctx, orgID)
	if err != nil {
		return nil, err
	}

	var summary strings.Builder
	connectedCount := 0
	for _, in := range integrations {
		if in.Status != "active" && in.Status != "syncing" {
			continue
		}
		connectedCount++
		fmt.Fprintf(&summary, "- %s (status: %s)", in.Provider, in.Status)
		if len(in.Config) > 0 && string(in.Config) != "{}" {
			fmt.Fprintf(&summary, ", sync summary: %s", string(in.Config))
		}
		summary.WriteString("\n")
	}

	if connectedCount == 0 {
		return []DiscoveredProject{}, nil
	}

	output, err := s.orchestrator.Execute(ctx, agent.AgentWorkspaceDiscovery, &agent.Input{
		Query:  "Connected integrations and what's available in each:\n" + summary.String(),
		OrgID:  orgID.String(),
		UserID: userID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("workspace discovery: %w", err)
	}

	var projects []DiscoveredProject
	if err := json.Unmarshal([]byte(output.Response), &projects); err != nil {
		return nil, fmt.Errorf("parse discovery response: %w", err)
	}
	return projects, nil
}

type SelectProjectsInput struct {
	ProjectNames []string `json:"project_names"`
	NewProject   string   `json:"new_project"`
}

// Select persists the projects the user chose (or created) as real
// records, creating any that don't already exist for this org.
func (s *DiscoveryService) Select(ctx context.Context, orgID, userID uuid.UUID, input SelectProjectsInput) ([]models.Project, error) {
	names := append([]string{}, input.ProjectNames...)
	if strings.TrimSpace(input.NewProject) != "" {
		names = append(names, strings.TrimSpace(input.NewProject))
	}

	var selected []models.Project
	for _, name := range names {
		existing, err := s.projectRepo.GetByName(ctx, orgID, name)
		if err == nil {
			selected = append(selected, *existing)
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}

		project := models.Project{
			OrganizationID: orgID,
			Name:           name,
			Status:         "active",
			CreatedBy:      &userID,
		}
		if err := s.projectRepo.Create(ctx, &project); err != nil {
			return nil, err
		}
		selected = append(selected, project)
	}

	return selected, nil
}
