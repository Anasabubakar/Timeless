"use client";

import { useState } from "react";
import {
  Link2,
  Plus,
  Check,
  AlertCircle,
  AlertTriangle,
  Clock,
  Settings,
  Trash2,
  RefreshCw,
  ShieldOff,
  Merge,
  KeyRound,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import {
  useIntegrations,
  useIntegrationDashboard,
  useZapierApps,
  useCreateIntegration,
  useDeleteIntegration,
  useRevokeIntegration,
  useConnectIntegration,
  useRotateCredentials,
  useDedupeCompanies,
  type Integration,
  type DashboardEntry,
} from "@/queries/integrations";
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
import { Input } from "@/components/ui/input";

const STATUS_CONFIG: Record<string, { icon: typeof Check; color: string; label: string }> = {
  active: { icon: Check, color: "bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300", label: "Up to date" },
  syncing: { icon: Clock, color: "bg-blue-50 text-blue-700 dark:bg-blue-950 dark:text-blue-300", label: "Syncing..." },
  retrying: { icon: RefreshCw, color: "bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300", label: "Retrying..." },
  error: { icon: AlertCircle, color: "bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300", label: "Attention required" },
  expired: { icon: AlertTriangle, color: "bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300", label: "Reconnect required" },
  revoked: { icon: ShieldOff, color: "bg-neutral-100 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400", label: "Revoked" },
  pending: { icon: Clock, color: "bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300", label: "Pending" },
  inactive: { icon: AlertCircle, color: "bg-neutral-100 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400", label: "Inactive" },
};

const PROVIDER_COLORS: Record<string, string> = {
  zapier: "bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300",
  notion: "bg-neutral-100 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300",
  apollo: "bg-indigo-50 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300",
};

type ProviderDef = {
  id: string;
  name: string;
  type: string;
  oauth: boolean;
  credentialFields?: { field: string; label: string; placeholder: string }[];
};

// Zapier is the PRIMARY integration gateway; Notion and Apollo are secondary.
// Only providers with a real, working backend client are listed here — no
// placeholder entries for unimplemented providers.
const AVAILABLE_PROVIDERS: ProviderDef[] = [
  { id: "zapier", name: "Zapier", type: "automation", oauth: false, credentialFields: [
    { field: "token", label: "Zapier MCP Connection Token", placeholder: "Paste the token from mcp.zapier.com's Connect tab" },
  ] },
  { id: "notion", name: "Notion", type: "crm", oauth: true },
  { id: "apollo", name: "Apollo", type: "enrichment", oauth: false, credentialFields: [
    { field: "api_key", label: "Apollo API Key", placeholder: "Paste your Apollo.io API key" },
  ] },
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

function formatDuration(ms?: number): string {
  if (!ms) return "—";
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

export default function IntegrationsPage() {
  const { data, isLoading } = useIntegrations();
  const { data: dashboardData } = useIntegrationDashboard();
  const [showCreate, setShowCreate] = useState(false);
  useNewParam(() => setShowCreate(true));
  const [managingId, setManagingId] = useState<string | null>(null);

  const dedupeCompanies = useDedupeCompanies();
  const rotateCredentials = useRotateCredentials();

  const integrations: Integration[] = data?.data ?? [];
  const dashboard: DashboardEntry[] = dashboardData?.data ?? [];
  const managingIntegration = integrations.find((i) => i.id === managingId);
  const managingEntry = dashboard.find((e) => e.integration.id === managingId);

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
            Connect Zapier, Notion, and Apollo to sync data automatically
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={dedupeCompanies.isPending}
            onClick={() =>
              dedupeCompanies.mutate(undefined, {
                onSuccess: (res) =>
                  toast.success(`Merged ${res.data.companies_merged} duplicate compan${res.data.companies_merged === 1 ? "y" : "ies"} across ${res.data.groups_found} group(s)`),
                onError: () => toast.error("Failed to merge duplicate companies"),
              })
            }
          >
            <Merge className="h-4 w-4 mr-2" />
            Merge duplicates
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={rotateCredentials.isPending}
            onClick={() =>
              rotateCredentials.mutate(undefined, {
                onSuccess: (res) =>
                  toast.success(`Rotated ${res.data.rotated}/${res.data.checked} stored credential(s) to the current key`),
                onError: () => toast.error("Failed to rotate credentials"),
              })
            }
          >
            <KeyRound className="h-4 w-4 mr-2" />
            Rotate keys
          </Button>
          <Button onClick={() => setShowCreate(true)}>
            <Plus className="h-4 w-4 mr-2" />
            Add Integration
          </Button>
        </div>
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
              Connect Zapier, Notion, or Apollo to sync data automatically — once, and it keeps working.
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
            const entry = dashboard.find((e) => e.integration.id === integration.id);
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
                  {entry && entry.failed_runs_24h > 0 && (
                    <p className="mt-2 text-xs text-amber-600 dark:text-amber-400">
                      {entry.failed_runs_24h} failed sync{entry.failed_runs_24h === 1 ? "" : "s"} in the last 24h
                    </p>
                  )}
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
          entry={managingEntry}
        />
      )}
    </motion.div>
  );
}

