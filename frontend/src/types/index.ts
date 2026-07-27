export interface User {
  id: string;
  organization_id: string;
  email: string;
  first_name: string;
  last_name: string;
  avatar_url?: string;
  phone?: string;
  job_title?: string;
  status: string;
  last_login_at?: string;
  roles?: Role[];
  created_at: string;
  onboarding_completed: boolean;
  onboarding_completed_at?: string;
}

export interface Role {
  id: string;
  organization_id: string;
  name: string;
  permissions: string[];
  is_system: boolean;
}

export interface Organization {
  id: string;
  name: string;
  slug: string;
  logo_url?: string;
  domain?: string;
  plan: string;
  settings: Record<string, unknown>;
  created_at: string;
}

export interface Company {
  id: string;
  organization_id: string;
  name: string;
  domain?: string;
  website?: string;
  logo_url?: string;
  description?: string;
  industry_id?: string;
  industry?: Industry;
  employee_count?: string;
  annual_revenue?: string;
  headquarters?: string;
  founded_year?: number;
  linkedin_url?: string;
  tags: string[];
  enrichment_data: Record<string, unknown>;
  score?: number;
  status: string;
  source?: string;
  decision_makers?: DecisionMaker[];
  created_at: string;
}

export interface Industry {
  id: string;
  name: string;
  slug: string;
  parent_id?: string;
}

export interface DecisionMaker {
  id: string;
  organization_id: string;
  company_id: string;
  first_name: string;
  last_name: string;
  title?: string;
  department?: string;
  email?: string;
  phone?: string;
  linkedin_url?: string;
  influence_level?: string;
  avatar_url?: string;
}

export interface Project {
  id: string;
  organization_id: string;
  name: string;
  description?: string;
  status: string;
  start_date?: string;
  end_date?: string;
  budget?: number;
  currency: string;
  created_at: string;
}

export interface Campaign {
  id: string;
  organization_id: string;
  project_id: string;
  project?: Project;
  name: string;
  description?: string;
  status: string;
  goal_amount?: number;
  raised_amount: number;
  currency: string;
  start_date?: string;
  end_date?: string;
  pipeline_stages: string[];
  sponsors?: Sponsor[];
  created_at: string;
}

export interface Sponsor {
  id: string;
  organization_id: string;
  campaign_id: string;
  company_id: string;
  stage: string;
  stage_entered_at: string;
  deal_value?: number;
  probability?: number;
  tier?: string;
  assigned_to?: string;
  primary_contact_id?: string;
  expected_close?: string;
  notes?: string;
  custom_fields: Record<string, unknown>;
  position: number;
  company?: Company;
  campaign?: Campaign;
  assigned_user?: User;
  proposals?: Proposal[];
  created_at: string;
}

export interface Proposal {
  id: string;
  sponsor_id: string;
  title: string;
  content?: string;
  status: string;
  version: number;
  amount?: number;
  valid_until?: string;
  sent_at?: string;
  viewed_at?: string;
  created_at: string;
}

export interface Contact {
  id: string;
  organization_id: string;
  company_id?: string;
  company?: Company;
  first_name: string;
  last_name: string;
  email?: string;
  phone?: string;
  title?: string;
  department?: string;
  linkedin_url?: string;
  tags: string[];
  custom_fields: Record<string, unknown>;
  last_contacted_at?: string;
  status: string;
  created_at: string;
}

export interface Activity {
  id: string;
  organization_id: string;
  type: string;
  subject: string;
  description?: string;
  entity_type: string;
  entity_id: string;
  user_id: string;
  user?: User;
  metadata: Record<string, unknown>;
  created_at: string;
}

export interface OutreachSequence {
  id: string;
  organization_id: string;
  name: string;
  description?: string;
  status: string;
  trigger_type?: string;
  trigger_config?: Record<string, unknown>;
  created_by?: string;
  steps?: SequenceStep[];
  created_at: string;
}

export interface SequenceStep {
  id: string;
  sequence_id: string;
  position: number;
  type: string;
  subject?: string;
  body?: string;
  template_id?: string;
  delay_days: number;
  delay_hours: number;
  conditions?: Record<string, unknown>;
}

export interface Enrollment {
  id: string;
  organization_id: string;
  sequence_id: string;
  contact_id: string;
  sponsor_id?: string;
  current_step: number;
  status: string;
  paused_at?: string;
  completed_at?: string;
  unsubscribed_at?: string;
  created_at: string;
}

export interface EnrollmentStats {
  total: number;
  active: number;
  completed: number;
  paused: number;
  unsubscribed: number;
}

export interface AuthTokens {
  access_token: string;
  refresh_token: string;
  expires_at: number;
}

export interface PaginatedResponse<T> {
  total: number;
  limit: number;
  offset: number;
  [key: string]: T[] | number;
}
