-- Agent Learning & Outcome Tracking
CREATE TABLE IF NOT EXISTS agent_outcomes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_type VARCHAR(50) NOT NULL,
    conversation_id UUID,
    query TEXT NOT NULL,
    response TEXT DEFAULT '',
    outcome VARCHAR(30) NOT NULL,
    score DOUBLE PRECISION DEFAULT 0,
    feedback TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_agent_outcomes_org_type ON agent_outcomes(organization_id, agent_type);
CREATE INDEX idx_agent_outcomes_outcome ON agent_outcomes(outcome);
CREATE INDEX idx_agent_outcomes_created ON agent_outcomes(created_at DESC);

CREATE TABLE IF NOT EXISTS agent_learned_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_type VARCHAR(50) NOT NULL,
    category VARCHAR(100) NOT NULL,
    preference TEXT NOT NULL,
    confidence DOUBLE PRECISION DEFAULT 0.5,
    learned_from INTEGER DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(organization_id, agent_type, category)
);

CREATE INDEX idx_agent_prefs_org_type ON agent_learned_preferences(organization_id, agent_type);
CREATE INDEX idx_agent_prefs_confidence ON agent_learned_preferences(confidence DESC);
