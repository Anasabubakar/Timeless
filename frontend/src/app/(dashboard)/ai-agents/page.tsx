"use client";

import {
  Brain,
  Search,
  Target,
  Building2,
  Users,
  Copy,
  Link,
  Mail,
  Calendar,
  BarChart3,
  Lightbulb,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { motion } from "motion/react";

const agents = [
  {
    id: "research",
    name: "Research Agent",
    description: "Discovers potential sponsors through web scraping, news analysis, and industry reports",
    icon: Search,
    status: "active",
    lastRun: "12m ago",
    color: "text-blue-500",
  },
  {
    id: "qualification",
    name: "Qualification Agent",
    description: "Scores and qualifies sponsors based on fit, budget signals, and sponsorship history",
    icon: Target,
    status: "active",
    lastRun: "8m ago",
    color: "text-emerald-500",
  },
  {
    id: "company-intel",
    name: "Company Intel Agent",
    description: "Enriches company profiles with funding, revenue, tech stack, and recent news",
    icon: Building2,
    status: "active",
    lastRun: "25m ago",
    color: "text-violet-500",
  },
  {
    id: "decision-maker",
    name: "Decision Maker Agent",
    description: "Identifies key stakeholders and decision makers at target companies",
    icon: Users,
    status: "idle",
    lastRun: "1h ago",
    color: "text-amber-500",
  },
  {
    id: "duplicate",
    name: "Duplicate Agent",
    description: "Detects and merges duplicate companies, contacts, and sponsors across the pipeline",
    icon: Copy,
    status: "idle",
    lastRun: "3h ago",
    color: "text-neutral-500",
  },
  {
    id: "crm",
    name: "CRM Sync Agent",
    description: "Synchronizes data with external CRMs like Salesforce, HubSpot, and Pipedrive",
    icon: Link,
    status: "idle",
    lastRun: "2h ago",
    color: "text-indigo-500",
  },
  {
    id: "outreach",
    name: "Outreach Agent",
    description: "Drafts personalized sponsorship proposals and follow-up emails",
    icon: Mail,
    status: "active",
    lastRun: "5m ago",
    color: "text-rose-500",
  },
  {
    id: "meeting",
    name: "Meeting Agent",
    description: "Prepares meeting briefs and follow-up summaries for sponsor calls",
    icon: Calendar,
    status: "idle",
    lastRun: "6h ago",
    color: "text-cyan-500",
  },
  {
    id: "analytics",
    name: "Analytics Agent",
    description: "Analyzes pipeline trends, conversion rates, and ROI across campaigns",
    icon: BarChart3,
    status: "active",
    lastRun: "15m ago",
    color: "text-orange-500",
  },
  {
    id: "strategy",
    name: "Strategy Agent",
    description: "Recommends pricing tiers, package structures, and negotiation approaches",
    icon: Lightbulb,
    status: "idle",
    lastRun: "12h ago",
    color: "text-yellow-500",
  },
];

export default function AIAgentsPage() {
  return (
    <motion.div className="space-y-6" initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.3 }}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">AI Agents</h1>
          <p className="text-sm text-muted-foreground">
            Autonomous agents that power your sponsorship intelligence
          </p>
        </div>
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5">
            <div className="h-2 w-2 rounded-full bg-emerald-500" />
            <span className="text-xs text-muted-foreground">
              {agents.filter((a) => a.status === "active").length} active
            </span>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        {agents.map((agent) => (
          <div
            key={agent.id}
            className="group rounded-xl border border-border bg-card p-4 transition-shadow hover:shadow-sm"
          >
            <div className="flex items-start gap-3">
              <div className={cn("flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-muted", agent.color)}>
                <agent.icon className="h-4 w-4" />
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex items-center justify-between">
                  <h3 className="text-sm font-medium">{agent.name}</h3>
                  <div className="flex items-center gap-1.5">
                    <div
                      className={cn(
                        "h-1.5 w-1.5 rounded-full",
                        agent.status === "active" ? "bg-emerald-500" : "bg-neutral-300"
                      )}
                    />
                    <span className="text-[10px] text-muted-foreground">{agent.status}</span>
                  </div>
                </div>
                <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
                  {agent.description}
                </p>
                <div className="mt-3 flex items-center justify-between">
                  <span className="text-[10px] text-muted-foreground/60">
                    Last run: {agent.lastRun}
                  </span>
                  <button className="rounded-md border border-border px-2 py-1 text-[10px] font-medium text-muted-foreground opacity-0 transition-opacity hover:bg-accent group-hover:opacity-100">
                    Configure
                  </button>
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>
    </motion.div>
  );
}
