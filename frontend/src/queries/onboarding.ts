import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useAuthStore } from "@/stores/auth";

export interface OnboardingState {
  id: string;
  organization_id: string;
  user_id: string;
  step: string;
  payload: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export function useOnboardingState() {
  return useQuery({
    queryKey: ["onboarding", "state"],
    queryFn: () => api.get<{ data: OnboardingState }>("/onboarding/state"),
  });
}

export function useSaveOnboardingState() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { step: string; payload: Record<string, unknown> }) =>
      api.patch<{ data: OnboardingState }>("/onboarding/state", data),
    onSuccess: (res) => {
      queryClient.setQueryData(["onboarding", "state"], { data: res.data });
    },
  });
}

export function useCompleteOnboarding() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => api.post<{ data: { onboarding_completed: boolean } }>("/onboarding/complete", {}),
    onSuccess: (res) => {
      const { user, tokens, setAuth } = useAuthStore.getState();
      if (user && tokens) {
        setAuth({ ...user, onboarding_completed: res.data.onboarding_completed }, tokens);
      }
      queryClient.invalidateQueries({ queryKey: ["onboarding", "state"] });
    },
  });
}
