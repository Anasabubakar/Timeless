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

export function useIntegrations() {
  return useQuery({
    queryKey: ["integrations"],
    queryFn: () => api.get<{ data: Integration[] }>("/integrations"),
  });
}

export function useIntegration(id: string) {
  return useQuery({
    queryKey: ["integrations", id],
    queryFn: () => api.get<{ data: Integration }>(`/integrations/${id}`),
    enabled: !!id,
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
