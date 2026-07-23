import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

export interface Webhook {
  id: string;
  organization_id: string;
  url: string;
  events: string[];
  secret: string;
  is_active: boolean;
  last_triggered_at?: string;
  failure_count: number;
  created_at: string;
  updated_at: string;
}

export interface WebhookDelivery {
  id: string;
  organization_id: string;
  webhook_id: string;
  event: string;
  payload: Record<string, unknown>;
  url: string;
  status: string;
  attempts: number;
  max_attempts: number;
  response_code?: number;
  response_body?: string;
  error?: string;
  next_retry_at?: string;
  delivered_at?: string;
  created_at: string;
}

export function useWebhooks() {
  return useQuery({
    queryKey: ["webhooks"],
    queryFn: () => api.get<{ data: Webhook[] }>("/webhooks"),
  });
}

export function useWebhook(id: string) {
  return useQuery({
    queryKey: ["webhooks", id],
    queryFn: () => api.get<{ data: Webhook }>(`/webhooks/${id}`),
    enabled: !!id,
  });
}

export function useWebhookDeliveries(webhookId: string) {
  return useQuery({
    queryKey: ["webhooks", webhookId, "deliveries"],
    queryFn: () => api.get<{ data: WebhookDelivery[] }>(`/webhooks/${webhookId}/deliveries`),
    enabled: !!webhookId,
  });
}

export function useCreateWebhook() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { url: string; events: string[] }) =>
      api.post<{ data: Webhook }>("/webhooks", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["webhooks"] });
    },
  });
}

export function useUpdateWebhook(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { url?: string; events?: string[]; is_active?: boolean }) =>
      api.patch<{ data: Webhook }>(`/webhooks/${id}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["webhooks"] });
      queryClient.invalidateQueries({ queryKey: ["webhooks", id] });
    },
  });
}

export function useDeleteWebhook() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.delete(`/webhooks/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["webhooks"] });
    },
  });
}

export function useRotateWebhookSecret() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      api.post<{ data: Webhook; secret: string }>(`/webhooks/${id}/rotate-secret`, {}),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ["webhooks"] });
      queryClient.invalidateQueries({ queryKey: ["webhooks", id] });
    },
  });
}

export function useTestWebhook() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      api.post<{ message: string }>(`/webhooks/${id}/test`, {}),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ["webhooks", id, "deliveries"] });
    },
  });
}
