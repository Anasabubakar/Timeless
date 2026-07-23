import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { Proposal } from "@/types";

export function useProposals(sponsorId?: string, status?: string) {
  return useQuery({
    queryKey: ["proposals", sponsorId, status],
    queryFn: () => {
      const params = new URLSearchParams();
      if (sponsorId) params.set("sponsor_id", sponsorId);
      if (status) params.set("status", status);
      const qs = params.toString();
      return api.get<{ data: Proposal[]; total: number }>(
        `/proposals${qs ? `?${qs}` : ""}`
      );
    },
  });
}

export function useProposal(id: string) {
  return useQuery({
    queryKey: ["proposals", id],
    queryFn: () => api.get<{ data: Proposal }>(`/proposals/${id}`),
    enabled: !!id,
  });
}

export function useCreateProposal() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<Proposal>) =>
      api.post<{ data: Proposal }>("/proposals", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["proposals"] });
    },
  });
}

export function useUpdateProposal(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<Proposal>) =>
      api.patch<{ data: Proposal }>(`/proposals/${id}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["proposals"] });
      queryClient.invalidateQueries({ queryKey: ["proposals", id] });
    },
  });
}

export function useDeleteProposal() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.delete(`/proposals/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["proposals"] });
    },
  });
}

export interface GenerateProposalInput {
  sponsor_id: string;
  tone?: string;
  package_tier?: string;
  custom_notes?: string;
}

export function useGenerateProposal() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: GenerateProposalInput) =>
      api.post<{ data: Proposal }>("/proposals/generate", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["proposals"] });
    },
  });
}
