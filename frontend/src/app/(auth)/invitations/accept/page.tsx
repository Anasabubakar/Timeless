"use client";

import { Suspense, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { motion } from "motion/react";
import { useAcceptInvitation } from "@/queries/invitations";

function AcceptInvitationContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get("token") || "";
  const acceptInvitation = useAcceptInvitation();

  const [form, setForm] = useState({ first_name: "", last_name: "", password: "" });
  const [error, setError] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    acceptInvitation.mutate(
      { token, ...form },
      {
        onSuccess: () => router.push("/dashboard"),
        onError: (err: any) => setError(err.message || "This invitation link is invalid or has expired"),
      }
    );
  }

  if (!token) {
    return (
      <motion.div className="rounded-xl border border-border bg-card p-8 shadow-sm" initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.3 }}>
        <div className="text-center">
          <h1 className="text-xl font-semibold">Invalid invitation link</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            This link is missing its invitation token. Ask whoever invited you to resend it.
          </p>
        </div>
      </motion.div>
    );
  }

  return (
    <motion.div className="rounded-xl border border-border bg-card p-8 shadow-sm" initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.3 }}>
      <div className="mb-6 text-center">
        <h1 className="text-xl font-semibold">Accept your invitation</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Set a password to finish joining your team on Timeless
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        {error && (
          <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
            {error}
          </div>
        )}

        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1.5">
            <label className="text-sm font-medium">First name</label>
            <input
              type="text"
              required
              autoFocus
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
          <label className="text-sm font-medium">Password</label>
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
          disabled={acceptInvitation.isPending}
          className="flex h-9 w-full items-center justify-center rounded-lg bg-primary text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
        >
          {acceptInvitation.isPending ? "Joining..." : "Accept & join"}
        </button>
      </form>
    </motion.div>
  );
}

export default function AcceptInvitationPage() {
  return (
    <Suspense fallback={null}>
      <AcceptInvitationContent />
    </Suspense>
  );
}
