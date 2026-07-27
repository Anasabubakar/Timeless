"use client";

import { useState } from "react";
import { Link2, Plus, Check, AlertCircle, Clock, Settings, Trash2, RefreshCw, MoreVertical } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { useIntegrations, useCreateIntegration, useDeleteIntegration, useConnectIntegration, type Integration } from "@/queries/integrations";
import { useAuthStore } from "@/stores/auth";
import { toast } from "sonner";
import { useEffect } from "react";
import { motion } from "motion/react";
import { useNewParam } from "@/hooks/use-new-param";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";

const STATUS_CONFIG: Record<string, { icon: typeof Check; color: string; label: string }> = {
  active: { icon: Check, color: "bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300", label: "Connected" },
  error: { icon: AlertCircle, color: "bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300", label: "Error" },
  pending: { icon: Clock, color: "bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300", label: "Pending" },
  inactive: { icon: AlertCircle, color: "bg-neutral-100 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400", label: "Inactive" },
};

const PROVIDER_COLORS: Record<string, string> = {
  salesforce: "bg-blue-50 text-blue-700 dark:bg-blue-950 dark:text-blue-300",
  hubspot: "bg-orange-50 text-orange-700 dark:bg-orange-950 dark:text-orange-300",
  slack: "bg-purple-50 text-purple-700 dark:bg-purple-950 dark:text-purple-300",
  google: "bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300",
  zapier: "bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300",
  notion: "bg-neutral-100 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300",
  apollo: "bg-indigo-50 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300",
  webhook: "bg-neutral-50 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300",
};

const AVAILABLE_PROVIDERS = [
  { id: "notion", name: "Notion", type: "crm", oauth: true },
  { id: "apollo", name: "Apollo", type: "enrichment", oauth: true },
  { id: "zapier", name: "Zapier", type: "automation", oauth: true },
  { id: "salesforce", name: "Salesforce", type: "crm", oauth: false },
  { id: "hubspot", name: "HubSpot", type: "crm", oauth: false },
  { id: "slack", name: "Slack", type: "messaging", oauth: false },
  { id: "google", name: "Google", type: "email", oauth: false },
  { id: "webhook", name: "Webhook", type: "webhook", oauth: false },
];

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
  const [showCreate, setShowCreate] = useState(false);
  useNewParam(() => setShowCreate(true));
  const [managingId, setManagingId] = useState<string | null>(null);

  const integrations: Integration[] = data?.data ?? [];
  const managingIntegration = integrations.find((i) => i.id === managingId);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const connected = params.get("connected");
    const error = params.get("error");
    if (connected) {
      toast.success(`${connected} connected — syncing your workspace`);
    } else if (error === "oauth_not_configured") {
      toast.error(`OAuth isn't configured for ${params.get("provider") ?? "this provider"} yet. Set its client ID/secret in the backend env.`);
    } else if (error) {
      toast.error(error);
    }
    if (connected || error) {
      window.history.replaceState(null, "", window.location.pathname);
    }
  }, []);

  return (
    <motion.div className="space-y-6" initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.3 }}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Integrations</h1>
          <p className="text-sm text-muted-foreground">
            Connect your tools and sync data across platforms
          </p>
        </div>
        <Button onClick={() => setShowCreate(true)}>
          <Plus className="h-4 w-4 mr-2" />
          Add Integration
        </Button>
      </div>

      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Card key={i} className="animate-pulse">
              <CardContent className="p-5 space-y-3">
                <div className="h-4 bg-neutral-200 dark:bg-neutral-700 rounded w-1/2" />
                <div className="h-3 bg-neutral-100 dark:bg-neutral-800 rounded w-2/3" />
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
            <Button className="mt-4" onClick={() => setShowCreate(true)}>
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
                      <div className={`flex h-9 w-9 items-center justify-center rounded-lg ${PROVIDER_COLORS[integration.provider] ?? "bg-neutral-50 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300"}`}>
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
                    <button
                      onClick={() => setManagingId(integration.id)}
                      className="flex items-center gap-1 rounded-md px-2 py-1 text-xs hover:bg-muted transition-colors"
                    >
                      <Settings className="h-3 w-3" />
                      Manage
                    </button>
                  </div>
                  {integration.last_error && (
                    <p className="mt-2 text-xs text-red-600 dark:text-red-400 line-clamp-2">{integration.last_error}</p>
                  )}
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}

      <AddIntegrationDialog open={showCreate} onOpenChange={setShowCreate} existingProviders={integrations.map((i) => i.provider)} />
      {managingIntegration && (
        <ManageIntegrationDialog
          open={!!managingId}
          onOpenChange={(v) => !v && setManagingId(null)}
          integration={managingIntegration}
        />
      )}
    </motion.div>
  );
}

