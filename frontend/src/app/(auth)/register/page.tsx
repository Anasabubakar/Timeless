"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { useAuthStore } from "@/stores/auth";
import { useJoinOrganization } from "@/queries/auth";
import { motion, AnimatePresence } from "motion/react";

type Step = "org" | "create" | "join";

function slugify(value: string) {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "");
}

export default function RegisterPage() {
  const router = useRouter();
  const { setAuth } = useAuthStore();
  const joinOrg = useJoinOrganization();

  const [step, setStep] = useState<Step>("org");
  const [orgName, setOrgName] = useState("");
  const [checking, setChecking] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const [form, setForm] = useState({
    first_name: "",
    last_name: "",
    email: "",
    password: "",
    org_password: "",
  });

  async function handleOrgSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!orgName.trim()) return;
    setError("");
    setChecking(true);
    try {
      const result = await api.get<{ exists: boolean; name?: string; slug?: string }>(
        `/auth/organizations/lookup?name=${encodeURIComponent(orgName)}`
      );
      setStep(result.exists ? "join" : "create");
    } catch (err: any) {
      setError(err.message || "Could not check that organization name — try again.");
    } finally {
      setChecking(false);
    }
  }

  function chooseAnotherOrg() {
    setStep("org");
    setError("");
    setForm((prev) => ({ ...prev, org_password: "" }));
  }

  async function handleCreateSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const res = await api.post<{ user: any; tokens: any }>("/auth/register", {
        first_name: form.first_name,
        last_name: form.last_name,
        email: form.email,
        password: form.password,
        org_name: orgName,
        org_slug: slugify(orgName),
        org_password: form.org_password,
      });
      setAuth(res.user, res.tokens);
      router.push("/onboarding");
    } catch (err: any) {
      setError(err.message || "Registration failed");
    } finally {
      setLoading(false);
    }
  }

  async function handleJoinSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    joinOrg.mutate(
      {
        first_name: form.first_name,
        last_name: form.last_name,
        email: form.email,
        password: form.password,
        org_slug: slugify(orgName),
        org_password: form.org_password,
      },
      {
        onSuccess: () => router.push("/onboarding"),
        onError: (err: any) => setError(err.message || "Could not join that organization"),
      }
    );
  }

  return (
    <motion.div className="rounded-xl border border-border bg-card p-8 shadow-sm" initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.3 }}>
      <div className="mb-6 text-center">
        <h1 className="text-xl font-semibold">
          {step === "org" && "Create your account"}
          {step === "create" && "Name your organization"}
          {step === "join" && "Join your team"}
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {step === "org" && "Start by telling us your organization's name"}
          {step === "create" && `"${orgName}" is new — set it up`}
          {step === "join" && `"${orgName}" already exists on Timeless`}
        </p>
      </div>

      <AnimatePresence mode="wait">
        {step === "org" && (
          <motion.form
            key="org"
            initial={{ opacity: 0, x: -8 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: 8 }}
            transition={{ duration: 0.15 }}
            onSubmit={handleOrgSubmit}
            className="space-y-4"
          >
            {error && (
              <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
                {error}
              </div>
            )}
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Organization name</label>
              <input
                type="text"
                required
                autoFocus
                value={orgName}
                onChange={(e) => setOrgName(e.target.value)}
                className="flex h-9 w-full rounded-lg border border-input bg-background px-3 text-sm transition-colors focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring"
                placeholder="Acme Inc"
              />
              <p className="text-xs text-muted-foreground">
                New organization? We'll help you create it. Already using Timeless with your team? We'll help you join.
              </p>
            </div>
            <button
              type="submit"
              disabled={checking || !orgName.trim()}
              className="flex h-9 w-full items-center justify-center rounded-lg bg-primary text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
            >
              {checking ? "Checking..." : "Continue"}
            </button>
          </motion.form>
        )}

        {step === "create" && (
          <motion.form
            key="create"
            initial={{ opacity: 0, x: 8 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: -8 }}
            transition={{ duration: 0.15 }}
            onSubmit={handleCreateSubmit}
            className="space-y-4"
          >
            {error && (
              <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
                {error}
              </div>
            )}

            <div className="rounded-lg border border-border bg-muted/30 px-3 py-2 text-sm">
              Creating <strong>{orgName}</strong>.{" "}
              <button type="button" onClick={chooseAnotherOrg} className="text-primary underline underline-offset-4">
                Not this org?
              </button>
            </div>

            <div className="space-y-1.5">
              <label className="text-sm font-medium">Organization password</label>
              <input
                type="password"
                required
                minLength={8}
                autoFocus
                value={form.org_password}
                onChange={(e) => setForm({ ...form, org_password: e.target.value })}
                className="flex h-9 w-full rounded-lg border border-input bg-background px-3 text-sm transition-colors focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring"
                placeholder="Min 8 characters"
              />
              <p className="text-xs text-muted-foreground">
                Share this with teammates — they'll need it to join {orgName || "your organization"}.
              </p>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <label className="text-sm font-medium">First name</label>
                <input
                  type="text"
                  required
                  value={form.first_name}
                  onChange={(e) => setForm({ ...form, first_name: e.target.value })}
                  className="flex h-9 w-full rounded-lg border border-input bg-background px-3 text-sm transition-colors focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring"
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium">Last name</label>
                <input
                  type="text"
                  required
                  value={form.last_name}
                  onChange={(e) => setForm({ ...form, last_name: e.target.value })}
                  className="flex h-9 w-full rounded-lg border border-input bg-background px-3 text-sm transition-colors focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring"
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <label className="text-sm font-medium">Email</label>
              <input
                type="email"
                required
                value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })}
                className="flex h-9 w-full rounded-lg border border-input bg-background px-3 text-sm transition-colors focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring"
                placeholder="you@company.com"
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-sm font-medium">Your password</label>
              <input
                type="password"
                required
                minLength={8}
                value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })}
                className="flex h-9 w-full rounded-lg border border-input bg-background px-3 text-sm transition-colors focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring"
                placeholder="Min 8 characters"
              />
            </div>

            <button
              type="submit"
              disabled={loading}
              className="flex h-9 w-full items-center justify-center rounded-lg bg-primary text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
            >
              {loading ? "Creating account..." : "Create organization & account"}
            </button>
          </motion.form>
        )}

        {step === "join" && (
          <motion.form
            key="join"
            initial={{ opacity: 0, x: 8 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: -8 }}
            transition={{ duration: 0.15 }}
            onSubmit={handleJoinSubmit}
            className="space-y-4"
          >
            {error && (
              <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
                {error}
              </div>
            )}

            <div className="rounded-lg border border-border bg-muted/30 px-3 py-2 text-sm">
              Joining <strong>{orgName}</strong>.{" "}
              <button type="button" onClick={chooseAnotherOrg} className="text-primary underline underline-offset-4">
                Wrong org? Try again
              </button>
            </div>

            <div className="space-y-1.5">
              <label className="text-sm font-medium">Organization password</label>
              <input
                type="password"
                required
                autoFocus
                value={form.org_password}
                onChange={(e) => setForm({ ...form, org_password: e.target.value })}
                className="flex h-9 w-full rounded-lg border border-input bg-background px-3 text-sm transition-colors focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring"
                placeholder="Ask an existing member for this"
              />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <label className="text-sm font-medium">First name</label>
                <input
                  type="text"
                  required
                  value={form.first_name}
                  onChange={(e) => setForm({ ...form, first_name: e.target.value })}
                  className="flex h-9 w-full rounded-lg border border-input bg-background px-3 text-sm transition-colors focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring"
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium">Last name</label>
                <input
                  type="text"
                  required
                  value={form.last_name}
                  onChange={(e) => setForm({ ...form, last_name: e.target.value })}
                  className="flex h-9 w-full rounded-lg border border-input bg-background px-3 text-sm transition-colors focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring"
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <label className="text-sm font-medium">Email</label>
              <input
                type="email"
                required
                value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })}
                className="flex h-9 w-full rounded-lg border border-input bg-background px-3 text-sm transition-colors focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring"
                placeholder="you@company.com"
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-sm font-medium">Your password</label>
              <input
                type="password"
                required
                minLength={8}
                value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })}
                className="flex h-9 w-full rounded-lg border border-input bg-background px-3 text-sm transition-colors focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring"
                placeholder="Min 8 characters"
              />
            </div>

            <button
              type="submit"
              disabled={joinOrg.isPending}
              className="flex h-9 w-full items-center justify-center rounded-lg bg-primary text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
            >
              {joinOrg.isPending ? "Joining..." : "Join organization"}
            </button>
          </motion.form>
        )}
      </AnimatePresence>

      <p className="mt-4 text-center text-sm text-muted-foreground">
        Already have an account?{" "}
        <Link href="/login" className="text-foreground underline underline-offset-4 hover:text-primary">
          Sign in
        </Link>
      </p>
    </motion.div>
  );
}
