import { useAuthStore } from "@/stores/auth";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

type FetchOptions = RequestInit & {
  token?: string;
};

class ApiClient {
  private baseUrl: string;
  // Shared in-flight refresh so N concurrent 401s trigger exactly one
  // /auth/refresh call instead of N. The backend blacklists a refresh
  // token the instant it's used (single-use rotation) — without this,
  // every concurrent request past the first got a 401 "token has been
  // revoked" on its own refresh attempt and force-logged the user out,
  // even though the session was perfectly valid a moment earlier.
  private refreshPromise: Promise<{ access_token: string; refresh_token: string }> | null = null;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  private getToken(): string | undefined {
    if (typeof window === "undefined") return undefined;
    return useAuthStore.getState().tokens?.access_token;
  }

  private refreshTokens(): Promise<{ access_token: string; refresh_token: string }> {
    if (this.refreshPromise) return this.refreshPromise;

    const { refresh_token } = useAuthStore.getState().tokens ?? {};
    if (!refresh_token) {
      return Promise.reject(new ApiError(401, "Unauthorized"));
    }

    this.refreshPromise = (async () => {
      const refreshRes = await fetch(`${this.baseUrl}/auth/refresh`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token }),
      });
      if (!refreshRes.ok) {
        throw new ApiError(401, "Session expired");
      }
      const data = await refreshRes.json();
      const fresh = { access_token: data.access_token, refresh_token: data.refresh_token };
      useAuthStore.getState().setTokens({ ...fresh, expires_at: 0 });
      return fresh;
    })();

    // Clear the shared promise once settled (success or failure) so a
    // later, genuinely new expiry can trigger its own refresh cycle
    // instead of forever reusing this one's outcome.
    this.refreshPromise.finally(() => {
      this.refreshPromise = null;
    });

    return this.refreshPromise;
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
      try {
        const fresh = await this.refreshTokens();
        const retryRes = await fetch(`${this.baseUrl}${endpoint}`, {
          ...rest,
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${fresh.access_token}`,
            ...headers,
          },
        });
        if (!retryRes.ok) {
          const body = await retryRes.json().catch(() => ({ message: retryRes.statusText }));
          throw new ApiError(retryRes.status, body.message || "Request failed");
        }
        return retryRes.json();
      } catch (e) {
        useAuthStore.getState().logout();
        if (typeof window !== "undefined") window.location.href = "/login";
        throw e instanceof ApiError ? e : new ApiError(401, "Session expired");
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

  put<T>(endpoint: string, data: unknown, token?: string) {
    return this.request<T>(endpoint, {
      method: "PUT",
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
