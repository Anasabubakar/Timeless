import { useMutation } from "@tanstack/react-query";
import { api } from "@/lib/api";

export interface UploadResult {
  key: string;
  url: string;
  size: number;
  content_type: string;
}

interface UploadOptions {
  folder?: "uploads" | "avatars" | "logos" | "proposals" | "documents" | "imports";
}

export function useFileUpload() {
  return useMutation({
    mutationFn: async ({ file, folder = "uploads" }: { file: File } & UploadOptions) => {
      const formData = new FormData();
      formData.append("file", file);
      formData.append("folder", folder);
      return api.upload<{ data: UploadResult }>("/files/upload", formData);
    },
  });
}

export function useFileDelete() {
  return useMutation({
    mutationFn: (key: string) =>
      api.delete(`/files?key=${encodeURIComponent(key)}`),
  });
}
