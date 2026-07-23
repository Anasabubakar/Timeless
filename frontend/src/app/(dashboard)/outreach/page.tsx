"use client";

import { Plus, Send, Clock, CheckCircle, XCircle, Mail, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { useSequences } from "@/queries/outreach";
import { useCommunications } from "@/queries/communications";

const statusConfig: Record<string, { icon: typeof Send; color: string; label: string }> = {
  draft: { icon: Clock, color: "text-neutral-500", label: "Draft" },
  sent: { icon: Send, color: "text-blue-500", label: "Sent" },
  opened: { icon: Mail, color: "text-amber-500", label: "Opened" },
  replied: { icon: CheckCircle, color: "text-emerald-500", label: "Replied" },
  bounced: { icon: XCircle, color: "text-red-500", label: "Bounced" },
};

function getCommStatus(comm: { status: string; replied_at?: string; opened_at?: string; bounced_at?: string }) {
  if (comm.replied_at) return "replied";
  if (comm.bounced_at) return "bounced";
  if (comm.opened_at) return "opened";
  return comm.status;
}

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

export default function OutreachPage() {
  const { data: seqData, isLoading: seqLoading } = useSequences();
  const { data: commData, isLoading: commLoading } = useCommunications({ type: "email", limit: 10 });

  const sequences = seqData?.data ?? [];
  const communications = commData?.data ?? [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Outreach</h1>
          <p className="text-sm text-muted-foreground">
            Email sequences and communication tracking
          </p>
        </div>
        <button className="flex h-8 items-center gap-1.5 rounded-lg bg-primary px-3 text-xs font-medium text-primary-foreground hover:bg-primary/90">
          <Plus className="h-3.5 w-3.5" />
          New Sequence
        </button>
      </div>

      {/* Sequences */}
      <div className="space-y-3">
        <h2 className="text-sm font-medium">Active Sequences</h2>
        {seqLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        ) : sequences.length === 0 ? (
          <div className="rounded-xl border border-dashed border-border p-8 text-center">
            <Mail className="mx-auto h-8 w-8 text-muted-foreground/50" />
            <p className="mt-2 text-sm text-muted-foreground">
              No sequences yet. Create one to start automated outreach.
            </p>
          </div>
        ) : (
          <div className="grid grid-cols-3 gap-3">
            {sequences.map((seq) => (
              <div
                key={seq.id}
                className="rounded-xl border border-border bg-card p-4 transition-shadow hover:shadow-sm"
              >
                <div className="flex items-start justify-between">
                  <h3 className="text-sm font-medium">{seq.name}</h3>
                  <span
                    className={cn(
                      "rounded-full px-2 py-0.5 text-[10px] font-medium",
                      seq.status === "active"
                        ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300"
                        : "bg-neutral-100 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400"
                    )}
                  >
                    {seq.status}
                  </span>
                </div>
                <p className="mt-1 text-xs text-muted-foreground">
                  {seq.steps?.length ?? 0} steps
                </p>
                {seq.description && (
                  <p className="mt-1 line-clamp-2 text-xs text-muted-foreground">
                    {seq.description}
                  </p>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Recent Emails */}
      <div className="space-y-3">
        <h2 className="text-sm font-medium">Recent Emails</h2>
        <div className="overflow-hidden rounded-xl border border-border">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border bg-muted/30">
                <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Subject</th>
                <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Direction</th>
                <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Status</th>
                <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Time</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {commLoading ? (
                <tr>
                  <td colSpan={4} className="px-4 py-8 text-center">
                    <Loader2 className="mx-auto h-5 w-5 animate-spin text-muted-foreground" />
                  </td>
                </tr>
              ) : communications.length === 0 ? (
                <tr>
                  <td colSpan={4} className="px-4 py-8 text-center text-sm text-muted-foreground">
                    No emails yet. Communications will appear here once sent.
                  </td>
                </tr>
              ) : (
                communications.map((comm) => {
                  const status = getCommStatus(comm);
                  const cfg = statusConfig[status] ?? statusConfig.sent;
                  return (
                    <tr key={comm.id} className="transition-colors hover:bg-muted/20">
                      <td className="px-4 py-3 text-sm">{comm.subject ?? "(No subject)"}</td>
                      <td className="px-4 py-3 text-xs text-muted-foreground capitalize">{comm.direction}</td>
                      <td className="px-4 py-3">
                        <span className={cn("flex items-center gap-1 text-xs font-medium", cfg.color)}>
                          <cfg.icon className="h-3 w-3" />
                          {cfg.label}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-xs text-muted-foreground">
                        {formatRelativeTime(comm.sent_at ?? comm.created_at)}
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
