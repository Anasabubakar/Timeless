"use client";

import { useState } from "react";
import { Plus, Zap, Play, Pause } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { useAutomations, useToggleAutomation, type Automation } from "@/queries/automations";

function formatLastRun(dateStr?: string): string {
  if (!dateStr) return "Never";
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "Just now";
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

export default function AutomationsPage() {
  const { data, isLoading } = useAutomations();
  const toggle = useToggleAutomation();

  const automations: Automation[] = data?.data ?? [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Automations</h1>
          <p className="text-sm text-muted-foreground">
            Automated workflows powered by triggers and AI agents
          </p>
        </div>
        <Button>
          <Plus className="h-4 w-4 mr-2" />
          New Automation
        </Button>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="animate-pulse rounded-xl border border-border bg-card px-5 py-4">
              <div className="h-4 bg-neutral-200 rounded w-1/3 mb-2" />
              <div className="h-3 bg-neutral-100 rounded w-2/3" />
            </div>
          ))}
        </div>
      ) : automations.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-16 text-center">
            <Zap className="h-12 w-12 text-muted-foreground/40 mb-4" />
            <h3 className="text-lg font-medium">No automations yet</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Create your first automation to streamline your workflow.
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-2">
          {automations.map((automation) => (
            <div
              key={automation.id}
              className="flex items-center gap-4 rounded-xl border border-border bg-card px-5 py-4 transition-shadow hover:shadow-sm"
            >
              <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-amber-50">
                <Zap className="h-4 w-4 text-amber-600" />
              </div>

              <div className="min-w-0 flex-1">
                <h3 className="text-sm font-medium">{automation.name}</h3>
                <div className="mt-0.5 flex items-center gap-2 text-xs text-muted-foreground">
                  <span className="font-medium text-foreground/70">Trigger:</span>
                  <span>{automation.trigger_type}</span>
                  {automation.description && (
                    <>
                      <span className="text-muted-foreground/40">—</span>
                      <span>{automation.description}</span>
                    </>
                  )}
                </div>
              </div>

              <div className="flex items-center gap-4">
                <div className="text-right">
                  <p className="text-xs font-medium">{automation.run_count} runs</p>
                  <p className="text-[10px] text-muted-foreground">Last: {formatLastRun(automation.last_run_at)}</p>
                </div>
                <button
                  onClick={() => toggle.mutate({ id: automation.id, is_active: !automation.is_active })}
                  className={cn(
                    "flex items-center gap-1.5 rounded-full px-2.5 py-1 transition-colors",
                    automation.is_active
                      ? "bg-emerald-50 text-emerald-700 hover:bg-emerald-100"
                      : "bg-neutral-100 text-neutral-600 hover:bg-neutral-200"
                  )}
                >
                  {automation.is_active ? (
                    <Play className="h-3 w-3" />
                  ) : (
                    <Pause className="h-3 w-3" />
                  )}
                  <span className="text-[10px] font-medium">
                    {automation.is_active ? "active" : "paused"}
                  </span>
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
