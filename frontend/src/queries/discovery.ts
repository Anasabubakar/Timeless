import { useMutation } from "@tanstack/react-query";
import { api } from "@/lib/api";

export interface DiscoveredProject {
  name: string;
  confidence: number;
  explanation: string;
  sources: string[];
  document_count: number;
  recent_activity: string;
}

export interface RecommendedGoal {
  title: string;
  description: string;
  automation_summary: string;
}

export interface PlannedAutomation {
  title: string;
  description: string;
  trigger_type: string;
  trigger_config: Record<string, unknown>;
  action_type: string;
}

export function useRunDiscovery() {
  return useMutation({
    mutationFn: () => api.post<{ data: DiscoveredProject[] }>("/onboarding/discovery/run", {}),
  });
}

export function useSelectProjects() {
  return useMutation({
    mutationFn: (data: { project_names: string[]; new_project?: string }) =>
      api.post<{ data: { id: string; name: string }[] }>("/onboarding/discovery/select", data),
  });
}

export function useRecommendGoals() {
  return useMutation({
    mutationFn: (data: { project_names: string[] }) =>
      api.post<{ data: RecommendedGoal[] }>("/onboarding/goals/recommend", data),
  });
}

export function usePlanAutomation() {
  return useMutation({
    mutationFn: (data: { goal: string }) =>
      api.post<{ data: PlannedAutomation[] }>("/onboarding/goals/plan", data),
  });
}

export function useApproveAutomation() {
  return useMutation({
    mutationFn: (data: { steps: PlannedAutomation[] }) =>
      api.post<{ data: unknown[] }>("/onboarding/goals/approve", data),
  });
}
