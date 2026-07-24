"use client";

import { useState, useEffect, Fragment, useRef } from "react";
import {
  Building2,
  User,
  Shield,
  Bell,
  Key,
  Loader2,
  Check,
  Webhook,
  Plus,
  RotateCcw,
  Trash2,
  Send,
  Copy,
  Eye,
  EyeOff,
  ChevronDown,
  ChevronRight,
  AlertTriangle,
  CheckCircle2,
  XCircle,
  Clock,
  Upload,
  FileUp,
} from "lucide-react";
import { useAuthStore } from "@/stores/auth";
import {
  useCurrentOrganization,
  useUpdateOrganization,
  useUpdateProfile,
  useChangePassword,
} from "@/queries/settings";
import {
  useWebhooks,
  useCreateWebhook,
  useUpdateWebhook,
  useDeleteWebhook,
  useRotateWebhookSecret,
  useTestWebhook,
  useWebhookDeliveries,
  type Webhook as WebhookType,
  type WebhookDelivery,
} from "@/queries/webhooks";
import { useTeamMembers, useInviteMember, useRemoveMember, useOrgRoles } from "@/queries/team";
import { useNotificationPreferences, useUpdateNotificationPreference } from "@/queries/notifications";
import { useImportCompanies, useImportContacts, useImportSponsors, type ImportResult } from "@/queries/import";

type Tab = "organization" | "profile" | "team" | "notifications" | "api" | "webhooks" | "import";

const WEBHOOK_EVENTS = [
  { value: "sponsor.created", label: "Sponsor Created" },
  { value: "sponsor.updated", label: "Sponsor Updated" },
  { value: "sponsor.deleted", label: "Sponsor Deleted" },
  { value: "sponsor.stage_changed", label: "Sponsor Stage Changed" },
  { value: "company.created", label: "Company Created" },
  { value: "company.updated", label: "Company Updated" },
  { value: "contact.created", label: "Contact Created" },
  { value: "contact.updated", label: "Contact Updated" },
  { value: "proposal.created", label: "Proposal Created" },
  { value: "proposal.updated", label: "Proposal Updated" },
  { value: "proposal.sent", label: "Proposal Sent" },
  { value: "campaign.created", label: "Campaign Created" },
  { value: "campaign.updated", label: "Campaign Updated" },
  { value: "communication.created", label: "Communication Created" },
  { value: "ai.task_completed", label: "AI Task Completed" },
];

export default function SettingsPage() {
  const [tab, setTab] = useState<Tab>("organization");

  const tabs: { id: Tab; label: string; icon: typeof Building2 }[] = [
    { id: "organization", label: "Organization", icon: Building2 },
    { id: "profile", label: "Profile", icon: User },
    { id: "team", label: "Team & Roles", icon: Shield },
    { id: "notifications", label: "Notifications", icon: Bell },
    { id: "api", label: "API Keys", icon: Key },
    { id: "webhooks", label: "Webhooks", icon: Webhook },
    { id: "import", label: "Import", icon: Upload },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Settings</h1>
        <p className="text-sm text-muted-foreground">
          Manage your organization and account preferences
        </p>
      </div>

      <div className="flex gap-6">
        <nav className="w-48 shrink-0 space-y-0.5">
          {tabs.map(({ id, label, icon: Icon }) => (
            <button
              key={id}
              onClick={() => setTab(id)}
              className={`flex w-full items-center gap-2 rounded-lg px-3 py-2 text-xs font-medium transition-colors ${
                tab === id
                  ? "bg-neutral-100 text-neutral-900 dark:bg-neutral-800 dark:text-neutral-100"
                  : "text-muted-foreground hover:bg-muted/50 hover:text-foreground"
              }`}
            >
              <Icon className="h-3.5 w-3.5" />
              {label}
            </button>
          ))}
        </nav>

        <div className="flex-1">
          {tab === "organization" && <OrgSettings />}
          {tab === "profile" && <ProfileSettings />}
          {tab === "team" && <TeamSettings />}
          {tab === "notifications" && <NotificationSettings />}
          {tab === "api" && <ApiSettings />}
          {tab === "webhooks" && <WebhookSettings />}
          {tab === "import" && <ImportSettings />}
        </div>
      </div>
    </div>
  );
}

// ── Webhook Settings ────────────────────────────────────────────────────────

