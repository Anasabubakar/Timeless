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

export interface UpdateOrganizationInput extends Partial<Organization> {
  // Required by the backend whenever name/slug/password is being
  // changed (OrganizationService.UpdateSecure) — Owner-only, and this is
  // how the frontend proves the caller actually knows the org's current
  // password rather than relying solely on their session being valid.
  current_password?: string;
  // New organization password, if rotating it. Never present in the
  // Organization type returned by the API (write-only).
  password?: string;
}

export function useUpdateOrganization() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateOrganizationInput) =>
      api.patch<{ organization: Organization }>("/organizations/current", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["organization"] });
    },
  });
}

export function useTransferOwnership() {
  return useMutation({
    mutationFn: (data: { new_owner_id: string; current_password: string }) =>
      api.post<{ message: string }>("/organizations/current/transfer-ownership", data),
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

export interface DeleteAccountInput {
  password: string;
  // Required (and must be true) when the caller is their organization's
  // only member — see AuthService.DeleteAccount. Ignored otherwise.
  confirm_org_deletion?: boolean;
}

// useDeleteAccount logs the user out locally on success — the backend
// has already revoked every session, so there's nothing left to keep
// around client-side either.
export function useDeleteAccount() {
  const { logout } = useAuthStore();
  return useMutation({
    mutationFn: (data: DeleteAccountInput) =>
      api.post<{ message: string }>("/profile/delete", data),
    onSuccess: () => {
      logout();
    },
  });
}
