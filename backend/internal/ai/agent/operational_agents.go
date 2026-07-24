package agent

import (
	"context"
	"fmt"

	"github.com/timeless/backend/internal/ai/provider"
)

type MeetingAgent struct {
	provider provider.Provider
}

func NewMeetingAgent(p provider.Provider) *MeetingAgent {
	return &MeetingAgent{provider: p}
}

func (a *MeetingAgent) Type() AgentType { return AgentMeeting }

func (a *MeetingAgent) SystemPrompt() string {
	return `You are a meeting preparation and follow-up specialist for sponsorship conversations.
Your responsibilities:
- Generate meeting preparation briefs with key talking points
- Suggest questions to ask based on sponsor profile and stage
- Create post-meeting summaries from notes or transcripts
- Extract action items and commitments from meeting context
- Draft follow-up communications based on meeting outcomes
- Track meeting cadence and suggest optimal scheduling

Provide structured meeting intelligence that accelerates deal progression.`
}

func (a *MeetingAgent) Execute(ctx context.Context, input *Input) (*Output, error) {
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: buildSystemPrompt(a.SystemPrompt(), input)},
		{Role: provider.RoleUser, Content: fmt.Sprintf("Sponsor: %s\nCompany: %s\nQuery: %s", input.SponsorID, input.CompanyID, input.Query)},
	}

	resp, err := a.provider.Complete(ctx, &provider.CompletionRequest{
		Model:       "gpt-4o",
		Messages:    messages,
		MaxTokens:   1536,
		Temperature: 0.6,
	})
	if err != nil {
		return nil, fmt.Errorf("meeting agent: %w", err)
	}

	return &Output{
		Response:   resp.Content,
		TokensUsed: resp.TokensUsed,
	}, nil
}

type AnalyticsAgent struct {
	provider provider.Provider
}

func NewAnalyticsAgent(p provider.Provider) *AnalyticsAgent {
	return &AnalyticsAgent{provider: p}
}

func (a *AnalyticsAgent) Type() AgentType { return AgentAnalytics }

func (a *AnalyticsAgent) SystemPrompt() string {
	return `You are a sponsorship analytics and insights specialist.
Your responsibilities:
- Analyze pipeline health metrics and conversion rates
- Identify trends in sponsor engagement and deal velocity
- Generate performance reports and ROI calculations
- Predict deal outcomes based on historical patterns
- Recommend pipeline optimizations and bottleneck fixes
- Compare campaign performance across time periods

Provide data-driven insights with specific metrics, percentages, and actionable recommendations.`
}

func (a *AnalyticsAgent) Execute(ctx context.Context, input *Input) (*Output, error) {
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: buildSystemPrompt(a.SystemPrompt(), input)},
		{Role: provider.RoleUser, Content: fmt.Sprintf("Campaign: %s\nQuery: %s", input.CampaignID, input.Query)},
	}

	resp, err := a.provider.Complete(ctx, &provider.CompletionRequest{
		Model:       "gpt-4o",
		Messages:    messages,
		MaxTokens:   2048,
		Temperature: 0.3,
	})
	if err != nil {
		return nil, fmt.Errorf("analytics agent: %w", err)
	}

	return &Output{
		Response:   resp.Content,
		TokensUsed: resp.TokensUsed,
	}, nil
}

type MemoryAgent struct {
	provider provider.Provider
}

func NewMemoryAgent(p provider.Provider) *MemoryAgent {
	return &MemoryAgent{provider: p}
}

func (a *MemoryAgent) Type() AgentType { return AgentMemory }

func (a *MemoryAgent) SystemPrompt() string {
	return `You are an organizational memory and knowledge management specialist.
Your responsibilities:
- Recall relevant past interactions, decisions, and context
- Connect current queries to historical knowledge
- Identify patterns across the organization's sponsorship history
- Surface relevant precedents for current situations
- Maintain context continuity across conversations
- Suggest knowledge gaps that should be filled

Provide contextual recall that makes every interaction more informed.`
}

func (a *MemoryAgent) Execute(ctx context.Context, input *Input) (*Output, error) {
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: buildSystemPrompt(a.SystemPrompt(), input)},
		{Role: provider.RoleUser, Content: input.Query},
	}

	resp, err := a.provider.Complete(ctx, &provider.CompletionRequest{
		Model:       "gpt-4o-mini",
		Messages:    messages,
		MaxTokens:   1024,
		Temperature: 0.3,
	})
	if err != nil {
		return nil, fmt.Errorf("memory agent: %w", err)
	}

	return &Output{
		Response:   resp.Content,
		TokensUsed: resp.TokensUsed,
	}, nil
}

type StrategyAgent struct {
	provider provider.Provider
}

func NewStrategyAgent(p provider.Provider) *StrategyAgent {
	return &StrategyAgent{provider: p}
}

func (a *StrategyAgent) Type() AgentType { return AgentStrategy }

func (a *StrategyAgent) SystemPrompt() string {
	return `You are a sponsorship strategy advisor providing high-level guidance.
Your responsibilities:
- Develop sponsorship pricing and packaging strategies
- Analyze market positioning relative to competing events/properties
- Recommend sponsor targeting criteria and ideal customer profiles
- Suggest campaign structures and timeline optimization
- Provide competitive intelligence on similar sponsorship programs
- Create long-term relationship development strategies

Provide strategic recommendations with clear rationale and expected outcomes.`
}

func (a *StrategyAgent) Execute(ctx context.Context, input *Input) (*Output, error) {
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: buildSystemPrompt(a.SystemPrompt(), input)},
		{Role: provider.RoleUser, Content: fmt.Sprintf("Campaign: %s\nQuery: %s", input.CampaignID, input.Query)},
	}

	resp, err := a.provider.Complete(ctx, &provider.CompletionRequest{
		Model:       "gpt-4o",
		Messages:    messages,
		MaxTokens:   2048,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, fmt.Errorf("strategy agent: %w", err)
	}

	return &Output{
		Response:   resp.Content,
		TokensUsed: resp.TokensUsed,
	}, nil
}
