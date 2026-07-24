import { useAuthStore } from "@/stores/auth";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

type FetchOptions = RequestInit & {
  token?: string;
};

class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  private getToken(): string | undefined {
    if (typeof window === "undefined") return undefined;
    return useAuthStore.getState().tokens?.access_token;
  }

  private async request<T>(endpoint: string, options: FetchOptions = {}): Promise<T> {
    const { token, headers, ...rest } = options;
    const authToken = token || this.getToken();

    const res = await fetch(`${this.baseUrl}${endpoint}`, {
      ...rest,
      headers: {
        "Content-Type": "application/json",
        ...(authToken ? { Authorization: `Bearer ${authToken}` } : {}),
        ...headers,
      },
    });

    if (res.status === 401 && !endpoint.includes("/auth/")) {
      const { tokens, setTokens, logout } = useAuthStore.getState();
      if (tokens?.refresh_token) {
        try {
          const refreshRes = await fetch(`${this.baseUrl}/auth/refresh`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ refresh_token: tokens.refresh_token }),
          });
          if (refreshRes.ok) {
            const data = await refreshRes.json();
            setTokens({ access_token: data.access_token, refresh_token: data.refresh_token, expires_at: 0 });
            const retryRes = await fetch(`${this.baseUrl}${endpoint}`, {
              ...rest,
              headers: {
                "Content-Type": "application/json",
                Authorization: `Bearer ${data.access_token}`,
                ...headers,
              },
            });
            if (!retryRes.ok) {
              const body = await retryRes.json().catch(() => ({ message: retryRes.statusText }));
              throw new ApiError(retryRes.status, body.message || "Request failed");
            }
            return retryRes.json();
          } else {
            logout();
            if (typeof window !== "undefined") window.location.href = "/login";
            throw new ApiError(401, "Session expired");
          }
        } catch (e) {
          if (e instanceof ApiError) throw e;
          logout();
          throw new ApiError(401, "Session expired");
        }
      } else {
        logout();
        if (typeof window !== "undefined") window.location.href = "/login";
        throw new ApiError(401, "Unauthorized");
      }
    }

    if (!res.ok) {
      const body = await res.json().catch(() => ({ message: res.statusText }));
      throw new ApiError(res.status, body.message || "Request failed");
    }

    return res.json();
  }

  get<T>(endpoint: string, token?: string) {
    return this.request<T>(endpoint, { method: "GET", token });
  }

  post<T>(endpoint: string, data: unknown, token?: string) {
    return this.request<T>(endpoint, {
      method: "POST",
      body: JSON.stringify(data),
      token,
    });
  }

  patch<T>(endpoint: string, data: unknown, token?: string) {
    return this.request<T>(endpoint, {
      method: "PATCH",
      body: JSON.stringify(data),
      token,
    });
  }

  delete<T>(endpoint: string, token?: string) {
    return this.request<T>(endpoint, { method: "DELETE", token });
  }

  async upload<T>(endpoint: string, formData: FormData, token?: string): Promise<T> {
    const authToken = token || this.getToken();
    const res = await fetch(`${this.baseUrl}${endpoint}`, {
      method: "POST",
      headers: {
        ...(authToken ? { Authorization: `Bearer ${authToken}` } : {}),
      },
      body: formData,
    });
    if (!res.ok) {
      const body = await res.json().catch(() => ({ message: res.statusText }));
      throw new ApiError(res.status, body.message || "Upload failed");
    }
    return res.json();
  }
}

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
    this.name = "ApiError";
  }
}

export const api = new ApiClient(API_URL);
