import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useAuthStore } from "@/stores/auth";
import type { Organization, User } from "@/types";

export function useCurrentOrganization() {
  return useQuery({
    queryKey: ["organization", "current"],
    queryFn: () =>
      api.get<{ organization: Organization }>("/organizations/current"),
  });
}

export function useUpdateOrganization() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<Organization>) =>
      api.patch<{ organization: Organization }>("/organizations/current", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["organization"] });
    },
  });
}

export function useUpdateProfile() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<User>) =>
      api.patch<{ user: User }>("/profile", data),
    onSuccess: (response) => {
      const currentTokens = useAuthStore.getState().tokens;
      if (currentTokens) {
        useAuthStore.getState().setAuth(response.user, currentTokens);
      }
      queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
    },
  });
}

export function useChangePassword() {
  return useMutation({
    mutationFn: (data: { current_password: string; new_password: string }) =>
      api.post<{ message: string }>("/profile/password", data),
  });
}
