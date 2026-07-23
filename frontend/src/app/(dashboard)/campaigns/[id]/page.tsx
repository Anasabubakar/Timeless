"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { ArrowLeft, Plus, Users, DollarSign, Target, Calendar } from "lucide-react";
import Link from "next/link";
import { useCampaign } from "@/queries/campaigns";
import { useSponsors, useUpdateSponsorStage } from "@/queries/sponsors";
import type { Sponsor } from "@/types";

const DEFAULT_STAGES = ["prospect", "contacted", "meeting", "proposal", "negotiation", "closed_won", "closed_lost"];

const STAGE_LABELS: Record<string, string> = {
  prospect: "Prospect",
  contacted: "Contacted",
  meeting: "Meeting",
  proposal: "Proposal",
  negotiation: "Negotiation",
  closed_won: "Won",
  closed_lost: "Lost",
};

export default function CampaignDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { data: campaign, isLoading } = useCampaign(id);
  const { data: sponsorsData } = useSponsors({ campaign_id: id });

  const sponsors: Sponsor[] = (sponsorsData as any)?.sponsors || [];
  const stages = (campaign as any)?.pipeline_stages || DEFAULT_STAGES;

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="h-8 w-48 animate-pulse rounded bg-muted" />
        <div className="h-64 animate-pulse rounded-xl bg-muted/30" />
      </div>
    );
  }

  if (!campaign) {
    return (
      <div className="flex flex-col items-center py-12">
        <p className="text-sm text-muted-foreground">Campaign not found</p>
        <Link href="/campaigns" className="mt-2 text-xs text-primary hover:underline">
          Back to campaigns
        </Link>
      </div>
    );
  }

  const progress = (campaign as any).goal_amount
    ? Math.round(((campaign as any).raised_amount / (campaign as any).goal_amount) * 100)
    : 0;

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Link
          href="/campaigns"
          className="flex h-7 w-7 items-center justify-center rounded-md hover:bg-accent"
        >
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <div className="flex-1">
          <h1 className="text-xl font-semibold tracking-tight">{(campaign as any).name}</h1>
          {(campaign as any).description && (
            <p className="text-sm text-muted-foreground">{(campaign as any).description}</p>
          )}
        </div>
        <span
          className={`rounded-full px-2.5 py-0.5 text-[11px] font-medium ${
            (campaign as any).status === "active"
              ? "bg-emerald-50 text-emerald-700"
              : "bg-neutral-100 text-neutral-600"
          }`}
        >
          {(campaign as any).status}
        </span>
      </div>

      <div className="grid grid-cols-4 gap-3">
        <StatCard icon={Users} label="Sponsors" value={String(sponsors.length)} />
        <StatCard
          icon={DollarSign}
          label="Raised"
          value={`$${((campaign as any).raised_amount || 0).toLocaleString()}`}
        />
        <StatCard icon={Target} label="Progress" value={`${progress}%`} />
        <StatCard
          icon={Calendar}
          label="Ends"
          value={
            (campaign as any).end_date
              ? new Date((campaign as any).end_date).toLocaleDateString()
              : "No date"
          }
        />
      </div>

      <div>
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-medium">Pipeline</h2>
          <button className="flex h-7 items-center gap-1 rounded-md border border-input px-2 text-[11px] font-medium hover:bg-accent">
            <Plus className="h-3 w-3" />
            Add Sponsor
          </button>
        </div>
        <KanbanBoard stages={stages} sponsors={sponsors} />
      </div>
    </div>
  );
}

function StatCard({ icon: Icon, label, value }: { icon: any; label: string; value: string }) {
  return (
    <div className="rounded-xl border border-border bg-card p-4">
      <div className="flex items-center gap-2 text-muted-foreground">
        <Icon className="h-3.5 w-3.5" />
        <span className="text-[11px]">{label}</span>
      </div>
      <p className="mt-1 text-lg font-semibold">{value}</p>
    </div>
  );
}

function KanbanBoard({ stages, sponsors }: { stages: string[]; sponsors: Sponsor[] }) {
  const updateStage = useUpdateSponsorStage("");

  const sponsorsByStage = stages.reduce<Record<string, Sponsor[]>>((acc, stage) => {
    acc[stage] = sponsors.filter((s) => s.stage === stage);
    return acc;
  }, {});

  return (
    <div className="flex gap-3 overflow-x-auto pb-4">
      {stages.filter(s => s !== "closed_lost").map((stage) => (
        <div key={stage} className="w-64 shrink-0">
          <div className="mb-2 flex items-center justify-between">
            <span className="text-xs font-medium text-muted-foreground">
              {STAGE_LABELS[stage] || stage}
            </span>
            <span className="text-[10px] text-muted-foreground">
              {sponsorsByStage[stage]?.length || 0}
            </span>
          </div>
          <div className="space-y-2 rounded-lg bg-neutral-50 p-2 min-h-[200px]">
            {(sponsorsByStage[stage] || []).map((sponsor) => (
              <div
                key={sponsor.id}
                className="rounded-lg border border-border bg-white p-3 shadow-sm"
              >
                <p className="text-xs font-medium">
                  {sponsor.company?.name || "Unknown Company"}
                </p>
                {sponsor.deal_value && (
                  <p className="mt-1 text-[11px] text-muted-foreground">
                    ${sponsor.deal_value.toLocaleString()}
                  </p>
                )}
                {sponsor.tier && (
                  <span className="mt-1.5 inline-block rounded-full bg-neutral-100 px-1.5 py-0.5 text-[9px] font-medium">
                    {sponsor.tier}
                  </span>
                )}
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}
