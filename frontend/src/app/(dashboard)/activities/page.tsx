"use client";

import { useState } from "react";
import { Filter, Mail, Phone, Calendar, MessageSquare, FileText, Zap } from "lucide-react";
import { useActivities } from "@/queries/activities";
import type { Activity } from "@/types";
import { motion } from "motion/react";

const TYPE_ICONS: Record<string, any> = {
  email: Mail,
  call: Phone,
  meeting: Calendar,
  note: MessageSquare,
  proposal: FileText,
  system: Zap,
};

const TYPE_COLORS: Record<string, string> = {
  email: "bg-blue-50 text-blue-600",
  call: "bg-green-50 text-green-600",
  meeting: "bg-violet-50 text-violet-600",
  note: "bg-amber-50 text-amber-600",
  proposal: "bg-rose-50 text-rose-600",
  system: "bg-neutral-100 text-neutral-600",
};

export default function ActivitiesPage() {
  const [typeFilter, setTypeFilter] = useState<string | undefined>();
  const { data, isLoading } = useActivities({ type: typeFilter, limit: 50 });

  const activities: Activity[] = (data as any)?.activities || [];

  return (
    <motion.div className="space-y-6" initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.3 }}>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Activity</h1>
          <p className="text-sm text-muted-foreground">
            Recent activity across your organization
          </p>
        </div>
      </div>

      <div className="flex items-center gap-2">
        <button
          onClick={() => setTypeFilter(undefined)}
          className={`h-7 rounded-full px-3 text-xs font-medium transition-colors ${
            !typeFilter ? "bg-neutral-900 text-white" : "bg-muted text-muted-foreground hover:bg-muted/80"
          }`}
        >
          All
        </button>
        {["email", "call", "meeting", "note", "proposal"].map((type) => (
          <button
            key={type}
            onClick={() => setTypeFilter(typeFilter === type ? undefined : type)}
            className={`h-7 rounded-full px-3 text-xs font-medium capitalize transition-colors ${
              typeFilter === type
                ? "bg-neutral-900 text-white"
                : "bg-muted text-muted-foreground hover:bg-muted/80"
            }`}
          >
            {type}
          </button>
        ))}
      </div>

      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="h-16 animate-pulse rounded-lg border border-border bg-muted/30" />
          ))}
        </div>
      ) : activities.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-12">
          <p className="text-sm text-muted-foreground">No activity recorded yet</p>
          <p className="mt-1 text-xs text-muted-foreground/60">
            Activities will appear as you interact with sponsors
          </p>
        </div>
      ) : (
        <div className="space-y-1">
          {activities.map((activity) => {
            const Icon = TYPE_ICONS[activity.type] || Zap;
            const colorClass = TYPE_COLORS[activity.type] || TYPE_COLORS.system;

            return (
              <div
                key={activity.id}
                className="flex items-start gap-3 rounded-lg px-3 py-3 transition-colors hover:bg-muted/30"
              >
                <div className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg ${colorClass}`}>
                  <Icon className="h-3.5 w-3.5" />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium">{activity.subject}</p>
                  {activity.description && (
                    <p className="mt-0.5 text-xs text-muted-foreground line-clamp-1">
                      {activity.description}
                    </p>
                  )}
                  <div className="mt-1 flex items-center gap-2 text-[10px] text-muted-foreground/60">
                    <span>{activity.entity_type}</span>
                    <span>·</span>
                    <span>{new Date(activity.created_at).toLocaleString()}</span>
                    {activity.user && (
                      <>
                        <span>·</span>
                        <span>
                          {activity.user.first_name} {activity.user.last_name}
                        </span>
                      </>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </motion.div>
  );
}
