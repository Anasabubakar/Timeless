import { useMutation } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useAuthStore } from "@/stores/auth";
import type { User, AuthTokens } from "@/types";

interface AcceptInvitationPayload {
  token: string;
  password: string;
  first_name: string;
  last_name: string;
}

interface AcceptInvitationResponse {
  user: User;
  tokens: AuthTokens;
}

// useAcceptInvitation redeems an invitation token — the invited person's
// account doesn't exist until this call succeeds, at which point it also
// logs them straight in (matches Register/useJoinOrganization).
export function useAcceptInvitation() {
  const { setAuth } = useAuthStore();
  return useMutation({
    mutationFn: (data: AcceptInvitationPayload) =>
      api.post<AcceptInvitationResponse>("/invitations/accept", data),
    onSuccess: (data) => {
      setAuth(data.user, data.tokens);
    },
  });
}
