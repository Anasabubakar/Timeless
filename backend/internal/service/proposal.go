package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/sponsoros/backend/internal/ai/provider"
	"github.com/sponsoros/backend/internal/models"
	"github.com/sponsoros/backend/internal/repository"
)

type ProposalService struct {
	repo        *repository.ProposalRepository
	sponsorRepo *repository.SponsorRepository
	companyRepo *repository.CompanyRepository
	aiProvider  provider.Provider
}

func NewProposalService(
	repo *repository.ProposalRepository,
	sponsorRepo *repository.SponsorRepository,
	companyRepo *repository.CompanyRepository,
	aiProvider provider.Provider,
) *ProposalService {
	return &ProposalService{
		repo:        repo,
		sponsorRepo: sponsorRepo,
		companyRepo: companyRepo,
		aiProvider:  aiProvider,
	}
}

func (s *ProposalService) List(ctx context.Context, orgID uuid.UUID, sponsorID *uuid.UUID, status string) ([]models.Proposal, int64, error) {
	return s.repo.List(ctx, orgID, sponsorID, status)
}

func (s *ProposalService) GetByID(ctx context.Context, orgID, id uuid.UUID) (*models.Proposal, error) {
	return s.repo.GetByID(ctx, orgID, id)
}

func (s *ProposalService) Create(ctx context.Context, proposal *models.Proposal) error {
	return s.repo.Create(ctx, proposal)
}

func (s *ProposalService) Update(ctx context.Context, proposal *models.Proposal) error {
	return s.repo.Update(ctx, proposal)
}

func (s *ProposalService) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	return s.repo.Delete(ctx, orgID, id)
}

type GenerateProposalInput struct {
	SponsorID   uuid.UUID `json:"sponsor_id"`
	Tone        string    `json:"tone"`
	PackageTier string    `json:"package_tier"`
	CustomNotes string    `json:"custom_notes"`
}

func (s *ProposalService) Generate(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, input *GenerateProposalInput) (*models.Proposal, error) {
	if s.aiProvider == nil {
		return nil, fmt.Errorf("AI provider not configured")
	}

	sponsor, err := s.sponsorRepo.GetByID(ctx, orgID, input.SponsorID)
	if err != nil {
		return nil, fmt.Errorf("sponsor not found: %w", err)
	}

	company, err := s.companyRepo.GetByID(ctx, orgID, sponsor.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("company not found: %w", err)
	}

	prompt := buildProposalPrompt(company, sponsor, input)

	resp, err := s.aiProvider.Complete(ctx, &provider.CompletionRequest{
		Model: "gpt-4o",
		Messages: []provider.Message{
			{Role: provider.RoleSystem, Content: proposalSystemPrompt},
			{Role: provider.RoleUser, Content: prompt},
		},
		MaxTokens:   4096,
		Temperature: 0.6,
	})
	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	title := fmt.Sprintf("Sponsorship Proposal - %s", company.Name)
	content := resp.Content
	proposal := &models.Proposal{
		OrganizationID: orgID,
		SponsorID:      input.SponsorID,
		Title:          title,
		Content:        &content,
		Status:         "draft",
		Version:        1,
		Amount:         sponsor.DealValue,
		CreatedBy:      &userID,
	}

	if err := s.repo.Create(ctx, proposal); err != nil {
		return nil, fmt.Errorf("failed to save proposal: %w", err)
	}

	return proposal, nil
}

const proposalSystemPrompt = `You are a sponsorship proposal writer for events and conferences.
Write professional, persuasive sponsorship proposals that clearly articulate value propositions.

Structure your proposals with:
1. A warm, personalized opening referencing the company
2. Event/conference overview and audience alignment
3. Sponsorship package details and benefits
4. ROI projections and visibility metrics
5. Clear next steps and timeline

Keep the tone professional but warm. Use concrete numbers and benefits.
Format in clean Markdown.`

func buildProposalPrompt(company *models.Company, sponsor *models.Sponsor, input *GenerateProposalInput) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Generate a sponsorship proposal for %s.\n\n", company.Name))
	b.WriteString("## Company Details\n")
	b.WriteString(fmt.Sprintf("- Company: %s\n", company.Name))
	if company.Domain != nil && *company.Domain != "" {
		b.WriteString(fmt.Sprintf("- Domain: %s\n", *company.Domain))
	}
	if company.Description != nil && *company.Description != "" {
		b.WriteString(fmt.Sprintf("- Description: %s\n", *company.Description))
	}
	if company.EmployeeCount != nil && *company.EmployeeCount != "" {
		b.WriteString(fmt.Sprintf("- Size: %s employees\n", *company.EmployeeCount))
	}
	if company.Industry != nil {
		b.WriteString(fmt.Sprintf("- Industry: %s\n", company.Industry.Name))
	}

	b.WriteString("\n## Sponsorship Context\n")
	b.WriteString(fmt.Sprintf("- Current Stage: %s\n", sponsor.Stage))
	if sponsor.DealValue != nil {
		b.WriteString(fmt.Sprintf("- Target Deal Value: $%.0f\n", *sponsor.DealValue))
	}
	if sponsor.Tier != nil {
		b.WriteString(fmt.Sprintf("- Tier: %s\n", *sponsor.Tier))
	}

	if input.PackageTier != "" {
		b.WriteString(fmt.Sprintf("- Requested Package Tier: %s\n", input.PackageTier))
	}
	if input.Tone != "" {
		b.WriteString(fmt.Sprintf("\n## Tone: %s\n", input.Tone))
	}
	if input.CustomNotes != "" {
		b.WriteString(fmt.Sprintf("\n## Additional Context\n%s\n", input.CustomNotes))
	}

	return b.String()
}
