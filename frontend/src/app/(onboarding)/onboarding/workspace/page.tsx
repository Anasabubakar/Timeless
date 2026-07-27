"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { motion } from "motion/react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useIntegrations, useConnectIntegration } from "@/queries/integrations";
import { useOnboardingState, useSaveOnboardingState } from "@/queries/onboarding";
import { useAuthStore } from "@/stores/auth";

const ZAPIER_APPS = [
  "Notion", "Gmail", "Google Calendar", "Google Meet", "Google Tasks", "Google Drive",
  "Sheets", "Slack", "Discord", "Airtable", "HubSpot", "ClickUp", "Monday", "Linear",
  "Trello", "Asana", "Apollo",
];

const NATIVE_PROVIDERS = [
  { key: "notion", label: "Notion", field: "token", placeholder: "secret_...", hint: "Internal integration token" },
  { key: "apollo", label: "Apollo", field: "api_key", placeholder: "Apollo API key", hint: "From Settings → Integrations → API" },
] as const;

function startOAuth(provider: string) {
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";
  const token = useAuthStore.getState().tokens?.access_token ?? "";
  window.location.href = `${apiUrl}/integrations/${provider}/oauth/start?token=${encodeURIComponent(token)}`;
}

export default function WorkspaceStepPage() {
  const router = useRouter();
  const { data: onboardingState } = useOnboardingState();
  const saveState = useSaveOnboardingState();
  const { data: integrationsData, refetch: refetchIntegrations } = useIntegrations();
  const connectIntegration = useConnectIntegration();

  const savedPayload = (onboardingState?.data.payload || {}) as Record<string, string>;
  const [zapierUrl, setZapierUrl] = useState(savedPayload.zapier_mcp_server_url || "");
  const [nativeValues, setNativeValues] = useState<Record<string, string>>({});
  const [connecting, setConnecting] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const autosaveTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (savedPayload.zapier_mcp_server_url) setZapierUrl(savedPayload.zapier_mcp_server_url);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [onboardingState?.data.id]);

  function autosave(payload: Record<string, string>) {
    if (autosaveTimer.current) clearTimeout(autosaveTimer.current);
    autosaveTimer.current = setTimeout(() => {
      saveState.mutate({ step: "workspace", payload });
    }, 600);
  }

  function handleZapierUrlChange(value: string) {
    setZapierUrl(value);
    autosave({ zapier_mcp_server_url: value });
  }

  const integrations = integrationsData?.data || [];
  const statusFor = (provider: string) => integrations.find((i) => i.provider === provider)?.status;

  async function handleConnect(provider: string, credentials: Record<string, string>) {
    setError(null);
    setConnecting(provider);
    try {
      await connectIntegration.mutateAsync({ provider, credentials });
      await refetchIntegrations();
    } catch (err: any) {
      setError(err.message || `Failed to connect ${provider}`);
    } finally {
      setConnecting(null);
    }
  }

  async function handleContinue() {
    await saveState.mutateAsync({ step: "discovery", payload: { zapier_mcp_server_url: zapierUrl } });
    router.push("/onboarding/discovery");
  }

  async function handleSkip() {
    await saveState.mutateAsync({ step: "discovery", payload: {} });
    router.push("/onboarding/discovery");
  }

  const zapierStatus = statusFor("zapier");
  const hasAnyConnection = integrations.some((i) => i.status === "syncing" || i.status === "active");

  return (
    <div className="space-y-6">
      <div className="text-center">
        <h1 className="text-2xl font-semibold">Connect your workspace</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Connect Zapier once to unlock all your apps. Timeless syncs the moment you connect.
        </p>
      </div>

      {error && (
        <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
          {error}
        </div>
      )}

      <motion.div initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.3 }}>
        <Card className="border-primary/30">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Zapier</CardTitle>
              {zapierStatus && (
                <Badge variant={zapierStatus === "active" ? "default" : "secondary"}>
                  {zapierStatus === "syncing" ? "Syncing your workspace..." : zapierStatus}
                </Badge>
              )}
            </div>
            <CardDescription>
              The primary way to connect Timeless to your tools — Notion, Gmail, Calendar, Meet,
              Tasks, Drive, Sheets, Slack, Discord, Airtable, HubSpot, CRM tools, ClickUp, Monday,
              Linear, Trello, Asana, Apollo, and more.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex flex-wrap gap-1.5">
              {ZAPIER_APPS.map((app) => (
                <span key={app} className="rounded-full border border-border px-2 py-0.5 text-xs text-muted-foreground">
                  {app}
                </span>
              ))}
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">MCP Server URL</label>
              <Input
                value={zapierUrl}
                onChange={(e) => handleZapierUrlChange(e.target.value)}
                placeholder="https://mcp.zapier.com/api/mcp/s/..."
              />
              <p className="text-xs text-muted-foreground">
                From your Zapier account: MCP → create a server → copy the MCP Server URL.
              </p>
            </div>
            <Button className="w-full" onClick={() => startOAuth("zapier")}>
              Connect Zapier with OAuth
            </Button>
            <Button
              variant="outline"
              className="w-full"
              disabled={!zapierUrl || connecting === "zapier"}
              onClick={() => handleConnect("zapier", { mcp_server_url: zapierUrl })}
            >
              {connecting === "zapier" ? "Connecting..." : "Connect with MCP Server URL"}
            </Button>
          </CardContent>
        </Card>
      </motion.div>

      <div className="flex items-center gap-3 text-xs text-muted-foreground">
        <div className="h-px flex-1 bg-border" />
        Or connect directly
        <div className="h-px flex-1 bg-border" />
      </div>

      <div className="grid grid-cols-2 gap-3">
        {NATIVE_PROVIDERS.map((provider) => {
          const status = statusFor(provider.key);
          return (
            <Card key={provider.key}>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="text-sm">{provider.label}</CardTitle>
                  {status && (
                    <Badge variant={status === "active" ? "default" : "secondary"} className="text-[10px]">
                      {status}
                    </Badge>
                  )}
                </div>
                <CardDescription className="text-xs">{provider.hint}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-2">
                <Input
                  type="password"
                  placeholder={provider.placeholder}
                  value={nativeValues[provider.key] || ""}
                  onChange={(e) =>
                    setNativeValues((prev) => ({ ...prev, [provider.key]: e.target.value }))
                  }
                />
                <Button size="sm" className="w-full" onClick={() => startOAuth(provider.key)}>
                  Connect with OAuth
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  className="w-full"
                  disabled={!nativeValues[provider.key] || connecting === provider.key}
                  onClick={() =>
                    handleConnect(provider.key, { [provider.field]: nativeValues[provider.key] })
                  }
                >
                  {connecting === provider.key ? "Connecting..." : "Use API key instead"}
                </Button>
              </CardContent>
            </Card>
          );
        })}
      </div>

      <div className="flex items-center justify-between pt-2">
        <Button variant="ghost" onClick={handleSkip}>
          Skip for now
        </Button>
        <Button onClick={handleContinue} disabled={!hasAnyConnection && !zapierUrl}>
          Continue
        </Button>
      </div>
    </div>
  );
}