function AddIntegrationDialog({ open, onOpenChange, existingProviders }: { open: boolean; onOpenChange: (v: boolean) => void; existingProviders: string[] }) {
  const createIntegration = useCreateIntegration();
  const connectIntegration = useConnectIntegration();
  const [manualProvider, setManualProvider] = useState<ProviderDef | null>(null);
  const [credentials, setCredentials] = useState<Record<string, string>>({});

  const handleConnect = (provider: ProviderDef) => {
    if (provider.oauth) {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";
      const token = useAuthStore.getState().tokens?.access_token ?? "";
      window.location.href = `${apiUrl}/integrations/${provider.id}/oauth/start?token=${encodeURIComponent(token)}`;
      return;
    }
    if (provider.credentialFields) {
      setManualProvider(provider);
      setCredentials({});
      return;
    }
    createIntegration.mutate(
      { provider: provider.id, type: provider.type, name: provider.name, status: "pending" },
      { onSuccess: () => onOpenChange(false) }
    );
  };

  const handleManualConnect = () => {
    if (!manualProvider) return;
    connectIntegration.mutate(
      { provider: manualProvider.id, credentials },
      {
        onSuccess: () => {
          toast.success(`${manualProvider.name} connected — syncing now`);
          setManualProvider(null);
          onOpenChange(false);
        },
        onError: (err) => {
          toast.error(err instanceof Error ? err.message : `Could not validate ${manualProvider.name} credentials`);
        },
      }
    );
  };

  if (manualProvider) {
    const allFilled = manualProvider.credentialFields!.every((f) => credentials[f.field]?.trim());
    return (
      <Dialog open={open} onOpenChange={(v) => { onOpenChange(v); if (!v) setManualProvider(null); }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Connect {manualProvider.name}</DialogTitle>
            <DialogDescription>
              We validate this immediately against {manualProvider.name}'s API before saving it — nothing is stored unless it's a real, working credential.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3 mt-2">
            {manualProvider.credentialFields!.map((f) => (
              <div key={f.field} className="space-y-1.5">
                <label htmlFor={f.field} className="text-sm font-medium">{f.label}</label>
                <Input
                  id={f.field}
                  type="password"
                  placeholder={f.placeholder}
                  value={credentials[f.field] ?? ""}
                  onChange={(e) => setCredentials((c) => ({ ...c, [f.field]: e.target.value }))}
                  autoComplete="off"
                />
              </div>
            ))}
          </div>
          <DialogFooter className="mt-4">
            <Button variant="outline" onClick={() => setManualProvider(null)}>Back</Button>
            <Button disabled={!allFilled || connectIntegration.isPending} onClick={handleManualConnect}>
              {connectIntegration.isPending ? "Validating…" : "Connect"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Integration</DialogTitle>
          <DialogDescription>Zapier is tried first for anything it can reach; Notion and Apollo are secondary.</DialogDescription>
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
                  <p className="text-[10px] text-muted-foreground capitalize">{provider.type}{provider.oauth ? " · OAuth" : " · API key"}</p>
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

function ManageIntegrationDialog({
  open,
  onOpenChange,
  integration,
  entry,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  integration: Integration;
  entry?: DashboardEntry;
}) {
  const deleteIntegration = useDeleteIntegration();
  const revokeIntegration = useRevokeIntegration();
  const connectIntegration = useConnectIntegration();
  const [confirming, setConfirming] = useState(false);
  const { data: zapierApps } = useZapierApps(integration.provider === "zapier" && integration.status === "active");

  const handleRevoke = () => {
    if (!confirming) {
      setConfirming(true);
      return;
    }
    revokeIntegration.mutate(integration.id, {
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
      <DialogContent className="max-h-[80vh] overflow-y-auto">
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
            {entry && (
              <div className="flex justify-between">
                <span>Failed syncs (24h)</span>
                <span className="font-medium text-foreground">{entry.failed_runs_24h}</span>
              </div>
            )}
            {integration.last_error && (
              <div className="rounded-lg bg-red-50 dark:bg-red-950 p-2 text-xs text-red-700 dark:text-red-300">
                {integration.last_error}
              </div>
            )}
          </div>

          {zapierApps && zapierApps.data.length > 0 && (
            <div className="space-y-2">
              <p className="text-xs font-medium text-foreground">
                Connected apps {zapierApps.agentic_mode ? "(agentic mode)" : "(classic mode)"}
              </p>
              <div className="flex flex-wrap gap-1.5">
                {zapierApps.data.map((app) => (
                  <Badge key={app.slug} variant="secondary" className="text-[10px]">
                    {app.slug} · {app.actions.length}
                  </Badge>
                ))}
              </div>
            </div>
          )}

          {entry && entry.recent_runs.length > 0 && (
            <div className="space-y-2">
              <p className="text-xs font-medium text-foreground">Sync history</p>
              <div className="space-y-1.5 max-h-48 overflow-y-auto">
                {entry.recent_runs.map((run) => (
                  <div key={run.id} className="flex items-center justify-between rounded-md border border-border px-2.5 py-1.5 text-xs">
                    <div className="flex flex-col">
                      <span className="font-medium text-foreground capitalize">{run.trigger} · {run.status}</span>
                      <span className="text-muted-foreground">
                        {new Date(run.started_at).toLocaleString()} · {run.records_synced} records · {formatDuration(run.duration_ms)}
                      </span>
                      {run.error && <span className="text-red-600 dark:text-red-400 line-clamp-1">{run.error}</span>}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          <div className="space-y-2 pt-2 border-t border-border">
            {(integration.status === "error" || integration.status === "expired") && (
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
              onClick={handleRevoke}
              disabled={revokeIntegration.isPending}
              className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-amber-600 dark:text-amber-400 hover:bg-amber-50 dark:hover:bg-amber-950 transition-colors"
            >
              <ShieldOff className="h-4 w-4" />
              {confirming ? "Click again to confirm revoke" : "Revoke access"}
            </button>
            <button
              onClick={() => deleteIntegration.mutate(integration.id, { onSuccess: () => onOpenChange(false) })}
              disabled={deleteIntegration.isPending}
              className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-950 transition-colors"
            >
              <Trash2 className="h-4 w-4" />
              Remove permanently
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
