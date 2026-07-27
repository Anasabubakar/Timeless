package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/timeless/backend/internal/ai/provider"
)

// extractJSON strips markdown code fences models sometimes wrap structured
// output in, so callers can json.Unmarshal the response directly.
func extractJSON(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "```") {
		trimmed = strings.TrimPrefix(trimmed, "```json")
		trimmed = strings.TrimPrefix(trimmed, "```")
		trimmed = strings.TrimSuffix(trimmed, "```")
		trimmed = strings.TrimSpace(trimmed)
	}
	return trimmed
}

// WorkspaceDiscoveryAgent inspects a coarse summary of what's connected
// (Notion pages found, Apollo contacts found, Zapier tools available) and
// infers what projects/workspaces the user is likely working on.
type WorkspaceDiscoveryAgent struct {
	provider provider.Provider
}

func NewWorkspaceDiscoveryAgent(p provider.Provider) *WorkspaceDiscoveryAgent {
	return &WorkspaceDiscoveryAgent{provider: p}
}

func (a *WorkspaceDiscoveryAgent) Type() AgentType { return AgentWorkspaceDiscovery }

func (a *WorkspaceDiscoveryAgent) SystemPrompt() string {
	return `You are a workspace discovery agent. Given a summary of a user's connected
integrations and what was found there, infer what distinct projects, teams, or
initiatives they're likely working on.

Respond with ONLY a JSON array (no prose, no markdown fences), where each item is:
{
  "name": "short project name",
  "confidence": 0-100,
  "explanation": "one sentence on why you inferred this, citing which connected source it came from",
  "sources": ["notion", "apollo", "zapier"],
  "document_count": integer estimate of related items,
  "recent_activity": "brief description or 'unknown' if the data doesn't indicate recency"
}

If the connected data is too sparse to confidently infer specific projects, return
1-2 lower-confidence generic entries (e.g. "General workspace") rather than
inventing specifics you have no evidence for. Never fabricate document counts —
base them on the numbers given to you.`
}

func (a *WorkspaceDiscoveryAgent) Execute(ctx context.Context, input *Input) (*Output, error) {
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: buildSystemPrompt(a.SystemPrompt(), input)},
		{Role: provider.RoleUser, Content: input.Query},
	}

	resp, err := a.provider.Complete(ctx, &provider.CompletionRequest{
		Model:       "",
		Messages:    messages,
		MaxTokens:   1024,
		Temperature: 0.4,
	})
	if err != nil {
		return nil, fmt.Errorf("workspace discovery agent completion: %w", err)
	}

	return &Output{
		Response:   extractJSON(resp.Content),
		TokensUsed: resp.TokensUsed,
	}, nil
}

// GoalRecommendationAgent suggests what the user might want to accomplish
// given the projects/workspace context they selected.
type GoalRecommendationAgent struct {
	provider provider.Provider
}

func NewGoalRecommendationAgent(p provider.Provider) *GoalRecommendationAgent {
	return &GoalRecommendationAgent{provider: p}
}

func (a *GoalRecommendationAgent) Type() AgentType { return AgentGoalRecommendation }

func (a *GoalRecommendationAgent) SystemPrompt() string {
	return `You are a goal recommendation agent for a sponsorship/partnerships operating
system. Given the projects/workspace context a user selected, recommend goals they
likely want to accomplish.

Respond with ONLY a JSON array (no prose, no markdown fences), where each item is:
{
  "title": "short goal, e.g. 'Research sponsorships'",
  "description": "one sentence describing the goal",
  "automation_summary": "one sentence on how this would be automated if chosen"
}

Ground recommendations in sponsorship/partnership operations: research, outreach,
CRM hygiene, proposal generation, meeting prep, lead enrichment. Return 4-6 goals,
ordered by relevance to the given context.`
}

func (a *GoalRecommendationAgent) Execute(ctx context.Context, input *Input) (*Output, error) {
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: buildSystemPrompt(a.SystemPrompt(), input)},
		{Role: provider.RoleUser, Content: input.Query},
	}

	resp, err := a.provider.Complete(ctx, &provider.CompletionRequest{
		Model:       "",
		Messages:    messages,
		MaxTokens:   1024,
		Temperature: 0.5,
	})
	if err != nil {
		return nil, fmt.Errorf("goal recommendation agent completion: %w", err)
	}

	return &Output{
		Response:   extractJSON(resp.Content),
		TokensUsed: resp.TokensUsed,
	}, nil
}

// AutomationPlanningAgent turns a chosen goal into a concrete list of
// automations Timeless can run on the user's behalf.
type AutomationPlanningAgent struct {
	provider provider.Provider
}

func NewAutomationPlanningAgent(p provider.Provider) *AutomationPlanningAgent {
	return &AutomationPlanningAgent{provider: p}
}

func (a *AutomationPlanningAgent) Type() AgentType { return AgentAutomationPlanning }

func (a *AutomationPlanningAgent) SystemPrompt() string {
	return `You are an automation planning agent. Given a goal the user chose, propose a
concrete automation plan made of discrete, schedulable automations.

Respond with ONLY a JSON array (no prose, no markdown fences), where each item is:
{
  "title": "short automation name, e.g. 'Daily research digest'",
  "description": "what it does and why it helps reach the goal",
  "trigger_type": one of "schedule", "event", "manual",
  "trigger_config": {"cron": "0 8 * * *"} for schedule triggers, or {} otherwise,
  "action_type": one of "research", "outreach", "crm_cleanup", "meeting_prep", "enrichment", "proposal", "analytics"
}

Return 3-5 automations. Keep each one narrowly scoped and genuinely useful —
do not propose vague or redundant steps.`
}

func (a *AutomationPlanningAgent) Execute(ctx context.Context, input *Input) (*Output, error) {
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: buildSystemPrompt(a.SystemPrompt(), input)},
		{Role: provider.RoleUser, Content: input.Query},
	}

	resp, err := a.provider.Complete(ctx, &provider.CompletionRequest{
		Model:       "",
		Messages:    messages,
		MaxTokens:   1024,
		Temperature: 0.4,
	})
	if err != nil {
		return nil, fmt.Errorf("automation planning agent completion: %w", err)
	}

	return &Output{
		Response:   extractJSON(resp.Content),
		TokensUsed: resp.TokensUsed,
	}, nil
}
