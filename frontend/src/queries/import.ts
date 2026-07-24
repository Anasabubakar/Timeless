import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

export interface ImportResult {
  entity: string;
  inserted: number;
  errors: number;
  row_errors: Array<{ row: number; error: string }>;
}

function importFile(entity: string) {
  return (file: File): Promise<ImportResult> => {
    const form = new FormData();
    form.append("file", file);
    return api.upload<ImportResult>(`/import/${entity}`, form);
  };
}

export function useImportCompanies() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: importFile("companies"),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["companies"] }),
  });
}

export function useImportContacts() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: importFile("contacts"),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["contacts"] }),
  });
}

export function useImportSponsors() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: importFile("sponsors"),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["sponsors"] }),
  });
}
