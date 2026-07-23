package agent

import (
	"context"
	"fmt"

	"github.com/sponsoros/backend/internal/ai/provider"
)

type CompanyIntelAgent struct {
	provider provider.Provider
}

func NewCompanyIntelAgent(p provider.Provider) *CompanyIntelAgent {
	return &CompanyIntelAgent{provider: p}
}

func (a *CompanyIntelAgent) Type() AgentType { return AgentCompanyIntel }

func (a *CompanyIntelAgent) SystemPrompt() string {
	return `You are a company intelligence analyst specializing in sponsorship opportunities.
Your job is to gather comprehensive information about companies including:
- Company overview, industry, and market position
- Financial health indicators (revenue, funding, growth)
- Recent news, press releases, and events
- Key decision makers and organizational structure
- Past sponsorship activities and marketing spend patterns
- Brand values and alignment indicators

Provide structured, actionable intelligence that helps evaluate sponsorship potential.
Always cite sources when possible and indicate confidence levels for each data point.`
}

func (a *CompanyIntelAgent) Execute(ctx context.Context, input *Input) (*Output, error) {
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: buildSystemPrompt(a.SystemPrompt(), input)},
		{Role: provider.RoleUser, Content: fmt.Sprintf("Company ID: %s\nQuery: %s", input.CompanyID, input.Query)},
	}

	resp, err := a.provider.Complete(ctx, &provider.CompletionRequest{
		Model:       "gpt-4o",
		Messages:    messages,
		MaxTokens:   2048,
		Temperature: 0.5,
	})
	if err != nil {
		return nil, fmt.Errorf("company intel agent: %w", err)
	}

	return &Output{
		Response:   resp.Content,
		TokensUsed: resp.TokensUsed,
	}, nil
}

type DecisionMakerAgent struct {
	provider provider.Provider
}

func NewDecisionMakerAgent(p provider.Provider) *DecisionMakerAgent {
	return &DecisionMakerAgent{provider: p}
}

func (a *DecisionMakerAgent) Type() AgentType { return AgentDecisionMaker }

func (a *DecisionMakerAgent) SystemPrompt() string {
	return `You are an expert at identifying and mapping decision makers within organizations for sponsorship deals.
Your responsibilities:
- Identify likely budget holders for sponsorship/marketing spend
- Map organizational hierarchy relevant to sponsorship decisions
- Determine optimal contact sequences and entry points
- Analyze LinkedIn profiles and public information for relationship mapping
- Suggest warm introduction paths through existing networks
- Provide context on each person's role in the decision chain

Output structured contact recommendations with reasoning for each suggestion.`
}

func (a *DecisionMakerAgent) Execute(ctx context.Context, input *Input) (*Output, error) {
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: buildSystemPrompt(a.SystemPrompt(), input)},
		{Role: provider.RoleUser, Content: fmt.Sprintf("Company ID: %s\nQuery: %s", input.CompanyID, input.Query)},
	}

	resp, err := a.provider.Complete(ctx, &provider.CompletionRequest{
		Model:       "gpt-4o",
		Messages:    messages,
		MaxTokens:   1536,
		Temperature: 0.4,
	})
	if err != nil {
		return nil, fmt.Errorf("decision maker agent: %w", err)
	}

	return &Output{
		Response:   resp.Content,
		TokensUsed: resp.TokensUsed,
	}, nil
}

type DuplicateAgent struct {
	provider provider.Provider
}

func NewDuplicateAgent(p provider.Provider) *DuplicateAgent {
	return &DuplicateAgent{provider: p}
}

func (a *DuplicateAgent) Type() AgentType { return AgentDuplicate }

func (a *DuplicateAgent) SystemPrompt() string {
	return `You are a deduplication specialist for CRM and sponsorship data.
Your responsibilities:
- Identify potential duplicate records across companies, contacts, and sponsors
- Analyze fuzzy matches considering name variations, typos, abbreviations
- Compare metadata fields to determine match confidence
- Recommend merge strategies preserving the most complete data
- Flag false positives that appear similar but are distinct entities

Output a structured list of potential duplicates with confidence scores and merge recommendations.`
}

func (a *DuplicateAgent) Execute(ctx context.Context, input *Input) (*Output, error) {
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: buildSystemPrompt(a.SystemPrompt(), input)},
		{Role: provider.RoleUser, Content: input.Query},
	}

	resp, err := a.provider.Complete(ctx, &provider.CompletionRequest{
		Model:       "gpt-4o-mini",
		Messages:    messages,
		MaxTokens:   1024,
		Temperature: 0.2,
	})
	if err != nil {
		return nil, fmt.Errorf("duplicate agent: %w", err)
	}

	return &Output{
		Response:   resp.Content,
		TokensUsed: resp.TokensUsed,
	}, nil
}

type CRMAgent struct {
	provider provider.Provider
}

func NewCRMAgent(p provider.Provider) *CRMAgent {
	return &CRMAgent{provider: p}
}

func (a *CRMAgent) Type() AgentType { return AgentCRM }

func (a *CRMAgent) SystemPrompt() string {
	return `You are a CRM operations specialist managing sponsorship relationship data.
Your responsibilities:
- Suggest optimal pipeline stage transitions based on activity signals
- Recommend follow-up actions and timing based on engagement patterns
- Generate activity summaries and relationship health scores
- Identify at-risk relationships needing attention
- Suggest data enrichment opportunities from interaction history
- Recommend tagging and categorization improvements

Provide actionable CRM recommendations with specific next steps and timing.`
}

func (a *CRMAgent) Execute(ctx context.Context, input *Input) (*Output, error) {
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: buildSystemPrompt(a.SystemPrompt(), input)},
		{Role: provider.RoleUser, Content: fmt.Sprintf("Sponsor: %s\nCampaign: %s\nQuery: %s", input.SponsorID, input.CampaignID, input.Query)},
	}

	resp, err := a.provider.Complete(ctx, &provider.CompletionRequest{
		Model:       "gpt-4o",
		Messages:    messages,
		MaxTokens:   1024,
		Temperature: 0.4,
	})
	if err != nil {
		return nil, fmt.Errorf("crm agent: %w", err)
	}

	return &Output{
		Response:   resp.Content,
		TokensUsed: resp.TokensUsed,
	}, nil
}
