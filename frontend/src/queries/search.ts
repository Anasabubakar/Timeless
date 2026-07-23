import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";

export interface SearchResult {
  id: string;
  type: "sponsor" | "company" | "contact" | "campaign";
  title: string;
  subtitle?: string;
  rank: number;
}

interface SearchResponse {
  data: SearchResult[];
  total: number;
  query: string;
}

export function useSearch(query: string, type?: string) {
  return useQuery({
    queryKey: ["search", query, type],
    queryFn: () => {
      const params = new URLSearchParams({ q: query });
      if (type) params.set("type", type);
      return api.get<SearchResponse>(`/search?${params}`);
    },
    enabled: query.length >= 2,
    staleTime: 30_000,
  });
}
