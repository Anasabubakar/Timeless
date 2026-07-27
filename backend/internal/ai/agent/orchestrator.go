package agent

import (
	"context"

	"github.com/google/uuid"
	"github.com/timeless/backend/internal/ai/provider"
)

type AgentType string

const (
	AgentResearch      AgentType = "research"
	AgentQualification AgentType = "qualification"
	AgentCompanyIntel  AgentType = "company_intel"
	AgentDecisionMaker AgentType = "decision_maker"
	AgentDuplicate     AgentType = "duplicate"
	AgentCRM           AgentType = "crm"
	AgentOutreach      AgentType = "outreach"
	AgentMeeting       AgentType = "meeting"
	AgentAnalytics     AgentType = "analytics"
	AgentMemory        AgentType = "memory"
	AgentStrategy      AgentType = "strategy"

	AgentWorkspaceDiscovery AgentType = "workspace_discovery"
	AgentGoalRecommendation AgentType = "goal_recommendation"
	AgentAutomationPlanning AgentType = "automation_planning"
)

type Agent interface {
	Type() AgentType
	Execute(ctx context.Context, input *Input) (*Output, error)
	SystemPrompt() string
}

type Input struct {
	Query      string                 `json:"query"`
	Context    map[string]interface{} `json:"context,omitempty"`
	OrgID      string                 `json:"org_id"`
	UserID     string                 `json:"user_id"`
	CampaignID string                 `json:"campaign_id,omitempty"`
	SponsorID  string                 `json:"sponsor_id,omitempty"`
	CompanyID  string                 `json:"company_id,omitempty"`
}

type Output struct {
	Response   string                 `json:"response"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Actions    []Action               `json:"actions,omitempty"`
	TokensUsed provider.TokenUsage    `json:"tokens_used"`
}

type Action struct {
	Type   string                 `json:"type"`
	Target string                 `json:"target"`
	Params map[string]interface{} `json:"params"`
}

type Orchestrator struct {
	registry *provider.Registry
	agents   map[AgentType]Agent
	learning *LearningService
}

func NewOrchestrator(registry *provider.Registry, learning *LearningService) *Orchestrator {
	o := &Orchestrator{
		registry: registry,
		agents:   make(map[AgentType]Agent),
		learning: learning,
	}
	o.initAgents()
	return o
}

func (o *Orchestrator) initAgents() {
	p, err := o.registry.Default()
	if err != nil {
		return
	}

	o.Register(NewResearchAgent(p))
	o.Register(NewQualificationAgent(p))
	o.Register(NewCompanyIntelAgent(p))
	o.Register(NewDecisionMakerAgent(p))
	o.Register(NewDuplicateAgent(p))
	o.Register(NewCRMAgent(p))
	o.Register(NewOutreachAgent(p))
	o.Register(NewMeetingAgent(p))
	o.Register(NewAnalyticsAgent(p))
	o.Register(NewMemoryAgent(p))
	o.Register(NewStrategyAgent(p))

	o.Register(NewWorkspaceDiscoveryAgent(p))
	o.Register(NewGoalRecommendationAgent(p))
	o.Register(NewAutomationPlanningAgent(p))
}

func (o *Orchestrator) Register(agent Agent) {
	o.agents[agent.Type()] = agent
}

func (o *Orchestrator) Execute(ctx context.Context, agentType AgentType, input *Input) (*Output, error) {
	agent, ok := o.agents[agentType]
	if !ok {
		return nil, &AgentNotFoundError{Type: agentType}
	}

	if o.learning != nil && input.OrgID != "" {
		orgID := uuid.MustParse(input.OrgID)
		if injection := o.learning.BuildContextInjection(ctx, orgID, agentType); injection != "" {
			if input.Context == nil {
				input.Context = make(map[string]interface{})
			}
			input.Context["learned_context"] = injection
		}
	}

	return agent.Execute(ctx, input)
}

func (o *Orchestrator) Route(ctx context.Context, input *Input) (AgentType, error) {
	p, err := o.registry.Default()
	if err != nil {
		return "", err
	}

	resp, err := p.Complete(ctx, &provider.CompletionRequest{
		Messages: []provider.Message{
			{
				Role:    provider.RoleSystem,
				Content: routingPrompt,
			},
			{
				Role:    provider.RoleUser,
				Content: input.Query,
			},
		},
		MaxTokens:   50,
		Temperature: 0,
	})
	if err != nil {
		return AgentResearch, nil
	}

	switch resp.Content {
	case "research":
		return AgentResearch, nil
	case "qualification":
		return AgentQualification, nil
	case "company_intel":
		return AgentCompanyIntel, nil
	case "decision_maker":
		return AgentDecisionMaker, nil
	case "outreach":
		return AgentOutreach, nil
	case "meeting":
		return AgentMeeting, nil
	case "analytics":
		return AgentAnalytics, nil
	case "strategy":
		return AgentStrategy, nil
	default:
		return AgentResearch, nil
	}
}

type AgentNotFoundError struct {
	Type AgentType
}

func (e *AgentNotFoundError) Error() string {
	return "agent not found: " + string(e.Type)
}

const routingPrompt = `You are a routing agent. Given a user query about sponsorship operations, determine which specialized agent should handle it. Respond with exactly one of these agent types:
- research: finding new sponsors, discovering companies, market research
- qualification: scoring sponsors, evaluating fit, budget analysis
- company_intel: enriching company data, finding news, tech stack analysis
- decision_maker: finding contacts, identifying stakeholders
- outreach: drafting emails, proposals, follow-ups
- meeting: preparing briefs, summarizing calls
- analytics: pipeline metrics, conversion rates, forecasting
- strategy: pricing, packaging, negotiation tactics

Respond with ONLY the agent type name, nothing else.`
