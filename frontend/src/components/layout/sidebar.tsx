"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Building2,
  Target,
  Users,
  Mail,
  BarChart3,
  Settings,
  Bot,
  Zap,
  FolderKanban,
  Activity,
  FileText,
  Link2,
} from "lucide-react";
import { cn } from "@/lib/utils";

const navigation = [
  { name: "Dashboard", href: "/dashboard", icon: LayoutDashboard },
  { name: "Campaigns", href: "/campaigns", icon: Target },
  { name: "Sponsors", href: "/sponsors", icon: FolderKanban },
  { name: "Proposals", href: "/proposals", icon: FileText },
  { name: "Companies", href: "/companies", icon: Building2 },
  { name: "Contacts", href: "/contacts", icon: Users },
  { name: "Activity", href: "/activities", icon: Activity },
  { name: "Outreach", href: "/outreach", icon: Mail },
  { name: "AI Agents", href: "/ai-agents", icon: Bot },
  { name: "Automations", href: "/automations", icon: Zap },
  { name: "Integrations", href: "/integrations", icon: Link2 },
  { name: "Analytics", href: "/analytics", icon: BarChart3 },
  { name: "Settings", href: "/settings", icon: Settings },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="fixed left-0 top-0 z-40 h-screen w-[240px] border-r border-border bg-card">
      <div className="flex h-14 items-center gap-2 border-b border-border px-5">
        <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-primary">
          <span className="text-xs font-bold text-primary-foreground">S</span>
        </div>
        <span className="text-sm font-semibold tracking-tight">SponsorOS</span>
      </div>

      <nav className="flex flex-col gap-0.5 p-3">
        {navigation.map((item) => {
          const isActive = pathname === item.href || pathname.startsWith(item.href + "/");
          return (
            <Link
              key={item.name}
              href={item.href}
              className={cn(
                "flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm transition-colors",
                isActive
                  ? "bg-accent text-accent-foreground font-medium"
                  : "text-muted-foreground hover:bg-accent/50 hover:text-foreground"
              )}
            >
              <item.icon className="h-4 w-4 shrink-0" />
              {item.name}
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}
