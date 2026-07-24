"use client";

import { Link2, Plus, Check, AlertCircle, Clock } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { useIntegrations, type Integration } from "@/queries/integrations";
import { motion } from "motion/react";

const STATUS_CONFIG: Record<string, { icon: typeof Check; color: string; label: string }> = {
  active: { icon: Check, color: "bg-emerald-50 text-emerald-700", label: "Connected" },
  error: { icon: AlertCircle, color: "bg-red-50 text-red-700", label: "Error" },
  pending: { icon: Clock, color: "bg-amber-50 text-amber-700", label: "Pending" },
  inactive: { icon: AlertCircle, color: "bg-neutral-100 text-neutral-600", label: "Inactive" },
};

const PROVIDER_COLORS: Record<string, string> = {
  salesforce: "bg-blue-50 text-blue-700",
  hubspot: "bg-orange-50 text-orange-700",
  slack: "bg-purple-50 text-purple-700",
  google: "bg-red-50 text-red-700",
  zapier: "bg-amber-50 text-amber-700",
  webhook: "bg-neutral-50 text-neutral-700",
};

function formatLastSync(dateStr?: string): string {
  if (!dateStr) return "Never synced";
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "Just now";
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

export default function IntegrationsPage() {
  const { data, isLoading } = useIntegrations();

  const integrations: Integration[] = data?.data ?? [];

  return (
    <motion.div className="space-y-6" initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.3 }}>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Integrations</h1>
          <p className="text-sm text-muted-foreground">
            Connect your tools and sync data across platforms
          </p>
        </div>
        <Button>
          <Plus className="h-4 w-4 mr-2" />
          Add Integration
        </Button>
      </div>

      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Card key={i} className="animate-pulse">
              <CardContent className="p-5 space-y-3">
                <div className="h-4 bg-neutral-200 rounded w-1/2" />
                <div className="h-3 bg-neutral-100 rounded w-2/3" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : integrations.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-16 text-center">
            <Link2 className="h-12 w-12 text-muted-foreground/40 mb-4" />
            <h3 className="text-lg font-medium">No integrations yet</h3>
            <p className="text-sm text-muted-foreground mt-1 max-w-sm">
              Connect your CRM, email, and communication tools to sync data automatically.
            </p>
            <Button className="mt-4">
              <Plus className="h-4 w-4 mr-2" />
              Add First Integration
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {integrations.map((integration) => {
            const status = STATUS_CONFIG[integration.status] ?? STATUS_CONFIG.inactive;
            const StatusIcon = status.icon;
            return (
              <Card key={integration.id} className="hover:shadow-md transition-shadow">
                <CardContent className="p-5">
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-3">
                      <div className={`flex h-9 w-9 items-center justify-center rounded-lg ${PROVIDER_COLORS[integration.provider] ?? "bg-neutral-50 text-neutral-700"}`}>
                        <Link2 className="h-4 w-4" />
                      </div>
                      <div>
                        <h3 className="text-sm font-medium">{integration.name}</h3>
                        <p className="text-xs text-muted-foreground capitalize">{integration.provider} &middot; {integration.type}</p>
                      </div>
                    </div>
                    <Badge className={status.color}>
                      <StatusIcon className="h-3 w-3 mr-1" />
                      {status.label}
                    </Badge>
                  </div>
                  <div className="mt-4 flex items-center justify-between text-xs text-muted-foreground">
                    <span>Last sync: {formatLastSync(integration.last_sync_at)}</span>
                  </div>
                  {integration.last_error && (
                    <p className="mt-2 text-xs text-red-600 line-clamp-2">{integration.last_error}</p>
                  )}
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}
    </motion.div>
  );
}
