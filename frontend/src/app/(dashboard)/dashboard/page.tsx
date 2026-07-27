"use client";

import {
  DollarSign,
  Building2,
  Target,
  TrendingUp,
  ArrowUpRight,
  Loader2,
} from "lucide-react";
import { motion } from "motion/react";
import { cn, formatCurrency, formatNumber } from "@/lib/utils";
import { useDashboardStats, usePipelineAnalytics, useRecentActivity } from "@/queries/analytics";
import { useIntegrations } from "@/queries/integrations";

const stageColors: Record<string, string> = {
  prospect: "bg-neutral-200",
  contacted: "bg-blue-200",
  qualified: "bg-indigo-200",
  proposal: "bg-violet-200",
  negotiation: "bg-amber-200",
  closed_won: "bg-emerald-200",
  closed_lost: "bg-red-200",
};

const stageLabels: Record<string, string> = {
  prospect: "Prospect",
  contacted: "Contacted",
  qualified: "Qualified",
  proposal: "Proposal",
  negotiation: "Negotiation",
  closed_won: "Closed Won",
  closed_lost: "Closed Lost",
};

function formatRelativeTime(dateStr: string) {
  const diff = Date.now() - new Date(dateStr).getTime();
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

export default function DashboardPage() {
  const { data: statsData, isLoading: statsLoading } = useDashboardStats();
  const { data: pipelineData, isLoading: pipelineLoading } = usePipelineAnalytics();
  const { data: activityData, isLoading: activityLoading } = useRecentActivity();
  const { data: integrationsData } = useIntegrations();

  const syncingIntegrations = (integrationsData?.data ?? []).filter((i) => i.status === "syncing");

  const stats = statsData?.data;
  const pipeline = pipelineData?.data ?? [];
  const activities = activityData?.data ?? [];
  const maxCount = Math.max(...pipeline.map((s) => s.count), 1);

  const kpis = [
    {
      name: "Total Pipeline",
      value: stats?.pipeline_value ?? 0,
      format: "currency" as const,
      icon: DollarSign,
    },
    {
      name: "Active Sponsors",
      value: stats?.total_sponsors ?? 0,
      format: "number" as const,
      icon: Target,
    },
    {
      name: "Companies Tracked",
      value: stats?.total_companies ?? 0,
      format: "number" as const,
      icon: Building2,
    },
    {
      name: "Win Rate",
      value: stats?.conversion_rate ?? 0,
      format: "percent" as const,
      icon: TrendingUp,
    },
  ];

  if (statsLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return (
    <motion.div className="space-y-6" initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.3 }}>
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
        <p className="text-sm text-muted-foreground">
          Your sponsorship operations at a glance
        </p>
      </div>

      {syncingIntegrations.length > 0 && (
        <motion.div
          className="flex items-center gap-3 rounded-xl border border-border bg-card px-4 py-3"
          initial={{ opacity: 0, y: -8 }}
          animate={{ opacity: 1, y: 0 }}
        >
          <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
          <span className="text-sm">
            Syncing your workspace... ({syncingIntegrations.map((i) => i.provider).join(", ")})
          </span>
        </motion.div>
      )}

      {/* KPI Cards */}
      <div className="grid grid-cols-2 gap-3 sm:gap-4 lg:grid-cols-4">
        {kpis.map((stat, i) => (
          <motion.div
            key={stat.name}
            className="rounded-xl border border-border bg-card p-4 sm:p-5"
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: i * 0.06 }}
          >
            <div className="flex items-center justify-between gap-2">
              <span className="truncate text-sm text-muted-foreground">{stat.name}</span>
              <stat.icon className="h-4 w-4 shrink-0 text-muted-foreground" />
            </div>
            <div className="mt-2 flex items-baseline gap-2">
              <span className="text-xl font-semibold tracking-tight sm:text-2xl">
                {stat.format === "currency"
                  ? formatCurrency(stat.value)
                  : stat.format === "percent"
                  ? `${stat.value.toFixed(1)}%`
                  : formatNumber(stat.value)}
              </span>
              {stat.value > 0 && (
                <span className="flex items-center text-xs font-medium text-emerald-600">
                  <ArrowUpRight className="mr-0.5 h-3 w-3" />
                </span>
              )}
            </div>
          </motion.div>
        ))}
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
        {/* Pipeline Overview */}
        <motion.div
          className="rounded-xl border border-border bg-card p-5 lg:col-span-2"
          initial={{ opacity: 0, y: 16 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.25 }}
        >
          <h2 className="text-sm font-medium">Pipeline Overview</h2>
          {pipelineLoading ? (
            <div className="mt-4 flex h-40 items-center justify-center">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          ) : pipeline.length === 0 ? (
            <p className="mt-4 text-sm text-muted-foreground">
              No sponsors in pipeline yet. Add sponsors to campaigns to see pipeline data.
            </p>
          ) : (
            <div className="mt-4 space-y-3">
              {pipeline.map((stage, i) => (
                <motion.div
                  key={stage.stage}
                  className="flex items-center gap-3"
                  initial={{ opacity: 0, x: -8 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ duration: 0.25, delay: 0.3 + i * 0.05 }}
                >
                  <span className="w-24 text-sm text-muted-foreground">
                    {stageLabels[stage.stage] ?? stage.stage}
                  </span>
                  <div className="flex-1">
                    <div className="h-7 overflow-hidden rounded-md bg-muted">
                      <div
                        className={cn(
                          "h-full rounded-md transition-all",
                          stageColors[stage.stage] ?? "bg-neutral-200"
                        )}
                        style={{
                          width: `${Math.max((stage.count / maxCount) * 100, 8)}%`,
                        }}
                      />
                    </div>
                  </div>
                  <span className="w-8 text-right text-sm font-medium">
                    {stage.count}
                  </span>
                </motion.div>
              ))}
            </div>
          )}
        </motion.div>

        {/* Recent Activity */}
        <motion.div
          className="rounded-xl border border-border bg-card p-5"
          initial={{ opacity: 0, y: 16 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.3 }}
        >
          <h2 className="text-sm font-medium">Recent Activity</h2>
          {activityLoading ? (
            <div className="mt-4 flex h-40 items-center justify-center">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          ) : activities.length === 0 ? (
            <p className="mt-4 text-sm text-muted-foreground">
              No activity yet. Actions will appear here as you work.
            </p>
          ) : (
            <div className="mt-4 space-y-4">
              {activities.slice(0, 8).map((item, i) => (
                <motion.div
                  key={item.id}
                  className="flex flex-col gap-0.5"
                  initial={{ opacity: 0, x: 8 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ duration: 0.2, delay: 0.35 + i * 0.04 }}
                >
                  <span className="text-sm">{item.subject || item.type}</span>
                  {item.description && (
                    <span className="text-xs text-muted-foreground">
                      {item.description}
                    </span>
                  )}
                  <span className="text-[11px] text-muted-foreground/60">
                    {formatRelativeTime(item.created_at)}
                  </span>
                </motion.div>
              ))}
            </div>
          )}
        </motion.div>
      </div>
    </motion.div>
  );
}
