"use client";

import { useState } from "react";
import { Plus, Send, Clock, CheckCircle, XCircle, Mail, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { useSequences, useCreateSequence } from "@/queries/outreach";
import { useCommunications } from "@/queries/communications";
import { motion } from "motion/react";
import { useNewParam } from "@/hooks/use-new-param";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";

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
  const [showCreate, setShowCreate] = useState(false);
  useNewParam(() => setShowCreate(true));
  const { data: seqData, isLoading: seqLoading } = useSequences();
  const { data: commData, isLoading: commLoading } = useCommunications({ type: "email", limit: 10 });

  const sequences = seqData?.data ?? [];
  const communications = commData?.data ?? [];

  return (
    <motion.div className="space-y-6" initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.3 }}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Outreach</h1>
          <p className="text-sm text-muted-foreground">
            Email sequences and communication tracking
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="flex h-8 items-center gap-1.5 rounded-lg bg-primary px-3 text-xs font-medium text-primary-foreground hover:bg-primary/90"
        >
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
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
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
        {commLoading ? (
          <div className="flex justify-center rounded-xl border border-border py-8">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        ) : communications.length === 0 ? (
          <div className="rounded-xl border border-border py-8 text-center text-sm text-muted-foreground">
            No emails yet. Communications will appear here once sent.
          </div>
        ) : (
          <>
            {/* Tablet/desktop: table */}
            <div className="hidden overflow-hidden rounded-xl border border-border sm:block">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-border bg-muted/30">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Subject</th>
                    <th className="hidden px-4 py-2.5 text-left text-xs font-medium text-muted-foreground lg:table-cell">Direction</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Status</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Time</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {communications.map((comm) => {
                    const status = getCommStatus(comm);
                    const cfg = statusConfig[status] ?? statusConfig.sent;
                    return (
                      <tr key={comm.id} className="transition-colors hover:bg-muted/20">
                        <td className="px-4 py-3 text-sm">{comm.subject ?? "(No subject)"}</td>
                        <td className="hidden px-4 py-3 text-xs text-muted-foreground capitalize lg:table-cell">{comm.direction}</td>
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
                  })}
                </tbody>
              </table>
            </div>

            {/* Phone: stacked cards */}
            <div className="flex flex-col gap-2 sm:hidden">
              {communications.map((comm) => {
                const status = getCommStatus(comm);
                const cfg = statusConfig[status] ?? statusConfig.sent;
                return (
                  <div key={comm.id} className="rounded-xl border border-border p-3.5">
                    <div className="flex items-start justify-between gap-2">
                      <p className="min-w-0 truncate text-sm font-medium">{comm.subject ?? "(No subject)"}</p>
                      <span className={cn("flex shrink-0 items-center gap-1 text-xs font-medium", cfg.color)}>
                        <cfg.icon className="h-3 w-3" />
                        {cfg.label}
                      </span>
                    </div>
                    <div className="mt-2 flex items-center gap-3 text-xs text-muted-foreground">
                      <span className="capitalize">{comm.direction}</span>
                      <span>{formatRelativeTime(comm.sent_at ?? comm.created_at)}</span>
                    </div>
                  </div>
                );
              })}
            </div>
          </>
        )}
      </div>

      <CreateSequenceDialog open={showCreate} onOpenChange={setShowCreate} />
    </motion.div>
  );
}

function CreateSequenceDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (v: boolean) => void }) {
  const createSequence = useCreateSequence();
  const [form, setForm] = useState({ name: "", description: "" });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    createSequence.mutate(
      { name: form.name, description: form.description || undefined, status: "draft" },
      {
        onSuccess: () => {
          onOpenChange(false);
          setForm({ name: "", description: "" });
        },
      }
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>New Sequence</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4 mt-4">
          <input
            placeholder="Sequence name"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            className="h-9 w-full rounded-[10px] border border-neutral-200 bg-background px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10 dark:border-neutral-700"
            required
          />
          <textarea
            placeholder="Description (optional)"
            value={form.description}
            onChange={(e) => setForm({ ...form, description: e.target.value })}
            className="min-h-[80px] w-full rounded-[10px] border border-neutral-200 bg-background px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10 dark:border-neutral-700"
          />
          <DialogFooter>
            <button
              type="button"
              onClick={() => onOpenChange(false)}
              className="h-8 rounded-lg border border-neutral-200 px-3 text-xs font-medium hover:bg-neutral-50 dark:border-neutral-700 dark:hover:bg-neutral-800"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={createSequence.isPending}
              className="h-8 rounded-lg bg-neutral-900 px-3 text-xs font-medium text-white hover:bg-neutral-800 disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900 dark:hover:bg-neutral-200"
            >
              {createSequence.isPending ? "Creating..." : "Create"}
            </button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
