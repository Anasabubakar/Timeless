import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { Activity, PaginatedResponse } from "@/types";

export function useActivities(params?: {
  entity_type?: string;
  entity_id?: string;
  type?: string;
  limit?: number;
  offset?: number;
}) {
  const searchParams = new URLSearchParams();
  if (params?.entity_type) searchParams.set("entity_type", params.entity_type);
  if (params?.entity_id) searchParams.set("entity_id", params.entity_id);
  if (params?.type) searchParams.set("type", params.type);
  if (params?.limit) searchParams.set("limit", String(params.limit));
  if (params?.offset) searchParams.set("offset", String(params.offset));

  return useQuery({
    queryKey: ["activities", params],
    queryFn: () => api.get<PaginatedResponse<Activity>>(`/activities?${searchParams}`),
  });
}

export function useCreateActivity() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<Activity>) => api.post<Activity>("/activities", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["activities"] });
    },
  });
}
