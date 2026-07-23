import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { OutreachSequence, Enrollment, EnrollmentStats } from "@/types";

export function useSequences(status?: string) {
  return useQuery({
    queryKey: ["sequences", status],
    queryFn: () => {
      const params = status ? `?status=${status}` : "";
      return api.get<{ data: OutreachSequence[]; total: number }>(
        `/sequences${params}`
      );
    },
  });
}

export function useSequence(id: string) {
  return useQuery({
    queryKey: ["sequences", id],
    queryFn: () => api.get<{ data: OutreachSequence }>(`/sequences/${id}`),
    enabled: !!id,
  });
}

export function useCreateSequence() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<OutreachSequence>) =>
      api.post<{ data: OutreachSequence }>("/sequences", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sequences"] });
    },
  });
}

export function useUpdateSequence(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<OutreachSequence>) =>
      api.patch<{ data: OutreachSequence }>(`/sequences/${id}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sequences"] });
      queryClient.invalidateQueries({ queryKey: ["sequences", id] });
    },
  });
}

export function useDeleteSequence() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.delete(`/sequences/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sequences"] });
    },
  });
}

export function useEnroll(sequenceId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { contact_id: string; sponsor_id?: string }) =>
      api.post<{ data: Enrollment }>(`/sequences/${sequenceId}/enroll`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sequences", sequenceId] });
      queryClient.invalidateQueries({ queryKey: ["enrollments"] });
    },
  });
}
