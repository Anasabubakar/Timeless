package agent

import (
	"context"
	"fmt"

	"github.com/timeless/backend/internal/ai/provider"
)

// buildSystemPrompt appends any learned context from the orchestrator to the
// agent's static system prompt so each model call benefits from past outcomes.
func buildSystemPrompt(base string, input *Input) string {
	if input.Context == nil {
		return base
	}
	if ctx, ok := input.Context["learned_context"].(string); ok && ctx != "" {
		return base + ctx
	}
	return base
}

type ResearchAgent struct {
	provider provider.Provider
}

func NewResearchAgent(p provider.Provider) *ResearchAgent {
	return &ResearchAgent{provider: p}
}

func (a *ResearchAgent) Type() AgentType { return AgentResearch }

func (a *ResearchAgent) SystemPrompt() string {
	return `You are an expert sponsorship research agent. Your job is to:
1. Find potential sponsors that match the organization's target profile
2. Analyze companies' sponsorship history and patterns
3. Identify emerging opportunities in the market
4. Surface news and events that indicate sponsorship potential

When researching, consider:
- Company size, industry, and growth trajectory
- Recent funding, acquisitions, or product launches
- Their marketing budget signals and brand priorities
- Past sponsorship activity and event participation
- Alignment with the organization's audience and values

Provide structured, actionable research with clear recommendations.`
}

func (a *ResearchAgent) Execute(ctx context.Context, input *Input) (*Output, error) {
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: buildSystemPrompt(a.SystemPrompt(), input)},
		{Role: provider.RoleUser, Content: input.Query},
	}

	resp, err := a.provider.Complete(ctx, &provider.CompletionRequest{
		Model:       "",
		Messages:    messages,
		MaxTokens:   2048,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, fmt.Errorf("research agent completion: %w", err)
	}

	return &Output{
		Response:   resp.Content,
		TokensUsed: resp.TokensUsed,
	}, nil
}

type QualificationAgent struct {
	provider provider.Provider
}

func NewQualificationAgent(p provider.Provider) *QualificationAgent {
	return &QualificationAgent{provider: p}
}

func (a *QualificationAgent) Type() AgentType { return AgentQualification }

func (a *QualificationAgent) SystemPrompt() string {
	return `You are a sponsorship qualification agent. Your job is to evaluate and score potential sponsors based on:
1. Budget fit - Can they afford the sponsorship levels offered?
2. Audience alignment - Does their target market overlap with the organization's audience?
3. Brand compatibility - Are their values and image compatible?
4. Timing signals - Are they actively spending on marketing/events?
5. Decision-maker accessibility - Can we reach the right people?

Score each criterion 1-10 and provide an overall qualification score (0-100).
Explain your reasoning and suggest next steps.`
}

func (a *QualificationAgent) Execute(ctx context.Context, input *Input) (*Output, error) {
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: buildSystemPrompt(a.SystemPrompt(), input)},
		{Role: provider.RoleUser, Content: input.Query},
	}

	resp, err := a.provider.Complete(ctx, &provider.CompletionRequest{
		Model:       "",
		Messages:    messages,
		MaxTokens:   1024,
		Temperature: 0.3,
	})
	if err != nil {
		return nil, fmt.Errorf("qualification agent completion: %w", err)
	}

	return &Output{
		Response:   resp.Content,
		TokensUsed: resp.TokensUsed,
	}, nil
}

type OutreachAgent struct {
	provider provider.Provider
}

func NewOutreachAgent(p provider.Provider) *OutreachAgent {
	return &OutreachAgent{provider: p}
}

func (a *OutreachAgent) Type() AgentType { return AgentOutreach }

func (a *OutreachAgent) SystemPrompt() string {
	return `You are a sponsorship outreach specialist. Your job is to craft compelling, personalized communications including:
1. Initial cold outreach emails that get responses
2. Follow-up sequences that nurture without being pushy
3. Sponsorship proposals tailored to each company
4. Meeting request emails and LinkedIn messages
5. Thank you and post-event communications

Guidelines:
- Keep emails concise (under 150 words for cold outreach)
- Lead with value, not asks
- Personalize based on the company's recent activity
- Include clear, specific CTAs
- Match tone to the recipient's communication style`
}

func (a *OutreachAgent) Execute(ctx context.Context, input *Input) (*Output, error) {
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: buildSystemPrompt(a.SystemPrompt(), input)},
		{Role: provider.RoleUser, Content: input.Query},
	}

	resp, err := a.provider.Complete(ctx, &provider.CompletionRequest{
		Model:       "",
		Messages:    messages,
		MaxTokens:   1024,
		Temperature: 0.8,
	})
	if err != nil {
		return nil, fmt.Errorf("outreach agent completion: %w", err)
	}

	return &Output{
		Response:   resp.Content,
		TokensUsed: resp.TokensUsed,
	}, nil
}
