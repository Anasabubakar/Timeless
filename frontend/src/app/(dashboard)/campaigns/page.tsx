"use client";

import { useState } from "react";
import { Plus, Search, Filter } from "lucide-react";
import { motion } from "motion/react";
import { useNewParam } from "@/hooks/use-new-param";
import { useCampaigns, useCreateCampaign } from "@/queries/campaigns";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import type { Campaign } from "@/types";

export default function CampaignsPage() {
  const [search, setSearch] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  useNewParam(() => setShowCreate(true));
  const { data, isLoading } = useCampaigns();

  const campaigns: Campaign[] = (data as any)?.campaigns || [];

  return (
    <motion.div className="space-y-6" initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.3 }}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Campaigns</h1>
          <p className="text-sm text-muted-foreground">
            Manage your sponsorship campaigns and track progress
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="flex h-8 items-center gap-1.5 rounded-lg bg-primary px-3 text-xs font-medium text-primary-foreground hover:bg-primary/90"
        >
          <Plus className="h-3.5 w-3.5" />
          New Campaign
        </button>
      </div>

      <div className="flex items-center gap-2">
        <div className="flex h-8 flex-1 items-center gap-2 rounded-lg border border-input bg-background px-3">
          <Search className="h-3.5 w-3.5 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search campaigns..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
          />
        </div>
        <button className="flex h-8 items-center gap-1.5 rounded-lg border border-input px-3 text-xs text-muted-foreground hover:bg-accent">
          <Filter className="h-3.5 w-3.5" />
          Filter
        </button>
      </div>

      {isLoading ? (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="h-36 animate-pulse rounded-xl border border-border bg-muted/30" />
          ))}
        </div>
      ) : campaigns.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-12">
          <p className="text-sm text-muted-foreground">No campaigns yet</p>
          <button
            onClick={() => setShowCreate(true)}
            className="mt-2 text-xs font-medium text-primary hover:underline"
          >
            Create your first campaign
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {campaigns
            .filter((c) => c.name.toLowerCase().includes(search.toLowerCase()))
            .map((campaign, i) => {
              const progress = campaign.goal_amount
                ? Math.round((campaign.raised_amount / campaign.goal_amount) * 100)
                : 0;

              return (
                <motion.div
                  key={campaign.id}
                  className="rounded-xl border border-border bg-card p-5 transition-shadow hover:shadow-sm"
                  initial={{ opacity: 0, y: 12 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.25, delay: i * 0.05 }}
                >
                  <div className="flex items-start justify-between">
                    <div>
                      <h3 className="text-sm font-medium">{campaign.name}</h3>
                      <p className="mt-0.5 text-xs text-muted-foreground">
                        {campaign.sponsors?.length || 0} sponsors
                      </p>
                    </div>
                    <span
                      className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                        campaign.status === "active"
                          ? "bg-emerald-50 text-emerald-700"
                          : campaign.status === "completed"
                          ? "bg-blue-50 text-blue-700"
                          : "bg-neutral-100 text-neutral-600"
                      }`}
                    >
                      {campaign.status}
                    </span>
                  </div>

                  {campaign.goal_amount && (
                    <div className="mt-4">
                      <div className="flex items-center justify-between text-xs">
                        <span className="text-muted-foreground">
                          ${campaign.raised_amount.toLocaleString()} / $
                          {campaign.goal_amount.toLocaleString()}
                        </span>
                        <span className="font-medium">{progress}%</span>
                      </div>
                      <div className="mt-1.5 h-1.5 overflow-hidden rounded-full bg-muted">
                        <div
                          className="h-full rounded-full bg-primary transition-all"
                          style={{ width: `${Math.min(progress, 100)}%` }}
                        />
                      </div>
                    </div>
                  )}

                  {campaign.end_date && (
                    <p className="mt-3 text-[10px] text-muted-foreground">
                      Ends {new Date(campaign.end_date).toLocaleDateString()}
                    </p>
                  )}
                </motion.div>
              );
            })}
        </div>
      )}

      <CreateCampaignDialog open={showCreate} onOpenChange={setShowCreate} />
    </motion.div>
  );
}

function CreateCampaignDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (v: boolean) => void }) {
  const createCampaign = useCreateCampaign();
  const [form, setForm] = useState({
    name: "",
    description: "",
    goal_amount: "",
    start_date: "",
    end_date: "",
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    createCampaign.mutate(
      {
        name: form.name,
        description: form.description,
        goal_amount: form.goal_amount ? Number(form.goal_amount) : undefined,
        start_date: form.start_date || undefined,
        end_date: form.end_date || undefined,
      },
      {
        onSuccess: () => {
          onOpenChange(false);
          setForm({ name: "", description: "", goal_amount: "", start_date: "", end_date: "" });
        },
      }
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>New Campaign</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4 mt-4">
          <input
            placeholder="Campaign name"
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
          <input
            placeholder="Goal amount"
            type="number"
            value={form.goal_amount}
            onChange={(e) => setForm({ ...form, goal_amount: e.target.value })}
            className="h-9 w-full rounded-[10px] border border-neutral-200 bg-background px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10 dark:border-neutral-700"
          />
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div>
              <label className="text-[11px] text-neutral-500 dark:text-neutral-400 mb-1 block">Start date</label>
              <input
                type="date"
                value={form.start_date}
                onChange={(e) => setForm({ ...form, start_date: e.target.value })}
                className="h-9 w-full rounded-[10px] border border-neutral-200 bg-background px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10 dark:border-neutral-700"
              />
            </div>
            <div>
              <label className="text-[11px] text-neutral-500 dark:text-neutral-400 mb-1 block">End date</label>
              <input
                type="date"
                value={form.end_date}
                onChange={(e) => setForm({ ...form, end_date: e.target.value })}
                className="h-9 w-full rounded-[10px] border border-neutral-200 bg-background px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10 dark:border-neutral-700"
              />
            </div>
          </div>
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
              disabled={createCampaign.isPending}
              className="h-8 rounded-lg bg-neutral-900 px-3 text-xs font-medium text-white hover:bg-neutral-800 disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900 dark:hover:bg-neutral-200"
            >
              {createCampaign.isPending ? "Creating..." : "Create"}
            </button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
