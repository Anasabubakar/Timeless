import { useMutation, useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useAuthStore } from "@/stores/auth";
import type { User, AuthTokens } from "@/types";

interface LoginPayload {
  email: string;
  password: string;
}

interface RegisterPayload {
  first_name: string;
  last_name: string;
  email: string;
  password: string;
  org_name: string;
  org_slug: string;
  // Becomes the shared secret future teammates use to join this
  // organization (see useJoinOrganization) — only ever sent on the
  // "create a new organization" branch of signup.
  org_password: string;
}

interface JoinPayload {
  first_name: string;
  last_name: string;
  email: string;
  password: string;
  org_slug: string;
  // The existing organization's shared password, not the joiner's own.
  org_password: string;
}

interface AuthResponse {
  user: User;
  tokens: AuthTokens;
}

export interface OrgLookupResult {
  exists: boolean;
  name?: string;
  slug?: string;
}

// useOrgLookup drives the signup form's branch between "create a new
// organization" and "join an existing one" — see LookupOrganization on
// the backend. Disabled until the caller has a non-trivial name to check
// so it doesn't fire on every keystroke from an empty field.
export function useOrgLookup(name: string, enabled: boolean) {
  return useQuery({
    queryKey: ["auth", "org-lookup", name],
    queryFn: () => api.get<OrgLookupResult>(`/auth/organizations/lookup?name=${encodeURIComponent(name)}`),
    enabled: enabled && name.trim().length > 1,
    staleTime: 10_000,
    retry: false,
  });
}

export function useJoinOrganization() {
  const { setAuth } = useAuthStore();
  return useMutation({
    mutationFn: (data: JoinPayload) => api.post<AuthResponse>("/auth/join", data),
    onSuccess: (data) => {
      setAuth(data.user, data.tokens);
    },
  });
}

export function useLogin() {
  const { setAuth } = useAuthStore();
  return useMutation({
    mutationFn: (data: LoginPayload) => api.post<AuthResponse>("/auth/login", data),
    onSuccess: (data) => {
      setAuth(data.user, data.tokens);
    },
  });
}

export function useRegister() {
  const { setAuth } = useAuthStore();
  return useMutation({
    mutationFn: (data: RegisterPayload) => api.post<AuthResponse>("/auth/register", data),
    onSuccess: (data) => {
      setAuth(data.user, data.tokens);
    },
  });
}

export function useLogout() {
  const { logout } = useAuthStore();
  return useMutation({
    mutationFn: () => api.post("/auth/logout", {}),
    onSettled: () => {
      logout();
    },
  });
}

export function useRefreshToken() {
  const { tokens, setTokens } = useAuthStore();
  return useMutation({
    mutationFn: () =>
      api.post<{ access_token: string; refresh_token: string }>("/auth/refresh", {
        refresh_token: tokens?.refresh_token,
      }),
    onSuccess: (data) => {
      setTokens({ access_token: data.access_token, refresh_token: data.refresh_token, expires_at: 0 });
    },
  });
}
