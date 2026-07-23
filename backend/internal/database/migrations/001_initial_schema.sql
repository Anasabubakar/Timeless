-- SponsorOS Full Schema Migration
-- Multi-tenant SaaS database with pgvector support

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "vector";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- =============================================================================
-- ORGANIZATIONS & TENANCY
-- =============================================================================

CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE,
    logo_url TEXT,
    domain VARCHAR(255),
    plan VARCHAR(50) NOT NULL DEFAULT 'free',
    settings JSONB NOT NULL DEFAULT '{}',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_organizations_slug ON organizations(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_organizations_deleted_at ON organizations(deleted_at);

-- =============================================================================
-- USERS & AUTH
-- =============================================================================

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255),
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    avatar_url TEXT,
    phone VARCHAR(50),
    job_title VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    last_login_at TIMESTAMPTZ,
    preferences JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_org ON users(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    permissions JSONB NOT NULL DEFAULT '[]',
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_roles_org ON roles(organization_id) WHERE deleted_at IS NULL;

CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE oauth_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    provider_id VARCHAR(255) NOT NULL,
    access_token TEXT,
    refresh_token TEXT,
    token_expires TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_oauth_provider ON oauth_accounts(provider, provider_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_oauth_user ON oauth_accounts(user_id) WHERE deleted_at IS NULL;

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(500) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_refresh_token ON refresh_tokens(token);
CREATE INDEX idx_refresh_user ON refresh_tokens(user_id);

-- =============================================================================
-- TEAMS
-- =============================================================================

CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_teams_org ON teams(organization_id) WHERE deleted_at IS NULL;

CREATE TABLE team_members (
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    PRIMARY KEY (team_id, user_id)
);

-- =============================================================================
-- INDUSTRIES
-- =============================================================================

CREATE TABLE industries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    parent_id UUID REFERENCES industries(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX idx_industries_name ON industries(name);
CREATE UNIQUE INDEX idx_industries_slug ON industries(slug);

-- =============================================================================
-- COMPANIES & CONTACTS
-- =============================================================================

CREATE TABLE companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255),
    website TEXT,
    logo_url TEXT,
    description TEXT,
    industry_id UUID REFERENCES industries(id) ON DELETE SET NULL,
    employee_count VARCHAR(50),
    annual_revenue VARCHAR(100),
    headquarters VARCHAR(255),
    founded_year INTEGER,
    linkedin_url TEXT,
    twitter_url TEXT,
    phone VARCHAR(50),
    address JSONB,
    tags TEXT[] NOT NULL DEFAULT '{}',
    enrichment_data JSONB NOT NULL DEFAULT '{}',
    score INTEGER,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    source VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_companies_org ON companies(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_companies_name ON companies USING gin (name gin_trgm_ops);
CREATE INDEX idx_companies_domain ON companies(organization_id, domain) WHERE deleted_at IS NULL;
CREATE INDEX idx_companies_industry ON companies(industry_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_companies_deleted_at ON companies(deleted_at);

CREATE TABLE decision_makers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    title VARCHAR(255),
    department VARCHAR(255),
    email VARCHAR(255),
    phone VARCHAR(50),
    linkedin_url TEXT,
    twitter_url TEXT,
    influence_level VARCHAR(20),
    bio TEXT,
    profile_data JSONB NOT NULL DEFAULT '{}',
    avatar_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_dm_org ON decision_makers(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_dm_company ON decision_makers(company_id) WHERE deleted_at IS NULL;

CREATE TABLE pain_points (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    severity VARCHAR(20),
    source VARCHAR(50),
    evidence JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_painpoints_company ON pain_points(company_id) WHERE deleted_at IS NULL;

-- =============================================================================
-- PROJECTS & CAMPAIGNS
-- =============================================================================

CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    start_date DATE,
    end_date DATE,
    budget NUMERIC(15,2),
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    settings JSONB NOT NULL DEFAULT '{}',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_projects_org ON projects(organization_id) WHERE deleted_at IS NULL;

CREATE TABLE campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    goal_amount NUMERIC(15,2),
    raised_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    start_date DATE,
    end_date DATE,
    pipeline_stages JSONB NOT NULL DEFAULT '["prospect","contacted","qualified","proposal","negotiation","closed_won","closed_lost"]',
    settings JSONB NOT NULL DEFAULT '{}',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_campaigns_org ON campaigns(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_campaigns_project ON campaigns(project_id) WHERE deleted_at IS NULL;

-- =============================================================================
-- SPONSORS (Pipeline)
-- =============================================================================

CREATE TABLE sponsors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    stage VARCHAR(50) NOT NULL DEFAULT 'prospect',
    stage_entered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deal_value NUMERIC(15,2),
    probability INTEGER,
    tier VARCHAR(50),
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    primary_contact_id UUID REFERENCES decision_makers(id) ON DELETE SET NULL,
    expected_close DATE,
    actual_close DATE,
    lost_reason TEXT,
    notes TEXT,
    custom_fields JSONB NOT NULL DEFAULT '{}',
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_sponsors_org ON sponsors(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_sponsors_campaign ON sponsors(campaign_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_sponsors_company ON sponsors(company_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_sponsors_stage ON sponsors(organization_id, campaign_id, stage) WHERE deleted_at IS NULL;
CREATE INDEX idx_sponsors_assigned ON sponsors(assigned_to) WHERE deleted_at IS NULL;

CREATE TABLE proposals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    sponsor_id UUID NOT NULL REFERENCES sponsors(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    version INTEGER NOT NULL DEFAULT 1,
    amount NUMERIC(15,2),
    valid_until DATE,
    sent_at TIMESTAMPTZ,
    viewed_at TIMESTAMPTZ,
    responded_at TIMESTAMPTZ,
    attachments JSONB NOT NULL DEFAULT '[]',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_proposals_org ON proposals(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_proposals_sponsor ON proposals(sponsor_id) WHERE deleted_at IS NULL;

-- =============================================================================
-- COMMUNICATIONS
-- =============================================================================

CREATE TABLE communications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    sponsor_id UUID REFERENCES sponsors(id) ON DELETE SET NULL,
    contact_id UUID REFERENCES decision_makers(id) ON DELETE SET NULL,
    type VARCHAR(50) NOT NULL,
    direction VARCHAR(10) NOT NULL DEFAULT 'outbound',
    subject VARCHAR(500),
    body TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    sent_at TIMESTAMPTZ,
    opened_at TIMESTAMPTZ,
    clicked_at TIMESTAMPTZ,
    replied_at TIMESTAMPTZ,
    bounced_at TIMESTAMPTZ,
    channel VARCHAR(50),
    external_id VARCHAR(255),
    template_id UUID,
    metadata JSONB NOT NULL DEFAULT '{}',
    sent_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_comms_org ON communications(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_comms_sponsor ON communications(sponsor_id) WHERE deleted_at IS NULL;

CREATE TABLE email_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    subject VARCHAR(500) NOT NULL,
    body TEXT NOT NULL,
    category VARCHAR(100),
    variables JSONB NOT NULL DEFAULT '[]',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_templates_org ON email_templates(organization_id) WHERE deleted_at IS NULL;

-- =============================================================================
-- TASKS & ACTIVITIES
-- =============================================================================

CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    priority VARCHAR(20) NOT NULL DEFAULT 'medium',
    due_date DATE,
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    entity_type VARCHAR(50),
    entity_id UUID,
    completed_at TIMESTAMPTZ,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_tasks_org ON tasks(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_assigned ON tasks(assigned_to, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_entity ON tasks(entity_type, entity_id) WHERE deleted_at IS NULL;

CREATE TABLE activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    description TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    ip_address VARCHAR(45),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_activities_org ON activities(organization_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_activities_entity ON activities(entity_type, entity_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_activities_user ON activities(user_id) WHERE deleted_at IS NULL;

-- =============================================================================
-- AI SYSTEM
-- =============================================================================

CREATE TABLE ai_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) NOT NULL,
    api_key_enc TEXT,
    base_url TEXT,
    models JSONB NOT NULL DEFAULT '[]',
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    is_global BOOLEAN NOT NULL DEFAULT FALSE,
    rate_limit INTEGER,
    config JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_ai_providers_org ON ai_providers(organization_id) WHERE deleted_at IS NULL;

CREATE TABLE ai_conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255),
    agent_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    context JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_ai_convos_org ON ai_conversations(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_ai_convos_user ON ai_conversations(user_id) WHERE deleted_at IS NULL;

CREATE TABLE ai_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    tool_calls JSONB,
    tool_results JSONB,
    tokens_used INTEGER,
    model VARCHAR(100),
    latency_ms INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_ai_msgs_convo ON ai_messages(conversation_id, created_at ASC) WHERE deleted_at IS NULL;

-- =============================================================================
-- KNOWLEDGE GRAPH & AI MEMORY
-- =============================================================================

CREATE TABLE knowledge_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    node_type VARCHAR(50) NOT NULL,
    entity_id UUID,
    label VARCHAR(500) NOT NULL,
    properties JSONB NOT NULL DEFAULT '{}',
    embedding vector(1536),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_kn_org ON knowledge_nodes(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_kn_type ON knowledge_nodes(organization_id, node_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_kn_entity ON knowledge_nodes(entity_id) WHERE deleted_at IS NULL AND entity_id IS NOT NULL;
CREATE INDEX idx_kn_embedding ON knowledge_nodes USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

CREATE TABLE knowledge_edges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    source_node_id UUID NOT NULL REFERENCES knowledge_nodes(id) ON DELETE CASCADE,
    target_node_id UUID NOT NULL REFERENCES knowledge_nodes(id) ON DELETE CASCADE,
    relation_type VARCHAR(100) NOT NULL,
    weight DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    properties JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_ke_org ON knowledge_edges(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_ke_source ON knowledge_edges(source_node_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_ke_target ON knowledge_edges(target_node_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_ke_relation ON knowledge_edges(relation_type) WHERE deleted_at IS NULL;

CREATE TABLE ai_memories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    memory_type VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    embedding vector(1536),
    importance DOUBLE PRECISION NOT NULL DEFAULT 0.5,
    access_count INTEGER NOT NULL DEFAULT 0,
    last_accessed TIMESTAMPTZ,
    entity_type VARCHAR(50),
    entity_id UUID,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_mem_org ON ai_memories(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_mem_type ON ai_memories(organization_id, memory_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_mem_entity ON ai_memories(entity_type, entity_id) WHERE deleted_at IS NULL AND entity_type IS NOT NULL;
CREATE INDEX idx_mem_embedding ON ai_memories USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- =============================================================================
-- INTEGRATIONS & AUTOMATIONS
-- =============================================================================

CREATE TABLE integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    provider VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'inactive',
    credentials JSONB,
    config JSONB NOT NULL DEFAULT '{}',
    last_sync_at TIMESTAMPTZ,
    last_error TEXT,
    webhook_url TEXT,
    webhook_secret TEXT,
    installed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_integrations_org ON integrations(organization_id) WHERE deleted_at IS NULL;

CREATE TABLE webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    integration_id UUID REFERENCES integrations(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    events JSONB NOT NULL DEFAULT '[]',
    secret VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_triggered TIMESTAMPTZ,
    failure_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_webhooks_org ON webhooks(organization_id) WHERE deleted_at IS NULL;

CREATE TABLE automations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    trigger_type VARCHAR(50) NOT NULL,
    trigger_config JSONB NOT NULL DEFAULT '{}',
    actions JSONB NOT NULL DEFAULT '[]',
    conditions JSONB NOT NULL DEFAULT '[]',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    run_count INTEGER NOT NULL DEFAULT 0,
    last_run_at TIMESTAMPTZ,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_automations_org ON automations(organization_id) WHERE deleted_at IS NULL;

-- =============================================================================
-- SEED DEFAULT INDUSTRIES
-- =============================================================================

INSERT INTO industries (name, slug) VALUES
    ('Technology', 'technology'),
    ('Finance', 'finance'),
    ('Healthcare', 'healthcare'),
    ('Education', 'education'),
    ('Retail', 'retail'),
    ('Manufacturing', 'manufacturing'),
    ('Media & Entertainment', 'media-entertainment'),
    ('Real Estate', 'real-estate'),
    ('Energy', 'energy'),
    ('Transportation', 'transportation'),
    ('Food & Beverage', 'food-beverage'),
    ('Sports', 'sports'),
    ('Automotive', 'automotive'),
    ('Telecommunications', 'telecommunications'),
    ('Consumer Goods', 'consumer-goods'),
    ('Professional Services', 'professional-services'),
    ('Non-Profit', 'non-profit'),
    ('Government', 'government')
ON CONFLICT (name) DO NOTHING;
