import { useMutation, useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';

interface AIAgent {
  type: string;
  name: string;
  description: string;
}

interface AIQueryInput {
  query: string;
  agent_type?: string;
  campaign_id?: string;
  sponsor_id?: string;
  company_id?: string;
}

interface AIQueryResponse {
  agent: string;
  response: string;
  data?: Record<string, unknown>;
  actions?: Array<{ type: string; target: string; params: Record<string, unknown> }>;
  tokens_used: { prompt_tokens: number; completion_tokens: number; total_tokens: number };
}

export function useAIAgents() {
  return useQuery({
    queryKey: ['ai-agents'],
    queryFn: () => api.get<{ agents: AIAgent[] }>('/ai/agents'),
  });
}

export function useAIQuery() {
  return useMutation({
    mutationFn: (input: AIQueryInput) =>
      api.post<AIQueryResponse>('/ai/query', input),
  });
}
