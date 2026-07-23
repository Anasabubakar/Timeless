import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { Sponsor, PaginatedResponse } from "@/types";

export function useSponsors(params?: { campaign_id?: string; stage?: string; page?: number; limit?: number }) {
  const { campaign_id, stage, page = 1, limit = 50 } = params || {};
  const searchParams = new URLSearchParams({ page: String(page), limit: String(limit) });
  if (campaign_id) searchParams.set("campaign_id", campaign_id);
  if (stage) searchParams.set("stage", stage);

  return useQuery({
    queryKey: ["sponsors", { campaign_id, stage, page, limit }],
    queryFn: () => api.get<PaginatedResponse<Sponsor>>(`/sponsors?${searchParams}`),
  });
}

export function useSponsor(id: string) {
  return useQuery({
    queryKey: ["sponsors", id],
    queryFn: () => api.get<Sponsor>(`/sponsors/${id}`),
    enabled: !!id,
  });
}

export function useCreateSponsor() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<Sponsor>) => api.post<Sponsor>("/sponsors", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sponsors"] });
    },
  });
}

export function useUpdateSponsor(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<Sponsor>) => api.patch<Sponsor>(`/sponsors/${id}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sponsors"] });
      queryClient.invalidateQueries({ queryKey: ["sponsors", id] });
    },
  });
}

export function useUpdateSponsorStage(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (stage: string) => api.patch<Sponsor>(`/sponsors/${id}/stage`, { stage }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sponsors"] });
    },
  });
}

export function useDeleteSponsor() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.delete(`/sponsors/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sponsors"] });
    },
  });
}