function AddIntegrationDialog({ open, onOpenChange, existingProviders }: { open: boolean; onOpenChange: (v: boolean) => void; existingProviders: string[] }) {
  const createIntegration = useCreateIntegration();
  const connectIntegration = useConnectIntegration();

  const handleConnect = (provider: typeof AVAILABLE_PROVIDERS[0]) => {
    if (provider.oauth) {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";
      const token = useAuthStore.getState().tokens?.access_token ?? "";
      window.location.href = `${apiUrl}/integrations/${provider.id}/oauth/start?token=${encodeURIComponent(token)}`;
      return;
    }

    createIntegration.mutate(
      { provider: provider.id, type: provider.type, name: provider.name, status: "pending" },
      { onSuccess: () => onOpenChange(false) }
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Integration</DialogTitle>
          <DialogDescription>Connect a new tool to your workspace</DialogDescription>
        </DialogHeader>
        <div className="mt-3 space-y-1">
          {AVAILABLE_PROVIDERS.map((provider) => {
            const connected = existingProviders.includes(provider.id);
            return (
              <button
                key={provider.id}
                onClick={() => !connected && handleConnect(provider)}
                disabled={connected || createIntegration.isPending}
                className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${PROVIDER_COLORS[provider.id] ?? "bg-neutral-100 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300"}`}>
                  <Link2 className="h-3.5 w-3.5" />
                </div>
                <div className="flex-1 text-left">
                  <p className="text-sm font-medium">{provider.name}</p>
                  <p className="text-[10px] text-muted-foreground capitalize">{provider.type}{provider.oauth ? " · OAuth" : ""}</p>
                </div>
                {connected && (
                  <Badge variant="secondary" className="text-[10px]">Connected</Badge>
                )}
              </button>
            );
          })}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function ManageIntegrationDialog({ open, onOpenChange, integration }: { open: boolean; onOpenChange: (v: boolean) => void; integration: Integration }) {
  const deleteIntegration = useDeleteIntegration();
  const connectIntegration = useConnectIntegration();
  const [confirming, setConfirming] = useState(false);

  const handleDisconnect = () => {
    if (!confirming) {
      setConfirming(true);
      return;
    }
    deleteIntegration.mutate(integration.id, {
      onSuccess: () => {
        onOpenChange(false);
        setConfirming(false);
      },
    });
  };

  const handleReconnect = () => {
    connectIntegration.mutate(
      { provider: integration.provider, credentials: {} },
      { onSuccess: () => onOpenChange(false) }
    );
  };

  const status = STATUS_CONFIG[integration.status] ?? STATUS_CONFIG.inactive;
  const StatusIcon = status.icon;

  return (
    <Dialog open={open} onOpenChange={(v) => { onOpenChange(v); if (!v) setConfirming(false); }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Manage {integration.name}</DialogTitle>
        </DialogHeader>
        <div className="mt-3 space-y-4">
          <div className="flex items-center gap-3 rounded-lg border border-border p-3">
            <div className={`flex h-9 w-9 items-center justify-center rounded-lg ${PROVIDER_COLORS[integration.provider] ?? "bg-neutral-100 dark:bg-neutral-800"}`}>
              <Link2 className="h-4 w-4" />
            </div>
            <div className="flex-1">
              <p className="text-sm font-medium">{integration.name}</p>
              <p className="text-xs text-muted-foreground capitalize">{integration.provider} &middot; {integration.type}</p>
            </div>
            <Badge className={status.color}>
              <StatusIcon className="h-3 w-3 mr-1" />
              {status.label}
            </Badge>
          </div>

          <div className="space-y-2 text-xs text-muted-foreground">
            <div className="flex justify-between">
              <span>Last sync</span>
              <span className="font-medium text-foreground">{formatLastSync(integration.last_sync_at)}</span>
            </div>
            {integration.last_error && (
              <div className="rounded-lg bg-red-50 dark:bg-red-950 p-2 text-xs text-red-700 dark:text-red-300">
                {integration.last_error}
              </div>
            )}
          </div>

          <div className="space-y-2 pt-2 border-t border-border">
            {integration.status === "error" && (
              <button
                onClick={handleReconnect}
                disabled={connectIntegration.isPending}
                className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm hover:bg-muted transition-colors"
              >
                <RefreshCw className="h-4 w-4" />
                Reconnect
              </button>
            )}
            <button
              onClick={handleDisconnect}
              disabled={deleteIntegration.isPending}
              className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-950 transition-colors"
            >
              <Trash2 className="h-4 w-4" />
              {confirming ? "Click again to confirm" : "Disconnect"}
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
