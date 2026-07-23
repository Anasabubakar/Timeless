import { useMutation } from "@tanstack/react-query";
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
}

interface AuthResponse {
  user: User;
  tokens: AuthTokens;
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
      setTokens({ access_token: data.access_token, refresh_token: data.refresh_token });
    },
  });
}
