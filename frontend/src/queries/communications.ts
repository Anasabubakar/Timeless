import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

export interface Communication {
  id: string;
  organization_id: string;
  sponsor_id?: string;
  contact_id?: string;
  type: string;
  direction: string;
  subject?: string;
  body?: string;
  status: string;
  sent_at?: string;
  opened_at?: string;
  clicked_at?: string;
  replied_at?: string;
  bounced_at?: string;
  channel?: string;
  external_id?: string;
  template_id?: string;
  metadata: Record<string, unknown>;
  sent_by?: string;
  created_at: string;
}

export interface CommunicationStats {
  sent: number;
  opened: number;
  clicked: number;
  replied: number;
  bounced: number;
}

export function useCommunications(params?: {
  status?: string;
  type?: string;
  direction?: string;
  sponsor_id?: string;
  limit?: number;
  offset?: number;
}) {
  const searchParams = new URLSearchParams();
  if (params?.status) searchParams.set("status", params.status);
  if (params?.type) searchParams.set("type", params.type);
  if (params?.direction) searchParams.set("direction", params.direction);
  if (params?.sponsor_id) searchParams.set("sponsor_id", params.sponsor_id);
  if (params?.limit) searchParams.set("limit", String(params.limit));
  if (params?.offset) searchParams.set("offset", String(params.offset));

  const qs = searchParams.toString();
  return useQuery({
    queryKey: ["communications", params],
    queryFn: () =>
      api.get<{ data: Communication[]; total: number }>(
        `/communications${qs ? `?${qs}` : ""}`
      ),
  });
}

export function useCommunication(id: string) {
  return useQuery({
    queryKey: ["communications", id],
    queryFn: () => api.get<{ data: Communication }>(`/communications/${id}`),
    enabled: !!id,
  });
}

export function useCommunicationStats() {
  return useQuery({
    queryKey: ["communications", "stats"],
    queryFn: () =>
      api.get<{ data: CommunicationStats }>("/communications/stats"),
  });
}

export function useCreateCommunication() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<Communication>) =>
      api.post<{ data: Communication }>("/communications", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["communications"] });
    },
  });
}

export function useUpdateCommunication(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<Communication>) =>
      api.patch<{ data: Communication }>(`/communications/${id}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["communications"] });
    },
  });
}

export function useDeleteCommunication() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.delete(`/communications/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["communications"] });
    },
  });
}
