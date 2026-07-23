"use client";

import { useState } from "react";
import { FileText, Plus, Sparkles, ExternalLink } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Select, Textarea } from "@/components/ui/select";
import { useProposals, useGenerateProposal } from "@/queries/proposals";
import { useSponsors } from "@/queries/sponsors";
import type { Proposal } from "@/types";

const STATUS_COLORS: Record<string, string> = {
  draft: "bg-neutral-100 text-neutral-700",
  sent: "bg-blue-100 text-blue-700",
  viewed: "bg-purple-100 text-purple-700",
  accepted: "bg-green-100 text-green-700",
  rejected: "bg-red-100 text-red-700",
};

function GenerateDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [sponsorId, setSponsorId] = useState("");
  const [tone, setTone] = useState("professional");
  const [packageTier, setPackageTier] = useState("");
  const [customNotes, setCustomNotes] = useState("");

  const { data: sponsorsData } = useSponsors();
  const generate = useGenerateProposal();

  const sponsors = sponsorsData?.data ?? [];

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!sponsorId) return;
    await generate.mutateAsync({ sponsor_id: sponsorId, tone, package_tier: packageTier, custom_notes: customNotes });
    onClose();
  }

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="bg-white rounded-xl border border-neutral-200 p-6 shadow-xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Sparkles className="h-4 w-4 text-purple-500" />
            Generate AI Proposal
          </DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4 mt-4">
          <div className="space-y-1.5">
            <label className="text-sm font-medium">Sponsor</label>
            <Select
              value={sponsorId}
              onChange={(e) => setSponsorId(e.target.value)}
              options={sponsors.map((s) => ({ value: s.id, label: s.company?.name ?? s.id }))}
              placeholder="Select a sponsor"
              required
            />
          </div>
          <div className="space-y-1.5">
            <label className="text-sm font-medium">Tone</label>
            <Select
              value={tone}
              onChange={(e) => setTone(e.target.value)}
              options={[
                { value: "professional", label: "Professional" },
                { value: "friendly", label: "Friendly" },
                { value: "formal", label: "Formal" },
                { value: "casual", label: "Casual" },
              ]}
            />
          </div>
          <div className="space-y-1.5">
            <label className="text-sm font-medium">Package Tier</label>
            <Input
              value={packageTier}
              onChange={(e) => setPackageTier(e.target.value)}
              placeholder="e.g. Gold, Platinum, Title Sponsor"
            />
          </div>
          <div className="space-y-1.5">
            <label className="text-sm font-medium">Custom Notes</label>
            <Textarea
              value={customNotes}
              onChange={(e) => setCustomNotes(e.target.value)}
              placeholder="Any specific details to include..."
              rows={3}
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>Cancel</Button>
            <Button type="submit" disabled={generate.isPending || !sponsorId}>
              {generate.isPending ? "Generating..." : "Generate"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

export default function ProposalsPage() {
  const [showGenerate, setShowGenerate] = useState(false);
  const { data, isLoading } = useProposals();

  const proposals: Proposal[] = data?.data ?? [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Proposals</h1>
          <p className="text-sm text-muted-foreground mt-1">
            AI-generated sponsorship proposals
          </p>
        </div>
        <Button onClick={() => setShowGenerate(true)}>
          <Sparkles className="h-4 w-4 mr-2" />
          Generate Proposal
        </Button>
      </div>

      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Card key={i} className="animate-pulse">
              <CardContent className="p-5 space-y-3">
                <div className="h-4 bg-neutral-200 rounded w-3/4" />
                <div className="h-3 bg-neutral-100 rounded w-1/2" />
                <div className="h-3 bg-neutral-100 rounded w-1/3" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : proposals.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-16 text-center">
            <FileText className="h-12 w-12 text-muted-foreground/40 mb-4" />
            <h3 className="text-lg font-medium">No proposals yet</h3>
            <p className="text-sm text-muted-foreground mt-1 max-w-sm">
              Generate your first AI-powered sponsorship proposal to get started.
            </p>
            <Button className="mt-4" onClick={() => setShowGenerate(true)}>
              <Sparkles className="h-4 w-4 mr-2" />
              Generate First Proposal
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {proposals.map((proposal) => (
            <Card key={proposal.id} className="hover:shadow-md transition-shadow cursor-pointer">
              <CardContent className="p-5">
                <div className="flex items-start justify-between">
                  <h3 className="font-medium text-sm line-clamp-2">{proposal.title}</h3>
                  <Badge className={STATUS_COLORS[proposal.status] ?? STATUS_COLORS.draft}>
                    {proposal.status}
                  </Badge>
                </div>
                <div className="mt-3 space-y-1 text-xs text-muted-foreground">
                  {proposal.amount && (
                    <p>Amount: ${proposal.amount.toLocaleString()}</p>
                  )}
                  <p>Version {proposal.version}</p>
                  <p>{new Date(proposal.created_at).toLocaleDateString()}</p>
                </div>
                {proposal.content && (
                  <p className="mt-3 text-xs text-muted-foreground line-clamp-3">
                    {proposal.content.slice(0, 150)}...
                  </p>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <GenerateDialog open={showGenerate} onClose={() => setShowGenerate(false)} />
    </div>
  );
}
