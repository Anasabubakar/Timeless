"use client";

import {
  DollarSign,
  Target,
  TrendingUp,
  ArrowUpRight,
  ArrowDownRight,
  ArrowRight,
  Loader2,
  Sparkles,
  Activity,
  Mail,
  FileText,
  UserPlus,
  Megaphone,
  Zap,
  BarChart3,
  Clock,
  CalendarDays,
} from "lucide-react";
import { cn, formatCurrency, formatNumber } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth";
import { useDashboardStats, usePipelineAnalytics, useRecentActivity } from "@/queries/analytics";
import { formatDistanceToNow } from "date-fns";
import Link from "next/link";

const stageColors: Record<string, string> = {
  prospect: "bg-neutral-300",
  contacted: "bg-blue-400",
  qualified: "bg-indigo-400",
  proposal: "bg-violet-400",
  negotiation: "bg-amber-400",
  closed_won: "bg-emerald-400",
  closed_lost: "bg-red-400",
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

function getGreeting() {
  const hour = new Date().getHours();
  if (hour < 12) return "Good morning";
  if (hour < 17) return "Good afternoon";
  return "Good evening";
}

function ActivityIcon({ type }: { type: string }) {
  const lower = type.toLowerCase();
  if (lower.includes("email") || lower.includes("mail")) return <Mail className="h-4 w-4 text-blue-500" />;
  if (lower.includes("meeting") || lower.includes("calendar")) return <CalendarDays className="h-4 w-4 text-violet-500" />;
  if (lower.includes("sponsor") || lower.includes("campaign")) return <Megaphone className="h-4 w-4 text-amber-500" />;
  if (lower.includes("contact") || lower.includes("user")) return <UserPlus className="h-4 w-4 text-emerald-500" />;
  if (lower.includes("deal") || lower.includes("proposal")) return <FileText className="h-4 w-4 text-indigo-500" />;
  if (lower.includes("task") || lower.includes("activity")) return <Activity className="h-4 w-4 text-rose-500" />;
  return <Zap className="h-4 w-4 text-neutral-400" />;
}

export default function DashboardPage() {
  const { user } = useAuthStore();
  const { data: statsData, isLoading: statsLoading } = useDashboardStats();
  const { data: pipelineData, isLoading: pipelineLoading } = usePipelineAnalytics();
  const { data: activityData, isLoading: activityLoading } = useRecentActivity();

  const stats = statsData?.data;
  const pipeline = pipelineData?.data ?? [];
  const activities = activityData?.data ?? [];
  const maxCount = Math.max(...pipeline.map((s) => s.count), 1);

  const greeting = getGreeting();
  const firstName = user?.first_name || "there";
  const todayFormatted = new Intl.DateTimeFormat("en-US", {
    weekday: "long",
    year: "numeric",
    month: "long",
    day: "numeric",
  }).format(new Date());

  const kpis = [
    {
      name: "Pipeline Value",
      value: stats?.pipeline_value ?? 0,
      format: "currency" as const,
      icon: DollarSign,
      trendUp: true,
      trendLabel: "+12.4%",
    },
    {
      name: "Total Revenue",
      value: stats?.total_revenue ?? 0,
      format: "currency" as const,
      icon: TrendingUp,
      trendUp: true,
      trendLabel: "+8.2%",
    },
    {
      name: "Conversion Rate",
      value: stats?.conversion_rate ?? 0,
      format: "percent" as const,
      icon: Target,
      trendUp: stats?.conversion_rate ? stats.conversion_rate > 30 : false,
      trendLabel: stats?.conversion_rate && stats.conversion_rate > 30 ? "+3.1%" : "-0.4%",
    },
    {
      name: "Active Campaigns",
      value: stats?.active_campaigns ?? 0,
      format: "number" as const,
      icon: Sparkles,
      trendUp: true,
      trendLabel: "+2",
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
    <div className="space-y-6">
      {/* Welcome Header */}
      <header className="rounded-2xl border border-border bg-gradient-to-br from-card to-neutral-900/20 p-6 shadow-sm">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="text-3xl font-extrabold tracking-tight text-foreground">
              {greeting}, {firstName}
            </h1>
            <p className="mt-1.5 text-sm text-muted-foreground">{todayFormatted}</p>
          </div>
          <span className="shrink-0 rounded-full bg-emerald-500/10 px-3 py-1 text-xs font-bold text-emerald-600 ring-1 ring-emerald-500/20">
            Live Dashboard
          </span>
        </div>
      </header>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {kpis.map((stat) => {
          const IconComp = stat.icon;
          return (
            <div
              key={stat.name}
              className="group relative overflow-hidden rounded-xl border border-border bg-card p-5 shadow-sm transition-all hover:-translate-y-0.5 hover:shadow-md"
            >
              <div className="absolute inset-y-0 right-0 w-1 bg-gradient-to-b from-transparent via-emerald-300/60 to-transparent opacity-0 transition-opacity duration-300 group-hover:opacity-100" />
              <div className="flex items-center justify-between">
                <span className="text-xs font-bold uppercase tracking-wider text-muted-foreground">
                  {stat.name}
                </span>
                <div className="rounded-lg bg-muted/60 p-1.5">
                  <IconComp className="h-4 w-4 text-foreground/70" />
                </div>
              </div>
              <div className="mt-3 flex items-baseline gap-3">
                <span className="text-3xl font-extrabold tracking-tight text-foreground">
                  {stat.format === "currency"
                    ? formatCurrency(stat.value)
                    : stat.format === "percent"
                    ? `${stat.value.toFixed(1)}%`
                    : formatNumber(stat.value)}
                </span>
                <span
                  className={cn(
                    "inline-flex items-center gap-0.5 rounded-full px-1.5 py-0.5 text-[11px] font-bold shadow-sm",
                    stat.trendUp
                      ? "bg-emerald-50 text-emerald-600 dark:bg-emerald-950/40 dark:text-emerald-400"
                      : "bg-red-50 text-red-500 dark:bg-red-950/40 dark:text-red-400"
                  )}
                >
                  {stat.trendUp ? (
                    <ArrowUpRight className="h-3 w-3" />
                  ) : (
                    <ArrowDownRight className="h-3 w-3" />
                  )}
                  {stat.trendLabel}
                </span>
              </div>
            </div>
          );
        })}
      </div>

      {/* Two-Column Layout */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        {/* LEFT COLUMN */}
        <div className="space-y-4 lg:col-span-2">
          {/* Recent Activity */}
          <section className="rounded-xl border border-border bg-card p-5 shadow-sm">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className="rounded-lg bg-violet-500/10 p-1.5">
                  <Activity className="h-4 w-4 text-violet-500" />
                </div>
                <h2 className="text-sm font-extrabold tracking-tight">Recent Activity</h2>
              </div>
              <span className="rounded-full bg-neutral-100 px-2 py-0.5 text-[10px] font-bold text-muted-foreground dark:bg-neutral-800">Last 10</span>
            </div>

            {activityLoading ? (
              <div className="mt-5 flex h-40 items-center justify-center">
                <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
              </div>
            ) : activities.length === 0 ? (
              <div className="mt-5 rounded-xl border border-dashed border-border bg-muted/30 p-8 text-center">
                <p className="text-sm font-medium text-muted-foreground">No activity yet.</p>
                <p className="text-xs text-muted-foreground/60 mt-1">Actions will appear here as you work.</p>
              </div>
            ) : (
              <div className="mt-4 space-y-0 divide-y divide-border/40">
                {activities.slice(0, 10).map((item) => (
                  <Link
                    key={item.id}
                    href="#"
                    className="group flex items-start gap-3 rounded-lg px-2 py-3 transition-colors hover:bg-neutral-50 dark:hover:bg-neutral-900/40"
                  >
                    <div className="mt-0.5 shrink-0 rounded-full bg-neutral-100 p-2 dark:bg-neutral-800">
                      <ActivityIcon type={item.type} />
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="text-sm font-semibold text-foreground truncate">
                        {item.subject || item.type}
                      </p>
                      {item.description && (
                        <p className="text-xs text-muted-foreground truncate">{item.description}</p>
                      )}
                      <div className="mt-1 flex items-center gap-2">
                        <span className="text-[11px] text-muted-foreground/70">
                          {formatDistanceToNow(new Date(item.created_at), { addSuffix: true })}
                        </span>
                        {item.user && (
                          <span className="text-[10px] text-muted-foreground/50">by {item.user.first_name}</span>
                        )}
                      </div>
                    </div>
                    <ArrowRight className="h-4 w-4 text-muted-foreground/20 group-hover:text-muted-foreground/60 transition-colors shrink-0 self-center" />
                  </Link>
                ))}
              </div>
            )}
          </section>

          {/* Pipeline Summary Mini Bars */}
          <section className="rounded-xl border border-border bg-card p-5 shadow-sm">
            <div className="flex items-center gap-2 mb-5">
              <div className="rounded-lg bg-indigo-500/10 p-1.5">
                <BarChart3 className="h-4 w-4 text-indigo-500" />
              </div>
              <h2 className="text-sm font-extrabold tracking-tight">Pipeline Summary</h2>
            </div>

            {pipelineLoading ? (
              <div className="flex h-32 items-center justify-center">
                <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
              </div>
            ) : pipeline.length === 0 ? (
              <div className="rounded-xl border border-dashed border-border bg-muted/30 p-8 text-center">
                <p className="text-sm font-medium text-muted-foreground">No pipeline stages yet.</p>
                <p className="text-xs text-muted-foreground/60 mt-1">Add sponsors to campaigns to see data.</p>
              </div>
            ) : (
              <div className="space-y-5">
                {pipeline.map((stage) => {
                  const label = stageLabels[stage.stage] ?? stage.stage;
                  const percent = Math.max((stage.count / maxCount) * 100, 6);
                  return (
                    <div key={stage.stage}>
                      <div className="flex items-center justify-between mb-1.5">
                        <span className="text-xs font-bold text-foreground">{label}</span>
                        <span className="text-xs font-extrabold tabular-nums text-muted-foreground">{stage.count}</span>
                      </div>
                      <div className="h-2.5 overflow-hidden rounded-full bg-neutral-100 dark:bg-neutral-800">
                        <div
                          className={cn("h-full rounded-full transition-all duration-700 ease-out", stageColors[stage.stage] ?? "bg-neutral-400")}
                          style={{ width: `${percent}%` }}
                        />
                      </div>
                      <div className="mt-1 flex items-center justify-between">
                        <span className="text-[10px] text-muted-foreground/50">{percent.toFixed(0)}% of pipeline</span>
                        <span className="text-[10px] font-bold text-muted-foreground tabular-nums">{formatCurrency(stage.value || 0)}</span>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </section>
        </div>

        {/* RIGHT COLUMN */}
        <div className="space-y-4 lg:col-span-1">
          {/* Quick Actions */}
          <section className="rounded-xl border border-border bg-card p-5 shadow-sm">
            <div className="flex items-center gap-2 mb-5">
              <div className="rounded-lg bg-amber-500/10 p-1.5">
                <Zap className="h-4 w-4 text-amber-500" />
              </div>
              <h2 className="text-sm font-extrabold tracking-tight">Quick Actions</h2>
            </div>
            <div className="grid grid-cols-2 gap-2.5">
              <Link
                href="/sponsors"
                className="group flex flex-col items-center gap-2.5 rounded-xl border border-border bg-neutral-50/80 p-4 text-center transition-all hover:border-amber-300 hover:bg-amber-50 hover:shadow-md dark:bg-neutral-900/40 dark:hover:bg-amber-950/20 dark:hover:border-amber-800/50"
              >
                <UsersIcon />
                <span className="text-xs font-bold text-foreground">Sponsors</span>
              </Link>
              <Link
                href="/campaigns"
                className="group flex flex-col items-center gap-2.5 rounded-xl border border-border bg-neutral-50/80 p-4 text-center transition-all hover:border-violet-300 hover:bg-violet-50 hover:shadow-md dark:bg-neutral-900/40 dark:hover:bg-violet-950/20 dark:hover:border-violet-800/50"
              >
                <CampaignIcon />
                <span className="text-xs font-bold text-foreground">Campaigns</span>
              </Link>
              <Link
                href="/ai-agents"
                className="group flex flex-col items-center gap-2.5 rounded-xl border border-border bg-neutral-50/80 p-4 text-center transition-all hover:border-emerald-300 hover:bg-emerald-50 hover:shadow-md dark:bg-neutral-900/40 dark:hover:bg-emerald-950/20 dark:hover:border-emerald-800/50"
              >
                <Sparkles className="h-6 w-6 text-emerald-500 group-hover:scale-110 transition-transform" />
                <span className="text-xs font-bold text-foreground">AI Agents</span>
              </Link>
              <Link
                href="/analytics"
                className="group flex flex-col items-center gap-2.5 rounded-xl border border-border bg-neutral-50/80 p-4 text-center transition-all hover:border-blue-300 hover:bg-blue-50 hover:shadow-md dark:bg-neutral-900/40 dark:hover:bg-blue-950/20 dark:hover:border-blue-800/50"
              >
                <TrendingUp className="h-6 w-6 text-blue-500 group-hover:scale-110 transition-transform" />
                <span className="text-xs font-bold text-foreground">Analytics</span>
              </Link>
            </div>
          </section>

          {/* Upcoming Placeholder */}
          <section className="rounded-xl border border-border bg-card p-5 shadow-sm">
            <div className="flex items-center gap-2 mb-5">
              <div className="rounded-lg bg-rose-500/10 p-1.5">
                <Clock className="h-4 w-4 text-rose-500" />
              </div>
              <h2 className="text-sm font-extrabold tracking-tight">Upcoming</h2>
            </div>
            <div className="space-y-3">
              <div className="rounded-xl border border-dashed border-border bg-muted/30 p-6 text-center">
                <p className="text-sm font-medium text-muted-foreground">No upcoming events scheduled.</p>
                <p className="text-xs text-muted-foreground/60 mt-1">Meetings and deadlines will appear here.</p>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}

function UsersIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-amber-500">
      <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
      <circle cx="9" cy="7" r="4" />
      <path d="M22 21v-2a4 4 0 0 0-3-3.87" />
      <path d="M16 3.13a4 4 0 0 1 0 7.75" />
    </svg>
  );
}

function CampaignIcon() {
  return <Megaphone className="h-6 w-6 text-violet-500 group-hover:scale-110 transition-transform" />;
}
