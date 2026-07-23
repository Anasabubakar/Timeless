"use client";

import { useState } from "react";
import {
  TrendingUp,
  DollarSign,
  Target,
  Activity,
  BarChart3,
  LineChart,
  ArrowUpRight,
  ArrowDownRight,
  Download,
  Loader2,
  Clock,
  CheckCircle2,
} from "lucide-react";
import { AreaChart, Area, BarChart, Bar, LineChart as ReLine, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Cell } from "recharts";
import { cn, formatCurrency, formatNumber } from "@/lib/utils";
import {
  useDashboardStats,
  usePipelineAnalytics,
  usePipelineFunnel,
  useTimeSeries,
  useDealVelocity,
  useRecentActivity,
} from "@/queries/analytics";
import { useAuthStore } from "@/stores/auth";

const stageLabels: Record<string, string> = {
  prospect: "Prospect",
  contacted: "Contacted",
  qualified: "Qualified",
  proposal: "Proposal",
  negotiation: "Negotiation",
  closed_won: "Closed Won",
  closed_lost: "Closed Lost",
};

const metricOptions = [
  { value: "sponsors", label: "Sponsors" },
  { value: "revenue", label: "Revenue" },
  { value: "pipeline_value", label: "Pipeline Value" },
  { value: "contacts", label: "Contacts" },
  { value: "deals_won", label: "Deals Won" },
  { value: "companies", label: "Companies" },
];

const stageColors: Record<string, string> = {
  prospect: "#6366f1",
  contacted: "#3b82f6",
  qualified: "#8b5cf6",
  proposal: "#f59e0b",
  negotiation: "#10b981",
  closed_won: "#10b981",
  closed_lost: "#f43f5e",
};

function downloadCSV(url: string, token: string | undefined, filename: string) {
  const authUrl = token ? `${url}?token=${token}` : url;
  fetch(authUrl, {
    headers: { Authorization: token ? `Bearer ${token}` : "" },
  })
    .then((res) => res.blob())
    .then((blob) => {
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = filename;
      a.click();
      URL.revokeObjectURL(a.href);
    })
    .catch(() => alert("Download failed. Please try again."));
}

