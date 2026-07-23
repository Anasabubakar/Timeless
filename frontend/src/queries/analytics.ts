import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

export interface DashboardStats {
  total_sponsors: number;
  active_campaigns: number;
  pipeline_value: number;
  conversion_rate: number;
  total_revenue: number;
  total_contacts: number;
  total_companies: number;
  closed_won: number;
  closed_lost: number;
  avg_deal_size: number;
  avg_deal_velocity: number;
  win_rate: number;
}

export interface PipelineStage {
  stage: string;
  count: number;
  value: number;
}

export interface ActivityItem {
  id: string;
  entity_type: string;
  entity_id: string;
  type: string;
  subject: string;
  description?: string;
  metadata: Record<string, unknown>;
  user?: { id: string; first_name: string; last_name: string };
  created_at: string;
}

export interface TimeSeriesPoint {
  date: string;
  value: number;
}

export interface FunnelStage {
  stage: string;
  count: number;
  value: number;
  percentage: number;
  avg_days_in_stage: number;
}

export interface VelocityPoint {
  month: string;
  avg_days: number;
  deals_won: number;
  total_value: number;
  avg_deal_size: number;
}

export function useDashboardStats() {
  return useQuery({
    queryKey: ["analytics", "dashboard"],
    queryFn: () => api.get<{ data: DashboardStats }>("/analytics/dashboard"),
  });
}

export function usePipelineAnalytics() {
  return useQuery({
    queryKey: ["analytics", "pipeline"],
    queryFn: () => api.get<{ data: PipelineStage[] }>("/analytics/pipeline"),
  });
}

export function useRecentActivity() {
  return useQuery({
    queryKey: ["analytics", "activity"],
    queryFn: () => api.get<{ data: ActivityItem[] }>("/analytics/activity"),
  });
}

export function useTimeSeries(metric: string, period = 30) {
  return useQuery({
    queryKey: ["analytics", "timeseries", metric, period],
    queryFn: () =>
      api.get<{ data: TimeSeriesPoint[]; metric: string; period: number }>(
        `/analytics/timeseries?metric=${encodeURIComponent(metric)}&period=${period}`
      ),
  });
}

export function usePipelineFunnel() {
  return useQuery({
    queryKey: ["analytics", "funnel"],
    queryFn: () =>
      api.get<{ data: FunnelStage[]; total: number }>("/analytics/funnel"),
  });
}

export function useDealVelocity() {
  return useQuery({
    queryKey: ["analytics", "velocity"],
    queryFn: () => api.get<{ data: VelocityPoint[] }>("/analytics/velocity"),
  });
}