function WebhookSettings() {
  const { data, isLoading } = useWebhooks();
  const [showCreate, setShowCreate] = useState(false);

  const webhooks = (data as { data: WebhookType[] } | undefined)?.data ?? [];

  if (isLoading) {
    return (
      <div className="flex h-40 items-center justify-center">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <SettingsSection title="Webhooks">
        <p className="text-xs text-muted-foreground">
          Webhooks notify external systems when events happen in SponsorOS.
          Configure endpoints to receive real-time event payloads with HMAC-SHA256 signed requests.
        </p>

        {!showCreate && (
          <button
            onClick={() => setShowCreate(true)}
            className="flex items-center gap-1.5 h-8 rounded-lg bg-neutral-900 px-3 text-xs font-medium text-white hover:bg-neutral-800 dark:bg-neutral-100 dark:text-neutral-900 dark:hover:bg-neutral-200"
          >
            <Plus className="h-3.5 w-3.5" />
            Add Webhook
          </button>
        )}

        {showCreate && (
          <WebhookCreateForm onClose={() => setShowCreate(false)} />
        )}
      </SettingsSection>

      {webhooks.length === 0 && !showCreate ? (
        <div className="flex flex-col items-center gap-2 py-12 text-center">
          <Webhook className="h-8 w-8 text-muted-foreground/50" />
          <p className="text-sm font-medium text-muted-foreground">No webhooks configured</p>
          <p className="text-xs text-muted-foreground/70">
            Add a webhook endpoint to receive event notifications
          </p>
        </div>
      ) : (
        <div className="space-y-3">
          {webhooks.map((wh) => (
            <WebhookCard key={wh.id} webhook={wh} />
          ))}
        </div>
      )}
    </div>
  );
}

function WebhookCreateForm({ onClose }: { onClose: () => void }) {
  const createWebhook = useCreateWebhook();
  const [url, setUrl] = useState("");
  const [selectedEvents, setSelectedEvents] = useState<string[]>([]);

  const toggleEvent = (event: string) => {
    setSelectedEvents((prev) =>
      prev.includes(event) ? prev.filter((e) => e !== event) : [...prev, event]
    );
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    createWebhook.mutate(
      { url, events: selectedEvents },
      {
        onSuccess: () => {
          setUrl("");
          setSelectedEvents([]);
          onClose();
        },
      }
    );
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4 rounded-lg border border-border p-4">
      <FieldGroup label="Endpoint URL">
        <input
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://example.com/webhooks"
          className="h-9 w-full rounded-[10px] border border-neutral-200 px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10 dark:border-neutral-700 dark:bg-neutral-900"
        />
      </FieldGroup>

      <FieldGroup label="Events">
        <div className="flex flex-wrap gap-2 pt-1">
          {WEBHOOK_EVENTS.map(({ value, label }) => (
            <button
              key={value}
              type="button"
              onClick={() => toggleEvent(value)}
              className={`rounded-md border px-2.5 py-1 text-[11px] font-medium transition-colors ${
                selectedEvents.includes(value)
                  ? "border-neutral-900 bg-neutral-900 text-white dark:border-neutral-100 dark:bg-neutral-100 dark:text-neutral-900"
                  : "border-neutral-200 text-neutral-600 hover:border-neutral-300 dark:border-neutral-700 dark:text-neutral-400"
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      </FieldGroup>

      <div className="flex items-center justify-end gap-2">
        <button
          type="button"
          onClick={onClose}
          className="h-8 rounded-lg border border-neutral-200 px-3 text-xs font-medium hover:bg-neutral-50 dark:border-neutral-700 dark:hover:bg-neutral-800"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={!url || selectedEvents.length === 0 || createWebhook.isPending}
          className="h-8 rounded-lg bg-neutral-900 px-4 text-xs font-medium text-white hover:bg-neutral-800 disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900 dark:hover:bg-neutral-200"
        >
          {createWebhook.isPending ? "Creating..." : "Create Webhook"}
        </button>
      </div>
    </form>
  );
}

function WebhookCard({ webhook }: { webhook: WebhookType }) {
  const [expanded, setExpanded] = useState(false);
  const [showSecret, setShowSecret] = useState(false);
  const [showDeliveries, setShowDeliveries] = useState(false);

  const deleteWebhook = useDeleteWebhook();
  const toggleActive = useUpdateWebhook(webhook.id);
  const rotateSecret = useRotateWebhookSecret();
  const testWebhook = useTestWebhook();

  const events: string[] = Array.isArray(webhook.events) ? webhook.events : [];

  const handleToggle = () => {
    toggleActive.mutate({ is_active: !webhook.is_active });
  };

  return (
    <div className="rounded-xl border border-border">
      <div
        className="flex items-center gap-3 p-4 cursor-pointer"
        onClick={() => setExpanded(!expanded)}
      >
        {expanded ? (
          <ChevronDown className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
        ) : (
          <ChevronRight className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
        )}

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium truncate">{webhook.url}</span>
            <span
              className={`inline-flex items-center rounded-full px-1.5 py-0.5 text-[10px] font-medium ${
                webhook.is_active
                  ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400"
                  : "bg-neutral-100 text-neutral-500 dark:bg-neutral-800 dark:text-neutral-400"
              }`}
            >
              {webhook.is_active ? "Active" : "Inactive"}
            </span>
            {webhook.failure_count > 0 && (
              <span className="inline-flex items-center gap-0.5 rounded-full bg-amber-50 px-1.5 py-0.5 text-[10px] font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-400">
                <AlertTriangle className="h-2.5 w-2.5" />
                {webhook.failure_count} failures
              </span>
            )}
          </div>
          <p className="text-[10px] text-muted-foreground mt-0.5">
            {events.length} event{events.length !== 1 ? "s" : ""} subscribed
          </p>
        </div>

        <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
          <button
            onClick={() => testWebhook.mutate(webhook.id)}
            disabled={!webhook.is_active || testWebhook.isPending}
            title="Send test"
            className="h-7 w-7 rounded-md border border-neutral-200 flex items-center justify-center hover:bg-neutral-50 disabled:opacity-40 dark:border-neutral-700 dark:hover:bg-neutral-800"
          >
            <Send className="h-3 w-3" />
          </button>
          <button
            onClick={handleToggle}
            disabled={toggleActive.isPending}
            title={webhook.is_active ? "Disable" : "Enable"}
            className={`relative h-5 w-9 rounded-full transition-colors ${
              webhook.is_active ? "bg-neutral-900 dark:bg-neutral-100" : "bg-neutral-200 dark:bg-neutral-700"
            }`}
          >
            <div
              className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition-transform dark:bg-neutral-900 ${
                webhook.is_active ? "translate-x-4" : "translate-x-0.5"
              }`}
            />
          </button>
        </div>
      </div>

      {expanded && (
        <div className="border-t border-border px-4 py-4 space-y-4">
          {/* Events */}
          <div>
            <p className="text-[11px] font-medium text-muted-foreground mb-1.5">Subscribed Events</p>
            <div className="flex flex-wrap gap-1.5">
              {events.map((evt) => (
                <span
                  key={evt}
                  className="rounded-md border border-neutral-200 bg-neutral-50 px-2 py-0.5 text-[10px] font-medium dark:border-neutral-700 dark:bg-neutral-800"
                >
                  {evt}
                </span>
              ))}
            </div>
          </div>

          {/* Secret */}
          <div>
            <p className="text-[11px] font-medium text-muted-foreground mb-1.5">Signing Secret</p>
            <div className="flex items-center gap-2">
              <code className="flex-1 rounded-md border border-neutral-200 bg-neutral-50 px-2.5 py-1.5 text-[11px] font-mono dark:border-neutral-700 dark:bg-neutral-800">
                {showSecret ? webhook.secret : "whsec_••••••••••••••••••••••••••••"}
              </code>
              <button
                onClick={() => setShowSecret(!showSecret)}
                title={showSecret ? "Hide" : "Reveal"}
                className="h-7 w-7 rounded-md border border-neutral-200 flex items-center justify-center hover:bg-neutral-50 dark:border-neutral-700 dark:hover:bg-neutral-800"
              >
                {showSecret ? <EyeOff className="h-3 w-3" /> : <Eye className="h-3 w-3" />}
              </button>
              <button
                onClick={() => navigator.clipboard.writeText(webhook.secret)}
                title="Copy"
                className="h-7 w-7 rounded-md border border-neutral-200 flex items-center justify-center hover:bg-neutral-50 dark:border-neutral-700 dark:hover:bg-neutral-800"
              >
                <Copy className="h-3 w-3" />
              </button>
              <button
                onClick={() => {
                  if (confirm("Are you sure? This will invalidate the current secret.")) {
                    rotateSecret.mutate(webhook.id);
                  }
                }}
                disabled={rotateSecret.isPending}
                title="Rotate secret"
                className="h-7 rounded-md border border-neutral-200 px-2 flex items-center gap-1 text-[10px] font-medium hover:bg-neutral-50 dark:border-neutral-700 dark:hover:bg-neutral-800"
              >
                <RotateCcw className="h-3 w-3" />
                Rotate
              </button>
            </div>
          </div>

          {/* Delivery Log Toggle */}
          <div>
            <button
              onClick={() => setShowDeliveries(!showDeliveries)}
              className="flex items-center gap-1.5 text-[11px] font-medium text-primary hover:underline"
            >
              {showDeliveries ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
              Recent Deliveries
            </button>
            {showDeliveries && <DeliveryLog webhookId={webhook.id} />}
          </div>

          {/* Actions */}
          <div className="flex items-center justify-end gap-2 pt-2 border-t border-border">
            <button
              onClick={() => {
                if (confirm("Delete this webhook? This cannot be undone.")) {
                  deleteWebhook.mutate(webhook.id);
                }
              }}
              disabled={deleteWebhook.isPending}
              className="flex items-center gap-1 h-7 rounded-md border border-red-200 px-2.5 text-[10px] font-medium text-red-600 hover:bg-red-50 dark:border-red-900 dark:text-red-400 dark:hover:bg-red-950"
            >
              <Trash2 className="h-3 w-3" />
              Delete
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

function DeliveryLog({ webhookId }: { webhookId: string }) {
  const { data, isLoading } = useWebhookDeliveries(webhookId);
  const deliveries = (data as { data: WebhookDelivery[] } | undefined)?.data ?? [];

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-4">
        <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (deliveries.length === 0) {
    return (
      <p className="py-3 text-[10px] text-muted-foreground">No deliveries yet</p>
    );
  }

  return (
    <div className="mt-2 space-y-1.5 max-h-60 overflow-y-auto">
      {deliveries.map((d) => (
        <div
          key={d.id}
          className="flex items-center gap-2 rounded-md border border-neutral-100 bg-neutral-50/50 px-2.5 py-2 dark:border-neutral-800 dark:bg-neutral-900/50"
        >
          <DeliveryStatusIcon status={d.status} />
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="text-[10px] font-medium">{d.event}</span>
              {d.response_code && (
                <span
                  className={`text-[9px] font-mono ${
                    d.response_code >= 200 && d.response_code < 300
                      ? "text-emerald-600"
                      : "text-red-500"
                  }`}
                >
                  HTTP {d.response_code}
                </span>
              )}
            </div>
            <p className="text-[9px] text-muted-foreground">
              {d.attempts}/{d.max_attempts} attempts
              {d.error ? ` • ${d.error}` : ""}
            </p>
          </div>
          <span className="text-[9px] text-muted-foreground shrink-0">
            {new Date(d.created_at).toLocaleString()}
          </span>
        </div>
      ))}
    </div>
  );
}

function DeliveryStatusIcon({ status }: { status: string }) {
  switch (status) {
    case "delivered":
      return <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500 shrink-0" />;
    case "failed":
      return <XCircle className="h-3.5 w-3.5 text-red-500 shrink-0" />;
    case "pending":
      return <Clock className="h-3.5 w-3.5 text-amber-500 shrink-0" />;
    default:
      return <Clock className="h-3.5 w-3.5 text-muted-foreground shrink-0" />;
  }
}

// ── Organization Settings ───────────────────────────────────────────────────

function OrgSettings() {
  const { data, isLoading } = useCurrentOrganization();
  const updateOrg = useUpdateOrganization();
  const [form, setForm] = useState({ name: "", slug: "", domain: "" });

  const org = data?.organization;

  useEffect(() => {
    if (org) {
      setForm({
        name: org.name ?? "",
        slug: org.slug ?? "",
        domain: org.domain ?? "",
      });
    }
  }, [org]);

  if (isLoading) {
    return (
      <div className="flex h-40 items-center justify-center">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    );
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    updateOrg.mutate({
      name: form.name,
      slug: form.slug,
      domain: form.domain,
    });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      <SettingsSection title="Organization Details">
        <div className="grid grid-cols-2 gap-4">
          <FieldGroup label="Organization Name">
            <input
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              className="h-9 w-full rounded-[10px] border border-neutral-200 px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10 dark:border-neutral-700 dark:bg-neutral-900"
            />
          </FieldGroup>
          <FieldGroup label="Slug">
            <input
              value={form.slug}
              onChange={(e) => setForm({ ...form, slug: e.target.value })}
              className="h-9 w-full rounded-[10px] border border-neutral-200 px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10 dark:border-neutral-700 dark:bg-neutral-900"
            />
          </FieldGroup>
        </div>
        <FieldGroup label="Domain">
          <input
            value={form.domain}
            onChange={(e) => setForm({ ...form, domain: e.target.value })}
            placeholder="yourcompany.com"
            className="h-9 w-full rounded-[10px] border border-neutral-200 px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10 dark:border-neutral-700 dark:bg-neutral-900"
          />
        </FieldGroup>
      </SettingsSection>

      <div className="flex items-center justify-end gap-2">
        {updateOrg.isSuccess && (
          <span className="flex items-center gap-1 text-xs text-emerald-600">
            <Check className="h-3 w-3" /> Saved
          </span>
        )}
        <button
          type="submit"
          disabled={updateOrg.isPending}
          className="h-8 rounded-lg bg-neutral-900 px-4 text-xs font-medium text-white hover:bg-neutral-800 disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900 dark:hover:bg-neutral-200"
        >
          {updateOrg.isPending ? "Saving..." : "Save Changes"}
        </button>
      </div>
    </form>
  );
}

// ── Profile Settings ────────────────────────────────────────────────────────

function ProfileSettings() {
  const user = useAuthStore((s) => s.user);
  const updateProfile = useUpdateProfile();
  const changePassword = useChangePassword();

  const [form, setForm] = useState({
    first_name: "",
    last_name: "",
    email: "",
    job_title: "",
  });
  const [passwords, setPasswords] = useState({
    current_password: "",
    new_password: "",
  });

  useEffect(() => {
    if (user) {
      setForm({
        first_name: user.first_name ?? "",
        last_name: user.last_name ?? "",
        email: user.email ?? "",
        job_title: user.job_title ?? "",
      });
    }
  }, [user]);

  const handleProfileSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    updateProfile.mutate({
      first_name: form.first_name,
      last_name: form.last_name,
      job_title: form.job_title,
    });
  };

  const handlePasswordSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    changePassword.mutate(passwords, {
      onSuccess: () => setPasswords({ current_password: "", new_password: "" }),
    });
  };

  return (
    <div className="space-y-6">
      <form onSubmit={handleProfileSubmit} className="space-y-6">
        <SettingsSection title="Personal Information">
          <div className="grid grid-cols-2 gap-4">
            <FieldGroup label="First Name">
              <input
                value={form.first_name}
                onChange={(e) => setForm({ ...form, first_name: e.target.value })}
                className="h-9 w-full rounded-[10px] border border-neutral-200 px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10 dark:border-neutral-700 dark:bg-neutral-900"
              />
            </FieldGroup>
            <FieldGroup label="Last Name">
              <input
                value={form.last_name}
                onChange={(e) => setForm({ ...form, last_name: e.target.value })}
                className="h-9 w-full rounded-[10px] border border-neutral-200 px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10 dark:border-neutral-700 dark:bg-neutral-900"
              />
            </FieldGroup>
          </div>
          <FieldGroup label="Email">
            <input
              type="email"
              value={form.email}
              disabled
              className="h-9 w-full rounded-[10px] border border-neutral-200 bg-muted/30 px-3 text-sm outline-none dark:border-neutral-700 dark:bg-neutral-800"
            />
          </FieldGroup>
          <FieldGroup label="Job Title">
            <input
              value={form.job_title}
              onChange={(e) => setForm({ ...form, job_title: e.target.value })}
              className="h-9 w-full rounded-[10px] border border-neutral-200 px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10 dark:border-neutral-700 dark:bg-neutral-900"
            />
          </FieldGroup>
        </SettingsSection>

        <div className="flex items-center justify-end gap-2">
          {updateProfile.isSuccess && (
            <span className="flex items-center gap-1 text-xs text-emerald-600">
              <Check className="h-3 w-3" /> Saved
            </span>
          )}
          <button
            type="submit"
            disabled={updateProfile.isPending}
            className="h-8 rounded-lg bg-neutral-900 px-4 text-xs font-medium text-white hover:bg-neutral-800 disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900 dark:hover:bg-neutral-200"
          >
            {updateProfile.isPending ? "Saving..." : "Update Profile"}
          </button>
        </div>
      </form>

      <form onSubmit={handlePasswordSubmit} className="space-y-6">
        <SettingsSection title="Change Password">
          <FieldGroup label="Current Password">
            <input
              type="password"
              value={passwords.current_password}
              onChange={(e) =>
                setPasswords({ ...passwords, current_password: e.target.value })
              }
              className="h-9 w-full rounded-[10px] border border-neutral-200 px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10 dark:border-neutral-700 dark:bg-neutral-900"
            />
          </FieldGroup>
          <FieldGroup label="New Password">
            <input
              type="password"
              value={passwords.new_password}
              onChange={(e) =>
                setPasswords({ ...passwords, new_password: e.target.value })
              }
              className="h-9 w-full rounded-[10px] border border-neutral-200 px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10 dark:border-neutral-700 dark:bg-neutral-900"
            />
          </FieldGroup>
          {changePassword.isError && (
            <p className="text-xs text-red-500">
              {(changePassword.error as Error)?.message ?? "Failed to change password"}
            </p>
          )}
          {changePassword.isSuccess && (
            <p className="text-xs text-emerald-600">Password updated successfully</p>
          )}
        </SettingsSection>

        <div className="flex justify-end">
          <button
            type="submit"
            disabled={changePassword.isPending || !passwords.current_password || !passwords.new_password}
            className="h-8 rounded-lg bg-neutral-900 px-4 text-xs font-medium text-white hover:bg-neutral-800 disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900 dark:hover:bg-neutral-200"
          >
            {changePassword.isPending ? "Updating..." : "Change Password"}
          </button>
        </div>
      </form>
    </div>
  );
}

// ── Other Tabs ──────────────────────────────────────────────────────────────

function TeamSettings() {
  const { data: membersData, isLoading } = useTeamMembers();
  const { data: rolesData } = useOrgRoles();
  const inviteMember = useInviteMember();
  const removeMember = useRemoveMember();
  const [showInvite, setShowInvite] = useState(false);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteFirstName, setInviteFirstName] = useState("");
  const [inviteLastName, setInviteLastName] = useState("");
  const [inviteRole, setInviteRole] = useState("member");

  const members = membersData?.data ?? [];
  const roles = rolesData?.data ?? [];

  const handleInvite = (e: React.FormEvent) => {
    e.preventDefault();
    if (!inviteEmail) return;
    inviteMember.mutate(
      { email: inviteEmail, first_name: inviteFirstName, last_name: inviteLastName, role: inviteRole },
      {
        onSuccess: () => {
          setInviteEmail("");
          setInviteFirstName("");
          setInviteLastName("");
          setInviteRole("member");
          setShowInvite(false);
        },
      }
    );
  };

  return (
    <div className="space-y-6">
      <SettingsSection title="Team Members">
        <div className="flex items-center justify-between">
          <p className="text-xs text-muted-foreground">
            {members.length} member{members.length !== 1 ? "s" : ""} in your organization
          </p>
          <button
            onClick={() => setShowInvite(!showInvite)}
            className="h-8 rounded-lg bg-neutral-900 px-3 text-xs font-medium text-white hover:bg-neutral-800 dark:bg-neutral-100 dark:text-neutral-900 dark:hover:bg-neutral-200"
          >
            Invite Member
          </button>
        </div>

        {showInvite && (
          <form onSubmit={handleInvite} className="flex items-end gap-2 rounded-lg border border-neutral-200 p-3 dark:border-neutral-700">
            <div className="space-y-1">
              <label className="text-xs font-medium text-neutral-700 dark:text-neutral-300">First Name</label>
              <input
                type="text"
                value={inviteFirstName}
                onChange={(e) => setInviteFirstName(e.target.value)}
                placeholder="First"
                className="h-8 w-full rounded-md border border-neutral-200 bg-white px-3 text-xs dark:border-neutral-700 dark:bg-neutral-800"
              />
            </div>
            <div className="space-y-1">
              <label className="text-xs font-medium text-neutral-700 dark:text-neutral-300">Last Name</label>
              <input
                type="text"
                value={inviteLastName}
                onChange={(e) => setInviteLastName(e.target.value)}
                placeholder="Last"
                className="h-8 w-full rounded-md border border-neutral-200 bg-white px-3 text-xs dark:border-neutral-700 dark:bg-neutral-800"
              />
            </div>
            <div className="flex-1 space-y-1">
              <label className="text-xs font-medium text-neutral-700 dark:text-neutral-300">Email</label>
              <input
                type="email"
                value={inviteEmail}
                onChange={(e) => setInviteEmail(e.target.value)}
                placeholder="colleague@company.com"
                className="h-8 w-full rounded-md border border-neutral-200 bg-white px-3 text-xs dark:border-neutral-700 dark:bg-neutral-800"
                required
              />
            </div>
            <div className="space-y-1">
              <label className="text-xs font-medium text-neutral-700 dark:text-neutral-300">Role</label>
              <select
                value={inviteRole}
                onChange={(e) => setInviteRole(e.target.value)}
                className="h-8 rounded-md border border-neutral-200 bg-white px-2 text-xs dark:border-neutral-700 dark:bg-neutral-800"
              >
                {(roles.length > 0 ? roles : [{ name: "admin" }, { name: "member" }, { name: "viewer" }]).map((r: { name: string }) => (
                  <option key={r.name} value={r.name}>{r.name}</option>
                ))}
              </select>
            </div>
            <button
              type="submit"
              disabled={inviteMember.isPending}
              className="h-8 rounded-lg bg-neutral-900 px-3 text-xs font-medium text-white hover:bg-neutral-800 disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900 dark:hover:bg-neutral-200"
            >
              {inviteMember.isPending ? "Sending..." : "Send Invite"}
            </button>
          </form>
        )}

        {isLoading ? (
          <div className="flex items-center gap-2 py-4 text-xs text-muted-foreground">
            <Loader2 className="h-3 w-3 animate-spin" /> Loading members...
          </div>
        ) : (
          <div className="divide-y divide-neutral-100 dark:divide-neutral-800">
            {members.map((member) => (
              <div key={member.id} className="flex items-center justify-between py-3">
                <div>
                  <p className="text-sm font-medium text-neutral-900 dark:text-neutral-100">
                    {member.first_name ? `${member.first_name} ${member.last_name}` : member.email}
                  </p>
                  {member.first_name && (
                    <p className="text-xs text-muted-foreground">{member.email}</p>
                  )}
                </div>
                <div className="flex items-center gap-2">
                  <span className="rounded-full bg-neutral-100 px-2 py-0.5 text-[10px] font-medium text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400">
                    {member.roles?.[0] ?? "member"}
                  </span>
                  <button
                    onClick={() => removeMember.mutate(member.id)}
                    className="rounded p-1 text-neutral-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
                    title="Remove member"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </SettingsSection>
    </div>
  );
}

function NotificationSettings() {
  const { data, isLoading } = useNotificationPreferences();
  const updatePref = useUpdateNotificationPreference();

  const prefs: Array<{ type: string; in_app: boolean; email: boolean }> = data?.data ?? [];

  const LABELS: Record<string, { label: string; description: string }> = {
    "pipeline.move": { label: "Sponsor stage changes", description: "When a sponsor moves pipeline stages" },
    "agent.complete": { label: "AI agent completions", description: "When AI agents finish tasks" },
    "outreach.reply": { label: "Outreach replies", description: "When a contact replies to outreach" },
    "deal.won": { label: "Deal won", description: "When a deal is marked as won" },
    "deal.lost": { label: "Deal lost", description: "When a deal is marked as lost" },
    "task.assigned": { label: "Task assigned", description: "When a task is assigned to you" },
    "team.invite": { label: "Team invites", description: "When someone invites you to a team" },
    "mention": { label: "Mentions", description: "When you are mentioned" },
    "webhook.failed": { label: "Webhook failures", description: "When a webhook delivery fails" },
    "system.alert": { label: "System alerts", description: "Important system notifications" },
  };

  const toggle = (type: string, channel: "in_app" | "email", current: boolean) => {
    updatePref.mutate({ type, [channel]: !current });
  };

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 py-8 text-xs text-muted-foreground">
        <Loader2 className="h-3 w-3 animate-spin" /> Loading preferences...
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <SettingsSection title="Notification Preferences">
        <div className="mb-2 grid grid-cols-[1fr_auto_auto] gap-x-6 text-[10px] font-medium uppercase tracking-wide text-muted-foreground">
          <span>Event</span>
          <span>In-app</span>
          <span>Email</span>
        </div>
        <div className="divide-y divide-neutral-100 dark:divide-neutral-800">
          {Object.entries(LABELS).map(([type, { label, description }]) => {
            const pref = prefs.find((p) => p.type === type);
            const inApp = pref?.in_app ?? true;
            const email = pref?.email ?? true;
            return (
              <div key={type} className="grid grid-cols-[1fr_auto_auto] items-center gap-x-6 py-3">
                <div>
                  <p className="text-sm font-medium text-neutral-900 dark:text-neutral-100">{label}</p>
                  <p className="text-xs text-muted-foreground">{description}</p>
                </div>
                <button
                  onClick={() => toggle(type, "in_app", inApp)}
                  className={`relative h-5 w-9 rounded-full transition-colors ${inApp ? "bg-neutral-900 dark:bg-neutral-100" : "bg-neutral-200 dark:bg-neutral-700"}`}
                >
                  <span className={`absolute top-0.5 h-4 w-4 rounded-full bg-white shadow transition-transform dark:bg-neutral-900 ${inApp ? "translate-x-4" : "translate-x-0.5"}`} />
                </button>
                <button
                  onClick={() => toggle(type, "email", email)}
                  className={`relative h-5 w-9 rounded-full transition-colors ${email ? "bg-neutral-900 dark:bg-neutral-100" : "bg-neutral-200 dark:bg-neutral-700"}`}
                >
                  <span className={`absolute top-0.5 h-4 w-4 rounded-full bg-white shadow transition-transform dark:bg-neutral-900 ${email ? "translate-x-4" : "translate-x-0.5"}`} />
                </button>
              </div>
            );
          })}
        </div>
      </SettingsSection>
    </div>
  );
}

function ImportSettings() {
  const importCompanies = useImportCompanies();
  const importContacts = useImportContacts();
  const importSponsors = useImportSponsors();
  const fileRef = useRef<HTMLInputElement>(null);
  const [entity, setEntity] = useState<"companies" | "contacts" | "sponsors">("companies");
  const [result, setResult] = useState<ImportResult | null>(null);

  const mutation = entity === "companies" ? importCompanies : entity === "contacts" ? importContacts : importSponsors;

  const handleFile = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setResult(null);
    mutation.mutate(file, {
      onSuccess: (data) => setResult(data),
    });
    if (fileRef.current) fileRef.current.value = "";
  };

  return (
    <div className="space-y-6">
      <SettingsSection title="Import Data">
        <p className="text-xs text-muted-foreground">
          Upload a CSV file to bulk-import records. The first row must be column headers.
          Headers are normalized (lowercased, spaces become underscores).
        </p>

        <div className="space-y-3">
          <div className="space-y-1">
            <label className="text-xs font-medium text-neutral-700 dark:text-neutral-300">Entity type</label>
            <select
              value={entity}
              onChange={(e) => { setEntity(e.target.value as typeof entity); setResult(null); }}
              className="h-8 w-48 rounded-md border border-neutral-200 bg-white px-2 text-xs dark:border-neutral-700 dark:bg-neutral-800"
            >
              <option value="companies">Companies</option>
              <option value="contacts">Contacts</option>
              <option value="sponsors">Sponsors</option>
            </select>
          </div>

          <div>
            <input ref={fileRef} type="file" accept=".csv" onChange={handleFile} className="hidden" />
            <button
              onClick={() => fileRef.current?.click()}
              disabled={mutation.isPending}
              className="flex items-center gap-1.5 h-8 rounded-lg bg-neutral-900 px-3 text-xs font-medium text-white hover:bg-neutral-800 disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900 dark:hover:bg-neutral-200"
            >
              {mutation.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <FileUp className="h-3.5 w-3.5" />}
              {mutation.isPending ? "Importing..." : "Upload CSV"}
            </button>
          </div>
        </div>

        {result && (
          <div className="mt-4 rounded-lg border border-neutral-200 p-4 dark:border-neutral-700">
            <div className="flex items-center gap-4 text-xs">
              <span className="flex items-center gap-1 text-green-600">
                <CheckCircle2 className="h-3.5 w-3.5" /> {result.inserted} imported
              </span>
              {result.errors > 0 && (
                <span className="flex items-center gap-1 text-red-600">
                  <XCircle className="h-3.5 w-3.5" /> {result.errors} failed
                </span>
              )}
            </div>
            {result.row_errors && result.row_errors.length > 0 && (
              <div className="mt-3 max-h-40 overflow-y-auto space-y-1">
                {result.row_errors.map((re, i) => (
                  <p key={i} className="text-[11px] text-red-600">
                    Row {re.row}: {re.error}
                  </p>
                ))}
              </div>
            )}
          </div>
        )}

        {mutation.isError && (
          <p className="text-xs text-red-600">Upload failed: {(mutation.error as Error).message}</p>
        )}
      </SettingsSection>

      <SettingsSection title="CSV Column Reference">
        <div className="space-y-3 text-xs text-muted-foreground">
          <div>
            <p className="font-medium text-neutral-700 dark:text-neutral-300">Companies</p>
            <p>name (required), domain, website, description, employee_count, annual_revenue, headquarters, phone, linkedin_url, twitter_url, source, status, founded_year</p>
          </div>
          <div>
            <p className="font-medium text-neutral-700 dark:text-neutral-300">Contacts</p>
            <p>first_name or name (required), last_name, email, phone, title, department, linkedin_url, notes, status, company_id or company_name</p>
          </div>
          <div>
            <p className="font-medium text-neutral-700 dark:text-neutral-300">Sponsors</p>
            <p>campaign_id (required), company_id or company_name (required), stage, tier, notes, deal_value, probability</p>
          </div>
        </div>
      </SettingsSection>
    </div>
  );
}

function ApiSettings() {
  return (
    <div className="space-y-6">
      <SettingsSection title="API Keys">
        <p className="text-xs text-muted-foreground">
          API keys allow external systems to access SponsorOS programmatically.
        </p>
        <button className="h-8 rounded-lg border border-neutral-200 px-3 text-xs font-medium hover:bg-neutral-50 dark:border-neutral-700 dark:hover:bg-neutral-800">
          Generate New Key
        </button>
      </SettingsSection>

      <SettingsSection title="Integrations">
        <p className="text-xs text-muted-foreground">
          Connect external services to automate workflows.
        </p>
        <div className="grid grid-cols-3 gap-3">
          {["Slack", "HubSpot", "Salesforce", "Google Calendar", "Zapier"].map((name) => (
            <div key={name} className="flex items-center justify-between rounded-lg border border-border p-3">
              <span className="text-xs font-medium">{name}</span>
              <button className="text-[10px] font-medium text-primary hover:underline">
                Connect
              </button>
            </div>
          ))}
        </div>
      </SettingsSection>
    </div>
  );
}

// ── Shared Components ───────────────────────────────────────────────────────

function SettingsSection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-border p-5 space-y-4">
      <h3 className="text-sm font-medium">{title}</h3>
      {children}
    </div>
  );
}

function FieldGroup({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label className="text-[11px] font-medium text-muted-foreground mb-1 block">{label}</label>
      {children}
    </div>
  );
}

function ToggleRow({
  label,
  description,
  enabled,
  onToggle,
}: {
  label: string;
  description: string;
  enabled: boolean;
  onToggle: () => void;
}) {
  return (
    <div className="flex items-center justify-between py-1">
      <div>
        <p className="text-xs font-medium">{label}</p>
        <p className="text-[10px] text-muted-foreground">{description}</p>
      </div>
      <button
        type="button"
        onClick={onToggle}
        className={`relative h-5 w-9 rounded-full transition-colors ${
          enabled ? "bg-neutral-900 dark:bg-neutral-100" : "bg-neutral-200 dark:bg-neutral-700"
        }`}
      >
        <div
          className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition-transform dark:bg-neutral-900 ${
            enabled ? "translate-x-4" : "translate-x-0.5"
          }`}
        />
      </button>
    </div>
  );
}
