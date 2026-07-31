import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

export interface Integration {
  id: string;
  organization_id: string;
  provider: string;
  type: string;
  name: string;
  status: string;
  config: Record<string, unknown>;
  last_sync_at?: string;
  last_error?: string;
  webhook_url?: string;
  installed_by?: string;
  created_at: string;
}

export interface SyncRun {
  id: string;
  integration_id: string;
  provider: string;
  trigger: string;
  status: string;
  started_at: string;
  finished_at?: string;
  duration_ms?: number;
  records_synced: number;
  warnings: string[];
  error?: string;
  attempt: number;
}

export interface DashboardEntry {
  integration: Integration;
  recent_runs: SyncRun[];
  failed_runs_24h: number;
  pending_jobs: number;
  synced_counts?: Record<string, number>;
  last_webhook_at?: string;
}

export interface ConnectedApp {
  slug: string;
  actions: string[];
}

export interface SyncedEntity {
  id: string;
  organization_id: string;
  integration_id: string;
  entity_type: string;
  entity_id: string;
  external_system: string;
  external_id: string;
  sync_state: string;
  version: number;
  source: string;
  last_modified_local?: string;
  last_modified_remote?: string;
  last_synced_at?: string;
  conflict_state?: string;
  conflict_resolution?: string;
  last_error?: string;
}

export interface SyncHistoryEntry {
  id: string;
  synced_entity_id: string;
  organization_id: string;
  action: string;
  source: string;
  error?: string;
  created_at: string;
}

export function useIntegrations() {
  return useQuery({
    queryKey: ["integrations"],
    queryFn: () => api.get<{ data: Integration[] }>("/integrations"),
    refetchInterval: (query) => {
      const integrations = query.state.data?.data ?? [];
      return integrations.some((i) => i.status === "syncing" || i.status === "retrying") ? 3000 : false;
    },
  });
}

export function useIntegration(id: string) {
  return useQuery({
    queryKey: ["integrations", id],
    queryFn: () => api.get<{ data: Integration }>(`/integrations/${id}`),
    enabled: !!id,
  });
}

export function useIntegrationDashboard() {
  return useQuery({
    queryKey: ["integrations", "dashboard"],
    queryFn: () => api.get<{ data: DashboardEntry[] }>("/integrations/dashboard"),
    refetchInterval: 5000,
  });
}

export function useSyncConflicts() {
  return useQuery({
    queryKey: ["integrations", "sync", "conflicts"],
    queryFn: () => api.get<{ data: SyncedEntity[] }>("/integrations/sync/conflicts"),
    refetchInterval: 10000,
  });
}

export function useSyncActivity(limit = 50) {
  return useQuery({
    queryKey: ["integrations", "sync", "activity", limit],
    queryFn: () => api.get<{ data: SyncHistoryEntry[] }>(`/integrations/sync/activity?limit=${limit}`),
    refetchInterval: 10000,
  });
}

export function useEnableInboundWebhook() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (provider: string) =>
      api.post<{ webhook_url: string }>(`/integrations/${provider}/webhook-token`, {}),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["integrations"] });
    },
  });
}

export function useZapierApps(enabled: boolean) {
  return useQuery({
    queryKey: ["integrations", "zapier", "apps"],
    queryFn: () => api.get<{ data: ConnectedApp[]; agentic_mode: boolean }>("/integrations/zapier/apps"),
    enabled,
    retry: false,
  });
}

export function useCreateIntegration() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<Integration>) =>
      api.post<{ data: Integration }>("/integrations", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["integrations"] });
    },
  });
}

export function useUpdateIntegration(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<Integration>) =>
      api.patch<{ data: Integration }>(`/integrations/${id}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["integrations"] });
      queryClient.invalidateQueries({ queryKey: ["integrations", id] });
    },
  });
}

export function useDeleteIntegration() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.delete(`/integrations/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["integrations"] });
    },
  });
}

export function useTriggerSync() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.post(`/integrations/${id}/sync`, {}),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["integrations"] });
    },
  });
}

export function useRevokeIntegration() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.post(`/integrations/${id}/revoke`, {}),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["integrations"] });
    },
  });
}

export function useConnectIntegration() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ provider, credentials }: { provider: string; credentials: Record<string, string> }) =>
      api.post<{ data: Integration }>(`/integrations/${provider}/connect`, { credentials }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["integrations"] });
    },
  });
}

export function useRotateCredentials() {
  return useMutation({
    mutationFn: () => api.post<{ data: { checked: number; rotated: number } }>("/integrations/rotate-credentials", {}),
  });
}

export function useDedupeCompanies() {
  return useMutation({
    mutationFn: () => api.post<{ data: { groups_found: number; companies_merged: number } }>("/companies/dedupe", {}),
  });
}
