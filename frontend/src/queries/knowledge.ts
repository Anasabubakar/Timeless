import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

export interface KnowledgeNode {
  id: string;
  organization_id: string;
  node_type: string;
  entity_id?: string;
  name: string;
  properties: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface KnowledgeEdge {
  id: string;
  organization_id: string;
  source_id: string;
  target_id: string;
  edge_type: string;
  weight: number;
  properties: Record<string, unknown>;
  created_at: string;
}

export interface MemoryEntry {
  id: string;
  organization_id: string;
  agent_type: string;
  content: string;
  summary: string;
  importance: number;
  access_count: number;
  last_accessed_at?: string;
  created_at: string;
  updated_at: string;
}

interface SearchResponse<T> {
  data: T[];
  count: number;
  query: string;
}

interface NeighborsResponse {
  nodes: KnowledgeNode[];
  edges: KnowledgeEdge[];
}

export function useSemanticSearch(query: string, type?: string, limit = 10) {
  const params = new URLSearchParams({ q: query, limit: String(limit) });
  if (type) params.set("type", type);

  return useQuery<SearchResponse<KnowledgeNode>>({
    queryKey: ["knowledge", "search", query, type, limit],
    queryFn: () => api.get(`/knowledge/search?${params}`),
    enabled: query.length >= 2,
    staleTime: 30_000,
  });
}

export function useMemorySearch(query: string, limit = 10) {
  const params = new URLSearchParams({ q: query, limit: String(limit) });

  return useQuery<SearchResponse<MemoryEntry>>({
    queryKey: ["knowledge", "memories", query, limit],
    queryFn: () => api.get(`/knowledge/memories?${params}`),
    enabled: query.length >= 2,
    staleTime: 30_000,
  });
}

export function useNodeNeighbors(nodeId: string, depth = 1) {
  return useQuery<NeighborsResponse>({
    queryKey: ["knowledge", "neighbors", nodeId, depth],
    queryFn: () => api.get(`/knowledge/nodes/${nodeId}/neighbors?depth=${depth}`),
    enabled: !!nodeId,
  });
}

export function useAddNode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { node_type: string; name: string; entity_id?: string; properties?: Record<string, unknown> }) =>
      api.post<KnowledgeNode>("/knowledge/nodes", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["knowledge"] });
    },
  });
}

export function useAddEdge() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { source_id: string; target_id: string; edge_type: string; weight?: number; properties?: Record<string, unknown> }) =>
      api.post<KnowledgeEdge>("/knowledge/edges", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["knowledge"] });
    },
  });
}

export function useStoreMemory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { agent_type: string; content: string; metadata?: Record<string, unknown> }) =>
      api.post<MemoryEntry>("/knowledge/memories", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["knowledge", "memories"] });
    },
  });
}
