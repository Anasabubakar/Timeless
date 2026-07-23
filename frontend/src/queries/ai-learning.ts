import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

export interface AgentOutcome {
  id: string;
  organization_id: string;
  agent_type: string;
  conversation_id?: string;
  query: string;
  response: string;
  outcome: "success" | "failure" | "neutral" | "positive_feedback" | "negative_feedback";
  score: number;
  feedback?: string;
  metadata: Record<string, unknown>;
  created_at: string;
}

export interface LearnedPreference {
  id: string;
  organization_id: string;
  agent_type: string;
  category: string;
  preference: string;
  confidence: number;
  learned_from: number;
  created_at: string;
  updated_at: string;
}

export function useAgentOutcomes(agentType: string) {
  return useQuery<{ data: AgentOutcome[] }>({
    queryKey: ["ai", "outcomes", agentType],
    queryFn: () => api.get(`/ai/outcomes?agent_type=${agentType}`),
    enabled: !!agentType,
    staleTime: 60_000,
  });
}

export function useAgentPreferences(agentType: string) {
  return useQuery<{ data: LearnedPreference[] }>({
    queryKey: ["ai", "preferences", agentType],
    queryFn: () => api.get(`/ai/preferences?agent_type=${agentType}`),
    enabled: !!agentType,
    staleTime: 60_000,
  });
}

export function useRecordOutcome() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: {
      agent_type: string;
      query: string;
      response?: string;
      outcome: string;
      score: number;
      feedback?: string;
      conversation_id?: string;
    }) => api.post<AgentOutcome>("/ai/outcomes", data),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ["ai", "outcomes", vars.agent_type] });
    },
  });
}

export function useSubmitFeedback() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: {
      agent_type: string;
      feedback: string;
      is_positive: boolean;
    }) => api.post("/ai/feedback", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai"] });
    },
  });
}

export function useStorePreference() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: {
      agent_type: string;
      category: string;
      preference: string;
      confidence?: number;
    }) => api.post<LearnedPreference>("/ai/preferences", data),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ["ai", "preferences", vars.agent_type] });
    },
  });
}
