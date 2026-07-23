import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

interface BatchUpdateInput {
  ids: string[];
  fields: Record<string, unknown>;
}

interface BatchDeleteInput {
  ids: string[];
}

interface BatchResult {
  updated?: number;
  deleted?: number;
}

export function useBatchUpdate(entity: "sponsors" | "companies" | "contacts") {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: BatchUpdateInput) =>
      api.post<BatchResult>(`/${entity}/batch/update`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [entity] });
    },
  });
}

export function useBatchDelete(entity: "sponsors" | "companies" | "contacts") {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: BatchDeleteInput) =>
      api.post<BatchResult>(`/${entity}/batch/delete`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [entity] });
    },
  });
}
