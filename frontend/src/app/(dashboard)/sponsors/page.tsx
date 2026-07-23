"use client";

import { useState } from "react";
import { Plus, GripVertical, Search, Filter } from "lucide-react";
import { cn } from "@/lib/utils";
import { useSponsors, useCreateSponsor } from "@/queries/sponsors";
import { useCompanies } from "@/queries/companies";
import { useCampaigns } from "@/queries/campaigns";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import type { Sponsor } from "@/types";

const STAGES = [
  { id: "prospect", name: "Prospect", color: "border-t-neutral-400" },
  { id: "contacted", name: "Contacted", color: "border-t-blue-400" },
  { id: "meeting", name: "Meeting", color: "border-t-indigo-400" },
  { id: "proposal", name: "Proposal", color: "border-t-violet-400" },
  { id: "negotiation", name: "Negotiation", color: "border-t-amber-400" },
  { id: "closed_won", name: "Won", color: "border-t-emerald-400" },
];

export default function SponsorsPage() {
  const [showCreate, setShowCreate] = useState(false);
  const [stageFilter, setStageFilter] = useState<string | undefined>();
  const { data, isLoading } = useSponsors({ stage: stageFilter });

  const sponsors: Sponsor[] = (data as any)?.sponsors || [];

  const sponsorsByStage = STAGES.reduce<Record<string, Sponsor[]>>((acc, stage) => {
    acc[stage.id] = sponsors.filter((s) => s.stage === stage.id);
    return acc;
  }, {});

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Pipeline</h1>
          <p className="text-sm text-muted-foreground">
            Drag sponsors between stages to update their status
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="flex h-8 items-center gap-1.5 rounded-lg bg-primary px-3 text-xs font-medium text-primary-foreground hover:bg-primary/90"
        >
          <Plus className="h-3.5 w-3.5" />
          Add Sponsor
        </button>
      </div>

      {isLoading ? (
        <div className="flex gap-3 overflow-x-auto pb-4">
          {STAGES.map((stage) => (
            <div
              key={stage.id}
              className="w-[280px] shrink-0 animate-pulse rounded-xl border border-border bg-muted/20 h-[320px]"
            />
          ))}
        </div>
      ) : sponsors.length === 0 && !stageFilter ? (
        <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-12">
          <p className="text-sm text-muted-foreground">No sponsors in the pipeline</p>
          <button
            onClick={() => setShowCreate(true)}
            className="mt-2 text-xs font-medium text-primary hover:underline"
          >
            Add your first sponsor
          </button>
        </div>
      ) : (
        <div className="flex gap-3 overflow-x-auto pb-4">
          {STAGES.map((stage) => (
            <div
              key={stage.id}
              className="flex w-[280px] shrink-0 flex-col rounded-xl border border-border bg-muted/20"
            >
              <div className={cn("border-t-2 rounded-t-xl px-3 py-2.5", stage.color)}>
                <div className="flex items-center justify-between">
                  <h3 className="text-xs font-medium">{stage.name}</h3>
                  <span className="flex h-5 min-w-5 items-center justify-center rounded-full bg-muted px-1.5 text-[10px] font-medium">
                    {sponsorsByStage[stage.id]?.length || 0}
                  </span>
                </div>
              </div>

              <div className="flex flex-1 flex-col gap-2 p-2 min-h-[200px]">
                {(sponsorsByStage[stage.id] || []).map((sponsor) => {
                  const daysInStage = sponsor.stage_entered_at
                    ? Math.floor(
                        (Date.now() - new Date(sponsor.stage_entered_at).getTime()) /
                          (1000 * 60 * 60 * 24)
                      )
                    : 0;

                  return (
                    <div
                      key={sponsor.id}
                      className="group cursor-grab rounded-lg border border-border bg-card p-3 shadow-sm transition-shadow hover:shadow-md active:cursor-grabbing"
                    >
                      <div className="flex items-start justify-between">
                        <div className="flex items-center gap-2">
                          <div className="flex h-7 w-7 items-center justify-center rounded-md bg-muted text-[10px] font-medium">
                            {sponsor.company?.name?.[0] || "?"}
                          </div>
                          <div>
                            <p className="text-sm font-medium">
                              {sponsor.company?.name || "Unknown"}
                            </p>
                            {sponsor.tier && (
                              <span className="text-[10px] text-muted-foreground">
                                {sponsor.tier}
                              </span>
                            )}
                          </div>
                        </div>
                        <GripVertical className="h-4 w-4 text-muted-foreground/0 transition-colors group-hover:text-muted-foreground" />
                      </div>
                      <div className="mt-2.5 flex items-center justify-between">
                        <span className="text-xs font-medium">
                          {sponsor.deal_value
                            ? `$${sponsor.deal_value.toLocaleString()}`
                            : "—"}
                        </span>
                        <span className="text-[10px] text-muted-foreground">
                          {daysInStage}d in stage
                        </span>
                      </div>
                    </div>
                  );
                })}

                <button
                  onClick={() => setShowCreate(true)}
                  className="flex items-center justify-center gap-1 rounded-lg border border-dashed border-border py-2 text-xs text-muted-foreground transition-colors hover:border-foreground/20 hover:text-foreground"
                >
                  <Plus className="h-3 w-3" />
                  Add
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      <CreateSponsorDialog open={showCreate} onOpenChange={setShowCreate} />
    </div>
  );
}

function CreateSponsorDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (v: boolean) => void }) {
  const createSponsor = useCreateSponsor();
  const { data: companiesData } = useCompanies(1, 100);
  const { data: campaignsData } = useCampaigns();
  const companies = (companiesData as any)?.companies || [];
  const campaigns = (campaignsData as any)?.campaigns || [];

  const [form, setForm] = useState({
    company_id: "",
    campaign_id: "",
    stage: "prospect",
    deal_value: "",
    tier: "",
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    createSponsor.mutate(
      {
        company_id: form.company_id,
        campaign_id: form.campaign_id,
        stage: form.stage,
        deal_value: form.deal_value ? Number(form.deal_value) : undefined,
        tier: form.tier || undefined,
      },
      {
        onSuccess: () => {
          onOpenChange(false);
          setForm({ company_id: "", campaign_id: "", stage: "prospect", deal_value: "", tier: "" });
        },
      }
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Sponsor</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4 mt-4">
          <select
            value={form.company_id}
            onChange={(e) => setForm({ ...form, company_id: e.target.value })}
            className="h-9 w-full rounded-[10px] border border-neutral-200 px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10"
            required
          >
            <option value="">Select company...</option>
            {companies.map((c: any) => (
              <option key={c.id} value={c.id}>{c.name}</option>
            ))}
          </select>
          <select
            value={form.campaign_id}
            onChange={(e) => setForm({ ...form, campaign_id: e.target.value })}
            className="h-9 w-full rounded-[10px] border border-neutral-200 px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10"
            required
          >
            <option value="">Select campaign...</option>
            {campaigns.map((c: any) => (
              <option key={c.id} value={c.id}>{c.name}</option>
            ))}
          </select>
          <div className="grid grid-cols-2 gap-3">
            <select
              value={form.stage}
              onChange={(e) => setForm({ ...form, stage: e.target.value })}
              className="h-9 rounded-[10px] border border-neutral-200 px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10"
            >
              {STAGES.map((s) => (
                <option key={s.id} value={s.id}>{s.name}</option>
              ))}
            </select>
            <input
              placeholder="Deal value"
              type="number"
              value={form.deal_value}
              onChange={(e) => setForm({ ...form, deal_value: e.target.value })}
              className="h-9 rounded-[10px] border border-neutral-200 px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10"
            />
          </div>
          <input
            placeholder="Tier (e.g. Gold, Silver, Platinum)"
            value={form.tier}
            onChange={(e) => setForm({ ...form, tier: e.target.value })}
            className="h-9 w-full rounded-[10px] border border-neutral-200 px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10"
          />
          <DialogFooter>
            <button
              type="button"
              onClick={() => onOpenChange(false)}
              className="h-8 rounded-lg border border-neutral-200 px-3 text-xs font-medium hover:bg-neutral-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={createSponsor.isPending}
              className="h-8 rounded-lg bg-neutral-900 px-3 text-xs font-medium text-white hover:bg-neutral-800 disabled:opacity-50"
            >
              {createSponsor.isPending ? "Adding..." : "Add Sponsor"}
            </button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