export default function AnalyticsPage() {
  const { tokens } = useAuthStore();
  const accessToken = tokens?.access_token;
  const [metric, setMetric] = useState("revenue");
  const [period, setPeriod] = useState(30);

  const { data: statsData, isLoading: statsLoading } = useDashboardStats();
  const { data: pipelineData, isLoading: pipelineLoading } = usePipelineAnalytics();
  const { data: funnelData, isLoading: funnelLoading } = usePipelineFunnel();
  const { data: tsData, isLoading: tsLoading } = useTimeSeries(metric, period);
  const { data: velocityData, isLoading: velocityLoading } = useDealVelocity();
  const { data: activityData, isLoading: activityLoading } = useRecentActivity();

  const stats = statsData?.data;
  const pipeline = pipelineData?.data ?? [];
  const funnel = funnelData?.data ?? [];
  const timeSeries = tsData?.data ?? [];
  const velocityPoints = velocityData?.data ?? [];
  const activities = activityData?.data ?? [];

  if (statsLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  const maxPipelineValue = Math.max(...pipeline.map((s) => s.value), 1);
  const maxFunnelCount = Math.max(...funnel.map((f) => f.count), 1);

  const kpiCards = [
    { name: "Total Revenue", value: formatCurrency(stats?.total_revenue ?? 0), change: "+12.3%", positive: true },
    { name: "Pipeline Value", value: formatCurrency(stats?.pipeline_value ?? 0), change: "+5.2%", positive: true },
    { name: "Conversion Rate", value: `${(stats?.conversion_rate ?? 0).toFixed(1)}%`, change: "+2.1%", positive: true },
    { name: "Win Rate", value: `${(stats?.win_rate ?? 0).toFixed(1)}%`, change: "+0.8%", positive: stats?.win_rate > 0 },
    { name: "Avg Deal Size", value: formatCurrency(stats?.avg_deal_size ?? 0), change: "+3.4%", positive: true },
    { name: "Avg Velocity", value: `${stats?.avg_deal_velocity ?? 0}d`, change: "-2d", positive: false },
    { name: "Deals Won", value: String(stats?.closed_won ?? 0), change: "+3", positive: true },
    { name: "Active Campaigns", value: String(stats?.active_campaigns ?? 0), change: "+1", positive: true },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Analytics</h1>
          <p className="text-sm text-muted-foreground">Pipeline performance and sponsorship metrics</p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => downloadCSV(`${process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1"}/analytics/export/sponsors`, accessToken, `sponsors_export_${new Date().toISOString().slice(0, 10)}.csv`)}
            className="inline-flex items-center gap-1.5 rounded-lg border border-border bg-card px-3 py-2 text-xs font-medium hover:bg-muted transition-colors"
          >
            <Download className="h-3.5 w-3.5" />
            Export Sponsors
          </button>
          <button
            onClick={() => downloadCSV(`${process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1"}/analytics/export/campaigns`, accessToken, `campaigns_export_${new Date().toISOString().slice(0, 10)}.csv`)}
            className="inline-flex items-center gap-1.5 rounded-lg border border-border bg-card px-3 py-2 text-xs font-medium hover:bg-muted transition-colors"
          >
            <Download className="h-3.5 w-3.5" />
            Export Campaigns
          </button>
        </div>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-4 gap-4">
        {kpiCards.map((k) => (
          <div key={k.name} className="rounded-xl border border-border bg-card p-5">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">{k.name}</span>
              <span className={cn("text-[10px] font-medium flex items-center gap-0.5", k.positive ? "text-emerald-500" : "text-red-400")}>{k.positive ? <ArrowUpRight className="h-3 w-3" /> : <ArrowDownRight className="h-3 w-3" />}{k.change}</span>
            </div>
            <div className="mt-2">
              <span className="text-2xl font-semibold tracking-tight">{k.value}</span>
            </div>
          </div>
        ))}
      </div>

      {/* Charts Row */}
      <div className="grid grid-cols-2 gap-4">
        {/* Time Series */}
        <div className="col-span-2 rounded-xl border border-border bg-card p-5">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-sm font-medium">Trend Over Time</h2>
              <p className="text-xs text-muted-foreground">{metricOptions.find((m) => m.value === metric)?.label} over the last {period} days</p>
            </div>
            <div className="flex gap-2 items-center">
              <select
                value={metric}
                onChange={(e) => setMetric(e.target.value)}
                className="h-8 rounded-md border border-border bg-background px-2 text-xs outline-none"
              >
                {metricOptions.map((m) => (
                  <option key={m.value} value={m.value}>{m.label}</option>
                ))}
              </select>
              <select
                value={period}
                onChange={(e) => setPeriod(Number(e.target.value))}
                className="h-8 rounded-md border border-border bg-background px-2 text-xs outline-none"
              >
                <option value={7}>7d</option>
                <option value={30}>30d</option>
                <option value={90}>90d</option>
              </select>
            </div>
          </div>
          <div className="h-72">
            {tsLoading ? (
              <div className="flex h-full items-center justify-center"><Loader2 className="h-5 w-5 animate-spin text-muted-foreground" /></div>
            ) : (
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={timeSeries}>
                  <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
                  <XAxis dataKey="date" tick={{ fontSize: 11 }} interval="preserveStartEnd" />
                  <YAxis tick={{ fontSize: 11 }} />
                  <Tooltip
                    contentStyle={{ borderRadius: 8, border: "1px solid hsl(var(--border))", background: "hsl(var(--card))", color: "hsl(var(--foreground))", fontSize: 12 }}
                  />
                  <Area type="monotone" dataKey="value" stroke="#10b981" fill="#10b981" fillOpacity={0.15} strokeWidth={2} />
                </AreaChart>
              </ResponsiveContainer>
            )}
          </div>
        </div>
      </div>

      {/* Funnel + Velocity */}
      <div className="grid grid-cols-2 gap-4">
        <div className="rounded-xl border border-border bg-card p-5">
          <h2 className="text-sm font-medium">Pipeline Funnel</h2>
          <p className="text-xs text-muted-foreground mb-4">Sponsor progression through stages</p>
          <div className="h-60">
            {funnelLoading ? (
              <div className="flex h-full items-center justify-center"><Loader2 className="h-5 w-5 animate-spin text-muted-foreground" /></div>
            ) : funnel.length === 0 ? (
              <p className="text-sm text-muted-foreground">No pipeline data yet.</p>
            ) : (
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={funnel} layout="vertical" barCategoryGap="15%">
                  <CartesianGrid horizontal={false} stroke="hsl(var(--border))" strokeDasharray="3 3" />
                  <XAxis type="number" tick={{ fontSize: 11 }} />
                  <YAxis type="category" dataKey="stage" tick={{ fontSize: 10 }} width={100} tickFormatter={(v: string) => stageLabels[v] ?? v} />
                  <Tooltip
                    contentStyle={{ borderRadius: 8, border: "1px solid hsl(var(--border))", background: "hsl(var(--card))", fontSize: 12 }}
                    formatter={(value: number) => [`${value}`, "Count"]}
                  />
                  <Bar dataKey="count" radius={[0, 4, 4, 0]} barSize={28}>
                    {funnel.map((entry) => (
                      <Cell key={entry.stage} fill={stageColors[entry.stage] || "#6366f1"} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            )}
          </div>
        </div>

        <div className="rounded-xl border border-border bg-card p-5">
          <h2 className="text-sm font-medium">Deal Velocity</h2>
          <p className="text-xs text-muted-foreground mb-4">Average days to close per month (last 12 months)</p>
          <div className="h-60">
            {velocityLoading ? (
              <div className="flex h-full items-center justify-center"><Loader2 className="h-5 w-5 animate-spin text-muted-foreground" /></div>
            ) : velocityPoints.length === 0 ? (
              <p className="text-sm text-muted-foreground">No velocity data yet.</p>
            ) : (
              <ResponsiveContainer width="100%" height="100%">
                <ReLine data={velocityPoints}>
                  <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
                  <XAxis dataKey="month" tick={{ fontSize: 11 }} />
                  <YAxis tick={{ fontSize: 11 }} />
                  <Tooltip
                    contentStyle={{ borderRadius: 8, border: "1px solid hsl(var(--border))", background: "hsl(var(--card))", fontSize: 12 }}
                  />
                  <Line type="monotone" dataKey="avg_days" stroke="#f59e0b" strokeWidth={2} dot={{ r: 3 }} activeDot={{ r: 5 }} />
                </ReLine>
              </ResponsiveContainer>
            )}
          </div>
        </div>
      </div>

      {/* Pipeline by Stage */}
      <div className="rounded-xl border border-border bg-card p-5">
        <h2 className="text-sm font-medium">Pipeline by Stage</h2>
        <p className="mt-0.5 text-xs text-muted-foreground">Sponsors and deal value at each stage</p>
        {pipelineLoading ? (
          <div className="mt-6 flex h-40 items-center justify-center">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        ) : pipeline.length === 0 ? (
          <p className="mt-6 text-sm text-muted-foreground">No pipeline data yet. Add sponsors to see stage breakdown.</p>
        ) : (
          <div className="mt-6 space-y-4">
            {pipeline.map((stage) => (
              <div key={stage.stage} className="space-y-1">
                <div className="flex items-center justify-between text-sm">
                  <span>{stageLabels[stage.stage] ?? stage.stage}</span>
                  <span className="text-muted-foreground">
                    {stage.count} sponsors &middot; {formatCurrency(stage.value)}
                  </span>
                </div>
                <div className="h-3 overflow-hidden rounded-md bg-muted">
                  <div
                    className={cn(
                      "h-full rounded-md bg-primary transition-all",
                      stage.stage === "closed_won" && "bg-emerald-500",
                      stage.stage === "closed_lost" && "bg-red-400"
                    )}
                    style={{ width: `${Math.max((stage.value / maxPipelineValue) * 100, 4)}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Funnel Details + Recent Activity */}
      <div className="grid grid-cols-2 gap-4">
        <div className="rounded-xl border border-border bg-card p-5">
          <h2 className="text-sm font-medium">Pipeline Funnel Details</h2>
          <p className="text-xs text-muted-foreground mb-4">Count, value, and average days per stage</p>
          {funnelLoading ? (
            <div className="flex h-40 items-center justify-center"><Loader2 className="h-5 w-5 animate-spin" /></div>
          ) : (
            <div className="space-y-2">
              {funnel.map((f) => (
                <div key={f.stage} className="flex items-center justify-between rounded-lg border border-border p-3 hover:bg-muted/30">
                  <div>
                    <p className="text-xs font-medium">{stageLabels[f.stage] ?? f.stage}</p>
                    <p className="text-[10px] text-muted-foreground">{f.count} sponsors · {formatCurrency(f.value)}</p>
                  </div>
                  <div className="text-right">
                    <span className="text-xs font-semibold">{f.percentage}%</span>
                    <p className="text-[10px] text-muted-foreground">{f.avg_days_in_stage}d avg</p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="rounded-xl border border-border bg-card p-5">
          <h2 className="text-sm font-medium">Recent Activity</h2>
          <p className="text-xs text-muted-foreground">Latest updates across your organization</p>
          {activityLoading ? (
            <div className="mt-4 flex h-40 items-center justify-center"><Loader2 className="h-5 w-5 animate-spin" /></div>
          ) : activities.length === 0 ? (
            <p className="mt-4 text-sm text-muted-foreground">No recent activity.</p>
          ) : (
            <div className="mt-4 space-y-3 max-h-[360px] overflow-y-auto">
              {activities.slice(0, 10).map((a) => (
                <div key={a.id} className="flex items-start gap-3 rounded-lg border border-border p-3 hover:bg-muted/20 transition-colors">
                  <div className="mt-0.5 h-8 w-8 shrink-0 rounded-full bg-primary/10 flex items-center justify-center">
                    <Activity className="h-4 w-4 text-primary" />
                  </div>
                  <div className="min-w-0">
                    <p className="text-xs font-medium truncate">{a.subject}</p>
                    <p className="text-[10px] text-muted-foreground truncate">{a.entity_type} · {a.created_at ? new Date(a.created_at).toLocaleDateString() : ""}</p>
                    {a.user && (
                      <p className="text-[10px] text-muted-foreground/70">by {a.user.first_name} {a.user.last_name}</p>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Summary cards */}
      <div className="grid grid-cols-3 gap-4">
        <div className="rounded-xl border border-border bg-card p-5">
          <p className="text-sm text-muted-foreground">Active Campaigns</p>
          <p className="mt-1 text-3xl font-semibold">{stats?.active_campaigns ?? 0}</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-5">
          <p className="text-sm text-muted-foreground">Deals Won</p>
          <p className="mt-1 text-3xl font-semibold">{stats?.closed_won ?? 0}</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-5">
          <p className="text-sm text-muted-foreground">Companies Tracked</p>
          <p className="mt-1 text-3xl font-semibold">{stats?.total_companies ?? 0}</p>
        </div>
      </div>
    </div>
  );
}
